package inference

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	schemas "github.com/cannonball10/foundation/schemas/inference"
)

// ProviderName identifies an inference provider.
type ProviderName string

const (
	ProviderAnthropic ProviderName = "anthropic"
	ProviderOpenAI    ProviderName = "openai"
	ProviderGoogle    ProviderName = "google"
)

// TextInferenceRouter dispatches completion requests to the correct provider
// based on the model name.
type TextInferenceRouter struct {
	providers map[ProviderName]TextInferenceConnector
}

var _ TextInferenceConnector = (*TextInferenceRouter)(nil)

// NewTextInferenceRouter creates a router from a map of providers.
func NewTextInferenceRouter(providers map[ProviderName]TextInferenceConnector) *TextInferenceRouter {
	return &TextInferenceRouter{providers: providers}
}

// DefaultTextInferenceRouter initializes providers based on available API keys.
// Skips providers whose keys are not set. Returns an error only if no providers are available.
func DefaultTextInferenceRouter(ctx context.Context) (*TextInferenceRouter, error) {
	providers := make(map[ProviderName]TextInferenceConnector)

	if openai, err := DefaultOpenAITextInference(ctx); err == nil {
		providers[ProviderOpenAI] = openai
		slog.Info("inference provider registered", "provider", ProviderOpenAI)
	} else {
		slog.Warn("inference provider skipped", "provider", ProviderOpenAI, "error", err)
	}

	if anthropic, err := DefaultAnthropicTextInference(ctx); err == nil {
		providers[ProviderAnthropic] = anthropic
		slog.Info("inference provider registered", "provider", ProviderAnthropic)
	} else {
		slog.Warn("inference provider skipped", "provider", ProviderAnthropic, "error", err)
	}

	if gemini, err := DefaultGeminiTextInference(ctx); err == nil {
		providers[ProviderGoogle] = gemini
		slog.Info("inference provider registered", "provider", ProviderGoogle)
	} else {
		slog.Warn("inference provider skipped", "provider", ProviderGoogle, "error", err)
	}

	if len(providers) == 0 {
		return nil, &InferenceError{
			Code:    ErrProviderUnavail,
			Message: "no inference providers available (check API keys)",
		}
	}

	return NewTextInferenceRouter(providers), nil
}

// Complete resolves the provider for the model and delegates.
func (r *TextInferenceRouter) Complete(ctx context.Context, req *schemas.CompletionRequest) (*schemas.CompletionResponse, error) {
	provider, err := r.resolveProvider(req.Model)
	if err != nil {
		return nil, err
	}
	return provider.Complete(ctx, req)
}

// CompleteStream resolves the provider for the model and delegates.
func (r *TextInferenceRouter) CompleteStream(ctx context.Context, req *schemas.CompletionRequest) (*CompletionStream, error) {
	provider, err := r.resolveProvider(req.Model)
	if err != nil {
		return nil, err
	}
	return provider.CompleteStream(ctx, req)
}

// resolveProvider determines which provider handles a given model name.
func (r *TextInferenceRouter) resolveProvider(model string) (TextInferenceConnector, error) {
	name := resolveProviderName(model)

	provider, ok := r.providers[name]
	if !ok {
		return nil, &InferenceError{
			Code:     ErrModelNotFound,
			Provider: string(name),
			Message:  fmt.Sprintf("provider %q not available for model %q", name, model),
		}
	}

	return provider, nil
}

// resolveProviderName maps a model name to a provider using prefix matching.
func resolveProviderName(model string) ProviderName {
	m := strings.ToLower(model)

	switch {
	case strings.HasPrefix(m, "claude"):
		return ProviderAnthropic
	case strings.HasPrefix(m, "sonnet"):
		return ProviderAnthropic
	case strings.HasPrefix(m, "haiku"):
		return ProviderAnthropic
	case strings.HasPrefix(m, "gpt-"),
		strings.HasPrefix(m, "o1"),
		strings.HasPrefix(m, "o3"),
		strings.HasPrefix(m, "o4"):
		return ProviderOpenAI
	case strings.HasPrefix(m, "gemini"):
		return ProviderGoogle
	default:
		// Default to OpenAI for unknown models
		return ProviderOpenAI
	}
}

//go:build integration

package inference

import (
	"context"
	"errors"
	"os"
	"testing"

	schemas "github.com/cannonball10/foundation/schemas/inference"
	"github.com/joho/godotenv"
)

func skipIfNoOpenAIKey(t *testing.T) {
	t.Helper()
	if os.Getenv("OPENAI_API_KEY") == "" {
		t.Skip("OPENAI_API_KEY not set")
	}
}

func skipIfNoAnthropicKey(t *testing.T) {
	t.Helper()
	if os.Getenv("ANTHROPIC_API_KEY") == "" {
		t.Skip("ANTHROPIC_API_KEY not set")
	}
}

func skipIfNoGoogleKey(t *testing.T) {
	t.Helper()
	if os.Getenv("GOOGLE_AI_API_KEY") == "" {
		t.Skip("GOOGLE_AI_API_KEY not set")
	}
}

func setupRouterTest(t *testing.T) *TextInferenceRouter {
	t.Helper()
	err := godotenv.Load()
	if err != nil {
		t.Fatalf("Failed to load .env file: %v", err)
	}
	if os.Getenv("OPENAI_API_KEY") == "" &&
		os.Getenv("ANTHROPIC_API_KEY") == "" &&
		os.Getenv("GOOGLE_AI_API_KEY") == "" {
		t.Skip("no inference API keys set")
	}
	router, err := DefaultTextInferenceRouter(context.Background())
	if err != nil {
		t.Fatalf("DefaultTextInferenceRouter failed: %v", err)
	}
	return router
}

func TestRouter_DispatchOpenAI(t *testing.T) {
	router := setupRouterTest(t)
	skipIfNoOpenAIKey(t)
	ctx := context.Background()

	resp, err := router.Complete(ctx, &schemas.CompletionRequest{
		Model: "gpt-4o-mini",
		Messages: []schemas.Message{
			{Role: schemas.MessageRole_User, Content: "Say hello in exactly one word."},
		},
		MaxTokens: 50,
	})
	if err != nil {
		t.Fatalf("Router Complete (OpenAI) failed: %v", err)
	}

	if resp.Message.TextContent() == "" {
		t.Errorf("expected non-empty text response")
	}
}

func TestRouter_DispatchAnthropic(t *testing.T) {
	router := setupRouterTest(t)
	skipIfNoAnthropicKey(t)
	ctx := context.Background()

	resp, err := router.Complete(ctx, &schemas.CompletionRequest{
		Model: "claude-haiku-4-5-20251001",
		Messages: []schemas.Message{
			{Role: schemas.MessageRole_User, Content: "Say hello in exactly one word."},
		},
		MaxTokens: 50,
	})
	if err != nil {
		t.Fatalf("Router Complete (Anthropic) failed: %v", err)
	}

	if resp.Message.TextContent() == "" {
		t.Errorf("expected non-empty text response")
	}
}

func TestRouter_DispatchGemini(t *testing.T) {
	router := setupRouterTest(t)
	skipIfNoGoogleKey(t)
	ctx := context.Background()

	resp, err := router.Complete(ctx, &schemas.CompletionRequest{
		Model: "gemini-2.0-flash-lite",
		Messages: []schemas.Message{
			{Role: schemas.MessageRole_User, Content: "Say hello in exactly one word."},
		},
		MaxTokens: 50,
	})
	if err != nil {
		t.Fatalf("Router Complete (Gemini) failed: %v", err)
	}

	if resp.Message.TextContent() == "" {
		t.Errorf("expected non-empty text response")
	}
}

func TestRouter_UnknownModelFails(t *testing.T) {
	router := NewTextInferenceRouter(map[ProviderName]TextInferenceConnector{})
	ctx := context.Background()

	_, err := router.Complete(ctx, &schemas.CompletionRequest{
		Model: "nonexistent-model",
		Messages: []schemas.Message{
			{Role: schemas.MessageRole_User, Content: "Hello"},
		},
		MaxTokens: 50,
	})
	if err == nil {
		t.Fatalf("expected error for unknown model, got nil")
	}

	var infErr *InferenceError
	if !errors.As(err, &infErr) {
		t.Fatalf("expected *InferenceError, got %T: %v", err, err)
	}
	if infErr.Code != ErrModelNotFound {
		t.Errorf("expected error code %q, got %q", ErrModelNotFound, infErr.Code)
	}
}

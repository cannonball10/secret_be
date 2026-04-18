package inference

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"log/slog"
	"math"
	"math/rand/v2"
	"net/http"
	"strings"
	"time"

	"github.com/anthropics/anthropic-sdk-go"
	schemas "github.com/cannonball10/foundation/schemas/inference"
)

const (
	// httpStatusOverloaded is Anthropic's custom status code for overloaded servers.
	httpStatusOverloaded = 529

	retryMaxAttempts  = 4             // 1 initial + 3 retries
	retryBaseDelay    = 1 * time.Second
	retryMaxDelay     = 30 * time.Second
	retryJitterFactor = 0.5 // ±50% jitter
)

// AnthropicTextInference implements TextInferenceConnector using Anthropic's API.
type AnthropicTextInference struct {
	client anthropic.Client
}

// Ensure AnthropicTextInference implements TextInferenceConnector.
var _ TextInferenceConnector = (*AnthropicTextInference)(nil)

// NewAnthropicTextInference creates a new Anthropic inference connector with the provided client.
func NewAnthropicTextInference(client anthropic.Client) *AnthropicTextInference {
	return &AnthropicTextInference{client: client}
}

// DefaultAnthropicTextInference creates a new Anthropic inference connector with default configuration.
// Reads ANTHROPIC_API_KEY from environment.
func DefaultAnthropicTextInference(ctx context.Context) (*AnthropicTextInference, error) {
	client := anthropic.NewClient()
	return NewAnthropicTextInference(client), nil
}

// Complete performs a non-streaming completion request with automatic retries
// for transient errors (429, 529, 502, 503, 504).
func (a *AnthropicTextInference) Complete(ctx context.Context, req *schemas.CompletionRequest) (*schemas.CompletionResponse, error) {
	params, err := a.buildAnthropicParams(req)
	if err != nil {
		return nil, wrapAnthropicError(err, "failed to build request params")
	}

	var lastErr error
	for attempt := range retryMaxAttempts {
		resp, err := a.client.Messages.New(ctx, params)
		if err == nil {
			return a.mapResponse(resp), nil
		}

		wrapped := wrapAnthropicError(err, "completion request failed")
		lastErr = wrapped

		var inferErr *InferenceError
		if !errors.As(wrapped, &inferErr) || !inferErr.Retryable {
			return nil, wrapped
		}

		if attempt == retryMaxAttempts-1 {
			break
		}

		delay := retryBackoff(attempt)
		slog.Warn("retrying anthropic request",
			"attempt", attempt+1,
			"status", inferErr.StatusCode,
			"delay", delay)

		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(delay):
		}
	}

	return nil, lastErr
}

// retryBackoff returns an exponential backoff duration with jitter for the given attempt.
func retryBackoff(attempt int) time.Duration {
	delay := retryBaseDelay * time.Duration(1<<min(attempt, 5))
	if delay > retryMaxDelay {
		delay = retryMaxDelay
	}
	// Add jitter: delay * (1 ± jitterFactor/2)
	jitter := 1.0 + retryJitterFactor*(rand.Float64()-0.5)
	return time.Duration(math.Round(float64(delay) * jitter))
}

// CompleteStream performs a streaming completion request.
func (a *AnthropicTextInference) CompleteStream(ctx context.Context, req *schemas.CompletionRequest) (*CompletionStream, error) {
	params, err := a.buildAnthropicParams(req)
	if err != nil {
		return nil, wrapAnthropicError(err, "failed to build request params")
	}

	stream := a.client.Messages.NewStreaming(ctx, params)

	var (
		currentToolCallID   string
		currentToolCallName string
		eventQueue          []schemas.StreamEvent
		queueIndex          int
		finishReason        schemas.FinishReason
		usage               *schemas.Usage
	)

	nextFn := func() bool {
		// Drain any queued events first
		if queueIndex < len(eventQueue) {
			return true
		}

		// Reset queue for next SDK event
		eventQueue = eventQueue[:0]
		queueIndex = 0

		// Loop past SDK events that don't produce user-visible events
		// (e.g. message_start, ping).
		for stream.Next() {
			event := stream.Current()

			switch event.Type {
			case "content_block_start":
				if event.ContentBlock.Type == "tool_use" {
					currentToolCallID = event.ContentBlock.ID
					currentToolCallName = event.ContentBlock.Name
					eventQueue = append(eventQueue, schemas.StreamEvent{
						Type:         schemas.StreamEventType_ToolCallDelta,
						ToolCallID:   currentToolCallID,
						ToolCallName: currentToolCallName,
					})
				}

			case "content_block_delta":
				if event.Delta.Type == "text_delta" && event.Delta.Text != "" {
					eventQueue = append(eventQueue, schemas.StreamEvent{
						Type:  schemas.StreamEventType_Delta,
						Delta: event.Delta.Text,
					})
				} else if event.Delta.Type == "input_json_delta" && event.Delta.PartialJSON != "" {
					eventQueue = append(eventQueue, schemas.StreamEvent{
						Type:         schemas.StreamEventType_ToolCallDelta,
						ToolCallID:   currentToolCallID,
						ToolCallName: currentToolCallName,
						ToolCallArgs: event.Delta.PartialJSON,
					})
				}

			case "content_block_stop":
				currentToolCallID = ""
				currentToolCallName = ""

			case "message_delta":
				// Capture finish reason and usage from message_delta
				if event.Delta.StopReason != "" {
					finishReason = mapAnthropicFinishReason(anthropic.StopReason(event.Delta.StopReason))
				}
				if event.Usage.OutputTokens > 0 {
					usage = &schemas.Usage{
						CompletionTokens: int(event.Usage.OutputTokens),
					}
				}

			case "message_stop":
				eventQueue = append(eventQueue, schemas.StreamEvent{
					Type:         schemas.StreamEventType_Done,
					FinishReason: finishReason,
					Usage:        usage,
				})
			}

			if len(eventQueue) > 0 {
				return true
			}
		}

		return false
	}

	eventFn := func() schemas.StreamEvent {
		if queueIndex < len(eventQueue) {
			evt := eventQueue[queueIndex]
			queueIndex++
			return evt
		}
		return schemas.StreamEvent{}
	}

	errFn := func() error {
		if err := stream.Err(); err != nil {
			return wrapAnthropicError(err, "stream error")
		}
		return nil
	}

	closeFn := func() error {
		return stream.Close()
	}

	return NewCompletionStream(nextFn, eventFn, errFn, closeFn), nil
}

// buildAnthropicParams converts our schema request to Anthropic SDK params.
func (a *AnthropicTextInference) buildAnthropicParams(req *schemas.CompletionRequest) (anthropic.MessageNewParams, error) {
	var params anthropic.MessageNewParams

	// System field is []TextBlockParam — build directly, not via NewTextBlock
	var systemBlocks []anthropic.TextBlockParam
	var messages []anthropic.MessageParam

	for _, msg := range req.Messages {
		switch msg.Role {
		case schemas.MessageRole_System:
			systemBlocks = append(systemBlocks, anthropic.TextBlockParam{
				Text: msg.Content,
			})

		case schemas.MessageRole_User:
			var contentBlocks []anthropic.ContentBlockParamUnion

			if len(msg.Parts) > 0 {
				for _, part := range msg.Parts {
					switch part.Type {
					case schemas.ContentBlockType_ToolResult:
						if part.ToolResult != nil {
							contentBlocks = append(contentBlocks, anthropic.NewToolResultBlock(
								part.ToolResult.ToolCallID,
								part.ToolResult.Content,
								part.ToolResult.IsError,
							))
						}
					case schemas.ContentBlockType_Image:
						if part.ImageContent != nil {
							contentBlocks = append(contentBlocks, anthropic.NewImageBlockBase64(
								part.ImageContent.MediaType,
								base64.StdEncoding.EncodeToString(part.ImageContent.Data),
							))
						}
					case schemas.ContentBlockType_Document:
						if part.DocumentContent != nil {
							contentBlocks = append(contentBlocks, anthropic.NewDocumentBlock(
								anthropic.Base64PDFSourceParam{
									Data: base64.StdEncoding.EncodeToString(part.DocumentContent.Data),
								},
							))
						}
					case schemas.ContentBlockType_Text:
						if part.TextContent != nil {
							contentBlocks = append(contentBlocks, anthropic.NewTextBlock(part.TextContent.Value))
						}
					}
				}
			}

			if msg.Content != "" {
				contentBlocks = append(contentBlocks, anthropic.NewTextBlock(msg.Content))
			}

			if len(contentBlocks) > 0 {
				messages = append(messages, anthropic.NewUserMessage(contentBlocks...))
			}

		case schemas.MessageRole_Tool:
			var contentBlocks []anthropic.ContentBlockParamUnion

			if len(msg.Parts) > 0 {
				for _, part := range msg.Parts {
					if part.Type == schemas.ContentBlockType_ToolResult && part.ToolResult != nil {
						contentBlocks = append(contentBlocks, anthropic.NewToolResultBlock(
							part.ToolResult.ToolCallID,
							part.ToolResult.Content,
							part.ToolResult.IsError,
						))
					}
				}
			}

			if len(contentBlocks) > 0 {
				messages = append(messages, anthropic.NewUserMessage(contentBlocks...))
			}

		case schemas.MessageRole_Assistant:
			var contentBlocks []anthropic.ContentBlockParamUnion

			if msg.Content != "" {
				contentBlocks = append(contentBlocks, anthropic.NewTextBlock(msg.Content))
			}

			if len(msg.Parts) > 0 {
				for _, part := range msg.Parts {
					if part.Type == schemas.ContentBlockType_ToolCall && part.ToolCall != nil {
						contentBlocks = append(contentBlocks, anthropic.NewToolUseBlock(
							part.ToolCall.ID,
							part.ToolCall.Arguments,
							part.ToolCall.Name,
						))
					}
				}
			}

			if len(contentBlocks) > 0 {
				messages = append(messages, anthropic.NewAssistantMessage(contentBlocks...))
			}
		}
	}

	// Set required fields
	params.Model = anthropic.Model(req.Model)
	params.Messages = messages

	// MaxTokens is required by Anthropic, default to 4096
	maxTokens := int64(req.MaxTokens)
	if maxTokens == 0 {
		maxTokens = 4096
	}
	params.MaxTokens = maxTokens

	if len(systemBlocks) > 0 {
		params.System = systemBlocks
	}

	if req.Temperature != nil {
		params.Temperature = anthropic.Float(*req.Temperature)
	}

	if req.TopP != nil {
		params.TopP = anthropic.Float(*req.TopP)
	}

	if len(req.StopWords) > 0 {
		params.StopSequences = req.StopWords
	}

	// Convert tools
	if len(req.Tools) > 0 {
		var tools []anthropic.ToolUnionParam
		for _, tool := range req.Tools {
			var schemaMap map[string]any
			if len(tool.Parameters) > 0 {
				if err := json.Unmarshal(tool.Parameters, &schemaMap); err != nil {
					return params, err
				}
			}

			inputSchema := anthropic.ToolInputSchemaParam{}
			if props, ok := schemaMap["properties"]; ok {
				inputSchema.Properties = props
			}
			if reqFields, ok := schemaMap["required"].([]any); ok {
				required := make([]string, 0, len(reqFields))
				for _, r := range reqFields {
					if s, ok := r.(string); ok {
						required = append(required, s)
					}
				}
				inputSchema.Required = required
			}

			toolParam := anthropic.ToolParam{
				Name:        tool.Name,
				Description: anthropic.String(tool.Description),
				InputSchema: inputSchema,
			}

			tools = append(tools, anthropic.ToolUnionParam{
				OfTool: &toolParam,
			})
		}
		params.Tools = tools
	}

	return params, nil
}

// mapResponse converts Anthropic response to our schema response.
func (a *AnthropicTextInference) mapResponse(resp *anthropic.Message) *schemas.CompletionResponse {
	msg := schemas.Message{
		Role: schemas.MessageRole_Assistant,
	}

	var hasToolCalls bool
	var textContent string

	for _, block := range resp.Content {
		switch block.Type {
		case "text":
			textContent = block.Text
			msg.Parts = append(msg.Parts, schemas.ContentBlock{
				Type: schemas.ContentBlockType_Text,
				TextContent: &schemas.TextContent{
					Value: block.Text,
				},
			})

		case "tool_use":
			hasToolCalls = true
			msg.Parts = append(msg.Parts, schemas.ContentBlock{
				Type: schemas.ContentBlockType_ToolCall,
				ToolCall: &schemas.ToolCall{
					ID:        block.ID,
					Name:      block.Name,
					Arguments: block.Input,
				},
			})
		}
	}

	if !hasToolCalls && textContent != "" {
		msg.Content = textContent
	}

	return &schemas.CompletionResponse{
		ID:           resp.ID,
		Model:        string(resp.Model),
		Message:      msg,
		FinishReason: mapAnthropicFinishReason(resp.StopReason),
		Usage: schemas.Usage{
			PromptTokens:     int(resp.Usage.InputTokens),
			CompletionTokens: int(resp.Usage.OutputTokens),
			TotalTokens:      int(resp.Usage.InputTokens + resp.Usage.OutputTokens),
		},
	}
}

// mapAnthropicFinishReason converts Anthropic stop reason to our schema finish reason.
func mapAnthropicFinishReason(stopReason anthropic.StopReason) schemas.FinishReason {
	switch stopReason {
	case anthropic.StopReasonEndTurn:
		return schemas.FinishReason_Stop
	case anthropic.StopReasonMaxTokens:
		return schemas.FinishReason_Length
	case anthropic.StopReasonToolUse:
		return schemas.FinishReason_ToolCall
	default:
		return schemas.FinishReason_Stop
	}
}

// wrapAnthropicError wraps an Anthropic SDK error into InferenceError.
func wrapAnthropicError(err error, message string) error {
	if err == nil {
		return nil
	}

	inferErr := &InferenceError{
		Code:     ErrUnknown,
		Provider: "anthropic",
		Message:  message,
		Cause:    err,
	}

	if httpErr, ok := err.(interface{ StatusCode() int }); ok {
		statusCode := httpErr.StatusCode()
		inferErr.StatusCode = statusCode

		switch statusCode {
		case http.StatusTooManyRequests:
			inferErr.Code = ErrRateLimit
			inferErr.Retryable = true
		case http.StatusUnauthorized:
			inferErr.Code = ErrAuthentication
		case http.StatusBadRequest:
			errMsg := err.Error()
			if strings.Contains(errMsg, "content_policy") || strings.Contains(errMsg, "content filter") {
				inferErr.Code = ErrContentPolicy
			} else {
				inferErr.Code = ErrInvalidRequest
			}
		case http.StatusNotFound:
			if strings.Contains(err.Error(), "model") {
				inferErr.Code = ErrModelNotFound
			}
		case http.StatusServiceUnavailable, http.StatusBadGateway, http.StatusGatewayTimeout, httpStatusOverloaded:
			inferErr.Code = ErrProviderUnavail
			inferErr.Retryable = true
		}
	}

	slog.Error("anthropic inference error",
		"code", inferErr.Code,
		"status", inferErr.StatusCode,
		"message", message,
		"error", err)

	return inferErr
}

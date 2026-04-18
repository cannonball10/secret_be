package inference

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"

	schemas "github.com/cannonball10/foundation/schemas/inference"
	openai "github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/responses"
)

// Ensure *OpenAITextInference implements TextInferenceConnector.
var _ TextInferenceConnector = (*OpenAITextInference)(nil)

// OpenAITextInference implements TextInferenceConnector using the OpenAI Responses API.
type OpenAITextInference struct {
	client *openai.Client
}

// NewOpenAITextInference returns an OpenAITextInference that uses the given client.
func NewOpenAITextInference(client *openai.Client) *OpenAITextInference {
	return &OpenAITextInference{client: client}
}

// DefaultOpenAITextInference creates an OpenAITextInference with default client configuration.
func DefaultOpenAITextInference(ctx context.Context) (*OpenAITextInference, error) {
	client := openai.NewClient(openai.DefaultClientOptions()...)
	return NewOpenAITextInference(&client), nil
}

// Complete implements TextInferenceConnector.
func (o *OpenAITextInference) Complete(ctx context.Context, req *schemas.CompletionRequest) (*schemas.CompletionResponse, error) {
	params, err := buildResponsesParams(req)
	if err != nil {
		return nil, wrapOpenAIError(err, 0)
	}

	res, err := o.client.Responses.New(ctx, params)
	if err != nil {
		slog.ErrorContext(ctx, "openai Complete failed", "model", req.Model, "error", err)
		return nil, wrapOpenAIError(err, extractStatusCode(err))
	}

	return mapResponsesOutput(res), nil
}

// CompleteStream implements TextInferenceConnector.
func (o *OpenAITextInference) CompleteStream(ctx context.Context, req *schemas.CompletionRequest) (*CompletionStream, error) {
	params, err := buildResponsesParams(req)
	if err != nil {
		return nil, wrapOpenAIError(err, 0)
	}

	stream := o.client.Responses.NewStreaming(ctx, params)

	var (
		currentEvent        schemas.StreamEvent
		streamErr           error
		currentToolCallID   string
		currentToolCallName string
	)

	advanceFn := func() bool {
		// Loop through stream events until we find one we need to emit
		for stream.Next() {
			event := stream.Current()

			switch event.Type {
			case "response.output_text.delta":
				currentEvent = schemas.StreamEvent{
					Type:  schemas.StreamEventType_Delta,
					Delta: event.Delta,
				}
				return true

			case "response.output_item.added":
				if event.Item.Type == "function_call" {
					currentToolCallID = event.Item.CallID
					currentToolCallName = event.Item.Name
					currentEvent = schemas.StreamEvent{
						Type:         schemas.StreamEventType_ToolCallDelta,
						ToolCallID:   currentToolCallID,
						ToolCallName: currentToolCallName,
					}
					return true
				}

			case "response.function_call_arguments.delta":
				currentEvent = schemas.StreamEvent{
					Type:         schemas.StreamEventType_ToolCallDelta,
					ToolCallID:   currentToolCallID,
					ToolCallName: currentToolCallName,
					ToolCallArgs: event.Delta,
				}
				return true

			case "response.completed":
				finishReason := mapResponsesFinishReason(&event.Response)
				usage := &schemas.Usage{
					PromptTokens:     int(event.Response.Usage.InputTokens),
					CompletionTokens: int(event.Response.Usage.OutputTokens),
					TotalTokens:      int(event.Response.Usage.TotalTokens),
				}
				currentEvent = schemas.StreamEvent{
					Type:         schemas.StreamEventType_Done,
					FinishReason: finishReason,
					Usage:        usage,
				}
				return true

			case "response.incomplete":
				finishReason := mapResponsesFinishReason(&event.Response)
				usage := &schemas.Usage{
					PromptTokens:     int(event.Response.Usage.InputTokens),
					CompletionTokens: int(event.Response.Usage.OutputTokens),
					TotalTokens:      int(event.Response.Usage.TotalTokens),
				}
				currentEvent = schemas.StreamEvent{
					Type:         schemas.StreamEventType_Done,
					FinishReason: finishReason,
					Usage:        usage,
				}
				return true

			case "response.failed":
				streamErr = fmt.Errorf("response failed: status=%s", event.Response.Status)
				return false

			default:
				// Skip irrelevant events
				continue
			}
		}

		// No more events
		streamErr = stream.Err()
		return false
	}

	return NewCompletionStream(
		advanceFn,
		func() schemas.StreamEvent { return currentEvent },
		func() error {
			if streamErr != nil {
				return wrapOpenAIError(streamErr, extractStatusCode(streamErr))
			}
			return nil
		},
		func() error { return stream.Close() },
	), nil
}

// buildResponsesParams converts a CompletionRequest to OpenAI Responses API parameters.
func buildResponsesParams(req *schemas.CompletionRequest) (responses.ResponseNewParams, error) {
	var instructions []string
	inputItems := make([]responses.ResponseInputItemUnionParam, 0, len(req.Messages))

	for i, msg := range req.Messages {
		content := extractTextContent(&msg)

		switch msg.Role {
		case schemas.MessageRole_System:
			instructions = append(instructions, content)

		case schemas.MessageRole_User:
			inputItems = append(inputItems, responses.ResponseInputItemUnionParam{
				OfMessage: &responses.EasyInputMessageParam{
					Role: responses.EasyInputMessageRoleUser,
					Content: responses.EasyInputMessageContentUnionParam{
						OfString: openai.String(content),
					},
				},
			})

		case schemas.MessageRole_Assistant:
			toolCalls := msg.ToolCalls()
			if len(toolCalls) > 0 {
				// Emit text content first if present
				if content != "" {
					inputItems = append(inputItems, responses.ResponseInputItemUnionParam{
						OfMessage: &responses.EasyInputMessageParam{
							Role: responses.EasyInputMessageRoleAssistant,
							Content: responses.EasyInputMessageContentUnionParam{
								OfString: openai.String(content),
							},
						},
					})
				}

				// Emit each tool call separately
				for _, tc := range toolCalls {
					inputItems = append(inputItems, responses.ResponseInputItemParamOfFunctionCall(
						string(tc.Arguments),
						tc.ID,
						tc.Name,
					))
				}
			} else {
				inputItems = append(inputItems, responses.ResponseInputItemUnionParam{
					OfMessage: &responses.EasyInputMessageParam{
						Role: responses.EasyInputMessageRoleAssistant,
						Content: responses.EasyInputMessageContentUnionParam{
							OfString: openai.String(content),
						},
					},
				})
			}

		case schemas.MessageRole_Tool:
			if len(msg.Parts) == 0 || msg.Parts[0].Type != schemas.ContentBlockType_ToolResult {
				return responses.ResponseNewParams{}, fmt.Errorf("tool message at index %d missing tool result", i)
			}
			toolResult := msg.Parts[0].ToolResult
			inputItems = append(inputItems, responses.ResponseInputItemParamOfFunctionCallOutput(
				toolResult.ToolCallID,
				toolResult.Content,
			))

		default:
			return responses.ResponseNewParams{}, fmt.Errorf("unsupported message role: %s", msg.Role)
		}
	}

	params := responses.ResponseNewParams{
		Model: req.Model,
		Input: responses.ResponseNewParamsInputUnion{
			OfInputItemList: inputItems,
		},
	}

	if len(instructions) > 0 {
		params.Instructions = openai.String(strings.Join(instructions, "\n\n"))
	}

	if req.MaxTokens > 0 {
		params.MaxOutputTokens = openai.Int(int64(req.MaxTokens))
	}
	if req.Temperature != nil {
		params.Temperature = openai.Float(*req.Temperature)
	}
	if req.TopP != nil {
		params.TopP = openai.Float(*req.TopP)
	}
	if len(req.StopWords) > 0 {
		slog.Warn("StopWords are not supported by the OpenAI Responses API and will be ignored")
	}

	if len(req.Tools) > 0 {
		tools := make([]responses.ToolUnionParam, 0, len(req.Tools))
		for _, t := range req.Tools {
			var paramsMap map[string]any
			if len(t.Parameters) > 0 {
				if err := json.Unmarshal(t.Parameters, &paramsMap); err != nil {
					return responses.ResponseNewParams{}, fmt.Errorf("failed to unmarshal tool parameters: %w", err)
				}
			}

			tools = append(tools, responses.ToolUnionParam{
				OfFunction: &responses.FunctionToolParam{
					Name:        t.Name,
					Description: openai.String(t.Description),
					Parameters:  paramsMap,
				},
			})
		}
		params.Tools = tools
	}

	return params, nil
}

// mapResponsesOutput converts a Responses API response to a CompletionResponse.
func mapResponsesOutput(res *responses.Response) *schemas.CompletionResponse {
	response := &schemas.CompletionResponse{
		ID:           res.ID,
		Model:        res.Model,
		FinishReason: mapResponsesFinishReason(res),
		Message: schemas.Message{
			Role: schemas.MessageRole_Assistant,
		},
		Usage: schemas.Usage{
			PromptTokens:     int(res.Usage.InputTokens),
			CompletionTokens: int(res.Usage.OutputTokens),
			TotalTokens:      int(res.Usage.TotalTokens),
		},
	}

	var textContent strings.Builder
	var parts []schemas.ContentBlock
	hasToolCalls := false

	for _, item := range res.Output {
		switch item.Type {
		case "message":
			for _, content := range item.Content {
				if content.Type == "output_text" {
					textContent.WriteString(content.Text)
				}
			}

		case "function_call":
			hasToolCalls = true
			parts = append(parts, schemas.ContentBlock{
				Type: schemas.ContentBlockType_ToolCall,
				ToolCall: &schemas.ToolCall{
					ID:        item.CallID,
					Name:      item.Name,
					Arguments: json.RawMessage(item.Arguments),
				},
			})
		}
	}

	if hasToolCalls {
		// If we have tool calls, prepend text content as a block if present
		if textContent.Len() > 0 {
			parts = append([]schemas.ContentBlock{{
				Type: schemas.ContentBlockType_Text,
				TextContent: &schemas.TextContent{
					Value: textContent.String(),
				},
			}}, parts...)
		}
		response.Message.Parts = parts
	} else {
		response.Message.Content = textContent.String()
	}

	return response
}

// mapResponsesFinishReason converts Responses API status/output to schema finish reason.
func mapResponsesFinishReason(res *responses.Response) schemas.FinishReason {
	// Check if any output item is a function call
	for _, item := range res.Output {
		if item.Type == "function_call" {
			return schemas.FinishReason_ToolCall
		}
	}

	// Check for length limit
	if res.Status == "incomplete" && res.IncompleteDetails.Reason == "max_output_tokens" {
		return schemas.FinishReason_Length
	}

	return schemas.FinishReason_Stop
}

// extractTextContent extracts text content from a message's Parts or Content field.
func extractTextContent(msg *schemas.Message) string {
	if len(msg.Parts) > 0 {
		var content strings.Builder
		for _, part := range msg.Parts {
			if part.Type == schemas.ContentBlockType_Text && part.TextContent != nil {
				content.WriteString(part.TextContent.Value)
			}
		}
		return content.String()
	}
	return msg.Content
}

// wrapOpenAIError wraps an error in an InferenceError with appropriate classification.
func wrapOpenAIError(err error, statusCode int) error {
	if err == nil {
		return nil
	}

	code := ErrUnknown
	retryable := false
	message := err.Error()

	switch statusCode {
	case http.StatusTooManyRequests:
		code = ErrRateLimit
		retryable = true
		message = "rate limit exceeded"
	case http.StatusUnauthorized:
		code = ErrAuthentication
		message = "authentication failed"
	case http.StatusForbidden:
		code = ErrAuthentication
		message = "forbidden"
	case http.StatusBadRequest:
		code = ErrInvalidRequest
		message = "invalid request"
	case http.StatusNotFound:
		code = ErrModelNotFound
		message = "model not found"
	case http.StatusInternalServerError, http.StatusBadGateway, http.StatusServiceUnavailable, http.StatusGatewayTimeout:
		code = ErrProviderUnavail
		retryable = true
		message = "provider unavailable"
	}

	return &InferenceError{
		Code:       code,
		Provider:   "openai",
		StatusCode: statusCode,
		Message:    message,
		Retryable:  retryable,
		Cause:      err,
	}
}

// extractStatusCode attempts to extract an HTTP status code from an error.
func extractStatusCode(err error) int {
	type statusCoder interface {
		StatusCode() int
	}

	if sc, ok := err.(statusCoder); ok {
		return sc.StatusCode()
	}

	return 0
}

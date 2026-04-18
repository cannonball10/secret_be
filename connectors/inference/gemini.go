package inference

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"

	"google.golang.org/genai"

	schemas "github.com/cannonball10/foundation/schemas/inference"
)

// GeminiTextInference implements TextInferenceConnector using Google's Gemini API.
type GeminiTextInference struct {
	client *genai.Client
}

var _ TextInferenceConnector = (*GeminiTextInference)(nil)

// NewGeminiTextInference creates a new Gemini text inference connector with the provided client.
func NewGeminiTextInference(client *genai.Client) *GeminiTextInference {
	return &GeminiTextInference{client: client}
}

// DefaultGeminiTextInference creates a new Gemini connector using the GOOGLE_AI_API_KEY environment variable.
func DefaultGeminiTextInference(ctx context.Context) (*GeminiTextInference, error) {
	apiKey := os.Getenv("GOOGLE_AI_API_KEY")
	if apiKey == "" {
		return nil, fmt.Errorf("GOOGLE_AI_API_KEY environment variable not set")
	}

	client, err := genai.NewClient(ctx, &genai.ClientConfig{
		APIKey:  apiKey,
		Backend: genai.BackendGeminiAPI,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create Gemini client: %w", err)
	}

	return &GeminiTextInference{client: client}, nil
}

// Complete performs a non-streaming text inference request.
func (g *GeminiTextInference) Complete(ctx context.Context, req *schemas.CompletionRequest) (*schemas.CompletionResponse, error) {
	contents, config := g.buildRequest(req)

	resp, err := g.client.Models.GenerateContent(ctx, req.Model, contents, config)
	if err != nil {
		slog.ErrorContext(ctx, "gemini Complete failed", "model", req.Model, "error", err)
		return nil, &InferenceError{
			Code:     ErrUnknown,
			Provider: "google",
			Message:  "generate content failed",
			Cause:    err,
		}
	}

	if len(resp.Candidates) == 0 {
		return nil, &InferenceError{
			Code:     ErrUnknown,
			Provider: "google",
			Message:  "no candidates in response",
		}
	}

	candidate := resp.Candidates[0]
	message := g.convertResponseToMessage(candidate)

	return &schemas.CompletionResponse{
		ID:           resp.ResponseID,
		Model:        req.Model,
		Message:      message,
		FinishReason: mapGeminiFinishReason(candidate.FinishReason),
		Usage:        mapGeminiUsage(resp.UsageMetadata),
	}, nil
}

// CompleteStream performs a streaming text inference request.
func (g *GeminiTextInference) CompleteStream(ctx context.Context, req *schemas.CompletionRequest) (*CompletionStream, error) {
	contents, config := g.buildRequest(req)

	responses := g.client.Models.GenerateContentStream(ctx, req.Model, contents, config)

	eventCh := make(chan schemas.StreamEvent, 16)
	errCh := make(chan error, 1)

	go func() {
		defer close(eventCh)

		var lastUsage schemas.Usage
		var lastFinishReason schemas.FinishReason

		for resp, iterErr := range responses {
			if iterErr != nil {
				errCh <- iterErr
				return
			}

			if resp.UsageMetadata != nil {
				lastUsage = mapGeminiUsage(resp.UsageMetadata)
			}

			if len(resp.Candidates) == 0 {
				continue
			}

			candidate := resp.Candidates[0]
			if candidate.FinishReason != "" {
				lastFinishReason = mapGeminiFinishReason(candidate.FinishReason)
			}

			if candidate.Content == nil || len(candidate.Content.Parts) == 0 {
				continue
			}

			for _, part := range candidate.Content.Parts {
				if part.Text != "" {
					eventCh <- schemas.StreamEvent{
						Type:  schemas.StreamEventType_Delta,
						Delta: part.Text,
					}
				} else if part.FunctionCall != nil {
					argsJSON, _ := json.Marshal(part.FunctionCall.Args)
					eventCh <- schemas.StreamEvent{
						Type:         schemas.StreamEventType_ToolCallDelta,
						ToolCallID:   part.FunctionCall.ID,
						ToolCallName: part.FunctionCall.Name,
						ToolCallArgs: string(argsJSON),
					}
				}
			}
		}

		// Send final done event
		eventCh <- schemas.StreamEvent{
			Type:         schemas.StreamEventType_Done,
			FinishReason: lastFinishReason,
			Usage:        &lastUsage,
		}
	}()

	// Wrap channels in CompletionStream
	var (
		currentEvent schemas.StreamEvent
		streamErr    error
		done         bool
	)

	nextFn := func() bool {
		if done {
			return false
		}
		select {
		case evt, ok := <-eventCh:
			if !ok {
				done = true
				select {
				case err := <-errCh:
					streamErr = err
				default:
				}
				return false
			}
			currentEvent = evt
			return true
		case err := <-errCh:
			streamErr = err
			done = true
			return false
		}
	}

	return NewCompletionStream(
		nextFn,
		func() schemas.StreamEvent { return currentEvent },
		func() error { return streamErr },
		func() error {
			// Drain remaining events
			for range eventCh {
			}
			return nil
		},
	), nil
}

// buildRequest constructs the Gemini API request from our schema.
func (g *GeminiTextInference) buildRequest(req *schemas.CompletionRequest) ([]*genai.Content, *genai.GenerateContentConfig) {
	var contents []*genai.Content
	var systemText string

	for _, msg := range req.Messages {
		switch msg.Role {
		case schemas.MessageRole_System:
			if systemText != "" {
				systemText += "\n\n"
			}
			systemText += msg.Content

		case schemas.MessageRole_User:
			parts := g.convertMessageParts(msg)
			contents = append(contents, &genai.Content{
				Role:  "user",
				Parts: parts,
			})

		case schemas.MessageRole_Assistant:
			parts := g.convertMessageParts(msg)
			contents = append(contents, &genai.Content{
				Role:  "model",
				Parts: parts,
			})

		case schemas.MessageRole_Tool:
			parts := g.convertToolMessageParts(msg)
			contents = append(contents, &genai.Content{
				Role:  "user",
				Parts: parts,
			})
		}
	}

	config := &genai.GenerateContentConfig{}

	if systemText != "" {
		config.SystemInstruction = &genai.Content{
			Parts: []*genai.Part{{Text: systemText}},
		}
	}

	if req.MaxTokens > 0 {
		config.MaxOutputTokens = int32(req.MaxTokens)
	}

	if req.Temperature != nil {
		temp := float32(*req.Temperature)
		config.Temperature = &temp
	}

	if req.TopP != nil {
		topP := float32(*req.TopP)
		config.TopP = &topP
	}

	if len(req.StopWords) > 0 {
		config.StopSequences = req.StopWords
	}

	if len(req.Tools) > 0 {
		functionDecls := make([]*genai.FunctionDeclaration, 0, len(req.Tools))
		for _, tool := range req.Tools {
			schema := g.jsonSchemaToGenaiSchema(tool.Parameters)

			functionDecls = append(functionDecls, &genai.FunctionDeclaration{
				Name:        tool.Name,
				Description: tool.Description,
				Parameters:  schema,
			})
		}

		config.Tools = []*genai.Tool{
			{FunctionDeclarations: functionDecls},
		}
	}

	return contents, config
}

// convertMessageParts converts a Message to Gemini Parts.
func (g *GeminiTextInference) convertMessageParts(msg schemas.Message) []*genai.Part {
	var parts []*genai.Part

	if msg.Content != "" && len(msg.Parts) == 0 {
		parts = append(parts, &genai.Part{Text: msg.Content})
		return parts
	}

	for _, block := range msg.Parts {
		switch block.Type {
		case schemas.ContentBlockType_Text:
			if block.TextContent != nil {
				parts = append(parts, &genai.Part{Text: block.TextContent.Value})
			}

		case schemas.ContentBlockType_ToolCall:
			if block.ToolCall != nil {
				var argsMap map[string]any
				if err := json.Unmarshal(block.ToolCall.Arguments, &argsMap); err != nil {
					slog.Warn("failed to unmarshal tool call args for Gemini", "error", err)
					argsMap = map[string]any{}
				}

				parts = append(parts, &genai.Part{
					FunctionCall: &genai.FunctionCall{
						ID:   block.ToolCall.ID,
						Name: block.ToolCall.Name,
						Args: argsMap,
					},
				})
			}
		}
	}

	if len(parts) == 0 {
		parts = append(parts, &genai.Part{Text: ""})
	}

	return parts
}

// convertToolMessageParts converts tool result messages to Gemini function response parts.
func (g *GeminiTextInference) convertToolMessageParts(msg schemas.Message) []*genai.Part {
	var parts []*genai.Part

	for _, block := range msg.Parts {
		if block.Type == schemas.ContentBlockType_ToolResult && block.ToolResult != nil {
			var responseMap map[string]any
			if err := json.Unmarshal([]byte(block.ToolResult.Content), &responseMap); err != nil {
				responseMap = map[string]any{"result": block.ToolResult.Content}
			}

			parts = append(parts, &genai.Part{
				FunctionResponse: &genai.FunctionResponse{
					Name:     block.ToolResult.ToolCallID,
					Response: responseMap,
				},
			})
		}
	}

	if len(parts) == 0 && msg.Content != "" {
		parts = append(parts, &genai.Part{Text: msg.Content})
	}

	return parts
}

// convertResponseToMessage converts a Gemini candidate to a Message.
func (g *GeminiTextInference) convertResponseToMessage(candidate *genai.Candidate) schemas.Message {
	msg := schemas.Message{
		Role: schemas.MessageRole_Assistant,
	}

	if candidate.Content == nil || len(candidate.Content.Parts) == 0 {
		return msg
	}

	var textParts []string

	for _, part := range candidate.Content.Parts {
		if part.Text != "" {
			textParts = append(textParts, part.Text)
			msg.Parts = append(msg.Parts, schemas.ContentBlock{
				Type: schemas.ContentBlockType_Text,
				TextContent: &schemas.TextContent{
					Value: part.Text,
				},
			})
		} else if part.FunctionCall != nil {
			argsJSON, _ := json.Marshal(part.FunctionCall.Args)
			toolCallID := part.FunctionCall.ID
			if toolCallID == "" {
				toolCallID = fmt.Sprintf("call_%s", part.FunctionCall.Name)
			}

			msg.Parts = append(msg.Parts, schemas.ContentBlock{
				Type: schemas.ContentBlockType_ToolCall,
				ToolCall: &schemas.ToolCall{
					ID:        toolCallID,
					Name:      part.FunctionCall.Name,
					Arguments: argsJSON,
				},
			})
		}
	}

	if len(textParts) > 0 {
		msg.Content = textParts[0]
	}

	return msg
}

// mapGeminiFinishReason maps Gemini finish reasons to our schema.
func mapGeminiFinishReason(reason genai.FinishReason) schemas.FinishReason {
	switch reason {
	case genai.FinishReasonStop:
		return schemas.FinishReason_Stop
	case genai.FinishReasonMaxTokens:
		return schemas.FinishReason_Length
	default:
		return schemas.FinishReason_Stop
	}
}

// mapGeminiUsage maps Gemini usage metadata to our schema.
func mapGeminiUsage(metadata *genai.GenerateContentResponseUsageMetadata) schemas.Usage {
	if metadata == nil {
		return schemas.Usage{}
	}
	return schemas.Usage{
		PromptTokens:     int(metadata.PromptTokenCount),
		CompletionTokens: int(metadata.CandidatesTokenCount),
		TotalTokens:      int(metadata.TotalTokenCount),
	}
}

// jsonSchemaToGenaiSchema converts JSON Schema (as json.RawMessage) to Gemini's Schema format.
func (g *GeminiTextInference) jsonSchemaToGenaiSchema(raw json.RawMessage) *genai.Schema {
	if len(raw) == 0 {
		return &genai.Schema{Type: genai.TypeObject}
	}

	var schemaMap map[string]any
	if err := json.Unmarshal(raw, &schemaMap); err != nil {
		slog.Warn("failed to unmarshal JSON schema for Gemini", "error", err)
		return &genai.Schema{Type: genai.TypeObject}
	}

	return g.convertSchemaMap(schemaMap)
}

// convertSchemaMap recursively converts a JSON Schema map to a Gemini Schema.
func (g *GeminiTextInference) convertSchemaMap(schemaMap map[string]any) *genai.Schema {
	schema := &genai.Schema{}

	if typeVal, ok := schemaMap["type"].(string); ok {
		schema.Type = mapGeminiSchemaType(typeVal)
	}

	if desc, ok := schemaMap["description"].(string); ok {
		schema.Description = desc
	}

	if enumVal, ok := schemaMap["enum"].([]any); ok {
		enumStrs := make([]string, 0, len(enumVal))
		for _, e := range enumVal {
			if s, ok := e.(string); ok {
				enumStrs = append(enumStrs, s)
			}
		}
		schema.Enum = enumStrs
	}

	if props, ok := schemaMap["properties"].(map[string]any); ok {
		schema.Properties = make(map[string]*genai.Schema)
		for key, val := range props {
			if propMap, ok := val.(map[string]any); ok {
				schema.Properties[key] = g.convertSchemaMap(propMap)
			}
		}
	}

	if reqVal, ok := schemaMap["required"].([]any); ok {
		required := make([]string, 0, len(reqVal))
		for _, r := range reqVal {
			if s, ok := r.(string); ok {
				required = append(required, s)
			}
		}
		schema.Required = required
	}

	if items, ok := schemaMap["items"].(map[string]any); ok {
		schema.Items = g.convertSchemaMap(items)
	}

	return schema
}

// mapGeminiSchemaType converts JSON Schema type strings to Gemini Type constants.
func mapGeminiSchemaType(typeStr string) genai.Type {
	switch typeStr {
	case "string":
		return genai.TypeString
	case "number":
		return genai.TypeNumber
	case "integer":
		return genai.TypeInteger
	case "boolean":
		return genai.TypeBoolean
	case "array":
		return genai.TypeArray
	case "object":
		return genai.TypeObject
	default:
		return genai.TypeString
	}
}

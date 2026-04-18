//go:build integration

package inference

import (
	"context"
	"encoding/json"
	"os"
	"testing"

	schemas "github.com/cannonball10/foundation/schemas/inference"
)

func setupGeminiTest(t *testing.T) *GeminiTextInference {
	t.Helper()
	err := loadEnvFromProjectRoot()
	if err != nil {
		t.Fatalf("Failed to load .env file: %v", err)
	}
	if os.Getenv("GOOGLE_AI_API_KEY") == "" {
		t.Skip("GOOGLE_AI_API_KEY not set")
	}
	connector, err := DefaultGeminiTextInference(context.Background())
	if err != nil {
		t.Fatalf("DefaultGeminiTextInference failed: %v", err)
	}
	return connector
}

func TestGemini_Complete(t *testing.T) {
	connector := setupGeminiTest(t)
	ctx := context.Background()

	resp, err := connector.Complete(ctx, &schemas.CompletionRequest{
		Model: "gemini-2.0-flash-lite",
		Messages: []schemas.Message{
			{Role: schemas.MessageRole_User, Content: "Say hello in exactly one word."},
		},
		MaxTokens: 50,
	})
	if err != nil {
		t.Fatalf("Complete failed: %v", err)
	}

	if resp.ID == "" {
		t.Errorf("expected non-empty response ID")
	}
	if resp.Model == "" {
		t.Errorf("expected non-empty model")
	}
	if resp.Message.TextContent() == "" {
		t.Errorf("expected non-empty text content")
	}
	if resp.FinishReason != schemas.FinishReason_Stop {
		t.Errorf("expected FinishReason_Stop, got %q", resp.FinishReason)
	}
	if resp.Usage.TotalTokens <= 0 {
		t.Errorf("expected TotalTokens > 0, got %d", resp.Usage.TotalTokens)
	}
}

func TestGemini_CompleteStream(t *testing.T) {
	connector := setupGeminiTest(t)
	ctx := context.Background()

	stream, err := connector.CompleteStream(ctx, &schemas.CompletionRequest{
		Model: "gemini-2.0-flash-lite",
		Messages: []schemas.Message{
			{Role: schemas.MessageRole_User, Content: "Say hello in exactly one word."},
		},
		MaxTokens: 50,
	})
	if err != nil {
		t.Fatalf("CompleteStream failed: %v", err)
	}
	defer stream.Close()

	deltaCount := 0
	gotDone := false

	for stream.Next() {
		event := stream.Event()
		switch event.Type {
		case schemas.StreamEventType_Delta:
			deltaCount++
		case schemas.StreamEventType_Done:
			gotDone = true
		}
	}

	if err := stream.Err(); err != nil {
		t.Fatalf("stream error: %v", err)
	}
	if deltaCount < 1 {
		t.Errorf("expected at least 1 delta event, got %d", deltaCount)
	}
	if !gotDone {
		t.Errorf("expected done event")
	}
}

func TestGemini_CompleteWithTools(t *testing.T) {
	connector := setupGeminiTest(t)
	ctx := context.Background()

	resp, err := connector.Complete(ctx, &schemas.CompletionRequest{
		Model: "gemini-2.0-flash-lite",
		Messages: []schemas.Message{
			{Role: schemas.MessageRole_User, Content: "What is the weather in San Francisco?"},
		},
		Tools:     []schemas.ToolDefinition{weatherToolDefinition()},
		MaxTokens: 100,
	})
	if err != nil {
		t.Fatalf("Complete with tools failed: %v", err)
	}

	// Note: Gemini may return FinishReason_Stop even on tool calls,
	// so we do not assert on FinishReason here.

	toolCalls := resp.Message.ToolCalls()
	if len(toolCalls) == 0 {
		t.Fatalf("expected at least one tool call")
	}

	tc := toolCalls[0]
	if tc.Name != "get_weather" {
		t.Errorf("expected tool call name %q, got %q", "get_weather", tc.Name)
	}

	var args map[string]interface{}
	if err := json.Unmarshal(tc.Arguments, &args); err != nil {
		t.Fatalf("failed to unmarshal tool call arguments: %v", err)
	}
}

func TestGemini_CompleteStreamWithTools(t *testing.T) {
	connector := setupGeminiTest(t)
	ctx := context.Background()

	stream, err := connector.CompleteStream(ctx, &schemas.CompletionRequest{
		Model: "gemini-2.0-flash-lite",
		Messages: []schemas.Message{
			{Role: schemas.MessageRole_User, Content: "What is the weather in San Francisco?"},
		},
		Tools:     []schemas.ToolDefinition{weatherToolDefinition()},
		MaxTokens: 100,
	})
	if err != nil {
		t.Fatalf("CompleteStream with tools failed: %v", err)
	}
	defer stream.Close()

	toolCallDeltaCount := 0
	gotDone := false

	for stream.Next() {
		event := stream.Event()
		switch event.Type {
		case schemas.StreamEventType_ToolCallDelta:
			toolCallDeltaCount++
		case schemas.StreamEventType_Done:
			gotDone = true
		}
	}

	if err := stream.Err(); err != nil {
		t.Fatalf("stream error: %v", err)
	}
	if toolCallDeltaCount < 1 {
		t.Errorf("expected at least 1 tool_call_delta event, got %d", toolCallDeltaCount)
	}
	if !gotDone {
		t.Errorf("expected done event")
	}
	// Note: Gemini done event finish reason is not asserted because
	// Gemini may not map tool calls to FinishReason_ToolCall.
}

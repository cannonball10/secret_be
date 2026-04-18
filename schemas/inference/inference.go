package inference

import (
	"encoding/json"
	"strings"
)

type MessageRole string

const (
	MessageRole_User      MessageRole = "user"
	MessageRole_Assistant MessageRole = "assistant"
	MessageRole_System    MessageRole = "system"
	MessageRole_Tool      MessageRole = "tool"
)

// ContentBlockType identifies the type of content in a message part.
type ContentBlockType string

const (
	ContentBlockType_Text       ContentBlockType = "text"
	ContentBlockType_ToolCall   ContentBlockType = "tool_call"
	ContentBlockType_ToolResult ContentBlockType = "tool_result"
	ContentBlockType_Image      ContentBlockType = "image"
	ContentBlockType_Document   ContentBlockType = "document"
)

// ContentBlock is a tagged union representing a single part of a message.
type ContentBlock struct {
	Type ContentBlockType `json:"type"`

	// Exactly one of the following should be non-nil, based on Type
	*TextContent
	*ToolCall
	*ToolResult
	*ImageContent
	*DocumentContent
}

// TextContent represents plain text content.
type TextContent struct {
	Value string `json:"value"`
}

// ToolCall represents a request to invoke a tool.
type ToolCall struct {
	ID        string          `json:"id"`
	Name      string          `json:"name"`
	Arguments json.RawMessage `json:"arguments"`
}

// ToolResult represents the result of a tool invocation.
type ToolResult struct {
	ToolCallID string `json:"toolCallId"`
	Content    string `json:"content"`
	IsError    bool   `json:"isError"`
}

// ImageContent represents an image for vision models.
type ImageContent struct {
	MediaType string `json:"mediaType"` // e.g. "image/png", "image/jpeg", "image/gif", "image/webp"
	Data      []byte `json:"data"`      // raw image bytes
}

// DocumentContent represents a document (PDF) for vision models.
type DocumentContent struct {
	MediaType string `json:"mediaType"` // e.g. "application/pdf"
	Data      []byte `json:"data"`      // raw document bytes
}

// Message is a resolved prompt ready for inference.
type Message struct {
	Role    MessageRole
	Content string

	// Parts provides structured multi-modal content.
	// When empty, Content is used for backward compatibility.
	Parts []ContentBlock
}

// TextContent returns the text content of the message.
// If Parts is empty, returns Content for backward compatibility.
// Otherwise, concatenates all text parts.
func (m *Message) TextContent() string {
	if len(m.Parts) == 0 {
		return m.Content
	}

	var b strings.Builder
	for _, part := range m.Parts {
		if part.Type == ContentBlockType_Text && part.TextContent != nil {
			b.WriteString(part.TextContent.Value)
		}
	}
	return b.String()
}

// ToolCalls extracts all tool call parts from the message.
func (m *Message) ToolCalls() []ToolCall {
	var calls []ToolCall
	for _, part := range m.Parts {
		if part.Type == ContentBlockType_ToolCall && part.ToolCall != nil {
			calls = append(calls, *part.ToolCall)
		}
	}
	return calls
}

// Example is an input/output pair for few-shot learning.
type Example struct {
	Input  string
	Output string
}

// ModelConfig holds LLM inference parameters.
type ModelConfig struct {
	Model       string   `json:"model,omitempty"`
	MaxTokens   int      `json:"maxTokens,omitempty"`
	Temperature *float64 `json:"temperature,omitempty"`
	TopP        *float64 `json:"topP,omitempty"`
	StopWords   []string `json:"stopWords,omitempty"`
}

// ToolDefinition defines a tool that can be called by the model.
type ToolDefinition struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Parameters  json.RawMessage `json:"parameters"` // JSON Schema
}

// CompletionRequest represents a request to generate a completion.
type CompletionRequest struct {
	Model       string           `json:"model"`
	Messages    []Message        `json:"messages"`
	Tools       []ToolDefinition `json:"tools,omitempty"`
	MaxTokens   int              `json:"maxTokens,omitempty"`
	Temperature *float64         `json:"temperature,omitempty"`
	TopP        *float64         `json:"topP,omitempty"`
	StopWords   []string         `json:"stopWords,omitempty"`
}

// FinishReason indicates why the model stopped generating.
type FinishReason string

const (
	FinishReason_Stop     FinishReason = "stop"
	FinishReason_Length   FinishReason = "length"
	FinishReason_ToolCall FinishReason = "tool_call"
)

// Usage tracks token consumption for a completion.
type Usage struct {
	PromptTokens     int `json:"promptTokens"`
	CompletionTokens int `json:"completionTokens"`
	TotalTokens      int `json:"totalTokens"`
}

// CompletionResponse represents a completed generation.
type CompletionResponse struct {
	ID           string       `json:"id"`
	Model        string       `json:"model"`
	Message      Message      `json:"message"`
	FinishReason FinishReason `json:"finishReason"`
	Usage        Usage        `json:"usage"`
}

// StreamEventType identifies the type of streaming event.
type StreamEventType string

const (
	StreamEventType_Delta        StreamEventType = "delta"
	StreamEventType_ToolCallDelta StreamEventType = "tool_call_delta"
	StreamEventType_Done         StreamEventType = "done"
)

// StreamEvent represents a single event in a streaming completion.
type StreamEvent struct {
	Type         StreamEventType `json:"type"`
	Delta        string          `json:"delta,omitempty"`
	ToolCallID   string          `json:"toolCallId,omitempty"`
	ToolCallName string          `json:"toolCallName,omitempty"`
	ToolCallArgs string          `json:"toolCallArgs,omitempty"`
	FinishReason FinishReason    `json:"finishReason,omitempty"`
	Usage        *Usage          `json:"usage,omitempty"`
}

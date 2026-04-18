package livekit

import "time"

// AgentDispatchMetadata contains configuration passed to a LiveKit agent at dispatch time.
// This struct is serialized to JSON and passed as metadata to the agent.
type AgentDispatchMetadata struct {
	// SystemPrompt is the system prompt that configures the AI agent's behavior.
	SystemPrompt string `json:"system_prompt"`

	// UserID identifies the user who initiated the agent session.
	UserID string `json:"user_id,omitempty"`

	// SessionID is a unique identifier for this agent session.
	SessionID string `json:"session_id"`

	// Tools is a list of tool names the agent is allowed to use.
	Tools []string `json:"tools,omitempty"`

	// Context contains additional key-value data for the agent.
	Context map[string]any `json:"context,omitempty"`
}

// RoomOptions configures room creation.
type RoomOptions struct {
	// EmptyTimeout is the duration after which an empty room is automatically deleted.
	EmptyTimeout time.Duration

	// MaxParticipants is the maximum number of participants allowed in the room.
	MaxParticipants uint32

	// Metadata is arbitrary room metadata.
	Metadata string
}

// Room represents a LiveKit room.
type Room struct {
	// Name is the unique room name.
	Name string

	// SID is the server-assigned room session ID.
	SID string

	// NumParticipants is the current number of participants in the room.
	NumParticipants uint32

	// MaxParticipants is the maximum allowed participants.
	MaxParticipants uint32

	// CreationTime is when the room was created.
	CreationTime time.Time

	// Metadata is the room's metadata.
	Metadata string
}

// AgentDispatch represents a dispatched agent.
type AgentDispatch struct {
	// AgentID is the unique identifier of the dispatched agent.
	AgentID string

	// RoomName is the room the agent was dispatched to.
	RoomName string

	// AgentName is the name of the agent that was dispatched.
	AgentName string
}

// TokenOptions configures token generation.
type TokenOptions struct {
	// TTL is the token's time-to-live duration.
	TTL time.Duration

	// CanPublish indicates whether the participant can publish tracks.
	CanPublish bool

	// CanSubscribe indicates whether the participant can subscribe to tracks.
	CanSubscribe bool

	// CanPublishData indicates whether the participant can publish data messages.
	CanPublishData bool

	// Metadata is participant metadata included in the token.
	Metadata string
}

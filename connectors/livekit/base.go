package livekit

import (
	"context"

	"github.com/cannonball10/foundation/schemas/livekit"
)

// LiveKitConnector provides an interface for interacting with LiveKit services.
type LiveKitConnector interface {
	// CreateRoom creates a new LiveKit room with the given name and options.
	CreateRoom(ctx context.Context, roomName string, opts *livekit.RoomOptions) (*livekit.Room, error)

	// GetRoom retrieves information about an existing room.
	GetRoom(ctx context.Context, roomName string) (*livekit.Room, error)

	// DeleteRoom removes a room and disconnects all participants.
	DeleteRoom(ctx context.Context, roomName string) error

	// DispatchAgent dispatches an agent to a room with the given metadata.
	DispatchAgent(ctx context.Context, roomName string, agentName string, metadata *livekit.AgentDispatchMetadata) (*livekit.AgentDispatch, error)

	// GenerateToken creates an access token for a participant to join a room.
	GenerateToken(ctx context.Context, roomName string, participantIdentity string, opts *livekit.TokenOptions) (string, error)
}

// DefaultLiveKitConnector returns the default LiveKit connector implementation.
func DefaultLiveKitConnector(ctx context.Context) (LiveKitConnector, error) {
	return DefaultLiveKitClientConnector(ctx)
}

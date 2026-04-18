package livekit

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"sync"
	"time"

	livekitschemas "github.com/cannonball10/foundation/schemas/livekit"
	"github.com/livekit/protocol/auth"
	livekitproto "github.com/livekit/protocol/livekit"
	lksdk "github.com/livekit/server-sdk-go/v2"
)

// LiveKitClient is the default implementation of LiveKitConnector.
type LiveKitClient struct {
	url       string
	apiKey    string
	apiSecret string

	roomClient  *lksdk.RoomServiceClient
	agentClient *lksdk.AgentDispatchClient
	clientOnce  sync.Once
	clientErr   error
}

// NewLiveKitClient creates a new LiveKitClient with the given credentials.
func NewLiveKitClient(url, apiKey, apiSecret string) *LiveKitClient {
	return &LiveKitClient{
		url:       url,
		apiKey:    apiKey,
		apiSecret: apiSecret,
	}
}

// NewLiveKitClientLazy creates a new LiveKitClient without initializing the SDK clients.
// The clients are initialized on first use.
func NewLiveKitClientLazy(url, apiKey, apiSecret string) *LiveKitClient {
	return &LiveKitClient{
		url:       url,
		apiKey:    apiKey,
		apiSecret: apiSecret,
	}
}

// DefaultLiveKitClientConnector creates a LiveKitClient using environment variables.
// Required environment variables:
//   - LIVEKIT_URL: The LiveKit server URL (e.g., ws://localhost:7880 or wss://your-server.com)
//   - LIVEKIT_API_KEY: The API key for authentication
//   - LIVEKIT_API_SECRET: The API secret for authentication
func DefaultLiveKitClientConnector(ctx context.Context) (*LiveKitClient, error) {
	url := os.Getenv("LIVEKIT_URL")
	apiKey := os.Getenv("LIVEKIT_API_KEY")
	apiSecret := os.Getenv("LIVEKIT_API_SECRET")

	if url == "" || apiKey == "" || apiSecret == "" {
		slog.WarnContext(ctx, "LiveKit environment variables not set, connector will be non-functional",
			"url_set", url != "",
			"api_key_set", apiKey != "",
			"api_secret_set", apiSecret != "")
	}

	return NewLiveKitClientLazy(url, apiKey, apiSecret), nil
}

// initClients initializes the SDK clients on first use.
func (c *LiveKitClient) initClients() error {
	c.clientOnce.Do(func() {
		if c.url == "" || c.apiKey == "" || c.apiSecret == "" {
			c.clientErr = errors.New("livekit credentials not configured")
			return
		}

		c.roomClient = lksdk.NewRoomServiceClient(c.url, c.apiKey, c.apiSecret)
		c.agentClient = lksdk.NewAgentDispatchServiceClient(c.url, c.apiKey, c.apiSecret)
	})
	return c.clientErr
}

// CreateRoom creates a new LiveKit room.
func (c *LiveKitClient) CreateRoom(ctx context.Context, roomName string, opts *livekitschemas.RoomOptions) (*livekitschemas.Room, error) {
	if err := c.initClients(); err != nil {
		return nil, fmt.Errorf("livekit client init: %w", err)
	}

	req := &livekitproto.CreateRoomRequest{
		Name: roomName,
	}

	if opts != nil {
		if opts.EmptyTimeout > 0 {
			req.EmptyTimeout = uint32(opts.EmptyTimeout.Seconds())
		}
		if opts.MaxParticipants > 0 {
			req.MaxParticipants = opts.MaxParticipants
		}
		if opts.Metadata != "" {
			req.Metadata = opts.Metadata
		}
	}

	room, err := c.roomClient.CreateRoom(ctx, req)
	if err != nil {
		slog.ErrorContext(ctx, "failed to create livekit room", "room", roomName, "error", err)
		return nil, fmt.Errorf("create room: %w", err)
	}

	return protoRoomToSchema(room), nil
}

// GetRoom retrieves information about an existing room.
func (c *LiveKitClient) GetRoom(ctx context.Context, roomName string) (*livekitschemas.Room, error) {
	if err := c.initClients(); err != nil {
		return nil, fmt.Errorf("livekit client init: %w", err)
	}

	rooms, err := c.roomClient.ListRooms(ctx, &livekitproto.ListRoomsRequest{
		Names: []string{roomName},
	})
	if err != nil {
		slog.ErrorContext(ctx, "failed to get livekit room", "room", roomName, "error", err)
		return nil, fmt.Errorf("get room: %w", err)
	}

	if len(rooms.Rooms) == 0 {
		return nil, nil
	}

	return protoRoomToSchema(rooms.Rooms[0]), nil
}

// DeleteRoom removes a room and disconnects all participants.
func (c *LiveKitClient) DeleteRoom(ctx context.Context, roomName string) error {
	if err := c.initClients(); err != nil {
		return fmt.Errorf("livekit client init: %w", err)
	}

	_, err := c.roomClient.DeleteRoom(ctx, &livekitproto.DeleteRoomRequest{
		Room: roomName,
	})
	if err != nil {
		slog.ErrorContext(ctx, "failed to delete livekit room", "room", roomName, "error", err)
		return fmt.Errorf("delete room: %w", err)
	}

	return nil
}

// DispatchAgent dispatches an agent to a room with the given metadata.
func (c *LiveKitClient) DispatchAgent(ctx context.Context, roomName string, agentName string, metadata *livekitschemas.AgentDispatchMetadata) (*livekitschemas.AgentDispatch, error) {
	if err := c.initClients(); err != nil {
		return nil, fmt.Errorf("livekit client init: %w", err)
	}

	var metadataJSON string
	if metadata != nil {
		data, err := json.Marshal(metadata)
		if err != nil {
			return nil, fmt.Errorf("marshal metadata: %w", err)
		}
		metadataJSON = string(data)
	}

	dispatch, err := c.agentClient.CreateDispatch(ctx, &livekitproto.CreateAgentDispatchRequest{
		Room:      roomName,
		AgentName: agentName,
		Metadata:  metadataJSON,
	})
	if err != nil {
		slog.ErrorContext(ctx, "failed to dispatch livekit agent", "room", roomName, "agent", agentName, "error", err)
		return nil, fmt.Errorf("dispatch agent: %w", err)
	}

	return &livekitschemas.AgentDispatch{
		AgentID:   dispatch.Id,
		RoomName:  roomName,
		AgentName: agentName,
	}, nil
}

// GenerateToken creates an access token for a participant to join a room.
func (c *LiveKitClient) GenerateToken(ctx context.Context, roomName string, participantIdentity string, opts *livekitschemas.TokenOptions) (string, error) {
	if c.apiKey == "" || c.apiSecret == "" {
		return "", errors.New("livekit credentials not configured")
	}

	at := auth.NewAccessToken(c.apiKey, c.apiSecret)

	// Set default TTL if not specified
	ttl := 24 * time.Hour
	if opts != nil && opts.TTL > 0 {
		ttl = opts.TTL
	}
	at.SetValidFor(ttl)

	grant := &auth.VideoGrant{
		Room:     roomName,
		RoomJoin: true,
	}

	if opts != nil {
		grant.CanPublish = &opts.CanPublish
		grant.CanSubscribe = &opts.CanSubscribe
		grant.CanPublishData = &opts.CanPublishData
	} else {
		// Default permissions for text chat
		canPublish := true
		canSubscribe := true
		canPublishData := true
		grant.CanPublish = &canPublish
		grant.CanSubscribe = &canSubscribe
		grant.CanPublishData = &canPublishData
	}

	at.SetVideoGrant(grant)
	at.SetIdentity(participantIdentity)

	if opts != nil && opts.Metadata != "" {
		at.SetMetadata(opts.Metadata)
	}

	token, err := at.ToJWT()
	if err != nil {
		slog.ErrorContext(ctx, "failed to generate livekit token", "room", roomName, "identity", participantIdentity, "error", err)
		return "", fmt.Errorf("generate token: %w", err)
	}

	return token, nil
}

// Ping verifies the LiveKit connection is healthy.
func (c *LiveKitClient) Ping(ctx context.Context) error {
	if err := c.initClients(); err != nil {
		return err
	}

	// Try to list rooms as a health check
	_, err := c.roomClient.ListRooms(ctx, &livekitproto.ListRoomsRequest{})
	if err != nil {
		return fmt.Errorf("livekit health check failed: %w", err)
	}
	return nil
}

// Close cleans up any resources. Currently a no-op as the SDK clients don't require explicit cleanup.
func (c *LiveKitClient) Close() error {
	return nil
}

// protoRoomToSchema converts a LiveKit protocol Room to our schema Room.
func protoRoomToSchema(r *livekitproto.Room) *livekitschemas.Room {
	return &livekitschemas.Room{
		Name:            r.Name,
		SID:             r.Sid,
		NumParticipants: r.NumParticipants,
		MaxParticipants: r.MaxParticipants,
		CreationTime:    time.Unix(r.CreationTime, 0),
		Metadata:        r.Metadata,
	}
}

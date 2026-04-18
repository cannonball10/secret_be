package game

import (
	"context"
	"log/slog"

	"github.com/cannonball10/foundation/models"
)

// Envelope is the wire-format wrapper for every event. Transports
// (WebSocket, SSE, Redis pub-sub, FCM, etc.) serialize this struct
// and deliver it to subscribers.
//
// An Envelope is produced from a GameEvent plus an audience scope:
//   - Broadcast   : visible to every player in the game
//   - PlayerScope : visible only to the named player (secret information
//     such as drawn policies, investigation results, private role)
type Envelope struct {
	GameID   string            `json:"gameId"`
	Event    *models.GameEvent `json:"event"`
	Audience Audience          `json:"audience"`
	// Payload carries decoded event-specific data so the client does
	// not need to cast models.GameEvent.Data.
	Payload any `json:"payload,omitempty"`
}

// Audience describes who should receive a given Envelope.
type Audience struct {
	// Scope is "broadcast" (everyone) or "player" (single recipient).
	Scope AudienceScope `json:"scope"`
	// PlayerID is set when Scope is AudiencePlayer.
	PlayerID string `json:"playerId,omitempty"`
}

// AudienceScope enumerates the delivery scopes.
type AudienceScope string

const (
	AudienceBroadcast AudienceScope = "broadcast"
	AudiencePlayer    AudienceScope = "player"
)

// Emitter is the transport-agnostic event sink for the game engine.
// Implementations fan out Envelopes to connected clients. The engine
// calls Emit once per state change; emission is best-effort and must
// not block or fail the transition (errors are logged, not returned).
type Emitter interface {
	Emit(ctx context.Context, env Envelope)
}

// NoopEmitter discards every event. It is the default for GameHandler
// so callers can build unit tests without wiring a real transport.
type NoopEmitter struct{}

// Emit implements Emitter.
func (NoopEmitter) Emit(_ context.Context, _ Envelope) {}

// SlogEmitter logs each envelope at info level. Handy during local
// development; swap for a real transport in production.
type SlogEmitter struct {
	Logger *slog.Logger
}

// Emit implements Emitter.
func (s SlogEmitter) Emit(ctx context.Context, env Envelope) {
	logger := s.Logger
	if logger == nil {
		logger = slog.Default()
	}
	logger.InfoContext(ctx, "game event",
		"gameId", env.GameID,
		"type", env.Event.Type,
		"scope", env.Audience.Scope,
		"playerId", env.Audience.PlayerID,
	)
}

// MultiEmitter fans out to multiple emitters so (for example) logging
// and a real transport can coexist.
type MultiEmitter []Emitter

// Emit implements Emitter.
func (m MultiEmitter) Emit(ctx context.Context, env Envelope) {
	for _, e := range m {
		e.Emit(ctx, env)
	}
}

// broadcast is the engine's internal helper to build and dispatch a
// broadcast envelope.
func (h *GameHandler) broadcast(ctx context.Context, event *models.GameEvent, payload any) {
	h.emitter.Emit(ctx, Envelope{
		GameID:   event.GameID,
		Event:    event,
		Audience: Audience{Scope: AudienceBroadcast},
		Payload:  payload,
	})
}

// whisper sends an envelope to a single player. Used for private
// information: role reveals on game start, drawn policies for the
// president, investigation results, policy peek, etc.
func (h *GameHandler) whisper(ctx context.Context, event *models.GameEvent, playerID string, payload any) {
	h.emitter.Emit(ctx, Envelope{
		GameID:   event.GameID,
		Event:    event,
		Audience: Audience{Scope: AudiencePlayer, PlayerID: playerID},
		Payload:  payload,
	})
}

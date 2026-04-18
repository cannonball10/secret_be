package models

import (
	"github.com/cannonball10/foundation/schemas/secrethitler"
	"github.com/cannonball10/foundation/utils"
)

// GameEventKeys provides key construction for the GameEvent model.
// PK: GAME#{gameId}, SK: EVENT#{ulid}
// Using a ULID sort key gives the event log a stable chronological order.
var GameEventKeys = NewKeyBuilder("GAME#", "EVENT#")

// GameEvent is a single entry in the ordered event log for a game. The
// event log is the source of truth for replaying a match and for building
// a player's passport after the game ends.
type GameEvent struct {
	Timestamps

	EventID string                 `json:"eventId"`
	GameID  string                 `json:"gameId"`
	Type    secrethitler.EventType `json:"type"`
	// ActorPlayerID is the player who caused this event (president,
	// voter, chancellor, etc). Empty for system events.
	ActorPlayerID string `json:"actorPlayerId,omitempty"`
	// TargetPlayerID is the player the action affected, if any.
	TargetPlayerID string `json:"targetPlayerId,omitempty"`
	// Data is an opaque event-specific payload; events that need
	// typed access should add their own fields to a parsed struct.
	Data map[string]any `json:"data,omitempty"`
}

// NewGameEvent creates a GameEvent with an auto-generated ULID.
func NewGameEvent(gameID string, eventType secrethitler.EventType, actorPlayerID string) *GameEvent {
	return &GameEvent{
		Timestamps:    NewTimestamps(),
		EventID:       utils.GenerateULID(),
		GameID:        gameID,
		Type:          eventType,
		ActorPlayerID: actorPlayerID,
	}
}

func (e *GameEvent) PK() string { return GameEventKeys.PK(e.GameID) }
func (e *GameEvent) SK() string { return GameEventKeys.SK(e.EventID) }

func (e *GameEvent) GSIs() map[int]GSIKeyPair { return nil }

// WithTarget sets the target player and returns the event for chaining.
func (e *GameEvent) WithTarget(targetPlayerID string) *GameEvent {
	e.TargetPlayerID = targetPlayerID
	return e
}

// WithData sets the event payload and returns the event for chaining.
func (e *GameEvent) WithData(data map[string]any) *GameEvent {
	e.Data = data
	return e
}

func init() {
	RegisterModel(GameEventKeys, func() Model { return &GameEvent{} })
}

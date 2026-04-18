// Package game implements the Secret Hitler game state machine. A
// GameHandler owns the rules for every transition; every method is the
// single authoritative way to change game state and always emits one or
// more events so connected devices can re-render.
package game

import (
	"github.com/cannonball10/foundation/connectors/database"
)

// GameHandler is the root of the game state machine.
//
// The handler is intentionally stateless: every mutation loads the
// latest Game + related records from the database, validates the
// transition, writes back, and emits events. Concurrency is handled at
// the DB layer (per-game writes should use conditional expressions on
// Updated/Version in a real deployment).
//
// The handler depends on a DatabaseConnector rather than the full
// connectors.Connectors struct so it can be unit-tested without wiring
// every other connector in the system.
type GameHandler struct {
	db      database.DatabaseConnector
	emitter Emitter
	clock   Clock
	rng     RNG
	config  Config
}

// Option configures a GameHandler at construction time.
type Option func(*GameHandler)

// WithEmitter overrides the default (no-op) event emitter.
func WithEmitter(e Emitter) Option {
	return func(h *GameHandler) { h.emitter = e }
}

// WithClock overrides the default real-time clock (useful in tests).
func WithClock(c Clock) Option {
	return func(h *GameHandler) { h.clock = c }
}

// WithRNG overrides the default crypto-backed RNG.
func WithRNG(r RNG) Option {
	return func(h *GameHandler) { h.rng = r }
}

// WithConfig overrides the default timer/config values.
func WithConfig(c Config) Option {
	return func(h *GameHandler) { h.config = c }
}

// NewGameHandler constructs a GameHandler with sensible defaults.
// Callers will typically override the Emitter with their transport
// (WebSocket/SSE/Redis pub-sub) via WithEmitter.
func NewGameHandler(db database.DatabaseConnector, opts ...Option) *GameHandler {
	h := &GameHandler{
		db:      db,
		emitter: NoopEmitter{},
		clock:   RealClock{},
		rng:     NewCryptoRNG(),
		config:  DefaultConfig(),
	}
	for _, opt := range opts {
		opt(h)
	}
	return h
}

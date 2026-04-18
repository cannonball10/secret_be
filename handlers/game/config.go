package game

import "time"

// Config holds tunable timeouts for the game engine. Each phase has a
// deadline; the scheduler (or a host-initiated force-progress call)
// triggers TimerExpired when the clock advances past it.
type Config struct {
	NominationTimeout  time.Duration
	ElectionTimeout    time.Duration
	LegislativeTimeout time.Duration
	ExecutiveTimeout   time.Duration
	VetoTimeout        time.Duration
}

// DefaultConfig returns reasonable defaults for a casual mobile game.
// Tweak per deployment via WithConfig.
func DefaultConfig() Config {
	return Config{
		NominationTimeout:  60 * time.Second,
		ElectionTimeout:    60 * time.Second,
		LegislativeTimeout: 45 * time.Second,
		ExecutiveTimeout:   45 * time.Second,
		VetoTimeout:        30 * time.Second,
	}
}

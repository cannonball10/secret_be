package replicant

import "fmt"

// RulesConfig parameterises every scalar the game engine reads at
// runtime. It is stamped onto the Game at creation time and never
// mutated afterwards — so if the host starts a short-game variant,
// all subsequent engine reads reflect that choice without having to
// pipe the override through every call site.
//
// The zero value is NOT a legal config (Validate will reject it).
// Call DefaultRules() to get a playable baseline. Unknown / future
// fields are added by appending to this struct; loadGame() upgrades
// old records via ensureRules().
type RulesConfig struct {
	// Seating bounds. These are inclusive — a 5-player game is legal
	// at MinPlayers=5. Must lie within [PackageMinPlayers, PackageMaxPlayers].
	MinPlayers int `json:"minPlayers"`
	MaxPlayers int `json:"maxPlayers"`

	// Deck composition. The shuffle at game start draws exactly
	// HumanProtocolsInDeck + AIProtocolsInDeck policies.
	HumanProtocolsInDeck int `json:"humanProtocolsInDeck"`
	AIProtocolsInDeck    int `json:"aiProtocolsInDeck"`

	// Win thresholds. A party wins when its policy count reaches or
	// exceeds these values.
	HumanPoliciesToWin int `json:"humanPoliciesToWin"`
	AIPoliciesToWin    int `json:"aiPoliciesToWin"`

	// CodesTransferAt is the count of enacted AI policies after which
	// electing the Prime (rogue) as Chancellor triggers an instant
	// AI-faction win. In the standard ruleset this is 3.
	CodesTransferAt int `json:"codesTransferAt"`

	// ElectionTrackerLimit is the number of consecutive failed
	// elections that trigger a top-deck enactment and reset term
	// limits. Vanilla: 3.
	ElectionTrackerLimit int `json:"electionTrackerLimit"`

	// VetoUnlockAt is the count of enacted AI policies after which
	// the Chancellor may propose vetoing the legislative agenda.
	// Vanilla: 5.
	VetoUnlockAt int `json:"vetoUnlockAt"`

	// EnableSingularity toggles the three-faction variant. When true,
	// StartGame seats one Singularity player (drawn from the liberal
	// allotment, see SingularityDistribution). The Singularity wins
	// alone if elected Chancellor after CodesTransferAt AI policies —
	// a kingmaker condition that competes with the rogue-chancellor
	// win. False preserves the vanilla 2-faction ruleset.
	EnableSingularity bool `json:"enableSingularity"`

	// CablePhaseMode controls when a Cable Phase interstitial runs
	// between chancellor nomination and the election vote. The phase
	// is where live chat (DMs, cabal cables) happens on-device; at
	// phase-end the server picks one cable to leak via the narrator.
	// Values: "disabled" (vanilla), "every_round" (default when the
	// host opts in), "on_activation" (reserved for a wiretap-style
	// power that only fires the phase when triggered).
	CablePhaseMode CablePhaseMode `json:"cablePhaseMode"`

	// CablePhaseDurationSec is how long the Cable Phase stays open
	// before the engine auto-advances to the election vote. Only
	// consulted when CablePhaseMode is non-disabled.
	CablePhaseDurationSec int `json:"cablePhaseDurationSec"`

	// CableLeakSilenceChance is the probability [0, 1] that the
	// Committee will go silent at phase-end — i.e. broadcast "no
	// traffic worth flagging" instead of leaking the top-scored
	// cable. Small non-zero values (~0.1-0.2) are the sweet spot:
	// enough uncertainty to keep players guessing whether their
	// silence means safety or just luck. 0 = always leak; 1 = always
	// silent.
	CableLeakSilenceChance float64 `json:"cableLeakSilenceChance"`

	// AnonymousCableLeaks, when true (default), strips the author from
	// the cable_leaked broadcast so the Committee's transmission lands
	// without attribution — the table has to guess who wrote it. When
	// false, the author is named explicitly, making leaks a direct
	// outing mechanism rather than a guessing game.
	AnonymousCableLeaks bool `json:"anonymousCableLeaks"`

	// DisableNarrator, when true, suppresses every narrator_speak
	// envelope the engine would otherwise fire (openings, closings,
	// execution eulogies, cable leak announcements). Useful when a
	// host wants a silent game, or when the Anthropic/ElevenLabs
	// budget is a concern. Default false.
	DisableNarrator bool `json:"disableNarrator"`
}

// PackageMinPlayers and PackageMaxPlayers are absolute bounds that
// apply regardless of RulesConfig — the role-distribution and
// executive-power tables only cover these counts, so the engine
// cannot legally seat games outside them.
const (
	PackageMinPlayers = 5
	PackageMaxPlayers = 10
)

// DefaultRules returns the vanilla Replicant ruleset. This is what
// NewGame stamps when no overrides are supplied, and what loadGame
// substitutes into records that predate the Rules field.
func DefaultRules() RulesConfig {
	return RulesConfig{
		MinPlayers:            PackageMinPlayers,
		MaxPlayers:            PackageMaxPlayers,
		HumanProtocolsInDeck:  6,
		AIProtocolsInDeck:     11,
		HumanPoliciesToWin:    5,
		AIPoliciesToWin:       6,
		CodesTransferAt:       3,
		ElectionTrackerLimit:  3,
		VetoUnlockAt:          5,
		CablePhaseMode:         CableModeDisabled,
		CablePhaseDurationSec:  45,
		CableLeakSilenceChance: 0.15,
		AnonymousCableLeaks:    true,
	}
}

// IsZero reports whether r is the zero-value RulesConfig — used by
// the engine to decide whether to substitute defaults when loading a
// pre-RulesConfig Game record.
func (r RulesConfig) IsZero() bool {
	return r == RulesConfig{}
}

// Validate returns a non-nil error describing the first invariant
// violation found, or nil if the config is internally consistent.
// The engine calls this at CreateGame to reject bogus overrides up
// front rather than discovering them mid-game.
func (r RulesConfig) Validate() error {
	if r.MinPlayers < PackageMinPlayers {
		return fmt.Errorf("MinPlayers=%d below package minimum %d", r.MinPlayers, PackageMinPlayers)
	}
	if r.MaxPlayers > PackageMaxPlayers {
		return fmt.Errorf("MaxPlayers=%d above package maximum %d", r.MaxPlayers, PackageMaxPlayers)
	}
	if r.MinPlayers > r.MaxPlayers {
		return fmt.Errorf("MinPlayers=%d > MaxPlayers=%d", r.MinPlayers, r.MaxPlayers)
	}
	if r.HumanProtocolsInDeck <= 0 || r.AIProtocolsInDeck <= 0 {
		return fmt.Errorf("deck must have positive protocol counts (got %d human, %d ai)",
			r.HumanProtocolsInDeck, r.AIProtocolsInDeck)
	}
	// The deck must contain enough cards to support both win thresholds —
	// otherwise a game can exhaust the draw pile before either side wins.
	if r.HumanProtocolsInDeck < r.HumanPoliciesToWin {
		return fmt.Errorf("deck has %d human protocols but %d needed to win",
			r.HumanProtocolsInDeck, r.HumanPoliciesToWin)
	}
	if r.AIProtocolsInDeck < r.AIPoliciesToWin {
		return fmt.Errorf("deck has %d AI protocols but %d needed to win",
			r.AIProtocolsInDeck, r.AIPoliciesToWin)
	}
	if r.HumanPoliciesToWin <= 0 || r.AIPoliciesToWin <= 0 {
		return fmt.Errorf("win thresholds must be positive (got %d/%d)",
			r.HumanPoliciesToWin, r.AIPoliciesToWin)
	}
	if r.CodesTransferAt < 0 || r.CodesTransferAt >= r.AIPoliciesToWin {
		return fmt.Errorf("CodesTransferAt=%d must satisfy 0 <= x < AIPoliciesToWin=%d",
			r.CodesTransferAt, r.AIPoliciesToWin)
	}
	if r.VetoUnlockAt < 0 || r.VetoUnlockAt > r.AIPoliciesToWin {
		return fmt.Errorf("VetoUnlockAt=%d must satisfy 0 <= x <= AIPoliciesToWin=%d",
			r.VetoUnlockAt, r.AIPoliciesToWin)
	}
	if r.ElectionTrackerLimit <= 0 {
		return fmt.Errorf("ElectionTrackerLimit=%d must be positive", r.ElectionTrackerLimit)
	}
	if r.EnableSingularity && r.MinPlayers < 6 {
		// 5-player games only have 3 liberal seats in the vanilla
		// distribution; subtracting one for a Singularity leaves the
		// humans outnumbered 2-vs-3. Require 6+ so at least one human
		// still survives the swap.
		return fmt.Errorf("EnableSingularity requires MinPlayers >= 6 (got %d)", r.MinPlayers)
	}
	switch r.CablePhaseMode {
	case CableModeDisabled, CableModeEveryRound, CableModeOnActivation:
		// known modes — ok
	default:
		return fmt.Errorf("unknown CablePhaseMode %q", r.CablePhaseMode)
	}
	if r.CablePhaseMode != CableModeDisabled && r.CablePhaseDurationSec <= 0 {
		return fmt.Errorf("CablePhaseDurationSec must be positive when CablePhaseMode=%q (got %d)",
			r.CablePhaseMode, r.CablePhaseDurationSec)
	}
	if r.CableLeakSilenceChance < 0 || r.CableLeakSilenceChance > 1 {
		return fmt.Errorf("CableLeakSilenceChance must be in [0, 1] (got %f)", r.CableLeakSilenceChance)
	}
	return nil
}

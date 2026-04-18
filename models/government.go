package models

import (
	"github.com/cannonball10/foundation/schemas/secrethitler"
	"github.com/cannonball10/foundation/utils"
)

// GovernmentKeys provides key construction for the Government model.
// PK: GAME#{gameId}, SK: GOV#{governmentId}
// ULID-based governmentId makes the SK time-sortable so governments
// can be listed chronologically with a begins_with("GOV#") query.
var GovernmentKeys = NewKeyBuilder("GAME#", "GOV#")

// Government records a nominated president/chancellor pair and what
// happened during their legislative session. A new Government is
// created at the start of each nomination phase.
type Government struct {
	Timestamps

	GovernmentID       string `json:"governmentId"`
	GameID             string `json:"gameId"`
	Round              int    `json:"round"`
	PresidentPlayerID  string `json:"presidentPlayerId"`
	ChancellorPlayerID string `json:"chancellorPlayerId,omitempty"`
	PresidentSeat      int    `json:"presidentSeat"`
	ChancellorSeat     *int   `json:"chancellorSeat,omitempty"`

	Status secrethitler.GovernmentStatus `json:"status"`

	// IsSpecialElection is true when the president was chosen via the
	// Special Election executive action rather than normal rotation.
	IsSpecialElection bool `json:"isSpecialElection"`

	// Vote outcome counts populated once the election resolves.
	JaVotes   int `json:"jaVotes"`
	NeinVotes int `json:"neinVotes"`

	// DrawnPolicies are the three policies drawn by the president after
	// a successful election, in draw order.
	DrawnPolicies []secrethitler.PolicyType `json:"-"`
	// PresidentDiscarded is the policy the president discarded.
	PresidentDiscarded *secrethitler.PolicyType `json:"-"`
	// ChancellorOptions is the set of two policies passed to the chancellor.
	ChancellorOptions []secrethitler.PolicyType `json:"-"`
	// EnactedPolicy is the policy the chancellor enacted (nil if vetoed
	// or not yet enacted).
	EnactedPolicy *secrethitler.PolicyType `json:"enactedPolicy,omitempty"`

	// VetoProposed and VetoAccepted track an optional veto flow once
	// 5 fascist policies are on the board.
	VetoProposed bool `json:"vetoProposed"`
	VetoAccepted bool `json:"vetoAccepted"`
}

// NewGovernment creates a Government in the proposed state.
func NewGovernment(id *string, gameID string, round int, presidentPlayerID string, presidentSeat int) *Government {
	if id == nil || *id == "" {
		ulid := utils.GenerateULID()
		id = &ulid
	}
	return &Government{
		Timestamps:        NewTimestamps(),
		GovernmentID:      *id,
		GameID:            gameID,
		Round:             round,
		PresidentPlayerID: presidentPlayerID,
		PresidentSeat:     presidentSeat,
		Status:            secrethitler.GovernmentStatusProposed,
	}
}

func (g *Government) PK() string { return GovernmentKeys.PK(g.GameID) }
func (g *Government) SK() string { return GovernmentKeys.SK(g.GovernmentID) }

func (g *Government) GSIs() map[int]GSIKeyPair { return nil }

// NominateChancellor sets the chancellor for this government.
func (g *Government) NominateChancellor(playerID string, seat int) {
	g.ChancellorPlayerID = playerID
	g.ChancellorSeat = &seat
	g.Touch()
}

// RecordElection finalizes the vote and transitions status based on the tally.
func (g *Government) RecordElection(ja, nein int) {
	g.JaVotes = ja
	g.NeinVotes = nein
	if ja > nein {
		g.Status = secrethitler.GovernmentStatusPassed
	} else {
		g.Status = secrethitler.GovernmentStatusRejected
	}
	g.Touch()
}

// RecordEnactment records the final enacted policy.
func (g *Government) RecordEnactment(policy secrethitler.PolicyType) {
	g.EnactedPolicy = &policy
	g.Status = secrethitler.GovernmentStatusEnacted
	g.Touch()
}

// RecordVeto records that a veto was proposed and resolved.
func (g *Government) RecordVeto(accepted bool) {
	g.VetoProposed = true
	g.VetoAccepted = accepted
	if accepted {
		g.Status = secrethitler.GovernmentStatusVetoed
	}
	g.Touch()
}

func init() {
	RegisterModel(GovernmentKeys, func() Model { return &Government{} })
}

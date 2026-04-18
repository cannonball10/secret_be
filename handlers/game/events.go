package game

import (
	"github.com/cannonball10/foundation/schemas/secrethitler"
)

// ProgressReason names why the engine advanced a phase. It is attached
// to transition events so clients can render "cutoff", "forced", etc.
type ProgressReason string

const (
	ReasonAction   ProgressReason = "action"    // normal player/host action
	ReasonTimeout  ProgressReason = "timeout"   // deadline expired
	ReasonForced   ProgressReason = "forced"    // host forced progress
	ReasonAllVoted ProgressReason = "all_voted" // every alive player voted
)

// --- typed event payloads --------------------------------------------------
//
// Each struct below corresponds to a secrethitler.EventType. They are
// attached to the Envelope.Payload field so clients get typed data
// without needing to re-interpret the map[string]any on GameEvent.Data.

// GameCreatedPayload announces a new lobby.
type GameCreatedPayload struct {
	JoinCode   string `json:"joinCode"`
	HostUserID string `json:"hostUserId"`
}

// PlayerJoinedPayload announces a player entering the lobby.
type PlayerJoinedPayload struct {
	PlayerID    string `json:"playerId"`
	DisplayName string `json:"displayName"`
	Seat        int    `json:"seat"`
}

// GameStartedPayload announces that the game has begun. Seating order
// and the initial president are fixed by this event.
type GameStartedPayload struct {
	PlayerCount          int `json:"playerCount"`
	InitialPresidentSeat int `json:"initialPresidentSeat"`
}

// RoleAssignedPayload is whispered individually to each player so they
// learn their own role (and, for fascists/Hitler in small games, their
// teammates).
type RoleAssignedPayload struct {
	Role      secrethitler.Role  `json:"role"`
	Party     secrethitler.Party `json:"party"`
	Teammates []TeammateInfo     `json:"teammates,omitempty"`
}

// TeammateInfo is the private view a fascist has of the cabal at game
// start (or Hitler, in small games).
type TeammateInfo struct {
	PlayerID    string            `json:"playerId"`
	DisplayName string            `json:"displayName"`
	Role        secrethitler.Role `json:"role"`
}

// ChancellorNominatedPayload accompanies a chancellor nomination.
type ChancellorNominatedPayload struct {
	PresidentPlayerID  string `json:"presidentPlayerId"`
	ChancellorPlayerID string `json:"chancellorPlayerId"`
	GovernmentID       string `json:"governmentId"`
	Deadline           string `json:"deadline,omitempty"` // RFC3339
}

// VoteCastPayload is broadcast when a player votes. The choice is NOT
// included; only the fact that the player voted (so everyone can see
// the vote tally fill up without spoilers).
type VoteCastPayload struct {
	GovernmentID string `json:"governmentId"`
	PlayerID     string `json:"playerId"`
}

// ElectionResultPayload reports the final vote tally.
type ElectionResultPayload struct {
	GovernmentID string                             `json:"governmentId"`
	Passed       bool                               `json:"passed"`
	JaVotes      int                                `json:"jaVotes"`
	NeinVotes    int                                `json:"neinVotes"`
	Votes        map[string]secrethitler.VoteChoice `json:"votes"` // keyed by playerId
	Reason       ProgressReason                     `json:"reason"`
}

// PoliciesDrawnPayload is whispered to the president after a successful
// election so they see the three policies they must choose from.
type PoliciesDrawnPayload struct {
	GovernmentID string                    `json:"governmentId"`
	Policies     []secrethitler.PolicyType `json:"policies"`
}

// PresidentDiscardedPayload is broadcast (counts only) and whispered to
// the chancellor (with the two remaining policies).
type PresidentDiscardedPayload struct {
	GovernmentID string                    `json:"governmentId"`
	Options      []secrethitler.PolicyType `json:"options,omitempty"` // chancellor-only
}

// ChancellorEnactedPayload is broadcast when a policy is placed on the board.
type ChancellorEnactedPayload struct {
	GovernmentID           string                  `json:"governmentId"`
	Policy                 secrethitler.PolicyType `json:"policy"`
	LiberalPoliciesEnacted int                     `json:"liberalPoliciesEnacted"`
	FascistPoliciesEnacted int                     `json:"fascistPoliciesEnacted"`
}

// VetoProposedPayload announces the chancellor requested a veto.
type VetoProposedPayload struct {
	GovernmentID string `json:"governmentId"`
}

// VetoResolvedPayload announces the president's response.
type VetoResolvedPayload struct {
	GovernmentID string `json:"governmentId"`
	Accepted     bool   `json:"accepted"`
}

// ExecutiveActionPayload announces the use of a presidential power. For
// policy peek the actual policies are whispered separately to the
// president; for investigate, the target's party is whispered.
type ExecutiveActionPayload struct {
	ActionID          string                           `json:"actionId"`
	Type              secrethitler.ExecutiveActionType `json:"type"`
	PresidentPlayerID string                           `json:"presidentPlayerId"`
	TargetPlayerID    string                           `json:"targetPlayerId,omitempty"`
}

// InvestigateResultPayload is whispered to the president after an
// Investigate Loyalty action.
type InvestigateResultPayload struct {
	ActionID       string             `json:"actionId"`
	TargetPlayerID string             `json:"targetPlayerId"`
	Party          secrethitler.Party `json:"party"`
}

// PolicyPeekPayload is whispered to the president after a Policy Peek.
type PolicyPeekPayload struct {
	ActionID string                    `json:"actionId"`
	Policies []secrethitler.PolicyType `json:"policies"` // top 3, in order
}

// ElectionTrackerPayload accompanies a tracker advance after a failed
// election or veto.
type ElectionTrackerPayload struct {
	Tracker int `json:"tracker"`
}

// TopDeckPayload announces a policy enacted by election-tracker advance.
type TopDeckPayload struct {
	Policy                 secrethitler.PolicyType `json:"policy"`
	LiberalPoliciesEnacted int                     `json:"liberalPoliciesEnacted"`
	FascistPoliciesEnacted int                     `json:"fascistPoliciesEnacted"`
}

// DeckReshuffledPayload announces the draw+discard piles were combined
// and reshuffled (happens whenever the draw pile has < 3 cards).
type DeckReshuffledPayload struct {
	RemainingInDraw int `json:"remainingInDraw"`
}

// PlayerExecutedPayload announces a player was executed by the
// Execution power.
type PlayerExecutedPayload struct {
	PlayerID       string `json:"playerId"`
	WasHitler      bool   `json:"wasHitler"`
	ExecutedBySeat int    `json:"executedBySeat"`
}

// GameEndedPayload announces the final result.
type GameEndedPayload struct {
	Winner       secrethitler.Party        `json:"winner"`
	WinCondition secrethitler.WinCondition `json:"winCondition"`
}

// PhaseChangedPayload is emitted on every phase transition so clients
// can drive their UI from a single event stream. It pairs nicely with
// more specific events (e.g. ChancellorNominatedPayload) but stands on
// its own when the engine is advanced by a timer or host action.
type PhaseChangedPayload struct {
	From     secrethitler.GamePhase `json:"from"`
	To       secrethitler.GamePhase `json:"to"`
	Reason   ProgressReason         `json:"reason"`
	Deadline string                 `json:"deadline,omitempty"` // RFC3339
}

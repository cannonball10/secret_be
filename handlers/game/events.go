package game

import (
	"github.com/cannonball10/foundation/schemas/replicant"
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
// Each struct below corresponds to a replicant.EventType. They are
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
	Round                int `json:"round"`
}

// RoleAssignedPayload is whispered individually to each player so they
// learn their own role (and, for fascists/Hitler in small games, their
// teammates).
type RoleAssignedPayload struct {
	Role      replicant.Role  `json:"role"`
	Party     replicant.Party `json:"party"`
	Teammates []TeammateInfo     `json:"teammates,omitempty"`
}

// TeammateInfo is the private view a fascist has of the cabal at game
// start (or Hitler, in small games).
type TeammateInfo struct {
	PlayerID    string            `json:"playerId"`
	DisplayName string            `json:"displayName"`
	Role        replicant.Role `json:"role"`
}

// ChancellorNominatedPayload accompanies a chancellor nomination. Round
// is included so clients can display the round counter without fetching
// the full game snapshot; it changes exactly when a new round begins.
type ChancellorNominatedPayload struct {
	PresidentPlayerID  string `json:"presidentPlayerId"`
	ChancellorPlayerID string `json:"chancellorPlayerId"`
	GovernmentID       string `json:"governmentId"`
	Deadline           string `json:"deadline,omitempty"` // RFC3339
	Round              int    `json:"round"`
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
	Votes        map[string]replicant.VoteChoice `json:"votes"` // keyed by playerId
	Reason       ProgressReason                     `json:"reason"`
}

// PoliciesDrawnPayload is whispered to the president after a successful
// election so they see the three policies they must choose from.
type PoliciesDrawnPayload struct {
	GovernmentID string                    `json:"governmentId"`
	Policies     []replicant.PolicyType `json:"policies"`
}

// PresidentDiscardedPayload is broadcast (counts only) and whispered to
// the chancellor (with the two remaining policies).
type PresidentDiscardedPayload struct {
	GovernmentID string                    `json:"governmentId"`
	Options      []replicant.PolicyType `json:"options,omitempty"` // chancellor-only
}

// ChancellorEnactedPayload is broadcast when a policy is placed on the board.
type ChancellorEnactedPayload struct {
	GovernmentID           string                  `json:"governmentId"`
	Policy                 replicant.PolicyType `json:"policy"`
	HumanPoliciesEnacted int                     `json:"humanPoliciesEnacted"`
	AIPoliciesEnacted int                     `json:"aiPoliciesEnacted"`
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
	Type              replicant.ExecutiveActionType `json:"type"`
	PresidentPlayerID string                           `json:"presidentPlayerId"`
	TargetPlayerID    string                           `json:"targetPlayerId,omitempty"`
}

// InvestigateResultPayload is whispered to the president after an
// Investigate Loyalty action.
type InvestigateResultPayload struct {
	ActionID       string             `json:"actionId"`
	TargetPlayerID string             `json:"targetPlayerId"`
	Party          replicant.Party `json:"party"`
}

// PolicyPeekPayload is whispered to the president after a Policy Peek.
type PolicyPeekPayload struct {
	ActionID string                    `json:"actionId"`
	Policies []replicant.PolicyType `json:"policies"` // top 3, in order
}

// ElectionTrackerPayload accompanies a tracker advance after a failed
// election or veto.
type ElectionTrackerPayload struct {
	Tracker int `json:"tracker"`
}

// TopDeckPayload announces a policy enacted by election-tracker advance.
type TopDeckPayload struct {
	Policy                 replicant.PolicyType `json:"policy"`
	HumanPoliciesEnacted int                     `json:"humanPoliciesEnacted"`
	AIPoliciesEnacted int                     `json:"aiPoliciesEnacted"`
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
	WasRogue      bool   `json:"wasRogue"`
	ExecutedBySeat int    `json:"executedBySeat"`
}

// GameEndedPayload announces the final result.
type GameEndedPayload struct {
	Winner       replicant.Party        `json:"winner"`
	WinCondition replicant.WinCondition `json:"winCondition"`
}

// NarratorSpeakPayload carries a synthesised narrator utterance. The
// Committee "speaks" over a game-wide broadcast envelope; host
// devices play AudioURL (a data-URL embedding the MP3) and render
// Script as typewriter text. Mobile clients can optionally render the
// transcript without playing audio.
type NarratorSpeakPayload struct {
	CueID    string `json:"cueId"`
	Cue      string `json:"cue"`
	Script   string `json:"script"`
	AudioURL string `json:"audioUrl"`
	AudioID  string `json:"audioId"`
	Format   string `json:"format"`
	TookMS   int64  `json:"tookMs"`
}

// CablePhaseOpenedPayload announces the start of a Cable Phase. The
// deadline is the phase's absolute cutoff in RFC3339 — clients render
// a countdown from (deadline - now). Chat send controls should become
// live on receipt and lock again on cable_phase_closed.
type CablePhaseOpenedPayload struct {
	GovernmentID string `json:"governmentId"`
	Deadline     string `json:"deadline"`
}

// CablePhaseClosedPayload announces the end of a Cable Phase. Reason
// mirrors ProgressReason (timeout / forced / all_voted-equivalent).
// LeakedMessageID references the cable (if any) the Committee chose
// to broadcast; empty when the Committee went silent or no cables
// were submitted.
type CablePhaseClosedPayload struct {
	GovernmentID     string         `json:"governmentId"`
	Reason           ProgressReason `json:"reason"`
	LeakedMessageID  string         `json:"leakedMessageId,omitempty"`
	TotalSubmissions int            `json:"totalSubmissions"`
}

// CableLeakedPayload carries the full body of the cable the narrator
// just broadcast — including a flag indicating whether the Committee
// chose silence instead of a leak. When Silenced is true, Body and
// Author are empty; the narrator_speak envelope fired alongside
// announces the silence verbally.
type CableLeakedPayload struct {
	GovernmentID    string  `json:"governmentId"`
	Silenced        bool    `json:"silenced"`
	MessageID       string  `json:"messageId,omitempty"`
	AuthorPlayerID  string  `json:"authorPlayerId,omitempty"`
	Author          string  `json:"author,omitempty"`
	Body            string  `json:"body,omitempty"`
	SubversionScore float64 `json:"subversionScore,omitempty"`
}

// SingularityCableFeedPayload is whispered only to the Singularity at
// cable-phase-close: every cable submitted this round, anonymised.
// The kingmaker role's intel edge — full chatter, no attribution.
type SingularityCableFeedPayload struct {
	GovernmentID string                    `json:"governmentId"`
	Cables       []AnonymousCableFeedEntry `json:"cables"`
}

// AnonymousCableFeedEntry is one row of the Singularity feed: message
// id + body + (optional) subversion score, with no author.
type AnonymousCableFeedEntry struct {
	MessageID       string  `json:"messageId"`
	Body            string  `json:"body"`
	SubversionScore float64 `json:"subversionScore,omitempty"`
}

// PhaseChangedPayload is emitted on every phase transition so clients
// can drive their UI from a single event stream. It pairs nicely with
// more specific events (e.g. ChancellorNominatedPayload) but stands on
// its own when the engine is advanced by a timer or host action.
//
// PresidentSeat is the CURRENT president at the time of the transition.
// Clients use it to recover the seat after rotation (normal round
// rollover) and after a Special Election, neither of which carry a
// dedicated "president changed" event.
type PhaseChangedPayload struct {
	From          replicant.GamePhase `json:"from"`
	To            replicant.GamePhase `json:"to"`
	Reason        ProgressReason         `json:"reason"`
	Deadline      string                 `json:"deadline,omitempty"` // RFC3339
	PresidentSeat int                    `json:"presidentSeat"`
}

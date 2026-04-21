package game

import (
	"context"

	"github.com/cannonball10/foundation/models"
	"github.com/cannonball10/foundation/schemas/database"
	"github.com/cannonball10/foundation/schemas/replicant"
)

// ResumeSnapshot is everything a client needs to rehydrate mid-game
// after a refresh / cold reload without waiting for SSE envelopes to
// re-announce state it already could have known. The server is the
// only place all of this lives cohesively — the mobile/host clients
// build the same view incrementally from whispers + broadcasts, and
// any gap in that chain leaves them stuck on "The President is
// choosing" when they actually know perfectly well who the President
// is.
//
// Caller-scoped fields (My*) are populated only when the caller is
// the player they pertain to. A non-president asking about their own
// DrawnPolicies gets nil; the president does not get someone else's
// investigation result. This keeps the same endpoint safe to call
// from any authenticated device.
type ResumeSnapshot struct {
	Game       *models.Game       `json:"game"`
	Players    []*models.Player   `json:"players"`
	Government *GovernmentPublic  `json:"government,omitempty"`
	Votes      []VoterRecord      `json:"votes,omitempty"`
	Action     *ExecutiveActionPublic `json:"action,omitempty"`

	// Caller-scoped.
	Me                  *models.Player            `json:"me,omitempty"`
	MyVote              *replicant.VoteChoice     `json:"myVote,omitempty"`
	MyDrawnPolicies     []replicant.PolicyType    `json:"myDrawnPolicies,omitempty"`
	MyChancellorOptions []replicant.PolicyType    `json:"myChancellorOptions,omitempty"`
	MyPeekedPolicies    []replicant.PolicyType    `json:"myPeekedPolicies,omitempty"`
	MyInvestigation     *InvestigationFinding     `json:"myInvestigation,omitempty"`
}

// GovernmentPublic strips the private DrawnPolicies / ChancellorOptions
// that the base Government model hides via json:"-". Everyone at the
// table is allowed to see who the president and chancellor are, what
// the vote tallies were, and whether a veto was proposed/accepted.
type GovernmentPublic struct {
	GovernmentID       string                     `json:"governmentId"`
	Round              int                        `json:"round"`
	PresidentPlayerID  string                     `json:"presidentPlayerId"`
	ChancellorPlayerID string                     `json:"chancellorPlayerId,omitempty"`
	PresidentSeat      int                        `json:"presidentSeat"`
	ChancellorSeat     *int                       `json:"chancellorSeat,omitempty"`
	Status             replicant.GovernmentStatus `json:"status"`
	IsSpecialElection  bool                       `json:"isSpecialElection"`
	JaVotes            int                        `json:"jaVotes"`
	NeinVotes          int                        `json:"neinVotes"`
	EnactedPolicy      *replicant.PolicyType      `json:"enactedPolicy,omitempty"`
	VetoProposed       bool                       `json:"vetoProposed"`
	VetoAccepted       bool                       `json:"vetoAccepted"`
}

// VoterRecord is the public half of a Vote — who cast, without
// revealing their choice. The tally is on GovernmentPublic; the set
// of voters is what drives the "already voted" indicator on every
// other player's screen.
type VoterRecord struct {
	PlayerID string `json:"playerId"`
}

// ExecutiveActionPublic mirrors ExecutiveAction for broadcast safely
// (no peeked policies).
type ExecutiveActionPublic struct {
	ActionID          string                         `json:"actionId"`
	Type              replicant.ExecutiveActionType  `json:"type"`
	PresidentPlayerID string                         `json:"presidentPlayerId"`
	TargetPlayerID    string                         `json:"targetPlayerId,omitempty"`
	RevealedParty     replicant.Party                `json:"revealedParty,omitempty"`
	Completed         bool                           `json:"completed"`
}

// InvestigationFinding is the private result of an investigate-loyalty
// power, returned to the president who performed it.
type InvestigationFinding struct {
	TargetPlayerID string          `json:"targetPlayerId"`
	Party          replicant.Party `json:"party"`
}

// LoadResumeSnapshot assembles the full reconcile bundle for the
// caller. Not all branches execute on every call — an idle lobby
// returns just Game + Players; a mid-election returns Government +
// Votes; only a president mid-policy-peek receives MyPeekedPolicies.
func (h *GameHandler) LoadResumeSnapshot(ctx context.Context, gameID, userID string) (*ResumeSnapshot, error) {
	game, err := h.loadGame(ctx, gameID)
	if err != nil {
		return nil, err
	}
	players, err := h.loadPlayers(ctx, gameID)
	if err != nil {
		return nil, err
	}

	var me *models.Player
	for _, p := range players {
		if p.UserID == userID {
			me = p
			break
		}
	}

	// Clone + scrub. Spectators + other players never see roles for
	// an in-progress game. The caller's own Player is returned
	// unscrubbed via Me so the client can restore their role reveal.
	scrubbed := make([]*models.Player, 0, len(players))
	revealRoles := game.Status == replicant.GameStatusCompleted
	for _, p := range players {
		if revealRoles || (me != nil && p.PlayerID == me.PlayerID) {
			scrubbed = append(scrubbed, p)
			continue
		}
		cp := *p
		cp.Role = ""
		cp.Party = ""
		scrubbed = append(scrubbed, &cp)
	}

	snap := &ResumeSnapshot{
		Game:    game,
		Players: scrubbed,
		Me:      me,
	}

	if game.CurrentGovernmentID == "" {
		return snap, nil
	}
	gov, err := h.loadGovernment(ctx, gameID, game.CurrentGovernmentID)
	if err != nil {
		// A missing government mid-game is recoverable — just skip
		// the gov-scoped fields and let the client render lobby-ish
		// waiting state rather than 500.
		return snap, nil
	}
	snap.Government = &GovernmentPublic{
		GovernmentID:       gov.GovernmentID,
		Round:              gov.Round,
		PresidentPlayerID:  gov.PresidentPlayerID,
		ChancellorPlayerID: gov.ChancellorPlayerID,
		PresidentSeat:      gov.PresidentSeat,
		ChancellorSeat:     gov.ChancellorSeat,
		Status:             gov.Status,
		IsSpecialElection:  gov.IsSpecialElection,
		JaVotes:            gov.JaVotes,
		NeinVotes:          gov.NeinVotes,
		EnactedPolicy:      gov.EnactedPolicy,
		VetoProposed:       gov.VetoProposed,
		VetoAccepted:       gov.VetoAccepted,
	}

	votes, err := h.loadVotesForGovernment(ctx, gameID, gov.GovernmentID)
	if err == nil {
		for _, v := range votes {
			snap.Votes = append(snap.Votes, VoterRecord{PlayerID: v.PlayerID})
			if me != nil && v.PlayerID == me.PlayerID {
				choice := v.Choice
				snap.MyVote = &choice
			}
		}
	}

	// President-private: policy cards drawn but not yet discarded.
	if me != nil && gov.PresidentPlayerID == me.PlayerID &&
		game.Phase == replicant.PhaseLegislativePresident &&
		len(gov.DrawnPolicies) > 0 && gov.PresidentDiscarded == nil {
		snap.MyDrawnPolicies = append([]replicant.PolicyType(nil), gov.DrawnPolicies...)
	}
	// Chancellor-private: the two options after the president's
	// discard. Populated on the chancellor's device only.
	if me != nil && gov.ChancellorPlayerID == me.PlayerID &&
		game.Phase == replicant.PhaseLegislativeChancellor &&
		len(gov.ChancellorOptions) > 0 && gov.EnactedPolicy == nil {
		snap.MyChancellorOptions = append([]replicant.PolicyType(nil), gov.ChancellorOptions...)
	}

	if game.PendingActionID != "" {
		actM, err := h.db.Get(ctx, nil, database.Key{
			"PK": models.ExecutiveActionKeys.PK(gameID),
			"SK": models.ExecutiveActionKeys.SK(game.PendingActionID),
		})
		if err == nil && actM != nil {
			act := actM.(*models.ExecutiveAction)
			snap.Action = &ExecutiveActionPublic{
				ActionID:          act.ActionID,
				Type:              act.Type,
				PresidentPlayerID: act.PresidentPlayerID,
				TargetPlayerID:    act.TargetPlayerID,
				RevealedParty:     act.RevealedParty,
				Completed:         act.Completed,
			}
			// Peeked policies are only for the president who
			// performed the peek. Copy on return so we don't hand
			// out the engine's internal slice.
			if me != nil && act.PresidentPlayerID == me.PlayerID &&
				act.Type == replicant.ActionPolicyPeek &&
				len(act.PeekedPolicies) > 0 {
				snap.MyPeekedPolicies = append([]replicant.PolicyType(nil), act.PeekedPolicies...)
			}
			if me != nil && act.PresidentPlayerID == me.PlayerID &&
				act.Type == replicant.ActionInvestigateLoyalty &&
				act.TargetPlayerID != "" && act.RevealedParty != "" {
				snap.MyInvestigation = &InvestigationFinding{
					TargetPlayerID: act.TargetPlayerID,
					Party:          act.RevealedParty,
				}
			}
		}
	}

	return snap, nil
}

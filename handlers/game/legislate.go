package game

import (
	"context"
	"fmt"

	"github.com/cannonball10/foundation/models"
	"github.com/cannonball10/foundation/schemas/secrethitler"
)

// PresidentDiscard is invoked by the current president to discard one
// of the three drawn policies. Remaining two are whispered to the
// chancellor and the phase advances to LegislativeChancellor.
func (h *GameHandler) PresidentDiscard(ctx context.Context, gameID, presidentPlayerID string, discardIndex int) error {
	game, err := h.loadGame(ctx, gameID)
	if err != nil {
		return err
	}
	if err := mustPhase(game, secrethitler.PhaseLegislativePresident); err != nil {
		return err
	}
	players, err := h.loadPlayers(ctx, gameID)
	if err != nil {
		return err
	}
	president := findPlayerBySeat(players, game.PresidentSeat)
	if president == nil || president.PlayerID != presidentPlayerID {
		return ErrNotPresident
	}

	gov, err := h.loadGovernment(ctx, gameID, game.CurrentGovernmentID)
	if err != nil {
		return err
	}
	if len(gov.DrawnPolicies) != 3 {
		return fmt.Errorf("%w: expected 3 drawn policies", ErrInvalidTransition)
	}
	if discardIndex < 0 || discardIndex >= 3 {
		return fmt.Errorf("%w: discard index out of range", ErrInvalidTransition)
	}

	discarded := gov.DrawnPolicies[discardIndex]
	remaining := make([]secrethitler.PolicyType, 0, 2)
	for i, p := range gov.DrawnPolicies {
		if i == discardIndex {
			continue
		}
		remaining = append(remaining, p)
	}
	gov.PresidentDiscarded = &discarded
	gov.ChancellorOptions = remaining
	discardPolicy(game, discarded)

	if err := h.db.Upsert(ctx, nil, gov); err != nil {
		return err
	}

	// Broadcast discard count (no card identity).
	pubEv := models.NewGameEvent(gameID, secrethitler.EventPresidentDiscarded, president.PlayerID)
	h.broadcast(ctx, pubEv, PresidentDiscardedPayload{GovernmentID: gov.GovernmentID})

	// Whisper remaining policies to the chancellor.
	chancellor := findPlayerBySeat(players, *gov.ChancellorSeat)
	if chancellor != nil {
		whisperEv := models.NewGameEvent(gameID, secrethitler.EventPresidentDiscarded, chancellor.PlayerID)
		h.whisper(ctx, whisperEv, chancellor.PlayerID, PresidentDiscardedPayload{
			GovernmentID: gov.GovernmentID,
			Options:      remaining,
		})
	}

	h.setPhase(ctx, game, secrethitler.PhaseLegislativeChancellor, ReasonAction)
	return h.saveGame(ctx, game)
}

// ChancellorEnact is invoked by the chancellor to enact one of the two
// remaining policies. Triggers win-condition checks, executive powers,
// and rotation back to Nomination (or end of game).
func (h *GameHandler) ChancellorEnact(ctx context.Context, gameID, chancellorPlayerID string, enactIndex int) error {
	game, err := h.loadGame(ctx, gameID)
	if err != nil {
		return err
	}
	if err := mustPhase(game, secrethitler.PhaseLegislativeChancellor); err != nil {
		return err
	}
	players, err := h.loadPlayers(ctx, gameID)
	if err != nil {
		return err
	}
	if game.ChancellorSeat == nil {
		return ErrInvalidTransition
	}
	chancellor := findPlayerBySeat(players, *game.ChancellorSeat)
	if chancellor == nil || chancellor.PlayerID != chancellorPlayerID {
		return ErrNotChancellor
	}

	gov, err := h.loadGovernment(ctx, gameID, game.CurrentGovernmentID)
	if err != nil {
		return err
	}
	if enactIndex < 0 || enactIndex >= len(gov.ChancellorOptions) {
		return fmt.Errorf("%w: enact index out of range", ErrInvalidTransition)
	}
	enacted := gov.ChancellorOptions[enactIndex]
	// The other option is discarded.
	for i, p := range gov.ChancellorOptions {
		if i == enactIndex {
			continue
		}
		discardPolicy(game, p)
	}
	gov.RecordEnactment(enacted)
	if err := h.db.Upsert(ctx, nil, gov); err != nil {
		return err
	}

	return h.applyEnactedPolicy(ctx, game, gov, players, enacted, false)
}

// applyEnactedPolicy increments the board counters, writes an
// EnactedPolicy record, emits events, and decides the next phase.
// topDeck=true signals this enactment came from the election tracker.
func (h *GameHandler) applyEnactedPolicy(ctx context.Context, game *models.Game, gov *models.Government, players []*models.Player, policy secrethitler.PolicyType, topDeck bool) error {
	if policy == secrethitler.PolicyHuman {
		game.HumanPoliciesEnacted++
	} else {
		game.AIPoliciesEnacted++
	}
	if game.AIPoliciesEnacted >= game.Rules.VetoUnlockAt {
		game.VetoUnlocked = true
	}
	// A successfully enacted policy (by government or by top-deck) clears
	// the election tracker per rulebook.
	game.ElectionTracker = 0

	sequence := game.HumanPoliciesEnacted + game.AIPoliciesEnacted
	governmentID := ""
	if gov != nil {
		governmentID = gov.GovernmentID
	}
	record := models.NewEnactedPolicy(game.GameID, sequence, policy, governmentID, topDeck)
	if err := h.db.Upsert(ctx, nil, record); err != nil {
		return err
	}

	if topDeck {
		ev := models.NewGameEvent(game.GameID, secrethitler.EventTopDeckEnacted, "")
		h.broadcast(ctx, ev, TopDeckPayload{
			Policy:                 policy,
			HumanPoliciesEnacted: game.HumanPoliciesEnacted,
			AIPoliciesEnacted: game.AIPoliciesEnacted,
		})
	} else {
		actor := ""
		if gov != nil {
			if chancellor := findPlayerBySeat(players, *gov.ChancellorSeat); chancellor != nil {
				actor = chancellor.PlayerID
			}
		}
		ev := models.NewGameEvent(game.GameID, secrethitler.EventChancellorEnacted, actor)
		h.broadcast(ctx, ev, ChancellorEnactedPayload{
			GovernmentID:           governmentID,
			Policy:                 policy,
			HumanPoliciesEnacted: game.HumanPoliciesEnacted,
			AIPoliciesEnacted: game.AIPoliciesEnacted,
		})
	}

	// Check policy-based win conditions.
	if game.HumanPoliciesEnacted >= game.Rules.HumanPoliciesToWin {
		return h.endGame(ctx, game, secrethitler.PartyHuman, secrethitler.WinHumanPolicies)
	}
	if game.AIPoliciesEnacted >= game.Rules.AIPoliciesToWin {
		return h.endGame(ctx, game, secrethitler.PartyAI, secrethitler.WinAIPolicies)
	}

	// Record term limits after a successful, non-topdeck enactment.
	if !topDeck && gov != nil {
		prevPres := gov.PresidentSeat
		prevChan := *gov.ChancellorSeat
		game.PreviousPresidentSeat = &prevPres
		game.PreviousChancellorSeat = &prevChan
	}

	// If a fascist policy was enacted (non-topdeck), check for a power.
	if !topDeck && policy == secrethitler.PolicyAI {
		power := secrethitler.PowerFor(game.PlayerCount, game.AIPoliciesEnacted)
		if power != "" {
			return h.enterExecutiveAction(ctx, game, gov, power)
		}
	}

	// Otherwise rotate to the next round.
	h.rotatePresident(game, players)
	h.setPhase(ctx, game, secrethitler.PhaseNomination, ReasonAction)
	return h.saveGame(ctx, game)
}

// topDeck enacts the top policy of the draw pile automatically when the
// election tracker advances to 3.
func (h *GameHandler) topDeck(ctx context.Context, game *models.Game) error {
	if len(game.DrawPile) == 0 {
		h.reshuffle(ctx, game)
	}
	top := game.DrawPile[0]
	game.DrawPile = game.DrawPile[1:]
	// Top-deck bypasses the Government record.
	players, err := h.loadPlayers(ctx, game.GameID)
	if err != nil {
		return err
	}
	return h.applyEnactedPolicy(ctx, game, nil, players, top, true)
}

// ProposeVeto is called by the chancellor during the legislative phase
// when veto is unlocked. The president must then confirm via ResolveVeto.
func (h *GameHandler) ProposeVeto(ctx context.Context, gameID, chancellorPlayerID string) error {
	game, err := h.loadGame(ctx, gameID)
	if err != nil {
		return err
	}
	if err := mustPhase(game, secrethitler.PhaseLegislativeChancellor); err != nil {
		return err
	}
	if !game.VetoUnlocked {
		return fmt.Errorf("%w: veto not unlocked", ErrInvalidTransition)
	}
	if game.ChancellorSeat == nil {
		return ErrInvalidTransition
	}
	players, err := h.loadPlayers(ctx, gameID)
	if err != nil {
		return err
	}
	chancellor := findPlayerBySeat(players, *game.ChancellorSeat)
	if chancellor == nil || chancellor.PlayerID != chancellorPlayerID {
		return ErrNotChancellor
	}

	gov, err := h.loadGovernment(ctx, gameID, game.CurrentGovernmentID)
	if err != nil {
		return err
	}
	gov.VetoProposed = true
	if err := h.db.Upsert(ctx, nil, gov); err != nil {
		return err
	}

	ev := models.NewGameEvent(gameID, secrethitler.EventVetoProposed, chancellorPlayerID)
	h.broadcast(ctx, ev, VetoProposedPayload{GovernmentID: gov.GovernmentID})

	h.setPhase(ctx, game, secrethitler.PhaseVetoRequested, ReasonAction)
	return h.saveGame(ctx, game)
}

// ResolveVeto is the president's response to a veto proposal. If
// accepted, both policies are discarded, the government is vetoed, and
// the election tracker advances. If rejected, the chancellor must
// enact one of the two policies and the veto right is spent for this
// government only.
func (h *GameHandler) ResolveVeto(ctx context.Context, gameID, presidentPlayerID string, accepted bool) error {
	game, err := h.loadGame(ctx, gameID)
	if err != nil {
		return err
	}
	if err := mustPhase(game, secrethitler.PhaseVetoRequested); err != nil {
		return err
	}
	players, err := h.loadPlayers(ctx, gameID)
	if err != nil {
		return err
	}
	president := findPlayerBySeat(players, game.PresidentSeat)
	if president == nil || president.PlayerID != presidentPlayerID {
		return ErrNotPresident
	}

	gov, err := h.loadGovernment(ctx, gameID, game.CurrentGovernmentID)
	if err != nil {
		return err
	}
	gov.RecordVeto(accepted)
	if err := h.db.Upsert(ctx, nil, gov); err != nil {
		return err
	}

	ev := models.NewGameEvent(gameID, secrethitler.EventVetoResolved, presidentPlayerID)
	h.broadcast(ctx, ev, VetoResolvedPayload{GovernmentID: gov.GovernmentID, Accepted: accepted})

	if accepted {
		// Discard both policies and advance the election tracker.
		for _, p := range gov.ChancellorOptions {
			discardPolicy(game, p)
		}
		_, _, err := h.onElectionFailed(ctx, game, players, ReasonAction)
		return err
	}
	// Rejected: chancellor must now enact. Return to the chancellor phase.
	h.setPhase(ctx, game, secrethitler.PhaseLegislativeChancellor, ReasonAction)
	return h.saveGame(ctx, game)
}

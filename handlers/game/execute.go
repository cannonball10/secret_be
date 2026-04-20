package game

import (
	"context"
	"fmt"

	"github.com/cannonball10/foundation/models"
	"github.com/cannonball10/foundation/schemas/secrethitler"
)

// enterExecutiveAction creates a pending ExecutiveAction record and
// advances the phase so the president can act.
func (h *GameHandler) enterExecutiveAction(ctx context.Context, game *models.Game, gov *models.Government, actionType secrethitler.ExecutiveActionType) error {
	action := models.NewExecutiveAction(nil, game.GameID, game.Round, actionType, gov.PresidentPlayerID)
	if err := h.db.Upsert(ctx, nil, action); err != nil {
		return err
	}
	game.PendingActionType = actionType
	game.PendingActionID = action.ActionID

	ev := models.NewGameEvent(game.GameID, secrethitler.EventExecutiveAction, gov.PresidentPlayerID)
	h.broadcast(ctx, ev, ExecutiveActionPayload{
		ActionID:          action.ActionID,
		Type:              actionType,
		PresidentPlayerID: gov.PresidentPlayerID,
	})

	// Policy peek resolves immediately: whisper the top 3 to the
	// president and auto-advance back to nomination.
	if actionType == secrethitler.ActionPolicyPeek {
		return h.resolvePolicyPeek(ctx, game, action)
	}

	h.setPhase(ctx, game, secrethitler.PhaseExecutiveAction, ReasonAction)
	return h.saveGame(ctx, game)
}

// resolvePolicyPeek shows the top three policies to the president
// privately, marks the action complete, and rotates to the next round.
func (h *GameHandler) resolvePolicyPeek(ctx context.Context, game *models.Game, action *models.ExecutiveAction) error {
	if len(game.DrawPile) < 3 {
		h.reshuffle(ctx, game)
	}
	peek := append([]secrethitler.PolicyType(nil), game.DrawPile[:3]...)
	action.PeekedPolicies = peek
	action.Complete()
	if err := h.db.Upsert(ctx, nil, action); err != nil {
		return err
	}

	ev := models.NewGameEvent(game.GameID, secrethitler.EventExecutiveAction, action.PresidentPlayerID)
	h.whisper(ctx, ev, action.PresidentPlayerID, PolicyPeekPayload{
		ActionID: action.ActionID,
		Policies: peek,
	})

	return h.completeExecutiveAction(ctx, game)
}

// ExecuteAction is invoked by the president during a PhaseExecutiveAction
// phase to resolve the pending power. targetPlayerID is required for
// investigate, special election, and execution; ignored for policy peek.
func (h *GameHandler) ExecuteAction(ctx context.Context, gameID, presidentPlayerID, targetPlayerID string) error {
	game, err := h.loadGame(ctx, gameID)
	if err != nil {
		return err
	}
	if err := mustPhase(game, secrethitler.PhaseExecutiveAction); err != nil {
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
	target := findPlayerByID(players, targetPlayerID)
	if target == nil {
		return ErrPlayerNotFound
	}
	if target.PlayerID == president.PlayerID {
		return fmt.Errorf("%w: cannot target self", ErrIneligibleCandidate)
	}
	if !target.IsAlive {
		return fmt.Errorf("%w: target is not alive", ErrIneligibleCandidate)
	}

	action, err := h.loadAction(ctx, gameID, game.PendingActionID)
	if err != nil {
		return err
	}
	action.TargetPlayerID = target.PlayerID

	switch game.PendingActionType {
	case secrethitler.ActionInvestigateLoyalty:
		if seatAlreadyInvestigated(target, president.Seat) {
			return fmt.Errorf("%w: already investigated this player", ErrIneligibleCandidate)
		}
		action.RevealedParty = target.Party
		target.MarkInvestigatedBy(president.Seat)
		if err := h.db.Upsert(ctx, nil, target); err != nil {
			return err
		}
		h.whisper(ctx, models.NewGameEvent(gameID, secrethitler.EventExecutiveAction, presidentPlayerID).WithTarget(target.PlayerID),
			presidentPlayerID,
			InvestigateResultPayload{
				ActionID:       action.ActionID,
				TargetPlayerID: target.PlayerID,
				Party:          target.Party,
			})

	case secrethitler.ActionSpecialElection:
		// Next nomination will seat the target as president; afterwards
		// presidency returns to the next seat in normal rotation.
		returnSeat := nextAliveSeat(players, game.PresidentSeat)
		game.SpecialElectionReturnSeat = &returnSeat
		game.PresidentSeat = target.Seat
		game.ChancellorSeat = nil

	case secrethitler.ActionExecution:
		target.Kill()
		if err := h.db.Upsert(ctx, nil, target); err != nil {
			return err
		}
		h.broadcast(ctx, models.NewGameEvent(gameID, secrethitler.EventPlayerExecuted, presidentPlayerID).WithTarget(target.PlayerID),
			PlayerExecutedPayload{
				PlayerID:       target.PlayerID,
				WasRogue:      target.IsRogue(),
				ExecutedBySeat: president.Seat,
			})
		if target.IsRogue() {
			action.Complete()
			if err := h.db.Upsert(ctx, nil, action); err != nil {
				return err
			}
			return h.endGame(ctx, game, secrethitler.PartyHuman, secrethitler.WinRogueExecuted)
		}

	default:
		return fmt.Errorf("%w: unknown pending action %q", ErrInvalidTransition, game.PendingActionType)
	}

	action.Complete()
	if err := h.db.Upsert(ctx, nil, action); err != nil {
		return err
	}
	return h.completeExecutiveAction(ctx, game)
}

// completeExecutiveAction clears the pending-action state, rotates the
// presidency (honouring any special-election return seat), and advances
// to Nomination.
func (h *GameHandler) completeExecutiveAction(ctx context.Context, game *models.Game) error {
	players, err := h.loadPlayers(ctx, game.GameID)
	if err != nil {
		return err
	}
	game.PendingActionType = ""
	game.PendingActionID = ""

	// If we just queued a special election the PresidentSeat has
	// already been set to the target; otherwise rotate normally.
	if game.SpecialElectionReturnSeat != nil && game.ChancellorSeat == nil {
		// president already set to target; just bump round and clear.
		game.Round++
		game.CurrentGovernmentID = ""
	} else {
		h.rotatePresident(game, players)
	}

	h.setPhase(ctx, game, secrethitler.PhaseNomination, ReasonAction)
	return h.saveGame(ctx, game)
}

// loadAction fetches an ExecutiveAction by ID.
func (h *GameHandler) loadAction(ctx context.Context, gameID, actionID string) (*models.ExecutiveAction, error) {
	m, err := h.db.Get(ctx, nil, models.ExecutiveActionKeys.CompositeKey(gameID, actionID))
	if err != nil {
		return nil, err
	}
	if m == nil {
		return nil, ErrInvalidTransition
	}
	return m.(*models.ExecutiveAction), nil
}

// seatAlreadyInvestigated returns true if the given investigator seat
// has already used Investigate Loyalty on the target.
func seatAlreadyInvestigated(target *models.Player, investigatorSeat int) bool {
	for _, s := range target.InvestigatedBySeats {
		if s == investigatorSeat {
			return true
		}
	}
	return false
}

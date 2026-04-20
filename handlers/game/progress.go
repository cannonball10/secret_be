package game

import (
	"context"
	"fmt"

	"github.com/cannonball10/foundation/models"
	"github.com/cannonball10/foundation/schemas/secrethitler"
)

// ForceProgress is invoked by the host to manually advance the phase
// when players are stalling. It behaves exactly like the phase timer
// expiring, but skips the deadline check.
func (h *GameHandler) ForceProgress(ctx context.Context, gameID, hostUserID string) error {
	game, err := h.loadGame(ctx, gameID)
	if err != nil {
		return err
	}
	if game.HostUserID != hostUserID {
		return ErrNotHost
	}
	return h.advance(ctx, game, ReasonForced)
}

// TimerExpired is invoked by the scheduler when a phase deadline has
// elapsed. It refuses to advance if the deadline is in the future,
// which keeps stale timer ticks from corrupting state.
func (h *GameHandler) TimerExpired(ctx context.Context, gameID string) error {
	game, err := h.loadGame(ctx, gameID)
	if err != nil {
		return err
	}
	if game.PhaseDeadline == nil {
		return fmt.Errorf("%w: no active deadline", ErrInvalidTransition)
	}
	if h.clock.Now().Before(*game.PhaseDeadline) {
		return ErrDeadlineNotReached
	}
	return h.advance(ctx, game, ReasonTimeout)
}

// advance chooses a phase-appropriate "force this forward" action and
// runs it. Shared by both TimerExpired and ForceProgress.
//
// By phase:
//   - Nomination              : no government exists yet; re-rotate president.
//   - Election                : resolve with whatever votes have landed.
//   - LegislativePresident    : auto-discard the first policy.
//   - LegislativeChancellor   : auto-enact the first remaining policy.
//   - VetoRequested           : auto-reject the veto (safer default).
//   - ExecutiveAction         : auto-pick the first eligible target,
//     or, for policy peek, re-emit the whisper.
func (h *GameHandler) advance(ctx context.Context, game *models.Game, reason ProgressReason) error {
	if game.Status != secrethitler.GameStatusInProgress {
		return fmt.Errorf("%w: game is not in progress", ErrInvalidTransition)
	}
	// Stash the reason so downstream setPhase calls can label their
	// emitted phase-change events (even when they're reached via
	// helpers like ChancellorEnact that don't take a reason argument).
	ctx = ctxWithReason(ctx, reason)
	players, err := h.loadPlayers(ctx, game.GameID)
	if err != nil {
		return err
	}

	switch game.Phase {
	case secrethitler.PhaseNomination:
		// Skip this president's turn: advance tracker and rotate.
		_, _, err := h.onElectionFailed(ctx, game, players, reason)
		return err

	case secrethitler.PhaseCablePhase:
		return h.advanceFromCablePhase(ctx, game, reason)

	case secrethitler.PhaseElection:
		votes, err := h.loadVotesForGovernment(ctx, game.GameID, game.CurrentGovernmentID)
		if err != nil {
			return err
		}
		_, _, err = h.resolveElection(ctx, game, players, votes, reason)
		return err

	case secrethitler.PhaseLegislativePresident:
		president := findPlayerBySeat(players, game.PresidentSeat)
		if president == nil {
			return ErrPlayerNotFound
		}
		// Default: discard the first drawn policy.
		return h.PresidentDiscard(ctx, game.GameID, president.PlayerID, 0)

	case secrethitler.PhaseLegislativeChancellor:
		if game.ChancellorSeat == nil {
			return ErrInvalidTransition
		}
		chancellor := findPlayerBySeat(players, *game.ChancellorSeat)
		if chancellor == nil {
			return ErrPlayerNotFound
		}
		return h.ChancellorEnact(ctx, game.GameID, chancellor.PlayerID, 0)

	case secrethitler.PhaseVetoRequested:
		president := findPlayerBySeat(players, game.PresidentSeat)
		if president == nil {
			return ErrPlayerNotFound
		}
		return h.ResolveVeto(ctx, game.GameID, president.PlayerID, false)

	case secrethitler.PhaseExecutiveAction:
		return h.autoExecuteAction(ctx, game, players)

	case secrethitler.PhaseLobby, secrethitler.PhaseGameOver:
		return fmt.Errorf("%w: cannot advance from %s", ErrInvalidTransition, game.Phase)
	}
	return fmt.Errorf("%w: unknown phase %s", ErrInvalidTransition, game.Phase)
}

// autoExecuteAction picks a reasonable default target for the pending
// executive action so the game can keep moving even if the president is
// AFK. For special election this picks the next alive seat. For
// investigate / execution, it picks the first alive, non-investigated
// player that isn't the president.
func (h *GameHandler) autoExecuteAction(ctx context.Context, game *models.Game, players []*models.Player) error {
	president := findPlayerBySeat(players, game.PresidentSeat)
	if president == nil {
		return ErrPlayerNotFound
	}
	var target *models.Player
	for _, p := range alivePlayers(players) {
		if p.PlayerID == president.PlayerID {
			continue
		}
		if game.PendingActionType == secrethitler.ActionInvestigateLoyalty && seatAlreadyInvestigated(p, president.Seat) {
			continue
		}
		target = p
		break
	}
	if target == nil {
		// Nothing eligible: just complete the action and rotate.
		return h.completeExecutiveAction(ctx, game)
	}
	return h.ExecuteAction(ctx, game.GameID, president.PlayerID, target.PlayerID)
}

// endGame transitions the game into GameOver and emits the final event.
func (h *GameHandler) endGame(ctx context.Context, game *models.Game, winner secrethitler.Party, cond secrethitler.WinCondition) error {
	game.Status = secrethitler.GameStatusCompleted
	game.Winner = winner
	game.WinCondition = cond
	now := h.clock.Now()
	game.EndedAt = &now
	game.PhaseDeadline = nil

	ev := models.NewGameEvent(game.GameID, secrethitler.EventGameEnded, "")
	h.broadcast(ctx, ev, GameEndedPayload{Winner: winner, WinCondition: cond})

	h.setPhase(ctx, game, secrethitler.PhaseGameOver, ReasonAction)
	return h.saveGame(ctx, game)
}

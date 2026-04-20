package game

import (
	"context"
	"fmt"
	"time"

	"github.com/cannonball10/foundation/models"
	"github.com/cannonball10/foundation/schemas/replicant"
)

// NominateChancellor is invoked by the current president to pick their
// running mate. It validates eligibility (not self, alive, not
// term-limited, not the previous president or chancellor) and advances
// the phase to Election.
func (h *GameHandler) NominateChancellor(ctx context.Context, gameID, presidentPlayerID, chancellorPlayerID string) (*models.Government, error) {
	game, err := h.loadGame(ctx, gameID)
	if err != nil {
		return nil, err
	}
	if err := mustPhase(game, replicant.PhaseNomination); err != nil {
		return nil, err
	}

	players, err := h.loadPlayers(ctx, gameID)
	if err != nil {
		return nil, err
	}
	president := findPlayerBySeat(players, game.PresidentSeat)
	if president == nil || president.PlayerID != presidentPlayerID {
		return nil, ErrNotPresident
	}
	chancellor := findPlayerByID(players, chancellorPlayerID)
	if chancellor == nil {
		return nil, ErrPlayerNotFound
	}
	if err := validateChancellorEligibility(game, president, chancellor, players); err != nil {
		return nil, err
	}

	gov := models.NewGovernment(nil, gameID, game.Round, president.PlayerID, president.Seat)
	gov.NominateChancellor(chancellor.PlayerID, chancellor.Seat)
	gov.IsSpecialElection = game.SpecialElectionReturnSeat != nil
	game.CurrentGovernmentID = gov.GovernmentID
	game.ChancellorSeat = &chancellor.Seat

	if err := h.db.Upsert(ctx, nil, gov); err != nil {
		return nil, err
	}

	// Pick the next phase. With CablePhase enabled we interpose a
	// cable-phase interstitial before the election vote; otherwise we
	// jump straight to election (vanilla flow). Either way the
	// deadline reported to the chancellor_nominated listener is the
	// *next* phase's, so clients can pace their UI correctly.
	nextPhase := replicant.PhaseElection
	if game.Rules.CablePhaseMode == replicant.CableModeEveryRound {
		nextPhase = replicant.PhaseCablePhase
	}
	deadlineStr := ""
	if d := h.deadlineFor(game, nextPhase); d != nil {
		deadlineStr = d.UTC().Format(time.RFC3339)
	}
	ev := models.NewGameEvent(gameID, replicant.EventChancellorNominated, president.PlayerID).
		WithTarget(chancellor.PlayerID)
	h.broadcast(ctx, ev, ChancellorNominatedPayload{
		PresidentPlayerID:  president.PlayerID,
		ChancellorPlayerID: chancellor.PlayerID,
		GovernmentID:       gov.GovernmentID,
		Deadline:           deadlineStr,
		Round:              game.Round,
	})

	h.setPhase(ctx, game, nextPhase, ReasonAction)
	if nextPhase == replicant.PhaseCablePhase {
		// Surface a dedicated opener event so clients can mount
		// chat controls without sniffing phase_changed. Payload is
		// empty today; step 3b adds chat-specific fields.
		openEv := models.NewGameEvent(gameID, replicant.EventCablePhaseOpened, "")
		h.broadcast(ctx, openEv, CablePhaseOpenedPayload{
			GovernmentID: gov.GovernmentID,
			Deadline:     deadlineStr,
		})
	}
	if err := h.saveGame(ctx, game); err != nil {
		return nil, err
	}
	return gov, nil
}

// validateChancellorEligibility enforces the term-limit and alive rules.
func validateChancellorEligibility(game *models.Game, president, chancellor *models.Player, players []*models.Player) error {
	if !chancellor.IsAlive {
		return fmt.Errorf("%w: chancellor is not alive", ErrIneligibleCandidate)
	}
	if chancellor.PlayerID == president.PlayerID {
		return fmt.Errorf("%w: cannot nominate self", ErrIneligibleCandidate)
	}
	// Term limits: the previous elected government cannot run again.
	// Exception: with 5 alive players, only the previous chancellor is
	// restricted (the president can serve again).
	aliveCount := len(alivePlayers(players))
	if game.PreviousChancellorSeat != nil && *game.PreviousChancellorSeat == chancellor.Seat {
		return fmt.Errorf("%w: previous chancellor is term-limited", ErrIneligibleCandidate)
	}
	if aliveCount > 5 && game.PreviousPresidentSeat != nil && *game.PreviousPresidentSeat == chancellor.Seat {
		return fmt.Errorf("%w: previous president is term-limited", ErrIneligibleCandidate)
	}
	return nil
}

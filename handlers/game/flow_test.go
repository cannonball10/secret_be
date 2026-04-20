package game

import (
	"context"
	"testing"

	"github.com/cannonball10/foundation/models"
	"github.com/cannonball10/foundation/schemas/secrethitler"
)

// findAnyEligibleChancellor returns an alive player who is not the
// current president and is not term-limited.
func findAnyEligibleChancellor(players []*models.Player, game *models.Game, president *models.Player) *models.Player {
	aliveCount := len(alivePlayers(players))
	for _, p := range players {
		if !p.IsAlive || p.PlayerID == president.PlayerID {
			continue
		}
		if game.PreviousChancellorSeat != nil && *game.PreviousChancellorSeat == p.Seat {
			continue
		}
		if aliveCount > 5 && game.PreviousPresidentSeat != nil && *game.PreviousPresidentSeat == p.Seat {
			continue
		}
		return p
	}
	return nil
}

// TestFullRound walks a government through every phase and verifies the
// engine ends up back in Nomination with updated term limits and board
// counters.
func TestFullRound_LiberalPolicyEnacted(t *testing.T) {
	h, _, _, _ := newTestHandler(t, 123)
	ctx := context.Background()
	g, _ := seedFiveLobby(t, h)

	started, err := h.StartGame(ctx, g.GameID, "user-host")
	if err != nil {
		t.Fatal(err)
	}
	startRound := started.Round
	players, _ := h.loadPlayers(ctx, g.GameID)
	president := findPlayerBySeat(players, started.PresidentSeat)
	chancellor := findAnyEligibleChancellor(players, started, president)
	if chancellor == nil {
		t.Fatal("no eligible chancellor")
	}
	if _, err := h.NominateChancellor(ctx, g.GameID, president.PlayerID, chancellor.PlayerID); err != nil {
		t.Fatalf("NominateChancellor: %v", err)
	}
	// Force an Election pass: everyone votes ja.
	for _, p := range players {
		if err := h.CastVote(ctx, g.GameID, p.PlayerID, secrethitler.VoteJa); err != nil {
			t.Fatalf("CastVote: %v", err)
		}
	}

	// Now in legislative_president; discard the first policy.
	gFresh, _ := h.loadGame(ctx, g.GameID)
	if gFresh.Phase != secrethitler.PhaseLegislativePresident {
		t.Fatalf("want legislative_president, got %s", gFresh.Phase)
	}
	if err := h.PresidentDiscard(ctx, g.GameID, president.PlayerID, 0); err != nil {
		t.Fatalf("PresidentDiscard: %v", err)
	}

	// Now in legislative_chancellor.
	gFresh, _ = h.loadGame(ctx, g.GameID)
	if gFresh.Phase != secrethitler.PhaseLegislativeChancellor {
		t.Fatalf("want legislative_chancellor, got %s", gFresh.Phase)
	}
	gov, _ := h.loadGovernment(ctx, g.GameID, gFresh.CurrentGovernmentID)

	// Pick a liberal option if available so we don't trigger a power.
	enactIdx := 0
	for i, p := range gov.ChancellorOptions {
		if p == secrethitler.PolicyHuman {
			enactIdx = i
			break
		}
	}
	enactedType := gov.ChancellorOptions[enactIdx]
	if err := h.ChancellorEnact(ctx, g.GameID, chancellor.PlayerID, enactIdx); err != nil {
		t.Fatalf("ChancellorEnact: %v", err)
	}

	gFinal, _ := h.loadGame(ctx, g.GameID)
	// If we enacted liberal, phase should be back to Nomination (no powers on liberal).
	// If we got forced to enact fascist (no liberal in hand) the phase
	// might be ExecutiveAction on some player counts; for 5p, first
	// fascist triggers no power so phase is still Nomination.
	if gFinal.Phase != secrethitler.PhaseNomination {
		t.Errorf("want nomination after enact, got %s (enacted=%s)", gFinal.Phase, enactedType)
	}
	if gFinal.Round <= startRound {
		t.Errorf("round should have advanced; was %d, now %d", startRound, gFinal.Round)
	}
	// Term limits: previous chancellor must be recorded.
	if gFinal.PreviousChancellorSeat == nil || *gFinal.PreviousChancellorSeat != chancellor.Seat {
		t.Errorf("previous chancellor seat = %v, want %d", gFinal.PreviousChancellorSeat, chancellor.Seat)
	}
}

func TestElectionTracker_FailedThreeInARowTopDecks(t *testing.T) {
	h, _, cap, _ := newTestHandler(t, 99)
	ctx := context.Background()
	g, _ := seedFiveLobby(t, h)
	started, err := h.StartGame(ctx, g.GameID, "user-host")
	if err != nil {
		t.Fatal(err)
	}

	// Fail three elections in a row via force-progress during nomination.
	_ = started
	for i := 0; i < 3; i++ {
		gFresh, _ := h.loadGame(ctx, g.GameID)
		if gFresh.Phase != secrethitler.PhaseNomination {
			t.Fatalf("iteration %d: want nomination, got %s", i, gFresh.Phase)
		}
		if err := h.ForceProgress(ctx, g.GameID, "user-host"); err != nil {
			t.Fatalf("ForceProgress iter %d: %v", i, err)
		}
	}
	// After 3 failed nominations the election tracker triggers a topdeck.
	// If the forced policy was liberal/fascist we just check that a
	// TopDeck event was emitted.
	tops := cap.envelopesOfType(string(secrethitler.EventTopDeckEnacted))
	if len(tops) == 0 {
		t.Error("expected a TopDeckEnacted event")
	}

	gFinal, _ := h.loadGame(ctx, g.GameID)
	if gFinal.ElectionTracker != 0 {
		t.Errorf("tracker should reset after topdeck, got %d", gFinal.ElectionTracker)
	}
}

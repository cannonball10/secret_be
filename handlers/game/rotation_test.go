package game

import (
	"context"
	"testing"

	"github.com/cannonball10/foundation/schemas/replicant"
)

// TestTopDeckDoesNotDoubleRotate forces a top-deck scenario and verifies
// the presidency advances exactly one seat (not two) across the topdeck.
func TestTopDeckDoesNotDoubleRotate(t *testing.T) {
	h, _, _, _ := newTestHandler(t, 1)
	ctx := context.Background()
	g, _ := seedFiveLobby(t, h)
	started, err := h.StartGame(ctx, g.GameID, "user-host")
	if err != nil {
		t.Fatal(err)
	}

	players, _ := h.loadPlayers(ctx, g.GameID)
	startSeat := started.PresidentSeat
	startRound := started.Round

	// Fail three elections to force a top-deck.
	for i := 0; i < 3; i++ {
		game, _ := h.loadGame(ctx, g.GameID)
		if game.Phase != replicant.PhaseNomination {
			t.Fatalf("iter %d: want nomination, got %s", i, game.Phase)
		}
		president := findPlayerBySeat(players, game.PresidentSeat)
		chancellor := pickChancellor(game, players, president)
		if _, err := h.NominateChancellor(ctx, g.GameID, president.PlayerID, chancellor.PlayerID); err != nil {
			t.Fatal(err)
		}
		for _, p := range players {
			_ = h.CastVote(ctx, g.GameID, p.PlayerID, replicant.VoteNein)
			g2, _ := h.loadGame(ctx, g.GameID)
			if g2.Phase != replicant.PhaseElection {
				break
			}
		}
	}

	after, _ := h.loadGame(ctx, g.GameID)
	if after.Status == replicant.GameStatusCompleted {
		t.Skip("topdeck ended the game; re-run with a seed that doesn't")
	}
	// Per rulebook: tracker reaches 3 → top-deck enacts → tracker
	// resets → next candidate is simply the next seat to the left. So
	// across 3 failed elections we rotate exactly 3 seats (one per
	// failure), not 4.
	expectedSeat := (startSeat + 3) % len(players)
	if after.PresidentSeat != expectedSeat {
		t.Errorf("after 3 failed elections + topdeck: seat=%d startSeat=%d want=%d",
			after.PresidentSeat, startSeat, expectedSeat)
	}
	if after.Round != startRound+3 {
		t.Errorf("round = %d, want %d", after.Round, startRound+3)
	}
}

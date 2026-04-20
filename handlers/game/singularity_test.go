package game

import (
	"context"
	"testing"

	"github.com/cannonball10/foundation/models"
	"github.com/cannonball10/foundation/schemas/secrethitler"
)

// TestSingularity_SeatedWhenEnabled verifies that StartGame deals
// exactly one Singularity and the expected counts of Human/AI/Rogue
// when RulesConfig.EnableSingularity is true.
func TestSingularity_SeatedWhenEnabled(t *testing.T) {
	ctx := context.Background()
	for n := 6; n <= 10; n++ {
		n := n
		t.Run("", func(t *testing.T) {
			h, _, _, _ := newTestHandler(t, uint64(n*13))
			g := seedLobby(t, h, n)

			// Enable Singularity on the stamped game before StartGame.
			g.Rules.EnableSingularity = true
			if err := h.saveGame(ctx, g); err != nil {
				t.Fatalf("saveGame: %v", err)
			}
			if _, err := h.StartGame(ctx, g.GameID, "user-host"); err != nil {
				t.Fatalf("StartGame: %v", err)
			}

			players, _ := h.loadPlayers(ctx, g.GameID)
			counts := map[secrethitler.Role]int{}
			for _, p := range players {
				counts[p.Role]++
			}
			baseLib, baseFas, baseHit, _, _ := secrethitler.RoleDistributionWithSingularity(n)
			if counts[secrethitler.RoleHuman] != baseLib {
				t.Errorf("n=%d humans=%d want=%d", n, counts[secrethitler.RoleHuman], baseLib)
			}
			if counts[secrethitler.RoleAI] != baseFas {
				t.Errorf("n=%d ai=%d want=%d", n, counts[secrethitler.RoleAI], baseFas)
			}
			if counts[secrethitler.RoleRogue] != baseHit {
				t.Errorf("n=%d rogue=%d want=%d", n, counts[secrethitler.RoleRogue], baseHit)
			}
			if counts[secrethitler.RoleSingularity] != 1 {
				t.Errorf("n=%d singularity=%d want=1", n, counts[secrethitler.RoleSingularity])
			}
		})
	}
}

// TestSingularity_KingmakerWin drives the engine to the codes-transfer
// threshold, then elects the Singularity as Chancellor and asserts the
// game ends with WinSingularityKingmaker (not WinAIPolicies or
// WinRogueElected).
func TestSingularity_KingmakerWin(t *testing.T) {
	ctx := context.Background()
	h, _, _, _ := newTestHandler(t, 42)
	g := seedLobby(t, h, 7)
	g.Rules.EnableSingularity = true
	if err := h.saveGame(ctx, g); err != nil {
		t.Fatalf("saveGame: %v", err)
	}
	if _, err := h.StartGame(ctx, g.GameID, "user-host"); err != nil {
		t.Fatalf("StartGame: %v", err)
	}

	// Pre-set AIPoliciesEnacted to the codes-transfer threshold so
	// the very next Chancellor election triggers the kingmaker check.
	game, _ := h.loadGame(ctx, g.GameID)
	game.AIPoliciesEnacted = game.Rules.CodesTransferAt
	if err := h.saveGame(ctx, game); err != nil {
		t.Fatalf("saveGame threshold: %v", err)
	}

	players, _ := h.loadPlayers(ctx, g.GameID)
	var sing *models.Player
	for _, p := range players {
		if p.IsSingularity() {
			sing = p
			break
		}
	}
	if sing == nil {
		t.Fatal("no singularity seated")
	}

	// Drive the phase state to an election where Singularity is the
	// nominated Chancellor. If the current president IS the Singularity,
	// advance one phase so presidency rotates.
	president := findPlayerBySeat(players, game.PresidentSeat)
	if president.PlayerID == sing.PlayerID {
		// Force a failed nomination cycle to rotate presidency. We
		// nominate the Singularity as chancellor then vote it down;
		// the tracker advances but the rotation moves us off the
		// Singularity seat.
		// Simpler: pick a non-sing alive player, nominate them, vote
		// down, then retry from the new president.
		var other *models.Player
		for _, p := range players {
			if p.PlayerID != sing.PlayerID && p.IsAlive {
				other = p
				break
			}
		}
		if _, err := h.NominateChancellor(ctx, g.GameID, president.PlayerID, other.PlayerID); err != nil {
			t.Fatalf("warm-up nominate: %v", err)
		}
		for _, p := range players {
			_ = h.CastVote(ctx, g.GameID, p.PlayerID, secrethitler.VoteNein)
			g2, _ := h.loadGame(ctx, g.GameID)
			if g2.Phase != secrethitler.PhaseElection {
				break
			}
		}
		// Reload — presidency has rotated.
		game, _ = h.loadGame(ctx, g.GameID)
		players, _ = h.loadPlayers(ctx, g.GameID)
		president = findPlayerBySeat(players, game.PresidentSeat)
		if president.PlayerID == sing.PlayerID {
			t.Fatal("presidency still on singularity after rotation; pick different seed")
		}
	}

	if _, err := h.NominateChancellor(ctx, g.GameID, president.PlayerID, sing.PlayerID); err != nil {
		t.Fatalf("NominateChancellor(sing): %v", err)
	}
	// Everyone votes ja so the Singularity gets elected.
	for _, p := range players {
		if p.IsAlive {
			_ = h.CastVote(ctx, g.GameID, p.PlayerID, secrethitler.VoteJa)
			g2, _ := h.loadGame(ctx, g.GameID)
			if g2.Status == secrethitler.GameStatusCompleted {
				break
			}
		}
	}

	final, _ := h.loadGame(ctx, g.GameID)
	if final.Status != secrethitler.GameStatusCompleted {
		t.Fatalf("game did not end: %s", summarize(final))
	}
	if final.WinCondition != secrethitler.WinSingularityKingmaker {
		t.Errorf("WinCondition=%q want=%q", final.WinCondition, secrethitler.WinSingularityKingmaker)
	}
	if final.Winner != secrethitler.PartySingularity {
		t.Errorf("Winner=%q want=%q", final.Winner, secrethitler.PartySingularity)
	}
}

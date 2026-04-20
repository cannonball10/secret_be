package models

import (
	"testing"

	"github.com/cannonball10/foundation/schemas/replicant"
)

func TestNewGame(t *testing.T) {
	g := NewGame(nil, "ABCDE", "user-1")

	if g.GameID == "" {
		t.Fatal("expected GameID to be generated")
	}
	if g.JoinCode != "ABCDE" {
		t.Errorf("JoinCode = %q, want %q", g.JoinCode, "ABCDE")
	}
	if g.HostUserID != "user-1" {
		t.Errorf("HostUserID = %q, want %q", g.HostUserID, "user-1")
	}
	if g.Status != replicant.GameStatusLobby {
		t.Errorf("Status = %q, want %q", g.Status, replicant.GameStatusLobby)
	}
	if g.Phase != replicant.PhaseLobby {
		t.Errorf("Phase = %q, want %q", g.Phase, replicant.PhaseLobby)
	}
	if g.CreatedAt.IsZero() || g.UpdatedAt.IsZero() {
		t.Error("expected timestamps to be set")
	}
}

func TestGameKeys(t *testing.T) {
	id := "G1"
	g := NewGame(&id, "ABCDE", "user-1")

	if got, want := g.PK(), "GAME#G1"; got != want {
		t.Errorf("PK() = %q, want %q", got, want)
	}
	if got, want := g.SK(), "GAME#G1"; got != want {
		t.Errorf("SK() = %q, want %q", got, want)
	}

	gsis := g.GSIs()
	if gsis[1].PK != "JOIN#ABCDE" {
		t.Errorf("GSI1.PK = %q, want JOIN#ABCDE", gsis[1].PK)
	}
	if gsis[1].SK != "GAME#G1" {
		t.Errorf("GSI1.SK = %q, want GAME#G1", gsis[1].SK)
	}
	if gsis[2].PK != "HOST#user-1" {
		t.Errorf("GSI2.PK = %q, want HOST#user-1", gsis[2].PK)
	}
	if gsis[2].SK != "GAME#G1" {
		t.Errorf("GSI2.SK = %q, want GAME#G1", gsis[2].SK)
	}
}

func TestGame_IsActive(t *testing.T) {
	g := NewGame(nil, "X", "u")
	if !g.IsActive() {
		t.Error("lobby should be active")
	}
	g.Status = replicant.GameStatusInProgress
	if !g.IsActive() {
		t.Error("in_progress should be active")
	}
	g.Status = replicant.GameStatusCompleted
	if g.IsActive() {
		t.Error("completed should not be active")
	}
}

func TestGame_RogueZoneActive(t *testing.T) {
	g := NewGame(nil, "X", "u")
	if g.RogueZoneActive() {
		t.Error("should not be in rogue zone with 0 AI policies")
	}
	g.AIPoliciesEnacted = g.Rules.CodesTransferAt
	if !g.RogueZoneActive() {
		t.Error("should be in rogue zone at threshold")
	}
}

func TestGameRegistration(t *testing.T) {
	m, err := Lookup("GAME#abc", "GAME#abc")
	if err != nil {
		t.Fatalf("Lookup failed: %v", err)
	}
	if _, ok := m.(*Game); !ok {
		t.Errorf("expected *Game, got %T", m)
	}
}

package models

import (
	"testing"

	"github.com/cannonball10/foundation/schemas/replicant"
)

func TestNewPlayer(t *testing.T) {
	p := NewPlayer(nil, "game-1", "user-1", "Alice", true)

	if p.PlayerID == "" {
		t.Fatal("expected PlayerID to be generated")
	}
	if p.GameID != "game-1" {
		t.Errorf("GameID = %q", p.GameID)
	}
	if p.UserID != "user-1" {
		t.Errorf("UserID = %q", p.UserID)
	}
	if p.DisplayName != "Alice" {
		t.Errorf("DisplayName = %q", p.DisplayName)
	}
	if !p.IsHost || !p.IsAlive || !p.IsConnected {
		t.Error("expected IsHost/IsAlive/IsConnected true")
	}
}

func TestPlayerKeys(t *testing.T) {
	id := "P1"
	p := NewPlayer(&id, "G1", "U1", "A", false)

	if got, want := p.PK(), "GAME#G1"; got != want {
		t.Errorf("PK() = %q, want %q", got, want)
	}
	if got, want := p.SK(), "PLAYER#P1"; got != want {
		t.Errorf("SK() = %q, want %q", got, want)
	}

	gsis := p.GSIs()
	if gsis[1].PK != "USER#U1" || gsis[1].SK != "GAME#G1" {
		t.Errorf("GSI1 = %+v", gsis[1])
	}
}

func TestPlayer_AssignRole(t *testing.T) {
	p := NewPlayer(nil, "g", "u", "a", false)
	p.AssignRole(replicant.RoleRogue)

	if p.Role != replicant.RoleRogue {
		t.Errorf("Role = %q", p.Role)
	}
	if p.Party != replicant.PartyAI {
		t.Errorf("Rogue should be in AI party, got %q", p.Party)
	}
	if !p.IsRogue() {
		t.Error("IsRogue() should be true")
	}

	p.AssignRole(replicant.RoleHuman)
	if p.Party != replicant.PartyHuman {
		t.Errorf("Human party expected, got %q", p.Party)
	}
}

func TestPlayer_Kill(t *testing.T) {
	p := NewPlayer(nil, "g", "u", "a", false)
	p.Kill()
	if p.IsAlive {
		t.Error("player should not be alive after Kill")
	}
}

func TestPlayer_MarkInvestigatedBy(t *testing.T) {
	p := NewPlayer(nil, "g", "u", "a", false)
	p.MarkInvestigatedBy(3)
	p.MarkInvestigatedBy(3) // idempotent
	p.MarkInvestigatedBy(4)

	if len(p.InvestigatedBySeats) != 2 {
		t.Errorf("expected 2 unique investigators, got %d", len(p.InvestigatedBySeats))
	}
}

func TestPlayerRegistration(t *testing.T) {
	m, err := Lookup("GAME#g", "PLAYER#p")
	if err != nil {
		t.Fatalf("Lookup failed: %v", err)
	}
	if _, ok := m.(*Player); !ok {
		t.Errorf("expected *Player, got %T", m)
	}
}

package models

import (
	"testing"

	"github.com/cannonball10/foundation/schemas/replicant"
)

func TestNewGameEvent(t *testing.T) {
	e := NewGameEvent("G1", replicant.EventGameStarted, "pres")
	if e.EventID == "" {
		t.Fatal("expected EventID to be generated")
	}
	if e.Type != replicant.EventGameStarted {
		t.Errorf("Type = %q", e.Type)
	}
}

func TestGameEventKeys(t *testing.T) {
	e := NewGameEvent("G1", replicant.EventVoteCast, "p")
	if e.PK() != "GAME#G1" {
		t.Errorf("PK() = %q", e.PK())
	}
	if got := e.SK(); got[:6] != "EVENT#" {
		t.Errorf("SK() = %q, want EVENT# prefix", got)
	}
}

func TestGameEventBuilders(t *testing.T) {
	e := NewGameEvent("G1", replicant.EventPlayerExecuted, "pres").
		WithTarget("victim").
		WithData(map[string]any{"reason": "president-power"})

	if e.TargetPlayerID != "victim" {
		t.Errorf("TargetPlayerID = %q", e.TargetPlayerID)
	}
	if e.Data["reason"] != "president-power" {
		t.Errorf("Data = %v", e.Data)
	}
}

func TestGameEventRegistration(t *testing.T) {
	m, err := Lookup("GAME#g", "EVENT#e")
	if err != nil {
		t.Fatalf("Lookup failed: %v", err)
	}
	if _, ok := m.(*GameEvent); !ok {
		t.Errorf("expected *GameEvent, got %T", m)
	}
}

package models

import (
	"testing"

	"github.com/cannonball10/foundation/schemas/secrethitler"
)

func TestNewExecutiveAction(t *testing.T) {
	a := NewExecutiveAction(nil, "G1", 5, secrethitler.ActionExecution, "pres")
	if a.ActionID == "" {
		t.Fatal("expected ActionID to be generated")
	}
	if a.Type != secrethitler.ActionExecution {
		t.Errorf("Type = %q", a.Type)
	}
	if a.Completed {
		t.Error("expected not-completed")
	}
}

func TestExecutiveActionKeys(t *testing.T) {
	id := "A1"
	a := NewExecutiveAction(&id, "G1", 1, secrethitler.ActionPolicyPeek, "p")
	if a.PK() != "GAME#G1" {
		t.Errorf("PK() = %q", a.PK())
	}
	if a.SK() != "ACTION#A1" {
		t.Errorf("SK() = %q", a.SK())
	}
}

func TestExecutiveAction_Complete(t *testing.T) {
	a := NewExecutiveAction(nil, "G1", 1, secrethitler.ActionPolicyPeek, "p")
	a.Complete()
	if !a.Completed {
		t.Error("expected Completed=true")
	}
}

func TestExecutiveActionRegistration(t *testing.T) {
	m, err := Lookup("GAME#g", "ACTION#a")
	if err != nil {
		t.Fatalf("Lookup failed: %v", err)
	}
	if _, ok := m.(*ExecutiveAction); !ok {
		t.Errorf("expected *ExecutiveAction, got %T", m)
	}
}

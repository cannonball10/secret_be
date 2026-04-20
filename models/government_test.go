package models

import (
	"testing"

	"github.com/cannonball10/foundation/schemas/replicant"
)

func TestNewGovernment(t *testing.T) {
	g := NewGovernment(nil, "G1", 3, "pres", 2)

	if g.GovernmentID == "" {
		t.Fatal("expected GovernmentID to be generated")
	}
	if g.GameID != "G1" || g.Round != 3 || g.PresidentPlayerID != "pres" || g.PresidentSeat != 2 {
		t.Errorf("unexpected government fields: %+v", g)
	}
	if g.Status != replicant.GovernmentStatusProposed {
		t.Errorf("initial status = %q", g.Status)
	}
}

func TestGovernmentKeys(t *testing.T) {
	id := "GOV1"
	g := NewGovernment(&id, "G1", 1, "pres", 0)
	if g.PK() != "GAME#G1" {
		t.Errorf("PK() = %q", g.PK())
	}
	if g.SK() != "GOV#GOV1" {
		t.Errorf("SK() = %q", g.SK())
	}
}

func TestGovernment_NominateChancellor(t *testing.T) {
	g := NewGovernment(nil, "G1", 1, "pres", 0)
	g.NominateChancellor("chan", 4)
	if g.ChancellorPlayerID != "chan" {
		t.Errorf("ChancellorPlayerID = %q", g.ChancellorPlayerID)
	}
	if g.ChancellorSeat == nil || *g.ChancellorSeat != 4 {
		t.Errorf("ChancellorSeat = %v", g.ChancellorSeat)
	}
}

func TestGovernment_RecordElection(t *testing.T) {
	g := NewGovernment(nil, "G1", 1, "pres", 0)

	g.RecordElection(3, 2)
	if g.Status != replicant.GovernmentStatusPassed {
		t.Errorf("expected Passed, got %q", g.Status)
	}

	g2 := NewGovernment(nil, "G1", 1, "pres", 0)
	g2.RecordElection(2, 3)
	if g2.Status != replicant.GovernmentStatusRejected {
		t.Errorf("expected Rejected, got %q", g2.Status)
	}

	// Tie loses (fails the majority).
	g3 := NewGovernment(nil, "G1", 1, "pres", 0)
	g3.RecordElection(2, 2)
	if g3.Status != replicant.GovernmentStatusRejected {
		t.Errorf("tie should be rejected, got %q", g3.Status)
	}
}

func TestGovernment_RecordEnactment(t *testing.T) {
	g := NewGovernment(nil, "G1", 1, "pres", 0)
	g.RecordEnactment(replicant.PolicyAI)

	if g.EnactedPolicy == nil || *g.EnactedPolicy != replicant.PolicyAI {
		t.Errorf("EnactedPolicy = %v", g.EnactedPolicy)
	}
	if g.Status != replicant.GovernmentStatusEnacted {
		t.Errorf("Status = %q", g.Status)
	}
}

func TestGovernment_RecordVeto(t *testing.T) {
	g := NewGovernment(nil, "G1", 1, "pres", 0)
	g.RecordVeto(true)

	if !g.VetoProposed || !g.VetoAccepted {
		t.Error("veto fields not set")
	}
	if g.Status != replicant.GovernmentStatusVetoed {
		t.Errorf("Status = %q", g.Status)
	}
}

func TestGovernmentRegistration(t *testing.T) {
	m, err := Lookup("GAME#g", "GOV#x")
	if err != nil {
		t.Fatalf("Lookup failed: %v", err)
	}
	if _, ok := m.(*Government); !ok {
		t.Errorf("expected *Government, got %T", m)
	}
}

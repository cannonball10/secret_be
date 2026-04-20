package models

import (
	"testing"

	"github.com/cannonball10/foundation/schemas/replicant"
)

func TestNewEnactedPolicy(t *testing.T) {
	p := NewEnactedPolicy("G1", 2, replicant.PolicyLiberal, "GOV1", false)
	if p.GameID != "G1" || p.Sequence != 2 || p.Type != replicant.PolicyLiberal {
		t.Errorf("fields = %+v", p)
	}
	if p.GovernmentID != "GOV1" {
		t.Errorf("GovernmentID = %q", p.GovernmentID)
	}
}

func TestEnactedPolicyKeys(t *testing.T) {
	p := NewEnactedPolicy("G1", 7, replicant.PolicyFascist, "", true)
	if p.PK() != "GAME#G1" {
		t.Errorf("PK() = %q", p.PK())
	}
	if p.SK() != "POLICY#0007" {
		t.Errorf("SK() = %q, want POLICY#0007 (zero-padded)", p.SK())
	}
}

func TestPolicyOrdering(t *testing.T) {
	// Make sure zero-padding yields correct lexicographic order.
	p1 := NewEnactedPolicy("G1", 2, replicant.PolicyLiberal, "", false).SK()
	p2 := NewEnactedPolicy("G1", 10, replicant.PolicyLiberal, "", false).SK()
	if !(p1 < p2) {
		t.Errorf("expected %q < %q", p1, p2)
	}
}

func TestEnactedPolicyRegistration(t *testing.T) {
	m, err := Lookup("GAME#g", "POLICY#0001")
	if err != nil {
		t.Fatalf("Lookup failed: %v", err)
	}
	if _, ok := m.(*EnactedPolicy); !ok {
		t.Errorf("expected *EnactedPolicy, got %T", m)
	}
}

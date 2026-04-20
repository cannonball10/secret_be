package models

import (
	"testing"

	"github.com/cannonball10/foundation/schemas/replicant"
)

func TestNewVote(t *testing.T) {
	v := NewVote("g", "gov", "p", replicant.VoteJa)
	if v.GameID != "g" || v.GovernmentID != "gov" || v.PlayerID != "p" {
		t.Errorf("fields = %+v", v)
	}
	if !v.IsJa() {
		t.Error("expected Ja")
	}
}

func TestVoteKeys(t *testing.T) {
	v := NewVote("G1", "GOV1", "P1", replicant.VoteNein)

	if v.PK() != "GAME#G1" {
		t.Errorf("PK() = %q", v.PK())
	}
	if v.SK() != "VOTE#GOV1#P1" {
		t.Errorf("SK() = %q, want VOTE#GOV1#P1", v.SK())
	}
}

func TestVoteRegistration(t *testing.T) {
	m, err := Lookup("GAME#g", "VOTE#gov#p")
	if err != nil {
		t.Fatalf("Lookup failed: %v", err)
	}
	if _, ok := m.(*Vote); !ok {
		t.Errorf("expected *Vote, got %T", m)
	}
}

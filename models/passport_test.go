package models

import (
	"testing"
	"time"

	"github.com/cannonball10/foundation/schemas/replicant"
)

func TestNewPassport(t *testing.T) {
	p := NewPassport("user-1")
	if p.UserID != "user-1" {
		t.Errorf("UserID = %q", p.UserID)
	}
	if p.GamesPlayed != 0 {
		t.Errorf("GamesPlayed = %d", p.GamesPlayed)
	}
}

func TestPassportKeys(t *testing.T) {
	p := NewPassport("U1")
	if p.PK() != "PASSPORT#U1" {
		t.Errorf("PK() = %q", p.PK())
	}
	if p.SK() != "PASSPORT#U1" {
		t.Errorf("SK() = %q", p.SK())
	}
}

func TestPassport_RecordGame(t *testing.T) {
	p := NewPassport("u")
	ended := time.Now().UTC()

	p.RecordGame(replicant.RoleHuman, true, ended)
	p.RecordGame(replicant.RoleAI, false, ended)
	p.RecordGame(replicant.RoleRogue, true, ended)
	p.RecordGame(replicant.RoleSingularity, false, ended)

	if p.GamesPlayed != 4 {
		t.Errorf("GamesPlayed = %d, want 4", p.GamesPlayed)
	}
	if p.GamesWon != 2 {
		t.Errorf("GamesWon = %d, want 2", p.GamesWon)
	}
	if p.WinsAsHuman != 1 || p.WinsAsRogue != 1 || p.WinsAsAI != 0 || p.WinsAsSingularity != 0 {
		t.Errorf("wins = human:%d ai:%d rogue:%d singularity:%d",
			p.WinsAsHuman, p.WinsAsAI, p.WinsAsRogue, p.WinsAsSingularity)
	}
	if p.TimesHuman != 1 || p.TimesAI != 1 || p.TimesRogue != 1 || p.TimesSingularity != 1 {
		t.Errorf("times = human:%d ai:%d rogue:%d singularity:%d",
			p.TimesHuman, p.TimesAI, p.TimesRogue, p.TimesSingularity)
	}
	if p.LastPlayedAt == nil {
		t.Error("expected LastPlayedAt to be set")
	}
}

func TestNewPassportEntry(t *testing.T) {
	start := time.Now().Add(-time.Hour)
	end := time.Now()
	e := NewPassportEntry("U1", "G1", "P1",
		replicant.RoleRogue, replicant.PartyAI,
		false, replicant.WinRogueExecuted,
		3, 7, start, end)

	if e.PK() != "PASSPORT#U1" {
		t.Errorf("PK() = %q", e.PK())
	}
	if e.SK() != "ENTRY#G1" {
		t.Errorf("SK() = %q", e.SK())
	}
	if e.Role != replicant.RoleRogue {
		t.Errorf("Role = %q", e.Role)
	}
	if e.Won {
		t.Error("expected Won=false")
	}
}

func TestPassportRegistration(t *testing.T) {
	m, err := Lookup("PASSPORT#u", "PASSPORT#u")
	if err != nil {
		t.Fatalf("Lookup failed: %v", err)
	}
	if _, ok := m.(*Passport); !ok {
		t.Errorf("expected *Passport, got %T", m)
	}

	m2, err := Lookup("PASSPORT#u", "ENTRY#g")
	if err != nil {
		t.Fatalf("Lookup failed: %v", err)
	}
	if _, ok := m2.(*PassportEntry); !ok {
		t.Errorf("expected *PassportEntry, got %T", m2)
	}
}

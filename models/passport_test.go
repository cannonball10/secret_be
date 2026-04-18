package models

import (
	"testing"
	"time"

	"github.com/cannonball10/foundation/schemas/secrethitler"
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

	p.RecordGame(secrethitler.RoleLiberal, true, ended)
	p.RecordGame(secrethitler.RoleFascist, false, ended)
	p.RecordGame(secrethitler.RoleHitler, true, ended)

	if p.GamesPlayed != 3 {
		t.Errorf("GamesPlayed = %d, want 3", p.GamesPlayed)
	}
	if p.GamesWon != 2 {
		t.Errorf("GamesWon = %d, want 2", p.GamesWon)
	}
	if p.WinsAsLiberal != 1 || p.WinsAsHitler != 1 || p.WinsAsFascist != 0 {
		t.Errorf("wins = lib:%d fas:%d hitler:%d",
			p.WinsAsLiberal, p.WinsAsFascist, p.WinsAsHitler)
	}
	if p.TimesLiberal != 1 || p.TimesFascist != 1 || p.TimesHitler != 1 {
		t.Errorf("times = lib:%d fas:%d hitler:%d",
			p.TimesLiberal, p.TimesFascist, p.TimesHitler)
	}
	if p.LastPlayedAt == nil {
		t.Error("expected LastPlayedAt to be set")
	}
}

func TestNewPassportEntry(t *testing.T) {
	start := time.Now().Add(-time.Hour)
	end := time.Now()
	e := NewPassportEntry("U1", "G1", "P1",
		secrethitler.RoleHitler, secrethitler.PartyFascist,
		false, secrethitler.WinHitlerExecuted,
		3, 7, start, end)

	if e.PK() != "PASSPORT#U1" {
		t.Errorf("PK() = %q", e.PK())
	}
	if e.SK() != "ENTRY#G1" {
		t.Errorf("SK() = %q", e.SK())
	}
	if e.Role != secrethitler.RoleHitler {
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

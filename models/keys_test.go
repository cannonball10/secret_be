package models

import (
	"testing"
)

func TestKeyBuilder_PK(t *testing.T) {
	kb := NewKeyBuilder("USER#", "USER#")
	if got := kb.PK("abc"); got != "USER#abc" {
		t.Errorf("PK() = %q, want %q", got, "USER#abc")
	}
}

func TestKeyBuilder_SK(t *testing.T) {
	kb := NewKeyBuilder("USER#", "USER#")
	if got := kb.SK("abc"); got != "USER#abc" {
		t.Errorf("SK() = %q, want %q", got, "USER#abc")
	}
}

func TestKeyBuilder_Key(t *testing.T) {
	kb := NewKeyBuilder("USER#", "USER#")
	key := kb.Key("abc")
	if key["PK"] != "USER#abc" {
		t.Errorf("Key PK = %q, want %q", key["PK"], "USER#abc")
	}
	if key["SK"] != "USER#abc" {
		t.Errorf("Key SK = %q, want %q", key["SK"], "USER#abc")
	}
}

func TestKeyBuilder_Pair(t *testing.T) {
	kb := NewKeyBuilder("PROVIDER#", "AUTH_ID#")
	pair := kb.Pair("CLERK", "ext123")
	if pair.PK != "PROVIDER#CLERK" {
		t.Errorf("Pair PK = %q, want %q", pair.PK, "PROVIDER#CLERK")
	}
	if pair.SK != "AUTH_ID#ext123" {
		t.Errorf("Pair SK = %q, want %q", pair.SK, "AUTH_ID#ext123")
	}
}

func TestKeyBuilder_Prefixes(t *testing.T) {
	kb := NewKeyBuilder("PFX_PK#", "PFX_SK#")
	pk, sk := kb.Prefixes()
	if pk != "PFX_PK#" {
		t.Errorf("PK prefix = %q, want %q", pk, "PFX_PK#")
	}
	if sk != "PFX_SK#" {
		t.Errorf("SK prefix = %q, want %q", sk, "PFX_SK#")
	}
}

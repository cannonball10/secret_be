package models

import (
	"errors"
	"testing"
)

func TestModelRegistry_RegisterAndLookup(t *testing.T) {
	r := NewModelRegistry()
	r.Register("FOO#", "FOO#", func() Model { return &User{} })

	m, err := r.Lookup("FOO#123", "FOO#123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := m.(*User); !ok {
		t.Errorf("expected *User, got %T", m)
	}
}

func TestModelRegistry_LookupUnregistered(t *testing.T) {
	r := NewModelRegistry()

	_, err := r.Lookup("UNKNOWN#1", "UNKNOWN#1")
	if !errors.Is(err, ErrModelNotFound) {
		t.Errorf("expected ErrModelNotFound, got %v", err)
	}
}

func TestModelRegistry_LookupEmptyPK(t *testing.T) {
	r := NewModelRegistry()

	_, err := r.Lookup("", "SK#1")
	if !errors.Is(err, ErrMissingPK) {
		t.Errorf("expected ErrMissingPK, got %v", err)
	}
}

func TestModelRegistry_LookupEmptySK(t *testing.T) {
	r := NewModelRegistry()

	_, err := r.Lookup("PK#1", "")
	if !errors.Is(err, ErrMissingSK) {
		t.Errorf("expected ErrMissingSK, got %v", err)
	}
}

func TestModelRegistry_DuplicateRegistration(t *testing.T) {
	r := NewModelRegistry()
	r.Register("DUP#", "DUP#", func() Model { return &User{} })
	r.Register("DUP#", "DUP#", func() Model { return &User{} })

	// Should still work — idempotent, no panic
	m, err := r.Lookup("DUP#1", "DUP#1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if m == nil {
		t.Fatal("expected non-nil model")
	}
}

func TestRegisterModel_UsesKeyBuilderPrefixes(t *testing.T) {
	r := NewModelRegistry()
	kb := NewKeyBuilder("KB#", "KB#")
	pk, sk := kb.Prefixes()
	r.Register(pk, sk, func() Model { return &User{} })

	m, err := r.Lookup("KB#abc", "KB#abc")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if m == nil {
		t.Fatal("expected non-nil model")
	}
}

package utils

import (
	"testing"
	"time"
)

func TestGenerateULID_NotEmpty(t *testing.T) {
	id := GenerateULID()
	if id == "" {
		t.Fatal("expected non-empty ULID")
	}
}

func TestGenerateULID_Length(t *testing.T) {
	id := GenerateULID()
	if len(id) != 26 {
		t.Errorf("ULID length = %d, want 26", len(id))
	}
}

func TestGenerateULID_Unique(t *testing.T) {
	a := GenerateULID()
	b := GenerateULID()
	if a == b {
		t.Errorf("two ULIDs should be unique, got %q twice", a)
	}
}

func TestGenerateULID_LexicographicallySortable(t *testing.T) {
	a := GenerateULID()
	// Sleep to ensure the next ULID gets a later timestamp
	time.Sleep(2 * time.Millisecond)
	b := GenerateULID()
	// b was generated after a, so b > a lexicographically
	if b <= a {
		t.Errorf("expected %q > %q (lexicographic order)", b, a)
	}
}

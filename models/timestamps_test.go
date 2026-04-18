package models

import (
	"testing"
	"time"
)

func TestNewTimestamps(t *testing.T) {
	before := time.Now().UTC()
	ts := NewTimestamps()
	after := time.Now().UTC()

	if ts.CreatedAt.IsZero() {
		t.Fatal("CreatedAt is zero")
	}
	if ts.UpdatedAt.IsZero() {
		t.Fatal("UpdatedAt is zero")
	}
	if !ts.CreatedAt.Equal(ts.UpdatedAt) {
		t.Error("CreatedAt and UpdatedAt should be equal on creation")
	}
	if ts.CreatedAt.Before(before) || ts.CreatedAt.After(after) {
		t.Error("CreatedAt not in expected range")
	}
	if ts.CreatedAt.Location() != time.UTC {
		t.Errorf("CreatedAt location = %v, want UTC", ts.CreatedAt.Location())
	}
}

func TestTimestamps_Touch(t *testing.T) {
	ts := NewTimestamps()
	originalCreated := ts.CreatedAt
	originalUpdated := ts.UpdatedAt

	// Small sleep to ensure time advances
	time.Sleep(time.Millisecond)
	ts.Touch()

	if !ts.CreatedAt.Equal(originalCreated) {
		t.Error("Touch should not modify CreatedAt")
	}
	if !ts.UpdatedAt.After(originalUpdated) {
		t.Error("Touch should advance UpdatedAt")
	}
	if ts.UpdatedAt.Location() != time.UTC {
		t.Errorf("UpdatedAt location = %v, want UTC", ts.UpdatedAt.Location())
	}
}

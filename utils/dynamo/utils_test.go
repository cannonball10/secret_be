package dynamo

import (
	"context"
	"testing"
	"time"
)

// --- ChunkSlice tests ---

func TestChunkSlice_ExactMultiple(t *testing.T) {
	items := []int{1, 2, 3, 4, 5, 6}
	chunks := ChunkSlice(items, 3)
	if len(chunks) != 2 {
		t.Fatalf("expected 2 chunks, got %d", len(chunks))
	}
	if len(chunks[0]) != 3 || len(chunks[1]) != 3 {
		t.Errorf("expected [3,3], got [%d,%d]", len(chunks[0]), len(chunks[1]))
	}
}

func TestChunkSlice_Remainder(t *testing.T) {
	items := []int{1, 2, 3, 4, 5}
	chunks := ChunkSlice(items, 3)
	if len(chunks) != 2 {
		t.Fatalf("expected 2 chunks, got %d", len(chunks))
	}
	if len(chunks[0]) != 3 || len(chunks[1]) != 2 {
		t.Errorf("expected [3,2], got [%d,%d]", len(chunks[0]), len(chunks[1]))
	}
}

func TestChunkSlice_SingleChunk(t *testing.T) {
	items := []int{1, 2}
	chunks := ChunkSlice(items, 5)
	if len(chunks) != 1 {
		t.Fatalf("expected 1 chunk, got %d", len(chunks))
	}
	if len(chunks[0]) != 2 {
		t.Errorf("expected chunk of 2, got %d", len(chunks[0]))
	}
}

func TestChunkSlice_Empty(t *testing.T) {
	items := []int{}
	chunks := ChunkSlice(items, 3)
	if len(chunks) != 1 {
		t.Fatalf("expected 1 chunk (empty), got %d", len(chunks))
	}
	if len(chunks[0]) != 0 {
		t.Errorf("expected empty chunk, got %d items", len(chunks[0]))
	}
}

func TestChunkSlice_ZeroLimit(t *testing.T) {
	items := []int{1, 2, 3}
	chunks := ChunkSlice(items, 0)
	if len(chunks) != 1 {
		t.Fatalf("expected 1 chunk for zero limit, got %d", len(chunks))
	}
}

// --- ResolveCollection tests ---

func TestResolveCollection_ExplicitName(t *testing.T) {
	name := "my-table"
	got, err := ResolveCollection(&name)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "my-table" {
		t.Errorf("got %q, want %q", got, "my-table")
	}
}

func TestResolveCollection_NilFallsToEnv(t *testing.T) {
	t.Setenv("foundation_DATABASE_TABLE", "env-table")
	got, err := ResolveCollection(nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "env-table" {
		t.Errorf("got %q, want %q", got, "env-table")
	}
}

func TestResolveCollection_EmptyFallsToEnv(t *testing.T) {
	t.Setenv("foundation_DATABASE_TABLE", "env-table")
	empty := ""
	got, err := ResolveCollection(&empty)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "env-table" {
		t.Errorf("got %q, want %q", got, "env-table")
	}
}

func TestResolveCollection_NoEnvReturnsError(t *testing.T) {
	t.Setenv("foundation_DATABASE_TABLE", "")
	_, err := ResolveCollection(nil)
	if err == nil {
		t.Fatal("expected error when no collection and no env var")
	}
}

// --- SleepWithContext tests ---

func TestSleepWithContext_CompletesNormally(t *testing.T) {
	err := SleepWithContext(context.Background(), time.Millisecond)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestSleepWithContext_ReturnsEarlyOnCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	err := SleepWithContext(ctx, 10*time.Second)
	if err == nil {
		t.Fatal("expected context cancellation error")
	}
}

package dataloader

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/cannonball10/foundation/models"
	"github.com/cannonball10/foundation/schemas/database"
	"github.com/cannonball10/foundation/testmocks"
)

// --- Loader tests ---

func TestLoader_Load_Single(t *testing.T) {
	want := &models.User{UserID: "u1"}
	db := &testmocks.DatabaseConnector{
		BulkGetFunc: func(_ context.Context, _ *string, keys []database.Key) ([]models.Model, error) {
			return []models.Model{want}, nil
		},
	}

	loader := NewLoader(db,
		func(k CompositeKey) database.Key { return models.UserKeys.Key(k.Part1) },
		func(u *models.User) CompositeKey { return CompositeKey{u.UserID, u.UserID} },
	)

	got, err := loader.LoadByID(context.Background(), "u1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got == nil {
		t.Fatal("expected non-nil user")
	}
	if got.UserID != "u1" {
		t.Errorf("UserID = %q, want %q", got.UserID, "u1")
	}
}

func TestLoader_Load_NotFound(t *testing.T) {
	db := &testmocks.DatabaseConnector{
		BulkGetFunc: func(_ context.Context, _ *string, _ []database.Key) ([]models.Model, error) {
			return []models.Model{}, nil
		},
	}

	loader := NewLoader(db,
		func(k CompositeKey) database.Key { return models.UserKeys.Key(k.Part1) },
		func(u *models.User) CompositeKey { return CompositeKey{u.UserID, u.UserID} },
	)

	got, err := loader.LoadByID(context.Background(), "missing")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != nil {
		t.Errorf("expected nil for not-found, got %+v", got)
	}
}

func TestLoader_Load_BulkGetError(t *testing.T) {
	bulkErr := errors.New("bulk get failed")
	db := &testmocks.DatabaseConnector{
		BulkGetFunc: func(_ context.Context, _ *string, _ []database.Key) ([]models.Model, error) {
			return nil, bulkErr
		},
	}

	loader := NewLoader(db,
		func(k CompositeKey) database.Key { return models.UserKeys.Key(k.Part1) },
		func(u *models.User) CompositeKey { return CompositeKey{u.UserID, u.UserID} },
	)

	_, err := loader.LoadByID(context.Background(), "u1")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestLoader_BatchFn_ReordersResults(t *testing.T) {
	u1 := &models.User{UserID: "u1"}
	u2 := &models.User{UserID: "u2"}
	u3 := &models.User{UserID: "u3"}

	db := &testmocks.DatabaseConnector{
		BulkGetFunc: func(_ context.Context, _ *string, _ []database.Key) ([]models.Model, error) {
			// Return out of order
			return []models.Model{u3, u1, u2}, nil
		},
	}

	loader := NewLoader(db,
		func(k CompositeKey) database.Key { return models.UserKeys.Key(k.Part1) },
		func(u *models.User) CompositeKey { return CompositeKey{u.UserID, u.UserID} },
	)

	// Use LoadManyByID to ensure batching
	ctx := context.Background()
	ids := []string{"u1", "u2", "u3"}
	results, errs := loader.LoadManyByID(ctx, ids)

	// Small wait for batch to complete
	time.Sleep(10 * time.Millisecond)

	for i, err := range errs {
		if err != nil {
			t.Fatalf("error at index %d: %v", i, err)
		}
	}

	if len(results) != 3 {
		t.Fatalf("expected 3 results, got %d", len(results))
	}

	// Results should be reordered to match input
	for i, id := range ids {
		if results[i] == nil {
			t.Errorf("results[%d] is nil, want UserID=%q", i, id)
			continue
		}
		if results[i].UserID != id {
			t.Errorf("results[%d].UserID = %q, want %q", i, results[i].UserID, id)
		}
	}
}

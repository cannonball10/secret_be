package user

import (
	"context"
	"errors"
	"testing"

	"github.com/cannonball10/foundation/connectors"
	"github.com/cannonball10/foundation/models"
	"github.com/cannonball10/foundation/schemas/authentication"
	"github.com/cannonball10/foundation/schemas/database"
	"github.com/cannonball10/foundation/testmocks"
	"github.com/hibiken/asynq"
)

// --- mock infrastructure ---

type mockDeps struct {
	connectors *connectors.Connectors
}

func (d *mockDeps) GetConnectors() *connectors.Connectors { return d.connectors }
func (d *mockDeps) GetTaskClient() *asynq.Client          { return nil }

func newMockDeps(db *testmocks.DatabaseConnector) *mockDeps {
	return &mockDeps{
		connectors: &connectors.Connectors{
			Database: db,
		},
	}
}

// --- GetUser tests ---

func TestGetUser_Success(t *testing.T) {
	want := &models.User{UserID: "u1", Email: "a@b.com"}
	db := &testmocks.DatabaseConnector{
		GetFunc: func(_ context.Context, _ *string, _ database.Key) (models.Model, error) {
			return want, nil
		},
	}
	h := NewUserHandler(newMockDeps(db))

	got, err := h.GetUser(context.Background(), "u1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.UserID != want.UserID {
		t.Errorf("UserID = %q, want %q", got.UserID, want.UserID)
	}
}

func TestGetUser_NotFound(t *testing.T) {
	db := &testmocks.DatabaseConnector{
		GetFunc: func(_ context.Context, _ *string, _ database.Key) (models.Model, error) {
			return nil, nil
		},
	}
	h := NewUserHandler(newMockDeps(db))

	got, err := h.GetUser(context.Background(), "missing")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != nil {
		t.Errorf("expected nil user, got %+v", got)
	}
}

func TestGetUser_DatabaseError(t *testing.T) {
	dbErr := errors.New("connection refused")
	db := &testmocks.DatabaseConnector{
		GetFunc: func(_ context.Context, _ *string, _ database.Key) (models.Model, error) {
			return nil, dbErr
		},
	}
	h := NewUserHandler(newMockDeps(db))

	_, err := h.GetUser(context.Background(), "u1")
	if !errors.Is(err, dbErr) {
		t.Errorf("expected %v, got %v", dbErr, err)
	}
}

// --- CreateUser tests ---

func TestCreateUser_Success(t *testing.T) {
	db := &testmocks.DatabaseConnector{
		UpsertFunc: func(_ context.Context, _ *string, _ models.Model) error {
			return nil
		},
	}
	h := NewUserHandler(newMockDeps(db))

	input := &models.User{UserID: "u1", Email: "a@b.com"}
	got, err := h.CreateUser(context.Background(), input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != input {
		t.Error("expected same user pointer returned")
	}
}

func TestCreateUser_DatabaseError(t *testing.T) {
	dbErr := errors.New("write failed")
	db := &testmocks.DatabaseConnector{
		UpsertFunc: func(_ context.Context, _ *string, _ models.Model) error {
			return dbErr
		},
	}
	h := NewUserHandler(newMockDeps(db))

	_, err := h.CreateUser(context.Background(), &models.User{})
	if !errors.Is(err, dbErr) {
		t.Errorf("expected %v, got %v", dbErr, err)
	}
}

// --- GetUserByAuthID tests ---

func TestGetUserByAuthID_Found(t *testing.T) {
	want := &models.User{UserID: "u1"}
	db := &testmocks.DatabaseConnector{
		QueryFunc: func(_ context.Context, _ *string, input database.QueryInput, opts database.QueryOptions) (*database.QueryOutput, error) {
			// Verify correct query construction
			if input.PartitionKey != "PROVIDER#CLERK" {
				t.Errorf("PartitionKey = %q, want %q", input.PartitionKey, "PROVIDER#CLERK")
			}
			if input.SortKey == nil || input.SortKey.EQ == nil {
				t.Fatal("expected SortKey EQ condition")
			}
			if *input.SortKey.EQ != "AUTHENTICATION_ID#ext123" {
				t.Errorf("SortKey.EQ = %q, want %q", *input.SortKey.EQ, "AUTHENTICATION_ID#ext123")
			}
			if *input.IndexName != "GSI1" {
				t.Errorf("IndexName = %q, want %q", *input.IndexName, "GSI1")
			}
			if opts.Limit != 1 {
				t.Errorf("Limit = %d, want 1", opts.Limit)
			}
			return &database.QueryOutput{Models: []any{want}}, nil
		},
	}
	h := NewUserHandler(newMockDeps(db))

	got, err := h.GetByAuthID(context.Background(), authentication.AuthenticationProvider_Clerk, "ext123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.UserID != want.UserID {
		t.Errorf("UserID = %q, want %q", got.UserID, want.UserID)
	}
}

func TestGetUserByAuthID_NotFound(t *testing.T) {
	db := &testmocks.DatabaseConnector{
		QueryFunc: func(_ context.Context, _ *string, _ database.QueryInput, _ database.QueryOptions) (*database.QueryOutput, error) {
			return &database.QueryOutput{Models: []any{}}, nil
		},
	}
	h := NewUserHandler(newMockDeps(db))

	got, err := h.GetByAuthID(context.Background(), authentication.AuthenticationProvider_Clerk, "none")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != nil {
		t.Errorf("expected nil, got %+v", got)
	}
}

func TestGetUserByAuthID_DatabaseError(t *testing.T) {
	dbErr := errors.New("query failed")
	db := &testmocks.DatabaseConnector{
		QueryFunc: func(_ context.Context, _ *string, _ database.QueryInput, _ database.QueryOptions) (*database.QueryOutput, error) {
			return nil, dbErr
		},
	}
	h := NewUserHandler(newMockDeps(db))

	_, err := h.GetByAuthID(context.Background(), authentication.AuthenticationProvider_Clerk, "ext123")
	if !errors.Is(err, dbErr) {
		t.Errorf("expected %v, got %v", dbErr, err)
	}
}

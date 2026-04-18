package dataloader

import (
	"context"
	"testing"

	"github.com/cannonball10/foundation/testmocks"
)

func TestWithLoaders_FromContext_RoundTrip(t *testing.T) {
	db := &testmocks.DatabaseConnector{}
	loaders := NewLoaders(db)
	ctx := WithLoaders(context.Background(), loaders)

	got := FromContext(ctx)
	if got == nil {
		t.Fatal("expected loaders in context")
	}
	if got.UserLoader == nil {
		t.Error("expected UserLoader to be non-nil")
	}
}

func TestFromContext_Panics_WhenMissing(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic when loaders missing from context")
		}
	}()
	FromContext(context.Background())
}

func TestNewLoaders_AllLoadersInitialized(t *testing.T) {
	db := &testmocks.DatabaseConnector{}
	loaders := NewLoaders(db)

	if loaders.UserLoader == nil {
		t.Error("expected UserLoader to be non-nil")
	}
}

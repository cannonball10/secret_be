package dependencies

import (
	"context"
	"testing"

	"github.com/cannonball10/foundation/connectors"
	"github.com/cannonball10/foundation/testmocks"
)

func TestWithDeps_FromContext_RoundTrip(t *testing.T) {
	deps := NewDependencies(DependenciesOptions{
		ConnectorsOptions: connectors.ConnectorsOptions{
			Database: &testmocks.DatabaseConnector{},
		},
	})

	ctx := WithDeps(context.Background(), deps)
	got := FromContext(ctx)
	if got == nil {
		t.Fatal("expected deps in context")
	}
}

func TestFromContext_Panics_WhenMissing(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic when deps missing from context")
		}
	}()
	FromContext(context.Background())
}

func TestNewDependencies_WiresHandlers(t *testing.T) {
	deps := NewDependencies(DependenciesOptions{
		ConnectorsOptions: connectors.ConnectorsOptions{
			Database: &testmocks.DatabaseConnector{},
		},
	})

	if deps.UserHandler == nil {
		t.Error("expected UserHandler to be non-nil")
	}
}

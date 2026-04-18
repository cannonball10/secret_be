package feature

import (
	"context"

	"github.com/cannonball10/foundation/schemas/feature"
)

// FeatureConnector is an abstract feature-flag interface that can be extended for
// concrete implementations like LaunchDarkly, etc.
type FeatureConnector interface {
	Bool(ctx context.Context, flagKey string, defaultValue bool, eval *feature.EvalContext) (bool, error)
}

func DefaultFeatureConnector(ctx context.Context) (FeatureConnector, error) {
	return DefaultLaunchDarklyFeatureConnector(ctx)
}

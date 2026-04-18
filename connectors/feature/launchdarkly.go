package feature

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/cannonball10/foundation/schemas/feature"
	"github.com/launchdarkly/go-sdk-common/v3/ldcontext"
	"github.com/launchdarkly/go-sdk-common/v3/ldvalue"
	ld "github.com/launchdarkly/go-server-sdk/v7"
)

const (
	defaultInitTimeout = 5 * time.Second
	anonymousKind      = "anonymous"
	anonymousKey       = "server"
)

// LaunchDarklyConfig configures the LaunchDarkly client. Nil values use defaults.
type LaunchDarklyConfig struct {
	InitTimeout time.Duration // Max wait for initial connection (default 5s)
}

// LaunchDarklyFeature implements FeatureConnector using the LaunchDarkly Go SDK.
type LaunchDarklyFeature struct {
	client *ld.LDClient
}

// NewLaunchDarklyFeature creates a LaunchDarkly feature connector. The client blocks until
// connected or InitTimeout expires. Implements ConnectorCloser via Close().
func NewLaunchDarklyFeature(sdkKey string, opts *LaunchDarklyConfig) (*LaunchDarklyFeature, error) {
	timeout := defaultInitTimeout
	if opts != nil && opts.InitTimeout > 0 {
		timeout = opts.InitTimeout
	}
	client, err := ld.MakeClient(sdkKey, timeout)
	if err != nil {
		return nil, fmt.Errorf("launchdarkly MakeClient: %w", err)
	}
	return &LaunchDarklyFeature{client: client}, nil
}

func DefaultLaunchDarklyFeatureConnector(ctx context.Context) (*LaunchDarklyFeature, error) {
	sdkKey := os.Getenv("LAUNCHDARKLY_SDK_KEY")
	if sdkKey == "" {
		return nil, errors.New("launchdarkly SDK key is not set")
	}
	client, err := ld.MakeClient(sdkKey, defaultInitTimeout)
	if err != nil {
		return nil, fmt.Errorf("launchdarkly MakeClient: %w", err)
	}
	return &LaunchDarklyFeature{client: client}, nil
}

// Bool evaluates a boolean feature flag.
func (l *LaunchDarklyFeature) Bool(ctx context.Context, flagKey string, defaultValue bool, eval *feature.EvalContext) (bool, error) {
	ldCtx := evalToLDContext(eval)
	val, err := l.client.BoolVariationCtx(ctx, flagKey, ldCtx, defaultValue)
	if err != nil {
		slog.ErrorContext(ctx, "launchdarkly Evaluate failed", "flag_key", flagKey, "error", err)
	}
	return val, err
}

// Close shuts down the LaunchDarkly client. Implements ConnectorCloser.
func (l *LaunchDarklyFeature) Close() error {
	return l.client.Close()
}

func evalToLDContext(eval *feature.EvalContext) ldcontext.Context {
	if eval == nil || eval.Key == "" {
		return ldcontext.NewWithKind(ldcontext.Kind(anonymousKind), anonymousKey)
	}
	kind := eval.Kind
	if kind == "" {
		kind = "user"
	}
	b := ldcontext.NewBuilder(eval.Key).Kind(ldcontext.Kind(kind))
	for k, v := range eval.Attributes {
		setAttr(b, k, v)
	}
	return b.Build()
}

func setAttr(b *ldcontext.Builder, name string, v any) {
	switch x := v.(type) {
	case bool:
		b.SetBool(name, x)
	case string:
		b.SetString(name, x)
	case int:
		b.SetInt(name, x)
	case int64:
		b.SetInt(name, int(x))
	case float64:
		b.SetFloat64(name, x)
	case int32:
		b.SetInt(name, int(x))
	case float32:
		b.SetFloat64(name, float64(x))
	default:
		b.SetValue(name, ldvalue.CopyArbitraryValue(v))
	}
}

package connectors

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/cannonball10/foundation/connectors/authentication"
	"github.com/cannonball10/foundation/connectors/cache"
	"github.com/cannonball10/foundation/connectors/channel"
	"github.com/cannonball10/foundation/connectors/database"
	"github.com/cannonball10/foundation/connectors/embedding"
	"github.com/cannonball10/foundation/connectors/feature"
	"github.com/cannonball10/foundation/connectors/graphdb"
	"github.com/cannonball10/foundation/connectors/inference"
	"github.com/cannonball10/foundation/connectors/livekit"
	"github.com/cannonball10/foundation/connectors/notification"
	"github.com/cannonball10/foundation/connectors/pdf"
	"github.com/cannonball10/foundation/connectors/ratelimit"
	"github.com/cannonball10/foundation/connectors/secret"
	"github.com/cannonball10/foundation/connectors/storage"
	"github.com/cannonball10/foundation/connectors/stt"
	"github.com/cannonball10/foundation/connectors/timeseries"
	"github.com/cannonball10/foundation/connectors/tts"
	"github.com/cannonball10/foundation/connectors/vector"
)

type Connectors struct {
	Authentication authentication.AuthenticationConnector
	Cache          cache.CacheConnector
	Channel        channel.ChannelConnector
	Database       database.DatabaseConnector
	Embedding      embedding.EmbeddingConnector
	Feature        feature.FeatureConnector
	GraphDB        graphdb.GraphDBConnector
	Inference      inference.InferenceConnector
	LiveKit        livekit.LiveKitConnector
	Notification   notification.NotificationConnector
	PDF            pdf.PDFConnector
	RateLimiter    ratelimit.RateLimiterConnector
	Secret         secret.SecretConnector
	Storage        storage.StorageConnector
	STT            stt.STTConnector
	Timeseries     timeseries.TimeseriesConnector
	TTS            tts.TTSConnector
	Vector         vector.VectorConnector
}

type ConnectorsOptions struct {
	Authentication authentication.AuthenticationConnector
	Cache          cache.CacheConnector
	Channel        channel.ChannelConnector
	Database       database.DatabaseConnector
	Embedding      embedding.EmbeddingConnector
	Feature        feature.FeatureConnector
	GraphDB        graphdb.GraphDBConnector
	Inference      inference.InferenceConnector
	LiveKit        livekit.LiveKitConnector
	Notification   notification.NotificationConnector
	PDF            pdf.PDFConnector
	RateLimiter    ratelimit.RateLimiterConnector
	Secret         secret.SecretConnector
	Storage        storage.StorageConnector
	STT            stt.STTConnector
	Timeseries     timeseries.TimeseriesConnector
	TTS            tts.TTSConnector
	Vector         vector.VectorConnector
}

func NewConnectors(opts ConnectorsOptions) *Connectors {
	deps := &Connectors{
		Authentication: opts.Authentication,
		Cache:          opts.Cache,
		Channel:        opts.Channel,
		Database:       opts.Database,
		Embedding:      opts.Embedding,
		Feature:        opts.Feature,
		GraphDB:        opts.GraphDB,
		Inference:      opts.Inference,
		LiveKit:        opts.LiveKit,
		Notification:   opts.Notification,
		PDF:            opts.PDF,
		RateLimiter:    opts.RateLimiter,
		Secret:         opts.Secret,
		Storage:        opts.Storage,
		STT:            opts.STT,
		Timeseries:     opts.Timeseries,
		TTS:            opts.TTS,
		Vector:         opts.Vector,
	}

	return deps
}

func DefaultConnectors(ctx context.Context) (*Connectors, error) {
	authentication, err := authentication.DefaultAuthenticationConnector(ctx)
	if err != nil {
		return nil, err
	}
	cache, err := cache.DefaultCacheConnector(ctx)
	if err != nil {
		return nil, err
	}
	channel := channel.DefaultChannelConnector(ctx)
	database, err := database.DefaultDatabaseConnector(ctx)
	if err != nil {
		return nil, err
	}
	embedding, err := embedding.DefaultEmbeddingConnector(ctx)
	if err != nil {
		return nil, err
	}
	feature, err := feature.DefaultFeatureConnector(ctx)
	if err != nil {
		return nil, err
	}
	graphdb, err := graphdb.DefaultGraphDBConnector(ctx)
	if err != nil {
		return nil, err
	}
	inferenceConn, err := inference.DefaultInferenceConnector(ctx)
	if err != nil {
		return nil, err
	}
	notification, err := notification.DefaultNotificationConnector(ctx)
	if err != nil {
		return nil, err
	}
	pdfConn, err := pdf.DefaultPDFConnector(ctx)
	if err != nil {
		return nil, err
	}
	rateLimiter, err := ratelimit.DefaultRateLimiterConnector(ctx)
	if err != nil {
		return nil, err
	}
	secret, err := secret.DefaultSecretConnector(ctx)
	if err != nil {
		return nil, err
	}
	storage, err := storage.DefaultStorageConnector(ctx)
	if err != nil {
		return nil, err
	}
	vector, err := vector.DefaultVectorConnector(ctx, embedding)
	if err != nil {
		return nil, err
	}
	ts, err := timeseries.DefaultTimeseriesConnector(ctx)
	if err != nil {
		return nil, err
	}
	livekitConn, err := livekit.DefaultLiveKitConnector(ctx)
	if err != nil {
		return nil, err
	}
	ttsConn, err := tts.DefaultTTSConnector(ctx)
	if err != nil {
		return nil, err
	}
	sttConn, err := stt.DefaultSTTConnector(ctx)
	if err != nil {
		return nil, err
	}
	return NewConnectors(ConnectorsOptions{
		Authentication: authentication,
		Cache:          cache,
		Channel:        channel,
		Database:       database,
		Embedding:      embedding,
		Feature:        feature,
		GraphDB:        graphdb,
		Inference:      inferenceConn,
		LiveKit:        livekitConn,
		Notification:   notification,
		PDF:            pdfConn,
		RateLimiter:    rateLimiter,
		Secret:         secret,
		Storage:        storage,
		STT:            sttConn,
		Timeseries:     ts,
		TTS:            ttsConn,
		Vector:         vector,
	}), nil
}

func (d *Connectors) values() []any {
	return []any{
		d.Authentication,
		d.Cache,
		d.Channel,
		d.Embedding,
		d.Database,
		d.Feature,
		d.GraphDB,
		d.Inference,
		d.LiveKit,
		d.Notification,
		d.PDF,
		d.RateLimiter,
		d.Secret,
		d.Storage,
		d.STT,
		d.Timeseries,
		d.TTS,
		d.Vector,
	}
}

// ConnectorInitializer is an optional lifecycle hook for setup.
type ConnectorInitializer interface {
	Init() error
}

// ConnectorCloser is an optional lifecycle hook for teardown.
type ConnectorCloser interface {
	Close() error
}

// ConnectorHealthChecker is an optional interface for connectors that support
// health/liveness checks (e.g. Redis PING, Neo4j VerifyConnectivity).
type ConnectorHealthChecker interface {
	Ping(ctx context.Context) error
}

// Init calls Init on any dependency that implements ConnectorInitializer.
func (d *Connectors) Init() error {
	for _, value := range d.values() {
		if initializer, ok := value.(ConnectorInitializer); ok {
			if err := initializer.Init(); err != nil {
				slog.Error("connector init failed", "connector", fmt.Sprintf("%T", value), "error", err)
				return err
			}
		}
	}
	slog.Info("all connectors initialized")
	return nil
}

// Close calls Close on any dependency that implements ConnectorCloser.
func (d *Connectors) Close() error {
	for _, value := range d.values() {
		if closer, ok := value.(ConnectorCloser); ok {
			if err := closer.Close(); err != nil {
				slog.Error("connector close failed", "connector", fmt.Sprintf("%T", value), "error", err)
				return err
			}
		}
	}
	return nil
}

// Health calls Ping on every connector that implements ConnectorHealthChecker.
// Returns nil if all healthy, or the first error encountered.
func (d *Connectors) Health(ctx context.Context) error {
	for _, value := range d.values() {
		if checker, ok := value.(ConnectorHealthChecker); ok {
			if err := checker.Ping(ctx); err != nil {
				slog.ErrorContext(ctx, "connector health check failed", "connector", fmt.Sprintf("%T", value), "error", err)
				return err
			}
		}
	}
	return nil
}

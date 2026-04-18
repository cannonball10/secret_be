package timeseries

import (
	"context"

	tsTypes "github.com/cannonball10/foundation/schemas/timeseries"
)

// TimeseriesConnector is an abstract interface for time-indexed data storage.
// Implementations may use TimescaleDB, InfluxDB, or other time-series databases.
type TimeseriesConnector interface {
	// WritePoints writes one or more data points to a measurement.
	WritePoints(ctx context.Context, measurement string, points []tsTypes.Point) error

	// Query retrieves raw data points within a time range.
	Query(ctx context.Context, input tsTypes.QueryInput) ([]tsTypes.Point, error)

	// QueryAggregate retrieves time-bucketed aggregations (for OHLCV candles, etc.).
	QueryAggregate(ctx context.Context, input tsTypes.AggregateQueryInput) ([]tsTypes.AggregatePoint, error)

	// CreateMeasurement creates a measurement/hypertable with the given options.
	CreateMeasurement(ctx context.Context, measurement string, opts *tsTypes.MeasurementOptions) error

	// DropMeasurement drops a measurement and all its data.
	DropMeasurement(ctx context.Context, measurement string) error
}

// DefaultTimeseriesConnector returns a TimescaleDB-backed connector using environment variables.
func DefaultTimeseriesConnector(ctx context.Context) (TimeseriesConnector, error) {
	return DefaultTimescaleConnector(ctx)
}

package timeseries

import "time"

// Point represents a single time-indexed data point.
type Point struct {
	Time   time.Time
	Tags   map[string]string  // Indexed dimensions (symbol, exchange)
	Fields map[string]float64 // Measured values (price, volume)
}

// QueryInput specifies parameters for querying raw data points.
type QueryInput struct {
	Measurement string
	Start       time.Time
	End         time.Time
	Tags        map[string]string // Filter by tag values
	Limit       int               // Maximum number of points to return
}

// AggregateQueryInput specifies parameters for time-bucketed aggregation queries.
type AggregateQueryInput struct {
	Measurement  string
	Start        time.Time
	End          time.Time
	Interval     time.Duration     // Time bucket size (e.g., 1 minute, 1 hour)
	Tags         map[string]string // Filter by tag values
	GroupBy      []string          // Tag columns to group by
	Aggregations []AggregationSpec // Aggregations to compute
}

// AggregationSpec defines a single aggregation to compute.
type AggregationSpec struct {
	OutputName string          // Name for the result column
	Field      string          // Source field to aggregate
	Type       AggregationType // Aggregation function
}

// AggregationType defines supported aggregation functions.
type AggregationType string

const (
	AggregationFirst AggregationType = "first"
	AggregationLast  AggregationType = "last"
	AggregationMin   AggregationType = "min"
	AggregationMax   AggregationType = "max"
	AggregationSum   AggregationType = "sum"
	AggregationAvg   AggregationType = "avg"
	AggregationCount AggregationType = "count"
)

// AggregatePoint represents a time-bucketed aggregation result.
type AggregatePoint struct {
	Time   time.Time              // Start of the time bucket
	Tags   map[string]string      // Grouped tag values
	Values map[string]float64     // Aggregated values keyed by OutputName
}

// MeasurementOptions specifies options when creating a measurement/hypertable.
type MeasurementOptions struct {
	// ChunkInterval is the time range for each chunk (default: 7 days).
	ChunkInterval time.Duration

	// RetentionPeriod, if set, enables automatic data expiration.
	RetentionPeriod time.Duration

	// CompressionAfter, if set, enables automatic compression after this duration.
	CompressionAfter time.Duration

	// TagColumns specifies which columns should be indexed as tags.
	TagColumns []string

	// FieldColumns specifies which columns store field values.
	FieldColumns []string
}

// DefaultMeasurementOptions returns sensible defaults for measurement creation.
func DefaultMeasurementOptions() *MeasurementOptions {
	return &MeasurementOptions{
		ChunkInterval: 7 * 24 * time.Hour, // 7 days
	}
}

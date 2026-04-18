package timeseries

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"sync"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	tsTypes "github.com/cannonball10/foundation/schemas/timeseries"
)

// TimescaleConnector implements TimeseriesConnector using TimescaleDB.
type TimescaleConnector struct {
	pool        *pgxpool.Pool
	poolOnce    sync.Once
	poolFactory func() (*pgxpool.Pool, error)
}

// NewTimescaleConnector returns a connector with an eagerly provided connection pool.
func NewTimescaleConnector(pool *pgxpool.Pool) *TimescaleConnector {
	return &TimescaleConnector{pool: pool}
}

// NewTimescaleConnectorLazy returns a connector that initializes the pool on first use.
func NewTimescaleConnectorLazy(factory func() (*pgxpool.Pool, error)) *TimescaleConnector {
	return &TimescaleConnector{poolFactory: factory}
}

// DefaultTimescaleConnector creates a connector using the TIMESCALE_DSN environment variable.
func DefaultTimescaleConnector(ctx context.Context) (*TimescaleConnector, error) {
	dsn := os.Getenv("TIMESCALE_DSN")
	if dsn == "" {
		return nil, errors.New("TIMESCALE_DSN environment variable is not set")
	}
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to create TimescaleDB pool: %w", err)
	}
	return NewTimescaleConnector(pool), nil
}

func (c *TimescaleConnector) getPool(ctx context.Context) (*pgxpool.Pool, error) {
	var initErr error
	c.poolOnce.Do(func() {
		if c.pool == nil && c.poolFactory != nil {
			c.pool, initErr = c.poolFactory()
		}
	})
	if initErr != nil {
		return nil, initErr
	}
	if c.pool == nil {
		return nil, errors.New("TimescaleDB pool is not initialized")
	}
	return c.pool, nil
}

// Ping checks TimescaleDB connectivity. Implements connectors.ConnectorHealthChecker.
func (c *TimescaleConnector) Ping(ctx context.Context) error {
	pool, err := c.getPool(ctx)
	if err != nil {
		return err
	}
	if err := pool.Ping(ctx); err != nil {
		slog.ErrorContext(ctx, "timescale Ping failed", "error", err)
		return err
	}
	return nil
}

// Close closes the connection pool. Implements connectors.ConnectorCloser.
func (c *TimescaleConnector) Close() error {
	if c.pool != nil {
		c.pool.Close()
	}
	return nil
}

// WritePoints writes data points to a measurement table.
func (c *TimescaleConnector) WritePoints(ctx context.Context, measurement string, points []tsTypes.Point) error {
	if len(points) == 0 {
		return nil
	}
	pool, err := c.getPool(ctx)
	if err != nil {
		return err
	}

	// Build batch insert
	batch := &pgx.Batch{}
	for _, p := range points {
		query := fmt.Sprintf(
			"INSERT INTO %s (time, tags, fields) VALUES ($1, $2, $3)",
			pgx.Identifier{measurement}.Sanitize(),
		)
		batch.Queue(query, p.Time, p.Tags, p.Fields)
	}

	results := pool.SendBatch(ctx, batch)
	defer results.Close()

	for range points {
		if _, err := results.Exec(); err != nil {
			slog.ErrorContext(ctx, "timescale WritePoints failed", "measurement", measurement, "error", err)
			return fmt.Errorf("failed to write point: %w", err)
		}
	}
	return nil
}

// Query retrieves raw data points within a time range.
func (c *TimescaleConnector) Query(ctx context.Context, input tsTypes.QueryInput) ([]tsTypes.Point, error) {
	pool, err := c.getPool(ctx)
	if err != nil {
		return nil, err
	}

	var conditions []string
	var args []any
	argIdx := 1

	conditions = append(conditions, fmt.Sprintf("time >= $%d", argIdx))
	args = append(args, input.Start)
	argIdx++

	conditions = append(conditions, fmt.Sprintf("time < $%d", argIdx))
	args = append(args, input.End)
	argIdx++

	for k, v := range input.Tags {
		conditions = append(conditions, fmt.Sprintf("tags->>%s = $%d", singleQuote(k), argIdx))
		args = append(args, v)
		argIdx++
	}

	query := fmt.Sprintf(
		"SELECT time, tags, fields FROM %s WHERE %s ORDER BY time ASC",
		pgx.Identifier{input.Measurement}.Sanitize(),
		strings.Join(conditions, " AND "),
	)
	if input.Limit > 0 {
		query += fmt.Sprintf(" LIMIT %d", input.Limit)
	}

	rows, err := pool.Query(ctx, query, args...)
	if err != nil {
		slog.ErrorContext(ctx, "timescale Query failed", "measurement", input.Measurement, "error", err)
		return nil, fmt.Errorf("query failed: %w", err)
	}
	defer rows.Close()

	var points []tsTypes.Point
	for rows.Next() {
		var p tsTypes.Point
		if err := rows.Scan(&p.Time, &p.Tags, &p.Fields); err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}
		points = append(points, p)
	}
	return points, rows.Err()
}

// QueryAggregate retrieves time-bucketed aggregations.
func (c *TimescaleConnector) QueryAggregate(ctx context.Context, input tsTypes.AggregateQueryInput) ([]tsTypes.AggregatePoint, error) {
	pool, err := c.getPool(ctx)
	if err != nil {
		return nil, err
	}

	// Build aggregation columns
	var selectCols []string
	selectCols = append(selectCols, fmt.Sprintf("time_bucket('%s', time) AS bucket", input.Interval.String()))

	for _, agg := range input.Aggregations {
		col := buildAggregation(agg)
		selectCols = append(selectCols, col)
	}

	// Add group-by tag columns
	for _, g := range input.GroupBy {
		selectCols = append(selectCols, fmt.Sprintf("tags->>%s AS %s", singleQuote(g), pgx.Identifier{g}.Sanitize()))
	}

	// Build WHERE conditions
	var conditions []string
	var args []any
	argIdx := 1

	conditions = append(conditions, fmt.Sprintf("time >= $%d", argIdx))
	args = append(args, input.Start)
	argIdx++

	conditions = append(conditions, fmt.Sprintf("time < $%d", argIdx))
	args = append(args, input.End)
	argIdx++

	for k, v := range input.Tags {
		conditions = append(conditions, fmt.Sprintf("tags->>%s = $%d", singleQuote(k), argIdx))
		args = append(args, v)
		argIdx++
	}

	// Build GROUP BY
	groupByCols := []string{"bucket"}
	for _, g := range input.GroupBy {
		groupByCols = append(groupByCols, fmt.Sprintf("tags->>%s", singleQuote(g)))
	}

	query := fmt.Sprintf(
		"SELECT %s FROM %s WHERE %s GROUP BY %s ORDER BY bucket ASC",
		strings.Join(selectCols, ", "),
		pgx.Identifier{input.Measurement}.Sanitize(),
		strings.Join(conditions, " AND "),
		strings.Join(groupByCols, ", "),
	)

	rows, err := pool.Query(ctx, query, args...)
	if err != nil {
		slog.ErrorContext(ctx, "timescale QueryAggregate failed", "measurement", input.Measurement, "error", err)
		return nil, fmt.Errorf("aggregate query failed: %w", err)
	}
	defer rows.Close()

	var results []tsTypes.AggregatePoint
	for rows.Next() {
		ap := tsTypes.AggregatePoint{
			Tags:   make(map[string]string),
			Values: make(map[string]float64),
		}

		// Build scan destinations
		scanDest := make([]any, 0, 1+len(input.Aggregations)+len(input.GroupBy))
		scanDest = append(scanDest, &ap.Time)

		valuePtrs := make([]*float64, len(input.Aggregations))
		for i := range input.Aggregations {
			valuePtrs[i] = new(float64)
			scanDest = append(scanDest, valuePtrs[i])
		}

		tagPtrs := make([]*string, len(input.GroupBy))
		for i := range input.GroupBy {
			tagPtrs[i] = new(string)
			scanDest = append(scanDest, tagPtrs[i])
		}

		if err := rows.Scan(scanDest...); err != nil {
			return nil, fmt.Errorf("failed to scan aggregate row: %w", err)
		}

		for i, agg := range input.Aggregations {
			ap.Values[agg.OutputName] = *valuePtrs[i]
		}
		for i, g := range input.GroupBy {
			ap.Tags[g] = *tagPtrs[i]
		}

		results = append(results, ap)
	}
	return results, rows.Err()
}

// CreateMeasurement creates a hypertable for storing time-series data.
func (c *TimescaleConnector) CreateMeasurement(ctx context.Context, measurement string, opts *tsTypes.MeasurementOptions) error {
	pool, err := c.getPool(ctx)
	if err != nil {
		return err
	}

	if opts == nil {
		opts = tsTypes.DefaultMeasurementOptions()
	}

	tableName := pgx.Identifier{measurement}.Sanitize()

	// Create the table
	createSQL := fmt.Sprintf(`
		CREATE TABLE IF NOT EXISTS %s (
			time TIMESTAMPTZ NOT NULL,
			tags JSONB,
			fields JSONB
		)
	`, tableName)

	if _, err := pool.Exec(ctx, createSQL); err != nil {
		slog.ErrorContext(ctx, "timescale CreateMeasurement table creation failed", "measurement", measurement, "error", err)
		return fmt.Errorf("failed to create table: %w", err)
	}

	// Convert to hypertable
	hypertableSQL := fmt.Sprintf(
		"SELECT create_hypertable('%s', 'time', if_not_exists => TRUE, chunk_time_interval => INTERVAL '%d seconds')",
		measurement,
		int64(opts.ChunkInterval.Seconds()),
	)

	if _, err := pool.Exec(ctx, hypertableSQL); err != nil {
		slog.ErrorContext(ctx, "timescale CreateMeasurement hypertable creation failed", "measurement", measurement, "error", err)
		return fmt.Errorf("failed to create hypertable: %w", err)
	}

	// Set up retention policy if specified
	if opts.RetentionPeriod > 0 {
		retentionSQL := fmt.Sprintf(
			"SELECT add_retention_policy('%s', INTERVAL '%d seconds', if_not_exists => TRUE)",
			measurement,
			int64(opts.RetentionPeriod.Seconds()),
		)
		if _, err := pool.Exec(ctx, retentionSQL); err != nil {
			slog.ErrorContext(ctx, "timescale CreateMeasurement retention policy failed", "measurement", measurement, "error", err)
			return fmt.Errorf("failed to add retention policy: %w", err)
		}
	}

	// Set up compression policy if specified
	if opts.CompressionAfter > 0 {
		enableCompressionSQL := fmt.Sprintf(
			"ALTER TABLE %s SET (timescaledb.compress, timescaledb.compress_segmentby = 'tags')",
			tableName,
		)
		if _, err := pool.Exec(ctx, enableCompressionSQL); err != nil {
			slog.ErrorContext(ctx, "timescale CreateMeasurement enable compression failed", "measurement", measurement, "error", err)
			return fmt.Errorf("failed to enable compression: %w", err)
		}

		compressionPolicySQL := fmt.Sprintf(
			"SELECT add_compression_policy('%s', INTERVAL '%d seconds', if_not_exists => TRUE)",
			measurement,
			int64(opts.CompressionAfter.Seconds()),
		)
		if _, err := pool.Exec(ctx, compressionPolicySQL); err != nil {
			slog.ErrorContext(ctx, "timescale CreateMeasurement compression policy failed", "measurement", measurement, "error", err)
			return fmt.Errorf("failed to add compression policy: %w", err)
		}
	}

	// Create index on tags for efficient filtering
	indexSQL := fmt.Sprintf(
		"CREATE INDEX IF NOT EXISTS %s_tags_idx ON %s USING GIN (tags)",
		measurement,
		tableName,
	)
	if _, err := pool.Exec(ctx, indexSQL); err != nil {
		slog.ErrorContext(ctx, "timescale CreateMeasurement index creation failed", "measurement", measurement, "error", err)
		return fmt.Errorf("failed to create tags index: %w", err)
	}

	return nil
}

// DropMeasurement drops a measurement table and all its data.
func (c *TimescaleConnector) DropMeasurement(ctx context.Context, measurement string) error {
	pool, err := c.getPool(ctx)
	if err != nil {
		return err
	}

	dropSQL := fmt.Sprintf("DROP TABLE IF EXISTS %s CASCADE", pgx.Identifier{measurement}.Sanitize())
	if _, err := pool.Exec(ctx, dropSQL); err != nil {
		slog.ErrorContext(ctx, "timescale DropMeasurement failed", "measurement", measurement, "error", err)
		return fmt.Errorf("failed to drop measurement: %w", err)
	}
	return nil
}

// buildAggregation generates the SQL for a single aggregation.
func buildAggregation(agg tsTypes.AggregationSpec) string {
	fieldAccess := fmt.Sprintf("(fields->>%s)::double precision", singleQuote(agg.Field))
	outputName := pgx.Identifier{agg.OutputName}.Sanitize()

	switch agg.Type {
	case tsTypes.AggregationFirst:
		return fmt.Sprintf("first(%s, time) AS %s", fieldAccess, outputName)
	case tsTypes.AggregationLast:
		return fmt.Sprintf("last(%s, time) AS %s", fieldAccess, outputName)
	case tsTypes.AggregationMin:
		return fmt.Sprintf("MIN(%s) AS %s", fieldAccess, outputName)
	case tsTypes.AggregationMax:
		return fmt.Sprintf("MAX(%s) AS %s", fieldAccess, outputName)
	case tsTypes.AggregationSum:
		return fmt.Sprintf("SUM(%s) AS %s", fieldAccess, outputName)
	case tsTypes.AggregationAvg:
		return fmt.Sprintf("AVG(%s) AS %s", fieldAccess, outputName)
	case tsTypes.AggregationCount:
		return fmt.Sprintf("COUNT(%s) AS %s", fieldAccess, outputName)
	default:
		return fmt.Sprintf("AVG(%s) AS %s", fieldAccess, outputName)
	}
}

// singleQuote wraps a string in single quotes for SQL.
func singleQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", "''") + "'"
}

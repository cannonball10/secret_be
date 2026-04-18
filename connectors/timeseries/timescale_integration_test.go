package timeseries_test

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/cannonball10/foundation/connectors/timeseries"
	tsTypes "github.com/cannonball10/foundation/schemas/timeseries"
	"github.com/joho/godotenv"
)

func TestTimescaleIntegration(t *testing.T) {
	godotenv.Load()
	if os.Getenv("TIMESCALE_DSN") == "" {
		t.Skip("TIMESCALE_DSN not set, skipping integration test")
	}

	ctx := context.Background()

	conn, err := timeseries.DefaultTimescaleConnector(ctx)
	if err != nil {
		t.Fatalf("Failed to create connector: %v", err)
	}
	defer conn.Close()

	// Test Ping
	if err := conn.Ping(ctx); err != nil {
		t.Fatalf("Ping failed: %v", err)
	}
	t.Log("Ping successful")

	measurement := "test_integration_" + time.Now().Format("20060102150405")

	// Test CreateMeasurement
	err = conn.CreateMeasurement(ctx, measurement, &tsTypes.MeasurementOptions{
		ChunkInterval: 24 * time.Hour,
	})
	if err != nil {
		t.Fatalf("CreateMeasurement failed: %v", err)
	}
	t.Logf("Created measurement: %s", measurement)

	// Cleanup on exit
	defer func() {
		if err := conn.DropMeasurement(ctx, measurement); err != nil {
			t.Errorf("DropMeasurement failed: %v", err)
		} else {
			t.Logf("Dropped measurement: %s", measurement)
		}
	}()

	// Test WritePoints
	now := time.Now().UTC().Truncate(time.Second)
	points := []tsTypes.Point{
		{
			Time:   now.Add(-2 * time.Hour),
			Tags:   map[string]string{"symbol": "BTC", "exchange": "binance"},
			Fields: map[string]float64{"price": 42000.50, "volume": 100.5},
		},
		{
			Time:   now.Add(-1 * time.Hour),
			Tags:   map[string]string{"symbol": "BTC", "exchange": "binance"},
			Fields: map[string]float64{"price": 42100.75, "volume": 150.2},
		},
		{
			Time:   now,
			Tags:   map[string]string{"symbol": "BTC", "exchange": "binance"},
			Fields: map[string]float64{"price": 42200.00, "volume": 200.0},
		},
	}

	err = conn.WritePoints(ctx, measurement, points)
	if err != nil {
		t.Fatalf("WritePoints failed: %v", err)
	}
	t.Log("Wrote 3 points")

	// Test Query
	result, err := conn.Query(ctx, tsTypes.QueryInput{
		Measurement: measurement,
		Start:       now.Add(-3 * time.Hour),
		End:         now.Add(time.Hour),
		Tags:        map[string]string{"symbol": "BTC"},
	})
	if err != nil {
		t.Fatalf("Query failed: %v", err)
	}
	if len(result) != 3 {
		t.Errorf("Expected 3 points, got %d", len(result))
	}
	t.Logf("Queried %d points", len(result))

	// Test QueryAggregate
	aggResult, err := conn.QueryAggregate(ctx, tsTypes.AggregateQueryInput{
		Measurement: measurement,
		Start:       now.Add(-3 * time.Hour),
		End:         now.Add(time.Hour),
		Interval:    3 * time.Hour,
		Tags:        map[string]string{"symbol": "BTC"},
		Aggregations: []tsTypes.AggregationSpec{
			{OutputName: "open", Field: "price", Type: tsTypes.AggregationFirst},
			{OutputName: "close", Field: "price", Type: tsTypes.AggregationLast},
			{OutputName: "high", Field: "price", Type: tsTypes.AggregationMax},
			{OutputName: "low", Field: "price", Type: tsTypes.AggregationMin},
			{OutputName: "volume", Field: "volume", Type: tsTypes.AggregationSum},
		},
	})
	if err != nil {
		t.Fatalf("QueryAggregate failed: %v", err)
	}
	if len(aggResult) == 0 {
		t.Error("Expected at least 1 aggregate point")
	}
	t.Logf("Got %d aggregate buckets", len(aggResult))

	if len(aggResult) > 0 {
		agg := aggResult[0]
		t.Logf("First bucket - open: %.2f, close: %.2f, high: %.2f, low: %.2f, volume: %.2f",
			agg.Values["open"], agg.Values["close"], agg.Values["high"], agg.Values["low"], agg.Values["volume"])
	}
}

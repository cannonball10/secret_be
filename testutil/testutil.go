//go:build integration

package testutil

import (
	"fmt"
	"net"
	"testing"
	"time"
)

// Service ports for local Docker services
const (
	RedisAddr    = "127.0.0.1:6379"
	DynamoAddr   = "127.0.0.1:8000"
	Neo4jAddr    = "127.0.0.1:7687"
	QdrantAddr   = "127.0.0.1:6334"
)

// SkipIfServiceUnavailable skips the test if the service at addr is not available.
func SkipIfServiceUnavailable(t *testing.T, addr, name string) {
	t.Helper()
	conn, err := net.DialTimeout("tcp", addr, 2*time.Second)
	if err != nil {
		t.Skipf("Skipping %s tests: service not available at %s", name, addr)
	}
	conn.Close()
}

// TestPrefix generates a unique prefix for test isolation.
// Format: test_<TestName>_<timestamp>_
func TestPrefix(t *testing.T) string {
	t.Helper()
	return fmt.Sprintf("test_%s_%d_", t.Name(), time.Now().UnixNano())
}

// Retry executes the given function up to maxAttempts times with exponential backoff.
// It returns nil on success, or the last error if all attempts fail.
func Retry(maxAttempts int, initialDelay time.Duration, fn func() error) error {
	var lastErr error
	delay := initialDelay

	for attempt := 1; attempt <= maxAttempts; attempt++ {
		if err := fn(); err == nil {
			return nil
		} else {
			lastErr = err
		}

		if attempt < maxAttempts {
			time.Sleep(delay)
			delay *= 2
		}
	}

	return lastErr
}

// RetryWithResult executes the given function up to maxAttempts times with exponential backoff.
// It returns the result and nil on success, or zero value and the last error if all attempts fail.
func RetryWithResult[T any](maxAttempts int, initialDelay time.Duration, fn func() (T, error)) (T, error) {
	var lastErr error
	var zero T
	delay := initialDelay

	for attempt := 1; attempt <= maxAttempts; attempt++ {
		result, err := fn()
		if err == nil {
			return result, nil
		}
		lastErr = err

		if attempt < maxAttempts {
			time.Sleep(delay)
			delay *= 2
		}
	}

	return zero, lastErr
}

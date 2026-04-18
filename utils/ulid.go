package utils

import (
	"crypto/rand"
	"time"

	"github.com/oklog/ulid/v2"
)

// GenerateULID generates a new ULID string.
// ULIDs are lexicographically sortable, URL-safe, and contain a timestamp.
func GenerateULID() string {
	entropy := rand.Reader
	ms := ulid.Timestamp(time.Now())
	id, err := ulid.New(ms, entropy)
	if err != nil {
		// This should never happen with proper entropy source
		// Fallback to timestamp-based generation
		ms = ulid.Timestamp(time.Now())
		id = ulid.MustNew(ms, entropy)
	}
	return id.String()
}

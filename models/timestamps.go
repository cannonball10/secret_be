package models

import "time"

// Timestamps tracks creation and modification times.
type Timestamps struct {
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

// NewTimestamps returns a Timestamps with both fields set to now (UTC).
func NewTimestamps() Timestamps {
	now := time.Now().UTC()
	return Timestamps{CreatedAt: now, UpdatedAt: now}
}

// Touch updates UpdatedAt to now (UTC).
func (t *Timestamps) Touch() {
	t.UpdatedAt = time.Now().UTC()
}

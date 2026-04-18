package models

import (
	"errors"
	"fmt"
	"strings"
	"sync"
)

var (
	ErrModelNotFound = errors.New("no model found matching PK/SK pattern")
	ErrMissingPK     = errors.New("PK is required but missing")
	ErrMissingSK     = errors.New("SK is required but missing")
)

// ModelFactory is a function that creates a new instance of a Model.
type ModelFactory func() Model

// modelEntry stores a registered model's prefixes and factory.
type modelEntry struct {
	pkPrefix string
	skPrefix string
	factory  ModelFactory
}

// ModelRegistry stores mappings of PK/SK prefixes to model factories.
type ModelRegistry struct {
	mu      sync.RWMutex
	entries []modelEntry
}

// NewModelRegistry creates a new ModelRegistry instance.
func NewModelRegistry() *ModelRegistry {
	return &ModelRegistry{
		entries: make([]modelEntry, 0),
	}
}

// Register registers a model type with its PK and SK prefixes.
// The factory function should return a new instance of the model type.
func (r *ModelRegistry) Register(pkPrefix, skPrefix string, factory ModelFactory) {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Check for duplicates
	for _, entry := range r.entries {
		if entry.pkPrefix == pkPrefix && entry.skPrefix == skPrefix {
			// Already registered, skip
			return
		}
	}

	r.entries = append(r.entries, modelEntry{
		pkPrefix: pkPrefix,
		skPrefix: skPrefix,
		factory:  factory,
	})
}

// Lookup finds and instantiates the correct model based on PK/SK values.
// It matches the PK and SK against registered prefixes and returns a new
// instance of the matching model type.
func (r *ModelRegistry) Lookup(pk, sk string) (Model, error) {
	if pk == "" {
		return nil, ErrMissingPK
	}
	if sk == "" {
		return nil, ErrMissingSK
	}

	r.mu.RLock()
	defer r.mu.RUnlock()

	// Find matching entry by checking if PK and SK start with registered prefixes
	for _, entry := range r.entries {
		if strings.HasPrefix(pk, entry.pkPrefix) && strings.HasPrefix(sk, entry.skPrefix) {
			return entry.factory(), nil
		}
	}

	return nil, fmt.Errorf("%w: PK=%q, SK=%q", ErrModelNotFound, pk, sk)
}

// Global registry instance
var globalRegistry = NewModelRegistry()

// Register registers a model type with the global registry.
func Register(pkPrefix, skPrefix string, factory ModelFactory) {
	globalRegistry.Register(pkPrefix, skPrefix, factory)
}

// RegisterModel registers a model type using a KeyBuilder's prefixes.
func RegisterModel(kb KeyBuilder, factory ModelFactory) {
	pk, sk := kb.Prefixes()
	globalRegistry.Register(pk, sk, factory)
}

// Lookup finds and instantiates the correct model from the global registry.
func Lookup(pk, sk string) (Model, error) {
	return globalRegistry.Lookup(pk, sk)
}

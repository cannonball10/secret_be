package models

import "github.com/cannonball10/foundation/schemas/database"

// KeyBuilder provides key construction for a DynamoDB model type.
type KeyBuilder struct {
	pkPrefix string
	skPrefix string
}

// NewKeyBuilder creates a KeyBuilder with the given PK and SK prefixes.
func NewKeyBuilder(pkPrefix, skPrefix string) KeyBuilder {
	return KeyBuilder{pkPrefix: pkPrefix, skPrefix: skPrefix}
}

// PK returns the partition key for the given ID.
func (kb KeyBuilder) PK(id string) string {
	return kb.pkPrefix + id
}

// SK returns the sort key for the given ID.
func (kb KeyBuilder) SK(id string) string {
	return kb.skPrefix + id
}

// Key returns a database.Key for the given ID.
func (kb KeyBuilder) Key(id string) database.Key {
	return database.Key{
		"PK": kb.PK(id),
		"SK": kb.SK(id),
	}
}

// Pair returns a GSIKeyPair for the given PK and SK values.
func (kb KeyBuilder) Pair(pkID, skID string) GSIKeyPair {
	return GSIKeyPair{
		PK: kb.PK(pkID),
		SK: kb.SK(skID),
	}
}

// CompositeKey returns a database.Key using separate PK and SK IDs.
func (kb KeyBuilder) CompositeKey(pkID, skID string) database.Key {
	return database.Key{
		"PK": kb.PK(pkID),
		"SK": kb.SK(skID),
	}
}

// Prefixes returns the PK and SK prefix strings.
func (kb KeyBuilder) Prefixes() (string, string) {
	return kb.pkPrefix, kb.skPrefix
}

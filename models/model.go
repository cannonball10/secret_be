package models

// GSIKeyPair holds a partition key and sort key for a Global Secondary Index.
type GSIKeyPair struct {
	PK string
	SK string
}

// Model is the interface that all DynamoDB-backed models must implement.
type Model interface {
	// PK returns the partition key for this model.
	PK() string

	// SK returns the sort key for this model.
	SK() string

	// GSIs returns the GSI key pairs keyed by index number (e.g. 1 → GSI1PK/GSI1SK).
	GSIs() map[int]GSIKeyPair
}

// Listable is an optional interface for models that support list queries
// via the LIST GSI. Models that implement this get LISTPK/LISTSK attributes
// written automatically.
type Listable interface {
	ListPK() string // Entity type partition key (e.g. "PROMPT")
	ListSK() string // Time-sortable key (e.g. ULID-based)
}

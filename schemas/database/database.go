package database

// Key represents the unique identifier for a document.
type Key map[string]string

// SortDirection determines the sort order for query results.
type SortDirection string

const (
	SortAscending  SortDirection = "asc"
	SortDescending SortDirection = "desc"
)

type IndexName string

const (
	IndexName_GSI1 IndexName = "GSI1"
	IndexName_GSI2 IndexName = "GSI2"
	IndexName_LIST IndexName = "LIST"
)

// SortKeyCondition expresses a condition on the sort key attribute.
// Exactly one field should be set.
type SortKeyCondition struct {
	EQ         *string
	BeginsWith *string
	Between    *[2]string
	LT         *string
	LE         *string
	GT         *string
	GE         *string
}

// QueryInput describes a DynamoDB Query operation.
type QueryInput struct {
	PartitionKey string
	SortKey      *SortKeyCondition
	IndexName    *IndexName
}

// QueryOptions provides optional behavior for query operations.
type QueryOptions struct {
	Limit      int
	Direction  SortDirection
	Projection []string
	Cursor     map[string]string
}

// QueryPage contains pagination state returned from a Query operation.
type QueryPage struct {
	NextCursor map[string]string // nil when no more pages
}

// QueryOutput wraps the results of a Query operation.
type QueryOutput struct {
	Models []any
	Page   *QueryPage
}

// BulkOptions provides optional behavior for bulk write operations.
type BulkOptions struct {
	Atomic bool
}

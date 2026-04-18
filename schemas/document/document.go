package document

// DocumentStatus represents the processing state of a document.
type DocumentStatus string

const (
	DocumentStatusPending    DocumentStatus = "pending"
	DocumentStatusProcessing DocumentStatus = "processing"
	DocumentStatusCompleted  DocumentStatus = "completed"
	DocumentStatusFailed     DocumentStatus = "failed"
)

// String returns the string representation of the status.
func (s DocumentStatus) String() string {
	return string(s)
}

// IsTerminal returns true if the status is a final state (completed or failed).
func (s DocumentStatus) IsTerminal() bool {
	return s == DocumentStatusCompleted || s == DocumentStatusFailed
}

// Default configuration values for PDF processing.
const (
	DefaultChunkSize    = 512
	DefaultChunkOverlap = 80
	DefaultCollection   = "documents"

	// CharsPerToken is the estimated number of characters per token.
	// Used for approximate token counting.
	CharsPerToken = 4
)

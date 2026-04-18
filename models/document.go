package models

import (
	"github.com/cannonball10/foundation/schemas/document"
	"github.com/cannonball10/foundation/utils"
)

// DocumentKeys provides key construction for the Document model.
// PK: DOC#{documentId}, SK: DOC#{documentId}
var DocumentKeys = NewKeyBuilder("DOC#", "DOC#")

// DocumentGSI1Keys provides key construction for Document GSI1 (status lookup).
// GSI1PK: STATUS#{status}, GSI1SK: DOC#{documentId}
var DocumentGSI1Keys = NewKeyBuilder("STATUS#", "DOC#")

// Document represents a PDF document that has been ingested into the system.
type Document struct {
	Timestamps

	DocumentID   string                  `json:"documentId"`
	Filename     string                  `json:"filename"`
	SourcePath   string                  `json:"sourcePath"`
	Collection   string                  `json:"collection"`
	Status       document.DocumentStatus `json:"status"`
	ChunkCount   int                     `json:"chunkCount"`
	TotalTokens  int                     `json:"totalTokens"`
	ErrorMessage string                  `json:"errorMessage,omitempty"`
	Metadata     map[string]any          `json:"metadata,omitempty"`
}

// NewDocument creates a new Document. If id is empty, a ULID will be auto-generated.
func NewDocument(id *string, filename, sourcePath, collection string) *Document {
	if id == nil || *id == "" {
		ulid := utils.GenerateULID()
		id = &ulid
	}

	return &Document{
		Timestamps:  NewTimestamps(),
		DocumentID:  *id,
		Filename:    filename,
		SourcePath:  sourcePath,
		Collection:  collection,
		Status:      document.DocumentStatusPending,
		ChunkCount:  0,
		TotalTokens: 0,
		Metadata:    make(map[string]any),
	}
}

// PK returns the partition key for this document.
func (d *Document) PK() string {
	return DocumentKeys.PK(d.DocumentID)
}

// SK returns the sort key for this document.
func (d *Document) SK() string {
	return DocumentKeys.SK(d.DocumentID)
}

// GSIs returns the GSI key pairs for this document.
func (d *Document) GSIs() map[int]GSIKeyPair {
	return map[int]GSIKeyPair{
		1: DocumentGSI1Keys.Pair(string(d.Status), d.DocumentID),
	}
}

// SetStatus updates the document status and touches the UpdatedAt timestamp.
func (d *Document) SetStatus(status document.DocumentStatus) {
	d.Status = status
	d.Touch()
}

// SetError sets the document to failed status with an error message.
func (d *Document) SetError(err error) {
	d.Status = document.DocumentStatusFailed
	if err != nil {
		d.ErrorMessage = err.Error()
	}
	d.Touch()
}

// SetCompleted marks the document as completed with chunk statistics.
func (d *Document) SetCompleted(chunkCount, totalTokens int) {
	d.Status = document.DocumentStatusCompleted
	d.ChunkCount = chunkCount
	d.TotalTokens = totalTokens
	d.Touch()
}

func init() {
	RegisterModel(DocumentKeys, func() Model { return &Document{} })
}

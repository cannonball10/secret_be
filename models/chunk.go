package models

import (
	"github.com/cannonball10/foundation/utils"
)

// ChunkKeys provides key construction for the Chunk model.
// PK: DOC#{documentId}, SK: CHUNK#{chunkId}
// This allows efficient retrieval of all chunks for a document.
var ChunkKeys = NewKeyBuilder("DOC#", "CHUNK#")

// Chunk represents a text segment from a processed document.
type Chunk struct {
	Timestamps

	ChunkID     string `json:"chunkId"`
	DocumentID  string `json:"documentId"`
	VectorID    string `json:"vectorId"`
	Sequence    int    `json:"sequence"`
	StartOffset int    `json:"startOffset"`
	EndOffset   int    `json:"endOffset"`
	TokenCount  int    `json:"tokenCount"`
	Text        string `json:"text"`
	PageNumber  int    `json:"pageNumber,omitempty"`
}

// NewChunk creates a new Chunk. If chunkID is empty, a ULID will be auto-generated.
func NewChunk(chunkID *string, documentID string, sequence int, text string, startOffset, endOffset, tokenCount int) *Chunk {
	if chunkID == nil || *chunkID == "" {
		ulid := utils.GenerateULID()
		chunkID = &ulid
	}

	return &Chunk{
		Timestamps:  NewTimestamps(),
		ChunkID:     *chunkID,
		DocumentID:  documentID,
		Sequence:    sequence,
		Text:        text,
		StartOffset: startOffset,
		EndOffset:   endOffset,
		TokenCount:  tokenCount,
	}
}

// PK returns the partition key for this chunk.
// Uses the document ID to allow efficient queries for all chunks of a document.
func (c *Chunk) PK() string {
	return ChunkKeys.PK(c.DocumentID)
}

// SK returns the sort key for this chunk.
func (c *Chunk) SK() string {
	return ChunkKeys.SK(c.ChunkID)
}

// GSIs returns the GSI key pairs for this chunk.
// Chunks don't need GSIs as they're always accessed via document ID.
func (c *Chunk) GSIs() map[int]GSIKeyPair {
	return nil
}

// SetVectorID sets the Qdrant vector point ID for this chunk.
func (c *Chunk) SetVectorID(vectorID string) {
	c.VectorID = vectorID
	c.Touch()
}

func init() {
	RegisterModel(ChunkKeys, func() Model { return &Chunk{} })
}

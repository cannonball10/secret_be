package models

import (
	"testing"

	"github.com/cannonball10/foundation/schemas/document"
)

func TestNewDocument(t *testing.T) {
	doc := NewDocument(nil, "test.pdf", "/path/to/test.pdf", "my-collection")

	if doc.DocumentID == "" {
		t.Error("DocumentID should be auto-generated")
	}

	if doc.Filename != "test.pdf" {
		t.Errorf("Filename = %q, want %q", doc.Filename, "test.pdf")
	}

	if doc.SourcePath != "/path/to/test.pdf" {
		t.Errorf("SourcePath = %q, want %q", doc.SourcePath, "/path/to/test.pdf")
	}

	if doc.Collection != "my-collection" {
		t.Errorf("Collection = %q, want %q", doc.Collection, "my-collection")
	}

	if doc.Status != document.DocumentStatusPending {
		t.Errorf("Status = %q, want %q", doc.Status, document.DocumentStatusPending)
	}
}

func TestDocument_Keys(t *testing.T) {
	id := "01H123ABC"
	doc := NewDocument(&id, "test.pdf", "/path", "collection")

	expectedPK := "DOC#01H123ABC"
	expectedSK := "DOC#01H123ABC"

	if doc.PK() != expectedPK {
		t.Errorf("PK() = %q, want %q", doc.PK(), expectedPK)
	}

	if doc.SK() != expectedSK {
		t.Errorf("SK() = %q, want %q", doc.SK(), expectedSK)
	}
}

func TestDocument_GSIs(t *testing.T) {
	doc := NewDocument(nil, "test.pdf", "/path", "collection")
	doc.Status = document.DocumentStatusCompleted

	gsis := doc.GSIs()

	if len(gsis) != 1 {
		t.Errorf("Expected 1 GSI, got %d", len(gsis))
	}

	gsi1 := gsis[1]
	expectedPK := "STATUS#completed"
	if gsi1.PK != expectedPK {
		t.Errorf("GSI1.PK = %q, want %q", gsi1.PK, expectedPK)
	}
}

func TestDocument_SetStatus(t *testing.T) {
	doc := NewDocument(nil, "test.pdf", "/path", "collection")
	originalUpdatedAt := doc.UpdatedAt

	doc.SetStatus(document.DocumentStatusProcessing)

	if doc.Status != document.DocumentStatusProcessing {
		t.Errorf("Status = %q, want %q", doc.Status, document.DocumentStatusProcessing)
	}

	if !doc.UpdatedAt.After(originalUpdatedAt) && doc.UpdatedAt != originalUpdatedAt {
		t.Error("UpdatedAt should be updated")
	}
}

func TestDocument_SetCompleted(t *testing.T) {
	doc := NewDocument(nil, "test.pdf", "/path", "collection")

	doc.SetCompleted(10, 5000)

	if doc.Status != document.DocumentStatusCompleted {
		t.Errorf("Status = %q, want %q", doc.Status, document.DocumentStatusCompleted)
	}

	if doc.ChunkCount != 10 {
		t.Errorf("ChunkCount = %d, want %d", doc.ChunkCount, 10)
	}

	if doc.TotalTokens != 5000 {
		t.Errorf("TotalTokens = %d, want %d", doc.TotalTokens, 5000)
	}
}

func TestDocument_SetError(t *testing.T) {
	doc := NewDocument(nil, "test.pdf", "/path", "collection")

	err := &testError{msg: "something went wrong"}
	doc.SetError(err)

	if doc.Status != document.DocumentStatusFailed {
		t.Errorf("Status = %q, want %q", doc.Status, document.DocumentStatusFailed)
	}

	if doc.ErrorMessage != "something went wrong" {
		t.Errorf("ErrorMessage = %q, want %q", doc.ErrorMessage, "something went wrong")
	}
}

type testError struct {
	msg string
}

func (e *testError) Error() string {
	return e.msg
}

package pdf

import (
	"context"
)

// ExtractResult contains the extracted text and metadata from a PDF.
type ExtractResult struct {
	// Text is the full extracted text content.
	Text string

	// Pages contains per-page text if available.
	Pages []PageContent

	// PageCount is the total number of pages in the PDF.
	PageCount int
}

// PageContent represents the extracted content from a single page.
type PageContent struct {
	PageNumber int
	Text       string
}

// PDFConnector defines the interface for PDF text extraction.
type PDFConnector interface {
	// ExtractText extracts all text content from a PDF file at the given path.
	ExtractText(ctx context.Context, filePath string) (*ExtractResult, error)
}

// DefaultPDFConnector returns the default PDF connector implementation.
func DefaultPDFConnector(ctx context.Context) (PDFConnector, error) {
	return NewPDFCPUConnector(), nil
}

package pdf

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	pdfErrors "github.com/cannonball10/foundation/errors/pdf"
	"github.com/pdfcpu/pdfcpu/pkg/api"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
)

var _ PDFConnector = (*PDFCPUConnector)(nil)

// PDFCPUConnector implements PDFConnector using the pdfcpu library.
type PDFCPUConnector struct{}

// NewPDFCPUConnector creates a new PDFCPUConnector.
func NewPDFCPUConnector() *PDFCPUConnector {
	return &PDFCPUConnector{}
}

// ExtractText extracts all text content from a PDF file.
func (c *PDFCPUConnector) ExtractText(ctx context.Context, filePath string) (*ExtractResult, error) {
	// Check if file exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return nil, fmt.Errorf("%w: %s", pdfErrors.ErrPDFFileNotFound, filePath)
	}

	// Open the PDF file and read context for page count
	pdfCtx, err := api.ReadContextFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", pdfErrors.ErrPDFParseError, err)
	}

	pageCount := pdfCtx.PageCount

	// Create a temporary directory for content extraction
	tempDir, err := os.MkdirTemp("", "pdfcpu-extract-*")
	if err != nil {
		return nil, fmt.Errorf("%w: failed to create temp dir: %v", pdfErrors.ErrPDFParseError, err)
	}
	defer os.RemoveAll(tempDir)

	// Extract content from all pages
	conf := model.NewDefaultConfiguration()
	if err := api.ExtractContentFile(filePath, tempDir, nil, conf); err != nil {
		return nil, fmt.Errorf("%w: %v", pdfErrors.ErrPDFParseError, err)
	}

	// Read extracted content files
	var allText strings.Builder
	pages := make([]PageContent, 0, pageCount)

	entries, err := os.ReadDir(tempDir)
	if err != nil {
		return nil, fmt.Errorf("%w: failed to read temp dir: %v", pdfErrors.ErrPDFParseError, err)
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		content, err := os.ReadFile(filepath.Join(tempDir, entry.Name()))
		if err != nil {
			continue
		}
		text := cleanExtractedText(string(content))
		if text != "" {
			if allText.Len() > 0 {
				allText.WriteString("\n\n")
			}
			allText.WriteString(text)
		}
	}

	// Try per-page extraction for better granularity
	pages, _ = c.extractPerPage(filePath, pageCount, conf, tempDir)

	text := allText.String()
	if strings.TrimSpace(text) == "" {
		return nil, pdfErrors.ErrPDFNoTextContent
	}

	result := &ExtractResult{
		Text:      text,
		PageCount: pageCount,
		Pages:     pages,
	}

	return result, nil
}

// extractPerPage attempts to extract text from each page individually.
func (c *PDFCPUConnector) extractPerPage(filePath string, pageCount int, conf *model.Configuration, tempDir string) ([]PageContent, error) {
	pages := make([]PageContent, 0, pageCount)

	for i := 1; i <= pageCount; i++ {
		pageDir := filepath.Join(tempDir, fmt.Sprintf("page_%d", i))
		if err := os.MkdirAll(pageDir, 0755); err != nil {
			continue
		}

		pageSelection := []string{fmt.Sprintf("%d", i)}
		if err := api.ExtractContentFile(filePath, pageDir, pageSelection, conf); err != nil {
			continue
		}

		// Read extracted page content
		entries, err := os.ReadDir(pageDir)
		if err != nil {
			continue
		}

		var pageText strings.Builder
		for _, entry := range entries {
			if entry.IsDir() {
				continue
			}
			content, err := os.ReadFile(filepath.Join(pageDir, entry.Name()))
			if err != nil {
				continue
			}
			text := cleanExtractedText(string(content))
			if text != "" {
				if pageText.Len() > 0 {
					pageText.WriteString("\n")
				}
				pageText.WriteString(text)
			}
		}

		pages = append(pages, PageContent{
			PageNumber: i,
			Text:       pageText.String(),
		})
	}

	return pages, nil
}

// cleanExtractedText normalizes extracted text by removing excessive whitespace.
func cleanExtractedText(text string) string {
	// Replace multiple newlines with double newlines
	text = strings.ReplaceAll(text, "\r\n", "\n")
	text = strings.ReplaceAll(text, "\r", "\n")

	// Collapse multiple spaces into single space
	var result strings.Builder
	prevSpace := false
	prevNewline := false

	for _, r := range text {
		if r == '\n' {
			if !prevNewline {
				result.WriteRune('\n')
				prevNewline = true
			}
			prevSpace = false
		} else if r == ' ' || r == '\t' {
			if !prevSpace && !prevNewline {
				result.WriteRune(' ')
				prevSpace = true
			}
		} else {
			result.WriteRune(r)
			prevSpace = false
			prevNewline = false
		}
	}

	return strings.TrimSpace(result.String())
}

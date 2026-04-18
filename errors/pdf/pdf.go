package pdf

import "errors"

var (
	ErrPDFClientUninitialized = errors.New("pdf client is not initialized")
	ErrPDFFileNotFound        = errors.New("pdf file not found")
	ErrPDFInvalidFile         = errors.New("invalid pdf file")
	ErrPDFNoTextContent       = errors.New("pdf contains no extractable text")
	ErrPDFParseError          = errors.New("failed to parse pdf")
)

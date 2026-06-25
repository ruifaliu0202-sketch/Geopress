package knowledge

import (
	"context"
	"errors"
	"fmt"
)

const (
	FileKindUnknown  FileKind = "unknown"
	FileKindText     FileKind = "text"
	FileKindMarkdown FileKind = "markdown"
	FileKindDOC      FileKind = "doc"
	FileKindDOCX     FileKind = "docx"
	FileKindPDF      FileKind = "pdf"
	FileKindImage    FileKind = "image"

	ExtractionStatusSucceeded   ExtractionStatus = "succeeded"
	ExtractionStatusUnsupported ExtractionStatus = "unsupported"
	ExtractionStatusEmpty       ExtractionStatus = "empty"
	ExtractionStatusFailed      ExtractionStatus = "failed"
)

var (
	ErrExtractionUnsupported = errors.New("knowledge extraction unsupported")
	ErrInvalidDocument       = errors.New("invalid knowledge document")
)

type FileKind string

type ExtractionStatus string

type FileType struct {
	Kind      FileKind
	MimeType  string
	Extension string
}

type ExtractInput struct {
	Filename string
	MimeType string
	Data     []byte
}

type OCRProvider interface {
	ExtractText(ctx context.Context, input OCRInput) (OCRResult, error)
}

type OCRInput struct {
	Filename string
	MimeType string
	Data     []byte
	FileType FileType
}

type OCRResult struct {
	Text     string
	Metadata map[string]any
}

type ExtractionResult struct {
	FileType     FileType
	Text         string
	Status       ExtractionStatus
	ErrorMessage string
	Metadata     map[string]any
}

type ProcessInput struct {
	Context              context.Context
	AssetTitle           string
	KnowledgeBaseName    string
	Filename             string
	MimeType             string
	Data                 []byte
	Summary              string
	Tags                 []string
	Metadata             map[string]any
	OCRProvider          OCRProvider
	OCRUnavailableReason string
	ChunkOptions         ChunkOptions
}

type ProcessResult struct {
	Extraction ExtractionResult
	Chunks     []Chunk
}

type ChunkOptions struct {
	MinChars     int
	MaxChars     int
	OverlapChars int
}

type ChunkInput struct {
	AssetTitle        string
	KnowledgeBaseName string
	Text              string
	Summary           string
	Tags              []string
	Metadata          map[string]any
	Options           ChunkOptions
}

type Chunk struct {
	Title      string         `json:"title"`
	Content    string         `json:"content"`
	SearchText string         `json:"searchText"`
	Summary    string         `json:"summary"`
	Tags       []string       `json:"tags"`
	Metadata   map[string]any `json:"metadata"`
	ChunkIndex int            `json:"chunkIndex"`
}

type UnsupportedExtractionError struct {
	Kind   FileKind
	Reason string
}

func (err UnsupportedExtractionError) Error() string {
	if err.Kind == "" {
		return ErrExtractionUnsupported.Error()
	}
	if err.Reason == "" {
		return fmt.Sprintf("%s: %s", ErrExtractionUnsupported, err.Kind)
	}
	return fmt.Sprintf("%s: %s: %s", ErrExtractionUnsupported, err.Kind, err.Reason)
}

func (err UnsupportedExtractionError) Unwrap() error {
	return ErrExtractionUnsupported
}

func Process(input ProcessInput) (ProcessResult, error) {
	extraction, err := ExtractText(ExtractInput{
		Filename: input.Filename,
		MimeType: input.MimeType,
		Data:     input.Data,
	})
	if err != nil {
		if errors.Is(err, ErrExtractionUnsupported) && requiresOCR(extraction.FileType.Kind) {
			extraction, err = extractWithOCR(input, extraction)
			if err != nil {
				return ProcessResult{Extraction: extraction, Chunks: []Chunk{}}, err
			}
		} else {
			return ProcessResult{Extraction: extraction, Chunks: []Chunk{}}, err
		}
	}

	chunks := ChunkText(ChunkInput{
		AssetTitle:        input.AssetTitle,
		KnowledgeBaseName: input.KnowledgeBaseName,
		Text:              extraction.Text,
		Summary:           input.Summary,
		Tags:              input.Tags,
		Metadata:          mergeMetadata(input.Metadata, extraction.Metadata),
		Options:           input.ChunkOptions,
	})

	return ProcessResult{
		Extraction: extraction,
		Chunks:     chunks,
	}, nil
}

func extractWithOCR(input ProcessInput, extraction ExtractionResult) (ExtractionResult, error) {
	if input.OCRProvider == nil {
		reason := input.OCRUnavailableReason
		if reason == "" {
			reason = "ai vision ocr provider is not configured"
		}
		err := UnsupportedExtractionError{Kind: extraction.FileType.Kind, Reason: reason}
		extraction.Status = ExtractionStatusUnsupported
		extraction.ErrorMessage = err.Error()
		return extraction, err
	}

	ctx := input.Context
	if ctx == nil {
		ctx = context.Background()
	}
	result, err := input.OCRProvider.ExtractText(ctx, OCRInput{
		Filename: input.Filename,
		MimeType: input.MimeType,
		Data:     input.Data,
		FileType: extraction.FileType,
	})
	if err != nil {
		extraction.Status = ExtractionStatusFailed
		extraction.ErrorMessage = err.Error()
		return extraction, err
	}

	text := CleanExtractedText(result.Text)
	if text == "" {
		err := fmt.Errorf("%w: OCR extracted text is empty", ErrInvalidDocument)
		extraction.Status = ExtractionStatusEmpty
		extraction.ErrorMessage = err.Error()
		return extraction, err
	}

	extraction.Text = text
	extraction.Status = ExtractionStatusSucceeded
	extraction.ErrorMessage = ""
	if extraction.Metadata == nil {
		extraction.Metadata = map[string]any{}
	}
	for key, value := range result.Metadata {
		extraction.Metadata[key] = value
	}
	return extraction, nil
}

func requiresOCR(kind FileKind) bool {
	return kind == FileKindImage || kind == FileKindPDF
}

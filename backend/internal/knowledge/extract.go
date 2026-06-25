package knowledge

import (
	"archive/zip"
	"bytes"
	"encoding/binary"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"path/filepath"
	"regexp"
	"strings"
	"unicode"
	"unicode/utf16"
	"unicode/utf8"
)

var (
	markdownExtensions = map[string]bool{".md": true, ".markdown": true, ".mdown": true, ".mkd": true}
	textExtensions     = map[string]bool{".txt": true, ".text": true, ".csv": true, ".log": true}
	imageExtensions    = map[string]bool{".png": true, ".jpg": true, ".jpeg": true, ".webp": true, ".gif": true, ".bmp": true, ".tif": true, ".tiff": true, ".heic": true}
	textLikeMIMEs      = map[string]bool{
		"text/plain":                 true,
		"text/csv":                   true,
		"application/csv":            true,
		"application/json":           true,
		"application/xml":            true,
		"application/x-yaml":         true,
		"application/yaml":           true,
		"text/xml":                   true,
		"text/x-markdown":            true,
		"text/markdown":              true,
		"application/octet-stream":   true,
		"application/x-empty":        true,
		"application/x-subrip":       true,
		"application/vnd.ms-excel":   true,
		"application/vnd.ms-outlook": true,
	}
)

func ExtractText(input ExtractInput) (ExtractionResult, error) {
	fileType := DetectFileType(input.Filename, input.MimeType, input.Data)
	result := ExtractionResult{
		FileType: fileType,
		Metadata: map[string]any{
			"filename":  input.Filename,
			"mimeType":  fileType.MimeType,
			"extension": fileType.Extension,
			"fileKind":  string(fileType.Kind),
			"byteSize":  len(input.Data),
		},
	}

	var (
		text string
		err  error
	)
	switch fileType.Kind {
	case FileKindText, FileKindMarkdown:
		text, err = extractPlainText(input.Data)
	case FileKindDOCX:
		text, err = extractDOCXText(input.Data)
	case FileKindDOC:
		err = UnsupportedExtractionError{Kind: fileType.Kind, Reason: "legacy .doc binary format requires an external converter"}
	case FileKindPDF:
		err = UnsupportedExtractionError{Kind: fileType.Kind, Reason: "pdf text extraction is not configured"}
	case FileKindImage:
		err = UnsupportedExtractionError{Kind: fileType.Kind, Reason: "ocr is not configured"}
	default:
		err = UnsupportedExtractionError{Kind: fileType.Kind, Reason: "file type is not supported for extraction"}
	}
	if err != nil {
		if errors.Is(err, ErrExtractionUnsupported) {
			result.Status = ExtractionStatusUnsupported
		} else {
			result.Status = ExtractionStatusFailed
		}
		result.ErrorMessage = err.Error()
		return result, err
	}

	text = CleanExtractedText(text)
	if text == "" {
		err = fmt.Errorf("%w: extracted text is empty", ErrInvalidDocument)
		result.Status = ExtractionStatusEmpty
		result.ErrorMessage = err.Error()
		return result, err
	}
	result.Text = text
	result.Status = ExtractionStatusSucceeded
	result.Metadata["charCount"] = utf8.RuneCountInString(text)
	return result, nil
}

func DetectFileType(filename string, mimeType string, data []byte) FileType {
	ext := strings.ToLower(filepath.Ext(filename))
	normalizedMIME := normalizeMIME(mimeType)
	if normalizedMIME == "" && len(data) > 0 {
		normalizedMIME = normalizeMIME(http.DetectContentType(data))
	}

	kind := FileKindUnknown
	switch {
	case markdownExtensions[ext]:
		kind = FileKindMarkdown
	case ext == ".docx" || normalizedMIME == "application/vnd.openxmlformats-officedocument.wordprocessingml.document":
		kind = FileKindDOCX
	case ext == ".doc" || normalizedMIME == "application/msword":
		kind = FileKindDOC
	case ext == ".pdf" || normalizedMIME == "application/pdf":
		kind = FileKindPDF
	case imageExtensions[ext] || strings.HasPrefix(normalizedMIME, "image/"):
		kind = FileKindImage
	case textExtensions[ext] || normalizedMIME == "text/markdown" || normalizedMIME == "text/x-markdown" || strings.HasPrefix(normalizedMIME, "text/") || textLikeMIMEs[normalizedMIME]:
		if normalizedMIME == "text/markdown" || normalizedMIME == "text/x-markdown" {
			kind = FileKindMarkdown
		} else {
			kind = FileKindText
		}
	}

	return FileType{
		Kind:      kind,
		MimeType:  normalizedMIME,
		Extension: ext,
	}
}

func CleanExtractedText(text string) string {
	text = strings.ReplaceAll(text, "\r\n", "\n")
	text = strings.ReplaceAll(text, "\r", "\n")
	text = strings.ReplaceAll(text, "\u00a0", " ")
	text = strings.TrimPrefix(text, "\ufeff")

	lines := strings.Split(text, "\n")
	cleaned := make([]string, 0, len(lines))
	blank := false
	for _, line := range lines {
		line = strings.TrimFunc(line, func(r rune) bool {
			return unicode.IsSpace(r) && r != '\n'
		})
		line = collapseInlineWhitespace(line)
		if line == "" {
			if !blank && len(cleaned) > 0 {
				cleaned = append(cleaned, "")
			}
			blank = true
			continue
		}
		cleaned = append(cleaned, line)
		blank = false
	}
	for len(cleaned) > 0 && cleaned[len(cleaned)-1] == "" {
		cleaned = cleaned[:len(cleaned)-1]
	}
	return strings.TrimSpace(strings.Join(cleaned, "\n"))
}

func extractPlainText(data []byte) (string, error) {
	if bytes.HasPrefix(data, []byte{0xff, 0xfe}) {
		return decodeUTF16(data[2:], binary.LittleEndian)
	}
	if bytes.HasPrefix(data, []byte{0xfe, 0xff}) {
		return decodeUTF16(data[2:], binary.BigEndian)
	}
	if !utf8.Valid(data) {
		return "", fmt.Errorf("%w: text file is not valid UTF-8", ErrInvalidDocument)
	}
	return string(data), nil
}

func decodeUTF16(data []byte, order binary.ByteOrder) (string, error) {
	if len(data)%2 != 0 {
		return "", fmt.Errorf("%w: utf-16 text has an odd byte count", ErrInvalidDocument)
	}
	codePoints := make([]uint16, 0, len(data)/2)
	for i := 0; i < len(data); i += 2 {
		codePoints = append(codePoints, order.Uint16(data[i:]))
	}
	return string(utf16.Decode(codePoints)), nil
}

func extractDOCXText(data []byte) (string, error) {
	reader, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return "", fmt.Errorf("%w: open docx zip: %v", ErrInvalidDocument, err)
	}
	for _, file := range reader.File {
		if file.Name != "word/document.xml" {
			continue
		}
		rc, err := file.Open()
		if err != nil {
			return "", fmt.Errorf("%w: open docx document xml: %v", ErrInvalidDocument, err)
		}
		defer rc.Close()
		return parseDOCXDocumentXML(rc)
	}
	return "", fmt.Errorf("%w: docx word/document.xml not found", ErrInvalidDocument)
}

func parseDOCXDocumentXML(reader io.Reader) (string, error) {
	decoder := xml.NewDecoder(reader)
	var builder strings.Builder
	paragraphHasText := false
	needsSpace := false

	for {
		token, err := decoder.Token()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return "", fmt.Errorf("%w: parse docx document xml: %v", ErrInvalidDocument, err)
		}

		switch tok := token.(type) {
		case xml.StartElement:
			switch tok.Name.Local {
			case "tab":
				if paragraphHasText {
					builder.WriteByte(' ')
					needsSpace = false
				}
			case "br", "cr":
				if builder.Len() > 0 {
					builder.WriteByte('\n')
					paragraphHasText = false
					needsSpace = false
				}
			}
		case xml.EndElement:
			switch tok.Name.Local {
			case "p", "tr":
				if paragraphHasText {
					builder.WriteString("\n\n")
					paragraphHasText = false
					needsSpace = false
				}
			case "tc":
				if paragraphHasText {
					builder.WriteByte('\t')
					needsSpace = false
				}
			}
		case xml.CharData:
			value := collapseInlineWhitespace(string(tok))
			if value == "" {
				continue
			}
			if needsSpace && builder.Len() > 0 {
				builder.WriteByte(' ')
			}
			builder.WriteString(value)
			paragraphHasText = true
			needsSpace = true
		}
	}

	return strings.TrimSpace(builder.String()), nil
}

func normalizeMIME(value string) string {
	value = strings.TrimSpace(strings.ToLower(value))
	if value == "" {
		return ""
	}
	mediaType, _, err := mime.ParseMediaType(value)
	if err == nil {
		return strings.ToLower(mediaType)
	}
	if idx := strings.Index(value, ";"); idx >= 0 {
		value = value[:idx]
	}
	return strings.TrimSpace(value)
}

var inlineWhitespaceRE = regexp.MustCompile(`[ \t\f\v]+`)

func collapseInlineWhitespace(value string) string {
	return inlineWhitespaceRE.ReplaceAllString(strings.TrimSpace(value), " ")
}

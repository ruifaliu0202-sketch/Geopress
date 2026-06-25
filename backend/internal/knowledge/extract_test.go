package knowledge

import (
	"archive/zip"
	"bytes"
	"errors"
	"strings"
	"testing"
)

func TestExtractTextPlainUTF8CleansWhitespace(t *testing.T) {
	result, err := ExtractText(ExtractInput{
		Filename: "notes.txt",
		MimeType: "text/plain; charset=utf-8",
		Data:     []byte("\ufeff第一段   内容\r\n\r\n\r\n第二段\t内容  "),
	})
	if err != nil {
		t.Fatalf("ExtractText returned error: %v", err)
	}

	if result.FileType.Kind != FileKindText {
		t.Fatalf("kind = %q, want %q", result.FileType.Kind, FileKindText)
	}
	want := "第一段 内容\n\n第二段 内容"
	if result.Text != want {
		t.Fatalf("text = %q, want %q", result.Text, want)
	}
	if result.Status != ExtractionStatusSucceeded {
		t.Fatalf("status = %q, want %q", result.Status, ExtractionStatusSucceeded)
	}
}

func TestExtractTextDOCXReadsDocumentXML(t *testing.T) {
	data := makeDOCX(t, `<w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main">
<w:body>
<w:p><w:r><w:t>标题</w:t></w:r></w:p>
<w:p><w:r><w:t>第一段</w:t></w:r><w:r><w:tab/></w:r><w:r><w:t>补充</w:t></w:r></w:p>
<w:tbl><w:tr><w:tc><w:p><w:r><w:t>表格A</w:t></w:r></w:p></w:tc><w:tc><w:p><w:r><w:t>表格B</w:t></w:r></w:p></w:tc></w:tr></w:tbl>
</w:body>
</w:document>`)

	result, err := ExtractText(ExtractInput{
		Filename: "brief.docx",
		MimeType: "application/octet-stream",
		Data:     data,
	})
	if err != nil {
		t.Fatalf("ExtractText returned error: %v", err)
	}

	if result.FileType.Kind != FileKindDOCX {
		t.Fatalf("kind = %q, want %q", result.FileType.Kind, FileKindDOCX)
	}
	for _, want := range []string{"标题", "第一段 补充", "表格A", "表格B"} {
		if !strings.Contains(result.Text, want) {
			t.Fatalf("text %q does not contain %q", result.Text, want)
		}
	}
}

func TestExtractTextImageReturnsUnsupported(t *testing.T) {
	_, err := ExtractText(ExtractInput{
		Filename: "photo.png",
		MimeType: "image/png",
		Data:     []byte{0x89, 0x50, 0x4e, 0x47},
	})
	if !errors.Is(err, ErrExtractionUnsupported) {
		t.Fatalf("err = %v, want ErrExtractionUnsupported", err)
	}
}

func makeDOCX(t *testing.T, documentXML string) []byte {
	t.Helper()

	var buffer bytes.Buffer
	writer := zip.NewWriter(&buffer)
	file, err := writer.Create("word/document.xml")
	if err != nil {
		t.Fatalf("create document xml: %v", err)
	}
	if _, err := file.Write([]byte(documentXML)); err != nil {
		t.Fatalf("write document xml: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("close zip: %v", err)
	}
	return buffer.Bytes()
}

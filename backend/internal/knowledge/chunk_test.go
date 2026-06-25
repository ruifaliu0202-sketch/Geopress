package knowledge

import (
	"fmt"
	"strings"
	"testing"
	"unicode/utf8"
)

func TestChunkTextUsesMarkdownHeadingsAndSearchTextContext(t *testing.T) {
	chunks := ChunkText(ChunkInput{
		AssetTitle:        "品牌手册",
		KnowledgeBaseName: "增长知识库",
		Text: strings.Join([]string{
			"# 账号定位",
			"面向新手用户，强调可信和可执行。",
			"",
			"## 内容节奏",
			"每周发布三篇教程，突出案例。",
		}, "\n"),
		Summary: "品牌运营摘要",
		Tags:    []string{"小红书", "增长", "小红书"},
		Metadata: map[string]any{
			"sourceUrl": "https://example.test/source",
			"author":    "ops",
			"ignored":   map[string]any{"nested": true},
		},
		Options: ChunkOptions{MinChars: 20, MaxChars: 80, OverlapChars: 0},
	})
	if len(chunks) != 2 {
		t.Fatalf("len(chunks) = %d, want 2: %#v", len(chunks), chunks)
	}
	if chunks[0].Title != "账号定位" {
		t.Fatalf("chunk[0].Title = %q, want heading title", chunks[0].Title)
	}
	if chunks[1].Title != "内容节奏" {
		t.Fatalf("chunk[1].Title = %q, want heading title", chunks[1].Title)
	}

	searchText := chunks[0].SearchText
	for _, want := range []string{
		"asset: 品牌手册",
		"knowledge_base: 增长知识库",
		"chunk: 账号定位",
		"summary: 品牌运营摘要",
		"tags: 小红书 增长",
		"sourceUrl: https://example.test/source",
		"author: ops",
		"content: 面向新手用户",
	} {
		if !strings.Contains(searchText, want) {
			t.Fatalf("searchText missing %q:\n%s", want, searchText)
		}
	}
	if strings.Contains(searchText, "ignored") {
		t.Fatalf("searchText should not include non-searchable metadata: %s", searchText)
	}
}

func TestChunkTextFallsBackTitleAndSplitsByLength(t *testing.T) {
	paragraphs := make([]string, 0, 5)
	for i := 0; i < 5; i++ {
		paragraphs = append(paragraphs, fmt.Sprintf("第%d段 %s", i+1, strings.Repeat("内容", 20)))
	}

	chunks := ChunkText(ChunkInput{
		AssetTitle:        "无标题资料",
		KnowledgeBaseName: "默认库",
		Text:              strings.Join(paragraphs, "\n\n"),
		Options:           ChunkOptions{MinChars: 40, MaxChars: 70, OverlapChars: 8},
	})
	if len(chunks) < 3 {
		t.Fatalf("len(chunks) = %d, want at least 3", len(chunks))
	}
	for i, chunk := range chunks {
		wantTitle := fmt.Sprintf("无标题资料 片段 %d", i+1)
		if chunk.Title != wantTitle {
			t.Fatalf("chunk[%d].Title = %q, want %q", i, chunk.Title, wantTitle)
		}
		if chunk.ChunkIndex != i {
			t.Fatalf("chunk[%d].ChunkIndex = %d, want %d", i, chunk.ChunkIndex, i)
		}
		if utf8.RuneCountInString(chunk.Content) > 70 {
			t.Fatalf("chunk[%d] length = %d, want <= 70: %q", i, utf8.RuneCountInString(chunk.Content), chunk.Content)
		}
	}
}

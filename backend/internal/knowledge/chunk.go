package knowledge

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
	"unicode/utf8"
)

const (
	defaultMinChunkChars = 300
	defaultMaxChunkChars = 1200
	defaultOverlapChars  = 120
)

type textBlock struct {
	title   string
	content string
}

var markdownHeadingRE = regexp.MustCompile(`^(#{1,6})\s+(.+?)\s*#*\s*$`)

func ChunkText(input ChunkInput) []Chunk {
	text := CleanExtractedText(input.Text)
	if text == "" {
		return []Chunk{}
	}

	options := normalizeChunkOptions(input.Options)
	blocks := splitTextBlocks(text)
	chunkContents := packBlocks(blocks, options)

	chunks := make([]Chunk, 0, len(chunkContents))
	for index, block := range chunkContents {
		content := CleanExtractedText(block.content)
		if content == "" {
			continue
		}
		title := strings.TrimSpace(block.title)
		if title == "" {
			title = fallbackChunkTitle(input.AssetTitle, index)
		}
		metadata := mergeMetadata(input.Metadata, map[string]any{
			"chunkTitle": title,
			"chunkIndex": index,
			"charCount":  utf8.RuneCountInString(content),
		})
		chunk := Chunk{
			Title:      title,
			Content:    content,
			Summary:    strings.TrimSpace(input.Summary),
			Tags:       cleanStringList(input.Tags),
			Metadata:   metadata,
			ChunkIndex: index,
		}
		chunk.SearchText = BuildSearchText(SearchTextInput{
			AssetTitle:        input.AssetTitle,
			KnowledgeBaseName: input.KnowledgeBaseName,
			ChunkTitle:        chunk.Title,
			Summary:           chunk.Summary,
			Tags:              chunk.Tags,
			Metadata:          metadata,
			Content:           chunk.Content,
		})
		chunks = append(chunks, chunk)
	}
	return chunks
}

type SearchTextInput struct {
	AssetTitle        string
	KnowledgeBaseName string
	ChunkTitle        string
	Summary           string
	Tags              []string
	Metadata          map[string]any
	Content           string
}

func BuildSearchText(input SearchTextInput) string {
	parts := []string{
		labeledText("asset", input.AssetTitle),
		labeledText("knowledge_base", input.KnowledgeBaseName),
		labeledText("chunk", input.ChunkTitle),
		labeledText("summary", input.Summary),
		labeledText("tags", strings.Join(cleanStringList(input.Tags), " ")),
		labeledText("metadata", searchableMetadata(input.Metadata)),
		labeledText("content", input.Content),
	}

	cleaned := make([]string, 0, len(parts))
	for _, part := range parts {
		part = CleanExtractedText(part)
		if part != "" {
			cleaned = append(cleaned, part)
		}
	}
	return strings.Join(cleaned, "\n")
}

func splitTextBlocks(text string) []textBlock {
	lines := strings.Split(text, "\n")
	blocks := make([]textBlock, 0)
	currentTitle := ""
	paragraph := make([]string, 0)
	hasHeading := false

	flush := func() {
		content := strings.TrimSpace(strings.Join(paragraph, "\n"))
		if content != "" {
			blocks = append(blocks, textBlock{title: currentTitle, content: content})
		}
		paragraph = paragraph[:0]
	}

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if matches := markdownHeadingRE.FindStringSubmatch(trimmed); matches != nil {
			flush()
			currentTitle = strings.TrimSpace(matches[2])
			hasHeading = true
			continue
		}
		if trimmed == "" {
			flush()
			continue
		}
		paragraph = append(paragraph, trimmed)
	}
	flush()

	if !hasHeading {
		for i := range blocks {
			blocks[i].title = ""
		}
	}
	if len(blocks) == 0 {
		return []textBlock{{content: text}}
	}
	return blocks
}

func packBlocks(blocks []textBlock, options ChunkOptions) []textBlock {
	chunks := make([]textBlock, 0)
	current := textBlock{}

	flush := func() {
		current.content = CleanExtractedText(current.content)
		if current.content != "" {
			chunks = append(chunks, current)
		}
		current = textBlock{}
	}

	for _, block := range blocks {
		block.content = CleanExtractedText(block.content)
		if block.content == "" {
			continue
		}
		if runeLen(block.content) > options.MaxChars {
			flush()
			chunks = append(chunks, splitLongBlock(block, options)...)
			continue
		}

		if current.content == "" {
			current = block
			continue
		}

		if block.title != "" && current.title != "" && block.title != current.title {
			flush()
			current = block
			continue
		}

		next := joinChunkContent(current.content, block.content)
		if runeLen(next) <= options.MaxChars || runeLen(current.content) < options.MinChars {
			current.content = next
			if current.title == "" {
				current.title = block.title
			}
			continue
		}

		flush()
		current = block
	}
	flush()

	if options.OverlapChars > 0 && len(chunks) > 1 {
		applyOverlap(chunks, options.OverlapChars, options.MaxChars)
	}
	return chunks
}

func splitLongBlock(block textBlock, options ChunkOptions) []textBlock {
	runes := []rune(block.content)
	if len(runes) == 0 {
		return []textBlock{}
	}

	step := options.MaxChars - options.OverlapChars
	if step <= 0 {
		step = options.MaxChars
	}

	parts := make([]textBlock, 0, (len(runes)/step)+1)
	for start := 0; start < len(runes); {
		end := start + options.MaxChars
		if end > len(runes) {
			end = len(runes)
		}
		part := strings.TrimSpace(string(runes[start:end]))
		if part != "" {
			parts = append(parts, textBlock{title: block.title, content: part})
		}
		if end == len(runes) {
			break
		}
		start += step
	}
	return parts
}

func applyOverlap(chunks []textBlock, overlapChars int, maxChars int) {
	for i := 1; i < len(chunks); i++ {
		if chunks[i].title != "" && chunks[i-1].title != "" && chunks[i].title != chunks[i-1].title {
			continue
		}
		previous := tailRunes(chunks[i-1].content, overlapChars)
		if previous == "" {
			continue
		}
		combined := CleanExtractedText(previous + "\n" + chunks[i].content)
		if runeLen(combined) > maxChars {
			combined = headRunes(combined, maxChars)
		}
		chunks[i].content = combined
	}
}

func normalizeChunkOptions(options ChunkOptions) ChunkOptions {
	if options.MinChars <= 0 {
		options.MinChars = defaultMinChunkChars
	}
	if options.MaxChars <= 0 {
		options.MaxChars = defaultMaxChunkChars
	}
	if options.MinChars > options.MaxChars {
		options.MinChars = options.MaxChars
	}
	if options.OverlapChars < 0 {
		options.OverlapChars = 0
	}
	if options.OverlapChars == 0 {
		options.OverlapChars = defaultOverlapChars
	}
	if options.OverlapChars >= options.MaxChars {
		options.OverlapChars = options.MaxChars / 5
	}
	return options
}

func fallbackChunkTitle(assetTitle string, index int) string {
	assetTitle = strings.TrimSpace(assetTitle)
	if assetTitle == "" {
		assetTitle = "知识资产"
	}
	return fmt.Sprintf("%s 片段 %d", assetTitle, index+1)
}

func labeledText(label string, value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	return label + ": " + value
}

func searchableMetadata(metadata map[string]any) string {
	if len(metadata) == 0 {
		return ""
	}
	keys := make([]string, 0, len(metadata))
	for key := range metadata {
		if isSearchableMetadataKey(key) {
			keys = append(keys, key)
		}
	}
	sort.Strings(keys)

	parts := make([]string, 0, len(keys))
	for _, key := range keys {
		value := metadata[key]
		switch typed := value.(type) {
		case string:
			if strings.TrimSpace(typed) != "" {
				parts = append(parts, key+": "+typed)
			}
		case []string:
			values := cleanStringList(typed)
			if len(values) > 0 {
				parts = append(parts, key+": "+strings.Join(values, " "))
			}
		case fmt.Stringer:
			parts = append(parts, key+": "+typed.String())
		case int, int32, int64, float32, float64, bool:
			parts = append(parts, fmt.Sprintf("%s: %v", key, typed))
		}
	}
	return strings.Join(parts, "\n")
}

func isSearchableMetadataKey(key string) bool {
	key = strings.ToLower(strings.TrimSpace(key))
	if key == "" {
		return false
	}
	switch key {
	case "source", "sourceurl", "source_url", "url", "author", "industry", "platform", "language", "contenttype", "content_type", "filekind", "filename", "mimetype", "extension":
		return true
	default:
		return strings.Contains(key, "title") || strings.Contains(key, "keyword") || strings.Contains(key, "topic")
	}
}

func mergeMetadata(base map[string]any, extra map[string]any) map[string]any {
	merged := make(map[string]any, len(base)+len(extra))
	for key, value := range base {
		merged[key] = value
	}
	for key, value := range extra {
		merged[key] = value
	}
	return merged
}

func cleanStringList(values []string) []string {
	cleaned := make([]string, 0, len(values))
	seen := map[string]bool{}
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" || seen[value] {
			continue
		}
		seen[value] = true
		cleaned = append(cleaned, value)
	}
	return cleaned
}

func joinChunkContent(left string, right string) string {
	left = strings.TrimSpace(left)
	right = strings.TrimSpace(right)
	switch {
	case left == "":
		return right
	case right == "":
		return left
	default:
		return left + "\n\n" + right
	}
}

func runeLen(value string) int {
	return utf8.RuneCountInString(value)
}

func tailRunes(value string, count int) string {
	runes := []rune(value)
	if len(runes) <= count {
		return strings.TrimSpace(value)
	}
	return strings.TrimSpace(string(runes[len(runes)-count:]))
}

func headRunes(value string, count int) string {
	runes := []rune(value)
	if len(runes) <= count {
		return strings.TrimSpace(value)
	}
	return strings.TrimSpace(string(runes[:count]))
}

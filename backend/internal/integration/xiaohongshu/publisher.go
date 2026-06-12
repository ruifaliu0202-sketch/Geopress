package xiaohongshu

import (
	"context"
	"errors"
	"fmt"
	"hash/fnv"
	"strings"
	"time"

	"geopress/backend/internal/integration/publisher"
)

const PlatformType = "xiaohongshu"

type MockHumanPublisher struct{}

func NewMockHumanPublisher() MockHumanPublisher {
	return MockHumanPublisher{}
}

func (MockHumanPublisher) PlatformType() string {
	return PlatformType
}

func (MockHumanPublisher) Prepare(ctx context.Context, req publisher.PrepareRequest) (publisher.PreparedPost, error) {
	select {
	case <-ctx.Done():
		return publisher.PreparedPost{}, ctx.Err()
	default:
	}

	if req.Platform.Type != PlatformType {
		return publisher.PreparedPost{}, errors.New("media account is not a xiaohongshu account")
	}

	title := xiaohongshuTitle(req.Content.Title)
	body := xiaohongshuBody(req.Content.Summary, req.Content.Body)
	if title == "" {
		return publisher.PreparedPost{}, errors.New("content title is required")
	}
	if body == "" {
		body = title
	}

	hashtags := xiaohongshuHashtags(req.Content.Keywords)
	topicText := strings.Join(hashtags, " ")
	fullBody := body
	if topicText != "" {
		fullBody = strings.TrimSpace(fullBody + "\n\n" + topicText)
	}

	preparedAt := req.RequestedAt
	if preparedAt.IsZero() {
		preparedAt = time.Now().UTC()
	}

	warnings := []string{
		"当前通道为 Mock 真人接口发布：不会调用小红书真实私有接口，也不会伪造 Cookie、签名或风控参数。",
	}
	if len(hashtags) == 0 {
		warnings = append(warnings, "当前内容没有关键词，发布前建议补充 2-5 个小红书话题。")
	}

	copyBlocks := []publisher.CopyBlock{
		{Label: "标题", Value: title},
		{Label: "正文", Value: fullBody},
	}
	if topicText != "" {
		copyBlocks = append(copyBlocks, publisher.CopyBlock{Label: "话题", Value: topicText})
	}

	return publisher.PreparedPost{
		Mode:         publisher.ModeMockHumanAPI,
		PlatformType: PlatformType,
		PlatformName: defaultString(req.Platform.Name, "小红书"),
		Title:        title,
		Body:         fullBody,
		Hashtags:     hashtags,
		CopyBlocks:   copyBlocks,
		Checklist: []string{
			"确认标题、正文、话题和素材满足小红书内容规范。",
			"确认当前账号已授权给真人发布服务或运营人员。",
			"调用 Mock 发布接口后保存返回的笔记 ID 和链接。",
		},
		Warnings:       warnings,
		CharacterCount: len([]rune(fullBody)),
		PreparedAt:     preparedAt,
	}, nil
}

func (MockHumanPublisher) Publish(ctx context.Context, req publisher.PublishRequest) (publisher.PublishResult, error) {
	select {
	case <-ctx.Done():
		return publisher.PublishResult{}, ctx.Err()
	default:
	}

	if strings.TrimSpace(req.PreparedPost.Title) == "" {
		return publisher.PublishResult{}, errors.New("prepared title is required")
	}
	if strings.TrimSpace(req.PreparedPost.Body) == "" {
		return publisher.PublishResult{}, errors.New("prepared body is required")
	}

	now := time.Now().UTC()
	noteID := mockNoteID(req.PreparedPost.Title, req.PreparedPost.Body, now)
	return publisher.PublishResult{
		Status:      "published",
		Message:     "Mock 真人接口已发布小红书笔记。",
		ExternalID:  noteID,
		ExternalURL: fmt.Sprintf("https://www.xiaohongshu.com/explore/%s", noteID),
		RawResponse: map[string]any{
			"provider":       "mock_human_api",
			"platform":       PlatformType,
			"noteId":         noteID,
			"publishedAt":    now.Format(time.RFC3339),
			"title":          req.PreparedPost.Title,
			"assetPathCount": len(req.AssetPaths),
		},
	}, nil
}

func xiaohongshuTitle(value string) string {
	title := strings.Join(strings.Fields(strings.TrimSpace(value)), " ")
	if len([]rune(title)) <= 60 {
		return title
	}
	runes := []rune(title)
	return string(runes[:60])
}

func xiaohongshuBody(summary string, body string) string {
	parts := []string{}
	if value := compactParagraphs(summary); value != "" {
		parts = append(parts, value)
	}
	if value := compactParagraphs(body); value != "" {
		parts = append(parts, value)
	}
	return strings.TrimSpace(strings.Join(parts, "\n\n"))
}

func xiaohongshuHashtags(values []string) []string {
	seen := map[string]bool{}
	hashtags := []string{}
	for _, value := range values {
		tag := normalizeHashtag(value)
		if tag == "" || seen[tag] {
			continue
		}
		seen[tag] = true
		hashtags = append(hashtags, tag)
		if len(hashtags) == 5 {
			break
		}
	}
	return hashtags
}

func normalizeHashtag(value string) string {
	value = strings.TrimSpace(strings.TrimPrefix(value, "#"))
	value = strings.Trim(value, " \t\r\n,，.。!！?？;；:：")
	value = strings.Join(strings.Fields(value), "")
	if value == "" {
		return ""
	}
	return "#" + value
}

func compactParagraphs(value string) string {
	value = strings.ReplaceAll(value, "\r\n", "\n")
	value = strings.ReplaceAll(value, "\r", "\n")
	lines := strings.Split(value, "\n")
	paragraphs := make([]string, 0, len(lines))
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		paragraphs = append(paragraphs, line)
	}
	return strings.Join(paragraphs, "\n")
}

func mockNoteID(title string, body string, now time.Time) string {
	hash := fnv.New64a()
	_, _ = hash.Write([]byte(title))
	_, _ = hash.Write([]byte(body))
	_, _ = hash.Write([]byte(now.Format(time.RFC3339Nano)))
	return fmt.Sprintf("mock-%x", hash.Sum64())
}

func defaultString(value string, fallback string) string {
	if value == "" {
		return fallback
	}
	return value
}

package manual

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"geopress/backend/internal/integration/publisher"
)

const ModeManualPublish = "manual_publish"

type PlatformConfig struct {
	Type            string
	DefaultName     string
	TitleMaxRunes   int
	HashtagMaxCount int
}

type Publisher struct {
	config PlatformConfig
}

func NewPublisher(config PlatformConfig) Publisher {
	if config.TitleMaxRunes <= 0 {
		config.TitleMaxRunes = 80
	}
	if config.HashtagMaxCount <= 0 {
		config.HashtagMaxCount = 5
	}
	return Publisher{config: config}
}

func (p Publisher) PlatformType() string {
	return p.config.Type
}

func (p Publisher) Prepare(ctx context.Context, req publisher.PrepareRequest) (publisher.PreparedPost, error) {
	select {
	case <-ctx.Done():
		return publisher.PreparedPost{}, ctx.Err()
	default:
	}

	if req.Platform.Type != p.config.Type {
		return publisher.PreparedPost{}, fmt.Errorf("media account is not a %s account", p.config.Type)
	}
	title := truncateRunes(compactInline(req.Content.Title), p.config.TitleMaxRunes)
	body := compactParagraphs(strings.TrimSpace(strings.Join([]string{req.Content.Summary, req.Content.Body}, "\n\n")))
	if title == "" {
		return publisher.PreparedPost{}, errors.New("content title is required")
	}
	if body == "" {
		body = title
	}

	hashtags := hashtags(req.Content.Keywords, p.config.HashtagMaxCount)
	topicText := strings.Join(hashtags, " ")
	copyBlocks := []publisher.CopyBlock{
		{Label: "标题", Value: title},
		{Label: "正文", Value: body},
	}
	if topicText != "" {
		copyBlocks = append(copyBlocks, publisher.CopyBlock{Label: "话题/标签", Value: topicText})
	}

	preparedAt := req.RequestedAt
	if preparedAt.IsZero() {
		preparedAt = time.Now().UTC()
	}
	platformName := strings.TrimSpace(req.Platform.Name)
	if platformName == "" {
		platformName = p.config.DefaultName
	}

	return publisher.PreparedPost{
		Mode:            ModeManualPublish,
		PlatformType:    p.config.Type,
		PlatformName:    platformName,
		PublishFormatID: defaultString(req.PublishFormatID, "article"),
		PublishMode:     "manual",
		Title:           title,
		Body:            body,
		Hashtags:        hashtags,
		CopyBlocks:      copyBlocks,
		Checklist: []string{
			fmt.Sprintf("复制标题、正文和标签到%s后台。", platformName),
			"人工确认封面、配图、分类和平台合规提示。",
			"发布后复制外部链接并在任务中确认结果。",
		},
		Warnings: []string{
			fmt.Sprintf("%s当前只声明人工发布能力，不会自动登录、保存 Cookie 或调用平台私有接口。", platformName),
		},
		CharacterCount: len([]rune(body)),
		PreparedAt:     preparedAt,
	}, nil
}

func (p Publisher) Publish(ctx context.Context, req publisher.PublishRequest) (publisher.PublishResult, error) {
	select {
	case <-ctx.Done():
		return publisher.PublishResult{}, ctx.Err()
	default:
	}
	return publisher.PublishResult{}, fmt.Errorf("%s only supports manual publish confirmation", defaultString(req.PreparedPost.PlatformName, p.config.DefaultName))
}

func compactInline(value string) string {
	return strings.Join(strings.Fields(strings.TrimSpace(value)), " ")
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
	return strings.Join(paragraphs, "\n\n")
}

func hashtags(values []string, maxCount int) []string {
	if maxCount <= 0 {
		maxCount = 5
	}
	seen := map[string]bool{}
	result := make([]string, 0, maxCount)
	for _, value := range values {
		tag := normalizeHashtag(value)
		if tag == "" || seen[tag] {
			continue
		}
		seen[tag] = true
		result = append(result, tag)
		if len(result) == maxCount {
			break
		}
	}
	return result
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

func truncateRunes(value string, maxRunes int) string {
	if maxRunes <= 0 {
		return value
	}
	runes := []rune(value)
	if len(runes) <= maxRunes {
		return value
	}
	return string(runes[:maxRunes])
}

func defaultString(value string, fallback string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return fallback
	}
	return value
}

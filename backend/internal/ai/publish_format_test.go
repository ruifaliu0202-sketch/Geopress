package ai

import (
	"strings"
	"testing"
)

func TestSelectPublishFormatXiaohongshuLongArticle(t *testing.T) {
	format := SelectPublishFormat(FormatXiaohongshuLongArticle)
	if format.ID != FormatXiaohongshuLongArticle {
		t.Fatalf("id = %q, want %q", format.ID, FormatXiaohongshuLongArticle)
	}
	if format.PlatformType != "xiaohongshu" {
		t.Fatalf("platform type = %q, want xiaohongshu", format.PlatformType)
	}
	if format.TitleMaxRunes != 64 {
		t.Fatalf("title max = %d, want 64", format.TitleMaxRunes)
	}
	if format.AutomationChannel != "playwright_xiaohongshu_long_article" {
		t.Fatalf("automation channel = %q", format.AutomationChannel)
	}
}

func TestPromptIncludesPublishFormatContract(t *testing.T) {
	format := SelectPublishFormat(FormatXiaohongshuLongArticle)
	prompt := BuildPrompt(GenerateRequest{
		ContentType:   FormatXiaohongshuLongArticle,
		Keywords:      []string{"内容飞轮"},
		Skill:         SelectWritingSkill(FormatXiaohongshuLongArticle),
		PublishFormat: format,
	})
	if !strings.Contains(prompt.User, "目标媒体平台发布格式 JSON") {
		t.Fatal("prompt does not include publish format section")
	}
	if !strings.Contains(prompt.User, FormatXiaohongshuLongArticle) {
		t.Fatalf("prompt does not include format id %q", FormatXiaohongshuLongArticle)
	}
	if !strings.Contains(prompt.User, "不要靠猜测平台格式") {
		t.Fatal("prompt does not include no-guessing rule")
	}
}

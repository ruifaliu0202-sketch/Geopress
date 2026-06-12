package xiaohongshu

import (
	"context"
	"testing"

	"geopress/backend/internal/integration/publisher"
	"geopress/backend/internal/model"
)

func TestBrowserLongArticlePublisherPrepare(t *testing.T) {
	prepared, err := NewBrowserLongArticlePublisher().Prepare(context.Background(), publisher.PrepareRequest{
		Platform:        model.MediaPlatform{Name: "小红书", Type: PlatformType},
		PublishFormatID: LongArticleFormatID,
		Content: model.Content{
			Title:    "独立顾问如何搭建内容飞轮",
			Summary:  "用稳定输出和案例沉淀提升获客效率。",
			Body:     "先确定目标读者。\n\n再把案例拆成可复用素材。",
			Keywords: []string{"独立顾问", "内容飞轮"},
		},
	})
	if err != nil {
		t.Fatalf("Prepare returned error: %v", err)
	}
	if prepared.Mode != publisher.ModeBrowserAutomation {
		t.Fatalf("mode = %q, want %q", prepared.Mode, publisher.ModeBrowserAutomation)
	}
	if prepared.PublishFormatID != LongArticleFormatID {
		t.Fatalf("format = %q, want %q", prepared.PublishFormatID, LongArticleFormatID)
	}
	if prepared.PublishMode != "long_article" {
		t.Fatalf("publish mode = %q, want long_article", prepared.PublishMode)
	}
	if prepared.Title == "" || prepared.Body == "" {
		t.Fatalf("expected title/body, got %#v", prepared)
	}
	if len(prepared.Checklist) == 0 {
		t.Fatal("expected checklist")
	}
}

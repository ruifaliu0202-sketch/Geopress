package browserplatform

import (
	"context"
	"testing"

	"geopress/backend/internal/integration/publisher"
	"geopress/backend/internal/model"
)

func TestPublisherPrepare(t *testing.T) {
	prepared, err := NewPublisher(Config{
		PlatformType:    "toutiao",
		PlatformName:    "头条号",
		PublishFormatID: "article",
		PublishMode:     "article",
	}).Prepare(context.Background(), publisher.PrepareRequest{
		Platform:        model.MediaPlatform{Name: "头条号", Type: "toutiao"},
		PublishFormatID: "article",
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
	if prepared.PlatformType != "toutiao" || prepared.PublishFormatID != "article" || prepared.PublishMode != "article" {
		t.Fatalf("unexpected prepared post contract: %#v", prepared)
	}
	if prepared.Title == "" || prepared.Body == "" {
		t.Fatalf("expected title/body, got %#v", prepared)
	}
	if len(prepared.Checklist) == 0 || len(prepared.CopyBlocks) < 2 {
		t.Fatalf("expected checklist and copy blocks: %#v", prepared)
	}
}

package xiaohongshu

import (
	"context"
	"strings"
	"testing"

	"geopress/backend/internal/integration/publisher"
	"geopress/backend/internal/model"
)

func TestMockHumanPublisherPrepare(t *testing.T) {
	prepared, err := NewMockHumanPublisher().Prepare(context.Background(), publisher.PrepareRequest{
		Platform: model.MediaPlatform{Name: "小红书", Type: PlatformType},
		Content: model.Content{
			Title:    "独立顾问如何搭建内容飞轮",
			Summary:  "用稳定输出和案例沉淀提升获客效率。",
			Body:     "先确定目标读者。\n\n再把案例拆成可复用素材。",
			Keywords: []string{"独立顾问", "内容飞轮", "#获客"},
		},
	})
	if err != nil {
		t.Fatalf("Prepare returned error: %v", err)
	}

	if prepared.Mode != publisher.ModeMockHumanAPI {
		t.Fatalf("mode = %q, want %q", prepared.Mode, publisher.ModeMockHumanAPI)
	}
	if prepared.PlatformType != PlatformType {
		t.Fatalf("platform type = %q, want %q", prepared.PlatformType, PlatformType)
	}
	if !strings.Contains(prepared.Body, "#独立顾问") || !strings.Contains(prepared.Body, "#获客") {
		t.Fatalf("body does not include expected hashtags: %q", prepared.Body)
	}
	if len(prepared.CopyBlocks) < 2 {
		t.Fatalf("copy blocks = %d, want at least 2", len(prepared.CopyBlocks))
	}
	if len(prepared.Checklist) == 0 {
		t.Fatal("checklist should not be empty")
	}
}

func TestMockHumanPublisherRejectsWrongPlatform(t *testing.T) {
	_, err := NewMockHumanPublisher().Prepare(context.Background(), publisher.PrepareRequest{
		Platform: model.MediaPlatform{Name: "WordPress", Type: "wordpress"},
		Content:  model.Content{Title: "title", Body: "body"},
	})
	if err == nil {
		t.Fatal("expected error for wrong platform")
	}
}

func TestMockHumanPublisherPublish(t *testing.T) {
	prepared, err := NewMockHumanPublisher().Prepare(context.Background(), publisher.PrepareRequest{
		Platform: model.MediaPlatform{Name: "小红书", Type: PlatformType},
		Content: model.Content{
			Title:    "独立顾问如何搭建内容飞轮",
			Summary:  "用稳定输出和案例沉淀提升获客效率。",
			Body:     "先确定目标读者。",
			Keywords: []string{"独立顾问"},
		},
	})
	if err != nil {
		t.Fatalf("Prepare returned error: %v", err)
	}

	result, err := NewMockHumanPublisher().Publish(context.Background(), publisher.PublishRequest{
		PreparedPost: prepared,
		AssetPaths:   []string{"/tmp/cover.png"},
	})
	if err != nil {
		t.Fatalf("Publish returned error: %v", err)
	}
	if result.Status != "published" {
		t.Fatalf("status = %q, want published", result.Status)
	}
	if result.ExternalID == "" || result.ExternalURL == "" {
		t.Fatalf("expected external id and url, got %#v", result)
	}
	if result.RawResponse["provider"] != "mock_human_api" {
		t.Fatalf("raw response provider = %#v", result.RawResponse["provider"])
	}
}

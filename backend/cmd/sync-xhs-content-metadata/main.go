package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"geopress/backend/internal/config"
	"geopress/backend/internal/database"
	"geopress/backend/internal/integration/xiaohongshu"
	"geopress/backend/internal/model"
)

func main() {
	workspaceID := flag.String("workspace-id", "wks_personal", "workspace id")
	accountID := flag.String("account-id", "acc_xhs_personal", "media account id")
	publishJobID := flag.String("publish-job-id", "", "optional publish job id to reconcile")
	contentID := flag.String("content-id", "", "optional content id")
	externalContentID := flag.String("external-content-id", "", "xiaohongshu note id; read from publish job attribution metadata when omitted")
	externalURL := flag.String("external-url", "", "optional external note url")
	title := flag.String("title", "", "optional note title for search fallback")
	profileDir := flag.String("profile-dir", "", "browser profile dir; defaults to runtime/browser-profiles/{workspace}/{account}")
	outputFile := flag.String("output", "", "optional collector output file")
	debugDir := flag.String("debug-dir", "", "optional collector debug dir")
	timeout := flag.Duration("timeout", 90*time.Second, "collector timeout")
	flag.Parse()

	if strings.TrimSpace(*workspaceID) == "" || strings.TrimSpace(*accountID) == "" {
		log.Fatal("workspace-id and account-id are required")
	}

	cfg := config.Load()
	db, err := database.Open(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("open database: %v", err)
	}
	defer func() {
		_ = db.Close()
	}()

	ctx, cancel := context.WithTimeout(context.Background(), *timeout+15*time.Second)
	defer cancel()

	item, found, err := db.GetMediaAccountMatrixItem(ctx, *workspaceID, *accountID)
	if err != nil {
		log.Fatalf("load media account: %v", err)
	}
	if !found {
		log.Fatalf("media account not found: workspace=%s account=%s", *workspaceID, *accountID)
	}
	if item.Platform.Type != xiaohongshu.PlatformType {
		log.Fatalf("media account platform is %s, not xiaohongshu", item.Platform.Type)
	}

	var publishJob model.PublishJob
	if strings.TrimSpace(*publishJobID) != "" {
		var ok bool
		publishJob, ok, err = db.GetPublishJobForMetrics(ctx, *workspaceID, *publishJobID)
		if err != nil {
			log.Fatalf("load publish job: %v", err)
		}
		if !ok {
			log.Fatalf("publish job not found: workspace=%s job=%s", *workspaceID, *publishJobID)
		}
		if publishJob.MediaAccountID != item.Account.ID {
			log.Fatalf("publish job account is %s, not %s", publishJob.MediaAccountID, item.Account.ID)
		}
	}

	content := model.Content{}
	resolvedContentID := firstNonEmpty(*contentID, publishJob.ContentID)
	if resolvedContentID != "" {
		var ok bool
		content, ok, err = db.GetContentForMetrics(ctx, *workspaceID, resolvedContentID)
		if err != nil {
			log.Fatalf("load content: %v", err)
		}
		if !ok {
			log.Fatalf("content not found: workspace=%s content=%s", *workspaceID, resolvedContentID)
		}
	}

	resolvedExternalID := firstNonEmpty(*externalContentID, stringFromAny(publishJob.AttributionMetadata["externalId"]), stringFromAny(publishJob.AttributionMetadata["externalContentId"]))
	resolvedExternalURL := firstNonEmpty(*externalURL, publishJob.ExternalURL, stringFromAny(publishJob.AttributionMetadata["externalUrl"]))
	resolvedTitle := firstNonEmpty(*title, content.Title)
	if resolvedExternalID == "" {
		log.Fatal("external-content-id is required when publish job attribution metadata has no externalId")
	}
	if resolvedContentID == "" {
		log.Fatal("content-id is required when publish-job-id is omitted or has no content_id")
	}

	profile := strings.TrimSpace(*profileDir)
	if profile == "" {
		profile = xiaohongshu.RuntimeBrowserProfilePath(*workspaceID, *accountID)
	}
	out := strings.TrimSpace(*outputFile)
	dbg := strings.TrimSpace(*debugDir)
	if out == "" || dbg == "" {
		defaultDebugDir := filepath.Join(defaultRuntimeRoot(profile), "runtime", "xhs-content-metadata-debug", *workspaceID, *accountID, resolvedExternalID)
		if out == "" {
			out = filepath.Join(defaultDebugDir, "content-metadata-result.json")
		}
		if dbg == "" {
			dbg = defaultDebugDir
		}
	}

	now := time.Now().UTC()
	job := model.MediaAccountSyncJob{
		ID:             fmt.Sprintf("masj_xhs_content_%d", now.UnixNano()),
		WorkspaceID:    item.Account.WorkspaceID,
		MediaAccountID: item.Account.ID,
		PlatformID:     item.Account.PlatformID,
		SyncType:       "content_metrics",
		Status:         "running",
		RequestedAt:    now,
		StartedAt:      &now,
		// 单篇指标按天重试，避免同一天重复调度生成多条同步任务。
		IdempotencyKey: fmt.Sprintf("xhs-content:%s:%s", resolvedExternalID, now.Format("2006-01-02")),
		RequestPayload: map[string]any{
			"profileDir":        profile,
			"externalContentId": resolvedExternalID,
			"externalUrl":       resolvedExternalURL,
			"title":             resolvedTitle,
			"contentId":         resolvedContentID,
			"publishJobId":      publishJob.ID,
		},
		ResultSummary: map[string]any{},
		CreatedAt:     now,
		UpdatedAt:     now,
	}

	collector := xiaohongshu.NewPlaywrightContentMetadataCollector()
	result, collectErr := collector.CollectContent(ctx, xiaohongshu.ContentMetadataCollectRequest{
		WorkspaceID:       *workspaceID,
		AccountID:         *accountID,
		ProfileDir:        profile,
		ExternalContentID: resolvedExternalID,
		ExternalURL:       resolvedExternalURL,
		Title:             resolvedTitle,
		ContentID:         resolvedContentID,
		PublishJobID:      publishJob.ID,
		OutputFile:        out,
		DebugDir:          dbg,
		ScreenshotDir:     dbg,
		ActionTimeout:     *timeout,
		SettleDelay:       2500 * time.Millisecond,
	})

	finishedAt := time.Now().UTC()
	job.FinishedAt = &finishedAt
	job.UpdatedAt = finishedAt
	if collectErr != nil && result.Status != "pending_reconcile" {
		job.Status = "failed"
		job.ErrorMessage = collectErr.Error()
		job.ResultSummary = syncSummaryFromResult(result, "")
		if err := db.SaveContentMetricsSyncResult(ctx, model.ContentMetric{}, job); err != nil {
			log.Fatalf("collector failed (%v), and sync job save failed: %v", collectErr, err)
		}
		log.Fatalf("collect xiaohongshu content metadata: %v", collectErr)
	}

	if result.Status == "pending_reconcile" || result.Metadata == nil {
		job.Status = "pending_reconcile"
		job.ErrorMessage = defaultString(errorFromDiagnostics(result.Diagnostics), "xiaohongshu note is not visible in creator note manager yet")
		job.ResultSummary = syncSummaryFromResult(result, "")
		if err := db.SaveContentMetricsSyncResult(ctx, model.ContentMetric{}, job); err != nil {
			log.Fatalf("save pending reconcile result: %v", err)
		}
		writeJSON(map[string]any{
			"ok":                false,
			"status":            job.Status,
			"workspaceId":       *workspaceID,
			"accountId":         *accountID,
			"externalContentId": resolvedExternalID,
			"syncJobId":         job.ID,
			"message":           job.ErrorMessage,
		})
		return
	}

	metric := buildContentMetric(item.Account, item.Platform, content, publishJob, result, now)
	job.Status = "completed"
	job.ErrorMessage = ""
	job.ResultSummary = syncSummaryFromResult(result, metric.ID)
	if err := db.SaveContentMetricsSyncResult(ctx, metric, job); err != nil {
		log.Fatalf("save content metrics sync result: %v", err)
	}

	writeJSON(map[string]any{
		"ok":                true,
		"status":            result.Status,
		"workspaceId":       *workspaceID,
		"accountId":         *accountID,
		"contentId":         metric.ContentID,
		"publishJobId":      metric.PublishJobID,
		"externalContentId": metric.ExternalContentID,
		"metricId":          metric.ID,
		"capturedAt":        metric.CapturedAt,
		"metrics": map[string]any{
			"viewCount":     metric.ViewCount,
			"likeCount":     metric.LikeCount,
			"commentCount":  metric.CommentCount,
			"shareCount":    metric.ShareCount,
			"favoriteCount": metric.FavoriteCount,
		},
	})
}

func buildContentMetric(account model.MediaAccount, platform model.MediaPlatform, content model.Content, publishJob model.PublishJob, result xiaohongshu.ContentMetadataCollectResult, requestedAt time.Time) model.ContentMetric {
	capturedAt := result.CapturedAt
	if result.Metadata != nil && !result.Metadata.CapturedAt.IsZero() {
		capturedAt = result.Metadata.CapturedAt
	}
	if capturedAt.IsZero() {
		capturedAt = time.Now().UTC()
	}
	metadata := result.Metadata
	metricDate := capturedAt.Format("2006-01-02")
	externalID := firstNonEmpty(metadata.ExternalContentID, result.ExternalContentID)
	externalURL := firstNonEmpty(metadata.ExternalURL, result.ExternalURL, publishJob.ExternalURL)
	values := metadata.Metrics

	return model.ContentMetric{
		ID:                fmt.Sprintf("cm_xhs_%s_%s", externalID, metricDate),
		WorkspaceID:       account.WorkspaceID,
		ContentID:         content.ID,
		PublishJobID:      publishJob.ID,
		MediaAccountID:    account.ID,
		PlatformID:        platform.ID,
		ExternalContentID: externalID,
		ExternalURL:       externalURL,
		MetricDate:        metricDate,
		CapturedAt:        capturedAt,
		ImpressionCount:   values.ImpressionCount,
		ViewCount:         values.ViewCount,
		LikeCount:         values.LikeCount,
		CommentCount:      values.CommentCount,
		ShareCount:        values.ShareCount,
		FavoriteCount:     values.FavoriteCount,
		ClickCount:        values.ClickCount,
		EngagementRate:    values.EngagementRate,
		AttributionMetadata: map[string]any{
			"platform":          xiaohongshu.PlatformType,
			"source":            result.DataSource,
			"collectorStatus":   result.Status,
			"confidence":        metadata.Confidence,
			"matchStrategy":     metadata.MatchStrategy,
			"matchScore":        metadata.MatchScore,
			"matchKeyword":      metadata.MatchKeyword,
			"sourceUrl":         metadata.SourceURL,
			"title":             metadata.Title,
			"publishedAt":       metadata.PublishedAt,
			"requestedAt":       requestedAt,
			"publishJobStatus":  publishJob.Status,
			"attributionSource": "xiaohongshu_content_metadata_sync",
		},
		RawMetrics: map[string]any{
			"collector": map[string]any{
				"status":            result.Status,
				"dataSource":        result.DataSource,
				"pageUrl":           result.PageURL,
				"externalContentId": result.ExternalContentID,
				"capturedAt":        result.CapturedAt,
			},
			"rawMetrics":  metadata.RawMetrics,
			"status":      metadata.Status,
			"statusText":  metadata.StatusText,
			"diagnostics": result.Diagnostics,
		},
		CreatedAt: capturedAt,
	}
}

func syncSummaryFromResult(result xiaohongshu.ContentMetadataCollectResult, metricID string) map[string]any {
	summary := map[string]any{
		"status":            result.Status,
		"externalContentId": result.ExternalContentID,
		"externalUrl":       result.ExternalURL,
		"capturedAt":        result.CapturedAt,
		"dataSource":        result.DataSource,
		"pageUrl":           result.PageURL,
		"metricId":          metricID,
	}
	if result.Metadata != nil {
		summary["viewCount"] = result.Metadata.Metrics.ViewCount
		summary["likeCount"] = result.Metadata.Metrics.LikeCount
		summary["commentCount"] = result.Metadata.Metrics.CommentCount
		summary["shareCount"] = result.Metadata.Metrics.ShareCount
		summary["favoriteCount"] = result.Metadata.Metrics.FavoriteCount
		summary["confidence"] = result.Metadata.Confidence
		summary["matchStrategy"] = result.Metadata.MatchStrategy
	}
	if diagnosticError := errorFromDiagnostics(result.Diagnostics); diagnosticError != "" {
		summary["diagnosticError"] = diagnosticError
	}
	return summary
}

func errorFromDiagnostics(value map[string]any) string {
	if value == nil {
		return ""
	}
	return stringFromAny(value["error"])
}

func writeJSON(value map[string]any) {
	encoded, _ := json.MarshalIndent(value, "", "  ")
	_, _ = os.Stdout.Write(append(encoded, '\n'))
}

func stringFromAny(value any) string {
	switch item := value.(type) {
	case string:
		return strings.TrimSpace(item)
	default:
		return ""
	}
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			return value
		}
	}
	return ""
}

func defaultString(value string, fallback string) string {
	if strings.TrimSpace(value) != "" {
		return strings.TrimSpace(value)
	}
	return fallback
}

func defaultRuntimeRoot(profileDir string) string {
	if value := strings.TrimSpace(os.Getenv("GEOPRESS_RUNTIME_ROOT")); value != "" {
		return value
	}
	if value := strings.TrimSpace(os.Getenv("GEOPRESS_PROJECT_ROOT")); value != "" {
		return value
	}
	profileDir = filepath.Clean(profileDir)
	marker := filepath.Join("runtime", "browser-profiles")
	if index := strings.Index(profileDir, marker); index > 0 {
		return filepath.Clean(profileDir[:index])
	}
	if wd, err := os.Getwd(); err == nil {
		if filepath.Base(wd) == "backend" {
			return filepath.Dir(wd)
		}
		return wd
	}
	return "."
}

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
	"geopress/backend/internal/integration/browserplatform"
	"geopress/backend/internal/integration/toutiao"
	"geopress/backend/internal/model"
)

func main() {
	workspaceID := flag.String("workspace-id", "wks_personal", "workspace id")
	accountID := flag.String("account-id", "", "media account id")
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
	if item.Platform.Type != toutiao.PlatformType {
		log.Fatalf("media account platform is %s, not toutiao", item.Platform.Type)
	}

	profile := strings.TrimSpace(*profileDir)
	if profile == "" {
		profile = browserplatform.RuntimeBrowserProfilePath(*workspaceID, *accountID)
	}
	out := strings.TrimSpace(*outputFile)
	dbg := strings.TrimSpace(*debugDir)
	if out == "" || dbg == "" {
		defaultDebugDir := filepath.Join(defaultRuntimeRoot(profile), "runtime", "toutiao-metadata-debug", *workspaceID, *accountID)
		if out == "" {
			out = filepath.Join(defaultDebugDir, "metadata-result.json")
		}
		if dbg == "" {
			dbg = defaultDebugDir
		}
	}

	now := time.Now().UTC()
	job := model.MediaAccountSyncJob{
		ID:             fmt.Sprintf("masj_toutiao_meta_%d", now.UnixNano()),
		WorkspaceID:    item.Account.WorkspaceID,
		MediaAccountID: item.Account.ID,
		PlatformID:     item.Account.PlatformID,
		SyncType:       "full",
		Status:         "running",
		RequestedAt:    now,
		StartedAt:      &now,
		IdempotencyKey: fmt.Sprintf("toutiao-metadata:%s", now.Format("20060102150405")),
		RequestPayload: map[string]any{
			"profileDir":  profile,
			"metadataUrl": toutiao.DefaultAccountMetadataURL,
		},
		ResultSummary: map[string]any{},
		CreatedAt:     now,
		UpdatedAt:     now,
	}

	collector := toutiao.NewPlaywrightAccountMetadataCollector()
	result, collectErr := collector.Collect(ctx, toutiao.AccountMetadataCollectRequest{
		WorkspaceID:   *workspaceID,
		AccountID:     *accountID,
		ProfileDir:    profile,
		OutputFile:    out,
		DebugDir:      dbg,
		ActionTimeout: *timeout,
		SettleDelay:   3 * time.Second,
	})

	finishedAt := time.Now().UTC()
	job.FinishedAt = &finishedAt
	job.UpdatedAt = finishedAt
	if collectErr != nil {
		job.Status = "failed"
		job.ErrorMessage = collectErr.Error()
		job.ResultSummary = map[string]any{
			"status":      result.Status,
			"pageUrl":     result.PageURL,
			"capturedAt":  result.CapturedAt,
			"loginState":  result.LoginState,
			"diagnostics": result.Diagnostics,
		}
		if _, saveErr := db.CreateMediaAccountSyncJob(ctx, job); saveErr != nil {
			log.Fatalf("collector failed (%v), and sync job save failed: %v", collectErr, saveErr)
		}
		log.Fatalf("collect toutiao metadata: %v", collectErr)
	}

	account, snapshot := buildSyncResult(item.Account, result, profile, now)
	job.Status = "completed"
	job.ErrorMessage = ""
	job.ResultSummary = map[string]any{
		"displayName":        result.Metadata.DisplayName,
		"userId":             result.Metadata.UserID,
		"mediaId":            result.Metadata.MediaID,
		"followerCount":      result.Metadata.FollowerCount,
		"contentCount":       result.Metadata.ContentCount,
		"totalReadPlayCount": result.Metadata.TotalReadPlayCount,
		"snapshotId":         snapshot.ID,
		"capturedAt":         result.CapturedAt,
		"dataSource":         result.DataSource,
	}

	if err := db.SaveMediaAccountMetricsSyncResult(ctx, account, snapshot, job); err != nil {
		log.Fatalf("save metadata sync result: %v", err)
	}

	encoded, _ := json.MarshalIndent(map[string]any{
		"ok":          true,
		"accountId":   account.ID,
		"workspaceId": account.WorkspaceID,
		"snapshotId":  snapshot.ID,
		"capturedAt":  snapshot.CapturedAt,
		"metadata": map[string]any{
			"displayName":        result.Metadata.DisplayName,
			"userId":             result.Metadata.UserID,
			"mediaId":            result.Metadata.MediaID,
			"followerCount":      result.Metadata.FollowerCount,
			"contentCount":       result.Metadata.ContentCount,
			"totalReadPlayCount": result.Metadata.TotalReadPlayCount,
		},
	}, "", "  ")
	_, _ = os.Stdout.Write(append(encoded, '\n'))
}

func buildSyncResult(account model.MediaAccount, result toutiao.AccountMetadataCollectResult, profileDir string, requestedAt time.Time) (model.MediaAccount, model.MediaAccountMetricSnapshot) {
	capturedAt := result.CapturedAt
	if capturedAt.IsZero() {
		capturedAt = time.Now().UTC()
	}
	metadata := cloneAnyMap(account.MatrixMetadata)
	metadata["platformMetadata"] = map[string]any{
		"displayName":        result.Metadata.DisplayName,
		"userId":             result.Metadata.UserID,
		"mediaId":            result.Metadata.MediaID,
		"avatarUrl":          result.Metadata.AvatarURL,
		"profileUrl":         result.Metadata.ProfileURL,
		"isCreator":          result.Metadata.IsCreator,
		"authType":           result.Metadata.AuthType,
		"followerCount":      result.Metadata.FollowerCount,
		"contentCount":       result.Metadata.ContentCount,
		"totalReadPlayCount": result.Metadata.TotalReadPlayCount,
		"source":             result.DataSource,
		"pageUrl":            result.PageURL,
		"profileDir":         profileDir,
		"capturedAt":         capturedAt,
	}
	metadata["lastMetadataSync"] = map[string]any{
		"source":      result.DataSource,
		"status":      result.Status,
		"capturedAt":  capturedAt,
		"requestedAt": requestedAt,
	}

	account.Name = firstNonEmpty(result.Metadata.DisplayName, account.Name)
	account.ExternalID = firstNonEmpty(result.Metadata.UserID, result.Metadata.MediaID, account.ExternalID)
	account.Status = "connected"
	account.HealthStatus = "healthy"
	account.HealthNotes = ""
	account.SyncEnabled = true
	account.LastProfileSyncedAt = &capturedAt
	account.LastMetricsSyncedAt = &capturedAt
	account.LastCheckedAt = capturedAt
	account.MatrixMetadata = metadata
	account.LastSyncStatus = "completed"
	account.LastSyncMessage = "toutiao metadata sync completed"

	rawMetrics := map[string]any{
		"userInfo":           result.Metadata.UserInfo,
		"homeStatistic":      result.Metadata.HomeStatistic,
		"worksSummary":       result.Metadata.WorksSummary,
		"worksListSummary":   result.Metadata.WorksListSummary,
		"visibleMetrics":     result.Metadata.VisibleMetrics,
		"collectorStatus":    result.Status,
		"collectorSelectors": result.Selectors,
		"diagnostics":        result.Diagnostics,
	}
	profile := map[string]any{
		"displayName": result.Metadata.DisplayName,
		"userId":      result.Metadata.UserID,
		"mediaId":     result.Metadata.MediaID,
		"avatarUrl":   result.Metadata.AvatarURL,
		"profileUrl":  result.Metadata.ProfileURL,
		"pageUrl":     result.PageURL,
	}

	snapshot := model.MediaAccountMetricSnapshot{
		ID:                 fmt.Sprintf("mams_toutiao_%s_%d", account.ID, capturedAt.UnixNano()),
		WorkspaceID:        account.WorkspaceID,
		MediaAccountID:     account.ID,
		PlatformID:         account.PlatformID,
		Source:             "browser_context_request",
		CapturedAt:         capturedAt,
		FollowerCount:      result.Metadata.FollowerCount,
		FollowingCount:     0,
		ContentCount:       result.Metadata.ContentCount,
		TotalLikeCount:     0,
		TotalFavoriteCount: 0,
		TotalCommentCount:  0,
		TotalShareCount:    0,
		AudienceSignals:    map[string]any{},
		Profile:            profile,
		RawMetrics:         rawMetrics,
		FreshnessStatus:    "fresh",
		CreatedAt:          capturedAt,
	}
	return account, snapshot
}

func cloneAnyMap(value map[string]any) map[string]any {
	result := map[string]any{}
	for key, item := range value {
		result[key] = item
	}
	return result
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

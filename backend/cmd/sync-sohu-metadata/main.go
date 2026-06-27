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
	"geopress/backend/internal/integration/sohu"
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
	if item.Platform.Type != sohu.PlatformType {
		log.Fatalf("media account platform is %s, not sohu", item.Platform.Type)
	}

	profile := strings.TrimSpace(*profileDir)
	if profile == "" {
		profile = browserplatform.RuntimeBrowserProfilePath(*workspaceID, *accountID)
	}
	out := strings.TrimSpace(*outputFile)
	dbg := strings.TrimSpace(*debugDir)
	if out == "" || dbg == "" {
		defaultDebugDir := filepath.Join(defaultRuntimeRoot(profile), "runtime", "sohu-metadata-debug", *workspaceID, *accountID)
		if out == "" {
			out = filepath.Join(defaultDebugDir, "metadata-result.json")
		}
		if dbg == "" {
			dbg = defaultDebugDir
		}
	}

	now := time.Now().UTC()
	job := model.MediaAccountSyncJob{
		ID:             fmt.Sprintf("masj_sohu_meta_%d", now.UnixNano()),
		WorkspaceID:    item.Account.WorkspaceID,
		MediaAccountID: item.Account.ID,
		PlatformID:     item.Account.PlatformID,
		SyncType:       "full",
		Status:         "running",
		RequestedAt:    now,
		StartedAt:      &now,
		IdempotencyKey: fmt.Sprintf("sohu-metadata:%s", now.Format("20060102150405")),
		RequestPayload: map[string]any{
			"profileDir":  profile,
			"metadataUrl": sohu.DefaultAccountMetadataURL,
		},
		ResultSummary: map[string]any{},
		CreatedAt:     now,
		UpdatedAt:     now,
	}

	collector := sohu.NewPlaywrightAccountMetadataCollector()
	result, collectErr := collector.Collect(ctx, sohu.AccountMetadataCollectRequest{
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
		log.Fatalf("collect sohu metadata: %v", collectErr)
	}

	account, snapshot := buildSyncResult(item.Account, result, profile, now)
	job.Status = "completed"
	job.ErrorMessage = ""
	job.ResultSummary = map[string]any{
		"displayName": result.Metadata.DisplayName,
		"accountId":   result.Metadata.AccountID,
		"orgId":       result.Metadata.OrgID,
		"statusName":  result.Metadata.StatusName,
		"snapshotId":  snapshot.ID,
		"capturedAt":  result.CapturedAt,
		"dataSource":  result.DataSource,
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
			"displayName": account.Name,
			"accountId":   result.Metadata.AccountID,
			"orgId":       result.Metadata.OrgID,
			"statusName":  result.Metadata.StatusName,
		},
	}, "", "  ")
	_, _ = os.Stdout.Write(append(encoded, '\n'))
}

func buildSyncResult(account model.MediaAccount, result sohu.AccountMetadataCollectResult, profileDir string, requestedAt time.Time) (model.MediaAccount, model.MediaAccountMetricSnapshot) {
	capturedAt := result.CapturedAt
	if capturedAt.IsZero() {
		capturedAt = time.Now().UTC()
	}
	metadata := cloneAnyMap(account.MatrixMetadata)
	metadata["platformMetadata"] = map[string]any{
		"displayName":     result.Metadata.DisplayName,
		"accountId":       result.Metadata.AccountID,
		"orgId":           result.Metadata.OrgID,
		"userId":          result.Metadata.UserID,
		"userCode":        result.Metadata.UserCode,
		"avatarUrl":       result.Metadata.AvatarURL,
		"profileUrl":      result.Metadata.ProfileURL,
		"accountType":     result.Metadata.AccountType,
		"accountTypeName": result.Metadata.AccountTypeName,
		"statusCode":      result.Metadata.StatusCode,
		"statusName":      result.Metadata.StatusName,
		"registerStatus":  result.Metadata.RegisterStatus,
		"securityScore":   result.Metadata.SecurityScore,
		"riskLevel":       result.Metadata.RiskLevel,
		"newUser":         result.Metadata.NewUser,
		"admin":           result.Metadata.Admin,
		"hasOperator":     result.Metadata.HasOperator,
		"userIsOperator":  result.Metadata.UserIsOperator,
		"orgStatusName":   result.Metadata.OrgStatusName,
		"provinceName":    result.Metadata.ProvinceName,
		"cityName":        result.Metadata.CityName,
		"source":          result.DataSource,
		"pageUrl":         result.PageURL,
		"profileDir":      profileDir,
		"capturedAt":      capturedAt,
	}
	metadata["lastMetadataSync"] = map[string]any{
		"source":      result.DataSource,
		"status":      result.Status,
		"capturedAt":  capturedAt,
		"requestedAt": requestedAt,
	}

	account.Name = firstNonEmpty(result.Metadata.DisplayName, account.Name)
	account.ExternalID = firstNonEmpty(result.Metadata.AccountID, account.ExternalID)
	account.Status = "connected"
	account.HealthStatus = sohuHealthStatus(result.Metadata)
	account.HealthNotes = sohuHealthNotes(result.Metadata)
	account.SyncEnabled = true
	account.LastProfileSyncedAt = &capturedAt
	account.LastMetricsSyncedAt = &capturedAt
	account.LastCheckedAt = capturedAt
	account.MatrixMetadata = metadata
	account.LastSyncStatus = "completed"
	account.LastSyncMessage = "sohu metadata sync completed"

	rawMetrics := map[string]any{
		"accountInfo":        result.Metadata.AccountInfo,
		"listAccount":        result.Metadata.ListAccount,
		"registerInfo":       result.Metadata.RegisterInfo,
		"rightsSummary":      result.Metadata.RightsSummary,
		"commonAuth":         result.Metadata.CommonAuth,
		"userCheck":          result.Metadata.UserCheck,
		"visibleText":        result.Metadata.VisibleText,
		"collectorStatus":    result.Status,
		"collectorSelectors": result.Selectors,
		"diagnostics":        result.Diagnostics,
	}
	profile := map[string]any{
		"displayName": result.Metadata.DisplayName,
		"accountId":   result.Metadata.AccountID,
		"orgId":       result.Metadata.OrgID,
		"avatarUrl":   result.Metadata.AvatarURL,
		"profileUrl":  result.Metadata.ProfileURL,
		"statusName":  result.Metadata.StatusName,
		"pageUrl":     result.PageURL,
	}

	snapshot := model.MediaAccountMetricSnapshot{
		ID:                 fmt.Sprintf("mams_sohu_%s_%d", account.ID, capturedAt.UnixNano()),
		WorkspaceID:        account.WorkspaceID,
		MediaAccountID:     account.ID,
		PlatformID:         account.PlatformID,
		Source:             "browser_context_request",
		CapturedAt:         capturedAt,
		FollowerCount:      0,
		FollowingCount:     0,
		ContentCount:       0,
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

func sohuHealthStatus(metadata sohu.AccountMetadataSnapshot) string {
	status := strings.TrimSpace(metadata.StatusName)
	if strings.Contains(status, "新手") || strings.Contains(metadata.VisibleText, "实名认证可解锁") {
		return "warning"
	}
	return "healthy"
}

func sohuHealthNotes(metadata sohu.AccountMetadataSnapshot) string {
	status := strings.TrimSpace(metadata.StatusName)
	if strings.Contains(status, "新手") || strings.Contains(metadata.VisibleText, "实名认证可解锁") {
		return "搜狐号当前处于新手期或未完成实名认证，发布和数据能力可能受限。"
	}
	return ""
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

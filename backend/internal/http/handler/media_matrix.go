package handler

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"geopress/backend/internal/http/middleware"
	"geopress/backend/internal/model"

	"github.com/gin-gonic/gin"
)

type createMediaAccountSyncJobRequest struct {
	SyncType       string         `json:"syncType"`
	IdempotencyKey string         `json:"idempotencyKey"`
	RequestPayload map[string]any `json:"requestPayload"`
}

func (h *WorkspaceHandler) ListMediaAccountMatrix(c *gin.Context) {
	workspaceID, ok := h.authorizedWorkspaceID(c)
	if !ok {
		return
	}

	if h.db != nil && h.db.SQL() != nil {
		dbCtx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
		defer cancel()
		items, err := h.db.ListMediaAccountMatrix(dbCtx, workspaceID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "media account matrix lookup failed"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"items": items})
		return
	}

	h.mu.RLock()
	items := h.mediaAccountMatrixSnapshotLocked(workspaceID)
	h.mu.RUnlock()
	c.JSON(http.StatusOK, gin.H{"items": items})
}

func (h *WorkspaceHandler) GetMediaAccountMatrixItem(c *gin.Context) {
	workspaceID, ok := h.authorizedWorkspaceID(c)
	if !ok {
		return
	}
	accountID := strings.TrimSpace(c.Param("accountId"))
	if accountID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "account id is required"})
		return
	}

	if h.db != nil && h.db.SQL() != nil {
		dbCtx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
		defer cancel()
		item, found, err := h.db.GetMediaAccountMatrixItem(dbCtx, workspaceID, accountID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "media account matrix lookup failed"})
			return
		}
		if !found {
			c.JSON(http.StatusNotFound, gin.H{"error": "media account not found"})
			return
		}
		c.JSON(http.StatusOK, item)
		return
	}

	h.mu.RLock()
	items := h.mediaAccountMatrixSnapshotLocked(workspaceID)
	h.mu.RUnlock()
	for _, item := range items {
		if item.Account.ID == accountID {
			c.JSON(http.StatusOK, item)
			return
		}
	}
	c.JSON(http.StatusNotFound, gin.H{"error": "media account not found"})
}

func (h *WorkspaceHandler) ListMediaAccountMetricSnapshots(c *gin.Context) {
	workspaceID, ok := h.authorizedWorkspaceID(c)
	if !ok {
		return
	}
	accountID := strings.TrimSpace(c.Param("accountId"))
	if accountID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "account id is required"})
		return
	}
	limit := boundedQueryInt(c, "limit", 90, 1, 365)

	h.mu.RLock()
	account, accountOK := h.mediaAccountByID(workspaceID, accountID)
	h.mu.RUnlock()
	if !accountOK {
		c.JSON(http.StatusNotFound, gin.H{"error": "media account not found"})
		return
	}

	if h.db != nil && h.db.SQL() != nil {
		dbCtx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
		defer cancel()
		items, err := h.db.ListMediaAccountMetricSnapshots(dbCtx, workspaceID, account.ID, limit)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "media account metric lookup failed"})
			return
		}
		// 数据新鲜度由快照 capturedAt 决定；没有平台数据时仍返回 []，让前端展示空态而不是异常。
		c.JSON(http.StatusOK, gin.H{"items": items})
		return
	}

	c.JSON(http.StatusOK, gin.H{"items": []model.MediaAccountMetricSnapshot{}})
}

func (h *WorkspaceHandler) ListContentMetrics(c *gin.Context) {
	workspaceID, ok := h.authorizedWorkspaceID(c)
	if !ok {
		return
	}
	accountID := strings.TrimSpace(c.Query("mediaAccountId"))
	contentID := strings.TrimSpace(c.Query("contentId"))
	limit := boundedQueryInt(c, "limit", 100, 1, 500)

	h.mu.RLock()
	if accountID != "" && !hasMediaAccount(h.accounts, workspaceID, accountID) {
		h.mu.RUnlock()
		c.JSON(http.StatusNotFound, gin.H{"error": "media account not found"})
		return
	}
	if contentID != "" && !hasContent(h.contents, workspaceID, contentID) {
		h.mu.RUnlock()
		c.JSON(http.StatusNotFound, gin.H{"error": "content not found"})
		return
	}
	h.mu.RUnlock()

	if h.db != nil && h.db.SQL() != nil {
		dbCtx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
		defer cancel()
		items, err := h.db.ListContentMetrics(dbCtx, workspaceID, accountID, contentID, limit)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "content metric lookup failed"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"items": items})
		return
	}

	c.JSON(http.StatusOK, gin.H{"items": []model.ContentMetric{}})
}

func (h *WorkspaceHandler) CreateMediaAccountSyncJob(c *gin.Context) {
	workspaceID, ok := h.authorizedWorkspaceID(c)
	if !ok {
		return
	}
	accountID := strings.TrimSpace(c.Param("accountId"))
	if accountID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "account id is required"})
		return
	}

	var req createMediaAccountSyncJobRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	syncType := strings.TrimSpace(req.SyncType)
	if syncType == "" {
		syncType = "metrics"
	}
	if !validMediaAccountSyncType(syncType) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "syncType is invalid"})
		return
	}

	now := time.Now().UTC()
	h.mu.RLock()
	account, accountOK := h.mediaAccountByID(workspaceID, accountID)
	platform, platformOK := h.mediaPlatformByID(account.PlatformID)
	h.mu.RUnlock()
	if !accountOK {
		c.JSON(http.StatusNotFound, gin.H{"error": "media account not found"})
		return
	}
	if !platformOK || !platform.Enabled {
		c.JSON(http.StatusBadRequest, gin.H{"error": "media platform is not enabled"})
		return
	}

	idempotencyKey := strings.TrimSpace(req.IdempotencyKey)
	if idempotencyKey == "" {
		idempotencyKey = fmt.Sprintf("%s:%s:%s", syncType, account.ID, now.Format("200601021504"))
	}
	payload := mergeAnyMaps(nonNilMap(req.RequestPayload), map[string]any{
		"requestedVia": "tenant_api",
		"platformType": platform.Type,
	})
	job := model.MediaAccountSyncJob{
		ID:                fmt.Sprintf("masj_%d", now.UnixNano()),
		WorkspaceID:       workspaceID,
		MediaAccountID:    account.ID,
		PlatformID:        account.PlatformID,
		RequestedByUserID: middleware.CurrentUserID(c),
		SyncType:          syncType,
		Status:            "queued",
		RequestedAt:       now,
		IdempotencyKey:    idempotencyKey,
		RequestPayload:    payload,
		ResultSummary: map[string]any{
			"message": "sync job queued; connector execution is not implemented in this skeleton",
		},
		CreatedAt: now,
		UpdatedAt: now,
	}

	if h.db != nil && h.db.SQL() != nil {
		dbCtx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
		defer cancel()
		saved, err := h.db.CreateMediaAccountSyncJob(dbCtx, job)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "media account sync job was not persisted"})
			return
		}
		// 同步幂等边界只保证同账号同 idempotencyKey 复用同一 job；真实平台抓取由后续 worker/outbox 接管。
		c.JSON(http.StatusAccepted, saved)
		return
	}

	h.mu.Lock()
	h.updateAccountSyncStateLocked(workspaceID, account.ID, job)
	h.mu.Unlock()
	c.JSON(http.StatusAccepted, job)
}

func (h *WorkspaceHandler) mediaAccountMatrixSnapshotLocked(workspaceID string) []model.MediaAccountMatrixItem {
	accounts := filterByWorkspace(h.accounts, workspaceID, func(item model.MediaAccount) string { return item.WorkspaceID })
	items := make([]model.MediaAccountMatrixItem, 0, len(accounts))
	for _, account := range accounts {
		platform, _ := h.mediaPlatformByID(account.PlatformID)
		freshness := accountSnapshotFreshness(account)
		items = append(items, model.MediaAccountMatrixItem{
			Account:            account,
			Platform:           platform,
			ContentMetricCount: 0,
			DataFreshness:      freshness,
			Warnings:           mediaAccountMatrixWarnings(account, platform, freshness),
		})
	}
	return items
}

func (h *WorkspaceHandler) updateAccountSyncStateLocked(workspaceID string, accountID string, job model.MediaAccountSyncJob) {
	for index := range h.accounts {
		account := &h.accounts[index]
		if account.WorkspaceID != workspaceID || account.ID != accountID {
			continue
		}
		account.LastSyncJobID = job.ID
		account.LastSyncStatus = job.Status
		account.LastSyncMessage = job.ErrorMessage
		if account.LastSyncMessage == "" {
			account.LastSyncMessage = "sync job queued"
		}
		return
	}
}

func boundedQueryInt(c *gin.Context, name string, fallback int, min int, max int) int {
	value := strings.TrimSpace(c.Query(name))
	if value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}
	if parsed < min {
		return min
	}
	if parsed > max {
		return max
	}
	return parsed
}

func validMediaAccountSyncType(value string) bool {
	switch value {
	case "profile", "metrics", "content_metrics", "full":
		return true
	default:
		return false
	}
}

func accountSnapshotFreshness(account model.MediaAccount) string {
	// 数据新鲜度用于运营判断，不代表授权状态；授权问题单独进入 health/warnings。
	if account.LastMetricsSyncedAt == nil {
		return "missing"
	}
	age := time.Since(*account.LastMetricsSyncedAt)
	if age <= 24*time.Hour {
		return "fresh"
	}
	if age <= 72*time.Hour {
		return "stale"
	}
	return "expired"
}

func mediaAccountMatrixWarnings(account model.MediaAccount, platform model.MediaPlatform, freshness string) []string {
	warnings := []string{}
	if platform.ID == "" {
		warnings = append(warnings, "media platform is missing")
	} else if !platform.Enabled {
		warnings = append(warnings, "media platform is disabled")
	}
	if account.Status != "connected" {
		warnings = append(warnings, "media account authorization is not connected")
	}
	if freshness == "missing" {
		warnings = append(warnings, "metrics have not been synced")
	} else if freshness == "expired" {
		warnings = append(warnings, "metrics are older than the freshness window")
	}
	return warnings
}

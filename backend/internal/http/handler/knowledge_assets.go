package handler

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"geopress/backend/internal/ai"
	"geopress/backend/internal/http/middleware"
	"geopress/backend/internal/knowledge"
	"geopress/backend/internal/model"

	"github.com/gin-gonic/gin"
)

const maxKnowledgeAssetUploadBytes = 20 << 20
const knowledgeAssetAIEnhancementQueueSize = 64

type knowledgeAssetAIEnhancementJob struct {
	AssetID           string
	WorkspaceID       string
	TaskID            string
	KnowledgeBaseName string
	Summary           string
	Tags              []string
}

type createKnowledgeAssetRequest struct {
	KnowledgeBaseID      string         `json:"knowledgeBaseId"`
	KnowledgeBaseIDs     []string       `json:"knowledgeBaseIds"`
	Title                string         `json:"title"`
	Text                 string         `json:"text"`
	Content              string         `json:"content"`
	MimeType             string         `json:"mimeType"`
	AssetType            string         `json:"assetType"`
	OriginalFilename     string         `json:"originalFilename"`
	Summary              string         `json:"summary"`
	Tags                 []string       `json:"tags"`
	Metadata             map[string]any `json:"metadata"`
	AIEnhancementEnabled bool           `json:"aiEnhancementEnabled"`
}

type updateKnowledgeAssetBasesRequest struct {
	KnowledgeBaseIDs []string `json:"knowledgeBaseIds"`
}

type parsedKnowledgeAssetInput struct {
	KnowledgeBaseIDs     []string
	Title                string
	Data                 []byte
	MimeType             string
	AssetType            string
	OriginalFilename     string
	Summary              string
	Tags                 []string
	Metadata             map[string]any
	AIEnhancementEnabled bool
}

type knowledgeAIVisualOCRProvider struct {
	provider    ai.OCRProvider
	workspaceID string
	userID      string
}

func (provider knowledgeAIVisualOCRProvider) ExtractText(ctx context.Context, input knowledge.OCRInput) (knowledge.OCRResult, error) {
	response, err := provider.provider.ExtractDocumentText(ctx, ai.OCRRequest{
		WorkspaceID: provider.workspaceID,
		UserID:      provider.userID,
		Filename:    input.Filename,
		MimeType:    input.FileType.MimeType,
		Data:        input.Data,
		FileKind:    string(input.FileType.Kind),
	})
	if err != nil {
		return knowledge.OCRResult{}, err
	}
	return knowledge.OCRResult{
		Text: response.Text,
		Metadata: map[string]any{
			"ocrProvider":     response.Provider,
			"ocrModel":        response.Model,
			"ocrInputTokens":  response.TokenUsage.InputTokens,
			"ocrOutputTokens": response.TokenUsage.OutputTokens,
			"ocrTotalTokens":  response.TokenUsage.TotalTokens,
		},
	}, nil
}

func (h *WorkspaceHandler) ListKnowledgeAssets(c *gin.Context) {
	workspaceID, ok := h.authorizedWorkspaceID(c)
	if !ok {
		return
	}
	knowledgeBaseID := strings.TrimSpace(c.Query("knowledgeBaseId"))

	h.mu.RLock()
	items := filterByWorkspace(h.knowledgeAssets, workspaceID, func(item model.KnowledgeAsset) string { return item.WorkspaceID })
	items = filterActiveKnowledgeAssets(items)
	if knowledgeBaseID != "" {
		items = filterKnowledgeAssetsByBase(items, knowledgeBaseID)
	}
	h.mu.RUnlock()

	c.JSON(http.StatusOK, gin.H{"items": items})
}

func (h *WorkspaceHandler) CreateKnowledgeAsset(c *gin.Context) {
	workspaceID, ok := h.authorizedWorkspaceID(c)
	if !ok {
		return
	}

	input, err := parseKnowledgeAssetInput(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if len(input.Data) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "asset content is required"})
		return
	}
	if !isAllowedKnowledgeAssetFile(input.OriginalFilename, input.MimeType, input.Data) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "file type is not supported"})
		return
	}

	userID := middleware.CurrentUserID(c)
	h.mu.RLock()
	if len(input.KnowledgeBaseIDs) > 0 && !h.hasKnowledgeBasesLocked(workspaceID, input.KnowledgeBaseIDs) {
		h.mu.RUnlock()
		c.JSON(http.StatusNotFound, gin.H{"error": "knowledge base not found"})
		return
	}
	user, userOK := h.userByID(userID)
	baseName := h.knowledgeBaseNamesLocked(workspaceID, input.KnowledgeBaseIDs)
	h.mu.RUnlock()
	if !userOK {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "user not found"})
		return
	}
	if input.AIEnhancementEnabled && !user.HasActiveVIP(time.Now().UTC()) {
		c.JSON(http.StatusForbidden, gin.H{"error": "VIP subscription is required for AI knowledge enhancement"})
		return
	}

	now := time.Now().UTC()
	assetID := fmt.Sprintf("kba_%d", now.UnixNano())
	taskID := fmt.Sprintf("kbpt_%d", now.UnixNano())
	checksum := sha256.Sum256(input.Data)
	metadata := cloneAnyMap(input.Metadata)
	metadata["byteSize"] = len(input.Data)
	metadata["source"] = "workspace_api"
	if input.Summary != "" {
		metadata["summary"] = input.Summary
	}
	if len(input.Tags) > 0 {
		metadata["tags"] = input.Tags
	}
	asset := model.KnowledgeAsset{
		ID:                   assetID,
		WorkspaceID:          workspaceID,
		KnowledgeBaseIDs:     input.KnowledgeBaseIDs,
		Title:                input.Title,
		AssetType:            defaultString(input.AssetType, "document"),
		MimeType:             input.MimeType,
		OriginalFilename:     input.OriginalFilename,
		StorageKey:           fmt.Sprintf("inline:%s", assetID),
		Checksum:             hex.EncodeToString(checksum[:]),
		SourceData:           append([]byte(nil), input.Data...),
		Status:               "processing",
		Progress:             20,
		AIEnhancementEnabled: input.AIEnhancementEnabled,
		AIEnhancementStatus:  knowledgeAssetAIStatus(input.AIEnhancementEnabled),
		Metadata:             metadata,
		CreatedAt:            now,
		UpdatedAt:            now,
	}
	task := model.KnowledgeProcessingTask{
		ID:          taskID,
		AssetID:     assetID,
		WorkspaceID: workspaceID,
		TaskType:    "extract",
		Status:      "running",
		Progress:    20,
		CreatedAt:   now,
		StartedAt:   &now,
		UpdatedAt:   now,
	}

	if err := h.saveKnowledgeAsset(c.Request.Context(), asset); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "knowledge asset was not persisted"})
		return
	}
	if err := h.saveKnowledgeProcessingTask(c.Request.Context(), task); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "knowledge processing task was not persisted"})
		return
	}

	h.mu.Lock()
	h.knowledgeAssets = upsertKnowledgeAsset(h.knowledgeAssets, asset)
	h.knowledgeProcessingTasks = upsertKnowledgeProcessingTask(h.knowledgeProcessingTasks, task)
	h.mu.Unlock()

	asset, task, chunks, err := h.processKnowledgeAsset(c.Request.Context(), asset, task, input, user, baseName)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "knowledge asset processing result was not persisted"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"asset":  asset,
		"task":   task,
		"chunks": chunks,
	})
}

func (h *WorkspaceHandler) GetKnowledgeAsset(c *gin.Context) {
	workspaceID, ok := h.authorizedWorkspaceID(c)
	if !ok {
		return
	}
	assetID := strings.TrimSpace(c.Param("assetId"))

	h.mu.RLock()
	asset, found := h.knowledgeAssetByIDLocked(workspaceID, assetID)
	h.mu.RUnlock()
	if !found {
		c.JSON(http.StatusNotFound, gin.H{"error": "knowledge asset not found"})
		return
	}

	c.JSON(http.StatusOK, asset)
}

func (h *WorkspaceHandler) UpdateKnowledgeAssetBases(c *gin.Context) {
	workspaceID, ok := h.authorizedWorkspaceID(c)
	if !ok {
		return
	}
	assetID := strings.TrimSpace(c.Param("assetId"))
	if assetID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "assetId is required"})
		return
	}

	var req updateKnowledgeAssetBasesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}
	knowledgeBaseIDs := cleanKnowledgeBaseIDs("", req.KnowledgeBaseIDs)

	now := time.Now().UTC()
	h.mu.RLock()
	asset, found := h.knowledgeAssetByIDLocked(workspaceID, assetID)
	if !found {
		h.mu.RUnlock()
		c.JSON(http.StatusNotFound, gin.H{"error": "knowledge asset not found"})
		return
	}
	if len(knowledgeBaseIDs) > 0 && !h.hasKnowledgeBasesLocked(workspaceID, knowledgeBaseIDs) {
		h.mu.RUnlock()
		c.JSON(http.StatusNotFound, gin.H{"error": "knowledge base not found"})
		return
	}
	h.mu.RUnlock()

	// 资产归类是生成检索的租户边界，同时同步 chunk 绑定，避免资产和片段出现可见性分歧。
	if err := h.assignKnowledgeAssetToBases(c.Request.Context(), workspaceID, asset.ID, knowledgeBaseIDs, now); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "knowledge asset base assignment was not persisted"})
		return
	}

	chunks := []model.KnowledgeChunk{}
	h.mu.Lock()
	for index := range h.knowledgeAssets {
		if h.knowledgeAssets[index].WorkspaceID == workspaceID && h.knowledgeAssets[index].ID == asset.ID {
			h.knowledgeAssets[index].KnowledgeBaseIDs = append([]string(nil), knowledgeBaseIDs...)
			h.knowledgeAssets[index].UpdatedAt = now
			asset = h.knowledgeAssets[index]
			break
		}
	}
	for index := range h.knowledgeChunks {
		if h.knowledgeChunks[index].WorkspaceID != workspaceID || h.knowledgeChunks[index].AssetID != asset.ID {
			continue
		}
		h.knowledgeChunks[index].KnowledgeBaseIDs = append([]string(nil), knowledgeBaseIDs...)
		h.knowledgeChunks[index].UpdatedAt = now
		chunks = append(chunks, h.knowledgeChunks[index])
	}
	h.recountKnowledgeBaseItemsLocked(workspaceID)
	h.mu.Unlock()

	c.JSON(http.StatusOK, gin.H{"asset": asset, "chunks": chunks})
}

func (h *WorkspaceHandler) TrashKnowledgeAsset(c *gin.Context) {
	workspaceID, ok := h.authorizedWorkspaceID(c)
	if !ok {
		return
	}
	assetID := strings.TrimSpace(c.Param("assetId"))
	if assetID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "assetId is required"})
		return
	}

	now := time.Now().UTC()
	expiresAt := now.Add(30 * 24 * time.Hour)
	h.mu.RLock()
	asset, found := h.knowledgeAssetByIDLocked(workspaceID, assetID)
	h.mu.RUnlock()
	if !found {
		c.JSON(http.StatusNotFound, gin.H{"error": "knowledge asset not found"})
		return
	}
	if isKnowledgeAssetTrashed(asset) {
		c.JSON(http.StatusOK, asset)
		return
	}

	previousStatus := asset.Status
	if previousStatus == "" || previousStatus == "archived" {
		previousStatus = "ready"
	}
	if err := h.trashKnowledgeAsset(c.Request.Context(), workspaceID, assetID, previousStatus, now, expiresAt); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "knowledge asset was not moved to trash"})
		return
	}

	asset.Status = "archived"
	asset.DeletedAt = &now
	asset.DeleteExpiresAt = &expiresAt
	if asset.Metadata == nil {
		asset.Metadata = map[string]any{}
	}
	asset.Metadata["previousStatus"] = previousStatus
	asset.UpdatedAt = now
	h.mu.Lock()
	h.replaceKnowledgeAssetLocked(asset)
	h.recountKnowledgeBaseItemsLocked(workspaceID)
	h.mu.Unlock()
	c.JSON(http.StatusOK, asset)
}

func (h *WorkspaceHandler) RestoreKnowledgeAsset(c *gin.Context) {
	workspaceID, ok := h.authorizedWorkspaceID(c)
	if !ok {
		return
	}
	assetID := strings.TrimSpace(c.Param("assetId"))
	if assetID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "assetId is required"})
		return
	}

	now := time.Now().UTC()
	h.mu.RLock()
	asset, found := h.knowledgeAssetByIDLocked(workspaceID, assetID)
	h.mu.RUnlock()
	if !found {
		c.JSON(http.StatusNotFound, gin.H{"error": "knowledge asset not found"})
		return
	}
	nextStatus := restoredKnowledgeAssetStatus(asset)
	if err := h.restoreKnowledgeAsset(c.Request.Context(), workspaceID, assetID, nextStatus, now); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "knowledge asset was not restored"})
		return
	}

	asset.Status = nextStatus
	asset.DeletedAt = nil
	asset.DeleteExpiresAt = nil
	if asset.Metadata != nil {
		delete(asset.Metadata, "previousStatus")
	}
	asset.UpdatedAt = now
	h.mu.Lock()
	h.replaceKnowledgeAssetLocked(asset)
	h.recountKnowledgeBaseItemsLocked(workspaceID)
	h.mu.Unlock()
	c.JSON(http.StatusOK, asset)
}

func (h *WorkspaceHandler) DeleteKnowledgeAsset(c *gin.Context) {
	workspaceID, ok := h.authorizedWorkspaceID(c)
	if !ok {
		return
	}
	assetID := strings.TrimSpace(c.Param("assetId"))
	if assetID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "assetId is required"})
		return
	}

	h.mu.RLock()
	_, found := h.knowledgeAssetByIDLocked(workspaceID, assetID)
	h.mu.RUnlock()
	if !found {
		c.JSON(http.StatusNotFound, gin.H{"error": "knowledge asset not found"})
		return
	}
	if err := h.deleteKnowledgeAsset(c.Request.Context(), workspaceID, assetID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "knowledge asset was not deleted"})
		return
	}

	h.mu.Lock()
	h.removeKnowledgeAssetLocked(workspaceID, assetID)
	h.removeKnowledgeAssetChunksLocked(workspaceID, assetID)
	h.removeKnowledgeAssetTasksLocked(workspaceID, assetID)
	h.recountKnowledgeBaseItemsLocked(workspaceID)
	h.mu.Unlock()

	c.JSON(http.StatusOK, gin.H{"deleted": true})
}

func (h *WorkspaceHandler) RetryKnowledgeAssetProcessing(c *gin.Context) {
	workspaceID, ok := h.authorizedWorkspaceID(c)
	if !ok {
		return
	}
	assetID := strings.TrimSpace(c.Param("assetId"))
	if assetID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "assetId is required"})
		return
	}

	userID := middleware.CurrentUserID(c)
	h.mu.RLock()
	asset, found := h.knowledgeAssetByIDLocked(workspaceID, assetID)
	user, userOK := h.userByID(userID)
	baseName := h.knowledgeBaseNamesLocked(workspaceID, asset.KnowledgeBaseIDs)
	h.mu.RUnlock()
	if !found {
		c.JSON(http.StatusNotFound, gin.H{"error": "knowledge asset not found"})
		return
	}
	if !userOK {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "user not found"})
		return
	}
	if isKnowledgeAssetTrashed(asset) {
		c.JSON(http.StatusConflict, gin.H{"error": "trashed knowledge asset must be restored before retry"})
		return
	}

	input, err := retryInputForKnowledgeAsset(asset)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	now := time.Now().UTC()
	task := model.KnowledgeProcessingTask{
		ID:          fmt.Sprintf("kbpt_retry_%s_%d", asset.ID, now.UnixNano()),
		AssetID:     asset.ID,
		WorkspaceID: asset.WorkspaceID,
		TaskType:    "extract_retry",
		Status:      "running",
		Progress:    20,
		CreatedAt:   now,
		StartedAt:   &now,
		UpdatedAt:   now,
	}
	asset.Status = "processing"
	asset.Progress = 20
	asset.ErrorMessage = ""
	asset.ExtractedText = ""
	asset.AIEnhancementStatus = knowledgeAssetAIStatus(asset.AIEnhancementEnabled)
	asset.UpdatedAt = now
	if err := h.saveKnowledgeAsset(c.Request.Context(), asset); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "knowledge asset retry state was not persisted"})
		return
	}
	if err := h.saveKnowledgeProcessingTask(c.Request.Context(), task); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "knowledge retry task was not persisted"})
		return
	}
	if err := h.replaceKnowledgeChunksForAsset(c.Request.Context(), asset.ID, asset.WorkspaceID, nil); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "knowledge chunks were not reset for retry"})
		return
	}

	h.mu.Lock()
	h.knowledgeAssets = upsertKnowledgeAsset(h.knowledgeAssets, asset)
	h.knowledgeProcessingTasks = upsertKnowledgeProcessingTask(h.knowledgeProcessingTasks, task)
	h.knowledgeChunks = removeKnowledgeChunksForAssetInMemory(h.knowledgeChunks, asset.ID, asset.WorkspaceID)
	h.mu.Unlock()

	asset, task, chunks, err := h.processKnowledgeAsset(c.Request.Context(), asset, task, input, user, baseName)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "knowledge asset retry result was not persisted"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"asset":  asset,
		"task":   task,
		"chunks": chunks,
	})
}

func (h *WorkspaceHandler) EnhanceKnowledgeAsset(c *gin.Context) {
	workspaceID, ok := h.authorizedWorkspaceID(c)
	if !ok {
		return
	}
	assetID := strings.TrimSpace(c.Param("assetId"))
	if assetID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "assetId is required"})
		return
	}

	userID := middleware.CurrentUserID(c)
	h.mu.RLock()
	asset, found := h.knowledgeAssetByIDLocked(workspaceID, assetID)
	user, userOK := h.userByID(userID)
	baseName := h.knowledgeBaseNamesLocked(workspaceID, asset.KnowledgeBaseIDs)
	chunks := filterKnowledgeChunksByAsset(filterByWorkspace(h.knowledgeChunks, workspaceID, func(item model.KnowledgeChunk) string { return item.WorkspaceID }), assetID)
	h.mu.RUnlock()
	if !found {
		c.JSON(http.StatusNotFound, gin.H{"error": "knowledge asset not found"})
		return
	}
	if !userOK {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "user not found"})
		return
	}
	if !user.HasActiveVIP(time.Now().UTC()) {
		c.JSON(http.StatusForbidden, gin.H{"error": "VIP subscription is required for AI knowledge enhancement"})
		return
	}
	if isKnowledgeAssetTrashed(asset) {
		c.JSON(http.StatusConflict, gin.H{"error": "trashed knowledge asset must be restored before AI enhancement"})
		return
	}
	if asset.Status != "ready" {
		c.JSON(http.StatusConflict, gin.H{"error": "knowledge asset must be ready before AI enhancement"})
		return
	}
	if asset.AIEnhancementStatus == "pending" || asset.AIEnhancementStatus == "processing" {
		c.JSON(http.StatusConflict, gin.H{"error": "knowledge asset AI enhancement is already running"})
		return
	}
	if !canStartKnowledgeAssetAIEnhancement(asset) {
		c.JSON(http.StatusConflict, gin.H{"error": "knowledge asset AI enhancement has already completed"})
		return
	}

	// 后置 AI 增强只处理已经可检索的资产；失败时保留基础解析结果，不影响主资产可用性。
	asset.AIEnhancementEnabled = true
	asset.AIEnhancementStatus = "disabled"
	input := aiEnhancementInputForKnowledgeAsset(asset)
	asset, err := h.enqueueKnowledgeAssetAIEnhancement(c.Request.Context(), asset, input, baseName)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "knowledge asset AI enhancement was not queued"})
		return
	}

	task, ok := h.latestKnowledgeProcessingTask(workspaceID, assetID, "ai_enhance")
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "knowledge asset AI enhancement task was not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"asset":  asset,
		"task":   task,
		"chunks": chunks,
	})
}

func (h *WorkspaceHandler) ListKnowledgeAssetChunks(c *gin.Context) {
	workspaceID, ok := h.authorizedWorkspaceID(c)
	if !ok {
		return
	}
	assetID := strings.TrimSpace(c.Param("assetId"))

	h.mu.RLock()
	if _, found := h.knowledgeAssetByIDLocked(workspaceID, assetID); !found {
		h.mu.RUnlock()
		c.JSON(http.StatusNotFound, gin.H{"error": "knowledge asset not found"})
		return
	}
	items := filterByWorkspace(h.knowledgeChunks, workspaceID, func(item model.KnowledgeChunk) string { return item.WorkspaceID })
	items = filterKnowledgeChunksByAsset(items, assetID)
	h.mu.RUnlock()

	c.JSON(http.StatusOK, gin.H{"items": items})
}

func (h *WorkspaceHandler) ListKnowledgeAssetTasks(c *gin.Context) {
	workspaceID, ok := h.authorizedWorkspaceID(c)
	if !ok {
		return
	}
	assetID := strings.TrimSpace(c.Param("assetId"))

	h.mu.RLock()
	if _, found := h.knowledgeAssetByIDLocked(workspaceID, assetID); !found {
		h.mu.RUnlock()
		c.JSON(http.StatusNotFound, gin.H{"error": "knowledge asset not found"})
		return
	}
	items := filterByWorkspace(h.knowledgeProcessingTasks, workspaceID, func(item model.KnowledgeProcessingTask) string { return item.WorkspaceID })
	items = filterKnowledgeTasksByAsset(items, assetID)
	h.mu.RUnlock()

	c.JSON(http.StatusOK, gin.H{"items": items})
}

func (h *WorkspaceHandler) processKnowledgeAsset(ctx context.Context, asset model.KnowledgeAsset, task model.KnowledgeProcessingTask, input parsedKnowledgeAssetInput, user model.User, knowledgeBaseName string) (model.KnowledgeAsset, model.KnowledgeProcessingTask, []model.KnowledgeChunk, error) {
	ocrProvider, ocrUnavailableReason := h.knowledgeOCRProvider(user, asset, input)
	result, err := knowledge.Process(knowledge.ProcessInput{
		Context:              ctx,
		AssetTitle:           asset.Title,
		KnowledgeBaseName:    knowledgeBaseName,
		Filename:             input.OriginalFilename,
		MimeType:             input.MimeType,
		Data:                 input.Data,
		Summary:              input.Summary,
		Tags:                 input.Tags,
		Metadata:             input.Metadata,
		OCRProvider:          ocrProvider,
		OCRUnavailableReason: ocrUnavailableReason,
	})

	now := time.Now().UTC()
	task.UpdatedAt = now
	task.FinishedAt = &now
	asset.UpdatedAt = now
	asset.Metadata = cloneAnyMap(asset.Metadata)
	for key, value := range result.Extraction.Metadata {
		asset.Metadata[key] = value
	}

	if err != nil {
		asset.Status = "failed"
		asset.Progress = 100
		asset.ErrorMessage = result.Extraction.ErrorMessage
		if asset.ErrorMessage == "" {
			asset.ErrorMessage = err.Error()
		}
		if asset.AIEnhancementEnabled {
			asset.AIEnhancementStatus = "skipped"
		}
		task.Status = "failed"
		task.Progress = 100
		task.ErrorMessage = asset.ErrorMessage
		persistErr := h.persistProcessedKnowledgeAsset(ctx, asset, task, nil)
		return asset, task, []model.KnowledgeChunk{}, persistErr
	}

	asset.Status = "ready"
	asset.Progress = 100
	asset.ErrorMessage = ""
	asset.ExtractedText = result.Extraction.Text
	if asset.MimeType == "" {
		asset.MimeType = result.Extraction.FileType.MimeType
	}
	task.Status = "succeeded"
	task.Progress = 100
	task.ErrorMessage = ""

	chunks := knowledgeChunksForAsset(asset, result.Chunks, now)
	if len(chunks) == 0 {
		asset.Status = "failed"
		asset.ErrorMessage = "no knowledge chunks were produced"
		if asset.AIEnhancementEnabled {
			asset.AIEnhancementStatus = "skipped"
		}
		task.Status = "failed"
		task.ErrorMessage = asset.ErrorMessage
	}

	if err := h.persistProcessedKnowledgeAsset(ctx, asset, task, chunks); err != nil {
		return asset, task, chunks, err
	}
	if asset.Status != "ready" || !asset.AIEnhancementEnabled {
		return asset, task, chunks, nil
	}
	asset, err = h.enqueueKnowledgeAssetAIEnhancement(ctx, asset, input, knowledgeBaseName)
	if err != nil {
		return asset, task, chunks, err
	}
	return asset, task, chunks, nil
}

func (h *WorkspaceHandler) knowledgeOCRProvider(user model.User, asset model.KnowledgeAsset, input parsedKnowledgeAssetInput) (knowledge.OCRProvider, string) {
	fileType := knowledge.DetectFileType(input.OriginalFilename, input.MimeType, input.Data)
	if fileType.Kind != knowledge.FileKindImage && fileType.Kind != knowledge.FileKindPDF {
		return nil, ""
	}
	asset.Metadata["ocrRequired"] = true
	asset.Metadata["ocrEntitlement"] = "paid_subscription"
	if !user.HasActiveVIP(time.Now().UTC()) {
		return nil, "AI vision OCR for images and PDFs requires a paid subscription"
	}
	ocrProvider := ai.NewOCRProvider(h.aiConfig.Snapshot())
	if ocrProvider == nil {
		return nil, "AI vision OCR provider is not configured"
	}
	return knowledgeAIVisualOCRProvider{
		provider:    ocrProvider,
		workspaceID: asset.WorkspaceID,
		userID:      user.ID,
	}, ""
}

func (h *WorkspaceHandler) enqueueKnowledgeAssetAIEnhancement(ctx context.Context, asset model.KnowledgeAsset, input parsedKnowledgeAssetInput, knowledgeBaseName string) (model.KnowledgeAsset, error) {
	now := time.Now().UTC()
	task := model.KnowledgeProcessingTask{
		ID:          fmt.Sprintf("kbpt_ai_%s_%d", asset.ID, now.UnixNano()),
		AssetID:     asset.ID,
		WorkspaceID: asset.WorkspaceID,
		TaskType:    "ai_enhance",
		Status:      "queued",
		Progress:    5,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	asset.AIEnhancementStatus = "pending"
	asset.UpdatedAt = now
	if err := h.saveKnowledgeAsset(ctx, asset); err != nil {
		return asset, err
	}
	if err := h.saveKnowledgeProcessingTask(ctx, task); err != nil {
		return asset, err
	}
	h.mu.Lock()
	h.knowledgeAssets = upsertKnowledgeAsset(h.knowledgeAssets, asset)
	h.knowledgeProcessingTasks = upsertKnowledgeProcessingTask(h.knowledgeProcessingTasks, task)
	h.mu.Unlock()

	job := knowledgeAssetAIEnhancementJob{
		AssetID:           asset.ID,
		WorkspaceID:       asset.WorkspaceID,
		TaskID:            task.ID,
		KnowledgeBaseName: knowledgeBaseName,
		Summary:           input.Summary,
		Tags:              append([]string(nil), input.Tags...),
	}
	if h.knowledgeAssetAIQueue == nil {
		return h.failQueuedKnowledgeAssetAIEnhancement(ctx, asset, task, errors.New("knowledge asset AI enhancement queue is not initialized"))
	}
	select {
	case h.knowledgeAssetAIQueue <- job:
		return asset, nil
	default:
		return h.failQueuedKnowledgeAssetAIEnhancement(ctx, asset, task, errors.New("knowledge asset AI enhancement queue is full"))
	}
}

func (h *WorkspaceHandler) failQueuedKnowledgeAssetAIEnhancement(ctx context.Context, asset model.KnowledgeAsset, task model.KnowledgeProcessingTask, cause error) (model.KnowledgeAsset, error) {
	now := time.Now().UTC()
	asset.AIEnhancementStatus = "failed"
	asset.UpdatedAt = now
	task.Status = "failed"
	task.Progress = 100
	task.ErrorMessage = cause.Error()
	task.FinishedAt = &now
	task.UpdatedAt = now
	if err := h.saveKnowledgeAsset(ctx, asset); err != nil {
		return asset, err
	}
	if err := h.saveKnowledgeProcessingTask(ctx, task); err != nil {
		return asset, err
	}
	h.mu.Lock()
	h.knowledgeAssets = upsertKnowledgeAsset(h.knowledgeAssets, asset)
	h.knowledgeProcessingTasks = upsertKnowledgeProcessingTask(h.knowledgeProcessingTasks, task)
	h.mu.Unlock()
	return asset, nil
}

func (h *WorkspaceHandler) startKnowledgeAssetAIWorker() {
	if h.knowledgeAssetAIQueue == nil {
		h.knowledgeAssetAIQueue = make(chan knowledgeAssetAIEnhancementJob, knowledgeAssetAIEnhancementQueueSize)
	}
	workerCtx, cancel := context.WithCancel(context.Background())
	h.knowledgeAssetAIWorkerCancel = cancel
	go func() {
		for {
			select {
			case <-workerCtx.Done():
				return
			case job := <-h.knowledgeAssetAIQueue:
				func() {
					defer func() {
						if recovered := recover(); recovered != nil {
							log.Printf("knowledge asset AI enhancement worker recovered panic for task %s: %v", job.TaskID, recovered)
							ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
							defer cancel()
							h.markKnowledgeAssetAIEnhancementJobFailed(ctx, job, fmt.Errorf("AI enhancement worker panic: %v", recovered))
						}
					}()
					timeout := time.Duration(h.aiConfig.Snapshot().RequestTimeout+10) * time.Second
					if timeout < 15*time.Second {
						timeout = 15 * time.Second
					}
					ctx, cancel := context.WithTimeout(workerCtx, timeout)
					defer cancel()
					if err := h.runKnowledgeAssetAIEnhancementJob(ctx, job); err != nil {
						log.Printf("knowledge asset AI enhancement job %s failed: %v", job.TaskID, err)
					}
				}()
			}
		}
	}()
}

func (h *WorkspaceHandler) stopKnowledgeAssetAIWorker() {
	if h.knowledgeAssetAIWorkerCancel != nil {
		h.knowledgeAssetAIWorkerCancel()
	}
}

func (h *WorkspaceHandler) runKnowledgeAssetAIEnhancementJob(ctx context.Context, job knowledgeAssetAIEnhancementJob) error {
	asset, task, defaultChunks, ok := h.knowledgeAssetAIEnhancementJobState(job)
	if !ok {
		return fmt.Errorf("knowledge asset AI enhancement state not found for asset %s task %s", job.AssetID, job.TaskID)
	}
	now := time.Now().UTC()
	asset.AIEnhancementStatus = "processing"
	asset.UpdatedAt = now
	task.Status = "running"
	task.Progress = 20
	task.StartedAt = &now
	task.UpdatedAt = now
	if err := h.saveKnowledgeAsset(ctx, asset); err != nil {
		return err
	}
	if err := h.saveKnowledgeProcessingTask(ctx, task); err != nil {
		return err
	}
	h.mu.Lock()
	h.knowledgeAssets = upsertKnowledgeAsset(h.knowledgeAssets, asset)
	h.knowledgeProcessingTasks = upsertKnowledgeProcessingTask(h.knowledgeProcessingTasks, task)
	h.mu.Unlock()

	if _, _, err := h.runKnowledgeAssetAIEnhancement(ctx, asset, task, job, defaultChunks); err != nil {
		return err
	}
	return nil
}

func (h *WorkspaceHandler) knowledgeAssetAIEnhancementJobState(job knowledgeAssetAIEnhancementJob) (model.KnowledgeAsset, model.KnowledgeProcessingTask, []model.KnowledgeChunk, bool) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	asset, found := h.knowledgeAssetByIDLocked(job.WorkspaceID, job.AssetID)
	if !found {
		return model.KnowledgeAsset{}, model.KnowledgeProcessingTask{}, nil, false
	}
	var task model.KnowledgeProcessingTask
	taskFound := false
	for _, item := range h.knowledgeProcessingTasks {
		if item.WorkspaceID == job.WorkspaceID && item.AssetID == job.AssetID && item.ID == job.TaskID {
			task = item
			taskFound = true
			break
		}
	}
	if !taskFound {
		return model.KnowledgeAsset{}, model.KnowledgeProcessingTask{}, nil, false
	}
	chunks := filterKnowledgeChunksByAsset(filterByWorkspace(h.knowledgeChunks, job.WorkspaceID, func(item model.KnowledgeChunk) string { return item.WorkspaceID }), job.AssetID)
	return asset, task, chunks, true
}

func (h *WorkspaceHandler) runKnowledgeAssetAIEnhancement(ctx context.Context, asset model.KnowledgeAsset, task model.KnowledgeProcessingTask, job knowledgeAssetAIEnhancementJob, defaultChunks []model.KnowledgeChunk) (model.KnowledgeAsset, []model.KnowledgeChunk, error) {
	source := strings.TrimSpace(asset.ExtractedText)
	if source == "" {
		source = joinKnowledgeChunkContent(defaultChunks)
	}
	formatReq := ai.FormatKnowledgeContentRequest{
		WorkspaceID: asset.WorkspaceID,
		Type:        defaultString(asset.AssetType, "knowledge_asset"),
		Title:       asset.Title,
		Content:     source,
	}
	provider := h.aiConfig.Provider()
	response, err := provider.FormatKnowledgeContent(ctx, formatReq)
	fallback := false
	fallbackError := ""
	if err != nil && provider.Name() != ai.ProviderMock {
		log.Printf("knowledge asset AI enhancement provider %s failed, falling back to mock: %v", provider.Name(), err)
		fallback = true
		fallbackError = err.Error()
		response, err = ai.NewMockProvider().FormatKnowledgeContent(ctx, formatReq)
	}
	if err != nil {
		// 默认抽取和规则分块已经可用；AI 增强失败只影响增强阶段状态，不能让资产整体不可检索。
		return h.finishKnowledgeAssetAIEnhancementFailure(ctx, asset, task, defaultChunks, err)
	}

	enhancedText := strings.TrimSpace(response.Content)
	if enhancedText == "" {
		return h.finishKnowledgeAssetAIEnhancementFailure(ctx, asset, task, defaultChunks, errors.New("AI enhancement returned empty content"))
	}
	metadata := cloneAnyMap(asset.Metadata)
	metadata["aiEnhanced"] = true
	metadata["aiEnhancementProvider"] = response.Provider
	metadata["aiEnhancementModel"] = response.Model
	if fallback {
		metadata["aiEnhancementFallback"] = true
		metadata["aiEnhancementFallbackError"] = fallbackError
	}
	enhanced := knowledge.ChunkText(knowledge.ChunkInput{
		AssetTitle:        asset.Title,
		KnowledgeBaseName: job.KnowledgeBaseName,
		Text:              enhancedText,
		Summary:           job.Summary,
		Tags:              job.Tags,
		Metadata:          metadata,
	})
	enhancedChunks := knowledgeChunksForAsset(asset, enhanced, time.Now().UTC())
	if len(enhancedChunks) == 0 {
		return h.finishKnowledgeAssetAIEnhancementFailure(ctx, asset, task, defaultChunks, errors.New("AI enhancement produced no knowledge chunks"))
	}

	finishedAt := time.Now().UTC()
	asset.ExtractedText = enhancedText
	asset.AIEnhancementStatus = "succeeded"
	asset.Metadata = metadata
	asset.UpdatedAt = finishedAt
	task.Status = "succeeded"
	task.Progress = 100
	task.ErrorMessage = ""
	task.FinishedAt = &finishedAt
	task.UpdatedAt = finishedAt

	if err := h.saveKnowledgeAsset(ctx, asset); err != nil {
		return asset, defaultChunks, err
	}
	if err := h.saveKnowledgeProcessingTask(ctx, task); err != nil {
		return asset, defaultChunks, err
	}
	if err := h.replaceKnowledgeChunksForAsset(ctx, asset.ID, asset.WorkspaceID, enhancedChunks); err != nil {
		return asset, defaultChunks, err
	}

	h.mu.Lock()
	h.knowledgeAssets = upsertKnowledgeAsset(h.knowledgeAssets, asset)
	h.knowledgeProcessingTasks = upsertKnowledgeProcessingTask(h.knowledgeProcessingTasks, task)
	h.knowledgeChunks = replaceKnowledgeChunksForAssetInMemory(h.knowledgeChunks, asset.ID, enhancedChunks)
	h.mu.Unlock()
	return asset, enhancedChunks, nil
}

func (h *WorkspaceHandler) markKnowledgeAssetAIEnhancementJobFailed(ctx context.Context, job knowledgeAssetAIEnhancementJob, cause error) {
	asset, task, defaultChunks, ok := h.knowledgeAssetAIEnhancementJobState(job)
	if !ok {
		return
	}
	if _, _, err := h.finishKnowledgeAssetAIEnhancementFailure(ctx, asset, task, defaultChunks, cause); err != nil {
		log.Printf("knowledge asset AI enhancement failure state was not persisted for task %s: %v", job.TaskID, err)
	}
}

func (h *WorkspaceHandler) finishKnowledgeAssetAIEnhancementFailure(ctx context.Context, asset model.KnowledgeAsset, task model.KnowledgeProcessingTask, defaultChunks []model.KnowledgeChunk, cause error) (model.KnowledgeAsset, []model.KnowledgeChunk, error) {
	now := time.Now().UTC()
	asset.AIEnhancementStatus = "failed"
	asset.UpdatedAt = now
	task.Status = "failed"
	task.Progress = 100
	task.ErrorMessage = cause.Error()
	task.FinishedAt = &now
	task.UpdatedAt = now
	if err := h.saveKnowledgeAsset(ctx, asset); err != nil {
		return asset, defaultChunks, err
	}
	if err := h.saveKnowledgeProcessingTask(ctx, task); err != nil {
		return asset, defaultChunks, err
	}
	h.mu.Lock()
	h.knowledgeAssets = upsertKnowledgeAsset(h.knowledgeAssets, asset)
	h.knowledgeProcessingTasks = upsertKnowledgeProcessingTask(h.knowledgeProcessingTasks, task)
	h.mu.Unlock()
	return asset, defaultChunks, nil
}

func (h *WorkspaceHandler) persistProcessedKnowledgeAsset(ctx context.Context, asset model.KnowledgeAsset, task model.KnowledgeProcessingTask, chunks []model.KnowledgeChunk) error {
	if err := h.saveKnowledgeAsset(ctx, asset); err != nil {
		return err
	}
	if err := h.saveKnowledgeProcessingTask(ctx, task); err != nil {
		return err
	}
	for _, chunk := range chunks {
		if err := h.saveKnowledgeChunk(ctx, chunk); err != nil {
			return err
		}
	}

	h.mu.Lock()
	h.knowledgeAssets = upsertKnowledgeAsset(h.knowledgeAssets, asset)
	h.knowledgeProcessingTasks = upsertKnowledgeProcessingTask(h.knowledgeProcessingTasks, task)
	for _, chunk := range chunks {
		h.knowledgeChunks = upsertKnowledgeChunk(h.knowledgeChunks, chunk)
	}
	h.mu.Unlock()
	return nil
}

func knowledgeChunksForAsset(asset model.KnowledgeAsset, chunks []knowledge.Chunk, now time.Time) []model.KnowledgeChunk {
	items := make([]model.KnowledgeChunk, 0, len(chunks))
	for _, chunk := range chunks {
		items = append(items, model.KnowledgeChunk{
			ID:               fmt.Sprintf("kbc_%s_%03d", asset.ID, chunk.ChunkIndex),
			AssetID:          asset.ID,
			WorkspaceID:      asset.WorkspaceID,
			KnowledgeBaseIDs: append([]string(nil), asset.KnowledgeBaseIDs...),
			ChunkIndex:       chunk.ChunkIndex,
			Title:            chunk.Title,
			Content:          chunk.Content,
			SearchText:       chunk.SearchText,
			Summary:          chunk.Summary,
			Tags:             append([]string(nil), chunk.Tags...),
			Metadata:         cloneAnyMap(chunk.Metadata),
			Enabled:          true,
			EmbeddingStatus:  "pending",
			UpdatedAt:        now,
		})
	}
	return items
}

func retryInputForKnowledgeAsset(asset model.KnowledgeAsset) (parsedKnowledgeAssetInput, error) {
	data := append([]byte(nil), asset.SourceData...)
	if len(data) == 0 && strings.TrimSpace(asset.ExtractedText) != "" {
		data = []byte(asset.ExtractedText)
	}
	if len(data) == 0 {
		return parsedKnowledgeAssetInput{}, errors.New("knowledge asset source data is not available for retry")
	}
	metadata := cloneAnyMap(asset.Metadata)
	metadata["retrySource"] = "knowledge_asset_retry"
	return parsedKnowledgeAssetInput{
		KnowledgeBaseIDs:     append([]string(nil), asset.KnowledgeBaseIDs...),
		Title:                asset.Title,
		Data:                 data,
		MimeType:             asset.MimeType,
		AssetType:            asset.AssetType,
		OriginalFilename:     asset.OriginalFilename,
		Summary:              stringMetadata(asset.Metadata, "summary"),
		Tags:                 stringSliceMetadata(asset.Metadata, "tags"),
		Metadata:             metadata,
		AIEnhancementEnabled: asset.AIEnhancementEnabled,
	}, nil
}

func aiEnhancementInputForKnowledgeAsset(asset model.KnowledgeAsset) parsedKnowledgeAssetInput {
	return parsedKnowledgeAssetInput{
		KnowledgeBaseIDs:     append([]string(nil), asset.KnowledgeBaseIDs...),
		Title:                asset.Title,
		MimeType:             asset.MimeType,
		AssetType:            asset.AssetType,
		OriginalFilename:     asset.OriginalFilename,
		Summary:              stringMetadata(asset.Metadata, "summary"),
		Tags:                 stringSliceMetadata(asset.Metadata, "tags"),
		Metadata:             cloneAnyMap(asset.Metadata),
		AIEnhancementEnabled: true,
	}
}

func canStartKnowledgeAssetAIEnhancement(asset model.KnowledgeAsset) bool {
	status := strings.TrimSpace(asset.AIEnhancementStatus)
	if status == "" {
		status = "disabled"
	}
	if !asset.AIEnhancementEnabled {
		return true
	}
	return status == "disabled" || status == "failed" || status == "skipped"
}

func joinKnowledgeChunkContent(chunks []model.KnowledgeChunk) string {
	parts := make([]string, 0, len(chunks))
	for _, chunk := range chunks {
		content := strings.TrimSpace(chunk.Content)
		if content != "" {
			parts = append(parts, content)
		}
	}
	return strings.Join(parts, "\n\n")
}

func parseKnowledgeAssetInput(c *gin.Context) (parsedKnowledgeAssetInput, error) {
	contentType := strings.ToLower(c.GetHeader("Content-Type"))
	if strings.HasPrefix(contentType, "multipart/form-data") {
		return parseMultipartKnowledgeAssetInput(c)
	}
	return parseJSONKnowledgeAssetInput(c)
}

func parseJSONKnowledgeAssetInput(c *gin.Context) (parsedKnowledgeAssetInput, error) {
	var req createKnowledgeAssetRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		return parsedKnowledgeAssetInput{}, errors.New("invalid request body")
	}
	text := strings.TrimSpace(req.Text)
	if text == "" {
		text = strings.TrimSpace(req.Content)
	}
	title := strings.TrimSpace(req.Title)
	if title == "" {
		title = fallbackKnowledgeAssetTitle(req.OriginalFilename)
	}
	filename := strings.TrimSpace(req.OriginalFilename)
	if filename == "" {
		filename = title + ".md"
	}
	mimeType := strings.TrimSpace(req.MimeType)
	if mimeType == "" {
		mimeType = "text/markdown"
	}
	return parsedKnowledgeAssetInput{
		KnowledgeBaseIDs:     cleanKnowledgeBaseIDs(req.KnowledgeBaseID, req.KnowledgeBaseIDs),
		Title:                title,
		Data:                 []byte(text),
		MimeType:             mimeType,
		AssetType:            strings.TrimSpace(req.AssetType),
		OriginalFilename:     filename,
		Summary:              strings.TrimSpace(req.Summary),
		Tags:                 cleanKeywords(req.Tags),
		Metadata:             cloneAnyMap(req.Metadata),
		AIEnhancementEnabled: req.AIEnhancementEnabled,
	}, nil
}

func parseMultipartKnowledgeAssetInput(c *gin.Context) (parsedKnowledgeAssetInput, error) {
	if err := c.Request.ParseMultipartForm(maxKnowledgeAssetUploadBytes); err != nil {
		return parsedKnowledgeAssetInput{}, errors.New("invalid multipart body")
	}
	file, header, err := c.Request.FormFile("file")
	if err != nil {
		return parsedKnowledgeAssetInput{}, errors.New("file is required")
	}
	defer file.Close()
	data, err := readLimitedKnowledgeAssetFile(file, maxKnowledgeAssetUploadBytes)
	if err != nil {
		return parsedKnowledgeAssetInput{}, err
	}

	title := strings.TrimSpace(c.PostForm("title"))
	if title == "" {
		title = fallbackKnowledgeAssetTitle(header.Filename)
	}
	mimeType := strings.TrimSpace(header.Header.Get("Content-Type"))
	if formMime := strings.TrimSpace(c.PostForm("mimeType")); formMime != "" {
		mimeType = formMime
	}
	aiEnabled, _ := strconv.ParseBool(strings.TrimSpace(c.PostForm("aiEnhancementEnabled")))
	return parsedKnowledgeAssetInput{
		KnowledgeBaseIDs:     cleanKnowledgeBaseIDs(c.PostForm("knowledgeBaseId"), splitFormList(c.PostForm("knowledgeBaseIds"))),
		Title:                title,
		Data:                 data,
		MimeType:             mimeType,
		AssetType:            strings.TrimSpace(c.PostForm("assetType")),
		OriginalFilename:     header.Filename,
		Summary:              strings.TrimSpace(c.PostForm("summary")),
		Tags:                 splitFormList(c.PostForm("tags")),
		Metadata:             map[string]any{"uploadFilename": header.Filename},
		AIEnhancementEnabled: aiEnabled,
	}, nil
}

func readLimitedKnowledgeAssetFile(file multipart.File, maxBytes int64) ([]byte, error) {
	reader := io.LimitReader(file, maxBytes+1)
	data, err := io.ReadAll(reader)
	if err != nil {
		return nil, errors.New("file could not be read")
	}
	if int64(len(data)) > maxBytes {
		return nil, fmt.Errorf("file exceeds %d bytes", maxBytes)
	}
	return data, nil
}

func isAllowedKnowledgeAssetFile(filename string, mimeType string, data []byte) bool {
	fileType := knowledge.DetectFileType(filename, mimeType, data)
	ext := strings.ToLower(filepath.Ext(filename))
	normalizedMIME := strings.ToLower(strings.TrimSpace(strings.Split(mimeType, ";")[0]))
	switch fileType.Kind {
	case knowledge.FileKindDOC, knowledge.FileKindDOCX:
		return ext == ".doc" || ext == ".docx" || normalizedMIME == "application/msword" || normalizedMIME == "application/vnd.openxmlformats-officedocument.wordprocessingml.document"
	case knowledge.FileKindText:
		return ext == ".txt" || normalizedMIME == "text/plain"
	case knowledge.FileKindMarkdown:
		return ext == ".md" || ext == ".markdown" || normalizedMIME == "text/markdown" || normalizedMIME == "text/x-markdown"
	case knowledge.FileKindPDF:
		return ext == ".pdf" || normalizedMIME == "application/pdf"
	case knowledge.FileKindImage:
		return true
	default:
		return false
	}
}

func (h *WorkspaceHandler) knowledgeAssetByIDLocked(workspaceID string, assetID string) (model.KnowledgeAsset, bool) {
	for _, item := range h.knowledgeAssets {
		if item.WorkspaceID == workspaceID && item.ID == assetID {
			return item, true
		}
	}
	return model.KnowledgeAsset{}, false
}

func (h *WorkspaceHandler) knowledgeBaseNamesLocked(workspaceID string, knowledgeBaseIDs []string) string {
	names := make([]string, 0, len(knowledgeBaseIDs))
	for _, knowledgeBaseID := range knowledgeBaseIDs {
		for _, item := range h.knowledgeBases {
			if item.WorkspaceID == workspaceID && item.ID == knowledgeBaseID {
				names = append(names, item.Name)
				break
			}
		}
	}
	return strings.Join(names, ", ")
}

func (h *WorkspaceHandler) latestKnowledgeProcessingTask(workspaceID string, assetID string, taskType string) (model.KnowledgeProcessingTask, bool) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	for _, item := range h.knowledgeProcessingTasks {
		if item.WorkspaceID == workspaceID && item.AssetID == assetID && item.TaskType == taskType {
			return item, true
		}
	}
	return model.KnowledgeProcessingTask{}, false
}

func (h *WorkspaceHandler) saveKnowledgeAsset(ctx context.Context, item model.KnowledgeAsset) error {
	dbCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	return h.db.SaveKnowledgeAsset(dbCtx, item)
}

func (h *WorkspaceHandler) saveKnowledgeChunk(ctx context.Context, item model.KnowledgeChunk) error {
	dbCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	return h.db.SaveKnowledgeChunk(dbCtx, item)
}

func (h *WorkspaceHandler) replaceKnowledgeChunksForAsset(ctx context.Context, assetID string, workspaceID string, chunks []model.KnowledgeChunk) error {
	dbCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	return h.db.ReplaceKnowledgeChunksForAsset(dbCtx, assetID, workspaceID, chunks)
}

func (h *WorkspaceHandler) assignKnowledgeAssetToBases(ctx context.Context, workspaceID string, assetID string, knowledgeBaseIDs []string, updatedAt time.Time) error {
	dbCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	return h.db.AssignKnowledgeAssetToBases(dbCtx, workspaceID, assetID, knowledgeBaseIDs, updatedAt)
}

func (h *WorkspaceHandler) trashKnowledgeAsset(ctx context.Context, workspaceID string, assetID string, previousStatus string, deletedAt time.Time, deleteExpiresAt time.Time) error {
	dbCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	return h.db.TrashKnowledgeAsset(dbCtx, workspaceID, assetID, previousStatus, deletedAt, deleteExpiresAt)
}

func (h *WorkspaceHandler) restoreKnowledgeAsset(ctx context.Context, workspaceID string, assetID string, status string, updatedAt time.Time) error {
	dbCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	return h.db.RestoreKnowledgeAsset(dbCtx, workspaceID, assetID, status, updatedAt)
}

func (h *WorkspaceHandler) deleteKnowledgeAsset(ctx context.Context, workspaceID string, assetID string) error {
	dbCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	return h.db.DeleteKnowledgeAsset(dbCtx, workspaceID, assetID)
}

func (h *WorkspaceHandler) saveKnowledgeProcessingTask(ctx context.Context, item model.KnowledgeProcessingTask) error {
	dbCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	return h.db.SaveKnowledgeProcessingTask(dbCtx, item)
}

func filterKnowledgeAssetsByBase(items []model.KnowledgeAsset, knowledgeBaseID string) []model.KnowledgeAsset {
	filtered := make([]model.KnowledgeAsset, 0, len(items))
	for _, item := range items {
		if stringSliceContains(item.KnowledgeBaseIDs, knowledgeBaseID) {
			filtered = append(filtered, item)
		}
	}
	return filtered
}

func filterActiveKnowledgeAssets(items []model.KnowledgeAsset) []model.KnowledgeAsset {
	filtered := make([]model.KnowledgeAsset, 0, len(items))
	for _, item := range items {
		if !isKnowledgeAssetTrashed(item) {
			filtered = append(filtered, item)
		}
	}
	return filtered
}

func filterTrashedKnowledgeAssets(items []model.KnowledgeAsset) []model.KnowledgeAsset {
	filtered := make([]model.KnowledgeAsset, 0, len(items))
	for _, item := range items {
		if isKnowledgeAssetTrashed(item) {
			filtered = append(filtered, item)
		}
	}
	return filtered
}

func filterKnowledgeChunksByAsset(items []model.KnowledgeChunk, assetID string) []model.KnowledgeChunk {
	filtered := make([]model.KnowledgeChunk, 0, len(items))
	for _, item := range items {
		if item.AssetID == assetID {
			filtered = append(filtered, item)
		}
	}
	return filtered
}

func filterKnowledgeTasksByAsset(items []model.KnowledgeProcessingTask, assetID string) []model.KnowledgeProcessingTask {
	filtered := make([]model.KnowledgeProcessingTask, 0, len(items))
	for _, item := range items {
		if item.AssetID == assetID {
			filtered = append(filtered, item)
		}
	}
	return filtered
}

func upsertKnowledgeAsset(items []model.KnowledgeAsset, next model.KnowledgeAsset) []model.KnowledgeAsset {
	for i, item := range items {
		if item.ID == next.ID {
			items[i] = next
			return items
		}
	}
	return append([]model.KnowledgeAsset{next}, items...)
}

func upsertKnowledgeChunk(items []model.KnowledgeChunk, next model.KnowledgeChunk) []model.KnowledgeChunk {
	for i, item := range items {
		if item.ID == next.ID {
			items[i] = next
			return items
		}
	}
	return append(items, next)
}

func (h *WorkspaceHandler) replaceKnowledgeAssetLocked(updated model.KnowledgeAsset) {
	h.knowledgeAssets = upsertKnowledgeAsset(h.knowledgeAssets, updated)
}

func (h *WorkspaceHandler) removeKnowledgeAssetLocked(workspaceID string, assetID string) {
	filtered := h.knowledgeAssets[:0]
	for _, item := range h.knowledgeAssets {
		if item.WorkspaceID == workspaceID && item.ID == assetID {
			continue
		}
		filtered = append(filtered, item)
	}
	h.knowledgeAssets = filtered
}

func (h *WorkspaceHandler) removeKnowledgeAssetChunksLocked(workspaceID string, assetID string) {
	filtered := h.knowledgeChunks[:0]
	for _, item := range h.knowledgeChunks {
		if item.WorkspaceID == workspaceID && item.AssetID == assetID {
			continue
		}
		filtered = append(filtered, item)
	}
	h.knowledgeChunks = filtered
}

func (h *WorkspaceHandler) removeKnowledgeAssetTasksLocked(workspaceID string, assetID string) {
	filtered := h.knowledgeProcessingTasks[:0]
	for _, item := range h.knowledgeProcessingTasks {
		if item.WorkspaceID == workspaceID && item.AssetID == assetID {
			continue
		}
		filtered = append(filtered, item)
	}
	h.knowledgeProcessingTasks = filtered
}

func replaceKnowledgeChunksForAssetInMemory(items []model.KnowledgeChunk, assetID string, next []model.KnowledgeChunk) []model.KnowledgeChunk {
	replaced := make([]model.KnowledgeChunk, 0, len(items)+len(next))
	replaced = append(replaced, next...)
	for _, item := range items {
		if item.AssetID != assetID {
			replaced = append(replaced, item)
		}
	}
	return replaced
}

func removeKnowledgeChunksForAssetInMemory(items []model.KnowledgeChunk, assetID string, workspaceID string) []model.KnowledgeChunk {
	filtered := make([]model.KnowledgeChunk, 0, len(items))
	for _, item := range items {
		if item.WorkspaceID == workspaceID && item.AssetID == assetID {
			continue
		}
		filtered = append(filtered, item)
	}
	return filtered
}

func upsertKnowledgeProcessingTask(items []model.KnowledgeProcessingTask, next model.KnowledgeProcessingTask) []model.KnowledgeProcessingTask {
	for i, item := range items {
		if item.ID == next.ID {
			items[i] = next
			return items
		}
	}
	return append([]model.KnowledgeProcessingTask{next}, items...)
}

func cloneAnyMap(values map[string]any) map[string]any {
	cloned := map[string]any{}
	for key, value := range values {
		cloned[key] = value
	}
	return cloned
}

func stringMetadata(values map[string]any, key string) string {
	if raw, ok := values[key].(string); ok {
		return strings.TrimSpace(raw)
	}
	return ""
}

func stringSliceMetadata(values map[string]any, key string) []string {
	raw, ok := values[key]
	if !ok {
		return []string{}
	}
	switch typed := raw.(type) {
	case []string:
		return cleanKeywords(typed)
	case []any:
		values := make([]string, 0, len(typed))
		for _, item := range typed {
			values = append(values, fmt.Sprint(item))
		}
		return cleanKeywords(values)
	default:
		return splitFormList(fmt.Sprint(raw))
	}
}

func fallbackKnowledgeAssetTitle(filename string) string {
	filename = strings.TrimSpace(filename)
	if filename == "" {
		return "Untitled knowledge asset"
	}
	base := strings.TrimSuffix(filepath.Base(filename), filepath.Ext(filename))
	base = strings.TrimSpace(base)
	if base == "" {
		return filename
	}
	return base
}

func splitFormList(value string) []string {
	if strings.TrimSpace(value) == "" {
		return []string{}
	}
	return cleanKeywords(strings.FieldsFunc(value, func(r rune) bool {
		return r == '\n' || r == '\r' || r == ',' || r == '，' || r == ';' || r == '；' || r == '、'
	}))
}

func stringSliceContains(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

func knowledgeAssetAIStatus(enabled bool) string {
	if enabled {
		return "pending"
	}
	return "disabled"
}

package handler

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"regexp"
	"strings"
	"time"

	"geopress/backend/internal/http/middleware"
	"geopress/backend/internal/model"

	"github.com/gin-gonic/gin"
)

type brandAssetRequest struct {
	Type        string            `json:"type"`
	Name        string            `json:"name"`
	Description string            `json:"description"`
	Content     string            `json:"content"`
	Channels    []string          `json:"channels"`
	Tags        []string          `json:"tags"`
	Source      string            `json:"source"`
	Status      string            `json:"status"`
	Metadata    map[string]string `json:"metadata"`
}

type brandGuardrailRequest struct {
	AssetID    string   `json:"assetId"`
	Name       string   `json:"name"`
	Category   string   `json:"category"`
	Channel    string   `json:"channel"`
	SourceType string   `json:"sourceType"`
	SourceID   string   `json:"sourceId"`
	Severity   string   `json:"severity"`
	Rules      []string `json:"rules"`
	Action     string   `json:"action"`
	Enabled    *bool    `json:"enabled"`
}

type complianceCheckRequest struct {
	ResourceType string `json:"resourceType"`
	ResourceID   string `json:"resourceId"`
	Channel      string `json:"channel"`
	Title        string `json:"title"`
	Content      string `json:"content"`
}

type createApprovalWorkflowRequest struct {
	ResourceType string                `json:"resourceType"`
	ResourceID   string                `json:"resourceId"`
	Name         string                `json:"name"`
	Status       string                `json:"status"`
	Stages       []model.ApprovalStage `json:"stages"`
}

type processApprovalTaskRequest struct {
	Decision string `json:"decision"`
	Comment  string `json:"comment"`
}

type createAgencyClientRelationRequest struct {
	ClientWorkspaceID string   `json:"clientWorkspaceId"`
	ClientName        string   `json:"clientName"`
	Status            string   `json:"status"`
	Scopes            []string `json:"scopes"`
	Notes             string   `json:"notes"`
}

type generateReportPackageRequest struct {
	Name        string   `json:"name"`
	ReportType  string   `json:"reportType"`
	Audience    string   `json:"audience"`
	PeriodStart string   `json:"periodStart"`
	PeriodEnd   string   `json:"periodEnd"`
	Sections    []string `json:"sections"`
}

var compliancePrivateDataPattern = regexp.MustCompile(`(?i)(\b\d{11}\b|\b[A-Z0-9._%+\-]+@[A-Z0-9.\-]+\.[A-Z]{2,}\b|身份证|护照号|银行卡)`)

func (h *WorkspaceHandler) ListBrandAssets(c *gin.Context) {
	workspaceID, ok := h.authorizedWorkspaceID(c)
	if !ok {
		return
	}

	items, err := h.brandAssetsForWorkspace(c.Request.Context(), workspaceID)
	if err != nil {
		log.Printf("brand assets list failed: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "brand assets were not loaded"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": items})
}

func (h *WorkspaceHandler) CreateBrandAsset(c *gin.Context) {
	workspaceID, ok := h.authorizedWorkspaceID(c)
	if !ok {
		return
	}

	var req brandAssetRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	item, ok := h.brandAssetFromRequest(workspaceID, "", req, time.Now().UTC())
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "name is required"})
		return
	}

	if err := h.saveBrandAsset(c.Request.Context(), item); err != nil {
		log.Printf("brand asset was not persisted: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "brand asset was not persisted"})
		return
	}

	h.mu.Lock()
	h.brandAssets = append([]model.BrandAsset{item}, h.brandAssets...)
	h.mu.Unlock()
	c.JSON(http.StatusCreated, item)
}

func (h *WorkspaceHandler) GetBrandAsset(c *gin.Context) {
	workspaceID, ok := h.authorizedWorkspaceID(c)
	if !ok {
		return
	}

	assetID := strings.TrimSpace(c.Param("assetId"))
	if assetID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "assetId is required"})
		return
	}

	item, found, err := h.brandAssetByID(c.Request.Context(), workspaceID, assetID)
	if err != nil {
		log.Printf("brand asset lookup failed: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "brand asset was not loaded"})
		return
	}
	if !found {
		c.JSON(http.StatusNotFound, gin.H{"error": "brand asset not found"})
		return
	}
	c.JSON(http.StatusOK, item)
}

func (h *WorkspaceHandler) UpdateBrandAsset(c *gin.Context) {
	workspaceID, ok := h.authorizedWorkspaceID(c)
	if !ok {
		return
	}

	assetID := strings.TrimSpace(c.Param("assetId"))
	if assetID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "assetId is required"})
		return
	}

	current, found, err := h.brandAssetByID(c.Request.Context(), workspaceID, assetID)
	if err != nil {
		log.Printf("brand asset lookup failed: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "brand asset was not loaded"})
		return
	}
	if !found {
		c.JSON(http.StatusNotFound, gin.H{"error": "brand asset not found"})
		return
	}

	var req brandAssetRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}
	updated, ok := h.brandAssetFromRequest(workspaceID, assetID, req, time.Now().UTC())
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "name is required"})
		return
	}
	updated.CreatedAt = current.CreatedAt

	if err := h.saveBrandAsset(c.Request.Context(), updated); err != nil {
		log.Printf("brand asset update failed: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "brand asset was not persisted"})
		return
	}

	h.mu.Lock()
	h.replaceBrandAssetLocked(updated)
	h.mu.Unlock()
	c.JSON(http.StatusOK, updated)
}

func (h *WorkspaceHandler) ArchiveBrandAsset(c *gin.Context) {
	workspaceID, ok := h.authorizedWorkspaceID(c)
	if !ok {
		return
	}

	assetID := strings.TrimSpace(c.Param("assetId"))
	if assetID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "assetId is required"})
		return
	}

	current, found, err := h.brandAssetByID(c.Request.Context(), workspaceID, assetID)
	if err != nil {
		log.Printf("brand asset lookup failed: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "brand asset was not loaded"})
		return
	}
	if !found {
		c.JSON(http.StatusNotFound, gin.H{"error": "brand asset not found"})
		return
	}

	now := time.Now().UTC()
	if err := h.archiveBrandAsset(c.Request.Context(), workspaceID, assetID, now); err != nil {
		log.Printf("brand asset archive failed: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "brand asset was not archived"})
		return
	}
	current.Status = model.BrandAssetArchived
	current.UpdatedAt = now

	h.mu.Lock()
	h.replaceBrandAssetLocked(current)
	h.mu.Unlock()
	c.JSON(http.StatusOK, current)
}

func (h *WorkspaceHandler) ListBrandGuardrails(c *gin.Context) {
	workspaceID, ok := h.authorizedWorkspaceID(c)
	if !ok {
		return
	}

	items, err := h.brandGuardrailsForWorkspace(c.Request.Context(), workspaceID)
	if err != nil {
		log.Printf("brand guardrails list failed: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "brand guardrails were not loaded"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": items})
}

func (h *WorkspaceHandler) CreateBrandGuardrail(c *gin.Context) {
	workspaceID, ok := h.authorizedWorkspaceID(c)
	if !ok {
		return
	}

	var req brandGuardrailRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	name := strings.TrimSpace(req.Name)
	rules := uniqueStrings(cleanKeywords(req.Rules))
	if name == "" || len(rules) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "name and rules are required"})
		return
	}

	assetID := strings.TrimSpace(req.AssetID)
	if assetID != "" {
		if _, found, err := h.brandAssetByID(c.Request.Context(), workspaceID, assetID); err != nil {
			log.Printf("brand guardrail asset lookup failed: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "brand asset was not loaded"})
			return
		} else if !found {
			c.JSON(http.StatusNotFound, gin.H{"error": "brand asset not found"})
			return
		}
	}

	severity, ok := normalizeRiskSeverity(req.Severity, "medium")
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "severity is invalid"})
		return
	}

	enabled := true
	if req.Enabled != nil {
		enabled = *req.Enabled
	}
	now := time.Now().UTC()
	sourceType := defaultString(strings.TrimSpace(req.SourceType), "manual")
	sourceID := strings.TrimSpace(req.SourceID)
	if assetID != "" && sourceID == "" {
		// 品牌标准来源要能回溯：guardrail 可由品牌资产、法务要求或平台规则派生。
		sourceType = "brand_asset"
		sourceID = assetID
	}
	item := model.BrandGuardrail{
		ID:          fmt.Sprintf("bgd_%d", now.UnixNano()),
		WorkspaceID: workspaceID,
		AssetID:     assetID,
		Name:        name,
		Category:    defaultString(strings.TrimSpace(req.Category), "brand"),
		Channel:     strings.TrimSpace(req.Channel),
		SourceType:  sourceType,
		SourceID:    sourceID,
		Severity:    severity,
		Rules:       rules,
		Action:      defaultString(strings.TrimSpace(req.Action), "review"),
		Enabled:     enabled,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	if err := h.saveBrandGuardrail(c.Request.Context(), item); err != nil {
		log.Printf("brand guardrail was not persisted: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "brand guardrail was not persisted"})
		return
	}

	h.mu.Lock()
	h.brandGuardrails = append([]model.BrandGuardrail{item}, h.brandGuardrails...)
	h.mu.Unlock()
	c.JSON(http.StatusCreated, item)
}

func (h *WorkspaceHandler) ListApprovalWorkflows(c *gin.Context) {
	workspaceID, ok := h.authorizedWorkspaceID(c)
	if !ok {
		return
	}

	items, err := h.approvalWorkflowsForWorkspace(c.Request.Context(), workspaceID)
	if err != nil {
		log.Printf("approval workflows list failed: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "approval workflows were not loaded"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": items})
}

func (h *WorkspaceHandler) CreateApprovalWorkflow(c *gin.Context) {
	workspaceID, ok := h.authorizedWorkspaceID(c)
	if !ok {
		return
	}

	var req createApprovalWorkflowRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	name := strings.TrimSpace(req.Name)
	resourceType := defaultString(strings.TrimSpace(req.ResourceType), "content")
	resourceID := strings.TrimSpace(req.ResourceID)
	stages := cleanApprovalStages(req.Stages)
	if name == "" || len(stages) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "name and stages are required"})
		return
	}

	status := model.ApprovalWorkflowStatus(defaultString(strings.TrimSpace(req.Status), string(model.ApprovalWorkflowActive)))
	if status != model.ApprovalWorkflowDraft && status != model.ApprovalWorkflowActive {
		c.JSON(http.StatusBadRequest, gin.H{"error": "status is invalid"})
		return
	}

	now := time.Now().UTC()
	workflow := model.ApprovalWorkflow{
		ID:              fmt.Sprintf("apw_%d", now.UnixNano()),
		WorkspaceID:     workspaceID,
		ResourceType:    resourceType,
		ResourceID:      resourceID,
		Name:            name,
		Status:          status,
		Stages:          stages,
		CreatedByUserID: middleware.CurrentUserID(c),
		CreatedAt:       now,
		UpdatedAt:       now,
	}
	tasks := approvalTasksFromWorkflow(workflow, now)

	// 审批状态由 workflow 定义生成，第一轮只创建待处理任务，后续复杂串/并签再扩展服务层。
	if err := h.saveApprovalWorkflowWithTasks(c.Request.Context(), workflow, tasks); err != nil {
		log.Printf("approval workflow was not persisted: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "approval workflow was not persisted"})
		return
	}

	h.mu.Lock()
	h.approvalWorkflows = append([]model.ApprovalWorkflow{workflow}, h.approvalWorkflows...)
	h.approvalTasks = append(tasks, h.approvalTasks...)
	h.mu.Unlock()
	c.JSON(http.StatusCreated, gin.H{"workflow": workflow, "tasks": tasks})
}

func (h *WorkspaceHandler) ListApprovalTasks(c *gin.Context) {
	workspaceID, ok := h.authorizedWorkspaceID(c)
	if !ok {
		return
	}

	items, err := h.approvalTasksForWorkspace(c.Request.Context(), workspaceID)
	if err != nil {
		log.Printf("approval tasks list failed: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "approval tasks were not loaded"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": items})
}

func (h *WorkspaceHandler) ProcessApprovalTask(c *gin.Context) {
	workspaceID, ok := h.authorizedWorkspaceID(c)
	if !ok {
		return
	}

	taskID := strings.TrimSpace(c.Param("taskId"))
	if taskID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "taskId is required"})
		return
	}

	var req processApprovalTaskRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	status, ok := approvalStatusForDecision(req.Decision)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "decision is invalid"})
		return
	}

	task, found, err := h.approvalTaskByID(c.Request.Context(), workspaceID, taskID)
	if err != nil {
		log.Printf("approval task lookup failed: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "approval task was not loaded"})
		return
	}
	if !found {
		c.JSON(http.StatusNotFound, gin.H{"error": "approval task not found"})
		return
	}
	if task.Status != model.ApprovalTaskPending {
		c.JSON(http.StatusConflict, gin.H{"error": "approval task is already processed"})
		return
	}

	now := time.Now().UTC()
	task.Status = status
	task.Decision = strings.TrimSpace(req.Decision)
	task.Comment = strings.TrimSpace(req.Comment)
	task.ProcessedByUserID = middleware.CurrentUserID(c)
	task.ProcessedAt = &now
	task.UpdatedAt = now

	// 审批状态只能从 pending 进入终态，避免多人重复处理覆盖法务/品牌决策。
	if err := h.saveApprovalTask(c.Request.Context(), task); err != nil {
		log.Printf("approval task was not persisted: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "approval task was not persisted"})
		return
	}

	h.mu.Lock()
	h.replaceApprovalTaskLocked(task)
	h.mu.Unlock()
	c.JSON(http.StatusOK, task)
}

func (h *WorkspaceHandler) ListComplianceChecks(c *gin.Context) {
	workspaceID, ok := h.authorizedWorkspaceID(c)
	if !ok {
		return
	}

	items, err := h.complianceChecksForWorkspace(c.Request.Context(), workspaceID)
	if err != nil {
		log.Printf("compliance checks list failed: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "compliance checks were not loaded"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": items})
}

func (h *WorkspaceHandler) SubmitComplianceCheck(c *gin.Context) {
	workspaceID, ok := h.authorizedWorkspaceID(c)
	if !ok {
		return
	}

	var req complianceCheckRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	resourceType := defaultString(strings.TrimSpace(req.ResourceType), "content")
	resourceID := strings.TrimSpace(req.ResourceID)
	channel := strings.TrimSpace(req.Channel)
	title := strings.TrimSpace(req.Title)
	body := strings.TrimSpace(req.Content)
	source := ""

	if body == "" && resourceType == "content" && resourceID != "" {
		h.mu.RLock()
		content, found := h.contentByID(workspaceID, resourceID)
		h.mu.RUnlock()
		if !found {
			c.JSON(http.StatusNotFound, gin.H{"error": "content not found"})
			return
		}
		title = content.Title
		body = strings.TrimSpace(strings.Join([]string{content.Title, content.Summary, content.Body}, "\n"))
		source = content.Source
	}
	if body == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "content or supported resourceId is required"})
		return
	}

	guardrails, err := h.brandGuardrailsForWorkspace(c.Request.Context(), workspaceID)
	if err != nil {
		log.Printf("compliance guardrails list failed: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "compliance inputs were not loaded"})
		return
	}

	now := time.Now().UTC()
	checkID := fmt.Sprintf("cck_%d", now.UnixNano())
	findings := buildComplianceFindings(checkID, workspaceID, resourceType, resourceID, channel, title, body, source, guardrails, now)
	riskLevel := riskLevelForFindings(findings)
	summary := "规则化合规检查未发现明确风险。"
	if len(findings) > 0 {
		summary = fmt.Sprintf("规则化合规检查发现 %d 条风险提示，最高风险等级为 %s。", len(findings), riskLevel)
	}
	check := model.ComplianceCheck{
		ID:              checkID,
		WorkspaceID:     workspaceID,
		ResourceType:    resourceType,
		ResourceID:      resourceID,
		Channel:         channel,
		Status:          model.ComplianceCheckCompleted,
		RiskLevel:       riskLevel,
		Summary:         summary,
		Findings:        findings,
		CreatedByUserID: middleware.CurrentUserID(c),
		CreatedAt:       now,
		UpdatedAt:       now,
	}

	// 合规证据只保存命中的短文本、规则来源和建议动作，不调用外部服务也不回传内部 SQL 细节。
	if err := h.saveComplianceCheck(c.Request.Context(), check); err != nil {
		log.Printf("compliance check was not persisted: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "compliance check was not persisted"})
		return
	}

	h.mu.Lock()
	h.complianceChecks = append([]model.ComplianceCheck{check}, h.complianceChecks...)
	h.mu.Unlock()
	c.JSON(http.StatusCreated, check)
}

func (h *WorkspaceHandler) ListAgencyClientRelations(c *gin.Context) {
	workspaceID, ok := h.authorizedWorkspaceID(c)
	if !ok {
		return
	}

	items, err := h.agencyClientRelationsForWorkspace(c.Request.Context(), workspaceID)
	if err != nil {
		log.Printf("agency client relations list failed: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "agency client relations were not loaded"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": items})
}

func (h *WorkspaceHandler) CreateAgencyClientRelation(c *gin.Context) {
	agencyWorkspaceID, ok := h.authorizedWorkspaceID(c)
	if !ok {
		return
	}

	var req createAgencyClientRelationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	clientWorkspaceID := strings.TrimSpace(req.ClientWorkspaceID)
	if clientWorkspaceID == "" || clientWorkspaceID == agencyWorkspaceID {
		c.JSON(http.StatusBadRequest, gin.H{"error": "clientWorkspaceId is invalid"})
		return
	}

	h.mu.RLock()
	clientWorkspace, clientOK := h.workspaceByID(clientWorkspaceID)
	h.mu.RUnlock()
	if !clientOK {
		c.JSON(http.StatusNotFound, gin.H{"error": "client workspace not found"})
		return
	}

	status := defaultString(strings.TrimSpace(req.Status), "active")
	if status != "active" && status != "paused" && status != "ended" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "status is invalid"})
		return
	}

	now := time.Now().UTC()
	item := model.AgencyClientRelation{
		ID:                fmt.Sprintf("acr_%d", now.UnixNano()),
		AgencyWorkspaceID: agencyWorkspaceID,
		ClientWorkspaceID: clientWorkspaceID,
		ClientName:        defaultString(strings.TrimSpace(req.ClientName), clientWorkspace.Name),
		Status:            status,
		Scopes:            uniqueStrings(cleanKeywords(req.Scopes)),
		Notes:             strings.TrimSpace(req.Notes),
		CreatedAt:         now,
		UpdatedAt:         now,
	}
	if len(item.Scopes) == 0 {
		item.Scopes = []string{"reports"}
	}

	// 机构权限边界：这里只记录代理与客户的业务关系，不授予跨工作区登录或数据读取权限。
	if err := h.saveAgencyClientRelation(c.Request.Context(), item); err != nil {
		log.Printf("agency client relation was not persisted: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "agency client relation was not persisted"})
		return
	}

	h.mu.Lock()
	h.agencyClientRelations = append([]model.AgencyClientRelation{item}, h.agencyClientRelations...)
	h.mu.Unlock()
	c.JSON(http.StatusCreated, item)
}

func (h *WorkspaceHandler) ListReportPackages(c *gin.Context) {
	workspaceID, ok := h.authorizedWorkspaceID(c)
	if !ok {
		return
	}

	items, err := h.reportPackagesForWorkspace(c.Request.Context(), workspaceID)
	if err != nil {
		log.Printf("report packages list failed: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "report packages were not loaded"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": items})
}

func (h *WorkspaceHandler) GenerateReportPackage(c *gin.Context) {
	workspaceID, ok := h.authorizedWorkspaceID(c)
	if !ok {
		return
	}

	var req generateReportPackageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	now := time.Now().UTC()
	periodEnd, ok := parseReportDate(req.PeriodEnd, now)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "periodEnd is invalid"})
		return
	}
	periodStart, ok := parseReportDate(req.PeriodStart, periodEnd.AddDate(0, 0, -30))
	if !ok || periodStart.After(periodEnd) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "periodStart is invalid"})
		return
	}

	metrics, err := h.reportMetrics(c.Request.Context(), workspaceID)
	if err != nil {
		log.Printf("report metrics load failed: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "report inputs were not loaded"})
		return
	}

	sections := uniqueStrings(cleanKeywords(req.Sections))
	if len(sections) == 0 {
		sections = []string{"account_growth", "content_delivery", "compliance_risks", "next_cycle_recommendations"}
	}
	item := model.ReportPackage{
		ID:                fmt.Sprintf("rpt_%d", now.UnixNano()),
		WorkspaceID:       workspaceID,
		Name:              defaultString(strings.TrimSpace(req.Name), "经营交付报告"),
		ReportType:        defaultString(strings.TrimSpace(req.ReportType), "monthly"),
		Audience:          defaultString(strings.TrimSpace(req.Audience), "management"),
		PeriodStart:       periodStart,
		PeriodEnd:         periodEnd,
		Status:            "generated",
		Sections:          sections,
		Metrics:           metrics,
		Summary:           "已生成第一轮占位报告包，后续可接入账号矩阵、活动、达人和费用指标。",
		GeneratedByUserID: middleware.CurrentUserID(c),
		CreatedAt:         now,
		UpdatedAt:         now,
	}

	if err := h.saveReportPackage(c.Request.Context(), item); err != nil {
		log.Printf("report package was not persisted: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "report package was not persisted"})
		return
	}

	h.mu.Lock()
	h.reportPackages = append([]model.ReportPackage{item}, h.reportPackages...)
	h.mu.Unlock()
	c.JSON(http.StatusCreated, item)
}

func (h *WorkspaceHandler) ListStrategyRecommendations(c *gin.Context) {
	workspaceID, ok := h.authorizedWorkspaceID(c)
	if !ok {
		return
	}

	items, err := h.strategyRecommendationsForWorkspace(c.Request.Context(), workspaceID)
	if err != nil {
		log.Printf("strategy recommendations list failed: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "strategy recommendations were not loaded"})
		return
	}
	if len(items) == 0 {
		items = h.placeholderStrategyRecommendations(c.Request.Context(), workspaceID)
	}
	c.JSON(http.StatusOK, gin.H{"items": items})
}

func (h *WorkspaceHandler) brandAssetFromRequest(workspaceID string, assetID string, req brandAssetRequest, now time.Time) (model.BrandAsset, bool) {
	name := strings.TrimSpace(req.Name)
	if name == "" {
		return model.BrandAsset{}, false
	}
	status := model.BrandAssetStatus(defaultString(strings.TrimSpace(req.Status), string(model.BrandAssetActive)))
	if status != model.BrandAssetActive && status != model.BrandAssetArchived {
		status = model.BrandAssetActive
	}
	if assetID == "" {
		assetID = fmt.Sprintf("bas_%d", now.UnixNano())
	}
	return model.BrandAsset{
		ID:          assetID,
		WorkspaceID: workspaceID,
		Type:        defaultString(strings.TrimSpace(req.Type), "brand"),
		Name:        name,
		Description: strings.TrimSpace(req.Description),
		Content:     strings.TrimSpace(req.Content),
		Channels:    uniqueStrings(cleanKeywords(req.Channels)),
		Tags:        uniqueStrings(cleanKeywords(req.Tags)),
		Source:      defaultString(strings.TrimSpace(req.Source), "manual"),
		Status:      status,
		Metadata:    cleanStringMap(req.Metadata),
		CreatedAt:   now,
		UpdatedAt:   now,
	}, true
}

func (h *WorkspaceHandler) brandAssetsForWorkspace(ctx context.Context, workspaceID string) ([]model.BrandAsset, error) {
	if h.db != nil && h.db.SQL() != nil {
		dbCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
		defer cancel()
		return h.db.ListBrandAssets(dbCtx, workspaceID)
	}
	h.mu.RLock()
	defer h.mu.RUnlock()
	return filterByWorkspace(h.brandAssets, workspaceID, func(item model.BrandAsset) string { return item.WorkspaceID }), nil
}

func (h *WorkspaceHandler) brandAssetByID(ctx context.Context, workspaceID string, assetID string) (model.BrandAsset, bool, error) {
	if h.db != nil && h.db.SQL() != nil {
		dbCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
		defer cancel()
		return h.db.BrandAssetByID(dbCtx, workspaceID, assetID)
	}
	h.mu.RLock()
	defer h.mu.RUnlock()
	for _, item := range h.brandAssets {
		if item.WorkspaceID == workspaceID && item.ID == assetID {
			return item, true, nil
		}
	}
	return model.BrandAsset{}, false, nil
}

func (h *WorkspaceHandler) saveBrandAsset(ctx context.Context, item model.BrandAsset) error {
	if h.db == nil || h.db.SQL() == nil {
		return nil
	}
	dbCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	return h.db.SaveBrandAsset(dbCtx, item)
}

func (h *WorkspaceHandler) archiveBrandAsset(ctx context.Context, workspaceID string, assetID string, updatedAt time.Time) error {
	if h.db == nil || h.db.SQL() == nil {
		return nil
	}
	dbCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	return h.db.ArchiveBrandAsset(dbCtx, workspaceID, assetID, updatedAt)
}

func (h *WorkspaceHandler) replaceBrandAssetLocked(item model.BrandAsset) {
	for index := range h.brandAssets {
		if h.brandAssets[index].WorkspaceID == item.WorkspaceID && h.brandAssets[index].ID == item.ID {
			h.brandAssets[index] = item
			return
		}
	}
	h.brandAssets = append([]model.BrandAsset{item}, h.brandAssets...)
}

func (h *WorkspaceHandler) brandGuardrailsForWorkspace(ctx context.Context, workspaceID string) ([]model.BrandGuardrail, error) {
	if h.db != nil && h.db.SQL() != nil {
		dbCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
		defer cancel()
		return h.db.ListBrandGuardrails(dbCtx, workspaceID)
	}
	h.mu.RLock()
	defer h.mu.RUnlock()
	return filterByWorkspace(h.brandGuardrails, workspaceID, func(item model.BrandGuardrail) string { return item.WorkspaceID }), nil
}

func (h *WorkspaceHandler) saveBrandGuardrail(ctx context.Context, item model.BrandGuardrail) error {
	if h.db == nil || h.db.SQL() == nil {
		return nil
	}
	dbCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	return h.db.SaveBrandGuardrail(dbCtx, item)
}

func (h *WorkspaceHandler) approvalWorkflowsForWorkspace(ctx context.Context, workspaceID string) ([]model.ApprovalWorkflow, error) {
	if h.db != nil && h.db.SQL() != nil {
		dbCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
		defer cancel()
		return h.db.ListApprovalWorkflows(dbCtx, workspaceID)
	}
	h.mu.RLock()
	defer h.mu.RUnlock()
	return filterByWorkspace(h.approvalWorkflows, workspaceID, func(item model.ApprovalWorkflow) string { return item.WorkspaceID }), nil
}

func (h *WorkspaceHandler) saveApprovalWorkflowWithTasks(ctx context.Context, workflow model.ApprovalWorkflow, tasks []model.ApprovalTask) error {
	if h.db == nil || h.db.SQL() == nil {
		return nil
	}
	dbCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	return h.db.SaveApprovalWorkflowWithTasks(dbCtx, workflow, tasks)
}

func (h *WorkspaceHandler) approvalTasksForWorkspace(ctx context.Context, workspaceID string) ([]model.ApprovalTask, error) {
	if h.db != nil && h.db.SQL() != nil {
		dbCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
		defer cancel()
		return h.db.ListApprovalTasks(dbCtx, workspaceID)
	}
	h.mu.RLock()
	defer h.mu.RUnlock()
	return filterByWorkspace(h.approvalTasks, workspaceID, func(item model.ApprovalTask) string { return item.WorkspaceID }), nil
}

func (h *WorkspaceHandler) approvalTaskByID(ctx context.Context, workspaceID string, taskID string) (model.ApprovalTask, bool, error) {
	if h.db != nil && h.db.SQL() != nil {
		dbCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
		defer cancel()
		return h.db.ApprovalTaskByID(dbCtx, workspaceID, taskID)
	}
	h.mu.RLock()
	defer h.mu.RUnlock()
	for _, item := range h.approvalTasks {
		if item.WorkspaceID == workspaceID && item.ID == taskID {
			return item, true, nil
		}
	}
	return model.ApprovalTask{}, false, nil
}

func (h *WorkspaceHandler) saveApprovalTask(ctx context.Context, item model.ApprovalTask) error {
	if h.db == nil || h.db.SQL() == nil {
		return nil
	}
	dbCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	return h.db.SaveApprovalTask(dbCtx, item)
}

func (h *WorkspaceHandler) replaceApprovalTaskLocked(item model.ApprovalTask) {
	for index := range h.approvalTasks {
		if h.approvalTasks[index].WorkspaceID == item.WorkspaceID && h.approvalTasks[index].ID == item.ID {
			h.approvalTasks[index] = item
			return
		}
	}
	h.approvalTasks = append([]model.ApprovalTask{item}, h.approvalTasks...)
}

func (h *WorkspaceHandler) complianceChecksForWorkspace(ctx context.Context, workspaceID string) ([]model.ComplianceCheck, error) {
	if h.db != nil && h.db.SQL() != nil {
		dbCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
		defer cancel()
		return h.db.ListComplianceChecks(dbCtx, workspaceID)
	}
	h.mu.RLock()
	defer h.mu.RUnlock()
	return filterByWorkspace(h.complianceChecks, workspaceID, func(item model.ComplianceCheck) string { return item.WorkspaceID }), nil
}

func (h *WorkspaceHandler) saveComplianceCheck(ctx context.Context, item model.ComplianceCheck) error {
	if h.db == nil || h.db.SQL() == nil {
		return nil
	}
	dbCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	return h.db.SaveComplianceCheckWithFindings(dbCtx, item)
}

func (h *WorkspaceHandler) agencyClientRelationsForWorkspace(ctx context.Context, workspaceID string) ([]model.AgencyClientRelation, error) {
	if h.db != nil && h.db.SQL() != nil {
		dbCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
		defer cancel()
		return h.db.ListAgencyClientRelations(dbCtx, workspaceID)
	}
	h.mu.RLock()
	defer h.mu.RUnlock()
	items := make([]model.AgencyClientRelation, 0, len(h.agencyClientRelations))
	for _, item := range h.agencyClientRelations {
		if item.AgencyWorkspaceID == workspaceID || item.ClientWorkspaceID == workspaceID {
			items = append(items, item)
		}
	}
	return items, nil
}

func (h *WorkspaceHandler) saveAgencyClientRelation(ctx context.Context, item model.AgencyClientRelation) error {
	if h.db == nil || h.db.SQL() == nil {
		return nil
	}
	dbCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	return h.db.SaveAgencyClientRelation(dbCtx, item)
}

func (h *WorkspaceHandler) reportPackagesForWorkspace(ctx context.Context, workspaceID string) ([]model.ReportPackage, error) {
	if h.db != nil && h.db.SQL() != nil {
		dbCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
		defer cancel()
		return h.db.ListReportPackages(dbCtx, workspaceID)
	}
	h.mu.RLock()
	defer h.mu.RUnlock()
	return filterByWorkspace(h.reportPackages, workspaceID, func(item model.ReportPackage) string { return item.WorkspaceID }), nil
}

func (h *WorkspaceHandler) saveReportPackage(ctx context.Context, item model.ReportPackage) error {
	if h.db == nil || h.db.SQL() == nil {
		return nil
	}
	dbCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	return h.db.SaveReportPackage(dbCtx, item)
}

func (h *WorkspaceHandler) strategyRecommendationsForWorkspace(ctx context.Context, workspaceID string) ([]model.StrategyRecommendation, error) {
	if h.db != nil && h.db.SQL() != nil {
		dbCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
		defer cancel()
		return h.db.ListStrategyRecommendations(dbCtx, workspaceID)
	}
	h.mu.RLock()
	defer h.mu.RUnlock()
	return filterByWorkspace(h.strategyRecommendations, workspaceID, func(item model.StrategyRecommendation) string { return item.WorkspaceID }), nil
}

func (h *WorkspaceHandler) saveStrategyRecommendation(ctx context.Context, item model.StrategyRecommendation) error {
	if h.db == nil || h.db.SQL() == nil {
		return nil
	}
	dbCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	return h.db.SaveStrategyRecommendation(dbCtx, item)
}

func buildComplianceFindings(checkID string, workspaceID string, resourceType string, resourceID string, channel string, title string, body string, source string, guardrails []model.BrandGuardrail, now time.Time) []model.ComplianceFinding {
	text := strings.TrimSpace(strings.Join([]string{title, body}, "\n"))
	lowerText := strings.ToLower(text)
	findings := []model.ComplianceFinding{}
	add := func(severity string, category string, evidence string, finding string, action string, sourceType string, sourceID string) {
		evidence = trimEvidence(evidence)
		if evidence == "" {
			evidence = trimEvidence(text)
		}
		findings = append(findings, model.ComplianceFinding{
			ID:          fmt.Sprintf("cfd_%d_%d", now.UnixNano(), len(findings)+1),
			CheckID:     checkID,
			WorkspaceID: workspaceID,
			Severity:    severity,
			Category:    category,
			Evidence:    evidence,
			Finding:     finding,
			Action:      action,
			SourceType:  sourceType,
			SourceID:    sourceID,
			CreatedAt:   now,
		})
	}

	for _, guardrail := range guardrails {
		if !guardrail.Enabled {
			continue
		}
		if guardrail.Channel != "" && channel != "" && guardrail.Channel != channel {
			continue
		}
		for _, rule := range guardrail.Rules {
			rule = strings.TrimSpace(rule)
			if rule == "" || !strings.Contains(lowerText, strings.ToLower(rule)) {
				continue
			}
			add(
				guardrail.Severity,
				defaultString(guardrail.Category, "brand_guardrail"),
				rule,
				"内容命中品牌 guardrail，需要按品牌标准复核。",
				defaultString(guardrail.Action, "review"),
				"brand_guardrail",
				guardrail.ID,
			)
		}
	}

	if containsAny(lowerText, []string{"100%", "百分百", "保证", "永久", "最强", "第一", "唯一"}) {
		add("medium", "claim_risk", matchedEvidence(text, []string{"100%", "百分百", "保证", "永久", "最强", "第一", "唯一"}), "存在绝对化或确定性承诺表达。", "改为有条件、可验证的描述。", resourceType, resourceID)
	}
	if containsAny(lowerText, []string{"治疗", "治愈", "疗效", "收益保证", "稳赚", "无风险"}) {
		add("high", "sensitive_claim", matchedEvidence(text, []string{"治疗", "治愈", "疗效", "收益保证", "稳赚", "无风险"}), "存在医疗、金融或高敏功效承诺风险。", "提交法务/合规复核并补充依据。", resourceType, resourceID)
	}
	if compliancePrivateDataPattern.MatchString(text) {
		add("high", "private_data", compliancePrivateDataPattern.FindString(text), "疑似包含手机号、邮箱、证件号或其他隐私信息。", "发布前删除或脱敏处理。", resourceType, resourceID)
	}
	if containsAny(lowerText, []string{"合作", "赞助", "推广", "广告"}) && !containsAny(lowerText, []string{"#广告", "广告合作", "商业合作", "利益相关"}) {
		add("medium", "advertising_disclosure", matchedEvidence(text, []string{"合作", "赞助", "推广", "广告"}), "疑似商业推广内容缺少清晰披露。", "补充广告/合作关系披露标签。", resourceType, resourceID)
	}
	if channel == "xiaohongshu" && len([]rune(body)) > 3000 {
		add("low", "platform_constraint", "正文超过 3000 字", "小红书长文可能需要拆分或精简。", "按平台发布格式压缩正文。", "platform_constraint", channel)
	}
	if strings.HasPrefix(source, "ai_") && !containsAny(lowerText, []string{"ai", "人工智能", "智能生成"}) {
		add("low", "ai_labeling", defaultString(title, "AI draft"), "AI 生成内容未见显式标识，部分场景可能需要披露。", "根据品牌和平台政策补充 AI 内容说明。", resourceType, resourceID)
	}
	return findings
}

func cleanApprovalStages(values []model.ApprovalStage) []model.ApprovalStage {
	result := make([]model.ApprovalStage, 0, len(values))
	for _, stage := range values {
		name := strings.TrimSpace(stage.Name)
		role := strings.TrimSpace(stage.ApproverRole)
		if name == "" && role == "" {
			continue
		}
		if name == "" {
			name = role
		}
		if role == "" {
			role = "reviewer"
		}
		required := stage.RequiredApprovals
		if required <= 0 {
			required = 1
		}
		result = append(result, model.ApprovalStage{
			Name:              name,
			ApproverRole:      role,
			RequiredApprovals: required,
		})
	}
	return result
}

func approvalTasksFromWorkflow(workflow model.ApprovalWorkflow, now time.Time) []model.ApprovalTask {
	tasks := make([]model.ApprovalTask, 0, len(workflow.Stages))
	for index, stage := range workflow.Stages {
		tasks = append(tasks, model.ApprovalTask{
			ID:           fmt.Sprintf("apt_%d_%d", now.UnixNano(), index+1),
			WorkspaceID:  workflow.WorkspaceID,
			WorkflowID:   workflow.ID,
			ResourceType: workflow.ResourceType,
			ResourceID:   workflow.ResourceID,
			StageName:    stage.Name,
			AssigneeRole: stage.ApproverRole,
			Status:       model.ApprovalTaskPending,
			CreatedAt:    now,
			UpdatedAt:    now,
		})
	}
	return tasks
}

func (h *WorkspaceHandler) placeholderStrategyRecommendations(ctx context.Context, workspaceID string) []model.StrategyRecommendation {
	now := time.Now().UTC()
	metrics, err := h.reportMetrics(ctx, workspaceID)
	if err != nil {
		log.Printf("strategy placeholder metrics failed: %v", err)
		metrics = map[string]any{}
	}

	items := []model.StrategyRecommendation{
		{
			ID:                 fmt.Sprintf("src_%d_1", now.UnixNano()),
			WorkspaceID:        workspaceID,
			SourceType:         "rule_placeholder",
			RecommendationType: "brand_asset",
			Title:              "补齐品牌资产与禁用表达",
			Rationale:          "品牌资产和 guardrail 是后续生成、审批、合规检查共用的品牌标准来源。",
			Evidence:           []string{fmt.Sprintf("brandAssetCount=%v", metrics["brandAssetCount"]), fmt.Sprintf("complianceCheckCount=%v", metrics["complianceCheckCount"])},
			Action:             "维护 approved_phrase、forbidden_phrase、legal_disclaimer 等品牌资产，并转成 guardrail。",
			Confidence:         0.45,
			Status:             "open",
			CreatedAt:          now,
			UpdatedAt:          now,
		},
		{
			ID:                 fmt.Sprintf("src_%d_2", now.UnixNano()),
			WorkspaceID:        workspaceID,
			SourceType:         "rule_placeholder",
			RecommendationType: "reporting",
			Title:              "建立月度交付报告节奏",
			Rationale:          "报告包把账号、内容、排期、合规和下一轮建议收敛成可交付材料。",
			Evidence:           []string{fmt.Sprintf("contentCount=%v", metrics["contentCount"]), fmt.Sprintf("publishJobCount=%v", metrics["publishJobCount"])},
			Action:             "按月生成 report package，并在客户复盘中补充人工结论。",
			Confidence:         0.4,
			Status:             "open",
			CreatedAt:          now,
			UpdatedAt:          now,
		},
	}

	// 策略推荐第一轮只基于本工作区快照生成低置信度占位，避免误用跨客户指标。
	for _, item := range items {
		if err := h.saveStrategyRecommendation(ctx, item); err != nil {
			log.Printf("strategy recommendation was not persisted: %v", err)
			continue
		}
		h.mu.Lock()
		h.strategyRecommendations = append([]model.StrategyRecommendation{item}, h.strategyRecommendations...)
		h.mu.Unlock()
	}
	return items
}

func (h *WorkspaceHandler) reportMetrics(ctx context.Context, workspaceID string) (map[string]any, error) {
	h.mu.RLock()
	contents := filterByWorkspace(h.contents, workspaceID, func(item model.Content) string { return item.WorkspaceID })
	schedules := filterByWorkspace(h.schedules, workspaceID, func(item model.PublishSchedule) string { return item.WorkspaceID })
	jobs := filterByWorkspace(h.jobs, workspaceID, func(item model.PublishJob) string { return item.WorkspaceID })
	accounts := filterByWorkspace(h.accounts, workspaceID, func(item model.MediaAccount) string { return item.WorkspaceID })
	h.mu.RUnlock()

	brandAssets, err := h.brandAssetsForWorkspace(ctx, workspaceID)
	if err != nil {
		return nil, err
	}
	complianceChecks, err := h.complianceChecksForWorkspace(ctx, workspaceID)
	if err != nil {
		return nil, err
	}
	recommendations, err := h.strategyRecommendationsForWorkspace(ctx, workspaceID)
	if err != nil {
		return nil, err
	}

	return map[string]any{
		"mediaAccountCount":           len(accounts),
		"contentCount":                len(contents),
		"publishedContentCount":       countContents(contents, model.ContentPublished),
		"scheduledContentCount":       countContents(contents, model.ContentScheduled),
		"publishScheduleCount":        len(schedules),
		"publishJobCount":             len(jobs),
		"failedPublishJobCount":       countJobs(jobs, model.PublishJobFailed),
		"brandAssetCount":             len(brandAssets),
		"complianceCheckCount":        len(complianceChecks),
		"strategyRecommendationCount": len(recommendations),
	}, nil
}

func normalizeRiskSeverity(value string, fallback string) (string, bool) {
	value = defaultString(strings.TrimSpace(value), fallback)
	switch value {
	case "info", "low", "medium", "high", "critical":
		return value, true
	default:
		return "", false
	}
}

func approvalStatusForDecision(decision string) (model.ApprovalTaskStatus, bool) {
	switch strings.TrimSpace(strings.ToLower(decision)) {
	case "approve", "approved":
		return model.ApprovalTaskApproved, true
	case "reject", "rejected":
		return model.ApprovalTaskRejected, true
	case "skip", "skipped":
		return model.ApprovalTaskSkipped, true
	case "cancel", "canceled":
		return model.ApprovalTaskCanceled, true
	default:
		return "", false
	}
}

func riskLevelForFindings(findings []model.ComplianceFinding) string {
	level := "none"
	for _, finding := range findings {
		if severityRank(finding.Severity) > severityRank(level) {
			level = finding.Severity
		}
	}
	if level == "info" {
		return "low"
	}
	return level
}

func severityRank(value string) int {
	switch value {
	case "critical":
		return 5
	case "high":
		return 4
	case "medium":
		return 3
	case "low":
		return 2
	case "info":
		return 1
	default:
		return 0
	}
}

func containsAny(value string, needles []string) bool {
	for _, needle := range needles {
		if strings.Contains(value, strings.ToLower(needle)) {
			return true
		}
	}
	return false
}

func matchedEvidence(value string, needles []string) string {
	lower := strings.ToLower(value)
	for _, needle := range needles {
		if strings.Contains(lower, strings.ToLower(needle)) {
			return needle
		}
	}
	return ""
}

func trimEvidence(value string) string {
	value = strings.Join(strings.Fields(strings.TrimSpace(value)), " ")
	if len([]rune(value)) <= 160 {
		return value
	}
	return string([]rune(value)[:160])
}

func parseReportDate(value string, fallback time.Time) (time.Time, bool) {
	value = strings.TrimSpace(value)
	if value == "" {
		return reportDateOnly(fallback), true
	}
	if parsed, err := time.Parse("2006-01-02", value); err == nil {
		return reportDateOnly(parsed), true
	}
	if parsed, err := time.Parse(time.RFC3339, value); err == nil {
		return reportDateOnly(parsed), true
	}
	return time.Time{}, false
}

func reportDateOnly(value time.Time) time.Time {
	year, month, day := value.UTC().Date()
	return time.Date(year, month, day, 0, 0, 0, 0, time.UTC)
}

func cleanStringMap(values map[string]string) map[string]string {
	result := map[string]string{}
	for key, value := range values {
		key = strings.TrimSpace(key)
		value = strings.TrimSpace(value)
		if key != "" && value != "" {
			result[key] = value
		}
	}
	return result
}

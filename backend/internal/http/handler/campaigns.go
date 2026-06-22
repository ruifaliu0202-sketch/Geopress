package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"geopress/backend/internal/model"

	"github.com/gin-gonic/gin"
)

type createCampaignRequest struct {
	Name            string               `json:"name"`
	Description     string               `json:"description"`
	Status          model.CampaignStatus `json:"status"`
	Goal            string               `json:"goal"`
	Products        []string             `json:"products"`
	TargetAudiences []string             `json:"targetAudiences"`
	Channels        []string             `json:"channels"`
	MediaAccountIDs []string             `json:"mediaAccountIds"`
	StartAt         *time.Time           `json:"startAt"`
	EndAt           *time.Time           `json:"endAt"`
	BudgetCents     int                  `json:"budgetCents"`
	Currency        string               `json:"currency"`
	ContentQuota    int                  `json:"contentQuota"`
	ApprovalPolicy  string               `json:"approvalPolicy"`
	SuccessMetrics  []string             `json:"successMetrics"`
	Metadata        map[string]any       `json:"metadata"`
}

type updateCampaignRequest struct {
	Name            *string               `json:"name"`
	Description     *string               `json:"description"`
	Status          *model.CampaignStatus `json:"status"`
	Goal            *string               `json:"goal"`
	Products        []string              `json:"products"`
	TargetAudiences []string              `json:"targetAudiences"`
	Channels        []string              `json:"channels"`
	MediaAccountIDs []string              `json:"mediaAccountIds"`
	StartAt         optionalCampaignTime  `json:"startAt"`
	EndAt           optionalCampaignTime  `json:"endAt"`
	BudgetCents     *int                  `json:"budgetCents"`
	Currency        *string               `json:"currency"`
	ContentQuota    *int                  `json:"contentQuota"`
	ApprovalPolicy  *string               `json:"approvalPolicy"`
	SuccessMetrics  []string              `json:"successMetrics"`
	Metadata        map[string]any        `json:"metadata"`
}

type optionalCampaignTime struct {
	set   bool
	value *time.Time
}

func (value *optionalCampaignTime) UnmarshalJSON(data []byte) error {
	value.set = true
	if strings.TrimSpace(string(data)) == "null" {
		value.value = nil
		return nil
	}
	var parsed time.Time
	if err := json.Unmarshal(data, &parsed); err != nil {
		return err
	}
	utc := parsed.UTC()
	value.value = &utc
	return nil
}

type createCampaignCalendarItemRequest struct {
	TopicID              string                           `json:"topicId"`
	ContentID            string                           `json:"contentId"`
	PublishScheduleID    string                           `json:"publishScheduleId"`
	PublishJobID         string                           `json:"publishJobId"`
	MediaAccountID       string                           `json:"mediaAccountId"`
	AssignedUserID       string                           `json:"assignedUserId"`
	Title                string                           `json:"title"`
	Brief                string                           `json:"brief"`
	ContentType          string                           `json:"contentType"`
	Channel              string                           `json:"channel"`
	PublishWindowStartAt *time.Time                       `json:"publishWindowStartAt"`
	PublishWindowEndAt   *time.Time                       `json:"publishWindowEndAt"`
	Status               model.CampaignCalendarItemStatus `json:"status"`
	DependencyItemIDs    []string                         `json:"dependencyItemIds"`
	ApprovalRequired     bool                             `json:"approvalRequired"`
	ApprovalStatus       string                           `json:"approvalStatus"`
	Metadata             map[string]any                   `json:"metadata"`
}

func (h *WorkspaceHandler) ListCampaigns(c *gin.Context) {
	workspaceID, ok := h.authorizedWorkspaceID(c)
	if !ok {
		return
	}

	h.mu.RLock()
	items := filterByWorkspace(h.campaigns, workspaceID, func(item model.Campaign) string { return item.WorkspaceID })
	h.mu.RUnlock()
	c.JSON(http.StatusOK, gin.H{"items": items})
}

func (h *WorkspaceHandler) CreateCampaign(c *gin.Context) {
	workspaceID, ok := h.authorizedWorkspaceID(c)
	if !ok {
		return
	}

	var req createCampaignRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	name := strings.TrimSpace(req.Name)
	if name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "name is required"})
		return
	}
	status := req.Status
	if status == "" {
		status = model.CampaignDraft
	}
	if !validCampaignStatus(status) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid campaign status"})
		return
	}
	// 新建战役只能从 draft/planned/active 进入，归档和完成态必须来自真实运营过程，避免前端直接制造终态数据。
	if status == model.CampaignPaused || status == model.CampaignCompleted || status == model.CampaignArchived {
		c.JSON(http.StatusConflict, gin.H{"error": "campaign cannot be created in this status"})
		return
	}
	if !validCampaignTimeline(req.StartAt, req.EndAt) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "endAt must be after startAt"})
		return
	}
	if req.BudgetCents < 0 || req.ContentQuota < 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "budgetCents and contentQuota cannot be negative"})
		return
	}
	mediaAccountIDs := uniqueStrings(cleanKeywords(req.MediaAccountIDs))

	h.mu.RLock()
	if !hasMediaAccounts(h.accounts, workspaceID, mediaAccountIDs) {
		h.mu.RUnlock()
		c.JSON(http.StatusNotFound, gin.H{"error": "media account not found"})
		return
	}
	h.mu.RUnlock()

	now := time.Now().UTC()
	campaign := model.Campaign{
		ID:              fmt.Sprintf("cmp_%d", now.UnixNano()),
		WorkspaceID:     workspaceID,
		Name:            name,
		Description:     strings.TrimSpace(req.Description),
		Status:          status,
		Goal:            strings.TrimSpace(req.Goal),
		Products:        uniqueStrings(cleanKeywords(req.Products)),
		TargetAudiences: uniqueStrings(cleanKeywords(req.TargetAudiences)),
		Channels:        uniqueStrings(cleanKeywords(req.Channels)),
		MediaAccountIDs: mediaAccountIDs,
		StartAt:         utcTimePtr(req.StartAt),
		EndAt:           utcTimePtr(req.EndAt),
		BudgetCents:     req.BudgetCents,
		Currency:        defaultString(strings.ToUpper(strings.TrimSpace(req.Currency)), "CNY"),
		ContentQuota:    req.ContentQuota,
		ApprovalPolicy:  defaultString(strings.TrimSpace(req.ApprovalPolicy), "manual"),
		SuccessMetrics:  uniqueStrings(cleanKeywords(req.SuccessMetrics)),
		Metadata:        defaultAnyMap(req.Metadata),
		CreatedAt:       now,
		UpdatedAt:       now,
	}

	if err := h.saveCampaign(c.Request.Context(), campaign); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "campaign was not persisted"})
		return
	}

	h.mu.Lock()
	h.campaigns = append([]model.Campaign{campaign}, h.campaigns...)
	h.mu.Unlock()
	c.JSON(http.StatusCreated, campaign)
}

func (h *WorkspaceHandler) UpdateCampaign(c *gin.Context) {
	workspaceID, ok := h.authorizedWorkspaceID(c)
	if !ok {
		return
	}
	campaignID := strings.TrimSpace(c.Param("campaignId"))
	if campaignID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "campaignId is required"})
		return
	}

	var req updateCampaignRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	h.mu.RLock()
	campaign, found := h.campaignByID(workspaceID, campaignID)
	if !found {
		h.mu.RUnlock()
		c.JSON(http.StatusNotFound, gin.H{"error": "campaign not found"})
		return
	}
	if req.Status != nil && !canTransitionCampaignStatus(campaign.Status, *req.Status) {
		h.mu.RUnlock()
		c.JSON(http.StatusConflict, gin.H{"error": "invalid campaign status transition"})
		return
	}
	mediaAccountIDs := campaign.MediaAccountIDs
	if req.MediaAccountIDs != nil {
		mediaAccountIDs = uniqueStrings(cleanKeywords(req.MediaAccountIDs))
	}
	if !hasMediaAccounts(h.accounts, workspaceID, mediaAccountIDs) {
		h.mu.RUnlock()
		c.JSON(http.StatusNotFound, gin.H{"error": "media account not found"})
		return
	}
	h.mu.RUnlock()

	if req.Name != nil {
		name := strings.TrimSpace(*req.Name)
		if name == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "name is required"})
			return
		}
		campaign.Name = name
	}
	if req.Description != nil {
		campaign.Description = strings.TrimSpace(*req.Description)
	}
	if req.Status != nil {
		// 状态机由后端兜底，确保归档/完成后的战役不能被普通更新重新投入执行。
		campaign.Status = *req.Status
	}
	if req.Goal != nil {
		campaign.Goal = strings.TrimSpace(*req.Goal)
	}
	if req.Products != nil {
		campaign.Products = uniqueStrings(cleanKeywords(req.Products))
	}
	if req.TargetAudiences != nil {
		campaign.TargetAudiences = uniqueStrings(cleanKeywords(req.TargetAudiences))
	}
	if req.Channels != nil {
		campaign.Channels = uniqueStrings(cleanKeywords(req.Channels))
	}
	campaign.MediaAccountIDs = mediaAccountIDs
	if req.StartAt.set {
		campaign.StartAt = req.StartAt.value
	}
	if req.EndAt.set {
		campaign.EndAt = req.EndAt.value
	}
	if !validCampaignTimeline(campaign.StartAt, campaign.EndAt) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "endAt must be after startAt"})
		return
	}
	if req.BudgetCents != nil {
		if *req.BudgetCents < 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "budgetCents cannot be negative"})
			return
		}
		campaign.BudgetCents = *req.BudgetCents
	}
	if req.Currency != nil {
		campaign.Currency = defaultString(strings.ToUpper(strings.TrimSpace(*req.Currency)), "CNY")
	}
	if req.ContentQuota != nil {
		if *req.ContentQuota < 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "contentQuota cannot be negative"})
			return
		}
		campaign.ContentQuota = *req.ContentQuota
	}
	if req.ApprovalPolicy != nil {
		campaign.ApprovalPolicy = defaultString(strings.TrimSpace(*req.ApprovalPolicy), "manual")
	}
	if req.SuccessMetrics != nil {
		campaign.SuccessMetrics = uniqueStrings(cleanKeywords(req.SuccessMetrics))
	}
	if req.Metadata != nil {
		campaign.Metadata = defaultAnyMap(req.Metadata)
	}
	campaign.UpdatedAt = time.Now().UTC()

	if err := h.saveCampaign(c.Request.Context(), campaign); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "campaign was not persisted"})
		return
	}

	h.mu.Lock()
	for index := range h.campaigns {
		if h.campaigns[index].WorkspaceID == workspaceID && h.campaigns[index].ID == campaignID {
			h.campaigns[index] = campaign
			break
		}
	}
	h.mu.Unlock()
	c.JSON(http.StatusOK, campaign)
}

func (h *WorkspaceHandler) ListCampaignCalendarItems(c *gin.Context) {
	workspaceID, ok := h.authorizedWorkspaceID(c)
	if !ok {
		return
	}
	campaignID := strings.TrimSpace(c.Param("campaignId"))

	h.mu.RLock()
	if _, found := h.campaignByID(workspaceID, campaignID); !found {
		h.mu.RUnlock()
		c.JSON(http.StatusNotFound, gin.H{"error": "campaign not found"})
		return
	}
	items := filterCampaignCalendarItems(h.campaignCalendarItems, workspaceID, campaignID)
	h.mu.RUnlock()
	c.JSON(http.StatusOK, gin.H{"items": items})
}

func (h *WorkspaceHandler) CreateCampaignCalendarItem(c *gin.Context) {
	workspaceID, ok := h.authorizedWorkspaceID(c)
	if !ok {
		return
	}
	campaignID := strings.TrimSpace(c.Param("campaignId"))

	var req createCampaignCalendarItemRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}
	title := strings.TrimSpace(req.Title)
	if title == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "title is required"})
		return
	}
	if !validCampaignCalendarWindow(req.PublishWindowStartAt, req.PublishWindowEndAt) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "publishWindowEndAt must be after publishWindowStartAt"})
		return
	}
	status := req.Status
	if status == "" {
		status = model.CampaignCalendarPlanned
	}
	if !validCampaignCalendarItemStatus(status) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid campaign calendar item status"})
		return
	}
	// 发布态必须来自真实发布结果或人工确认，不能由计划日历创建时直接伪造。
	if status == model.CampaignCalendarPublished {
		c.JSON(http.StatusConflict, gin.H{"error": "calendar item cannot be created as published"})
		return
	}

	topicID := strings.TrimSpace(req.TopicID)
	contentID := strings.TrimSpace(req.ContentID)
	scheduleID := strings.TrimSpace(req.PublishScheduleID)
	jobID := strings.TrimSpace(req.PublishJobID)
	accountID := strings.TrimSpace(req.MediaAccountID)

	h.mu.RLock()
	if _, found := h.campaignByID(workspaceID, campaignID); !found {
		h.mu.RUnlock()
		c.JSON(http.StatusNotFound, gin.H{"error": "campaign not found"})
		return
	}
	if topicID != "" && !h.campaignTopicExists(workspaceID, campaignID, topicID) {
		h.mu.RUnlock()
		c.JSON(http.StatusNotFound, gin.H{"error": "campaign topic not found"})
		return
	}
	if contentID != "" && !hasContent(h.contents, workspaceID, contentID) {
		h.mu.RUnlock()
		c.JSON(http.StatusNotFound, gin.H{"error": "content not found"})
		return
	}
	if scheduleID != "" && !h.publishScheduleExists(workspaceID, scheduleID) {
		h.mu.RUnlock()
		c.JSON(http.StatusNotFound, gin.H{"error": "publish schedule not found"})
		return
	}
	if jobID != "" && !h.publishJobExists(workspaceID, jobID) {
		h.mu.RUnlock()
		c.JSON(http.StatusNotFound, gin.H{"error": "publish job not found"})
		return
	}
	if accountID != "" && !hasMediaAccount(h.accounts, workspaceID, accountID) {
		h.mu.RUnlock()
		c.JSON(http.StatusNotFound, gin.H{"error": "media account not found"})
		return
	}
	assignedUserID := strings.TrimSpace(req.AssignedUserID)
	if assignedUserID != "" && !workspaceMemberExists(h.members, workspaceID, assignedUserID) {
		h.mu.RUnlock()
		c.JSON(http.StatusNotFound, gin.H{"error": "assigned user not found"})
		return
	}
	dependencyItemIDs := uniqueStrings(cleanKeywords(req.DependencyItemIDs))
	if !campaignCalendarDependenciesExist(h.campaignCalendarItems, workspaceID, campaignID, dependencyItemIDs) {
		h.mu.RUnlock()
		c.JSON(http.StatusNotFound, gin.H{"error": "dependency calendar item not found"})
		return
	}
	h.mu.RUnlock()

	now := time.Now().UTC()
	item := model.CampaignCalendarItem{
		ID:                   fmt.Sprintf("cci_%d", now.UnixNano()),
		WorkspaceID:          workspaceID,
		CampaignID:           campaignID,
		TopicID:              topicID,
		ContentID:            contentID,
		PublishScheduleID:    scheduleID,
		PublishJobID:         jobID,
		MediaAccountID:       accountID,
		AssignedUserID:       assignedUserID,
		Title:                title,
		Brief:                strings.TrimSpace(req.Brief),
		ContentType:          defaultString(strings.TrimSpace(req.ContentType), "article"),
		Channel:              strings.TrimSpace(req.Channel),
		PublishWindowStartAt: utcTimePtr(req.PublishWindowStartAt),
		PublishWindowEndAt:   utcTimePtr(req.PublishWindowEndAt),
		Status:               status,
		DependencyItemIDs:    dependencyItemIDs,
		ApprovalRequired:     req.ApprovalRequired,
		ApprovalStatus:       defaultApprovalStatus(req.ApprovalRequired, req.ApprovalStatus),
		Metadata:             defaultAnyMap(req.Metadata),
		CreatedAt:            now,
		UpdatedAt:            now,
	}

	// 日历项只保存与内容/排程/任务的 nullable 引用，不在这里创建发布计划；这样 Campaign 计划可以先沉淀策略，再由后续编排流程生成具体发布任务。
	if err := h.saveCampaignCalendarItem(c.Request.Context(), item); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "campaign calendar item was not persisted"})
		return
	}

	h.mu.Lock()
	h.campaignCalendarItems = append(h.campaignCalendarItems, item)
	h.mu.Unlock()
	c.JSON(http.StatusCreated, item)
}

func (h *WorkspaceHandler) CampaignReportSummary(c *gin.Context) {
	workspaceID, ok := h.authorizedWorkspaceID(c)
	if !ok {
		return
	}
	campaignID := strings.TrimSpace(c.Param("campaignId"))

	h.mu.RLock()
	campaign, found := h.campaignByID(workspaceID, campaignID)
	if !found {
		h.mu.RUnlock()
		c.JSON(http.StatusNotFound, gin.H{"error": "campaign not found"})
		return
	}
	items := filterCampaignCalendarItems(h.campaignCalendarItems, workspaceID, campaignID)
	metrics := filterCampaignMetrics(h.campaignMetrics, workspaceID, campaignID)
	rollups := filterCampaignRollups(h.campaignRollups, workspaceID, campaignID)
	schedules := filterByWorkspace(h.schedules, workspaceID, func(item model.PublishSchedule) string { return item.WorkspaceID })
	jobs := filterByWorkspace(h.jobs, workspaceID, func(item model.PublishJob) string { return item.WorkspaceID })
	h.mu.RUnlock()

	// 第一轮报表优先做可解释的基础聚合；未来指标采集接入后，metric/rollup 表会补齐更细的趋势与推荐依据。
	summary := buildCampaignReportSummary(campaign, items, metrics, rollups, schedules, jobs)
	c.JSON(http.StatusOK, summary)
}

func (h *WorkspaceHandler) campaignByID(workspaceID string, campaignID string) (model.Campaign, bool) {
	for _, campaign := range h.campaigns {
		if campaign.WorkspaceID == workspaceID && campaign.ID == campaignID {
			return campaign, true
		}
	}
	return model.Campaign{}, false
}

func (h *WorkspaceHandler) campaignTopicExists(workspaceID, campaignID, topicID string) bool {
	for _, topic := range h.campaignTopics {
		if topic.WorkspaceID == workspaceID && topic.CampaignID == campaignID && topic.ID == topicID {
			return true
		}
	}
	return false
}

func (h *WorkspaceHandler) publishScheduleExists(workspaceID, scheduleID string) bool {
	for _, schedule := range h.schedules {
		if schedule.WorkspaceID == workspaceID && schedule.ID == scheduleID {
			return true
		}
	}
	return false
}

func (h *WorkspaceHandler) publishJobExists(workspaceID, jobID string) bool {
	for _, job := range h.jobs {
		if job.WorkspaceID == workspaceID && job.ID == jobID {
			return true
		}
	}
	return false
}

func workspaceMemberExists(members []model.WorkspaceMember, workspaceID, userID string) bool {
	for _, member := range members {
		if member.WorkspaceID == workspaceID && member.UserID == userID {
			return true
		}
	}
	return false
}

func validCampaignStatus(status model.CampaignStatus) bool {
	switch status {
	case model.CampaignDraft, model.CampaignPlanned, model.CampaignActive, model.CampaignPaused, model.CampaignCompleted, model.CampaignArchived:
		return true
	default:
		return false
	}
}

func canTransitionCampaignStatus(from, to model.CampaignStatus) bool {
	if from == to {
		return true
	}
	if !validCampaignStatus(to) {
		return false
	}
	// 战役状态代表运营承诺：归档态不可恢复，完成态只能归档，避免历史报表口径被后续更新改写。
	allowed := map[model.CampaignStatus][]model.CampaignStatus{
		model.CampaignDraft:     {model.CampaignPlanned, model.CampaignArchived},
		model.CampaignPlanned:   {model.CampaignDraft, model.CampaignActive, model.CampaignPaused, model.CampaignArchived},
		model.CampaignActive:    {model.CampaignPaused, model.CampaignCompleted, model.CampaignArchived},
		model.CampaignPaused:    {model.CampaignActive, model.CampaignCompleted, model.CampaignArchived},
		model.CampaignCompleted: {model.CampaignArchived},
		model.CampaignArchived:  {},
	}
	for _, next := range allowed[from] {
		if next == to {
			return true
		}
	}
	return false
}

func validCampaignCalendarItemStatus(status model.CampaignCalendarItemStatus) bool {
	switch status {
	case model.CampaignCalendarPlanned, model.CampaignCalendarDrafting, model.CampaignCalendarReview, model.CampaignCalendarScheduled, model.CampaignCalendarPublished, model.CampaignCalendarSkipped, model.CampaignCalendarCanceled:
		return true
	default:
		return false
	}
}

func validCampaignTimeline(startAt, endAt *time.Time) bool {
	if startAt == nil || endAt == nil {
		return true
	}
	return !endAt.Before(*startAt)
}

func validCampaignCalendarWindow(startAt, endAt *time.Time) bool {
	if startAt == nil || endAt == nil {
		return true
	}
	return !endAt.Before(*startAt)
}

func utcTimePtr(value *time.Time) *time.Time {
	if value == nil {
		return nil
	}
	utc := value.UTC()
	return &utc
}

func defaultAnyMap(value map[string]any) map[string]any {
	if value == nil {
		return map[string]any{}
	}
	return value
}

func hasMediaAccounts(accounts []model.MediaAccount, workspaceID string, accountIDs []string) bool {
	for _, accountID := range accountIDs {
		if !hasMediaAccount(accounts, workspaceID, accountID) {
			return false
		}
	}
	return true
}

func campaignCalendarDependenciesExist(items []model.CampaignCalendarItem, workspaceID, campaignID string, dependencyItemIDs []string) bool {
	for _, dependencyItemID := range dependencyItemIDs {
		found := false
		for _, item := range items {
			if item.WorkspaceID == workspaceID && item.CampaignID == campaignID && item.ID == dependencyItemID {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	return true
}

func defaultApprovalStatus(required bool, value string) string {
	value = strings.TrimSpace(value)
	if value != "" {
		return value
	}
	if required {
		return "pending"
	}
	return "not_required"
}

func filterCampaignCalendarItems(items []model.CampaignCalendarItem, workspaceID string, campaignID string) []model.CampaignCalendarItem {
	filtered := make([]model.CampaignCalendarItem, 0, len(items))
	for _, item := range items {
		if item.WorkspaceID == workspaceID && item.CampaignID == campaignID {
			filtered = append(filtered, item)
		}
	}
	return filtered
}

func filterCampaignMetrics(items []model.CampaignMetric, workspaceID string, campaignID string) []model.CampaignMetric {
	filtered := make([]model.CampaignMetric, 0, len(items))
	for _, item := range items {
		if item.WorkspaceID == workspaceID && item.CampaignID == campaignID {
			filtered = append(filtered, item)
		}
	}
	return filtered
}

func filterCampaignRollups(items []model.CampaignRollup, workspaceID string, campaignID string) []model.CampaignRollup {
	filtered := make([]model.CampaignRollup, 0, len(items))
	for _, item := range items {
		if item.WorkspaceID == workspaceID && item.CampaignID == campaignID {
			filtered = append(filtered, item)
		}
	}
	return filtered
}

func buildCampaignReportSummary(
	campaign model.Campaign,
	calendarItems []model.CampaignCalendarItem,
	metrics []model.CampaignMetric,
	rollups []model.CampaignRollup,
	schedules []model.PublishSchedule,
	jobs []model.PublishJob,
) model.CampaignReportSummary {
	statusCounts := map[string]int{}
	metricTotals := map[string]float64{}
	contentIDs := map[string]bool{}
	jobIDs := map[string]bool{}
	scheduleIDs := map[string]bool{}
	jobStatusByID := map[string]model.PublishJobStatus{}
	scheduledItemCount := 0
	publishedItemCount := 0
	failedItemCount := 0

	for _, job := range jobs {
		jobStatusByID[job.ID] = job.Status
	}
	for _, item := range calendarItems {
		status := string(item.Status)
		statusCounts[status]++
		if item.ContentID != "" {
			contentIDs[item.ContentID] = true
		}
		if item.PublishJobID != "" {
			jobIDs[item.PublishJobID] = true
		}
		if item.PublishScheduleID != "" {
			scheduleIDs[item.PublishScheduleID] = true
		}
		switch item.Status {
		case model.CampaignCalendarScheduled:
			scheduledItemCount++
		case model.CampaignCalendarPublished:
			publishedItemCount++
		case model.CampaignCalendarSkipped, model.CampaignCalendarCanceled:
			failedItemCount++
		default:
			switch jobStatusByID[item.PublishJobID] {
			case model.PublishJobQueued, model.PublishJobRunning, model.PublishJobManual, model.PublishJobRetrying:
				scheduledItemCount++
			case model.PublishJobSucceeded:
				publishedItemCount++
			case model.PublishJobFailed:
				failedItemCount++
			}
		}
	}
	// Campaign 只引用既有发布计划/任务，不反向改旧发布表；报表在读取侧把这些 nullable 链接聚合起来。
	for _, schedule := range schedules {
		if scheduleIDs[schedule.ID] {
			if schedule.ContentID != "" {
				contentIDs[schedule.ContentID] = true
			}
			continue
		}
		if schedule.ContentID != "" && contentIDs[schedule.ContentID] {
			scheduleIDs[schedule.ID] = true
		}
	}
	for _, metric := range metrics {
		metricTotals[metric.MetricName] += metric.MetricValue
	}
	for _, job := range jobs {
		if jobIDs[job.ID] {
			continue
		}
		if job.ScheduleID != "" && scheduleIDs[job.ScheduleID] {
			jobIDs[job.ID] = true
			continue
		}
		if job.ContentID != "" && contentIDs[job.ContentID] {
			jobIDs[job.ID] = true
		}
	}

	recommendations := []model.CampaignRecommendation{}
	if len(calendarItems) == 0 {
		recommendations = append(recommendations, model.CampaignRecommendation{
			Type:   "planning",
			Title:  "补齐内容日历",
			Reason: "当前战役尚未创建日历项，后续无法做选题、发布窗口和账号投放复盘。",
			Metadata: map[string]any{
				"campaignStatus": campaign.Status,
			},
		})
	}
	if campaign.ContentQuota > 0 && len(calendarItems) < campaign.ContentQuota {
		recommendations = append(recommendations, model.CampaignRecommendation{
			Type:   "quota",
			Title:  "增加计划内容量",
			Reason: "已计划内容少于战役内容配额，建议补充选题或降低配额。",
			Metadata: map[string]any{
				"contentQuota":      campaign.ContentQuota,
				"calendarItemCount": len(calendarItems),
			},
		})
	}

	return model.CampaignReportSummary{
		WorkspaceID:         campaign.WorkspaceID,
		CampaignID:          campaign.ID,
		Status:              campaign.Status,
		CalendarItemCount:   len(calendarItems),
		ContentCount:        len(contentIDs),
		PublishJobCount:     len(jobIDs),
		PlannedItemCount:    statusCounts[string(model.CampaignCalendarPlanned)],
		ScheduledItemCount:  scheduledItemCount,
		PublishedItemCount:  publishedItemCount,
		FailedItemCount:     failedItemCount,
		StatusCounts:        statusCounts,
		MetricTotals:        metricTotals,
		Metrics:             metrics,
		Rollups:             rollups,
		Recommendations:     recommendations,
		ReportingWindowFrom: campaign.StartAt,
		ReportingWindowTo:   campaign.EndAt,
		UpdatedAt:           time.Now().UTC(),
	}
}

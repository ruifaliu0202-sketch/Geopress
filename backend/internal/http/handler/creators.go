package handler

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"

	"geopress/backend/internal/http/middleware"
	"geopress/backend/internal/model"

	"github.com/gin-gonic/gin"
)

type creatorDetailResponse struct {
	Creator       model.Creator               `json:"creator"`
	MediaAccounts []model.CreatorMediaAccount `json:"mediaAccounts"`
	Shortlists    []model.CreatorShortlist    `json:"shortlists"`
}

type createCreatorShortlistRequest struct {
	CreatorID           string `json:"creatorId"`
	Name                string `json:"name"`
	FitScore            int    `json:"fitScore"`
	QualificationStatus string `json:"qualificationStatus"`
	BrandSafetyLevel    string `json:"brandSafetyLevel"`
	BrandSafetyNotes    string `json:"brandSafetyNotes"`
	OperatorNotes       string `json:"operatorNotes"`
}

type createCreatorCampaignBriefRequest struct {
	Title                   string   `json:"title"`
	Objective               string   `json:"objective"`
	ProductName             string   `json:"productName"`
	TargetAudience          string   `json:"targetAudience"`
	PlatformTargets         []string `json:"platformTargets"`
	DeliverableRequirements []string `json:"deliverableRequirements"`
	DisclosureRequirements  []string `json:"disclosureRequirements"`
	ProhibitedClaims        []string `json:"prohibitedClaims"`
	AuthorizationScope      string   `json:"authorizationScope"`
	ContentUsageRights      string   `json:"contentUsageRights"`
	ReviewWindowHours       int      `json:"reviewWindowHours"`
	DeadlineAt              string   `json:"deadlineAt"`
	BudgetCents             int      `json:"budgetCents"`
	Currency                string   `json:"currency"`
	Status                  string   `json:"status"`
}

type createCreatorOrderRequest struct {
	BriefID                 string   `json:"briefId"`
	CreatorID               string   `json:"creatorId"`
	PriceCents              int      `json:"priceCents"`
	DepositCents            int      `json:"depositCents"`
	ServiceFeeCents         int      `json:"serviceFeeCents"`
	Currency                string   `json:"currency"`
	DisclosureRequirements  []string `json:"disclosureRequirements"`
	DeliverableRequirements []string `json:"deliverableRequirements"`
	AuthorizationScope      string   `json:"authorizationScope"`
	ContentUsageRights      string   `json:"contentUsageRights"`
	DueAt                   string   `json:"dueAt"`
	LastMessage             string   `json:"lastMessage"`
}

type creatorOrderResponse struct {
	Order      model.CreatorOrder      `json:"order"`
	Settlement model.CreatorSettlement `json:"settlement"`
}

type submitCreatorDeliverableRequest struct {
	Type      string   `json:"type"`
	Title     string   `json:"title"`
	Content   string   `json:"content"`
	AssetURLs []string `json:"assetUrls"`
}

type reviewCreatorDeliverableRequest struct {
	Decision string `json:"decision"`
	Feedback string `json:"feedback"`
}

type recordCreatorPublicationProofRequest struct {
	ExternalURL          string `json:"externalUrl"`
	PublicationProofURL  string `json:"publicationProofUrl"`
	PublicationProofNote string `json:"publicationProofNote"`
	DisclosureText       string `json:"disclosureText"`
	Notes                string `json:"notes"`
	PublishedAt          string `json:"publishedAt"`
}

type creatorPublicationProofResponse struct {
	Deliverable model.CreatorDeliverable        `json:"deliverable"`
	Order       model.CreatorOrder              `json:"order"`
	Settlement  model.CreatorSettlement         `json:"settlement"`
	Evidence    model.CreatorComplianceEvidence `json:"evidence"`
}

func (h *WorkspaceHandler) ListCreators(c *gin.Context) {
	workspaceID, ok := h.authorizedWorkspaceID(c)
	if !ok {
		return
	}

	if !h.loadCreatorSnapshot(c.Request.Context(), workspaceID) {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "creators were not loaded"})
		return
	}

	h.mu.RLock()
	items := append([]model.Creator(nil), h.creators...)
	sort.SliceStable(items, func(i, j int) bool {
		if items[i].VerificationState != items[j].VerificationState {
			return items[i].VerificationState == model.CreatorVerificationVerified
		}
		return items[i].UpdatedAt.After(items[j].UpdatedAt)
	})
	h.mu.RUnlock()

	if items == nil {
		items = []model.Creator{}
	}
	c.JSON(http.StatusOK, gin.H{"items": items})
}

func (h *WorkspaceHandler) GetCreator(c *gin.Context) {
	workspaceID, ok := h.authorizedWorkspaceID(c)
	if !ok {
		return
	}
	creatorID := strings.TrimSpace(c.Param("creatorId"))
	if creatorID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "creatorId is required"})
		return
	}

	if !h.loadCreatorSnapshot(c.Request.Context(), workspaceID) {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "creator was not loaded"})
		return
	}

	h.mu.RLock()
	creator, creatorOK := h.creatorByIDLocked(creatorID)
	// 达人媒体账号是公开合作资料和授权边界，不向租户暴露登录凭据或代登能力。
	accounts := h.creatorMediaAccountsByCreatorLocked(creatorID)
	shortlists := filterByWorkspace(h.creatorShortlists, workspaceID, func(item model.CreatorShortlist) string { return item.WorkspaceID })
	shortlists = filterCreatorShortlists(shortlists, creatorID)
	h.mu.RUnlock()

	if !creatorOK {
		c.JSON(http.StatusNotFound, gin.H{"error": "creator not found"})
		return
	}
	if accounts == nil {
		accounts = []model.CreatorMediaAccount{}
	}
	if shortlists == nil {
		shortlists = []model.CreatorShortlist{}
	}
	c.JSON(http.StatusOK, creatorDetailResponse{
		Creator:       creator,
		MediaAccounts: accounts,
		Shortlists:    shortlists,
	})
}

func (h *WorkspaceHandler) ListCreatorShortlists(c *gin.Context) {
	workspaceID, ok := h.authorizedWorkspaceID(c)
	if !ok {
		return
	}
	if !h.loadCreatorSnapshot(c.Request.Context(), workspaceID) {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "creator shortlists were not loaded"})
		return
	}

	h.mu.RLock()
	items := filterByWorkspace(h.creatorShortlists, workspaceID, func(item model.CreatorShortlist) string { return item.WorkspaceID })
	h.mu.RUnlock()
	if items == nil {
		items = []model.CreatorShortlist{}
	}
	c.JSON(http.StatusOK, gin.H{"items": items})
}

func (h *WorkspaceHandler) CreateCreatorShortlist(c *gin.Context) {
	workspaceID, ok := h.authorizedWorkspaceID(c)
	if !ok {
		return
	}
	var req createCreatorShortlistRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	creatorID := strings.TrimSpace(req.CreatorID)
	if creatorID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "creatorId is required"})
		return
	}
	if req.FitScore < 0 || req.FitScore > 100 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "fitScore must be between 0 and 100"})
		return
	}
	qualificationStatus := defaultString(strings.TrimSpace(req.QualificationStatus), "watching")
	if !validCreatorQualificationStatus(qualificationStatus) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "unsupported qualificationStatus"})
		return
	}
	if !h.loadCreatorSnapshot(c.Request.Context(), workspaceID) {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "creator shortlist was not loaded"})
		return
	}

	h.mu.RLock()
	_, creatorOK := h.creatorByIDLocked(creatorID)
	h.mu.RUnlock()
	if !creatorOK {
		c.JSON(http.StatusNotFound, gin.H{"error": "creator not found"})
		return
	}

	now := time.Now().UTC()
	item := model.CreatorShortlist{
		ID:                  fmt.Sprintf("csl_%d", now.UnixNano()),
		WorkspaceID:         workspaceID,
		CreatorID:           creatorID,
		Name:                defaultString(strings.TrimSpace(req.Name), "default"),
		FitScore:            req.FitScore,
		QualificationStatus: qualificationStatus,
		BrandSafetyLevel:    defaultString(strings.TrimSpace(req.BrandSafetyLevel), "unknown"),
		BrandSafetyNotes:    strings.TrimSpace(req.BrandSafetyNotes),
		OperatorNotes:       strings.TrimSpace(req.OperatorNotes),
		CreatedByUserID:     middleware.CurrentUserID(c),
		CreatedAt:           now,
		UpdatedAt:           now,
	}

	// 达人 shortlist 是品牌侧的选型记录，不代表租户获得达人账号登录权。
	if err := h.saveCreatorShortlist(c.Request.Context(), item); err != nil {
		log.Printf("creator shortlist was not persisted: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "creator shortlist was not persisted"})
		return
	}

	h.mu.Lock()
	h.upsertCreatorShortlistLocked(item)
	h.mu.Unlock()
	c.JSON(http.StatusCreated, item)
}

func (h *WorkspaceHandler) ListCreatorCampaignBriefs(c *gin.Context) {
	workspaceID, ok := h.authorizedWorkspaceID(c)
	if !ok {
		return
	}
	if !h.loadCreatorSnapshot(c.Request.Context(), workspaceID) {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "creator briefs were not loaded"})
		return
	}
	h.mu.RLock()
	items := filterByWorkspace(h.creatorBriefs, workspaceID, func(item model.CreatorCampaignBrief) string { return item.WorkspaceID })
	h.mu.RUnlock()
	if items == nil {
		items = []model.CreatorCampaignBrief{}
	}
	c.JSON(http.StatusOK, gin.H{"items": items})
}

func (h *WorkspaceHandler) CreateCreatorCampaignBrief(c *gin.Context) {
	workspaceID, ok := h.authorizedWorkspaceID(c)
	if !ok {
		return
	}
	var req createCreatorCampaignBriefRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}
	title := strings.TrimSpace(req.Title)
	if title == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "title is required"})
		return
	}
	if req.BudgetCents < 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "budgetCents must be non-negative"})
		return
	}
	deadlineAt, validTime := parseOptionalTime(req.DeadlineAt)
	if !validTime {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid deadlineAt"})
		return
	}
	status := model.CreatorCampaignBriefStatus(defaultString(strings.TrimSpace(req.Status), string(model.CreatorCampaignBriefDraft)))
	if !validCreatorCampaignBriefStatus(status) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "unsupported brief status"})
		return
	}

	now := time.Now().UTC()
	item := model.CreatorCampaignBrief{
		ID:                      fmt.Sprintf("cbr_%d", now.UnixNano()),
		WorkspaceID:             workspaceID,
		Title:                   title,
		Objective:               strings.TrimSpace(req.Objective),
		ProductName:             strings.TrimSpace(req.ProductName),
		TargetAudience:          strings.TrimSpace(req.TargetAudience),
		PlatformTargets:         cleanKeywords(req.PlatformTargets),
		DeliverableRequirements: cleanKeywords(req.DeliverableRequirements),
		DisclosureRequirements:  cleanKeywords(req.DisclosureRequirements),
		ProhibitedClaims:        cleanKeywords(req.ProhibitedClaims),
		AuthorizationScope:      strings.TrimSpace(req.AuthorizationScope),
		ContentUsageRights:      strings.TrimSpace(req.ContentUsageRights),
		ReviewWindowHours:       defaultPositiveInt(req.ReviewWindowHours, 72),
		DeadlineAt:              deadlineAt,
		BudgetCents:             req.BudgetCents,
		Currency:                defaultString(strings.ToUpper(strings.TrimSpace(req.Currency)), "CNY"),
		Status:                  status,
		CreatedByUserID:         middleware.CurrentUserID(c),
		CreatedAt:               now,
		UpdatedAt:               now,
	}
	// 合规披露、禁用宣称、授权范围和内容使用权在 brief 阶段固化，后续订单默认继承这些边界。
	if err := h.saveCreatorCampaignBrief(c.Request.Context(), item); err != nil {
		log.Printf("creator brief was not persisted: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "creator brief was not persisted"})
		return
	}

	h.mu.Lock()
	h.creatorBriefs = append([]model.CreatorCampaignBrief{item}, h.creatorBriefs...)
	h.mu.Unlock()
	c.JSON(http.StatusCreated, item)
}

func (h *WorkspaceHandler) ListCreatorOrders(c *gin.Context) {
	workspaceID, ok := h.authorizedWorkspaceID(c)
	if !ok {
		return
	}
	if !h.loadCreatorSnapshot(c.Request.Context(), workspaceID) {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "creator orders were not loaded"})
		return
	}
	h.mu.RLock()
	items := filterByWorkspace(h.creatorOrders, workspaceID, func(item model.CreatorOrder) string { return item.WorkspaceID })
	h.mu.RUnlock()
	if items == nil {
		items = []model.CreatorOrder{}
	}
	c.JSON(http.StatusOK, gin.H{"items": items})
}

func (h *WorkspaceHandler) CreateCreatorOrder(c *gin.Context) {
	workspaceID, ok := h.authorizedWorkspaceID(c)
	if !ok {
		return
	}
	var req createCreatorOrderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}
	briefID := strings.TrimSpace(req.BriefID)
	creatorID := strings.TrimSpace(req.CreatorID)
	if briefID == "" || creatorID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "briefId and creatorId are required"})
		return
	}
	if req.PriceCents < 0 || req.DepositCents < 0 || req.ServiceFeeCents < 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "pricing fields must be non-negative"})
		return
	}
	dueAt, validTime := parseOptionalTime(req.DueAt)
	if !validTime {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid dueAt"})
		return
	}
	if !h.loadCreatorSnapshot(c.Request.Context(), workspaceID) {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "creator order was not loaded"})
		return
	}
	h.mu.RLock()
	brief, briefOK := h.creatorBriefByIDLocked(workspaceID, briefID)
	_, creatorOK := h.creatorByIDLocked(creatorID)
	h.mu.RUnlock()
	if !briefOK {
		c.JSON(http.StatusNotFound, gin.H{"error": "creator brief not found"})
		return
	}
	if !creatorOK {
		c.JSON(http.StatusNotFound, gin.H{"error": "creator not found"})
		return
	}

	now := time.Now().UTC()
	priceCents := req.PriceCents
	if priceCents == 0 {
		priceCents = brief.BudgetCents
	}
	serviceFeeCents := req.ServiceFeeCents
	if serviceFeeCents == 0 && priceCents > 0 {
		serviceFeeCents = priceCents / 10
	}
	if serviceFeeCents > priceCents {
		c.JSON(http.StatusBadRequest, gin.H{"error": "serviceFeeCents cannot exceed priceCents"})
		return
	}
	disclosureRequirements := cleanKeywords(req.DisclosureRequirements)
	if len(disclosureRequirements) == 0 {
		disclosureRequirements = brief.DisclosureRequirements
	}
	deliverableRequirements := cleanKeywords(req.DeliverableRequirements)
	if len(deliverableRequirements) == 0 {
		deliverableRequirements = brief.DeliverableRequirements
	}
	authorizationScope := firstNonEmptyString(req.AuthorizationScope, brief.AuthorizationScope)
	contentUsageRights := firstNonEmptyString(req.ContentUsageRights, brief.ContentUsageRights)
	if len(disclosureRequirements) == 0 || authorizationScope == "" || contentUsageRights == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "disclosureRequirements, authorizationScope, and contentUsageRights are required"})
		return
	}
	order := model.CreatorOrder{
		ID:                      fmt.Sprintf("cor_%d", now.UnixNano()),
		WorkspaceID:             workspaceID,
		BriefID:                 briefID,
		CreatorID:               creatorID,
		Status:                  model.CreatorOrderProposed,
		PriceCents:              priceCents,
		DepositCents:            req.DepositCents,
		ServiceFeeCents:         serviceFeeCents,
		Currency:                defaultString(strings.ToUpper(strings.TrimSpace(req.Currency)), defaultString(brief.Currency, "CNY")),
		DisclosureRequirements:  disclosureRequirements,
		DeliverableRequirements: deliverableRequirements,
		AuthorizationScope:      authorizationScope,
		ContentUsageRights:      contentUsageRights,
		DueAt:                   dueAt,
		LastMessage:             strings.TrimSpace(req.LastMessage),
		CreatedByUserID:         middleware.CurrentUserID(c),
		CreatedAt:               now,
		UpdatedAt:               now,
	}
	settlement := model.CreatorSettlement{
		ID:                 fmt.Sprintf("cst_%d", now.UnixNano()),
		WorkspaceID:        workspaceID,
		OrderID:            order.ID,
		CreatorID:          creatorID,
		Status:             model.CreatorSettlementPending,
		PriceCents:         order.PriceCents,
		DepositCents:       order.DepositCents,
		ServiceFeeCents:    order.ServiceFeeCents,
		CreatorPayoutCents: creatorPayoutCents(order.PriceCents, order.ServiceFeeCents),
		Currency:           order.Currency,
		CreatedAt:          now,
		UpdatedAt:          now,
	}

	// 订单是达人合作的合同化操作单元；状态从 proposed 开始，后续提交/审核/发布必须由后端校验。
	if err := h.saveCreatorOrderWithSettlement(c.Request.Context(), order, settlement); err != nil {
		log.Printf("creator order was not persisted: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "creator order was not persisted"})
		return
	}

	h.mu.Lock()
	h.creatorOrders = append([]model.CreatorOrder{order}, h.creatorOrders...)
	h.creatorSettlements = append([]model.CreatorSettlement{settlement}, h.creatorSettlements...)
	h.mu.Unlock()
	c.JSON(http.StatusCreated, creatorOrderResponse{Order: order, Settlement: settlement})
}

func (h *WorkspaceHandler) SubmitCreatorDeliverable(c *gin.Context) {
	workspaceID, ok := h.authorizedWorkspaceID(c)
	if !ok {
		return
	}
	orderID := strings.TrimSpace(c.Param("orderId"))
	if orderID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "orderId is required"})
		return
	}
	var req submitCreatorDeliverableRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}
	if strings.TrimSpace(req.Title) == "" && strings.TrimSpace(req.Content) == "" && len(cleanKeywords(req.AssetURLs)) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "deliverable content or assetUrls are required"})
		return
	}
	if !h.loadCreatorSnapshot(c.Request.Context(), workspaceID) {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "creator order was not loaded"})
		return
	}

	now := time.Now().UTC()
	h.mu.RLock()
	order, orderOK := h.creatorOrderByIDLocked(workspaceID, orderID)
	revision := h.nextCreatorDeliverableRevisionLocked(workspaceID, orderID)
	h.mu.RUnlock()
	if !orderOK {
		c.JSON(http.StatusNotFound, gin.H{"error": "creator order not found"})
		return
	}
	if !creatorOrderAllowsSubmission(order.Status) {
		c.JSON(http.StatusConflict, gin.H{"error": "creator order does not accept deliverable submission in current status"})
		return
	}

	deliverable := model.CreatorDeliverable{
		ID:          fmt.Sprintf("cdl_%d", now.UnixNano()),
		WorkspaceID: workspaceID,
		OrderID:     order.ID,
		CreatorID:   order.CreatorID,
		Type:        defaultString(strings.TrimSpace(req.Type), "draft"),
		Title:       strings.TrimSpace(req.Title),
		Content:     strings.TrimSpace(req.Content),
		AssetURLs:   cleanKeywords(req.AssetURLs),
		Status:      model.CreatorDeliverableSubmitted,
		Revision:    revision,
		SubmittedAt: now,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	order.Status = model.CreatorOrderSubmitted
	order.LastMessage = "达人已提交交付物，等待品牌审核。"
	order.UpdatedAt = now

	// 交付物提交会推进订单到 submitted，避免前端绕过审核流程直接发布。
	if err := h.saveCreatorDeliverableWithOrderAndEvidence(c.Request.Context(), deliverable, order, model.CreatorComplianceEvidence{}); err != nil {
		log.Printf("creator deliverable was not persisted: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "creator deliverable was not persisted"})
		return
	}

	h.mu.Lock()
	h.creatorDeliverables = append([]model.CreatorDeliverable{deliverable}, h.creatorDeliverables...)
	h.upsertCreatorOrderLocked(order)
	h.mu.Unlock()
	c.JSON(http.StatusCreated, deliverable)
}

func (h *WorkspaceHandler) ReviewCreatorDeliverable(c *gin.Context) {
	workspaceID, ok := h.authorizedWorkspaceID(c)
	if !ok {
		return
	}
	deliverableID := strings.TrimSpace(c.Param("deliverableId"))
	if deliverableID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "deliverableId is required"})
		return
	}
	var req reviewCreatorDeliverableRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}
	decision := strings.TrimSpace(strings.ToLower(req.Decision))
	if decision != "approve" && decision != "request_revision" && decision != "reject" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "decision must be approve, request_revision, or reject"})
		return
	}
	if !h.loadCreatorSnapshot(c.Request.Context(), workspaceID) {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "creator deliverable was not loaded"})
		return
	}

	now := time.Now().UTC()
	h.mu.RLock()
	deliverable, deliverableOK := h.creatorDeliverableByIDLocked(workspaceID, deliverableID)
	order, orderOK := h.creatorOrderByIDLocked(workspaceID, deliverable.OrderID)
	h.mu.RUnlock()
	if !deliverableOK {
		c.JSON(http.StatusNotFound, gin.H{"error": "creator deliverable not found"})
		return
	}
	if !orderOK {
		c.JSON(http.StatusBadRequest, gin.H{"error": "creator order reference is missing"})
		return
	}
	if deliverable.Status != model.CreatorDeliverableSubmitted {
		c.JSON(http.StatusConflict, gin.H{"error": "creator deliverable is not waiting for review"})
		return
	}

	deliverable.ReviewFeedback = strings.TrimSpace(req.Feedback)
	deliverable.ReviewedAt = &now
	deliverable.UpdatedAt = now
	switch decision {
	case "approve":
		deliverable.Status = model.CreatorDeliverableApproved
		order.Status = model.CreatorOrderApproved
		order.LastMessage = "交付物已通过品牌审核，等待达人发布证明。"
	case "request_revision":
		deliverable.Status = model.CreatorDeliverableRevisionRequested
		order.Status = model.CreatorOrderInProgress
		order.LastMessage = defaultString(deliverable.ReviewFeedback, "品牌要求达人修改后重新提交。")
	case "reject":
		deliverable.Status = model.CreatorDeliverableRejected
		order.Status = model.CreatorOrderDisputed
		order.LastMessage = defaultString(deliverable.ReviewFeedback, "交付物被拒绝，订单进入争议状态。")
	}
	order.UpdatedAt = now

	evidence := model.CreatorComplianceEvidence{
		ID:                 fmt.Sprintf("cev_%d", now.UnixNano()),
		WorkspaceID:        workspaceID,
		OrderID:            order.ID,
		DeliverableID:      deliverable.ID,
		CreatorID:          order.CreatorID,
		EvidenceType:       model.CreatorEvidenceReviewLog,
		AuthorizationScope: order.AuthorizationScope,
		ContentUsageRights: order.ContentUsageRights,
		Notes:              fmt.Sprintf("decision=%s feedback=%s", decision, deliverable.ReviewFeedback),
		CreatedByUserID:    middleware.CurrentUserID(c),
		CreatedAt:          now,
	}

	// 审核日志是合规证据，需与交付物/订单状态一起保存，便于事后解释品牌审批链路。
	if err := h.saveCreatorDeliverableWithOrderAndEvidence(c.Request.Context(), deliverable, order, evidence); err != nil {
		log.Printf("creator deliverable review was not persisted: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "creator deliverable review was not persisted"})
		return
	}

	h.mu.Lock()
	h.upsertCreatorDeliverableLocked(deliverable)
	h.upsertCreatorOrderLocked(order)
	h.creatorComplianceEvidence = append([]model.CreatorComplianceEvidence{evidence}, h.creatorComplianceEvidence...)
	h.mu.Unlock()
	c.JSON(http.StatusOK, deliverable)
}

func (h *WorkspaceHandler) RecordCreatorPublicationProof(c *gin.Context) {
	workspaceID, ok := h.authorizedWorkspaceID(c)
	if !ok {
		return
	}
	deliverableID := strings.TrimSpace(c.Param("deliverableId"))
	if deliverableID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "deliverableId is required"})
		return
	}
	var req recordCreatorPublicationProofRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}
	externalURL := strings.TrimSpace(req.ExternalURL)
	if !validExternalURL(externalURL) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "valid externalUrl is required"})
		return
	}
	disclosureText := strings.TrimSpace(req.DisclosureText)
	if disclosureText == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "disclosureText is required"})
		return
	}
	publishedAt, ok := parseOptionalTime(req.PublishedAt)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid publishedAt"})
		return
	}
	if !h.loadCreatorSnapshot(c.Request.Context(), workspaceID) {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "creator deliverable was not loaded"})
		return
	}

	now := time.Now().UTC()
	if publishedAt == nil {
		publishedAt = &now
	}
	h.mu.RLock()
	deliverable, deliverableOK := h.creatorDeliverableByIDLocked(workspaceID, deliverableID)
	order, orderOK := h.creatorOrderByIDLocked(workspaceID, deliverable.OrderID)
	settlement, settlementOK := h.creatorSettlementByOrderIDLocked(workspaceID, deliverable.OrderID)
	h.mu.RUnlock()
	if !deliverableOK {
		c.JSON(http.StatusNotFound, gin.H{"error": "creator deliverable not found"})
		return
	}
	if !orderOK || !settlementOK {
		c.JSON(http.StatusBadRequest, gin.H{"error": "creator order or settlement reference is missing"})
		return
	}
	if deliverable.Status != model.CreatorDeliverableApproved && deliverable.Status != model.CreatorDeliverablePublished {
		c.JSON(http.StatusConflict, gin.H{"error": "creator deliverable must be approved before publication proof"})
		return
	}

	deliverable.Status = model.CreatorDeliverablePublished
	deliverable.ExternalURL = externalURL
	deliverable.PublicationProofURL = strings.TrimSpace(req.PublicationProofURL)
	deliverable.PublicationProofNote = strings.TrimSpace(req.PublicationProofNote)
	deliverable.PublishedAt = publishedAt
	deliverable.UpdatedAt = now
	order.Status = model.CreatorOrderPublished
	order.LastMessage = "已记录达人发布证明，结算进入待付款条件。"
	order.UpdatedAt = now
	// 结算状态只在发布证明和合规证据一并入库后推进，避免未披露广告内容提前进入付款流程。
	settlement.Status = model.CreatorSettlementPayable
	settlement.UpdatedAt = now

	evidence := model.CreatorComplianceEvidence{
		ID:                 fmt.Sprintf("cev_%d", now.UnixNano()),
		WorkspaceID:        workspaceID,
		OrderID:            order.ID,
		DeliverableID:      deliverable.ID,
		CreatorID:          order.CreatorID,
		EvidenceType:       model.CreatorEvidencePublicationProof,
		DisclosureText:     disclosureText,
		AuthorizationScope: order.AuthorizationScope,
		ContentUsageRights: order.ContentUsageRights,
		ExternalURL:        externalURL,
		FileURL:            deliverable.PublicationProofURL,
		Notes:              strings.TrimSpace(req.Notes),
		CreatedByUserID:    middleware.CurrentUserID(c),
		CreatedAt:          now,
	}

	// 发布证明必须保留广告披露、授权范围和内容使用权证据，保护品牌、达人和平台的事后追溯。
	if err := h.saveCreatorPublicationProof(c.Request.Context(), deliverable, order, settlement, evidence); err != nil {
		log.Printf("creator publication proof was not persisted: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "creator publication proof was not persisted"})
		return
	}

	h.mu.Lock()
	h.upsertCreatorDeliverableLocked(deliverable)
	h.upsertCreatorOrderLocked(order)
	h.upsertCreatorSettlementLocked(settlement)
	h.creatorComplianceEvidence = append([]model.CreatorComplianceEvidence{evidence}, h.creatorComplianceEvidence...)
	h.mu.Unlock()
	c.JSON(http.StatusOK, creatorPublicationProofResponse{
		Deliverable: deliverable,
		Order:       order,
		Settlement:  settlement,
		Evidence:    evidence,
	})
}

func (h *WorkspaceHandler) ListCreatorDeliverables(c *gin.Context) {
	workspaceID, ok := h.authorizedWorkspaceID(c)
	if !ok {
		return
	}
	if !h.loadCreatorSnapshot(c.Request.Context(), workspaceID) {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "creator deliverables were not loaded"})
		return
	}
	h.mu.RLock()
	items := filterByWorkspace(h.creatorDeliverables, workspaceID, func(item model.CreatorDeliverable) string { return item.WorkspaceID })
	h.mu.RUnlock()
	if items == nil {
		items = []model.CreatorDeliverable{}
	}
	c.JSON(http.StatusOK, gin.H{"items": items})
}

func (h *WorkspaceHandler) ListCreatorSettlements(c *gin.Context) {
	workspaceID, ok := h.authorizedWorkspaceID(c)
	if !ok {
		return
	}
	if !h.loadCreatorSnapshot(c.Request.Context(), workspaceID) {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "creator settlements were not loaded"})
		return
	}
	h.mu.RLock()
	items := filterByWorkspace(h.creatorSettlements, workspaceID, func(item model.CreatorSettlement) string { return item.WorkspaceID })
	h.mu.RUnlock()
	if items == nil {
		items = []model.CreatorSettlement{}
	}
	c.JSON(http.StatusOK, gin.H{"items": items})
}

func (h *WorkspaceHandler) ListCreatorComplianceEvidence(c *gin.Context) {
	workspaceID, ok := h.authorizedWorkspaceID(c)
	if !ok {
		return
	}
	if !h.loadCreatorSnapshot(c.Request.Context(), workspaceID) {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "creator compliance evidence was not loaded"})
		return
	}
	h.mu.RLock()
	items := filterByWorkspace(h.creatorComplianceEvidence, workspaceID, func(item model.CreatorComplianceEvidence) string { return item.WorkspaceID })
	h.mu.RUnlock()
	if items == nil {
		items = []model.CreatorComplianceEvidence{}
	}
	c.JSON(http.StatusOK, gin.H{"items": items})
}

func (h *WorkspaceHandler) loadCreatorSnapshot(ctx context.Context, workspaceID string) bool {
	if h.db == nil || h.db.SQL() == nil {
		return true
	}
	dbCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	snapshot, err := h.db.LoadCreatorSnapshot(dbCtx, workspaceID)
	if err != nil {
		log.Printf("creator snapshot load failed: %v", err)
		return false
	}
	h.mu.Lock()
	h.creators = snapshot.Creators
	h.creatorMediaAccounts = snapshot.MediaAccounts
	h.creatorShortlists = snapshot.Shortlists
	h.creatorBriefs = snapshot.Briefs
	h.creatorOrders = snapshot.Orders
	h.creatorDeliverables = snapshot.Deliverables
	h.creatorSettlements = snapshot.Settlements
	h.creatorComplianceEvidence = snapshot.ComplianceEvidence
	h.mu.Unlock()
	return true
}

func (h *WorkspaceHandler) saveCreatorShortlist(ctx context.Context, item model.CreatorShortlist) error {
	if h.db == nil || h.db.SQL() == nil {
		return nil
	}
	dbCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	return h.db.SaveCreatorShortlist(dbCtx, item)
}

func (h *WorkspaceHandler) saveCreatorCampaignBrief(ctx context.Context, item model.CreatorCampaignBrief) error {
	if h.db == nil || h.db.SQL() == nil {
		return nil
	}
	dbCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	return h.db.SaveCreatorCampaignBrief(dbCtx, item)
}

func (h *WorkspaceHandler) saveCreatorOrderWithSettlement(ctx context.Context, order model.CreatorOrder, settlement model.CreatorSettlement) error {
	if h.db == nil || h.db.SQL() == nil {
		return nil
	}
	dbCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	return h.db.SaveCreatorOrderWithSettlement(dbCtx, order, settlement)
}

func (h *WorkspaceHandler) saveCreatorDeliverableWithOrderAndEvidence(ctx context.Context, deliverable model.CreatorDeliverable, order model.CreatorOrder, evidence model.CreatorComplianceEvidence) error {
	if h.db == nil || h.db.SQL() == nil {
		return nil
	}
	dbCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	return h.db.SaveCreatorDeliverableWithOrderAndEvidence(dbCtx, deliverable, order, evidence)
}

func (h *WorkspaceHandler) saveCreatorPublicationProof(ctx context.Context, deliverable model.CreatorDeliverable, order model.CreatorOrder, settlement model.CreatorSettlement, evidence model.CreatorComplianceEvidence) error {
	if h.db == nil || h.db.SQL() == nil {
		return nil
	}
	dbCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	return h.db.SaveCreatorPublicationProof(dbCtx, deliverable, order, settlement, evidence)
}

func (h *WorkspaceHandler) creatorByIDLocked(creatorID string) (model.Creator, bool) {
	for _, item := range h.creators {
		if item.ID == creatorID {
			return item, true
		}
	}
	return model.Creator{}, false
}

func (h *WorkspaceHandler) creatorMediaAccountsByCreatorLocked(creatorID string) []model.CreatorMediaAccount {
	items := []model.CreatorMediaAccount{}
	for _, item := range h.creatorMediaAccounts {
		if item.CreatorID == creatorID {
			items = append(items, item)
		}
	}
	return items
}

func (h *WorkspaceHandler) creatorBriefByIDLocked(workspaceID string, briefID string) (model.CreatorCampaignBrief, bool) {
	for _, item := range h.creatorBriefs {
		if item.WorkspaceID == workspaceID && item.ID == briefID {
			return item, true
		}
	}
	return model.CreatorCampaignBrief{}, false
}

func (h *WorkspaceHandler) creatorOrderByIDLocked(workspaceID string, orderID string) (model.CreatorOrder, bool) {
	for _, item := range h.creatorOrders {
		if item.WorkspaceID == workspaceID && item.ID == orderID {
			return item, true
		}
	}
	return model.CreatorOrder{}, false
}

func (h *WorkspaceHandler) creatorDeliverableByIDLocked(workspaceID string, deliverableID string) (model.CreatorDeliverable, bool) {
	for _, item := range h.creatorDeliverables {
		if item.WorkspaceID == workspaceID && item.ID == deliverableID {
			return item, true
		}
	}
	return model.CreatorDeliverable{}, false
}

func (h *WorkspaceHandler) creatorSettlementByOrderIDLocked(workspaceID string, orderID string) (model.CreatorSettlement, bool) {
	for _, item := range h.creatorSettlements {
		if item.WorkspaceID == workspaceID && item.OrderID == orderID {
			return item, true
		}
	}
	return model.CreatorSettlement{}, false
}

func (h *WorkspaceHandler) nextCreatorDeliverableRevisionLocked(workspaceID string, orderID string) int {
	maxRevision := 0
	for _, item := range h.creatorDeliverables {
		if item.WorkspaceID == workspaceID && item.OrderID == orderID && item.Revision > maxRevision {
			maxRevision = item.Revision
		}
	}
	return maxRevision + 1
}

func (h *WorkspaceHandler) upsertCreatorShortlistLocked(item model.CreatorShortlist) {
	for index := range h.creatorShortlists {
		current := &h.creatorShortlists[index]
		if current.WorkspaceID == item.WorkspaceID && current.CreatorID == item.CreatorID && current.Name == item.Name {
			h.creatorShortlists[index] = item
			return
		}
	}
	h.creatorShortlists = append([]model.CreatorShortlist{item}, h.creatorShortlists...)
}

func (h *WorkspaceHandler) upsertCreatorOrderLocked(item model.CreatorOrder) {
	for index := range h.creatorOrders {
		if h.creatorOrders[index].WorkspaceID == item.WorkspaceID && h.creatorOrders[index].ID == item.ID {
			h.creatorOrders[index] = item
			return
		}
	}
	h.creatorOrders = append([]model.CreatorOrder{item}, h.creatorOrders...)
}

func (h *WorkspaceHandler) upsertCreatorDeliverableLocked(item model.CreatorDeliverable) {
	for index := range h.creatorDeliverables {
		if h.creatorDeliverables[index].WorkspaceID == item.WorkspaceID && h.creatorDeliverables[index].ID == item.ID {
			h.creatorDeliverables[index] = item
			return
		}
	}
	h.creatorDeliverables = append([]model.CreatorDeliverable{item}, h.creatorDeliverables...)
}

func (h *WorkspaceHandler) upsertCreatorSettlementLocked(item model.CreatorSettlement) {
	for index := range h.creatorSettlements {
		if h.creatorSettlements[index].WorkspaceID == item.WorkspaceID && h.creatorSettlements[index].ID == item.ID {
			h.creatorSettlements[index] = item
			return
		}
	}
	h.creatorSettlements = append([]model.CreatorSettlement{item}, h.creatorSettlements...)
}

func filterCreatorShortlists(items []model.CreatorShortlist, creatorID string) []model.CreatorShortlist {
	filtered := []model.CreatorShortlist{}
	for _, item := range items {
		if item.CreatorID == creatorID {
			filtered = append(filtered, item)
		}
	}
	return filtered
}

func validCreatorQualificationStatus(value string) bool {
	switch value {
	case "watching", "qualified", "rejected", "ordered":
		return true
	default:
		return false
	}
}

func validCreatorCampaignBriefStatus(value model.CreatorCampaignBriefStatus) bool {
	switch value {
	case model.CreatorCampaignBriefDraft, model.CreatorCampaignBriefActive, model.CreatorCampaignBriefArchived:
		return true
	default:
		return false
	}
}

func creatorOrderAllowsSubmission(status model.CreatorOrderStatus) bool {
	switch status {
	case model.CreatorOrderProposed, model.CreatorOrderAccepted, model.CreatorOrderInProgress:
		return true
	default:
		return false
	}
}

func defaultPositiveInt(value int, fallback int) int {
	if value <= 0 {
		return fallback
	}
	return value
}

func creatorPayoutCents(priceCents int, serviceFeeCents int) int {
	payout := priceCents - serviceFeeCents
	if payout < 0 {
		return 0
	}
	return payout
}

func validExternalURL(value string) bool {
	parsed, err := url.ParseRequestURI(value)
	return err == nil && parsed.Scheme != "" && parsed.Host != ""
}

package handler

import (
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"geopress/backend/internal/http/middleware"
	"geopress/backend/internal/model"

	"github.com/gin-gonic/gin"
)

type WorkspaceHandler struct {
	mu             sync.RWMutex
	users          []model.User
	workspaces     []model.Workspace
	members        []model.WorkspaceMember
	knowledgeBases []model.KnowledgeBase
	knowledgeItems []model.KnowledgeItem
	platforms      []model.MediaPlatform
	accounts       []model.MediaAccount
	contents       []model.Content
	schedules      []model.PublishSchedule
	jobs           []model.PublishJob
}

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type createKnowledgeBaseRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

type createKnowledgeItemRequest struct {
	KnowledgeBaseID string `json:"knowledgeBaseId"`
	Type            string `json:"type"`
	Title           string `json:"title"`
	Content         string `json:"content"`
}

type createMediaAccountRequest struct {
	PlatformID string `json:"platformId"`
	Name       string `json:"name"`
	ExternalID string `json:"externalId"`
}

type generateContentRequest struct {
	Keywords        []string `json:"keywords"`
	ContentType     string   `json:"contentType"`
	KnowledgeBaseID string   `json:"knowledgeBaseId"`
}

type createContentRequest struct {
	Title           string   `json:"title"`
	Summary         string   `json:"summary"`
	Body            string   `json:"body"`
	Author          string   `json:"author"`
	KnowledgeBaseID string   `json:"knowledgeBaseId"`
	Keywords        []string `json:"keywords"`
}

type createPublishScheduleRequest struct {
	Name           string                         `json:"name"`
	ContentID      string                         `json:"contentId"`
	MediaAccountID string                         `json:"mediaAccountId"`
	Frequency      model.PublishScheduleFrequency `json:"frequency"`
	NextRunAt      time.Time                      `json:"nextRunAt"`
}

type createMediaPlatformRequest struct {
	Name               string   `json:"name"`
	Type               string   `json:"type"`
	Enabled            bool     `json:"enabled"`
	SupportsArticle    bool     `json:"supportsArticle"`
	SupportsImage      bool     `json:"supportsImage"`
	SupportsScheduling bool     `json:"supportsScheduling"`
	CredentialFields   []string `json:"credentialFields"`
}

func NewWorkspaceHandler() *WorkspaceHandler {
	now := time.Now().UTC()
	expiresAt := now.AddDate(0, 2, 0)

	return &WorkspaceHandler{
		users: []model.User{
			{
				ID:              "usr_demo",
				Name:            "Ava Chen",
				Email:           "demo@geopress.local",
				IsPlatformAdmin: true,
				CreatedAt:       now.AddDate(0, -3, 0),
			},
			{
				ID:              "usr_growth",
				Name:            "Noah Wang",
				Email:           "growth@geopress.local",
				IsPlatformAdmin: false,
				CreatedAt:       now.AddDate(0, -2, 0),
			},
		},
		workspaces: []model.Workspace{
			{
				ID:        "wks_personal",
				Name:      "Ava 的个人工作区",
				Type:      model.WorkspacePersonal,
				Plan:      "Personal",
				Status:    "active",
				Industry:  "独立创作者",
				Language:  "zh-CN",
				Tone:      "专业、清晰、克制",
				CreatedAt: now.AddDate(0, -3, 0),
			},
			{
				ID:        "wks_acme",
				Name:      "Acme Growth Team",
				Type:      model.WorkspaceCompany,
				Plan:      "Team",
				Status:    "active",
				Industry:  "B2B SaaS",
				Language:  "zh-CN",
				Tone:      "可信、实用、面向增长负责人",
				CreatedAt: now.AddDate(0, -2, 0),
			},
		},
		members: []model.WorkspaceMember{
			{UserID: "usr_demo", WorkspaceID: "wks_personal", Role: "owner"},
			{UserID: "usr_demo", WorkspaceID: "wks_acme", Role: "admin"},
			{UserID: "usr_growth", WorkspaceID: "wks_acme", Role: "editor"},
		},
		knowledgeBases: []model.KnowledgeBase{
			{
				ID:          "kb_brand",
				WorkspaceID: "wks_acme",
				Name:        "品牌与产品资料",
				Description: "公司定位、产品价值、目标客户和常用表达。",
				ItemCount:   2,
				UpdatedAt:   now.Add(-5 * time.Hour),
			},
			{
				ID:          "kb_personal",
				WorkspaceID: "wks_personal",
				Name:        "个人写作素材",
				Description: "个人介绍、服务范围、案例和写作风格。",
				ItemCount:   1,
				UpdatedAt:   now.Add(-24 * time.Hour),
			},
		},
		knowledgeItems: []model.KnowledgeItem{
			{
				ID:              "kbi_1001",
				KnowledgeBaseID: "kb_brand",
				WorkspaceID:     "wks_acme",
				Type:            "brand",
				Title:           "品牌定位",
				Content:         "Acme 面向 B2B SaaS 团队，帮助市场和增长负责人规划内容生产、分发和复盘。",
				Enabled:         true,
				UpdatedAt:       now.Add(-5 * time.Hour),
			},
			{
				ID:              "kbi_1002",
				KnowledgeBaseID: "kb_brand",
				WorkspaceID:     "wks_acme",
				Type:            "audience",
				Title:           "目标受众",
				Content:         "主要读者是市场负责人、内容运营、创始人和增长团队。",
				Enabled:         true,
				UpdatedAt:       now.Add(-6 * time.Hour),
			},
			{
				ID:              "kbi_2001",
				KnowledgeBaseID: "kb_personal",
				WorkspaceID:     "wks_personal",
				Type:            "style",
				Title:           "写作风格",
				Content:         "文章应直接、具体，避免夸张营销话术，强调可执行建议。",
				Enabled:         true,
				UpdatedAt:       now.Add(-24 * time.Hour),
			},
		},
		platforms: []model.MediaPlatform{
			{
				ID:                 "plt_wordpress",
				Name:               "WordPress",
				Type:               "wordpress",
				Enabled:            true,
				SupportsArticle:    true,
				SupportsImage:      true,
				SupportsScheduling: true,
				CredentialFields:   []string{"siteUrl", "username", "applicationPassword"},
			},
			{
				ID:                 "plt_wechat",
				Name:               "微信公众号",
				Type:               "wechat_official",
				Enabled:            true,
				SupportsArticle:    true,
				SupportsImage:      true,
				SupportsScheduling: false,
				CredentialFields:   []string{"appId", "appSecret"},
			},
			{
				ID:                 "plt_zhihu",
				Name:               "知乎",
				Type:               "zhihu",
				Enabled:            false,
				SupportsArticle:    true,
				SupportsImage:      true,
				SupportsScheduling: false,
				CredentialFields:   []string{"accessToken"},
			},
		},
		accounts: []model.MediaAccount{
			{
				ID:            "acc_wordpress",
				WorkspaceID:   "wks_acme",
				PlatformID:    "plt_wordpress",
				Name:          "Acme 官网博客",
				ExternalID:    "blog.acme.test",
				Status:        "connected",
				ExpiresAt:     &expiresAt,
				LastCheckedAt: now.Add(-2 * time.Hour),
			},
			{
				ID:            "acc_personal_wp",
				WorkspaceID:   "wks_personal",
				PlatformID:    "plt_wordpress",
				Name:          "Ava 个人博客",
				ExternalID:    "ava.example.test",
				Status:        "connected",
				ExpiresAt:     &expiresAt,
				LastCheckedAt: now.Add(-6 * time.Hour),
			},
		},
		contents: []model.Content{
			{
				ID:              "cnt_1001",
				WorkspaceID:     "wks_acme",
				KnowledgeBaseID: "kb_brand",
				Title:           "Q3 SaaS 增长内容规划",
				Summary:         "围绕获客、转化和留存的内容发布计划。",
				Body:            "这是一篇示例草稿，用于展示内容生命周期和排程发布。",
				Keywords:        []string{"SaaS", "增长", "内容营销"},
				Status:          model.ContentScheduled,
				Author:          "Ava Chen",
				Source:          "mock_ai",
				UpdatedAt:       now.Add(-2 * time.Hour),
			},
			{
				ID:              "cnt_2001",
				WorkspaceID:     "wks_personal",
				KnowledgeBaseID: "kb_personal",
				Title:           "独立顾问如何搭建内容飞轮",
				Summary:         "用稳定输出和案例沉淀提升获客效率。",
				Body:            "这是一篇个人工作区示例内容。",
				Keywords:        []string{"独立顾问", "内容飞轮"},
				Status:          model.ContentDraft,
				Author:          "Ava Chen",
				Source:          "manual",
				UpdatedAt:       now.Add(-20 * time.Hour),
			},
		},
		schedules: []model.PublishSchedule{
			{
				ID:             "sch_1001",
				WorkspaceID:    "wks_acme",
				Name:           "每周三官网博客",
				ContentID:      "cnt_1001",
				MediaAccountID: "acc_wordpress",
				Frequency:      model.PublishScheduleWeekly,
				NextRunAt:      now.Add(48 * time.Hour),
				Enabled:        true,
				CreatedAt:      now.Add(-24 * time.Hour),
			},
		},
		jobs: []model.PublishJob{
			{
				ID:             "job_9001",
				WorkspaceID:    "wks_acme",
				ScheduleID:     "sch_1001",
				ContentID:      "cnt_1001",
				MediaAccountID: "acc_wordpress",
				Status:         model.PublishJobQueued,
				ScheduledAt:    now.Add(48 * time.Hour),
				LastMessage:    "Waiting for scheduled publish window.",
			},
		},
	}
}

func (h *WorkspaceHandler) Register(router gin.IRouter, auth gin.HandlerFunc) {
	router.POST("/auth/login", h.Login)

	protected := router.Group("/")
	protected.Use(auth)
	protected.GET("/me", h.Me)
	protected.GET("/workspaces", h.ListWorkspaces)
	protected.GET("/overview", h.Overview)
	protected.GET("/knowledge-bases", h.ListKnowledgeBases)
	protected.POST("/knowledge-bases", h.CreateKnowledgeBase)
	protected.GET("/knowledge-items", h.ListKnowledgeItems)
	protected.POST("/knowledge-items", h.CreateKnowledgeItem)
	protected.GET("/media-platforms", h.ListMediaPlatforms)
	protected.GET("/media-accounts", h.ListMediaAccounts)
	protected.POST("/media-accounts", h.CreateMediaAccount)
	protected.GET("/contents", h.ListContents)
	protected.POST("/contents", h.CreateContent)
	protected.POST("/contents/generate", h.GenerateContent)
	protected.GET("/publish-schedules", h.ListPublishSchedules)
	protected.POST("/publish-schedules", h.CreatePublishSchedule)
	protected.GET("/publish-jobs", h.ListPublishJobs)

	admin := protected.Group("/admin")
	admin.Use(h.requirePlatformAdmin())
	admin.GET("/overview", h.AdminOverview)
	admin.GET("/users", h.AdminListUsers)
	admin.GET("/workspaces", h.AdminListWorkspaces)
	admin.GET("/workspace-members", h.AdminListWorkspaceMembers)
	admin.GET("/media-platforms", h.AdminListMediaPlatforms)
	admin.POST("/media-platforms", h.AdminCreateMediaPlatform)
	admin.GET("/media-accounts", h.AdminListMediaAccounts)
}

func (h *WorkspaceHandler) Login(c *gin.Context) {
	var req loginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	email := strings.TrimSpace(strings.ToLower(req.Email))
	if email == "" {
		email = "demo@geopress.local"
	}

	userID := "usr_demo"
	token := "demo-token"
	if email == "growth@geopress.local" {
		userID = "usr_growth"
		token = "growth-token"
	}

	h.mu.RLock()
	user, _ := h.userByID(userID)
	workspaces := h.workspacesForUser(userID)
	h.mu.RUnlock()

	c.JSON(http.StatusOK, gin.H{
		"token":      token,
		"user":       user,
		"workspaces": workspaces,
	})
}

func (h *WorkspaceHandler) Me(c *gin.Context) {
	userID := middleware.CurrentUserID(c)

	h.mu.RLock()
	user, ok := h.userByID(userID)
	workspaces := h.workspacesForUser(userID)
	h.mu.RUnlock()

	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "user not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"user": user, "workspaces": workspaces})
}

func (h *WorkspaceHandler) ListWorkspaces(c *gin.Context) {
	userID := middleware.CurrentUserID(c)

	h.mu.RLock()
	workspaces := h.workspacesForUser(userID)
	h.mu.RUnlock()

	c.JSON(http.StatusOK, gin.H{"items": workspaces})
}

func (h *WorkspaceHandler) Overview(c *gin.Context) {
	workspaceID, ok := h.authorizedWorkspaceID(c)
	if !ok {
		return
	}

	h.mu.RLock()
	knowledgeBases := filterByWorkspace(h.knowledgeBases, workspaceID, func(item model.KnowledgeBase) string { return item.WorkspaceID })
	accounts := filterByWorkspace(h.accounts, workspaceID, func(item model.MediaAccount) string { return item.WorkspaceID })
	contents := filterByWorkspace(h.contents, workspaceID, func(item model.Content) string { return item.WorkspaceID })
	schedules := filterByWorkspace(h.schedules, workspaceID, func(item model.PublishSchedule) string { return item.WorkspaceID })
	jobs := filterByWorkspace(h.jobs, workspaceID, func(item model.PublishJob) string { return item.WorkspaceID })
	h.mu.RUnlock()

	c.JSON(http.StatusOK, gin.H{
		"workspaceId":        workspaceID,
		"knowledgeBaseCount": len(knowledgeBases),
		"mediaAccountCount":  len(accounts),
		"contentCount":       len(contents),
		"scheduleCount":      len(schedules),
		"publishJobCount":    len(jobs),
		"draftCount":         countContents(contents, model.ContentDraft),
		"queuedJobs":         countJobs(jobs, model.PublishJobQueued),
		"failedJobs":         countJobs(jobs, model.PublishJobFailed),
	})
}

func (h *WorkspaceHandler) ListKnowledgeBases(c *gin.Context) {
	workspaceID, ok := h.authorizedWorkspaceID(c)
	if !ok {
		return
	}

	h.mu.RLock()
	items := filterByWorkspace(h.knowledgeBases, workspaceID, func(item model.KnowledgeBase) string { return item.WorkspaceID })
	h.mu.RUnlock()
	c.JSON(http.StatusOK, gin.H{"items": items})
}

func (h *WorkspaceHandler) CreateKnowledgeBase(c *gin.Context) {
	workspaceID, ok := h.authorizedWorkspaceID(c)
	if !ok {
		return
	}

	var req createKnowledgeBaseRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	name := strings.TrimSpace(req.Name)
	if name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "name is required"})
		return
	}

	now := time.Now().UTC()
	item := model.KnowledgeBase{
		ID:          fmt.Sprintf("kb_%d", now.UnixNano()),
		WorkspaceID: workspaceID,
		Name:        name,
		Description: strings.TrimSpace(req.Description),
		ItemCount:   0,
		UpdatedAt:   now,
	}

	h.mu.Lock()
	h.knowledgeBases = append([]model.KnowledgeBase{item}, h.knowledgeBases...)
	h.mu.Unlock()
	c.JSON(http.StatusCreated, item)
}

func (h *WorkspaceHandler) ListKnowledgeItems(c *gin.Context) {
	workspaceID, ok := h.authorizedWorkspaceID(c)
	if !ok {
		return
	}

	h.mu.RLock()
	items := filterByWorkspace(h.knowledgeItems, workspaceID, func(item model.KnowledgeItem) string { return item.WorkspaceID })
	h.mu.RUnlock()
	c.JSON(http.StatusOK, gin.H{"items": items})
}

func (h *WorkspaceHandler) CreateKnowledgeItem(c *gin.Context) {
	workspaceID, ok := h.authorizedWorkspaceID(c)
	if !ok {
		return
	}

	var req createKnowledgeItemRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	title := strings.TrimSpace(req.Title)
	content := strings.TrimSpace(req.Content)
	if title == "" || content == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "title and content are required"})
		return
	}

	now := time.Now().UTC()
	item := model.KnowledgeItem{
		ID:              fmt.Sprintf("kbi_%d", now.UnixNano()),
		KnowledgeBaseID: strings.TrimSpace(req.KnowledgeBaseID),
		WorkspaceID:     workspaceID,
		Type:            defaultString(strings.TrimSpace(req.Type), "note"),
		Title:           title,
		Content:         content,
		Enabled:         true,
		UpdatedAt:       now,
	}

	h.mu.Lock()
	h.knowledgeItems = append([]model.KnowledgeItem{item}, h.knowledgeItems...)
	h.bumpKnowledgeBaseCount(workspaceID, item.KnowledgeBaseID, 1)
	h.mu.Unlock()
	c.JSON(http.StatusCreated, item)
}

func (h *WorkspaceHandler) ListMediaPlatforms(c *gin.Context) {
	h.mu.RLock()
	items := append([]model.MediaPlatform(nil), h.platforms...)
	h.mu.RUnlock()
	c.JSON(http.StatusOK, gin.H{"items": items})
}

func (h *WorkspaceHandler) ListMediaAccounts(c *gin.Context) {
	workspaceID, ok := h.authorizedWorkspaceID(c)
	if !ok {
		return
	}

	h.mu.RLock()
	items := filterByWorkspace(h.accounts, workspaceID, func(item model.MediaAccount) string { return item.WorkspaceID })
	h.mu.RUnlock()
	c.JSON(http.StatusOK, gin.H{"items": items})
}

func (h *WorkspaceHandler) CreateMediaAccount(c *gin.Context) {
	workspaceID, ok := h.authorizedWorkspaceID(c)
	if !ok {
		return
	}

	var req createMediaAccountRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	name := strings.TrimSpace(req.Name)
	platformID := strings.TrimSpace(req.PlatformID)
	if name == "" || platformID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "name and platformId are required"})
		return
	}

	now := time.Now().UTC()
	account := model.MediaAccount{
		ID:            fmt.Sprintf("acc_%d", now.UnixNano()),
		WorkspaceID:   workspaceID,
		PlatformID:    platformID,
		Name:          name,
		ExternalID:    strings.TrimSpace(req.ExternalID),
		Status:        "connected",
		LastCheckedAt: now,
	}

	h.mu.Lock()
	h.accounts = append([]model.MediaAccount{account}, h.accounts...)
	h.mu.Unlock()
	c.JSON(http.StatusCreated, account)
}

func (h *WorkspaceHandler) ListContents(c *gin.Context) {
	workspaceID, ok := h.authorizedWorkspaceID(c)
	if !ok {
		return
	}

	h.mu.RLock()
	items := filterByWorkspace(h.contents, workspaceID, func(item model.Content) string { return item.WorkspaceID })
	h.mu.RUnlock()
	c.JSON(http.StatusOK, gin.H{"items": items})
}

func (h *WorkspaceHandler) CreateContent(c *gin.Context) {
	workspaceID, ok := h.authorizedWorkspaceID(c)
	if !ok {
		return
	}

	var req createContentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	title := strings.TrimSpace(req.Title)
	if title == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "title is required"})
		return
	}

	now := time.Now().UTC()
	content := model.Content{
		ID:              fmt.Sprintf("cnt_%d", now.UnixNano()),
		WorkspaceID:     workspaceID,
		KnowledgeBaseID: strings.TrimSpace(req.KnowledgeBaseID),
		Title:           title,
		Summary:         strings.TrimSpace(req.Summary),
		Body:            strings.TrimSpace(req.Body),
		Keywords:        cleanKeywords(req.Keywords),
		Status:          model.ContentDraft,
		Author:          defaultString(strings.TrimSpace(req.Author), "Current User"),
		Source:          "manual",
		UpdatedAt:       now,
	}

	h.mu.Lock()
	h.contents = append([]model.Content{content}, h.contents...)
	h.mu.Unlock()
	c.JSON(http.StatusCreated, content)
}

func (h *WorkspaceHandler) GenerateContent(c *gin.Context) {
	workspaceID, ok := h.authorizedWorkspaceID(c)
	if !ok {
		return
	}

	var req generateContentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	keywords := cleanKeywords(req.Keywords)
	if len(keywords) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "at least one keyword is required"})
		return
	}

	now := time.Now().UTC()
	knowledgeSummary := h.knowledgeSummary(workspaceID, req.KnowledgeBaseID)
	title := fmt.Sprintf("%s：从策略到执行", keywords[0])
	summary := fmt.Sprintf("围绕 %s 的 mock AI 草稿，已结合当前工作区知识库上下文。", strings.Join(keywords, "、"))
	body := fmt.Sprintf("关键词：%s\n\n知识库上下文：%s\n\n这是一篇 mock AI 生成草稿，用于先打通生成、编辑、排程和发布流程。后续接入真实模型时，将在这里替换 AI Provider。", strings.Join(keywords, "、"), knowledgeSummary)

	content := model.Content{
		ID:              fmt.Sprintf("cnt_%d", now.UnixNano()),
		WorkspaceID:     workspaceID,
		KnowledgeBaseID: strings.TrimSpace(req.KnowledgeBaseID),
		Title:           title,
		Summary:         summary,
		Body:            body,
		Keywords:        keywords,
		Status:          model.ContentDraft,
		Author:          "Mock AI",
		Source:          defaultString(strings.TrimSpace(req.ContentType), "mock_ai"),
		UpdatedAt:       now,
	}

	h.mu.Lock()
	h.contents = append([]model.Content{content}, h.contents...)
	h.mu.Unlock()
	c.JSON(http.StatusCreated, content)
}

func (h *WorkspaceHandler) ListPublishSchedules(c *gin.Context) {
	workspaceID, ok := h.authorizedWorkspaceID(c)
	if !ok {
		return
	}

	h.mu.RLock()
	items := filterByWorkspace(h.schedules, workspaceID, func(item model.PublishSchedule) string { return item.WorkspaceID })
	h.mu.RUnlock()
	c.JSON(http.StatusOK, gin.H{"items": items})
}

func (h *WorkspaceHandler) CreatePublishSchedule(c *gin.Context) {
	workspaceID, ok := h.authorizedWorkspaceID(c)
	if !ok {
		return
	}

	var req createPublishScheduleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	contentID := strings.TrimSpace(req.ContentID)
	accountID := strings.TrimSpace(req.MediaAccountID)
	if contentID == "" || accountID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "contentId and mediaAccountId are required"})
		return
	}

	now := time.Now().UTC()
	nextRunAt := req.NextRunAt
	if nextRunAt.IsZero() {
		nextRunAt = now.Add(time.Hour)
	}
	frequency := req.Frequency
	if frequency == "" {
		frequency = model.PublishScheduleOnce
	}

	h.mu.Lock()
	defer h.mu.Unlock()

	if !hasContent(h.contents, workspaceID, contentID) {
		c.JSON(http.StatusNotFound, gin.H{"error": "content not found"})
		return
	}
	if !hasMediaAccount(h.accounts, workspaceID, accountID) {
		c.JSON(http.StatusNotFound, gin.H{"error": "media account not found"})
		return
	}

	schedule := model.PublishSchedule{
		ID:             fmt.Sprintf("sch_%d", now.UnixNano()),
		WorkspaceID:    workspaceID,
		Name:           defaultString(strings.TrimSpace(req.Name), "未命名发布计划"),
		ContentID:      contentID,
		MediaAccountID: accountID,
		Frequency:      frequency,
		NextRunAt:      nextRunAt.UTC(),
		Enabled:        true,
		CreatedAt:      now,
	}
	job := model.PublishJob{
		ID:             fmt.Sprintf("job_%d", now.UnixNano()),
		WorkspaceID:    workspaceID,
		ScheduleID:     schedule.ID,
		ContentID:      contentID,
		MediaAccountID: accountID,
		Status:         model.PublishJobQueued,
		ScheduledAt:    schedule.NextRunAt,
		LastMessage:    "Waiting for scheduled publish window.",
	}

	h.schedules = append([]model.PublishSchedule{schedule}, h.schedules...)
	h.jobs = append([]model.PublishJob{job}, h.jobs...)
	c.JSON(http.StatusCreated, gin.H{"schedule": schedule, "job": job})
}

func (h *WorkspaceHandler) ListPublishJobs(c *gin.Context) {
	workspaceID, ok := h.authorizedWorkspaceID(c)
	if !ok {
		return
	}

	h.mu.RLock()
	items := filterByWorkspace(h.jobs, workspaceID, func(item model.PublishJob) string { return item.WorkspaceID })
	h.mu.RUnlock()
	c.JSON(http.StatusOK, gin.H{"items": items})
}

func (h *WorkspaceHandler) AdminOverview(c *gin.Context) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	c.JSON(http.StatusOK, gin.H{
		"userCount":          len(h.users),
		"workspaceCount":     len(h.workspaces),
		"memberCount":        len(h.members),
		"mediaPlatformCount": len(h.platforms),
		"mediaAccountCount":  len(h.accounts),
		"contentCount":       len(h.contents),
		"publishJobCount":    len(h.jobs),
	})
}

func (h *WorkspaceHandler) AdminListUsers(c *gin.Context) {
	h.mu.RLock()
	items := append([]model.User(nil), h.users...)
	h.mu.RUnlock()
	c.JSON(http.StatusOK, gin.H{"items": items})
}

func (h *WorkspaceHandler) AdminListWorkspaces(c *gin.Context) {
	h.mu.RLock()
	items := append([]model.Workspace(nil), h.workspaces...)
	h.mu.RUnlock()
	c.JSON(http.StatusOK, gin.H{"items": items})
}

func (h *WorkspaceHandler) AdminListWorkspaceMembers(c *gin.Context) {
	h.mu.RLock()
	items := append([]model.WorkspaceMember(nil), h.members...)
	h.mu.RUnlock()
	c.JSON(http.StatusOK, gin.H{"items": items})
}

func (h *WorkspaceHandler) AdminListMediaPlatforms(c *gin.Context) {
	h.ListMediaPlatforms(c)
}

func (h *WorkspaceHandler) AdminCreateMediaPlatform(c *gin.Context) {
	var req createMediaPlatformRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	name := strings.TrimSpace(req.Name)
	platformType := strings.TrimSpace(req.Type)
	if name == "" || platformType == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "name and type are required"})
		return
	}

	platform := model.MediaPlatform{
		ID:                 fmt.Sprintf("plt_%d", time.Now().UTC().UnixNano()),
		Name:               name,
		Type:               platformType,
		Enabled:            req.Enabled,
		SupportsArticle:    req.SupportsArticle,
		SupportsImage:      req.SupportsImage,
		SupportsScheduling: req.SupportsScheduling,
		CredentialFields:   cleanKeywords(req.CredentialFields),
	}

	h.mu.Lock()
	h.platforms = append([]model.MediaPlatform{platform}, h.platforms...)
	h.mu.Unlock()
	c.JSON(http.StatusCreated, platform)
}

func (h *WorkspaceHandler) AdminListMediaAccounts(c *gin.Context) {
	h.mu.RLock()
	items := append([]model.MediaAccount(nil), h.accounts...)
	h.mu.RUnlock()
	c.JSON(http.StatusOK, gin.H{"items": items})
}

func (h *WorkspaceHandler) requirePlatformAdmin() gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := middleware.CurrentUserID(c)

		h.mu.RLock()
		user, ok := h.userByID(userID)
		h.mu.RUnlock()

		if !ok || !user.IsPlatformAdmin {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "platform admin access required"})
			return
		}

		c.Next()
	}
}

func (h *WorkspaceHandler) authorizedWorkspaceID(c *gin.Context) (string, bool) {
	userID := middleware.CurrentUserID(c)
	workspaceID := middleware.CurrentWorkspaceID(c)

	h.mu.RLock()
	defer h.mu.RUnlock()

	if workspaceID == "" {
		workspaces := h.workspacesForUser(userID)
		if len(workspaces) == 0 {
			c.JSON(http.StatusForbidden, gin.H{"error": "workspace access denied"})
			return "", false
		}
		return workspaces[0].ID, true
	}

	if !h.userCanAccessWorkspace(userID, workspaceID) {
		c.JSON(http.StatusForbidden, gin.H{"error": "workspace access denied"})
		return "", false
	}

	return workspaceID, true
}

func (h *WorkspaceHandler) userByID(userID string) (model.User, bool) {
	for _, user := range h.users {
		if user.ID == userID {
			return user, true
		}
	}
	return model.User{}, false
}

func (h *WorkspaceHandler) workspacesForUser(userID string) []model.Workspace {
	workspaceIDs := map[string]bool{}
	for _, member := range h.members {
		if member.UserID == userID {
			workspaceIDs[member.WorkspaceID] = true
		}
	}

	items := make([]model.Workspace, 0, len(workspaceIDs))
	for _, workspace := range h.workspaces {
		if workspaceIDs[workspace.ID] {
			items = append(items, workspace)
		}
	}
	return items
}

func (h *WorkspaceHandler) userCanAccessWorkspace(userID, workspaceID string) bool {
	for _, member := range h.members {
		if member.UserID == userID && member.WorkspaceID == workspaceID {
			return true
		}
	}
	return false
}

func (h *WorkspaceHandler) bumpKnowledgeBaseCount(workspaceID, knowledgeBaseID string, delta int) {
	for index := range h.knowledgeBases {
		item := &h.knowledgeBases[index]
		if item.WorkspaceID == workspaceID && item.ID == knowledgeBaseID {
			item.ItemCount += delta
			item.UpdatedAt = time.Now().UTC()
			return
		}
	}
}

func (h *WorkspaceHandler) knowledgeSummary(workspaceID, knowledgeBaseID string) string {
	h.mu.RLock()
	defer h.mu.RUnlock()

	parts := []string{}
	for _, item := range h.knowledgeItems {
		if item.WorkspaceID != workspaceID || !item.Enabled {
			continue
		}
		if knowledgeBaseID != "" && item.KnowledgeBaseID != knowledgeBaseID {
			continue
		}
		parts = append(parts, item.Title+" - "+item.Content)
	}

	if len(parts) == 0 {
		return "当前工作区尚未维护可用知识条目。"
	}
	return strings.Join(parts, "；")
}

func filterByWorkspace[T any](items []T, workspaceID string, getWorkspaceID func(T) string) []T {
	filtered := make([]T, 0, len(items))
	for _, item := range items {
		if getWorkspaceID(item) == workspaceID {
			filtered = append(filtered, item)
		}
	}
	return filtered
}

func countContents(contents []model.Content, status model.ContentStatus) int {
	total := 0
	for _, content := range contents {
		if content.Status == status {
			total++
		}
	}
	return total
}

func countJobs(jobs []model.PublishJob, status model.PublishJobStatus) int {
	total := 0
	for _, job := range jobs {
		if job.Status == status {
			total++
		}
	}
	return total
}

func hasContent(contents []model.Content, workspaceID, contentID string) bool {
	for _, content := range contents {
		if content.WorkspaceID == workspaceID && content.ID == contentID {
			return true
		}
	}
	return false
}

func hasMediaAccount(accounts []model.MediaAccount, workspaceID, accountID string) bool {
	for _, account := range accounts {
		if account.WorkspaceID == workspaceID && account.ID == accountID {
			return true
		}
	}
	return false
}

func cleanKeywords(values []string) []string {
	keywords := make([]string, 0, len(values))
	for _, value := range values {
		keyword := strings.TrimSpace(value)
		if keyword != "" {
			keywords = append(keywords, keyword)
		}
	}
	return keywords
}

func defaultString(value, fallback string) string {
	if value == "" {
		return fallback
	}
	return value
}

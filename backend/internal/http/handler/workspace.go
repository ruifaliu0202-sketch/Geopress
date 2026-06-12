package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"sync"
	"time"
	"unicode"

	"geopress/backend/internal/ai"
	"geopress/backend/internal/database"
	"geopress/backend/internal/http/middleware"
	publishing "geopress/backend/internal/integration/publisher"
	"geopress/backend/internal/integration/xiaohongshu"
	"geopress/backend/internal/model"

	"github.com/gin-gonic/gin"
)

type WorkspaceHandler struct {
	mu             sync.RWMutex
	db             *database.DB
	aiConfig       *ai.RuntimeConfig
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
	generations    []model.GenerationRequest
	loginSessions  map[string]mediaAccountLoginSession
	browserLogin   xiaohongshu.BrowserLoginService
}

type mediaAccountLoginSession struct {
	ID         string
	ExpiresAt  time.Time
	ProfileDir string
	LoginURL   string
	StateFile  string
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
	PlatformID  string `json:"platformId"`
	Name        string `json:"name"`
	ExternalID  string `json:"externalId"`
	LoginMethod string `json:"loginMethod"`
	PhoneNumber string `json:"phoneNumber"`
}

type startMediaAccountBrowserLoginRequest struct {
}

type completeMediaAccountBrowserLoginRequest struct {
	SessionID string `json:"sessionId"`
}

type generateContentRequest struct {
	Keywords        []string `json:"keywords"`
	ContentType     string   `json:"contentType"`
	KnowledgeBaseID string   `json:"knowledgeBaseId"`
	PublishFormatID string   `json:"publishFormatId"`
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

type preparePublishRequest struct {
	ContentID       string   `json:"contentId"`
	MediaAccountID  string   `json:"mediaAccountId"`
	PublishFormatID string   `json:"publishFormatId"`
	AssetPaths      []string `json:"assetPaths"`
	RunNow          bool     `json:"runNow"`
}

type confirmPublishRequest struct {
	ExternalURL string `json:"externalUrl"`
	Message     string `json:"message"`
}

type runPublishJobRequest struct {
	AssetPaths   []string                `json:"assetPaths"`
	PreparedPost publishing.PreparedPost `json:"preparedPost"`
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

type updateAIConfigRequest struct {
	Provider              string `json:"provider"`
	OpenAIAPIKey          string `json:"openAIAPIKey"`
	OpenAIBaseURL         string `json:"openAIBaseUrl"`
	OpenAIModel           string `json:"openAIModel"`
	RequestTimeoutSeconds int    `json:"requestTimeoutSeconds"`
	ClearAPIKey           bool   `json:"clearAPIKey"`
}

func NewWorkspaceHandler(db *database.DB, aiConfig *ai.RuntimeConfig) *WorkspaceHandler {
	if aiConfig == nil {
		aiConfig = ai.NewRuntimeConfig(ai.Config{Provider: ai.ProviderMock})
	}

	now := time.Now().UTC()
	expiresAt := now.AddDate(0, 2, 0)

	h := &WorkspaceHandler{
		db:            db,
		aiConfig:      aiConfig,
		loginSessions: map[string]mediaAccountLoginSession{},
		browserLogin:  xiaohongshu.NewPlaywrightBrowserLoginService(),
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
			{
				ID:                 "plt_xiaohongshu",
				Name:               "小红书",
				Type:               xiaohongshu.PlatformType,
				Enabled:            true,
				SupportsArticle:    true,
				SupportsImage:      true,
				SupportsScheduling: false,
				CredentialFields:   []string{"qrLogin"},
			},
			{
				ID:                 "plt_local_publisher",
				Name:               "本机发布平台",
				Type:               "local_publisher",
				Enabled:            true,
				SupportsArticle:    true,
				SupportsImage:      true,
				SupportsScheduling: false,
				CredentialFields:   []string{"phoneNumber"},
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
			{
				ID:             "acc_xhs_acme",
				WorkspaceID:    "wks_acme",
				PlatformID:     "plt_xiaohongshu",
				Name:           "Acme 小红书",
				ExternalID:     "AcmeGrowth",
				LoginMethod:    "qr",
				CredentialMeta: map[string]string{},
				Status:         "pending_login",
				LastCheckedAt:  now.Add(-90 * time.Minute),
			},
			{
				ID:             "acc_xhs_personal",
				WorkspaceID:    "wks_personal",
				PlatformID:     "plt_xiaohongshu",
				Name:           "Ava 小红书",
				ExternalID:     "AvaCreator",
				LoginMethod:    "qr",
				CredentialMeta: map[string]string{},
				Status:         "pending_login",
				LastCheckedAt:  now.Add(-3 * time.Hour),
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

	h.seedDatabase(context.Background())
	return h
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
	protected.POST("/media-accounts/:accountId/browser-login/start", h.StartMediaAccountBrowserLogin)
	protected.POST("/media-accounts/:accountId/browser-login/complete", h.CompleteMediaAccountBrowserLogin)
	protected.GET("/contents", h.ListContents)
	protected.POST("/contents", h.CreateContent)
	protected.POST("/contents/generate", h.GenerateContent)
	protected.GET("/publish-schedules", h.ListPublishSchedules)
	protected.POST("/publish-schedules", h.CreatePublishSchedule)
	protected.GET("/publish-jobs", h.ListPublishJobs)
	protected.POST("/publish/prepare", h.PreparePublish)
	protected.POST("/publish-jobs/:jobId/run", h.RunPublishJob)
	protected.POST("/publish-jobs/:jobId/confirm", h.ConfirmPublishJob)

	admin := protected.Group("/admin")
	admin.Use(h.requirePlatformAdmin())
	admin.GET("/overview", h.AdminOverview)
	admin.GET("/users", h.AdminListUsers)
	admin.GET("/workspaces", h.AdminListWorkspaces)
	admin.GET("/workspace-members", h.AdminListWorkspaceMembers)
	admin.GET("/media-platforms", h.AdminListMediaPlatforms)
	admin.POST("/media-platforms", h.AdminCreateMediaPlatform)
	admin.GET("/media-accounts", h.AdminListMediaAccounts)
	admin.GET("/ai-config", h.AdminGetAIConfig)
	admin.PUT("/ai-config", h.AdminUpdateAIConfig)
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
	h.mu.RLock()
	platform, platformOK := h.mediaPlatformByID(platformID)
	h.mu.RUnlock()
	if !platformOK {
		c.JSON(http.StatusNotFound, gin.H{"error": "media platform not found"})
		return
	}
	if !platform.Enabled {
		c.JSON(http.StatusBadRequest, gin.H{"error": "media platform is not enabled"})
		return
	}

	phoneNumber, phoneOK := cleanPhoneNumber(req.PhoneNumber)
	if !phoneOK {
		c.JSON(http.StatusBadRequest, gin.H{"error": "phoneNumber is invalid"})
		return
	}

	requiresPhone := platformRequiresCredential(platform, "phoneNumber")
	requiresQR := platformRequiresCredential(platform, "qrLogin")
	loginMethod := strings.TrimSpace(strings.ToLower(req.LoginMethod))
	if loginMethod == "" {
		if requiresQR {
			loginMethod = "qr"
		} else if phoneNumber != "" || requiresPhone {
			loginMethod = "phone"
		} else {
			loginMethod = "manual"
		}
	}
	if loginMethod != "manual" && loginMethod != "phone" && loginMethod != "qr" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "loginMethod is invalid"})
		return
	}
	if requiresQR && loginMethod != "qr" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "qr login is required for this media platform"})
		return
	}
	if requiresPhone && loginMethod != "phone" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "phone login is required for this media platform"})
		return
	}
	if loginMethod == "phone" && phoneNumber == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "phoneNumber is required for phone login"})
		return
	}

	credentialMeta := map[string]string{}
	if loginMethod == "phone" && phoneNumber != "" {
		credentialMeta["phoneNumber"] = phoneNumber
	}
	status := "connected"
	if loginMethod == "phone" || loginMethod == "qr" {
		status = "pending_login"
	}

	account := model.MediaAccount{
		ID:             fmt.Sprintf("acc_%d", now.UnixNano()),
		WorkspaceID:    workspaceID,
		PlatformID:     platformID,
		Name:           name,
		ExternalID:     strings.TrimSpace(req.ExternalID),
		LoginMethod:    loginMethod,
		CredentialMeta: credentialMeta,
		Status:         status,
		LastCheckedAt:  now,
	}

	h.mu.Lock()
	h.accounts = append([]model.MediaAccount{account}, h.accounts...)
	h.mu.Unlock()
	c.JSON(http.StatusCreated, account)
}

func (h *WorkspaceHandler) StartMediaAccountBrowserLogin(c *gin.Context) {
	h.startMediaAccountBrowserLogin(c)
}

func (h *WorkspaceHandler) CompleteMediaAccountBrowserLogin(c *gin.Context) {
	h.completeMediaAccountBrowserLogin(c)
}

func (h *WorkspaceHandler) startMediaAccountBrowserLogin(c *gin.Context) {
	workspaceID, ok := h.authorizedWorkspaceID(c)
	if !ok {
		return
	}

	accountID := strings.TrimSpace(c.Param("accountId"))
	if accountID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "account id is required"})
		return
	}

	var req startMediaAccountBrowserLoginRequest
	_ = c.ShouldBindJSON(&req)

	now := time.Now().UTC()
	expiresAt := now.Add(5 * time.Minute)
	sessionID := fmt.Sprintf("xhs_login_%d", now.UnixNano())
	profileDir := browserProfilePath(workspaceID, accountID)
	stateFile := xiaohongshu.BrowserLoginStateFile(profileDir)

	h.mu.RLock()
	account, accountOK := h.mediaAccountByID(workspaceID, accountID)
	if !accountOK {
		h.mu.RUnlock()
		c.JSON(http.StatusNotFound, gin.H{"error": "media account not found"})
		return
	}
	platform, platformOK := h.mediaPlatformByID(account.PlatformID)
	h.mu.RUnlock()

	if !platformOK || !supportsBrowserLogin(platform.Type) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "media account does not support browser login"})
		return
	}
	if account.LoginMethod != "qr" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "media account login method is not qr"})
		return
	}

	loginResult, err := h.browserLogin.Start(c.Request.Context(), xiaohongshu.BrowserLoginStartRequest{
		WorkspaceID: workspaceID,
		AccountID:   accountID,
		SessionID:   sessionID,
		ProfileDir:  profileDir,
		LoginURL:    xiaohongshu.DefaultLoginURL,
		StateFile:   stateFile,
	})
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}

	var updated model.MediaAccount
	h.mu.Lock()
	defer h.mu.Unlock()
	for index := range h.accounts {
		if h.accounts[index].WorkspaceID != workspaceID || h.accounts[index].ID != accountID {
			continue
		}
		account := &h.accounts[index]
		if account.CredentialMeta == nil {
			account.CredentialMeta = map[string]string{}
		}

		account.CredentialMeta["qrLoginStartedAt"] = loginResult.StartedAt.Format(time.RFC3339)
		account.CredentialMeta["browserSessionMode"] = "playwright_persistent_context"
		account.CredentialMeta["browserProfile"] = loginResult.ProfileDir
		account.CredentialMeta["browserLoginStateFile"] = loginResult.StateFile
		account.CredentialMeta["loginSessionId"] = loginResult.SessionID
		account.Status = "qr_waiting"
		if loginResult.AlreadyLoggedIn {
			account.Status = "connected"
			account.CredentialMeta["qrLoginCompletedAt"] = loginResult.StartedAt.Format(time.RFC3339)
		}
		account.LastCheckedAt = now
		updated = *account
		h.loginSessions[loginSessionKey(workspaceID, accountID)] = mediaAccountLoginSession{
			ID:         loginResult.SessionID,
			ExpiresAt:  expiresAt,
			ProfileDir: loginResult.ProfileDir,
			LoginURL:   loginResult.LoginURL,
			StateFile:  loginResult.StateFile,
		}
		break
	}
	if updated.ID == "" {
		c.JSON(http.StatusNotFound, gin.H{"error": "media account not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"account":          updated,
		"expiresAt":        expiresAt,
		"mode":             "playwright_persistent_context",
		"qrScreenshotData": loginResult.QRScreenshotData,
		"qrLoginUrl":       loginResult.PageURL,
		"sessionId":        loginResult.SessionID,
		"browserProfile":   loginResult.ProfileDir,
		"stateFile":        loginResult.StateFile,
	})
}

func (h *WorkspaceHandler) completeMediaAccountBrowserLogin(c *gin.Context) {
	workspaceID, ok := h.authorizedWorkspaceID(c)
	if !ok {
		return
	}

	accountID := strings.TrimSpace(c.Param("accountId"))
	if accountID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "account id is required"})
		return
	}

	var req completeMediaAccountBrowserLoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	sessionID := strings.TrimSpace(req.SessionID)
	if sessionID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "sessionId is required"})
		return
	}

	now := time.Now().UTC()
	key := loginSessionKey(workspaceID, accountID)

	h.mu.RLock()
	loginSession, sessionOK := h.loginSessions[key]
	if !sessionOK {
		h.mu.RUnlock()
		c.JSON(http.StatusBadRequest, gin.H{"error": "qr login session was not started"})
		return
	}
	if now.After(loginSession.ExpiresAt) {
		h.mu.RUnlock()
		h.mu.Lock()
		delete(h.loginSessions, key)
		h.mu.Unlock()
		c.JSON(http.StatusBadRequest, gin.H{"error": "qr login session expired"})
		return
	}
	if loginSession.ID != sessionID {
		h.mu.RUnlock()
		c.JSON(http.StatusBadRequest, gin.H{"error": "qr login session is invalid"})
		return
	}
	account, accountOK := h.mediaAccountByID(workspaceID, accountID)
	if !accountOK {
		h.mu.RUnlock()
		c.JSON(http.StatusNotFound, gin.H{"error": "media account not found"})
		return
	}
	platform, platformOK := h.mediaPlatformByID(account.PlatformID)
	h.mu.RUnlock()

	if !platformOK || !supportsBrowserLogin(platform.Type) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "media account does not support browser login"})
		return
	}

	loginResult, err := h.browserLogin.Complete(c.Request.Context(), xiaohongshu.BrowserLoginCompleteRequest{
		WorkspaceID: workspaceID,
		AccountID:   accountID,
		SessionID:   sessionID,
		ProfileDir:  loginSession.ProfileDir,
		LoginURL:    loginSession.LoginURL,
		StateFile:   loginSession.StateFile,
	})
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	h.mu.Lock()
	defer h.mu.Unlock()
	for index := range h.accounts {
		account := &h.accounts[index]
		if account.WorkspaceID != workspaceID || account.ID != accountID {
			continue
		}
		if account.CredentialMeta == nil {
			account.CredentialMeta = map[string]string{}
		}

		account.CredentialMeta["qrLoginCompletedAt"] = loginResult.CompletedAt.Format(time.RFC3339)
		account.CredentialMeta["browserSessionMode"] = "playwright_persistent_context"
		account.CredentialMeta["browserProfile"] = loginResult.ProfileDir
		account.CredentialMeta["browserLoginStateFile"] = loginResult.StateFile
		account.CredentialMeta["loginSessionId"] = sessionID
		account.Status = "connected"
		account.LastCheckedAt = now
		delete(h.loginSessions, key)

		c.JSON(http.StatusOK, *account)
		return
	}

	c.JSON(http.StatusNotFound, gin.H{"error": "media account not found"})
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
	userID := middleware.CurrentUserID(c)

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
	knowledgeBaseID := strings.TrimSpace(req.KnowledgeBaseID)
	contentType := publishFormatOrContentType(req.PublishFormatID, req.ContentType, ai.FormatGenericArticle)
	skill := ai.SelectWritingSkill(contentType)
	publishFormat := ai.SelectPublishFormat(skill.ContentType)

	h.mu.RLock()
	workspace, _ := h.workspaceByID(workspaceID)
	chunks := h.retrieveKnowledgeChunksLocked(workspaceID, knowledgeBaseID, keywords, 8)
	h.mu.RUnlock()

	aiReq := ai.GenerateRequest{
		WorkspaceID:     workspaceID,
		UserID:          userID,
		KnowledgeBaseID: knowledgeBaseID,
		ContentType:     skill.ContentType,
		Keywords:        keywords,
		Workspace: ai.WorkspaceContext{
			Name:     workspace.Name,
			Type:     workspace.Type,
			Industry: workspace.Industry,
			Language: workspace.Language,
			Tone:     workspace.Tone,
		},
		Skill:           skill,
		PublishFormat:   publishFormat,
		KnowledgeChunks: chunks,
	}

	provider := h.aiConfig.Provider()
	response, err := provider.Generate(c.Request.Context(), aiReq)
	if err != nil {
		h.recordGeneration(c.Request.Context(), model.GenerationRequest{
			ID:                    fmt.Sprintf("gen_%d", now.UnixNano()),
			WorkspaceID:           workspaceID,
			UserID:                userID,
			KnowledgeBaseID:       knowledgeBaseID,
			Provider:              provider.Name(),
			Model:                 provider.Model(),
			ContentType:           skill.ContentType,
			Keywords:              keywords,
			Prompt:                encodeJSON(ai.BuildPrompt(aiReq)),
			PromptVersion:         ai.PromptVersion,
			SkillID:               skill.ID,
			SkillVersion:          skill.Version,
			RetrievedKnowledgeIDs: knowledgeChunkIDs(chunks),
			Status:                "failed",
			ErrorMessage:          err.Error(),
			CreatedAt:             now,
		})
		c.JSON(http.StatusBadGateway, gin.H{"error": "content generation failed"})
		return
	}

	if err := response.Draft.Validate(); err != nil {
		h.recordGeneration(c.Request.Context(), model.GenerationRequest{
			ID:                    fmt.Sprintf("gen_%d", now.UnixNano()),
			WorkspaceID:           workspaceID,
			UserID:                userID,
			KnowledgeBaseID:       knowledgeBaseID,
			Provider:              response.Provider,
			Model:                 response.Model,
			ContentType:           skill.ContentType,
			Keywords:              keywords,
			Prompt:                encodeJSON(response.Prompt),
			PromptVersion:         response.PromptVersion,
			SkillID:               response.SkillID,
			SkillVersion:          response.SkillVersion,
			RetrievedKnowledgeIDs: response.RetrievedIDs,
			RawOutput:             string(response.RawOutput),
			Status:                "failed",
			ErrorMessage:          err.Error(),
			CreatedAt:             now,
		})
		c.JSON(http.StatusBadGateway, gin.H{"error": "generated content is invalid"})
		return
	}

	content := model.Content{
		ID:              fmt.Sprintf("cnt_%d", now.UnixNano()),
		WorkspaceID:     workspaceID,
		KnowledgeBaseID: knowledgeBaseID,
		Title:           strings.TrimSpace(response.Draft.Title),
		Summary:         strings.TrimSpace(response.Draft.Summary),
		Body:            strings.TrimSpace(response.Draft.Body),
		Keywords:        cleanKeywords(response.Draft.Keywords),
		Status:          model.ContentDraft,
		Author:          "AI Writer",
		Source:          "ai_" + defaultString(response.Provider, "mock") + ":" + publishFormat.ID,
		UpdatedAt:       now,
	}

	h.mu.Lock()
	h.contents = append([]model.Content{content}, h.contents...)
	h.mu.Unlock()

	if err := h.saveContent(c.Request.Context(), content); err != nil {
		log.Printf("generated content was not persisted: %v", err)
	}

	h.recordGeneration(c.Request.Context(), model.GenerationRequest{
		ID:                    fmt.Sprintf("gen_%d", now.UnixNano()),
		WorkspaceID:           workspaceID,
		UserID:                userID,
		KnowledgeBaseID:       knowledgeBaseID,
		ContentID:             content.ID,
		Provider:              response.Provider,
		Model:                 response.Model,
		ContentType:           skill.ContentType,
		Keywords:              keywords,
		Prompt:                encodeJSON(response.Prompt),
		PromptVersion:         response.PromptVersion,
		SkillID:               response.SkillID,
		SkillVersion:          response.SkillVersion,
		RetrievedKnowledgeIDs: response.RetrievedIDs,
		RawOutput:             string(response.RawOutput),
		ParsedOutput:          encodeJSON(response.Draft),
		InputTokens:           response.TokenUsage.InputTokens,
		OutputTokens:          response.TokenUsage.OutputTokens,
		TotalTokens:           response.TokenUsage.TotalTokens,
		Status:                "succeeded",
		CreatedAt:             now,
	})

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

func (h *WorkspaceHandler) PreparePublish(c *gin.Context) {
	workspaceID, ok := h.authorizedWorkspaceID(c)
	if !ok {
		return
	}

	var req preparePublishRequest
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
	h.mu.RLock()
	workspace, workspaceOK := h.workspaceByID(workspaceID)
	content, contentOK := h.contentByID(workspaceID, contentID)
	account, accountOK := h.mediaAccountByID(workspaceID, accountID)
	platform, platformOK := h.mediaPlatformByID(account.PlatformID)
	h.mu.RUnlock()

	if !workspaceOK {
		c.JSON(http.StatusNotFound, gin.H{"error": "workspace not found"})
		return
	}
	if !contentOK {
		c.JSON(http.StatusNotFound, gin.H{"error": "content not found"})
		return
	}
	if !accountOK {
		c.JSON(http.StatusNotFound, gin.H{"error": "media account not found"})
		return
	}
	if !platformOK || !platform.Enabled {
		c.JSON(http.StatusBadRequest, gin.H{"error": "media platform is not enabled"})
		return
	}

	publisher, supported := publisherForPlatform(platform.Type)
	if !supported {
		c.JSON(http.StatusBadRequest, gin.H{"error": "media platform does not support publish preparation"})
		return
	}

	publishFormatID := publishFormatOrContentType(req.PublishFormatID, "", ai.FormatXiaohongshuLongArticle)
	prepared, err := publisher.Prepare(c.Request.Context(), publishing.PrepareRequest{
		Workspace:       workspace,
		Content:         content,
		Account:         account,
		Platform:        platform,
		PublishFormatID: publishFormatID,
		RequestedAt:     now,
	})
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	job := model.PublishJob{
		ID:             fmt.Sprintf("job_%d", now.UnixNano()),
		WorkspaceID:    workspaceID,
		ContentID:      content.ID,
		MediaAccountID: account.ID,
		Status:         model.PublishJobManual,
		ScheduledAt:    now,
		LastMessage:    "小红书长文发布内容已生成，等待确认后通过浏览器发布。",
	}

	h.mu.Lock()
	h.jobs = append([]model.PublishJob{job}, h.jobs...)
	h.updateContentStatusLocked(workspaceID, content.ID, model.ContentScheduled)
	h.mu.Unlock()

	if req.RunNow {
		result, err := h.runPublish(c.Request.Context(), workspace, account, platform, job.ID, prepared, req.AssetPaths)
		if err != nil {
			c.JSON(http.StatusBadGateway, gin.H{"error": err.Error(), "job": h.jobSnapshot(workspaceID, job.ID), "preparedPost": prepared})
			return
		}
		c.JSON(http.StatusCreated, gin.H{
			"job":           h.jobSnapshot(workspaceID, job.ID),
			"preparedPost":  prepared,
			"publishResult": result,
		})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"job":          job,
		"preparedPost": prepared,
	})
}

func (h *WorkspaceHandler) RunPublishJob(c *gin.Context) {
	workspaceID, ok := h.authorizedWorkspaceID(c)
	if !ok {
		return
	}

	jobID := strings.TrimSpace(c.Param("jobId"))
	if jobID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "job id is required"})
		return
	}

	var req runPublishJobRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	h.mu.RLock()
	job, jobOK := h.publishJobByID(workspaceID, jobID)
	content, contentOK := h.contentByID(workspaceID, job.ContentID)
	account, accountOK := h.mediaAccountByID(workspaceID, job.MediaAccountID)
	platform, platformOK := h.mediaPlatformByID(account.PlatformID)
	workspace, workspaceOK := h.workspaceByID(workspaceID)
	h.mu.RUnlock()

	if !jobOK {
		c.JSON(http.StatusNotFound, gin.H{"error": "publish job not found"})
		return
	}
	if !contentOK || !accountOK || !platformOK || !workspaceOK {
		c.JSON(http.StatusBadRequest, gin.H{"error": "publish job references missing content or account"})
		return
	}

	publisher, supported := publisherForPlatform(platform.Type)
	if !supported {
		c.JSON(http.StatusBadRequest, gin.H{"error": "media platform does not support publish"})
		return
	}

	prepared := req.PreparedPost
	if strings.TrimSpace(prepared.Title) == "" || strings.TrimSpace(prepared.Body) == "" {
		var err error
		prepared, err = publisher.Prepare(c.Request.Context(), publishing.PrepareRequest{
			Workspace:       workspace,
			Content:         content,
			Account:         account,
			Platform:        platform,
			PublishFormatID: ai.FormatXiaohongshuLongArticle,
			RequestedAt:     time.Now().UTC(),
		})
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
	}
	if prepared.PlatformType == "" {
		prepared.PlatformType = platform.Type
	}
	if prepared.PlatformName == "" {
		prepared.PlatformName = platform.Name
	}
	if prepared.PublishFormatID == "" {
		prepared.PublishFormatID = ai.FormatXiaohongshuLongArticle
	}
	if prepared.PublishMode == "" {
		prepared.PublishMode = "long_article"
	}

	result, err := h.runPublish(c.Request.Context(), workspace, account, platform, jobID, prepared, req.AssetPaths)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error(), "job": h.jobSnapshot(workspaceID, jobID), "preparedPost": prepared})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"job":           h.jobSnapshot(workspaceID, jobID),
		"preparedPost":  prepared,
		"publishResult": result,
	})
}

func (h *WorkspaceHandler) ConfirmPublishJob(c *gin.Context) {
	workspaceID, ok := h.authorizedWorkspaceID(c)
	if !ok {
		return
	}

	jobID := strings.TrimSpace(c.Param("jobId"))
	if jobID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "job id is required"})
		return
	}

	var req confirmPublishRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	externalURL := strings.TrimSpace(req.ExternalURL)
	parsedURL, err := url.ParseRequestURI(externalURL)
	if err != nil || parsedURL.Scheme == "" || parsedURL.Host == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "valid externalUrl is required"})
		return
	}

	h.mu.Lock()
	defer h.mu.Unlock()

	for index := range h.jobs {
		job := &h.jobs[index]
		if job.WorkspaceID != workspaceID || job.ID != jobID {
			continue
		}

		job.Status = model.PublishJobSucceeded
		job.ExternalURL = externalURL
		job.LastMessage = defaultString(strings.TrimSpace(req.Message), "已人工确认发布完成。")
		h.updateContentStatusLocked(workspaceID, job.ContentID, model.ContentPublished)
		c.JSON(http.StatusOK, job)
		return
	}

	c.JSON(http.StatusNotFound, gin.H{"error": "publish job not found"})
}

func (h *WorkspaceHandler) runPublish(
	ctx context.Context,
	workspace model.Workspace,
	account model.MediaAccount,
	platform model.MediaPlatform,
	jobID string,
	prepared publishing.PreparedPost,
	assetPaths []string,
) (publishing.PublishResult, error) {
	workspaceID := workspace.ID
	h.mu.Lock()
	h.updateJobStatusLocked(workspaceID, jobID, model.PublishJobRunning, "", "正在通过已登录浏览器发布小红书长文。")
	h.mu.Unlock()

	publisher, supported := publisherForPlatform(prepared.PlatformType)
	if !supported {
		return publishing.PublishResult{}, fmt.Errorf("unsupported platform type: %s", prepared.PlatformType)
	}

	profileDir, stateFile := xiaohongshu.BrowserProfileMetadata(account, workspaceID)
	result, err := publisher.Publish(ctx, publishing.PublishRequest{
		Workspace:    workspace,
		Account:      account,
		Platform:     platform,
		PreparedPost: prepared,
		AssetPaths:   cleanKeywords(assetPaths),
		ProfileDir:   profileDir,
		StateFile:    stateFile,
	})
	if err != nil {
		h.mu.Lock()
		h.updateJobStatusLocked(workspaceID, jobID, model.PublishJobFailed, "", err.Error())
		h.mu.Unlock()
		return publishing.PublishResult{}, err
	}

	status := model.PublishJobManual
	message := defaultString(result.Message, "已打开浏览器并完成小红书发布准备。")
	if result.Status == "published" {
		status = model.PublishJobSucceeded
	} else if result.Status == "submitted_pending_verification" {
		status = model.PublishJobManual
		if strings.TrimSpace(message) == "" {
			message = "已尝试浏览器发布，但未检测到小红书明确成功提示，请人工核对后确认。"
		}
	}

	h.mu.Lock()
	job := h.updateJobStatusLocked(workspaceID, jobID, status, result.ExternalURL, message)
	if result.Status == "published" {
		h.updateContentStatusLocked(workspaceID, job.ContentID, model.ContentPublished)
	}
	h.mu.Unlock()

	return result, nil
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

func (h *WorkspaceHandler) AdminGetAIConfig(c *gin.Context) {
	c.JSON(http.StatusOK, h.aiConfig.Public())
}

func (h *WorkspaceHandler) AdminUpdateAIConfig(c *gin.Context) {
	var req updateAIConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	provider := strings.TrimSpace(strings.ToLower(req.Provider))
	if provider == "" {
		provider = ai.ProviderMock
	}
	if provider != ai.ProviderMock && provider != ai.ProviderOpenAI {
		c.JSON(http.StatusBadRequest, gin.H{"error": "unsupported AI provider"})
		return
	}

	updated := h.aiConfig.Update(ai.Config{
		Provider:       provider,
		OpenAIAPIKey:   strings.TrimSpace(req.OpenAIAPIKey),
		OpenAIBaseURL:  strings.TrimSpace(req.OpenAIBaseURL),
		OpenAIModel:    strings.TrimSpace(req.OpenAIModel),
		RequestTimeout: req.RequestTimeoutSeconds,
	}, req.ClearAPIKey)

	c.JSON(http.StatusOK, ai.NewRuntimeConfig(updated).Public())
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

func (h *WorkspaceHandler) seedDatabase(ctx context.Context) {
	dbCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	if err := h.db.SeedWorkspaceData(dbCtx, h.users, h.workspaces, h.members, h.knowledgeBases); err != nil {
		log.Printf("database seed failed, continuing in memory mode: %v", err)
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

func (h *WorkspaceHandler) workspaceByID(workspaceID string) (model.Workspace, bool) {
	for _, workspace := range h.workspaces {
		if workspace.ID == workspaceID {
			return workspace, true
		}
	}
	return model.Workspace{}, false
}

func (h *WorkspaceHandler) contentByID(workspaceID, contentID string) (model.Content, bool) {
	for _, content := range h.contents {
		if content.WorkspaceID == workspaceID && content.ID == contentID {
			return content, true
		}
	}
	return model.Content{}, false
}

func (h *WorkspaceHandler) mediaAccountByID(workspaceID, accountID string) (model.MediaAccount, bool) {
	for _, account := range h.accounts {
		if account.WorkspaceID == workspaceID && account.ID == accountID {
			return account, true
		}
	}
	return model.MediaAccount{}, false
}

func (h *WorkspaceHandler) mediaPlatformByID(platformID string) (model.MediaPlatform, bool) {
	for _, platform := range h.platforms {
		if platform.ID == platformID {
			return platform, true
		}
	}
	return model.MediaPlatform{}, false
}

func (h *WorkspaceHandler) publishJobByID(workspaceID, jobID string) (model.PublishJob, bool) {
	for _, job := range h.jobs {
		if job.WorkspaceID == workspaceID && job.ID == jobID {
			return job, true
		}
	}
	return model.PublishJob{}, false
}

func (h *WorkspaceHandler) jobSnapshot(workspaceID, jobID string) model.PublishJob {
	h.mu.RLock()
	defer h.mu.RUnlock()
	job, _ := h.publishJobByID(workspaceID, jobID)
	return job
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

func (h *WorkspaceHandler) updateContentStatusLocked(workspaceID, contentID string, status model.ContentStatus) {
	for index := range h.contents {
		content := &h.contents[index]
		if content.WorkspaceID == workspaceID && content.ID == contentID {
			content.Status = status
			content.UpdatedAt = time.Now().UTC()
			return
		}
	}
}

func (h *WorkspaceHandler) updateJobStatusLocked(workspaceID, jobID string, status model.PublishJobStatus, externalURL string, message string) model.PublishJob {
	for index := range h.jobs {
		job := &h.jobs[index]
		if job.WorkspaceID == workspaceID && job.ID == jobID {
			job.Status = status
			if externalURL != "" {
				job.ExternalURL = externalURL
			}
			if message != "" {
				job.LastMessage = message
			}
			return *job
		}
	}
	return model.PublishJob{}
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

func (h *WorkspaceHandler) retrieveKnowledgeChunksLocked(workspaceID, knowledgeBaseID string, keywords []string, limit int) []ai.KnowledgeChunk {
	type scoredChunk struct {
		score int
		item  model.KnowledgeItem
	}

	scored := []scoredChunk{}
	fallback := []model.KnowledgeItem{}
	for _, item := range h.knowledgeItems {
		if item.WorkspaceID != workspaceID || !item.Enabled {
			continue
		}
		if knowledgeBaseID != "" && item.KnowledgeBaseID != knowledgeBaseID {
			continue
		}
		fallback = append(fallback, item)
		score := knowledgeScore(item, keywords)
		if score > 0 {
			scored = append(scored, scoredChunk{score: score, item: item})
		}
	}

	if len(scored) == 0 {
		for _, item := range fallback {
			scored = append(scored, scoredChunk{score: 0, item: item})
		}
	}

	sort.SliceStable(scored, func(i, j int) bool {
		return scored[i].score > scored[j].score
	})

	if limit <= 0 || limit > len(scored) {
		limit = len(scored)
	}

	chunks := make([]ai.KnowledgeChunk, 0, limit)
	for _, item := range scored[:limit] {
		chunks = append(chunks, ai.KnowledgeChunk{
			ID:              item.item.ID,
			KnowledgeBaseID: item.item.KnowledgeBaseID,
			Type:            item.item.Type,
			Title:           item.item.Title,
			Content:         item.item.Content,
		})
	}
	return chunks
}

func knowledgeScore(item model.KnowledgeItem, keywords []string) int {
	title := strings.ToLower(item.Title)
	content := strings.ToLower(item.Content)
	score := 0
	for _, keyword := range keywords {
		keyword = strings.ToLower(strings.TrimSpace(keyword))
		if keyword == "" {
			continue
		}
		if strings.Contains(title, keyword) {
			score += 3
		}
		if strings.Contains(content, keyword) {
			score++
		}
	}
	return score
}

func knowledgeChunkIDs(chunks []ai.KnowledgeChunk) []string {
	ids := make([]string, 0, len(chunks))
	for _, chunk := range chunks {
		ids = append(ids, chunk.ID)
	}
	return ids
}

func (h *WorkspaceHandler) recordGeneration(ctx context.Context, item model.GenerationRequest) {
	h.mu.Lock()
	h.generations = append([]model.GenerationRequest{item}, h.generations...)
	h.mu.Unlock()

	dbCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	if err := h.db.SaveGenerationRequest(dbCtx, item); err != nil {
		log.Printf("generation request log was not persisted: %v", err)
	}
}

func (h *WorkspaceHandler) saveContent(ctx context.Context, item model.Content) error {
	dbCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	return h.db.SaveContent(dbCtx, item)
}

func encodeJSON(value any) string {
	data, err := json.Marshal(value)
	if err != nil {
		return "{}"
	}
	return string(data)
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

func cleanPhoneNumber(value string) (string, bool) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", true
	}

	digits := 0
	for _, char := range value {
		switch {
		case unicode.IsDigit(char):
			digits++
		case char == '+' || char == '-' || char == ' ' || char == '(' || char == ')':
		default:
			return "", false
		}
	}
	if digits < 7 || digits > 20 {
		return "", false
	}
	return value, true
}

func platformRequiresCredential(platform model.MediaPlatform, field string) bool {
	for _, item := range platform.CredentialFields {
		if item == field {
			return true
		}
	}
	return false
}

func loginSessionKey(workspaceID, accountID string) string {
	return workspaceID + ":" + accountID
}

func supportsBrowserLogin(platformType string) bool {
	return platformType == xiaohongshu.PlatformType
}

func browserProfilePath(workspaceID, accountID string) string {
	return xiaohongshu.RuntimeBrowserProfilePath(workspaceID, accountID)
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

func publishFormatOrContentType(values ...string) string {
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			return value
		}
	}
	return ai.FormatGenericArticle
}

func publisherForPlatform(platformType string) (publishing.Publisher, bool) {
	switch platformType {
	case xiaohongshu.PlatformType:
		return xiaohongshu.NewBrowserLongArticlePublisher(), true
	default:
		return nil, false
	}
}

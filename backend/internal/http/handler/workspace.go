package handler

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"regexp"
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
	"geopress/backend/internal/systemconfig"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
)

type WorkspaceHandler struct {
	mu                     sync.RWMutex
	db                     *database.DB
	aiConfig               *ai.RuntimeConfig
	users                  []model.User
	subscriptionPlans      []model.SubscriptionPlan
	workspaces             []model.Workspace
	members                []model.WorkspaceMember
	knowledgeBases         []model.KnowledgeBase
	knowledgeItems         []model.KnowledgeItem
	platformKnowledgeBases []model.PlatformKnowledgeBase
	platformKnowledgeItems []model.PlatformKnowledgeItem
	platforms              []model.MediaPlatform
	accounts               []model.MediaAccount
	contents               []model.Content
	schedules              []model.PublishSchedule
	jobs                   []model.PublishJob
	generations            []model.GenerationRequest
	tokenUsageEvents       []model.AITokenUsageEvent
	userSessions           map[string]string
	browserLogin           xiaohongshu.BrowserLoginService
}

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type registerRequest struct {
	Name          string `json:"name"`
	Email         string `json:"email"`
	Password      string `json:"password"`
	WorkspaceName string `json:"workspaceName"`
}

type completeOnboardingRequest struct {
	WorkspaceID        string   `json:"workspaceId"`
	Industry           string   `json:"industry"`
	Tones              []string `json:"tones"`
	SubscriptionPlanID string   `json:"subscriptionPlanId"`
	SkipSubscription   bool     `json:"skipSubscription"`
}

type createKnowledgeBaseRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

type createKnowledgeItemRequest struct {
	KnowledgeBaseID  string   `json:"knowledgeBaseId"`
	KnowledgeBaseIDs []string `json:"knowledgeBaseIds"`
	Type             string   `json:"type"`
	Title            string   `json:"title"`
	Content          string   `json:"content"`
}

type formatKnowledgeItemRequest struct {
	Type    string `json:"type"`
	Title   string `json:"title"`
	Content string `json:"content"`
}

type formatKnowledgeItemResponse struct {
	Content       string        `json:"content"`
	Provider      string        `json:"provider"`
	Model         string        `json:"model"`
	Usage         ai.TokenUsage `json:"tokenUsage"`
	Fallback      bool          `json:"fallback"`
	FallbackError string        `json:"fallbackError,omitempty"`
}

type assignKnowledgeItemsToBasesRequest struct {
	KnowledgeItemIDs []string `json:"knowledgeItemIds"`
	KnowledgeBaseIDs []string `json:"knowledgeBaseIds"`
}

type createPlatformKnowledgeBaseRequest struct {
	Name              string `json:"name"`
	Description       string `json:"description"`
	Category          string `json:"category"`
	PriceCents        int    `json:"priceCents"`
	Currency          string `json:"currency"`
	MarketplaceListed bool   `json:"marketplaceListed"`
}

type createPlatformKnowledgeItemRequest struct {
	KnowledgeBaseID  string   `json:"knowledgeBaseId"`
	KnowledgeBaseIDs []string `json:"knowledgeBaseIds"`
	Type             string   `json:"type"`
	Title            string   `json:"title"`
	Content          string   `json:"content"`
	Enabled          *bool    `json:"enabled"`
}

type createMediaAccountRequest struct {
	PlatformID          string         `json:"platformId"`
	Name                string         `json:"name"`
	ExternalID          string         `json:"externalId"`
	LoginMethod         string         `json:"loginMethod"`
	PhoneNumber         string         `json:"phoneNumber"`
	AccountGroup        string         `json:"accountGroup"`
	OwnershipType       string         `json:"ownershipType"`
	OperatingRole       string         `json:"operatingRole"`
	Persona             string         `json:"persona"`
	Positioning         string         `json:"positioning"`
	TargetAudience      string         `json:"targetAudience"`
	ContentCategories   []string       `json:"contentCategories"`
	HealthNotes         string         `json:"healthNotes"`
	AuthorizationScopes []string       `json:"authorizationScopes"`
	SyncEnabled         bool           `json:"syncEnabled"`
	MatrixMetadata      map[string]any `json:"matrixMetadata"`
}

type startMediaAccountBrowserLoginRequest struct {
}

type completeMediaAccountBrowserLoginRequest struct {
	SessionID string `json:"sessionId"`
}

type generateContentRequest struct {
	Keywords         []string `json:"keywords"`
	KeywordPrompt    string   `json:"keywordPrompt"`
	ContentType      string   `json:"contentType"`
	KnowledgeBaseID  string   `json:"knowledgeBaseId"`
	KnowledgeBaseIDs []string `json:"knowledgeBaseIds"`
	PublishFormatID  string   `json:"publishFormatId"`
	MediaAccountID   string   `json:"mediaAccountId"`
}

type generateContentResponse struct {
	Content model.Content      `json:"content"`
	Trace   ai.GenerationTrace `json:"trace"`
}

type createContentRequest struct {
	Title                    string         `json:"title"`
	Summary                  string         `json:"summary"`
	Body                     string         `json:"body"`
	Author                   string         `json:"author"`
	KnowledgeBaseID          string         `json:"knowledgeBaseId"`
	Keywords                 []string       `json:"keywords"`
	AttributedMediaAccountID string         `json:"attributedMediaAccountId"`
	AttributionMetadata      map[string]any `json:"attributionMetadata"`
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
	Provider              string                        `json:"provider"`
	OpenAIAPIKey          string                        `json:"openAIAPIKey"`
	OpenAIBaseURL         string                        `json:"openAIBaseUrl"`
	OpenAIModel           string                        `json:"openAIModel"`
	RequestTimeoutSeconds int                           `json:"requestTimeoutSeconds"`
	ClearAPIKey           bool                          `json:"clearAPIKey"`
	GenerationPipeline    ai.GenerationPipelineSettings `json:"generationPipeline"`
}

type updateUserSubscriptionRequest struct {
	SubscriptionTier      model.SubscriptionTier   `json:"subscriptionTier"`
	SubscriptionStatus    model.SubscriptionStatus `json:"subscriptionStatus"`
	SubscriptionExpiresAt string                   `json:"subscriptionExpiresAt"`
}

const registrationPasswordMinLength = 8

var registrationEmailPattern = regexp.MustCompile(`^[^\s@]+@[^\s@]+\.[^\s@]+$`)

func NewWorkspaceHandler(db *database.DB, aiConfig *ai.RuntimeConfig) *WorkspaceHandler {
	h, err := NewWorkspaceHandlerWithError(db, aiConfig)
	if err != nil {
		panic(err)
	}
	return h
}

func NewWorkspaceHandlerWithError(db *database.DB, aiConfig *ai.RuntimeConfig) (*WorkspaceHandler, error) {
	if aiConfig == nil {
		aiConfig = ai.NewRuntimeConfig(ai.Config{Provider: ai.ProviderMock})
	}

	now := time.Now().UTC()
	demoSubscriptionExpiresAt := now.AddDate(1, 0, 0)

	h := &WorkspaceHandler{
		db:           db,
		aiConfig:     aiConfig,
		userSessions: map[string]string{},
		browserLogin: xiaohongshu.NewPlaywrightBrowserLoginService(),
		subscriptionPlans: []model.SubscriptionPlan{
			{
				ID:                      model.SubscriptionPlanFree,
				Name:                    "Free",
				Tier:                    model.SubscriptionTierFree,
				PriceCents:              0,
				Currency:                "USD",
				MonthlyTokenBudgetCents: 0,
				InputTokenPricePer1K:    1,
				OutputTokenPricePer1K:   4,
				Enabled:                 true,
			},
			{
				ID:                      model.SubscriptionPlanVIP,
				Name:                    "VIP",
				Tier:                    model.SubscriptionTierVIP,
				PriceCents:              10000,
				Currency:                "USD",
				MonthlyTokenBudgetCents: 10000,
				InputTokenPricePer1K:    1,
				OutputTokenPricePer1K:   4,
				Enabled:                 true,
			},
		},
		users: []model.User{
			{
				ID:                        "usr_demo",
				Name:                      "Ava Chen",
				Email:                     "demo@geopress.local",
				IsPlatformAdmin:           true,
				SubscriptionTier:          model.SubscriptionTierVIP,
				SubscriptionPlanID:        model.SubscriptionPlanVIP,
				SubscriptionStatus:        model.SubscriptionStatusActive,
				SubscriptionExpiresAt:     &demoSubscriptionExpiresAt,
				MonthlyTokenBudgetCents:   10000,
				SubscriptionCurrentPeriod: now.Format("2006-01"),
				OnboardingCompleted:       true,
				OnboardingCompletedAt:     &now,
				CreatedAt:                 now.AddDate(0, -3, 0),
			},
			{
				ID:                        "usr_growth",
				Name:                      "Noah Wang",
				Email:                     "growth@geopress.local",
				IsPlatformAdmin:           false,
				SubscriptionTier:          model.SubscriptionTierFree,
				SubscriptionPlanID:        model.SubscriptionPlanFree,
				SubscriptionStatus:        model.SubscriptionStatusActive,
				SubscriptionCurrentPeriod: now.Format("2006-01"),
				OnboardingCompleted:       true,
				OnboardingCompletedAt:     &now,
				CreatedAt:                 now.AddDate(0, -2, 0),
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
				ID:               "kbi_1001",
				KnowledgeBaseIDs: []string{"kb_brand"},
				WorkspaceID:      "wks_acme",
				Type:             "brand",
				Title:            "品牌定位",
				Content:          "Acme 面向 B2B SaaS 团队，帮助市场和增长负责人规划内容生产、分发和复盘。",
				Enabled:          true,
				UpdatedAt:        now.Add(-5 * time.Hour),
			},
			{
				ID:               "kbi_1002",
				KnowledgeBaseIDs: []string{"kb_brand"},
				WorkspaceID:      "wks_acme",
				Type:             "audience",
				Title:            "目标受众",
				Content:          "主要读者是市场负责人、内容运营、创始人和增长团队。",
				Enabled:          true,
				UpdatedAt:        now.Add(-6 * time.Hour),
			},
			{
				ID:               "kbi_2001",
				KnowledgeBaseIDs: []string{"kb_personal"},
				WorkspaceID:      "wks_personal",
				Type:             "style",
				Title:            "写作风格",
				Content:          "文章应直接、具体，避免夸张营销话术，强调可执行建议。",
				Enabled:          true,
				UpdatedAt:        now.Add(-24 * time.Hour),
			},
		},
		platformKnowledgeBases: []model.PlatformKnowledgeBase{
			{
				ID:                "pkb_xhs_local_life",
				Name:              "小红书本地生活种草包",
				Description:       "适合餐饮、门店、本地服务账号的选题、结构和表达规则。",
				Category:          "小红书",
				PriceCents:        9900,
				Currency:          "CNY",
				MarketplaceListed: true,
				ItemCount:         2,
				UpdatedAt:         now.Add(-12 * time.Hour),
			},
			{
				ID:                "pkb_b2b_saas_seo",
				Name:              "B2B SaaS SEO 文章包",
				Description:       "面向 SaaS 官网博客的文章结构、受众痛点和 CTA 写法。",
				Category:          "SEO",
				PriceCents:        12900,
				Currency:          "CNY",
				MarketplaceListed: false,
				ItemCount:         1,
				UpdatedAt:         now.Add(-18 * time.Hour),
			},
		},
		platformKnowledgeItems: []model.PlatformKnowledgeItem{
			{
				ID:               "pki_xhs_1001",
				KnowledgeBaseIDs: []string{"pkb_xhs_local_life"},
				Type:             "structure",
				Title:            "本地生活笔记结构",
				Content:          "开头直接给场景和人群，正文按到店理由、体验细节、价格/预约信息、避坑提醒组织，结尾用低压行动引导。",
				Enabled:          true,
				UpdatedAt:        now.Add(-12 * time.Hour),
			},
			{
				ID:               "pki_xhs_1002",
				KnowledgeBaseIDs: []string{"pkb_xhs_local_life"},
				Type:             "compliance",
				Title:            "本地生活表达边界",
				Content:          "避免绝对化承诺和医疗化效果描述，优惠信息需明确适用条件，体验描述不要伪装成未披露广告。",
				Enabled:          true,
				UpdatedAt:        now.Add(-12 * time.Hour),
			},
			{
				ID:               "pki_seo_1001",
				KnowledgeBaseIDs: []string{"pkb_b2b_saas_seo"},
				Type:             "template",
				Title:            "SaaS SEO 文章骨架",
				Content:          "标题围绕具体问题，导语定义读者处境，正文使用问题-原因-操作步骤-指标复盘结构，结尾连接产品能力但避免硬广。",
				Enabled:          true,
				UpdatedAt:        now.Add(-18 * time.Hour),
			},
		},
		platforms: []model.MediaPlatform{
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
		},
		accounts: []model.MediaAccount{
			{
				ID:                  "acc_xhs_acme",
				WorkspaceID:         "wks_acme",
				PlatformID:          "plt_xiaohongshu",
				Name:                "Acme 小红书",
				ExternalID:          "AcmeGrowth",
				LoginMethod:         "qr",
				CredentialMeta:      map[string]string{},
				Status:              "pending_login",
				AccountGroup:        "品牌主账号",
				OwnershipType:       "owned",
				OperatingRole:       "primary",
				Persona:             "B2B SaaS 增长顾问",
				Positioning:         "增长内容与运营方法论",
				TargetAudience:      "市场负责人、内容运营、创始人",
				ContentCategories:   []string{"SaaS", "增长", "内容营销"},
				HealthStatus:        "needs_authorization",
				AuthorizationScopes: []string{"profile:read"},
				LastSyncStatus:      "never_synced",
				MatrixMetadata:      map[string]any{},
				LastCheckedAt:       now.Add(-90 * time.Minute),
			},
			{
				ID:                  "acc_xhs_personal",
				WorkspaceID:         "wks_personal",
				PlatformID:          "plt_xiaohongshu",
				Name:                "Ava 小红书",
				ExternalID:          "AvaCreator",
				LoginMethod:         "qr",
				CredentialMeta:      map[string]string{},
				Status:              "pending_login",
				AccountGroup:        "个人账号",
				OwnershipType:       "owned",
				OperatingRole:       "primary",
				Persona:             "独立顾问",
				Positioning:         "个人服务与方法沉淀",
				TargetAudience:      "独立顾问、早期创业者",
				ContentCategories:   []string{"独立顾问", "内容飞轮"},
				HealthStatus:        "needs_authorization",
				AuthorizationScopes: []string{"profile:read"},
				LastSyncStatus:      "never_synced",
				MatrixMetadata:      map[string]any{},
				LastCheckedAt:       now.Add(-3 * time.Hour),
			},
		},
		contents: []model.Content{
			{
				ID:                       "cnt_1001",
				WorkspaceID:              "wks_acme",
				KnowledgeBaseID:          "kb_brand",
				AttributedMediaAccountID: "acc_xhs_acme",
				Title:                    "Q3 SaaS 增长内容规划",
				Summary:                  "围绕获客、转化和留存的内容发布计划。",
				Body:                     "这是一篇示例草稿，用于展示内容生命周期和排程发布。",
				Keywords:                 []string{"SaaS", "增长", "内容营销"},
				Status:                   model.ContentScheduled,
				Author:                   "Ava Chen",
				Source:                   "mock_ai",
				AttributionMetadata: map[string]any{
					"mediaAccountId":    "acc_xhs_acme",
					"attributionSource": "seed_demo",
					"attributedAt":      now.Add(-2 * time.Hour).Format(time.RFC3339),
				},
				UpdatedAt: now.Add(-2 * time.Hour),
			},
			{
				ID:                  "cnt_2001",
				WorkspaceID:         "wks_personal",
				KnowledgeBaseID:     "kb_personal",
				Title:               "独立顾问如何搭建内容飞轮",
				Summary:             "用稳定输出和案例沉淀提升获客效率。",
				Body:                "这是一篇个人工作区示例内容。",
				Keywords:            []string{"独立顾问", "内容飞轮"},
				Status:              model.ContentDraft,
				Author:              "Ava Chen",
				Source:              "manual",
				AttributionMetadata: map[string]any{},
				UpdatedAt:           now.Add(-20 * time.Hour),
			},
		},
		schedules: []model.PublishSchedule{
			{
				ID:             "sch_1001",
				WorkspaceID:    "wks_acme",
				Name:           "每周三小红书长文",
				ContentID:      "cnt_1001",
				MediaAccountID: "acc_xhs_acme",
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
				MediaAccountID: "acc_xhs_acme",
				Status:         model.PublishJobManual,
				ScheduledAt:    now.Add(48 * time.Hour),
				LastMessage:    "小红书发布需要登录浏览器确认。",
				AttributionMetadata: map[string]any{
					"contentId":         "cnt_1001",
					"mediaAccountId":    "acc_xhs_acme",
					"scheduleId":        "sch_1001",
					"attributionSource": "seed_demo",
					"attributedAt":      now.Add(-24 * time.Hour).Format(time.RFC3339),
				},
			},
		},
	}

	if !h.seedDatabase(context.Background()) {
		return nil, fmt.Errorf("database seed failed")
	}
	if !h.loadDatabaseSnapshot(context.Background()) {
		return nil, fmt.Errorf("database snapshot load failed")
	}
	return h, nil
}

func (h *WorkspaceHandler) Register(router gin.IRouter, auth gin.HandlerFunc) {
	router.POST("/auth/login", h.Login)
	router.POST("/auth/register", h.RegisterUser)

	protected := router.Group("/")
	protected.Use(auth)
	protected.Use(h.databaseSnapshotMiddleware())
	protected.GET("/me", h.Me)
	protected.GET("/workspaces", h.ListWorkspaces)
	protected.GET("/subscription-plans", h.ListSubscriptionPlans)
	protected.POST("/onboarding/complete", h.CompleteOnboarding)
	protected.GET("/overview", h.Overview)
	protected.GET("/knowledge-bases", h.ListKnowledgeBases)
	protected.POST("/knowledge-bases", h.CreateKnowledgeBase)
	protected.GET("/knowledge-items", h.ListKnowledgeItems)
	protected.POST("/knowledge-items", h.CreateKnowledgeItem)
	protected.POST("/knowledge-items/format", h.FormatKnowledgeItem)
	protected.POST("/knowledge-items/assign-bases", h.AssignKnowledgeItemsToBases)
	protected.GET("/media-platforms", h.ListMediaPlatforms)
	protected.GET("/media-accounts", h.ListMediaAccounts)
	protected.POST("/media-accounts", h.CreateMediaAccount)
	protected.GET("/media-account-matrix", h.ListMediaAccountMatrix)
	protected.GET("/media-account-matrix/:accountId", h.GetMediaAccountMatrixItem)
	protected.GET("/media-account-matrix/:accountId/metric-snapshots", h.ListMediaAccountMetricSnapshots)
	protected.GET("/content-metrics", h.ListContentMetrics)
	protected.POST("/media-account-matrix/:accountId/sync-jobs", h.CreateMediaAccountSyncJob)
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
	admin.PUT("/users/:userId/subscription", h.AdminUpdateUserSubscription)
	admin.GET("/workspaces", h.AdminListWorkspaces)
	admin.GET("/workspace-members", h.AdminListWorkspaceMembers)
	admin.GET("/platform-knowledge-bases", h.AdminListPlatformKnowledgeBases)
	admin.POST("/platform-knowledge-bases", h.AdminCreatePlatformKnowledgeBase)
	admin.PUT("/platform-knowledge-bases/:knowledgeBaseId", h.AdminUpdatePlatformKnowledgeBase)
	admin.GET("/platform-knowledge-items", h.AdminListPlatformKnowledgeItems)
	admin.POST("/platform-knowledge-items", h.AdminCreatePlatformKnowledgeItem)
	admin.PUT("/platform-knowledge-items/:knowledgeItemId", h.AdminUpdatePlatformKnowledgeItem)
	admin.GET("/media-platforms", h.AdminListMediaPlatforms)
	admin.POST("/media-platforms", h.AdminCreateMediaPlatform)
	admin.PUT("/media-platforms/:platformId", h.AdminUpdateMediaPlatform)
	admin.GET("/media-accounts", h.AdminListMediaAccounts)
	admin.GET("/ai-config", h.AdminGetAIConfig)
	admin.PUT("/ai-config", h.AdminUpdateAIConfig)
}

func (h *WorkspaceHandler) Login(c *gin.Context) {
	if !h.loadDatabaseSnapshot(c.Request.Context()) {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "database snapshot load failed"})
		return
	}

	var req loginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	email := strings.TrimSpace(strings.ToLower(req.Email))
	password := req.Password
	if h.db != nil && h.db.SQL() != nil {
		if email == "" || password == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "email and password are required"})
			return
		}

		dbCtx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
		credentials, ok, err := h.db.UserCredentialsByEmail(dbCtx, email)
		cancel()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "user lookup failed"})
			return
		}
		if !ok || bcrypt.CompareHashAndPassword([]byte(credentials.PasswordHash), []byte(password)) != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid email or password"})
			return
		}

		token, err := h.createLoginSession(c.Request.Context(), credentials.User.ID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "login session was not created"})
			return
		}

		h.mu.RLock()
		workspaces := h.workspacesForUser(credentials.User.ID)
		h.mu.RUnlock()

		c.JSON(http.StatusOK, gin.H{
			"token":      token,
			"user":       credentials.User,
			"workspaces": workspaces,
		})
		return
	}

	userID, token := "usr_demo", "demo-token"
	if email == "" {
		email = "demo@geopress.local"
	}
	if password != "demo" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid email or password"})
		return
	}
	switch email {
	case "demo@geopress.local":
	case "growth@geopress.local":
		userID = "usr_growth"
		token = "growth-token"
	default:
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid email or password"})
		return
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

func (h *WorkspaceHandler) RegisterUser(c *gin.Context) {
	if !h.loadDatabaseSnapshot(c.Request.Context()) {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "database snapshot load failed"})
		return
	}

	var req registerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	name := strings.TrimSpace(req.Name)
	email := strings.TrimSpace(strings.ToLower(req.Email))
	password := req.Password
	workspaceName := strings.TrimSpace(req.WorkspaceName)
	if name == "" || email == "" || password == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "name, email and password are required"})
		return
	}
	if !registrationEmailPattern.MatchString(email) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "email is invalid"})
		return
	}
	if len([]rune(password)) < registrationPasswordMinLength {
		c.JSON(http.StatusBadRequest, gin.H{"error": "password must be at least 8 characters"})
		return
	}
	if workspaceName == "" {
		workspaceName = fmt.Sprintf("%s 的个人工作区", name)
	}

	if h.db != nil && h.db.SQL() != nil {
		dbCtx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
		_, exists, err := h.db.UserCredentialsByEmail(dbCtx, email)
		cancel()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "user lookup failed"})
			return
		}
		if exists {
			c.JSON(http.StatusConflict, gin.H{"error": "email is already registered"})
			return
		}
	}

	userID, err := secureRandomID("usr")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "user id was not created"})
		return
	}
	workspaceID, err := secureRandomID("wks")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "workspace id was not created"})
		return
	}
	passwordHash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "password was not processed"})
		return
	}

	now := time.Now().UTC()
	user := model.User{
		ID:                        userID,
		Name:                      name,
		Email:                     email,
		IsPlatformAdmin:           false,
		SubscriptionTier:          model.SubscriptionTierFree,
		SubscriptionPlanID:        model.SubscriptionPlanFree,
		SubscriptionStatus:        model.SubscriptionStatusActive,
		SubscriptionCurrentPeriod: now.Format("2006-01"),
		OnboardingCompleted:       false,
		CreatedAt:                 now,
	}
	workspace := model.Workspace{
		ID:        workspaceID,
		Name:      workspaceName,
		Type:      model.WorkspacePersonal,
		Plan:      "Personal",
		Status:    "active",
		Industry:  "",
		Language:  "zh-CN",
		Tone:      "专业、清晰",
		CreatedAt: now,
	}
	member := model.WorkspaceMember{
		UserID:      user.ID,
		WorkspaceID: workspace.ID,
		Role:        "owner",
	}

	if h.db != nil && h.db.SQL() != nil {
		dbCtx, cancel := context.WithTimeout(c.Request.Context(), 3*time.Second)
		err = h.db.CreateRegisteredUser(dbCtx, user, string(passwordHash), workspace, member)
		cancel()
		if err != nil {
			log.Printf("user registration was not persisted: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "user was not registered"})
			return
		}
	} else {
		h.mu.Lock()
		if h.emailExistsLocked(email) {
			h.mu.Unlock()
			c.JSON(http.StatusConflict, gin.H{"error": "email is already registered"})
			return
		}
		h.users = append(h.users, user)
		h.workspaces = append(h.workspaces, workspace)
		h.members = append(h.members, member)
		h.mu.Unlock()
	}

	token, err := h.createLoginSession(c.Request.Context(), user.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "login session was not created"})
		return
	}
	if !h.loadDatabaseSnapshot(c.Request.Context()) {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "database snapshot load failed"})
		return
	}

	h.mu.RLock()
	responseUser, ok := h.userByID(user.ID)
	workspaces := h.workspacesForUser(user.ID)
	h.mu.RUnlock()
	if !ok {
		responseUser = user
		workspaces = []model.Workspace{workspace}
	}

	c.JSON(http.StatusCreated, gin.H{
		"token":      token,
		"user":       responseUser,
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

func (h *WorkspaceHandler) ListSubscriptionPlans(c *gin.Context) {
	h.mu.RLock()
	items := make([]model.SubscriptionPlan, 0, len(h.subscriptionPlans))
	for _, plan := range h.subscriptionPlans {
		if plan.Enabled {
			items = append(items, plan)
		}
	}
	h.mu.RUnlock()
	c.JSON(http.StatusOK, gin.H{"items": items})
}

func (h *WorkspaceHandler) CompleteOnboarding(c *gin.Context) {
	userID := middleware.CurrentUserID(c)

	var req completeOnboardingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	workspaceID := strings.TrimSpace(req.WorkspaceID)
	if workspaceID == "" {
		workspaceID = middleware.CurrentWorkspaceID(c)
	}
	industry := strings.TrimSpace(req.Industry)
	tones := uniqueStrings(cleanKeywords(req.Tones))
	if industry == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "industry is required"})
		return
	}
	if len(tones) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "at least one tone is required"})
		return
	}
	if len(tones) > 5 {
		tones = tones[:5]
	}

	planID := model.SubscriptionPlanID(strings.TrimSpace(req.SubscriptionPlanID))
	if req.SkipSubscription || planID == "" {
		planID = model.SubscriptionPlanFree
	}

	now := time.Now().UTC()
	h.mu.RLock()
	if !h.userCanAccessWorkspace(userID, workspaceID) {
		h.mu.RUnlock()
		c.JSON(http.StatusForbidden, gin.H{"error": "workspace access denied"})
		return
	}
	plan, planOK := h.subscriptionPlanByID(planID)
	user, userOK := h.userByID(userID)
	workspace, workspaceOK := h.workspaceByID(workspaceID)
	h.mu.RUnlock()
	if !userOK {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "user not found"})
		return
	}
	if !workspaceOK {
		c.JSON(http.StatusNotFound, gin.H{"error": "workspace not found"})
		return
	}
	if !planOK || !plan.Enabled {
		c.JSON(http.StatusBadRequest, gin.H{"error": "subscription plan is invalid"})
		return
	}

	workspace.Industry = industry
	workspace.Tone = strings.Join(tones, "、")
	user.SubscriptionTier = plan.Tier
	user.SubscriptionPlanID = plan.ID
	user.SubscriptionStatus = model.SubscriptionStatusActive
	user.MonthlyTokenBudgetCents = plan.MonthlyTokenBudgetCents
	user.SubscriptionCurrentPeriod = now.Format("2006-01")
	user.OnboardingCompleted = true
	user.OnboardingCompletedAt = &now
	if plan.ID == model.SubscriptionPlanVIP && user.SubscriptionExpiresAt == nil {
		expiresAt := now.AddDate(0, 1, 0)
		user.SubscriptionExpiresAt = &expiresAt
	}
	if plan.ID == model.SubscriptionPlanFree {
		user.SubscriptionExpiresAt = nil
	}

	if err := h.completeUserOnboarding(c.Request.Context(), user, workspace); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "onboarding was not persisted"})
		return
	}

	h.mu.Lock()
	for index := range h.users {
		if h.users[index].ID == user.ID {
			h.users[index] = user
			break
		}
	}
	for index := range h.workspaces {
		if h.workspaces[index].ID == workspace.ID {
			h.workspaces[index] = workspace
			break
		}
	}
	h.mu.Unlock()

	c.JSON(http.StatusOK, gin.H{
		"user":      user,
		"workspace": workspace,
	})
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

	if err := h.saveKnowledgeBase(c.Request.Context(), item); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "knowledge base was not persisted"})
		return
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
	knowledgeBaseIDs := cleanKnowledgeBaseIDs(req.KnowledgeBaseID, req.KnowledgeBaseIDs)
	if len(knowledgeBaseIDs) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "at least one knowledge base is required"})
		return
	}

	now := time.Now().UTC()
	item := model.KnowledgeItem{
		ID:               fmt.Sprintf("kbi_%d", now.UnixNano()),
		KnowledgeBaseIDs: knowledgeBaseIDs,
		WorkspaceID:      workspaceID,
		Type:             defaultString(strings.TrimSpace(req.Type), "note"),
		Title:            title,
		Content:          content,
		Enabled:          true,
		UpdatedAt:        now,
	}

	h.mu.RLock()
	if len(knowledgeBaseIDs) > 0 && !h.hasKnowledgeBasesLocked(workspaceID, knowledgeBaseIDs) {
		h.mu.RUnlock()
		c.JSON(http.StatusNotFound, gin.H{"error": "knowledge base not found"})
		return
	}
	h.mu.RUnlock()

	if err := h.saveKnowledgeItem(c.Request.Context(), item); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "knowledge item was not persisted"})
		return
	}

	h.mu.Lock()
	h.knowledgeItems = append([]model.KnowledgeItem{item}, h.knowledgeItems...)
	h.recountKnowledgeBaseItemsLocked(workspaceID)
	h.mu.Unlock()
	c.JSON(http.StatusCreated, item)
}

func (h *WorkspaceHandler) FormatKnowledgeItem(c *gin.Context) {
	workspaceID, ok := h.authorizedWorkspaceID(c)
	if !ok {
		return
	}
	userID := middleware.CurrentUserID(c)

	h.mu.RLock()
	user, userOK := h.userByID(userID)
	h.mu.RUnlock()
	if !userOK {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "user not found"})
		return
	}
	if !user.HasActiveVIP(time.Now().UTC()) {
		c.JSON(http.StatusForbidden, gin.H{"error": "VIP subscription is required for knowledge content formatting"})
		return
	}

	var req formatKnowledgeItemRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	content := strings.TrimSpace(req.Content)
	if content == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "content is required"})
		return
	}

	formatReq := ai.FormatKnowledgeContentRequest{
		WorkspaceID: workspaceID,
		UserID:      userID,
		Type:        strings.TrimSpace(req.Type),
		Title:       strings.TrimSpace(req.Title),
		Content:     content,
	}
	provider := h.aiConfig.Provider()
	response, err := provider.FormatKnowledgeContent(c.Request.Context(), formatReq)
	fallback := false
	fallbackError := ""
	if err != nil {
		if provider.Name() == ai.ProviderMock {
			c.JSON(http.StatusBadGateway, gin.H{"error": "knowledge content formatting failed"})
			return
		}
		log.Printf("knowledge content formatting provider %s failed, falling back to mock: %v", provider.Name(), err)
		fallback = true
		fallbackError = err.Error()
		response, err = ai.NewMockProvider().FormatKnowledgeContent(c.Request.Context(), formatReq)
		if err != nil {
			c.JSON(http.StatusBadGateway, gin.H{"error": "knowledge content formatting failed"})
			return
		}
	}

	c.JSON(http.StatusOK, formatKnowledgeItemResponse{
		Content:       strings.TrimSpace(response.Content),
		Provider:      response.Provider,
		Model:         response.Model,
		Usage:         response.TokenUsage,
		Fallback:      fallback,
		FallbackError: fallbackError,
	})
}

func (h *WorkspaceHandler) AssignKnowledgeItemsToBases(c *gin.Context) {
	workspaceID, ok := h.authorizedWorkspaceID(c)
	if !ok {
		return
	}

	var req assignKnowledgeItemsToBasesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	knowledgeItemIDs := uniqueStrings(cleanKeywords(req.KnowledgeItemIDs))
	knowledgeBaseIDs := cleanKnowledgeBaseIDs("", req.KnowledgeBaseIDs)
	if len(knowledgeItemIDs) == 0 || len(knowledgeBaseIDs) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "knowledgeItemIds and knowledgeBaseIds are required"})
		return
	}

	now := time.Now().UTC()
	h.mu.RLock()

	if !h.hasKnowledgeBasesLocked(workspaceID, knowledgeBaseIDs) {
		h.mu.RUnlock()
		c.JSON(http.StatusNotFound, gin.H{"error": "knowledge base not found"})
		return
	}

	foundItemIDs := map[string]bool{}
	for index := range h.knowledgeItems {
		item := &h.knowledgeItems[index]
		if item.WorkspaceID != workspaceID || !containsString(knowledgeItemIDs, item.ID) {
			continue
		}
		foundItemIDs[item.ID] = true
	}
	if len(foundItemIDs) != len(knowledgeItemIDs) {
		h.mu.RUnlock()
		c.JSON(http.StatusNotFound, gin.H{"error": "knowledge item not found"})
		return
	}
	h.mu.RUnlock()

	if err := h.assignKnowledgeItemsToBases(c.Request.Context(), workspaceID, knowledgeItemIDs, knowledgeBaseIDs); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "knowledge item assignment was not persisted"})
		return
	}

	updated := make([]model.KnowledgeItem, 0, len(knowledgeItemIDs))
	h.mu.Lock()
	for index := range h.knowledgeItems {
		item := &h.knowledgeItems[index]
		if item.WorkspaceID != workspaceID || !foundItemIDs[item.ID] {
			continue
		}
		item.KnowledgeBaseIDs = mergeStringSet(item.KnowledgeBaseIDs, knowledgeBaseIDs)
		item.UpdatedAt = now
		updated = append(updated, *item)
	}

	h.recountKnowledgeBaseItemsLocked(workspaceID)
	h.mu.Unlock()
	c.JSON(http.StatusOK, gin.H{"items": updated})
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

	// 平台能力先从账号绑定方式收敛：小红书等 QR 平台不接受普通用户绕过受支持的授权路径。
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
		ID:                  fmt.Sprintf("acc_%d", now.UnixNano()),
		WorkspaceID:         workspaceID,
		PlatformID:          platformID,
		Name:                name,
		ExternalID:          strings.TrimSpace(req.ExternalID),
		LoginMethod:         loginMethod,
		CredentialMeta:      credentialMeta,
		Status:              status,
		AccountGroup:        strings.TrimSpace(req.AccountGroup),
		OwnershipType:       defaultString(strings.TrimSpace(req.OwnershipType), "owned"),
		OperatingRole:       defaultString(strings.TrimSpace(req.OperatingRole), "primary"),
		Persona:             strings.TrimSpace(req.Persona),
		Positioning:         strings.TrimSpace(req.Positioning),
		TargetAudience:      strings.TrimSpace(req.TargetAudience),
		ContentCategories:   uniqueStrings(cleanKeywords(req.ContentCategories)),
		HealthStatus:        mediaAccountHealthFromStatus(status),
		HealthNotes:         strings.TrimSpace(req.HealthNotes),
		AuthorizationScopes: uniqueStrings(cleanKeywords(req.AuthorizationScopes)),
		SyncEnabled:         req.SyncEnabled,
		LastSyncStatus:      "never_synced",
		MatrixMetadata:      nonNilMap(req.MatrixMetadata),
		LastCheckedAt:       now,
	}

	if err := h.saveMediaAccount(c.Request.Context(), account); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "media account was not persisted"})
		return
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
		account.CredentialMeta["browserLoginUrl"] = loginResult.LoginURL
		account.CredentialMeta["browserLoginStateFile"] = loginResult.StateFile
		account.CredentialMeta["loginSessionId"] = loginResult.SessionID
		account.CredentialMeta["loginSessionExpiresAt"] = expiresAt.Format(time.RFC3339)
		account.Status = "qr_waiting"
		if loginResult.AlreadyLoggedIn {
			account.Status = "connected"
			account.CredentialMeta["qrLoginCompletedAt"] = loginResult.StartedAt.Format(time.RFC3339)
		}
		account.HealthStatus = mediaAccountHealthFromStatus(account.Status)
		account.LastCheckedAt = now
		updated = *account
		break
	}
	h.mu.Unlock()
	if updated.ID == "" {
		c.JSON(http.StatusNotFound, gin.H{"error": "media account not found"})
		return
	}
	if err := h.saveMediaAccount(c.Request.Context(), updated); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "media account login state was not persisted"})
		return
	}
	if err := h.saveMediaAccountLoginSession(c.Request.Context(), model.MediaAccountLoginSession{
		ID:          loginResult.SessionID,
		WorkspaceID: workspaceID,
		AccountID:   accountID,
		Platform:    platform.Type,
		ProfileDir:  loginResult.ProfileDir,
		LoginURL:    loginResult.LoginURL,
		StateFile:   loginResult.StateFile,
		Status:      "active",
		ExpiresAt:   expiresAt,
		CreatedAt:   now,
		UpdatedAt:   now,
	}); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "media account login session was not persisted"})
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

	loginSession, sessionOK, err := h.latestMediaAccountLoginSession(c.Request.Context(), workspaceID, accountID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "media account login session lookup failed"})
		return
	}
	if !sessionOK {
		c.JSON(http.StatusBadRequest, gin.H{"error": "qr login session was not started"})
		return
	}
	if now.After(loginSession.ExpiresAt) {
		_ = h.expireMediaAccountLoginSession(c.Request.Context(), loginSession.ID)
		c.JSON(http.StatusBadRequest, gin.H{"error": "qr login session expired"})
		return
	}
	if loginSession.ID != sessionID {
		c.JSON(http.StatusBadRequest, gin.H{"error": "qr login session is invalid"})
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

	var updated model.MediaAccount
	h.mu.Lock()
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
		account.CredentialMeta["browserLoginUrl"] = loginSession.LoginURL
		account.CredentialMeta["browserLoginStateFile"] = loginResult.StateFile
		account.CredentialMeta["loginSessionId"] = sessionID
		account.Status = "connected"
		account.HealthStatus = mediaAccountHealthFromStatus(account.Status)
		account.LastCheckedAt = now
		updated = *account
		break
	}
	h.mu.Unlock()

	if updated.ID == "" {
		c.JSON(http.StatusNotFound, gin.H{"error": "media account not found"})
		return
	}
	if err := h.saveMediaAccount(c.Request.Context(), updated); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "media account login state was not persisted"})
		return
	}
	if err := h.completeMediaAccountLoginSession(c.Request.Context(), loginSession.ID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "media account login session was not finalized"})
		return
	}

	c.JSON(http.StatusOK, updated)
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
	attributedAccountID := strings.TrimSpace(req.AttributedMediaAccountID)
	if attributedAccountID != "" {
		h.mu.RLock()
		accountOK := hasMediaAccount(h.accounts, workspaceID, attributedAccountID)
		h.mu.RUnlock()
		if !accountOK {
			c.JSON(http.StatusNotFound, gin.H{"error": "attributed media account not found"})
			return
		}
	}

	now := time.Now().UTC()
	content := model.Content{
		ID:                       fmt.Sprintf("cnt_%d", now.UnixNano()),
		WorkspaceID:              workspaceID,
		KnowledgeBaseID:          strings.TrimSpace(req.KnowledgeBaseID),
		AttributedMediaAccountID: attributedAccountID,
		Title:                    title,
		Summary:                  strings.TrimSpace(req.Summary),
		Body:                     strings.TrimSpace(req.Body),
		Keywords:                 cleanKeywords(req.Keywords),
		Status:                   model.ContentDraft,
		Author:                   defaultString(strings.TrimSpace(req.Author), "Current User"),
		Source:                   "manual",
		AttributionMetadata:      contentAttributionMetadata(nonNilMap(req.AttributionMetadata), attributedAccountID, "manual_create", now),
		UpdatedAt:                now,
	}

	if err := h.saveContent(c.Request.Context(), content); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "content was not persisted"})
		return
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
	keywordPrompt := strings.TrimSpace(req.KeywordPrompt)
	if len(keywords) == 0 {
		keywords = extractKeywordsFromMarkdownPrompt(keywordPrompt)
	}
	if len(keywords) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "at least one keyword is required"})
		return
	}

	now := time.Now().UTC()
	knowledgeBaseIDs := cleanKnowledgeBaseIDs(req.KnowledgeBaseID, req.KnowledgeBaseIDs)
	attributedAccountID := strings.TrimSpace(req.MediaAccountID)
	contentType, ok := ai.NormalizeContentType(publishFormatOrContentType(req.PublishFormatID, req.ContentType, ai.FormatGenericArticle))
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "unsupported content type"})
		return
	}
	skill := ai.SelectWritingSkill(contentType)
	publishFormat := ai.SelectPublishFormat(skill.ContentType)

	h.mu.RLock()
	user, userOK := h.userByID(userID)
	if !userOK {
		h.mu.RUnlock()
		c.JSON(http.StatusUnauthorized, gin.H{"error": "user not found"})
		return
	}
	if !h.hasKnowledgeBasesLocked(workspaceID, knowledgeBaseIDs) {
		h.mu.RUnlock()
		c.JSON(http.StatusNotFound, gin.H{"error": "knowledge base not found"})
		return
	}
	if attributedAccountID != "" && !hasMediaAccount(h.accounts, workspaceID, attributedAccountID) {
		h.mu.RUnlock()
		c.JSON(http.StatusNotFound, gin.H{"error": "media account not found"})
		return
	}
	workspace, _ := h.workspaceByID(workspaceID)
	chunks := h.retrieveKnowledgeChunksLocked(workspaceID, knowledgeBaseIDs, keywords, 8)
	h.mu.RUnlock()
	primaryKnowledgeBaseID := firstString(knowledgeBaseIDs)

	aiReq := ai.GenerateRequest{
		WorkspaceID:      workspaceID,
		UserID:           userID,
		KnowledgeBaseIDs: knowledgeBaseIDs,
		ContentType:      skill.ContentType,
		Keywords:         keywords,
		KeywordPrompt:    keywordPrompt,
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
	pipeline := h.aiConfig.GenerationPipelineForUser(user)
	trace := ai.GenerationTrace{
		SubscriptionTier: string(defaultUserSubscriptionTier(user.SubscriptionTier)),
		Pipeline:         pipeline,
		Steps:            []ai.GenerationTraceStep{},
		RetrievedIDs:     knowledgeChunkIDs(chunks),
	}

	var analysis ai.GenerationStageResponse
	var generationUsage ai.TokenUsage
	if pipeline.InputAnalysis {
		stage, err := provider.AnalyzeGenerationInput(c.Request.Context(), aiReq)
		if err != nil {
			trace.AddStep(ai.GenerationStageInputAnalysis, "输入分析", "failed", "输入分析失败", nil, []string{err.Error()})
			h.recordGeneration(c.Request.Context(), failedGenerationLog(now, workspaceID, userID, primaryKnowledgeBaseID, provider, skill, keywords, chunks, trace, err))
			c.JSON(http.StatusBadGateway, gin.H{"error": "content generation input analysis failed"})
			return
		}
		analysis = stage
		generationUsage = addTokenUsage(generationUsage, stage.TokenUsage)
		trace.AddStage(ai.GenerationStageInputAnalysis, "输入分析", stage)
	} else {
		trace.AddStep(ai.GenerationStageInputAnalysis, "输入分析", "skipped", "当前订阅链路未启用独立输入分析。", nil, nil)
	}

	trace.AddStep(ai.GenerationStageRetrieval, "知识检索", "succeeded", fmt.Sprintf("已检索 %d 个知识片段。", len(chunks)), knowledgeTraceDetails(chunks, knowledgeBaseIDs), warningsForRetrievedChunks(chunks))

	if pipeline.ContentPlan {
		stage, err := provider.PlanGeneration(c.Request.Context(), aiReq, analysis)
		if err != nil {
			trace.AddStep(ai.GenerationStagePlan, "创作计划", "failed", "创作计划失败", nil, []string{err.Error()})
			h.recordGeneration(c.Request.Context(), failedGenerationLog(now, workspaceID, userID, primaryKnowledgeBaseID, provider, skill, keywords, chunks, trace, err))
			c.JSON(http.StatusBadGateway, gin.H{"error": "content generation planning failed"})
			return
		}
		generationUsage = addTokenUsage(generationUsage, stage.TokenUsage)
		trace.AddStage(ai.GenerationStagePlan, "创作计划", stage)
	} else {
		trace.AddStep(ai.GenerationStagePlan, "创作计划", "skipped", "当前订阅链路未启用独立创作计划。", nil, nil)
	}

	response, err := provider.Generate(c.Request.Context(), aiReq)
	if err != nil {
		trace.AddStep(ai.GenerationStageDraft, "草稿生成", "failed", "草稿生成失败", nil, []string{err.Error()})
		h.recordGeneration(c.Request.Context(), failedGenerationLog(now, workspaceID, userID, primaryKnowledgeBaseID, provider, skill, keywords, chunks, trace, err))
		c.JSON(http.StatusBadGateway, gin.H{"error": "content generation failed"})
		return
	}
	generationUsage = addTokenUsage(generationUsage, response.TokenUsage)
	trace.AddStep(ai.GenerationStageDraft, "草稿生成", "succeeded", "已生成结构化草稿。", []string{
		"标题：" + strings.TrimSpace(response.Draft.Title),
		fmt.Sprintf("章节数：%d", len(response.Draft.Sections)),
		fmt.Sprintf("使用知识条目：%d 个", len(response.Draft.UsedKnowledgeIDs)),
	}, response.Draft.Warnings)

	if err := response.Draft.Validate(); err != nil {
		trace.AddStep(ai.GenerationStageDraft, "草稿结构校验", "failed", "生成结果未通过结构校验。", nil, []string{err.Error()})
		h.recordGeneration(c.Request.Context(), failedGenerationLog(now, workspaceID, userID, primaryKnowledgeBaseID, provider, skill, keywords, chunks, trace, err))
		c.JSON(http.StatusBadGateway, gin.H{"error": "generated content is invalid"})
		return
	}

	var quality ai.GenerationStageResponse
	if pipeline.QualityCheck {
		stage, err := provider.CheckGeneratedDraft(c.Request.Context(), aiReq, response.Draft)
		if err != nil {
			trace.AddStep(ai.GenerationStageQualityCheck, "质量检查", "failed", "质量检查失败", nil, []string{err.Error()})
			h.recordGeneration(c.Request.Context(), failedGenerationLog(now, workspaceID, userID, primaryKnowledgeBaseID, provider, skill, keywords, chunks, trace, err))
			c.JSON(http.StatusBadGateway, gin.H{"error": "content quality check failed"})
			return
		}
		quality = stage
		generationUsage = addTokenUsage(generationUsage, stage.TokenUsage)
		trace.AddStage(ai.GenerationStageQualityCheck, "质量检查", stage)
	} else {
		trace.AddStep(ai.GenerationStageQualityCheck, "质量检查", "skipped", "当前订阅链路未启用独立质量检查。", nil, nil)
	}

	rewriteRounds := pipeline.RewriteRounds
	if !pipeline.QualityCheck {
		rewriteRounds = 0
	}
	for round := 0; round < rewriteRounds; round++ {
		rewritten, err := provider.RewriteGeneratedDraft(c.Request.Context(), aiReq, response.Draft, quality)
		if err != nil {
			trace.AddStep(ai.GenerationStageRewrite, "草稿重写", "failed", fmt.Sprintf("第 %d 轮重写失败。", round+1), nil, []string{err.Error()})
			h.recordGeneration(c.Request.Context(), failedGenerationLog(now, workspaceID, userID, primaryKnowledgeBaseID, provider, skill, keywords, chunks, trace, err))
			c.JSON(http.StatusBadGateway, gin.H{"error": "content rewrite failed"})
			return
		}
		response = mergeGenerateResponses(response, rewritten)
		generationUsage = addTokenUsage(generationUsage, rewritten.TokenUsage)
		trace.AddStep(ai.GenerationStageRewrite, "草稿重写", "succeeded", fmt.Sprintf("已完成第 %d 轮草稿重写。", round+1), []string{
			"基于质量检查结果修正草稿。",
			"仍保持 draft 状态，等待人工审核。",
		}, response.Draft.Warnings)
	}
	if rewriteRounds == 0 {
		trace.AddStep(ai.GenerationStageRewrite, "草稿重写", "skipped", "当前订阅链路未启用自动重写。", nil, nil)
	}

	content := model.Content{
		ID:                       fmt.Sprintf("cnt_%d", now.UnixNano()),
		WorkspaceID:              workspaceID,
		KnowledgeBaseID:          primaryKnowledgeBaseID,
		AttributedMediaAccountID: attributedAccountID,
		Title:                    strings.TrimSpace(response.Draft.Title),
		Summary:                  strings.TrimSpace(response.Draft.Summary),
		Body:                     strings.TrimSpace(response.Draft.Body),
		Keywords:                 cleanKeywords(response.Draft.Keywords),
		Status:                   model.ContentDraft,
		Author:                   "AI Writer",
		Source:                   "ai_" + defaultString(response.Provider, "mock") + ":" + publishFormat.ID,
		AttributionMetadata: contentAttributionMetadata(map[string]any{
			"publishFormatId": publishFormat.ID,
			"writingSkillId":  skill.ID,
			"contentType":     skill.ContentType,
		}, attributedAccountID, "ai_generation", now),
		UpdatedAt: now,
	}

	if err := h.saveContent(c.Request.Context(), content); err != nil {
		trace.AddStep(ai.GenerationStagePersist, "保存草稿", "failed", "AI 输出保存数据库失败。", nil, []string{err.Error()})
		h.recordGeneration(c.Request.Context(), failedGenerationLog(now, workspaceID, userID, primaryKnowledgeBaseID, provider, skill, keywords, chunks, trace, err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "generated content was not persisted"})
		return
	}

	h.mu.Lock()
	h.contents = append([]model.Content{content}, h.contents...)
	h.mu.Unlock()

	generationID := fmt.Sprintf("gen_%d", now.UnixNano())
	trace.AddStep(ai.GenerationStagePersist, "保存草稿", "succeeded", "AI 输出已保存为草稿，未自动发布。", []string{"内容 ID：" + content.ID}, nil)
	trace.Warnings = collectTraceWarnings(trace)

	h.recordGeneration(c.Request.Context(), model.GenerationRequest{
		ID:                    generationID,
		WorkspaceID:           workspaceID,
		UserID:                userID,
		KnowledgeBaseID:       primaryKnowledgeBaseID,
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
		ParsedOutput:          encodeJSON(gin.H{"draft": response.Draft, "trace": trace}),
		InputTokens:           generationUsage.InputTokens,
		OutputTokens:          generationUsage.OutputTokens,
		TotalTokens:           generationUsage.TotalTokens,
		Status:                "succeeded",
		CreatedAt:             now,
	})
	if generationUsage.TotalTokens > 0 {
		response.TokenUsage = generationUsage
		h.recordAITokenUsage(c.Request.Context(), h.buildAITokenUsageEvent(generationID, workspaceID, userID, response, now))
	}

	c.JSON(http.StatusCreated, generateContentResponse{Content: content, Trace: trace})
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

	h.mu.RLock()
	content, contentOK := h.contentByID(workspaceID, contentID)
	account, accountOK := h.mediaAccountByID(workspaceID, accountID)
	if !contentOK {
		h.mu.RUnlock()
		c.JSON(http.StatusNotFound, gin.H{"error": "content not found"})
		return
	}
	if !accountOK {
		h.mu.RUnlock()
		c.JSON(http.StatusNotFound, gin.H{"error": "media account not found"})
		return
	}
	platform, _ := h.mediaPlatformByID(account.PlatformID)
	h.mu.RUnlock()

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
		ID:                  fmt.Sprintf("job_%d", now.UnixNano()),
		WorkspaceID:         workspaceID,
		ScheduleID:          schedule.ID,
		ContentID:           content.ID,
		MediaAccountID:      account.ID,
		Status:              model.PublishJobQueued,
		ScheduledAt:         schedule.NextRunAt,
		LastMessage:         "Waiting for scheduled publish window.",
		AttributionMetadata: publishAttributionMetadata(content.ID, account.ID, platform.ID, schedule.ID, "publish_schedule", now),
	}

	if err := h.savePublishScheduleWithJob(c.Request.Context(), schedule, job); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "publish schedule was not persisted"})
		return
	}

	h.mu.Lock()
	h.schedules = append([]model.PublishSchedule{schedule}, h.schedules...)
	h.jobs = append([]model.PublishJob{job}, h.jobs...)
	h.mu.Unlock()
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
		ID:                  fmt.Sprintf("job_%d", now.UnixNano()),
		WorkspaceID:         workspaceID,
		ContentID:           content.ID,
		MediaAccountID:      account.ID,
		Status:              model.PublishJobManual,
		ScheduledAt:         now,
		LastMessage:         "小红书长文发布内容已生成，等待确认后通过浏览器发布。",
		AttributionMetadata: publishAttributionMetadata(content.ID, account.ID, platform.ID, "", "publish_prepare", now),
	}
	scheduledContent := content
	scheduledContent.Status = model.ContentScheduled
	scheduledContent.AttributedMediaAccountID = account.ID
	scheduledContent.AttributionMetadata = contentAttributionMetadata(scheduledContent.AttributionMetadata, account.ID, "publish_prepare", now)
	scheduledContent.UpdatedAt = now

	if err := h.savePublishJobWithContent(c.Request.Context(), job, scheduledContent); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "publish preparation was not persisted"})
		return
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

	var updatedJob model.PublishJob
	var updatedContent model.Content
	h.mu.Lock()
	for index := range h.jobs {
		job := &h.jobs[index]
		if job.WorkspaceID != workspaceID || job.ID != jobID {
			continue
		}

		job.Status = model.PublishJobSucceeded
		job.ExternalURL = externalURL
		job.LastMessage = defaultString(strings.TrimSpace(req.Message), "已人工确认发布完成。")
		job.AttributionMetadata = mergeAnyMaps(job.AttributionMetadata, map[string]any{
			"externalUrl":       externalURL,
			"attributionSource": "manual_publish_confirm",
			"attributedAt":      time.Now().UTC().Format(time.RFC3339),
		})
		h.updateContentStatusLocked(workspaceID, job.ContentID, model.ContentPublished)
		updatedJob = *job
		updatedContent, _ = h.contentByID(workspaceID, job.ContentID)
		if updatedContent.ID != "" {
			updatedContent.AttributedMediaAccountID = job.MediaAccountID
			updatedContent.AttributionMetadata = contentAttributionMetadata(updatedContent.AttributionMetadata, job.MediaAccountID, "manual_publish_confirm", time.Now().UTC())
			h.replaceContentLocked(updatedContent)
		}
		break
	}
	h.mu.Unlock()

	if updatedJob.ID == "" {
		c.JSON(http.StatusNotFound, gin.H{"error": "publish job not found"})
		return
	}
	if updatedContent.ID != "" {
		if err := h.savePublishJobWithContent(c.Request.Context(), updatedJob, updatedContent); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "publish confirmation was not persisted"})
			return
		}
	} else if err := h.savePublishJob(c.Request.Context(), updatedJob); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "publish job was not persisted"})
		return
	}

	c.JSON(http.StatusOK, updatedJob)
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
	runningJob := h.updateJobStatusLocked(workspaceID, jobID, model.PublishJobRunning, "", "正在通过已登录浏览器发布小红书长文。")
	h.mu.Unlock()
	if runningJob.ID != "" {
		if err := h.savePublishJob(ctx, runningJob); err != nil {
			return publishing.PublishResult{}, fmt.Errorf("persist running publish job: %w", err)
		}
	}

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
		failedJob := h.updateJobStatusLocked(workspaceID, jobID, model.PublishJobFailed, "", err.Error())
		if failedJob.ID != "" {
			failedJob.AttributionMetadata = mergeAnyMaps(failedJob.AttributionMetadata, map[string]any{
				"attributionSource": "publish_run_failed",
				"platformId":        platform.ID,
				"errorMessage":      err.Error(),
				"attributedAt":      time.Now().UTC().Format(time.RFC3339),
			})
			h.replacePublishJobLocked(failedJob)
		}
		h.mu.Unlock()
		if failedJob.ID != "" {
			if saveErr := h.savePublishJob(ctx, failedJob); saveErr != nil {
				return publishing.PublishResult{}, fmt.Errorf("publish failed: %w; persist failed job: %w", err, saveErr)
			}
		}
		return publishing.PublishResult{}, err
	}

	status := model.PublishJobManual
	message := defaultString(result.Message, "已打开浏览器并完成小红书发布准备。")
	if publishResultSucceeded(result) {
		status = model.PublishJobSucceeded
	} else if result.Status == "submitted_pending_verification" {
		status = model.PublishJobManual
		if strings.TrimSpace(message) == "" {
			message = "已尝试浏览器发布，但未检测到小红书明确成功提示，请人工核对后确认。"
		}
	}

	h.mu.Lock()
	job := h.updateJobStatusLocked(workspaceID, jobID, status, result.ExternalURL, message)
	if job.ID != "" {
		job.AttributionMetadata = mergeAnyMaps(job.AttributionMetadata, map[string]any{
			"platformId":        platform.ID,
			"externalUrl":       result.ExternalURL,
			"externalId":        result.ExternalID,
			"publishResult":     result.Status,
			"attributionSource": "publish_run",
			"attributedAt":      time.Now().UTC().Format(time.RFC3339),
		})
		h.replacePublishJobLocked(job)
	}
	var publishedContent model.Content
	if publishResultSucceeded(result) {
		h.updateContentStatusLocked(workspaceID, job.ContentID, model.ContentPublished)
		publishedContent, _ = h.contentByID(workspaceID, job.ContentID)
		if publishedContent.ID != "" {
			publishedContent.AttributedMediaAccountID = account.ID
			publishedContent.AttributionMetadata = contentAttributionMetadata(publishedContent.AttributionMetadata, account.ID, "publish_run", time.Now().UTC())
			h.replaceContentLocked(publishedContent)
		}
	}
	h.mu.Unlock()
	if job.ID != "" {
		if publishedContent.ID != "" {
			if err := h.savePublishJobWithContent(ctx, job, publishedContent); err != nil {
				return publishing.PublishResult{}, fmt.Errorf("persist publish result: %w", err)
			}
		} else if err := h.savePublishJob(ctx, job); err != nil {
			return publishing.PublishResult{}, fmt.Errorf("persist publish job: %w", err)
		}
	}

	return result, nil
}

func (h *WorkspaceHandler) AdminOverview(c *gin.Context) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	c.JSON(http.StatusOK, gin.H{
		"userCount":                  len(h.users),
		"workspaceCount":             len(h.workspaces),
		"memberCount":                len(h.members),
		"platformKnowledgeBaseCount": len(h.platformKnowledgeBases),
		"platformKnowledgeItemCount": len(h.platformKnowledgeItems),
		"mediaPlatformCount":         len(h.platforms),
		"mediaAccountCount":          len(h.accounts),
		"contentCount":               len(h.contents),
		"publishJobCount":            len(h.jobs),
	})
}

func (h *WorkspaceHandler) AdminListUsers(c *gin.Context) {
	h.mu.RLock()
	items := append([]model.User(nil), h.users...)
	h.mu.RUnlock()
	c.JSON(http.StatusOK, gin.H{"items": items})
}

func (h *WorkspaceHandler) AdminUpdateUserSubscription(c *gin.Context) {
	userID := strings.TrimSpace(c.Param("userId"))
	if userID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "user id is required"})
		return
	}

	var req updateUserSubscriptionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	tier := req.SubscriptionTier
	if tier == "" {
		tier = model.SubscriptionTierFree
	}
	if tier != model.SubscriptionTierFree && tier != model.SubscriptionTierVIP {
		c.JSON(http.StatusBadRequest, gin.H{"error": "unsupported subscription tier"})
		return
	}

	status := req.SubscriptionStatus
	if status == "" {
		status = model.SubscriptionStatusActive
	}
	if !validSubscriptionStatus(status) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "unsupported subscription status"})
		return
	}

	expiresAt, ok := parseOptionalTime(req.SubscriptionExpiresAt)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid subscriptionExpiresAt"})
		return
	}

	h.mu.Lock()
	var updatedUser model.User
	for index := range h.users {
		user := &h.users[index]
		if user.ID != userID {
			continue
		}
		plan, _ := h.subscriptionPlanByTierLocked(tier)
		user.SubscriptionTier = tier
		user.SubscriptionPlanID = plan.ID
		user.SubscriptionStatus = status
		user.SubscriptionExpiresAt = expiresAt
		user.MonthlyTokenBudgetCents = plan.MonthlyTokenBudgetCents
		user.SubscriptionCurrentPeriod = defaultString(user.SubscriptionCurrentPeriod, time.Now().UTC().Format("2006-01"))
		updatedUser = *user
		break
	}
	h.mu.Unlock()

	if updatedUser.ID == "" {
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}

	if err := h.updateUserSubscription(c.Request.Context(), updatedUser); err != nil {
		log.Printf("user subscription was not persisted: %v", err)
	}

	c.JSON(http.StatusOK, updatedUser)
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

func (h *WorkspaceHandler) AdminListPlatformKnowledgeBases(c *gin.Context) {
	h.mu.RLock()
	items := append([]model.PlatformKnowledgeBase(nil), h.platformKnowledgeBases...)
	h.mu.RUnlock()
	c.JSON(http.StatusOK, gin.H{"items": items})
}

func (h *WorkspaceHandler) AdminCreatePlatformKnowledgeBase(c *gin.Context) {
	var req createPlatformKnowledgeBaseRequest
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
	item := model.PlatformKnowledgeBase{
		ID:                fmt.Sprintf("pkb_%d", now.UnixNano()),
		Name:              name,
		Description:       strings.TrimSpace(req.Description),
		Category:          defaultString(strings.TrimSpace(req.Category), "general"),
		PriceCents:        req.PriceCents,
		Currency:          defaultString(strings.ToUpper(strings.TrimSpace(req.Currency)), "CNY"),
		MarketplaceListed: req.MarketplaceListed,
		ItemCount:         0,
		UpdatedAt:         now,
	}

	if err := h.savePlatformKnowledgeBase(c.Request.Context(), item); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "platform knowledge base was not persisted"})
		return
	}

	h.mu.Lock()
	h.platformKnowledgeBases = append([]model.PlatformKnowledgeBase{item}, h.platformKnowledgeBases...)
	h.mu.Unlock()
	c.JSON(http.StatusCreated, item)
}

func (h *WorkspaceHandler) AdminUpdatePlatformKnowledgeBase(c *gin.Context) {
	knowledgeBaseID := strings.TrimSpace(c.Param("knowledgeBaseId"))
	if knowledgeBaseID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "knowledgeBaseId is required"})
		return
	}

	var req createPlatformKnowledgeBaseRequest
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
	h.mu.Lock()
	var updated model.PlatformKnowledgeBase
	for index := range h.platformKnowledgeBases {
		item := &h.platformKnowledgeBases[index]
		if item.ID != knowledgeBaseID {
			continue
		}
		item.Name = name
		item.Description = strings.TrimSpace(req.Description)
		item.Category = defaultString(strings.TrimSpace(req.Category), "general")
		item.PriceCents = req.PriceCents
		item.Currency = defaultString(strings.ToUpper(strings.TrimSpace(req.Currency)), "CNY")
		item.MarketplaceListed = req.MarketplaceListed
		item.UpdatedAt = now
		updated = *item
		break
	}
	h.mu.Unlock()

	if updated.ID == "" {
		c.JSON(http.StatusNotFound, gin.H{"error": "platform knowledge base not found"})
		return
	}
	if err := h.savePlatformKnowledgeBase(c.Request.Context(), updated); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "platform knowledge base was not persisted"})
		return
	}

	c.JSON(http.StatusOK, updated)
}

func (h *WorkspaceHandler) AdminListPlatformKnowledgeItems(c *gin.Context) {
	h.mu.RLock()
	items := append([]model.PlatformKnowledgeItem(nil), h.platformKnowledgeItems...)
	h.mu.RUnlock()
	c.JSON(http.StatusOK, gin.H{"items": items})
}

func (h *WorkspaceHandler) AdminCreatePlatformKnowledgeItem(c *gin.Context) {
	var req createPlatformKnowledgeItemRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	knowledgeBaseIDs := cleanKnowledgeBaseIDs(req.KnowledgeBaseID, req.KnowledgeBaseIDs)
	title := strings.TrimSpace(req.Title)
	content := strings.TrimSpace(req.Content)
	if len(knowledgeBaseIDs) == 0 || title == "" || content == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "knowledgeBaseIds, title and content are required"})
		return
	}

	now := time.Now().UTC()
	item := model.PlatformKnowledgeItem{
		ID:               fmt.Sprintf("pki_%d", now.UnixNano()),
		KnowledgeBaseIDs: knowledgeBaseIDs,
		Type:             defaultString(strings.TrimSpace(req.Type), "note"),
		Title:            title,
		Content:          content,
		Enabled:          optionalBool(req.Enabled, true),
		UpdatedAt:        now,
	}

	h.mu.RLock()
	if !h.hasPlatformKnowledgeBasesLocked(knowledgeBaseIDs) {
		h.mu.RUnlock()
		c.JSON(http.StatusNotFound, gin.H{"error": "platform knowledge base not found"})
		return
	}
	h.mu.RUnlock()

	if err := h.savePlatformKnowledgeItem(c.Request.Context(), item); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "platform knowledge item was not persisted"})
		return
	}

	h.mu.Lock()
	h.platformKnowledgeItems = append([]model.PlatformKnowledgeItem{item}, h.platformKnowledgeItems...)
	h.recountPlatformKnowledgeBaseItemsLocked()
	h.mu.Unlock()
	c.JSON(http.StatusCreated, item)
}

func (h *WorkspaceHandler) AdminUpdatePlatformKnowledgeItem(c *gin.Context) {
	knowledgeItemID := strings.TrimSpace(c.Param("knowledgeItemId"))
	if knowledgeItemID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "knowledgeItemId is required"})
		return
	}

	var req createPlatformKnowledgeItemRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	knowledgeBaseIDs := cleanKnowledgeBaseIDs(req.KnowledgeBaseID, req.KnowledgeBaseIDs)
	title := strings.TrimSpace(req.Title)
	content := strings.TrimSpace(req.Content)
	if len(knowledgeBaseIDs) == 0 || title == "" || content == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "knowledgeBaseIds, title and content are required"})
		return
	}

	now := time.Now().UTC()
	h.mu.RLock()
	if !h.hasPlatformKnowledgeBasesLocked(knowledgeBaseIDs) {
		h.mu.RUnlock()
		c.JSON(http.StatusNotFound, gin.H{"error": "platform knowledge base not found"})
		return
	}
	h.mu.RUnlock()

	var updated model.PlatformKnowledgeItem
	h.mu.Lock()
	for index := range h.platformKnowledgeItems {
		item := &h.platformKnowledgeItems[index]
		if item.ID != knowledgeItemID {
			continue
		}
		item.KnowledgeBaseIDs = knowledgeBaseIDs
		item.Type = defaultString(strings.TrimSpace(req.Type), "note")
		item.Title = title
		item.Content = content
		item.Enabled = optionalBool(req.Enabled, true)
		item.UpdatedAt = now
		h.recountPlatformKnowledgeBaseItemsLocked()
		updated = *item
		break
	}
	h.mu.Unlock()

	if updated.ID == "" {
		c.JSON(http.StatusNotFound, gin.H{"error": "platform knowledge item not found"})
		return
	}
	if err := h.savePlatformKnowledgeItem(c.Request.Context(), updated); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "platform knowledge item was not persisted"})
		return
	}

	c.JSON(http.StatusOK, updated)
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
	if platformType != xiaohongshu.PlatformType {
		c.JSON(http.StatusBadRequest, gin.H{"error": "only xiaohongshu media platform is supported"})
		return
	}

	h.mu.RLock()
	for _, platform := range h.platforms {
		if platform.Type == platformType {
			h.mu.RUnlock()
			c.JSON(http.StatusConflict, gin.H{"error": "media platform type already exists"})
			return
		}
	}
	h.mu.RUnlock()

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
	if platform.ID != "plt_xiaohongshu" {
		platform.ID = "plt_xiaohongshu"
	}
	if len(platform.CredentialFields) == 0 {
		platform.CredentialFields = []string{"qrLogin"}
	}

	if err := h.saveMediaPlatform(c.Request.Context(), platform); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "media platform was not persisted"})
		return
	}

	h.mu.Lock()
	h.platforms = append([]model.MediaPlatform{platform}, h.platforms...)
	h.mu.Unlock()
	c.JSON(http.StatusCreated, platform)
}

func (h *WorkspaceHandler) AdminUpdateMediaPlatform(c *gin.Context) {
	platformID := strings.TrimSpace(c.Param("platformId"))
	if platformID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "platformId is required"})
		return
	}

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
	if platformType != xiaohongshu.PlatformType {
		c.JSON(http.StatusBadRequest, gin.H{"error": "only xiaohongshu media platform is supported"})
		return
	}

	h.mu.RLock()
	platformIndex := -1
	for index, platform := range h.platforms {
		if platform.ID == platformID {
			platformIndex = index
			continue
		}
		if platform.Type == platformType {
			h.mu.RUnlock()
			c.JSON(http.StatusConflict, gin.H{"error": "media platform type already exists"})
			return
		}
	}
	if platformIndex == -1 {
		h.mu.RUnlock()
		c.JSON(http.StatusNotFound, gin.H{"error": "media platform not found"})
		return
	}
	h.mu.RUnlock()

	updated := model.MediaPlatform{
		ID:                 platformID,
		Name:               name,
		Type:               platformType,
		Enabled:            req.Enabled,
		SupportsArticle:    req.SupportsArticle,
		SupportsImage:      req.SupportsImage,
		SupportsScheduling: req.SupportsScheduling,
		CredentialFields:   cleanKeywords(req.CredentialFields),
	}
	if len(updated.CredentialFields) == 0 {
		updated.CredentialFields = []string{"qrLogin"}
	}
	if err := h.saveMediaPlatform(c.Request.Context(), updated); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "media platform was not persisted"})
		return
	}

	h.mu.Lock()
	h.platforms[platformIndex] = updated
	h.mu.Unlock()
	c.JSON(http.StatusOK, updated)
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

	next := ai.Config{
		Provider:           provider,
		OpenAIAPIKey:       strings.TrimSpace(req.OpenAIAPIKey),
		OpenAIBaseURL:      strings.TrimSpace(req.OpenAIBaseURL),
		OpenAIModel:        strings.TrimSpace(req.OpenAIModel),
		RequestTimeout:     req.RequestTimeoutSeconds,
		GenerationPipeline: req.GenerationPipeline,
	}
	current := h.aiConfig.Snapshot()
	if next.OpenAIAPIKey == "" && !req.ClearAPIKey {
		next.OpenAIAPIKey = current.OpenAIAPIKey
	}
	if req.ClearAPIKey {
		next.OpenAIAPIKey = ""
	}
	updated := ai.NewRuntimeConfig(next).Snapshot()
	if h.db != nil && h.db.SQL() != nil {
		persistCtx, cancel := context.WithTimeout(c.Request.Context(), 3*time.Second)
		defer cancel()
		if err := systemconfig.SaveAIConfig(persistCtx, h.db, updated, req.ClearAPIKey, middleware.CurrentUserID(c)); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "AI config was not persisted"})
			return
		}
	}
	h.aiConfig.Update(updated, req.ClearAPIKey)

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

func (h *WorkspaceHandler) seedDatabase(ctx context.Context) bool {
	dbCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	if err := h.db.SeedWorkspaceData(
		dbCtx,
		h.users,
		h.workspaces,
		h.members,
		h.knowledgeBases,
		h.knowledgeItems,
		h.platformKnowledgeBases,
		h.platformKnowledgeItems,
		h.platforms,
		h.accounts,
		h.contents,
		h.schedules,
		h.jobs,
	); err != nil {
		log.Printf("database seed failed: %v", err)
		return false
	}
	return true
}

func (h *WorkspaceHandler) databaseSnapshotMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if !h.loadDatabaseSnapshot(c.Request.Context()) {
			c.AbortWithStatusJSON(http.StatusServiceUnavailable, gin.H{"error": "database snapshot load failed"})
			return
		}
		c.Next()
	}
}

func (h *WorkspaceHandler) loadDatabaseSnapshot(ctx context.Context) bool {
	if h.db == nil || h.db.SQL() == nil {
		return true
	}

	dbCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	snapshot, err := h.db.LoadSnapshot(dbCtx)
	if err != nil {
		log.Printf("database snapshot load failed: %v", err)
		return false
	}

	h.mu.Lock()
	h.users = snapshot.Users
	h.subscriptionPlans = snapshot.SubscriptionPlans
	h.workspaces = snapshot.Workspaces
	h.members = snapshot.Members
	h.knowledgeBases = snapshot.KnowledgeBases
	h.knowledgeItems = snapshot.KnowledgeItems
	h.platformKnowledgeBases = snapshot.PlatformKnowledgeBases
	h.platformKnowledgeItems = snapshot.PlatformKnowledgeItems
	h.platforms = snapshot.Platforms
	h.accounts = snapshot.Accounts
	h.contents = snapshot.Contents
	h.schedules = snapshot.Schedules
	h.jobs = snapshot.Jobs
	h.generations = snapshot.Generations
	h.tokenUsageEvents = snapshot.TokenUsageEvents
	h.mu.Unlock()

	return true
}

func (h *WorkspaceHandler) createLoginSession(ctx context.Context, userID string) (string, error) {
	token, err := secureRandomID("gp")
	if err != nil {
		return "", err
	}
	if h.db == nil || h.db.SQL() == nil {
		h.mu.Lock()
		h.userSessions[token] = userID
		h.mu.Unlock()
		return token, nil
	}

	expiresAt := time.Now().UTC().AddDate(0, 0, 30)
	dbCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	if err := h.db.CreateUserSession(dbCtx, token, userID, &expiresAt); err != nil {
		return "", err
	}
	return token, nil
}

func (h *WorkspaceHandler) ResolveUserSession(_ context.Context, token string) (string, bool, error) {
	h.mu.RLock()
	userID, ok := h.userSessions[token]
	h.mu.RUnlock()
	return userID, ok, nil
}

func secureRandomID(prefix string) (string, error) {
	data := make([]byte, 24)
	if _, err := rand.Read(data); err != nil {
		return "", err
	}
	return fmt.Sprintf("%s_%x", prefix, data), nil
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

func (h *WorkspaceHandler) subscriptionPlanByID(planID model.SubscriptionPlanID) (model.SubscriptionPlan, bool) {
	for _, plan := range h.subscriptionPlans {
		if plan.ID == planID {
			return plan, true
		}
	}
	return model.SubscriptionPlan{}, false
}

func (h *WorkspaceHandler) subscriptionPlanByTierLocked(tier model.SubscriptionTier) (model.SubscriptionPlan, bool) {
	for _, plan := range h.subscriptionPlans {
		if plan.Tier == tier && plan.Enabled {
			return plan, true
		}
	}
	for _, plan := range h.subscriptionPlans {
		if plan.ID == model.SubscriptionPlanFree {
			return plan, true
		}
	}
	return model.SubscriptionPlan{ID: model.SubscriptionPlanFree, Tier: model.SubscriptionTierFree, Enabled: true}, false
}

func (h *WorkspaceHandler) emailExistsLocked(email string) bool {
	for _, user := range h.users {
		if strings.EqualFold(user.Email, email) {
			return true
		}
	}
	return false
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

func (h *WorkspaceHandler) replacePublishJobLocked(updated model.PublishJob) {
	for index := range h.jobs {
		if h.jobs[index].WorkspaceID == updated.WorkspaceID && h.jobs[index].ID == updated.ID {
			h.jobs[index] = updated
			return
		}
	}
}

func (h *WorkspaceHandler) replaceContentLocked(updated model.Content) {
	for index := range h.contents {
		if h.contents[index].WorkspaceID == updated.WorkspaceID && h.contents[index].ID == updated.ID {
			h.contents[index] = updated
			return
		}
	}
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

func (h *WorkspaceHandler) recountKnowledgeBaseItemsLocked(workspaceID string) {
	counts := map[string]int{}
	for _, item := range h.knowledgeItems {
		if item.WorkspaceID != workspaceID {
			continue
		}
		for _, knowledgeBaseID := range item.KnowledgeBaseIDs {
			counts[knowledgeBaseID]++
		}
	}

	now := time.Now().UTC()
	for index := range h.knowledgeBases {
		item := &h.knowledgeBases[index]
		if item.WorkspaceID != workspaceID {
			continue
		}
		item.ItemCount = counts[item.ID]
		item.UpdatedAt = now
	}
}

func (h *WorkspaceHandler) hasKnowledgeBasesLocked(workspaceID string, knowledgeBaseIDs []string) bool {
	for _, knowledgeBaseID := range knowledgeBaseIDs {
		found := false
		for _, item := range h.knowledgeBases {
			if item.WorkspaceID == workspaceID && item.ID == knowledgeBaseID {
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

func (h *WorkspaceHandler) bumpPlatformKnowledgeBaseCount(knowledgeBaseID string, delta int) {
	for index := range h.platformKnowledgeBases {
		item := &h.platformKnowledgeBases[index]
		if item.ID == knowledgeBaseID {
			item.ItemCount += delta
			if item.ItemCount < 0 {
				item.ItemCount = 0
			}
			item.UpdatedAt = time.Now().UTC()
			return
		}
	}
}

func (h *WorkspaceHandler) recountPlatformKnowledgeBaseItemsLocked() {
	counts := map[string]int{}
	for _, item := range h.platformKnowledgeItems {
		for _, knowledgeBaseID := range item.KnowledgeBaseIDs {
			counts[knowledgeBaseID]++
		}
	}

	now := time.Now().UTC()
	for index := range h.platformKnowledgeBases {
		item := &h.platformKnowledgeBases[index]
		item.ItemCount = counts[item.ID]
		item.UpdatedAt = now
	}
}

func (h *WorkspaceHandler) hasPlatformKnowledgeBasesLocked(knowledgeBaseIDs []string) bool {
	for _, knowledgeBaseID := range knowledgeBaseIDs {
		if !hasPlatformKnowledgeBase(h.platformKnowledgeBases, knowledgeBaseID) {
			return false
		}
	}
	return true
}

func (h *WorkspaceHandler) touchPlatformKnowledgeBase(knowledgeBaseID string, updatedAt time.Time) {
	for index := range h.platformKnowledgeBases {
		item := &h.platformKnowledgeBases[index]
		if item.ID == knowledgeBaseID {
			item.UpdatedAt = updatedAt
			return
		}
	}
}

func (h *WorkspaceHandler) retrieveKnowledgeChunksLocked(workspaceID string, knowledgeBaseIDs []string, keywords []string, limit int) []ai.KnowledgeChunk {
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
		if len(knowledgeBaseIDs) > 0 && !intersectsString(item.KnowledgeBaseIDs, knowledgeBaseIDs) {
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
			ID:               item.item.ID,
			KnowledgeBaseIDs: item.item.KnowledgeBaseIDs,
			Type:             item.item.Type,
			Title:            item.item.Title,
			Content:          item.item.Content,
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

func (h *WorkspaceHandler) saveKnowledgeBase(ctx context.Context, item model.KnowledgeBase) error {
	dbCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	return h.db.SaveKnowledgeBase(dbCtx, item)
}

func (h *WorkspaceHandler) saveKnowledgeItem(ctx context.Context, item model.KnowledgeItem) error {
	dbCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	return h.db.SaveKnowledgeItem(dbCtx, item)
}

func (h *WorkspaceHandler) assignKnowledgeItemsToBases(ctx context.Context, workspaceID string, knowledgeItemIDs []string, knowledgeBaseIDs []string) error {
	dbCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	return h.db.AssignKnowledgeItemsToBases(dbCtx, workspaceID, knowledgeItemIDs, knowledgeBaseIDs)
}

func (h *WorkspaceHandler) savePlatformKnowledgeBase(ctx context.Context, item model.PlatformKnowledgeBase) error {
	dbCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	return h.db.SavePlatformKnowledgeBase(dbCtx, item)
}

func (h *WorkspaceHandler) savePlatformKnowledgeItem(ctx context.Context, item model.PlatformKnowledgeItem) error {
	dbCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	return h.db.SavePlatformKnowledgeItem(dbCtx, item)
}

func (h *WorkspaceHandler) saveMediaPlatform(ctx context.Context, item model.MediaPlatform) error {
	dbCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	return h.db.SaveMediaPlatform(dbCtx, item)
}

func (h *WorkspaceHandler) saveMediaAccount(ctx context.Context, item model.MediaAccount) error {
	dbCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	return h.db.SaveMediaAccount(dbCtx, item)
}

func (h *WorkspaceHandler) saveMediaAccountLoginSession(ctx context.Context, item model.MediaAccountLoginSession) error {
	if h.db == nil || h.db.SQL() == nil {
		return nil
	}
	dbCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	return h.db.SaveMediaAccountLoginSession(dbCtx, item)
}

func (h *WorkspaceHandler) latestMediaAccountLoginSession(ctx context.Context, workspaceID string, accountID string) (model.MediaAccountLoginSession, bool, error) {
	if h.db != nil && h.db.SQL() != nil {
		dbCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
		defer cancel()
		return h.db.LatestMediaAccountLoginSession(dbCtx, workspaceID, accountID)
	}

	h.mu.RLock()
	account, ok := h.mediaAccountByID(workspaceID, accountID)
	h.mu.RUnlock()
	if !ok {
		return model.MediaAccountLoginSession{}, false, nil
	}
	return mediaAccountLoginSessionFromMetadata(account)
}

func (h *WorkspaceHandler) completeMediaAccountLoginSession(ctx context.Context, sessionID string) error {
	if h.db == nil || h.db.SQL() == nil {
		return nil
	}
	dbCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	return h.db.CompleteMediaAccountLoginSession(dbCtx, sessionID)
}

func (h *WorkspaceHandler) expireMediaAccountLoginSession(ctx context.Context, sessionID string) error {
	if h.db == nil || h.db.SQL() == nil {
		return nil
	}
	dbCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	return h.db.ExpireMediaAccountLoginSession(dbCtx, sessionID)
}

func (h *WorkspaceHandler) savePublishScheduleWithJob(ctx context.Context, schedule model.PublishSchedule, job model.PublishJob) error {
	dbCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	return h.db.SavePublishScheduleWithJob(dbCtx, schedule, job)
}

func (h *WorkspaceHandler) savePublishJob(ctx context.Context, item model.PublishJob) error {
	dbCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	return h.db.SavePublishJob(dbCtx, item)
}

func (h *WorkspaceHandler) savePublishJobWithContent(ctx context.Context, job model.PublishJob, content model.Content) error {
	dbCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	return h.db.SavePublishJobWithContent(dbCtx, job, content)
}

func (h *WorkspaceHandler) updateUserSubscription(ctx context.Context, user model.User) error {
	if h.db == nil || h.db.SQL() == nil {
		return nil
	}
	dbCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	return h.db.UpdateUserSubscription(dbCtx, user)
}

func (h *WorkspaceHandler) completeUserOnboarding(ctx context.Context, user model.User, workspace model.Workspace) error {
	if h.db == nil || h.db.SQL() == nil {
		return nil
	}
	dbCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	return h.db.CompleteUserOnboarding(dbCtx, user, workspace)
}

func (h *WorkspaceHandler) recordAITokenUsage(ctx context.Context, event model.AITokenUsageEvent) {
	if event.ID == "" || event.UserID == "" || event.WorkspaceID == "" {
		return
	}

	h.mu.Lock()
	h.tokenUsageEvents = append([]model.AITokenUsageEvent{event}, h.tokenUsageEvents...)
	for index := range h.users {
		if h.users[index].ID != event.UserID {
			continue
		}
		h.users[index].MonthlyTokenUsedCents += event.TotalCostCents
		h.users[index].MonthlyTokenInputUsed += event.InputTokens
		h.users[index].MonthlyTokenOutputUsed += event.OutputTokens
		h.users[index].SubscriptionCurrentPeriod = event.BillingPeriod
		break
	}
	h.mu.Unlock()

	if h.db == nil || h.db.SQL() == nil {
		return
	}
	dbCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	if err := h.db.RecordAITokenUsageEvent(dbCtx, event); err != nil {
		log.Printf("ai token usage event was not persisted: %v", err)
	}
}

func (h *WorkspaceHandler) buildAITokenUsageEvent(generationID, workspaceID, userID string, response ai.GenerateResponse, now time.Time) model.AITokenUsageEvent {
	h.mu.RLock()
	user, _ := h.userByID(userID)
	plan, planOK := h.subscriptionPlanByID(user.SubscriptionPlanID)
	h.mu.RUnlock()
	if !planOK {
		plan, _ = h.subscriptionPlanByID(model.SubscriptionPlanFree)
	}
	inputCost := billableTokenCostCents(response.TokenUsage.InputTokens, plan.InputTokenPricePer1K)
	outputCost := billableTokenCostCents(response.TokenUsage.OutputTokens, plan.OutputTokenPricePer1K)
	return model.AITokenUsageEvent{
		ID:                  fmt.Sprintf("aiu_%d", now.UnixNano()),
		UserID:              userID,
		WorkspaceID:         workspaceID,
		GenerationRequestID: generationID,
		Provider:            response.Provider,
		Model:               response.Model,
		SubscriptionPlanID:  plan.ID,
		InputTokens:         response.TokenUsage.InputTokens,
		OutputTokens:        response.TokenUsage.OutputTokens,
		TotalTokens:         response.TokenUsage.TotalTokens,
		InputCostCents:      inputCost,
		OutputCostCents:     outputCost,
		TotalCostCents:      inputCost + outputCost,
		BillingPeriod:       defaultString(user.SubscriptionCurrentPeriod, now.Format("2006-01")),
		CreatedAt:           now,
	}
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

func supportsBrowserLogin(platformType string) bool {
	return platformType == xiaohongshu.PlatformType
}

func mediaAccountLoginSessionFromMetadata(account model.MediaAccount) (model.MediaAccountLoginSession, bool, error) {
	meta := account.CredentialMeta
	if meta == nil {
		return model.MediaAccountLoginSession{}, false, nil
	}
	sessionID := strings.TrimSpace(meta["loginSessionId"])
	if sessionID == "" {
		return model.MediaAccountLoginSession{}, false, nil
	}
	expiresAt := time.Now().UTC().Add(5 * time.Minute)
	if rawExpiresAt := strings.TrimSpace(meta["loginSessionExpiresAt"]); rawExpiresAt != "" {
		parsed, err := time.Parse(time.RFC3339, rawExpiresAt)
		if err != nil {
			return model.MediaAccountLoginSession{}, false, err
		}
		expiresAt = parsed
	}
	return model.MediaAccountLoginSession{
		ID:          sessionID,
		WorkspaceID: account.WorkspaceID,
		AccountID:   account.ID,
		ProfileDir:  strings.TrimSpace(meta["browserProfile"]),
		LoginURL:    firstNonEmptyString(strings.TrimSpace(meta["browserLoginUrl"]), xiaohongshu.DefaultLoginURL),
		StateFile:   strings.TrimSpace(meta["browserLoginStateFile"]),
		Status:      "active",
		ExpiresAt:   expiresAt,
	}, true, nil
}

func firstNonEmptyString(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func publishResultSucceeded(result publishing.PublishResult) bool {
	if result.Status == "published" {
		return true
	}
	if result.Status != "submitted_pending_verification" {
		return false
	}
	rawStatus, ok := result.RawResponse["rawStatus"].(map[string]any)
	if !ok {
		return false
	}
	publishOutcome, ok := rawStatus["publishOutcome"].(map[string]any)
	if !ok {
		return false
	}
	leftEditor, ok := publishOutcome["leftEditor"].(bool)
	return ok && leftEditor
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

func hasPlatformKnowledgeBase(items []model.PlatformKnowledgeBase, knowledgeBaseID string) bool {
	for _, item := range items {
		if item.ID == knowledgeBaseID {
			return true
		}
	}
	return false
}

func cleanKnowledgeBaseIDs(single string, values []string) []string {
	items := make([]string, 0, len(values)+1)
	if single = strings.TrimSpace(single); single != "" {
		items = append(items, single)
	}
	items = append(items, values...)
	return uniqueStrings(cleanKeywords(items))
}

func uniqueStrings(values []string) []string {
	seen := map[string]bool{}
	result := make([]string, 0, len(values))
	for _, value := range values {
		if value == "" || seen[value] {
			continue
		}
		seen[value] = true
		result = append(result, value)
	}
	return result
}

func mergeStringSet(current []string, additions []string) []string {
	merged := make([]string, 0, len(current)+len(additions))
	merged = append(merged, current...)
	merged = append(merged, additions...)
	return uniqueStrings(cleanKeywords(merged))
}

func containsString(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

func intersectsString(left []string, right []string) bool {
	for _, value := range left {
		if containsString(right, value) {
			return true
		}
	}
	return false
}

func firstString(values []string) string {
	if len(values) == 0 {
		return ""
	}
	return values[0]
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

func nonNilMap(value map[string]any) map[string]any {
	if value == nil {
		return map[string]any{}
	}
	return value
}

func mergeAnyMaps(base map[string]any, overrides map[string]any) map[string]any {
	merged := map[string]any{}
	for key, value := range base {
		merged[key] = value
	}
	for key, value := range overrides {
		if value == nil {
			continue
		}
		if text, ok := value.(string); ok && text == "" {
			continue
		}
		merged[key] = value
	}
	return merged
}

func contentAttributionMetadata(base map[string]any, mediaAccountID string, source string, now time.Time) map[string]any {
	metadata := mergeAnyMaps(base, map[string]any{
		"mediaAccountId":    mediaAccountID,
		"attributionSource": source,
		"attributedAt":      now.UTC().Format(time.RFC3339),
	})
	if mediaAccountID == "" {
		delete(metadata, "mediaAccountId")
	}
	return metadata
}

func publishAttributionMetadata(contentID string, accountID string, platformID string, scheduleID string, source string, now time.Time) map[string]any {
	// 内容表现回流依赖这些稳定键做最小归因，不绑定具体平台抓取实现。
	return mergeAnyMaps(map[string]any{}, map[string]any{
		"contentId":         contentID,
		"mediaAccountId":    accountID,
		"platformId":        platformID,
		"scheduleId":        scheduleID,
		"attributionSource": source,
		"attributedAt":      now.UTC().Format(time.RFC3339),
	})
}

func mediaAccountHealthFromStatus(status string) string {
	switch status {
	case "connected":
		return "healthy"
	case "pending_login", "qr_waiting":
		return "needs_authorization"
	case "expired":
		return "expired"
	default:
		return "unknown"
	}
}

func extractKeywordsFromMarkdownPrompt(value string) []string {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}

	lines := strings.Split(value, "\n")
	inCoreThemes := false
	keywords := []string{}
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "## ") {
			inCoreThemes = strings.Contains(line, "核心主题")
			continue
		}
		if !inCoreThemes {
			continue
		}
		line = strings.TrimSpace(strings.TrimPrefix(strings.TrimPrefix(line, "-"), "*"))
		if line != "" {
			keywords = append(keywords, line)
		}
	}
	if len(keywords) > 0 {
		return cleanKeywords(keywords)
	}

	parts := strings.FieldsFunc(value, func(r rune) bool {
		return r == '\n' || r == '\r' || r == ',' || r == '，' || r == ';' || r == '；' || r == '、'
	})
	return cleanKeywords(parts)
}

func defaultString(value, fallback string) string {
	if value == "" {
		return fallback
	}
	return value
}

func validSubscriptionStatus(value model.SubscriptionStatus) bool {
	switch value {
	case model.SubscriptionStatusActive, model.SubscriptionStatusInactive, model.SubscriptionStatusExpired, model.SubscriptionStatusCanceled:
		return true
	default:
		return false
	}
}

func parseOptionalTime(value string) (*time.Time, bool) {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil, true
	}
	for _, layout := range []string{time.RFC3339Nano, time.RFC3339, "2006-01-02"} {
		parsed, err := time.Parse(layout, value)
		if err == nil {
			utc := parsed.UTC()
			return &utc, true
		}
	}
	return nil, false
}

func optionalBool(value *bool, fallback bool) bool {
	if value == nil {
		return fallback
	}
	return *value
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

func defaultUserSubscriptionTier(value model.SubscriptionTier) model.SubscriptionTier {
	if value == "" {
		return model.SubscriptionTierFree
	}
	return value
}

func billableTokenCostCents(tokens int, pricePer1K int) int {
	if tokens <= 0 || pricePer1K <= 0 {
		return 0
	}
	return (tokens*pricePer1K + 999) / 1000
}

func knowledgeTraceDetails(chunks []ai.KnowledgeChunk, knowledgeBaseIDs []string) []string {
	details := []string{}
	if len(knowledgeBaseIDs) == 0 {
		details = append(details, "检索范围：全部知识库包。")
	} else {
		details = append(details, fmt.Sprintf("检索范围：%d 个知识库包。", len(knowledgeBaseIDs)))
	}
	if len(chunks) == 0 {
		details = append(details, "未检索到可用知识片段。")
		return details
	}
	for _, chunk := range chunks {
		details = append(details, fmt.Sprintf("%s（%s）", chunk.Title, chunk.Type))
		if len(details) >= 6 {
			break
		}
	}
	return details
}

func warningsForRetrievedChunks(chunks []ai.KnowledgeChunk) []string {
	if len(chunks) == 0 {
		return []string{"没有检索到知识片段，草稿需要人工补充事实依据。"}
	}
	return nil
}

func collectTraceWarnings(trace ai.GenerationTrace) []string {
	warnings := []string{}
	for _, step := range trace.Steps {
		warnings = append(warnings, step.Warnings...)
	}
	return uniqueStrings(cleanKeywords(warnings))
}

func failedGenerationLog(
	now time.Time,
	workspaceID string,
	userID string,
	knowledgeBaseID string,
	provider ai.Provider,
	skill ai.WritingSkill,
	keywords []string,
	chunks []ai.KnowledgeChunk,
	trace ai.GenerationTrace,
	err error,
) model.GenerationRequest {
	return model.GenerationRequest{
		ID:                    fmt.Sprintf("gen_%d", now.UnixNano()),
		WorkspaceID:           workspaceID,
		UserID:                userID,
		KnowledgeBaseID:       knowledgeBaseID,
		Provider:              provider.Name(),
		Model:                 provider.Model(),
		ContentType:           skill.ContentType,
		Keywords:              keywords,
		Prompt:                encodeJSON(gin.H{"trace": trace}),
		PromptVersion:         ai.PromptVersion,
		SkillID:               skill.ID,
		SkillVersion:          skill.Version,
		RetrievedKnowledgeIDs: knowledgeChunkIDs(chunks),
		ParsedOutput:          encodeJSON(trace),
		Status:                "failed",
		ErrorMessage:          err.Error(),
		CreatedAt:             now,
	}
}

func mergeGenerateResponses(current ai.GenerateResponse, next ai.GenerateResponse) ai.GenerateResponse {
	if next.Prompt.System == "" && next.Prompt.User == "" {
		next.Prompt = current.Prompt
	}
	if next.Provider == "" {
		next.Provider = current.Provider
	}
	if next.Model == "" {
		next.Model = current.Model
	}
	if next.PromptVersion == "" {
		next.PromptVersion = current.PromptVersion
	}
	if next.SkillID == "" {
		next.SkillID = current.SkillID
	}
	if next.SkillVersion == "" {
		next.SkillVersion = current.SkillVersion
	}
	if len(next.RetrievedIDs) == 0 {
		next.RetrievedIDs = current.RetrievedIDs
	}
	next.TokenUsage.InputTokens += current.TokenUsage.InputTokens
	next.TokenUsage.OutputTokens += current.TokenUsage.OutputTokens
	next.TokenUsage.TotalTokens += current.TokenUsage.TotalTokens
	return next
}

func addTokenUsage(current ai.TokenUsage, next ai.TokenUsage) ai.TokenUsage {
	current.InputTokens += next.InputTokens
	current.OutputTokens += next.OutputTokens
	current.TotalTokens += next.TotalTokens
	return current
}

func publisherForPlatform(platformType string) (publishing.Publisher, bool) {
	switch platformType {
	case xiaohongshu.PlatformType:
		return xiaohongshu.NewBrowserLongArticlePublisher(), true
	default:
		return nil, false
	}
}

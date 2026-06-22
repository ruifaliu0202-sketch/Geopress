package model

import "time"

type SubscriptionTier string

const (
	SubscriptionTierFree SubscriptionTier = "free"
	SubscriptionTierVIP  SubscriptionTier = "vip"
)

type SubscriptionStatus string

const (
	SubscriptionStatusActive   SubscriptionStatus = "active"
	SubscriptionStatusInactive SubscriptionStatus = "inactive"
	SubscriptionStatusExpired  SubscriptionStatus = "expired"
	SubscriptionStatusCanceled SubscriptionStatus = "canceled"
)

type SubscriptionPlanID string

const (
	SubscriptionPlanFree SubscriptionPlanID = "free"
	SubscriptionPlanVIP  SubscriptionPlanID = "vip"
)

type User struct {
	ID                        string             `json:"id"`
	Name                      string             `json:"name"`
	Email                     string             `json:"email"`
	IsPlatformAdmin           bool               `json:"isPlatformAdmin"`
	SubscriptionTier          SubscriptionTier   `json:"subscriptionTier"`
	SubscriptionPlanID        SubscriptionPlanID `json:"subscriptionPlanId"`
	SubscriptionStatus        SubscriptionStatus `json:"subscriptionStatus"`
	SubscriptionExpiresAt     *time.Time         `json:"subscriptionExpiresAt,omitempty"`
	MonthlyTokenBudgetCents   int                `json:"monthlyTokenBudgetCents"`
	MonthlyTokenUsedCents     int                `json:"monthlyTokenUsedCents"`
	MonthlyTokenInputUsed     int                `json:"monthlyTokenInputUsed"`
	MonthlyTokenOutputUsed    int                `json:"monthlyTokenOutputUsed"`
	SubscriptionCurrentPeriod string             `json:"subscriptionCurrentPeriod"`
	OnboardingCompleted       bool               `json:"onboardingCompleted"`
	OnboardingCompletedAt     *time.Time         `json:"onboardingCompletedAt,omitempty"`
	CreatedAt                 time.Time          `json:"createdAt"`
}

func (user User) HasActiveVIP(now time.Time) bool {
	if user.SubscriptionTier != SubscriptionTierVIP || user.SubscriptionStatus != SubscriptionStatusActive {
		return false
	}
	if user.SubscriptionExpiresAt != nil && !user.SubscriptionExpiresAt.After(now) {
		return false
	}
	return true
}

func (user User) TokenBudgetRemainingCents() int {
	remaining := user.MonthlyTokenBudgetCents - user.MonthlyTokenUsedCents
	if remaining < 0 {
		return 0
	}
	return remaining
}

type SubscriptionPlan struct {
	ID                      SubscriptionPlanID `json:"id"`
	Name                    string             `json:"name"`
	Tier                    SubscriptionTier   `json:"tier"`
	PriceCents              int                `json:"priceCents"`
	Currency                string             `json:"currency"`
	MonthlyTokenBudgetCents int                `json:"monthlyTokenBudgetCents"`
	InputTokenPricePer1K    int                `json:"inputTokenPricePer1k"`
	OutputTokenPricePer1K   int                `json:"outputTokenPricePer1k"`
	Enabled                 bool               `json:"enabled"`
}

type AITokenUsageEvent struct {
	ID                  string             `json:"id"`
	UserID              string             `json:"userId"`
	WorkspaceID         string             `json:"workspaceId"`
	GenerationRequestID string             `json:"generationRequestId"`
	Provider            string             `json:"provider"`
	Model               string             `json:"model"`
	SubscriptionPlanID  SubscriptionPlanID `json:"subscriptionPlanId"`
	InputTokens         int                `json:"inputTokens"`
	OutputTokens        int                `json:"outputTokens"`
	TotalTokens         int                `json:"totalTokens"`
	InputCostCents      int                `json:"inputCostCents"`
	OutputCostCents     int                `json:"outputCostCents"`
	TotalCostCents      int                `json:"totalCostCents"`
	BillingPeriod       string             `json:"billingPeriod"`
	CreatedAt           time.Time          `json:"createdAt"`
}

type WorkspaceType string

const (
	WorkspacePersonal WorkspaceType = "personal"
	WorkspaceCompany  WorkspaceType = "company"
)

type Workspace struct {
	ID        string        `json:"id"`
	Name      string        `json:"name"`
	Type      WorkspaceType `json:"type"`
	Plan      string        `json:"plan"`
	Status    string        `json:"status"`
	Industry  string        `json:"industry"`
	Language  string        `json:"language"`
	Tone      string        `json:"tone"`
	CreatedAt time.Time     `json:"createdAt"`
}

type WorkspaceMember struct {
	UserID      string `json:"userId"`
	WorkspaceID string `json:"workspaceId"`
	Role        string `json:"role"`
}

type KnowledgeBase struct {
	ID          string    `json:"id"`
	WorkspaceID string    `json:"workspaceId"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	ItemCount   int       `json:"itemCount"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

type KnowledgeItem struct {
	ID               string    `json:"id"`
	KnowledgeBaseIDs []string  `json:"knowledgeBaseIds"`
	WorkspaceID      string    `json:"workspaceId"`
	Type             string    `json:"type"`
	Title            string    `json:"title"`
	Content          string    `json:"content"`
	Enabled          bool      `json:"enabled"`
	UpdatedAt        time.Time `json:"updatedAt"`
}

type PlatformKnowledgeBase struct {
	ID                string    `json:"id"`
	Name              string    `json:"name"`
	Description       string    `json:"description"`
	Category          string    `json:"category"`
	PriceCents        int       `json:"priceCents"`
	Currency          string    `json:"currency"`
	MarketplaceListed bool      `json:"marketplaceListed"`
	ItemCount         int       `json:"itemCount"`
	UpdatedAt         time.Time `json:"updatedAt"`
}

type PlatformKnowledgeItem struct {
	ID               string    `json:"id"`
	KnowledgeBaseIDs []string  `json:"knowledgeBaseIds"`
	Type             string    `json:"type"`
	Title            string    `json:"title"`
	Content          string    `json:"content"`
	Enabled          bool      `json:"enabled"`
	UpdatedAt        time.Time `json:"updatedAt"`
}

type MediaPlatform struct {
	ID                 string   `json:"id"`
	Name               string   `json:"name"`
	Type               string   `json:"type"`
	Enabled            bool     `json:"enabled"`
	SupportsArticle    bool     `json:"supportsArticle"`
	SupportsImage      bool     `json:"supportsImage"`
	SupportsScheduling bool     `json:"supportsScheduling"`
	CredentialFields   []string `json:"credentialFields"`
}

type MediaAccount struct {
	ID             string            `json:"id"`
	WorkspaceID    string            `json:"workspaceId"`
	PlatformID     string            `json:"platformId"`
	Name           string            `json:"name"`
	ExternalID     string            `json:"externalId"`
	LoginMethod    string            `json:"loginMethod"`
	CredentialMeta map[string]string `json:"credentialMeta,omitempty"`
	Status         string            `json:"status"`
	ExpiresAt      *time.Time        `json:"expiresAt,omitempty"`
	LastCheckedAt  time.Time         `json:"lastCheckedAt"`
}

type MediaAccountLoginSession struct {
	ID          string
	WorkspaceID string
	AccountID   string
	Platform    string
	ProfileDir  string
	LoginURL    string
	StateFile   string
	Status      string
	ExpiresAt   time.Time
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type ContentStatus string

const (
	ContentDraft     ContentStatus = "draft"
	ContentReview    ContentStatus = "review"
	ContentApproved  ContentStatus = "approved"
	ContentScheduled ContentStatus = "scheduled"
	ContentPublished ContentStatus = "published"
	ContentFailed    ContentStatus = "failed"
	ContentArchived  ContentStatus = "archived"
)

type Content struct {
	ID              string        `json:"id"`
	WorkspaceID     string        `json:"workspaceId"`
	KnowledgeBaseID string        `json:"knowledgeBaseId"`
	Title           string        `json:"title"`
	Summary         string        `json:"summary"`
	Body            string        `json:"body"`
	Keywords        []string      `json:"keywords"`
	Status          ContentStatus `json:"status"`
	Author          string        `json:"author"`
	Source          string        `json:"source"`
	UpdatedAt       time.Time     `json:"updatedAt"`
}

type GenerationRequest struct {
	ID                    string    `json:"id"`
	WorkspaceID           string    `json:"workspaceId"`
	UserID                string    `json:"userId"`
	KnowledgeBaseID       string    `json:"knowledgeBaseId"`
	ContentID             string    `json:"contentId"`
	Provider              string    `json:"provider"`
	Model                 string    `json:"model"`
	ContentType           string    `json:"contentType"`
	Keywords              []string  `json:"keywords"`
	Prompt                string    `json:"prompt"`
	PromptVersion         string    `json:"promptVersion"`
	SkillID               string    `json:"skillId"`
	SkillVersion          string    `json:"skillVersion"`
	RetrievedKnowledgeIDs []string  `json:"retrievedKnowledgeIds"`
	RawOutput             string    `json:"rawOutput"`
	ParsedOutput          string    `json:"parsedOutput"`
	InputTokens           int       `json:"inputTokens"`
	OutputTokens          int       `json:"outputTokens"`
	TotalTokens           int       `json:"totalTokens"`
	Status                string    `json:"status"`
	ErrorMessage          string    `json:"errorMessage"`
	CreatedAt             time.Time `json:"createdAt"`
}

type CampaignStatus string

const (
	CampaignDraft     CampaignStatus = "draft"
	CampaignPlanned   CampaignStatus = "planned"
	CampaignActive    CampaignStatus = "active"
	CampaignPaused    CampaignStatus = "paused"
	CampaignCompleted CampaignStatus = "completed"
	CampaignArchived  CampaignStatus = "archived"
)

type Campaign struct {
	ID              string         `json:"id"`
	WorkspaceID     string         `json:"workspaceId"`
	Name            string         `json:"name"`
	Description     string         `json:"description"`
	Status          CampaignStatus `json:"status"`
	Goal            string         `json:"goal"`
	Products        []string       `json:"products"`
	TargetAudiences []string       `json:"targetAudiences"`
	Channels        []string       `json:"channels"`
	MediaAccountIDs []string       `json:"mediaAccountIds"`
	StartAt         *time.Time     `json:"startAt,omitempty"`
	EndAt           *time.Time     `json:"endAt,omitempty"`
	BudgetCents     int            `json:"budgetCents"`
	Currency        string         `json:"currency"`
	ContentQuota    int            `json:"contentQuota"`
	ApprovalPolicy  string         `json:"approvalPolicy"`
	SuccessMetrics  []string       `json:"successMetrics"`
	Metadata        map[string]any `json:"metadata"`
	CreatedAt       time.Time      `json:"createdAt"`
	UpdatedAt       time.Time      `json:"updatedAt"`
}

type CampaignTopicStatus string

const (
	CampaignTopicIdea      CampaignTopicStatus = "idea"
	CampaignTopicPlanned   CampaignTopicStatus = "planned"
	CampaignTopicDrafted   CampaignTopicStatus = "drafted"
	CampaignTopicApproved  CampaignTopicStatus = "approved"
	CampaignTopicScheduled CampaignTopicStatus = "scheduled"
	CampaignTopicPublished CampaignTopicStatus = "published"
	CampaignTopicCanceled  CampaignTopicStatus = "canceled"
)

type CampaignTopic struct {
	ID             string              `json:"id"`
	WorkspaceID    string              `json:"workspaceId"`
	CampaignID     string              `json:"campaignId"`
	Title          string              `json:"title"`
	Brief          string              `json:"brief"`
	ContentType    string              `json:"contentType"`
	TargetAudience string              `json:"targetAudience"`
	FunnelStage    string              `json:"funnelStage"`
	Keywords       []string            `json:"keywords"`
	Status         CampaignTopicStatus `json:"status"`
	ContentID      string              `json:"contentId"`
	Metadata       map[string]any      `json:"metadata"`
	CreatedAt      time.Time           `json:"createdAt"`
	UpdatedAt      time.Time           `json:"updatedAt"`
}

type CampaignCalendarItemStatus string

const (
	CampaignCalendarPlanned   CampaignCalendarItemStatus = "planned"
	CampaignCalendarDrafting  CampaignCalendarItemStatus = "drafting"
	CampaignCalendarReview    CampaignCalendarItemStatus = "review"
	CampaignCalendarScheduled CampaignCalendarItemStatus = "scheduled"
	CampaignCalendarPublished CampaignCalendarItemStatus = "published"
	CampaignCalendarSkipped   CampaignCalendarItemStatus = "skipped"
	CampaignCalendarCanceled  CampaignCalendarItemStatus = "canceled"
)

type CampaignCalendarItem struct {
	ID                   string                     `json:"id"`
	WorkspaceID          string                     `json:"workspaceId"`
	CampaignID           string                     `json:"campaignId"`
	TopicID              string                     `json:"topicId"`
	ContentID            string                     `json:"contentId"`
	PublishScheduleID    string                     `json:"publishScheduleId"`
	PublishJobID         string                     `json:"publishJobId"`
	MediaAccountID       string                     `json:"mediaAccountId"`
	AssignedUserID       string                     `json:"assignedUserId"`
	Title                string                     `json:"title"`
	Brief                string                     `json:"brief"`
	ContentType          string                     `json:"contentType"`
	Channel              string                     `json:"channel"`
	PublishWindowStartAt *time.Time                 `json:"publishWindowStartAt,omitempty"`
	PublishWindowEndAt   *time.Time                 `json:"publishWindowEndAt,omitempty"`
	Status               CampaignCalendarItemStatus `json:"status"`
	DependencyItemIDs    []string                   `json:"dependencyItemIds"`
	ApprovalRequired     bool                       `json:"approvalRequired"`
	ApprovalStatus       string                     `json:"approvalStatus"`
	Metadata             map[string]any             `json:"metadata"`
	CreatedAt            time.Time                  `json:"createdAt"`
	UpdatedAt            time.Time                  `json:"updatedAt"`
}

type CampaignMetric struct {
	ID             string         `json:"id"`
	WorkspaceID    string         `json:"workspaceId"`
	CampaignID     string         `json:"campaignId"`
	CalendarItemID string         `json:"calendarItemId"`
	ContentID      string         `json:"contentId"`
	PublishJobID   string         `json:"publishJobId"`
	MediaAccountID string         `json:"mediaAccountId"`
	MetricName     string         `json:"metricName"`
	MetricValue    float64        `json:"metricValue"`
	MetricUnit     string         `json:"metricUnit"`
	Source         string         `json:"source"`
	CollectedAt    time.Time      `json:"collectedAt"`
	Metadata       map[string]any `json:"metadata"`
	CreatedAt      time.Time      `json:"createdAt"`
}

type CampaignRollup struct {
	ID              string         `json:"id"`
	WorkspaceID     string         `json:"workspaceId"`
	CampaignID      string         `json:"campaignId"`
	PeriodStart     time.Time      `json:"periodStart"`
	PeriodEnd       time.Time      `json:"periodEnd"`
	ContentCount    int            `json:"contentCount"`
	ScheduledCount  int            `json:"scheduledCount"`
	PublishedCount  int            `json:"publishedCount"`
	FailedCount     int            `json:"failedCount"`
	ImpressionCount int64          `json:"impressionCount"`
	EngagementCount int64          `json:"engagementCount"`
	ClickCount      int64          `json:"clickCount"`
	ConversionCount int64          `json:"conversionCount"`
	SpendCents      int            `json:"spendCents"`
	RevenueCents    int            `json:"revenueCents"`
	Metadata        map[string]any `json:"metadata"`
	CreatedAt       time.Time      `json:"createdAt"`
}

type CampaignRecommendation struct {
	Type     string         `json:"type"`
	Title    string         `json:"title"`
	Reason   string         `json:"reason"`
	Metadata map[string]any `json:"metadata"`
}

type CampaignReportSummary struct {
	WorkspaceID         string                   `json:"workspaceId"`
	CampaignID          string                   `json:"campaignId"`
	Status              CampaignStatus           `json:"status"`
	CalendarItemCount   int                      `json:"calendarItemCount"`
	ContentCount        int                      `json:"contentCount"`
	PublishJobCount     int                      `json:"publishJobCount"`
	PlannedItemCount    int                      `json:"plannedItemCount"`
	ScheduledItemCount  int                      `json:"scheduledItemCount"`
	PublishedItemCount  int                      `json:"publishedItemCount"`
	FailedItemCount     int                      `json:"failedItemCount"`
	StatusCounts        map[string]int           `json:"statusCounts"`
	MetricTotals        map[string]float64       `json:"metricTotals"`
	Metrics             []CampaignMetric         `json:"metrics"`
	Rollups             []CampaignRollup         `json:"rollups"`
	Recommendations     []CampaignRecommendation `json:"recommendations"`
	ReportingWindowFrom *time.Time               `json:"reportingWindowFrom,omitempty"`
	ReportingWindowTo   *time.Time               `json:"reportingWindowTo,omitempty"`
	UpdatedAt           time.Time                `json:"updatedAt"`
}

type PublishScheduleFrequency string

const (
	PublishScheduleOnce    PublishScheduleFrequency = "once"
	PublishScheduleDaily   PublishScheduleFrequency = "daily"
	PublishScheduleWeekly  PublishScheduleFrequency = "weekly"
	PublishScheduleMonthly PublishScheduleFrequency = "monthly"
)

type PublishSchedule struct {
	ID             string                   `json:"id"`
	WorkspaceID    string                   `json:"workspaceId"`
	Name           string                   `json:"name"`
	ContentID      string                   `json:"contentId"`
	MediaAccountID string                   `json:"mediaAccountId"`
	Frequency      PublishScheduleFrequency `json:"frequency"`
	NextRunAt      time.Time                `json:"nextRunAt"`
	Enabled        bool                     `json:"enabled"`
	CreatedAt      time.Time                `json:"createdAt"`
}

type PublishJobStatus string

const (
	PublishJobQueued    PublishJobStatus = "queued"
	PublishJobRunning   PublishJobStatus = "running"
	PublishJobManual    PublishJobStatus = "manual_pending"
	PublishJobSucceeded PublishJobStatus = "succeeded"
	PublishJobFailed    PublishJobStatus = "failed"
	PublishJobRetrying  PublishJobStatus = "retrying"
)

type PublishJob struct {
	ID             string           `json:"id"`
	WorkspaceID    string           `json:"workspaceId"`
	ScheduleID     string           `json:"scheduleId"`
	ContentID      string           `json:"contentId"`
	MediaAccountID string           `json:"mediaAccountId"`
	Status         PublishJobStatus `json:"status"`
	ScheduledAt    time.Time        `json:"scheduledAt"`
	ExternalURL    string           `json:"externalUrl"`
	LastMessage    string           `json:"lastMessage"`
}

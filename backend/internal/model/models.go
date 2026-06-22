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

type BrandAssetStatus string

const (
	BrandAssetActive   BrandAssetStatus = "active"
	BrandAssetArchived BrandAssetStatus = "archived"
)

type BrandAsset struct {
	ID          string            `json:"id"`
	WorkspaceID string            `json:"workspaceId"`
	Type        string            `json:"type"`
	Name        string            `json:"name"`
	Description string            `json:"description"`
	Content     string            `json:"content"`
	Channels    []string          `json:"channels"`
	Tags        []string          `json:"tags"`
	Source      string            `json:"source"`
	Status      BrandAssetStatus  `json:"status"`
	Metadata    map[string]string `json:"metadata"`
	CreatedAt   time.Time         `json:"createdAt"`
	UpdatedAt   time.Time         `json:"updatedAt"`
}

type BrandGuardrail struct {
	ID          string    `json:"id"`
	WorkspaceID string    `json:"workspaceId"`
	AssetID     string    `json:"assetId"`
	Name        string    `json:"name"`
	Category    string    `json:"category"`
	Channel     string    `json:"channel"`
	SourceType  string    `json:"sourceType"`
	SourceID    string    `json:"sourceId"`
	Severity    string    `json:"severity"`
	Rules       []string  `json:"rules"`
	Action      string    `json:"action"`
	Enabled     bool      `json:"enabled"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

type ApprovalWorkflowStatus string

const (
	ApprovalWorkflowDraft     ApprovalWorkflowStatus = "draft"
	ApprovalWorkflowActive    ApprovalWorkflowStatus = "active"
	ApprovalWorkflowCompleted ApprovalWorkflowStatus = "completed"
	ApprovalWorkflowCanceled  ApprovalWorkflowStatus = "canceled"
)

type ApprovalStage struct {
	Name              string `json:"name"`
	ApproverRole      string `json:"approverRole"`
	RequiredApprovals int    `json:"requiredApprovals"`
}

type ApprovalWorkflow struct {
	ID              string                 `json:"id"`
	WorkspaceID     string                 `json:"workspaceId"`
	ResourceType    string                 `json:"resourceType"`
	ResourceID      string                 `json:"resourceId"`
	Name            string                 `json:"name"`
	Status          ApprovalWorkflowStatus `json:"status"`
	Stages          []ApprovalStage        `json:"stages"`
	CreatedByUserID string                 `json:"createdByUserId"`
	CreatedAt       time.Time              `json:"createdAt"`
	UpdatedAt       time.Time              `json:"updatedAt"`
}

type ApprovalTaskStatus string

const (
	ApprovalTaskPending  ApprovalTaskStatus = "pending"
	ApprovalTaskApproved ApprovalTaskStatus = "approved"
	ApprovalTaskRejected ApprovalTaskStatus = "rejected"
	ApprovalTaskSkipped  ApprovalTaskStatus = "skipped"
	ApprovalTaskCanceled ApprovalTaskStatus = "canceled"
)

type ApprovalTask struct {
	ID                string             `json:"id"`
	WorkspaceID       string             `json:"workspaceId"`
	WorkflowID        string             `json:"workflowId"`
	ResourceType      string             `json:"resourceType"`
	ResourceID        string             `json:"resourceId"`
	StageName         string             `json:"stageName"`
	AssigneeUserID    string             `json:"assigneeUserId"`
	AssigneeRole      string             `json:"assigneeRole"`
	Status            ApprovalTaskStatus `json:"status"`
	Decision          string             `json:"decision"`
	Comment           string             `json:"comment"`
	ProcessedByUserID string             `json:"processedByUserId"`
	DueAt             *time.Time         `json:"dueAt,omitempty"`
	ProcessedAt       *time.Time         `json:"processedAt,omitempty"`
	CreatedAt         time.Time          `json:"createdAt"`
	UpdatedAt         time.Time          `json:"updatedAt"`
}

type ComplianceCheckStatus string

const (
	ComplianceCheckQueued    ComplianceCheckStatus = "queued"
	ComplianceCheckRunning   ComplianceCheckStatus = "running"
	ComplianceCheckCompleted ComplianceCheckStatus = "completed"
	ComplianceCheckFailed    ComplianceCheckStatus = "failed"
)

type ComplianceFinding struct {
	ID          string    `json:"id"`
	CheckID     string    `json:"checkId"`
	WorkspaceID string    `json:"workspaceId"`
	Severity    string    `json:"severity"`
	Category    string    `json:"category"`
	Evidence    string    `json:"evidence"`
	Finding     string    `json:"finding"`
	Action      string    `json:"action"`
	SourceType  string    `json:"sourceType"`
	SourceID    string    `json:"sourceId"`
	CreatedAt   time.Time `json:"createdAt"`
}

type ComplianceCheck struct {
	ID              string                `json:"id"`
	WorkspaceID     string                `json:"workspaceId"`
	ResourceType    string                `json:"resourceType"`
	ResourceID      string                `json:"resourceId"`
	Channel         string                `json:"channel"`
	Status          ComplianceCheckStatus `json:"status"`
	RiskLevel       string                `json:"riskLevel"`
	Summary         string                `json:"summary"`
	Findings        []ComplianceFinding   `json:"findings"`
	CreatedByUserID string                `json:"createdByUserId"`
	CreatedAt       time.Time             `json:"createdAt"`
	UpdatedAt       time.Time             `json:"updatedAt"`
}

type AgencyClientRelation struct {
	ID                string    `json:"id"`
	AgencyWorkspaceID string    `json:"agencyWorkspaceId"`
	ClientWorkspaceID string    `json:"clientWorkspaceId"`
	ClientName        string    `json:"clientName"`
	Status            string    `json:"status"`
	Scopes            []string  `json:"scopes"`
	Notes             string    `json:"notes"`
	CreatedAt         time.Time `json:"createdAt"`
	UpdatedAt         time.Time `json:"updatedAt"`
}

type ReportPackage struct {
	ID                string         `json:"id"`
	WorkspaceID       string         `json:"workspaceId"`
	Name              string         `json:"name"`
	ReportType        string         `json:"reportType"`
	Audience          string         `json:"audience"`
	PeriodStart       time.Time      `json:"periodStart"`
	PeriodEnd         time.Time      `json:"periodEnd"`
	Status            string         `json:"status"`
	Sections          []string       `json:"sections"`
	Metrics           map[string]any `json:"metrics"`
	Summary           string         `json:"summary"`
	GeneratedByUserID string         `json:"generatedByUserId"`
	CreatedAt         time.Time      `json:"createdAt"`
	UpdatedAt         time.Time      `json:"updatedAt"`
}

type StrategyRecommendation struct {
	ID                 string    `json:"id"`
	WorkspaceID        string    `json:"workspaceId"`
	SourceType         string    `json:"sourceType"`
	RecommendationType string    `json:"recommendationType"`
	Title              string    `json:"title"`
	Rationale          string    `json:"rationale"`
	Evidence           []string  `json:"evidence"`
	Action             string    `json:"action"`
	Confidence         float64   `json:"confidence"`
	Status             string    `json:"status"`
	CreatedAt          time.Time `json:"createdAt"`
	UpdatedAt          time.Time `json:"updatedAt"`
}

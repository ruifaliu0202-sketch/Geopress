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
	SkillPackageVersionID string    `json:"skillPackageVersionId"`
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

type SkillPackageStatus string

const (
	SkillPackageStatusDraft      SkillPackageStatus = "draft"
	SkillPackageStatusInReview   SkillPackageStatus = "in_review"
	SkillPackageStatusApproved   SkillPackageStatus = "approved"
	SkillPackageStatusPublished  SkillPackageStatus = "published"
	SkillPackageStatusRejected   SkillPackageStatus = "rejected"
	SkillPackageStatusDeprecated SkillPackageStatus = "deprecated"
)

type SkillPackageVersionStatus string

const (
	SkillPackageVersionDraft      SkillPackageVersionStatus = "draft"
	SkillPackageVersionSubmitted  SkillPackageVersionStatus = "submitted"
	SkillPackageVersionApproved   SkillPackageVersionStatus = "approved"
	SkillPackageVersionRejected   SkillPackageVersionStatus = "rejected"
	SkillPackageVersionPublished  SkillPackageVersionStatus = "published"
	SkillPackageVersionDeprecated SkillPackageVersionStatus = "deprecated"
)

type SkillPackageAssetType string

const (
	SkillPackageAssetPrompt  SkillPackageAssetType = "prompt"
	SkillPackageAssetSchema  SkillPackageAssetType = "schema"
	SkillPackageAssetRule    SkillPackageAssetType = "rule"
	SkillPackageAssetExample SkillPackageAssetType = "example"
	SkillPackageAssetQA      SkillPackageAssetType = "qa"
	SkillPackageAssetPublish SkillPackageAssetType = "publish"
)

type SkillPackageReviewDecision string

const (
	SkillPackageReviewSubmitted SkillPackageReviewDecision = "submitted"
	SkillPackageReviewApproved  SkillPackageReviewDecision = "approved"
	SkillPackageReviewRejected  SkillPackageReviewDecision = "rejected"
)

type WorkspaceSkillEntitlementStatus string

const (
	WorkspaceSkillEntitlementActive      WorkspaceSkillEntitlementStatus = "active"
	WorkspaceSkillEntitlementExpired     WorkspaceSkillEntitlementStatus = "expired"
	WorkspaceSkillEntitlementCanceled    WorkspaceSkillEntitlementStatus = "canceled"
	WorkspaceSkillEntitlementUninstalled WorkspaceSkillEntitlementStatus = "uninstalled"
)

type WorkspaceSkillEntitlementSource string

const (
	WorkspaceSkillEntitlementTrial        WorkspaceSkillEntitlementSource = "trial"
	WorkspaceSkillEntitlementPurchase     WorkspaceSkillEntitlementSource = "purchase"
	WorkspaceSkillEntitlementSubscription WorkspaceSkillEntitlementSource = "subscription"
	WorkspaceSkillEntitlementAdminGrant   WorkspaceSkillEntitlementSource = "admin_grant"
)

type SkillPackageUsageMetricType string

const (
	SkillPackageUsageGeneration  SkillPackageUsageMetricType = "generation"
	SkillPackageUsageFormatting  SkillPackageUsageMetricType = "formatting"
	SkillPackageUsageQA          SkillPackageUsageMetricType = "qa"
	SkillPackageUsagePublishPrep SkillPackageUsageMetricType = "publish_prep"
)

type SkillPackageRevenueMetricType string

const (
	SkillPackageRevenuePurchase     SkillPackageRevenueMetricType = "purchase"
	SkillPackageRevenueSubscription SkillPackageRevenueMetricType = "subscription"
	SkillPackageRevenueRefund       SkillPackageRevenueMetricType = "refund"
	SkillPackageRevenuePayout       SkillPackageRevenueMetricType = "payout"
)

type SkillPackage struct {
	ID                      string             `json:"id"`
	Name                    string             `json:"name"`
	Slug                    string             `json:"slug"`
	Description             string             `json:"description"`
	Category                string             `json:"category"`
	TargetPlatform          string             `json:"targetPlatform"`
	TargetIndustry          string             `json:"targetIndustry"`
	SupportedContentFormats []string           `json:"supportedContentFormats"`
	AuthorID                string             `json:"authorId"`
	AuthorName              string             `json:"authorName"`
	ListingStatus           SkillPackageStatus `json:"listingStatus"`
	PriceCents              int                `json:"priceCents"`
	Currency                string             `json:"currency"`
	RevenueShareBps         int                `json:"revenueShareBps"`
	LatestVersionID         string             `json:"latestVersionId"`
	PublishedVersionID      string             `json:"publishedVersionId"`
	CreatedAt               time.Time          `json:"createdAt"`
	UpdatedAt               time.Time          `json:"updatedAt"`
}

type SkillPackageVersion struct {
	ID               string                    `json:"id"`
	PackageID        string                    `json:"packageId"`
	Version          string                    `json:"version"`
	Status           SkillPackageVersionStatus `json:"status"`
	PromptContract   string                    `json:"promptContract"`
	OutputSchema     string                    `json:"outputSchema"`
	QualityRules     string                    `json:"qualityRules"`
	QARules          string                    `json:"qaRules"`
	PublishPrepRules string                    `json:"publishPrepRules"`
	ChangeNote       string                    `json:"changeNote"`
	SubmittedAt      *time.Time                `json:"submittedAt,omitempty"`
	ReviewedAt       *time.Time                `json:"reviewedAt,omitempty"`
	PublishedAt      *time.Time                `json:"publishedAt,omitempty"`
	CreatedAt        time.Time                 `json:"createdAt"`
	UpdatedAt        time.Time                 `json:"updatedAt"`
}

type SkillPackageAsset struct {
	ID        string                `json:"id"`
	PackageID string                `json:"packageId"`
	VersionID string                `json:"versionId"`
	Type      SkillPackageAssetType `json:"type"`
	Title     string                `json:"title"`
	Content   string                `json:"content"`
	Metadata  map[string]string     `json:"metadata"`
	CreatedAt time.Time             `json:"createdAt"`
}

type SkillPackageExample struct {
	ID             string            `json:"id"`
	PackageID      string            `json:"packageId"`
	VersionID      string            `json:"versionId"`
	Title          string            `json:"title"`
	Input          string            `json:"input"`
	ExpectedOutput string            `json:"expectedOutput"`
	Notes          string            `json:"notes"`
	Metadata       map[string]string `json:"metadata"`
	CreatedAt      time.Time         `json:"createdAt"`
}

type SkillPackageReview struct {
	ID         string                     `json:"id"`
	PackageID  string                     `json:"packageId"`
	VersionID  string                     `json:"versionId"`
	ReviewerID string                     `json:"reviewerId"`
	Decision   SkillPackageReviewDecision `json:"decision"`
	Comment    string                     `json:"comment"`
	CreatedAt  time.Time                  `json:"createdAt"`
}

type WorkspaceSkillEntitlement struct {
	ID                     string                          `json:"id"`
	WorkspaceID            string                          `json:"workspaceId"`
	PackageID              string                          `json:"packageId"`
	VersionID              string                          `json:"versionId"`
	Status                 WorkspaceSkillEntitlementStatus `json:"status"`
	Source                 WorkspaceSkillEntitlementSource `json:"source"`
	Seats                  int                             `json:"seats"`
	PriceCents             int                             `json:"priceCents"`
	Currency               string                          `json:"currency"`
	CurrentPeriod          string                          `json:"currentPeriod"`
	CurrentPeriodStartedAt *time.Time                      `json:"currentPeriodStartedAt,omitempty"`
	CurrentPeriodEndsAt    *time.Time                      `json:"currentPeriodEndsAt,omitempty"`
	InstalledAt            time.Time                       `json:"installedAt"`
	ExpiresAt              *time.Time                      `json:"expiresAt,omitempty"`
	CreatedAt              time.Time                       `json:"createdAt"`
	UpdatedAt              time.Time                       `json:"updatedAt"`
}

type SkillPackageUsageMetric struct {
	ID                  string                      `json:"id"`
	WorkspaceID         string                      `json:"workspaceId"`
	PackageID           string                      `json:"packageId"`
	VersionID           string                      `json:"versionId"`
	GenerationRequestID string                      `json:"generationRequestId"`
	ContentID           string                      `json:"contentId"`
	MetricType          SkillPackageUsageMetricType `json:"metricType"`
	Count               int                         `json:"count"`
	Status              string                      `json:"status"`
	CreatedAt           time.Time                   `json:"createdAt"`
}

type SkillPackageRevenueMetric struct {
	ID                 string                        `json:"id"`
	PackageID          string                        `json:"packageId"`
	VersionID          string                        `json:"versionId"`
	WorkspaceID        string                        `json:"workspaceId"`
	EntitlementID      string                        `json:"entitlementId"`
	MetricType         SkillPackageRevenueMetricType `json:"metricType"`
	AmountCents        int                           `json:"amountCents"`
	Currency           string                        `json:"currency"`
	AuthorRevenueCents int                           `json:"authorRevenueCents"`
	PlatformFeeCents   int                           `json:"platformFeeCents"`
	BillingPeriod      string                        `json:"billingPeriod"`
	CreatedAt          time.Time                     `json:"createdAt"`
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

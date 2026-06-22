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

type CreatorVerificationState string

const (
	CreatorVerificationUnverified CreatorVerificationState = "unverified"
	CreatorVerificationPending    CreatorVerificationState = "pending"
	CreatorVerificationVerified   CreatorVerificationState = "verified"
	CreatorVerificationRejected   CreatorVerificationState = "rejected"
)

type CreatorAvailabilityStatus string

const (
	CreatorAvailabilityAvailable   CreatorAvailabilityStatus = "available"
	CreatorAvailabilityLimited     CreatorAvailabilityStatus = "limited"
	CreatorAvailabilityUnavailable CreatorAvailabilityStatus = "unavailable"
)

type Creator struct {
	ID                  string                    `json:"id"`
	DisplayName         string                    `json:"displayName"`
	LegalName           string                    `json:"legalName,omitempty"`
	Bio                 string                    `json:"bio"`
	AvatarURL           string                    `json:"avatarUrl"`
	ContactEmail        string                    `json:"contactEmail,omitempty"`
	Verticals           []string                  `json:"verticals"`
	AudienceAttributes  map[string]string         `json:"audienceAttributes"`
	BasePriceCents      int                       `json:"basePriceCents"`
	Currency            string                    `json:"currency"`
	AvailabilityStatus  CreatorAvailabilityStatus `json:"availabilityStatus"`
	CollaborationPolicy string                    `json:"collaborationPolicy"`
	VerificationState   CreatorVerificationState  `json:"verificationState"`
	BrandSafetyLevel    string                    `json:"brandSafetyLevel"`
	CreatedAt           time.Time                 `json:"createdAt"`
	UpdatedAt           time.Time                 `json:"updatedAt"`
}

type CreatorMediaAccount struct {
	ID                    string            `json:"id"`
	CreatorID             string            `json:"creatorId"`
	PlatformID            string            `json:"platformId"`
	PlatformName          string            `json:"platformName"`
	Handle                string            `json:"handle"`
	ProfileURL            string            `json:"profileUrl"`
	FollowerCount         int               `json:"followerCount"`
	AverageEngagementRate float64           `json:"averageEngagementRate"`
	Verticals             []string          `json:"verticals"`
	AudienceAttributes    map[string]string `json:"audienceAttributes"`
	AccountAccessMode     string            `json:"accountAccessMode"`
	Verified              bool              `json:"verified"`
	CreatedAt             time.Time         `json:"createdAt"`
	UpdatedAt             time.Time         `json:"updatedAt"`
}

type CreatorShortlist struct {
	ID                  string    `json:"id"`
	WorkspaceID         string    `json:"workspaceId"`
	CreatorID           string    `json:"creatorId"`
	Name                string    `json:"name"`
	FitScore            int       `json:"fitScore"`
	QualificationStatus string    `json:"qualificationStatus"`
	BrandSafetyLevel    string    `json:"brandSafetyLevel"`
	BrandSafetyNotes    string    `json:"brandSafetyNotes"`
	OperatorNotes       string    `json:"operatorNotes"`
	CreatedByUserID     string    `json:"createdByUserId"`
	CreatedAt           time.Time `json:"createdAt"`
	UpdatedAt           time.Time `json:"updatedAt"`
}

type CreatorCampaignBriefStatus string

const (
	CreatorCampaignBriefDraft    CreatorCampaignBriefStatus = "draft"
	CreatorCampaignBriefActive   CreatorCampaignBriefStatus = "active"
	CreatorCampaignBriefArchived CreatorCampaignBriefStatus = "archived"
)

type CreatorCampaignBrief struct {
	ID                      string                     `json:"id"`
	WorkspaceID             string                     `json:"workspaceId"`
	Title                   string                     `json:"title"`
	Objective               string                     `json:"objective"`
	ProductName             string                     `json:"productName"`
	TargetAudience          string                     `json:"targetAudience"`
	PlatformTargets         []string                   `json:"platformTargets"`
	DeliverableRequirements []string                   `json:"deliverableRequirements"`
	DisclosureRequirements  []string                   `json:"disclosureRequirements"`
	ProhibitedClaims        []string                   `json:"prohibitedClaims"`
	AuthorizationScope      string                     `json:"authorizationScope"`
	ContentUsageRights      string                     `json:"contentUsageRights"`
	ReviewWindowHours       int                        `json:"reviewWindowHours"`
	DeadlineAt              *time.Time                 `json:"deadlineAt,omitempty"`
	BudgetCents             int                        `json:"budgetCents"`
	Currency                string                     `json:"currency"`
	Status                  CreatorCampaignBriefStatus `json:"status"`
	CreatedByUserID         string                     `json:"createdByUserId"`
	CreatedAt               time.Time                  `json:"createdAt"`
	UpdatedAt               time.Time                  `json:"updatedAt"`
}

type CreatorOrderStatus string

const (
	CreatorOrderProposed   CreatorOrderStatus = "proposed"
	CreatorOrderAccepted   CreatorOrderStatus = "accepted"
	CreatorOrderInProgress CreatorOrderStatus = "in_progress"
	CreatorOrderSubmitted  CreatorOrderStatus = "submitted"
	CreatorOrderApproved   CreatorOrderStatus = "approved"
	CreatorOrderPublished  CreatorOrderStatus = "published"
	CreatorOrderCompleted  CreatorOrderStatus = "completed"
	CreatorOrderCanceled   CreatorOrderStatus = "canceled"
	CreatorOrderDisputed   CreatorOrderStatus = "disputed"
)

type CreatorOrder struct {
	ID                      string             `json:"id"`
	WorkspaceID             string             `json:"workspaceId"`
	BriefID                 string             `json:"briefId"`
	CreatorID               string             `json:"creatorId"`
	Status                  CreatorOrderStatus `json:"status"`
	PriceCents              int                `json:"priceCents"`
	DepositCents            int                `json:"depositCents"`
	ServiceFeeCents         int                `json:"serviceFeeCents"`
	Currency                string             `json:"currency"`
	DisclosureRequirements  []string           `json:"disclosureRequirements"`
	DeliverableRequirements []string           `json:"deliverableRequirements"`
	AuthorizationScope      string             `json:"authorizationScope"`
	ContentUsageRights      string             `json:"contentUsageRights"`
	DueAt                   *time.Time         `json:"dueAt,omitempty"`
	AcceptedAt              *time.Time         `json:"acceptedAt,omitempty"`
	CompletedAt             *time.Time         `json:"completedAt,omitempty"`
	LastMessage             string             `json:"lastMessage"`
	CreatedByUserID         string             `json:"createdByUserId"`
	CreatedAt               time.Time          `json:"createdAt"`
	UpdatedAt               time.Time          `json:"updatedAt"`
}

type CreatorDeliverableStatus string

const (
	CreatorDeliverableSubmitted         CreatorDeliverableStatus = "submitted"
	CreatorDeliverableRevisionRequested CreatorDeliverableStatus = "revision_requested"
	CreatorDeliverableApproved          CreatorDeliverableStatus = "approved"
	CreatorDeliverableRejected          CreatorDeliverableStatus = "rejected"
	CreatorDeliverablePublished         CreatorDeliverableStatus = "published"
)

type CreatorDeliverable struct {
	ID                   string                   `json:"id"`
	WorkspaceID          string                   `json:"workspaceId"`
	OrderID              string                   `json:"orderId"`
	CreatorID            string                   `json:"creatorId"`
	Type                 string                   `json:"type"`
	Title                string                   `json:"title"`
	Content              string                   `json:"content"`
	AssetURLs            []string                 `json:"assetUrls"`
	Status               CreatorDeliverableStatus `json:"status"`
	ExternalURL          string                   `json:"externalUrl"`
	PublicationProofURL  string                   `json:"publicationProofUrl"`
	PublicationProofNote string                   `json:"publicationProofNote"`
	ReviewFeedback       string                   `json:"reviewFeedback"`
	Revision             int                      `json:"revision"`
	SubmittedAt          time.Time                `json:"submittedAt"`
	ReviewedAt           *time.Time               `json:"reviewedAt,omitempty"`
	PublishedAt          *time.Time               `json:"publishedAt,omitempty"`
	CreatedAt            time.Time                `json:"createdAt"`
	UpdatedAt            time.Time                `json:"updatedAt"`
}

type CreatorSettlementStatus string

const (
	CreatorSettlementPending  CreatorSettlementStatus = "pending"
	CreatorSettlementInvoiced CreatorSettlementStatus = "invoiced"
	CreatorSettlementPayable  CreatorSettlementStatus = "payable"
	CreatorSettlementPaid     CreatorSettlementStatus = "paid"
	CreatorSettlementRefunded CreatorSettlementStatus = "refunded"
	CreatorSettlementDisputed CreatorSettlementStatus = "disputed"
	CreatorSettlementCanceled CreatorSettlementStatus = "canceled"
)

type CreatorSettlement struct {
	ID                 string                  `json:"id"`
	WorkspaceID        string                  `json:"workspaceId"`
	OrderID            string                  `json:"orderId"`
	CreatorID          string                  `json:"creatorId"`
	Status             CreatorSettlementStatus `json:"status"`
	PriceCents         int                     `json:"priceCents"`
	DepositCents       int                     `json:"depositCents"`
	ServiceFeeCents    int                     `json:"serviceFeeCents"`
	CreatorPayoutCents int                     `json:"creatorPayoutCents"`
	Currency           string                  `json:"currency"`
	InvoiceID          string                  `json:"invoiceId"`
	DueAt              *time.Time              `json:"dueAt,omitempty"`
	PaidAt             *time.Time              `json:"paidAt,omitempty"`
	CreatedAt          time.Time               `json:"createdAt"`
	UpdatedAt          time.Time               `json:"updatedAt"`
}

type CreatorComplianceEvidenceType string

const (
	CreatorEvidenceAdDisclosure     CreatorComplianceEvidenceType = "ad_disclosure"
	CreatorEvidenceAuthorization    CreatorComplianceEvidenceType = "authorization_record"
	CreatorEvidenceUsageRights      CreatorComplianceEvidenceType = "usage_rights"
	CreatorEvidenceReviewLog        CreatorComplianceEvidenceType = "review_log"
	CreatorEvidencePublicationProof CreatorComplianceEvidenceType = "publication_proof"
	CreatorEvidenceDisputeRecord    CreatorComplianceEvidenceType = "dispute_record"
)

type CreatorComplianceEvidence struct {
	ID                 string                        `json:"id"`
	WorkspaceID        string                        `json:"workspaceId"`
	OrderID            string                        `json:"orderId"`
	DeliverableID      string                        `json:"deliverableId"`
	CreatorID          string                        `json:"creatorId"`
	EvidenceType       CreatorComplianceEvidenceType `json:"evidenceType"`
	DisclosureText     string                        `json:"disclosureText"`
	AuthorizationScope string                        `json:"authorizationScope"`
	ContentUsageRights string                        `json:"contentUsageRights"`
	ExternalURL        string                        `json:"externalUrl"`
	FileURL            string                        `json:"fileUrl"`
	Notes              string                        `json:"notes"`
	CreatedByUserID    string                        `json:"createdByUserId"`
	CreatedAt          time.Time                     `json:"createdAt"`
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

package model

import (
	"time"

	"geopress/backend/internal/domain"
)

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
	ID                 string                           `json:"id"`
	Name               string                           `json:"name"`
	Type               string                           `json:"type"`
	Enabled            bool                             `json:"enabled"`
	SupportsArticle    bool                             `json:"supportsArticle"`
	SupportsImage      bool                             `json:"supportsImage"`
	SupportsScheduling bool                             `json:"supportsScheduling"`
	CredentialFields   []string                         `json:"credentialFields"`
	Capabilities       domain.MediaPlatformCapabilities `json:"capabilities"`
}

func (platform *MediaPlatform) EnsureCapabilities() {
	if platform == nil {
		return
	}
	if platform.CredentialFields == nil {
		platform.CredentialFields = []string{}
	}
	if platform.Capabilities.IsZero() {
		if platform.ID == "plt_xiaohongshu" || platform.Type == "xiaohongshu" {
			// 小红书已有浏览器二维码登录和浏览器发布路径，旧管理端请求缺省时也要保留这个能力契约。
			platform.Capabilities = domain.DefaultXiaohongshuCapabilities()
			return
		}
		// 外部平台能力是后续矩阵、采集、发布共用的边界；旧布尔字段只作为兼容输入。
		platform.Capabilities = domain.LegacyCapabilities(platform.SupportsArticle, platform.SupportsImage, platform.SupportsScheduling, platform.CredentialFields)
		return
	}
	platform.Capabilities = platform.Capabilities.WithDefaults()
}

type MediaAccount struct {
	ID                  string            `json:"id"`
	WorkspaceID         string            `json:"workspaceId"`
	PlatformID          string            `json:"platformId"`
	Name                string            `json:"name"`
	ExternalID          string            `json:"externalId"`
	LoginMethod         string            `json:"loginMethod"`
	CredentialMeta      map[string]string `json:"credentialMeta,omitempty"`
	Status              string            `json:"status"`
	AccountGroup        string            `json:"accountGroup"`
	OwnershipType       string            `json:"ownershipType"`
	OperatingRole       string            `json:"operatingRole"`
	Persona             string            `json:"persona"`
	Positioning         string            `json:"positioning"`
	TargetAudience      string            `json:"targetAudience"`
	ContentCategories   []string          `json:"contentCategories"`
	HealthStatus        string            `json:"healthStatus"`
	HealthNotes         string            `json:"healthNotes"`
	AuthorizationScopes []string          `json:"authorizationScopes"`
	SyncEnabled         bool              `json:"syncEnabled"`
	LastSyncJobID       string            `json:"lastSyncJobId"`
	LastSyncStatus      string            `json:"lastSyncStatus"`
	LastSyncMessage     string            `json:"lastSyncMessage"`
	LastProfileSyncedAt *time.Time        `json:"lastProfileSyncedAt,omitempty"`
	LastMetricsSyncedAt *time.Time        `json:"lastMetricsSyncedAt,omitempty"`
	NextSyncAt          *time.Time        `json:"nextSyncAt,omitempty"`
	MatrixMetadata      map[string]any    `json:"matrixMetadata"`
	ExpiresAt           *time.Time        `json:"expiresAt,omitempty"`
	LastCheckedAt       time.Time         `json:"lastCheckedAt"`
}

type MediaAccountMetricSnapshot struct {
	ID                 string         `json:"id"`
	WorkspaceID        string         `json:"workspaceId"`
	MediaAccountID     string         `json:"mediaAccountId"`
	PlatformID         string         `json:"platformId"`
	Source             string         `json:"source"`
	CapturedAt         time.Time      `json:"capturedAt"`
	FollowerCount      int            `json:"followerCount"`
	FollowingCount     int            `json:"followingCount"`
	ContentCount       int            `json:"contentCount"`
	TotalLikeCount     int            `json:"totalLikeCount"`
	TotalFavoriteCount int            `json:"totalFavoriteCount"`
	TotalCommentCount  int            `json:"totalCommentCount"`
	TotalShareCount    int            `json:"totalShareCount"`
	EngagementRate     float64        `json:"engagementRate"`
	AudienceSignals    map[string]any `json:"audienceSignals"`
	Profile            map[string]any `json:"profile"`
	RawMetrics         map[string]any `json:"rawMetrics"`
	FreshnessStatus    string         `json:"freshnessStatus"`
	CreatedAt          time.Time      `json:"createdAt"`
}

type ContentMetric struct {
	ID                  string         `json:"id"`
	WorkspaceID         string         `json:"workspaceId"`
	ContentID           string         `json:"contentId"`
	PublishJobID        string         `json:"publishJobId"`
	MediaAccountID      string         `json:"mediaAccountId"`
	PlatformID          string         `json:"platformId"`
	ExternalContentID   string         `json:"externalContentId"`
	ExternalURL         string         `json:"externalUrl"`
	MetricDate          string         `json:"metricDate"`
	CapturedAt          time.Time      `json:"capturedAt"`
	ImpressionCount     int            `json:"impressionCount"`
	ViewCount           int            `json:"viewCount"`
	LikeCount           int            `json:"likeCount"`
	CommentCount        int            `json:"commentCount"`
	ShareCount          int            `json:"shareCount"`
	FavoriteCount       int            `json:"favoriteCount"`
	ClickCount          int            `json:"clickCount"`
	EngagementRate      float64        `json:"engagementRate"`
	AttributionMetadata map[string]any `json:"attributionMetadata"`
	RawMetrics          map[string]any `json:"rawMetrics"`
	CreatedAt           time.Time      `json:"createdAt"`
}

type MediaAccountSyncJob struct {
	ID                string         `json:"id"`
	WorkspaceID       string         `json:"workspaceId"`
	MediaAccountID    string         `json:"mediaAccountId"`
	PlatformID        string         `json:"platformId"`
	RequestedByUserID string         `json:"requestedByUserId"`
	SyncType          string         `json:"syncType"`
	Status            string         `json:"status"`
	RequestedAt       time.Time      `json:"requestedAt"`
	StartedAt         *time.Time     `json:"startedAt,omitempty"`
	FinishedAt        *time.Time     `json:"finishedAt,omitempty"`
	IdempotencyKey    string         `json:"idempotencyKey"`
	RequestPayload    map[string]any `json:"requestPayload"`
	ResultSummary     map[string]any `json:"resultSummary"`
	ErrorMessage      string         `json:"errorMessage"`
	CreatedAt         time.Time      `json:"createdAt"`
	UpdatedAt         time.Time      `json:"updatedAt"`
}

type MediaAccountMatrixItem struct {
	Account            MediaAccount                `json:"account"`
	Platform           MediaPlatform               `json:"platform"`
	LatestSnapshot     *MediaAccountMetricSnapshot `json:"latestSnapshot,omitempty"`
	LatestSyncJob      *MediaAccountSyncJob        `json:"latestSyncJob,omitempty"`
	ContentMetricCount int                         `json:"contentMetricCount"`
	DataFreshness      string                      `json:"dataFreshness"`
	Warnings           []string                    `json:"warnings"`
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
	ID                       string         `json:"id"`
	WorkspaceID              string         `json:"workspaceId"`
	KnowledgeBaseID          string         `json:"knowledgeBaseId"`
	AttributedMediaAccountID string         `json:"attributedMediaAccountId"`
	Title                    string         `json:"title"`
	Summary                  string         `json:"summary"`
	Body                     string         `json:"body"`
	Keywords                 []string       `json:"keywords"`
	Status                   ContentStatus  `json:"status"`
	Author                   string         `json:"author"`
	Source                   string         `json:"source"`
	AttributionMetadata      map[string]any `json:"attributionMetadata"`
	UpdatedAt                time.Time      `json:"updatedAt"`
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
	ID                  string           `json:"id"`
	WorkspaceID         string           `json:"workspaceId"`
	ScheduleID          string           `json:"scheduleId"`
	ContentID           string           `json:"contentId"`
	MediaAccountID      string           `json:"mediaAccountId"`
	Status              PublishJobStatus `json:"status"`
	ScheduledAt         time.Time        `json:"scheduledAt"`
	ExternalURL         string           `json:"externalUrl"`
	LastMessage         string           `json:"lastMessage"`
	AttributionMetadata map[string]any   `json:"attributionMetadata"`
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

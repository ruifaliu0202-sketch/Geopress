package domain

import "time"

type ProductLine string

const (
	ProductLineWorkspaceCore         ProductLine = "workspace_core"
	ProductLineMediaMatrix           ProductLine = "media_matrix"
	ProductLineCreatorCollaboration  ProductLine = "creator_collaboration"
	ProductLineSkillPackage          ProductLine = "skill_package"
	ProductLineCampaign              ProductLine = "campaign"
	ProductLineCommercialEntitlement ProductLine = "commercial_entitlement"
)

type ResourceType string

const (
	ResourceWorkspace           ResourceType = "workspace"
	ResourceMediaPlatform       ResourceType = "media_platform"
	ResourceMediaAccount        ResourceType = "media_account"
	ResourceCreator             ResourceType = "creator"
	ResourceCreatorOrder        ResourceType = "creator_order"
	ResourceSkillPackage        ResourceType = "skill_package"
	ResourceSkillPackageVersion ResourceType = "skill_package_version"
	ResourceCampaign            ResourceType = "campaign"
	ResourceContent             ResourceType = "content"
	ResourcePublishJob          ResourceType = "publish_job"
	ResourceEntitlement         ResourceType = "entitlement"
	ResourceReview              ResourceType = "review"
	ResourceAuditLog            ResourceType = "audit_log"
)

type OwnershipScope string

const (
	OwnershipScopePlatform  OwnershipScope = "platform"
	OwnershipScopeWorkspace OwnershipScope = "workspace"
	OwnershipScopeUser      OwnershipScope = "user"
	OwnershipScopeCreator   OwnershipScope = "creator"
)

type ResourceRef struct {
	Type           ResourceType   `json:"type"`
	ID             string         `json:"id"`
	WorkspaceID    string         `json:"workspaceId,omitempty"`
	OwnershipScope OwnershipScope `json:"ownershipScope"`
}

type WorkspacePermission string

const (
	WorkspacePermissionView      WorkspacePermission = "workspace.view"
	WorkspacePermissionManage    WorkspacePermission = "workspace.manage"
	WorkspacePermissionEdit      WorkspacePermission = "workspace.edit"
	WorkspacePermissionPublish   WorkspacePermission = "workspace.publish"
	WorkspacePermissionBilling   WorkspacePermission = "workspace.billing"
	WorkspacePermissionAdminOnly WorkspacePermission = "platform.admin"
)

type WorkspaceRole string

const (
	WorkspaceRoleOwner  WorkspaceRole = "owner"
	WorkspaceRoleAdmin  WorkspaceRole = "admin"
	WorkspaceRoleEditor WorkspaceRole = "editor"
	WorkspaceRoleViewer WorkspaceRole = "viewer"
)

func (role WorkspaceRole) CanManageWorkspace() bool {
	return role == WorkspaceRoleOwner || role == WorkspaceRoleAdmin
}

func (role WorkspaceRole) CanEditWorkspaceContent() bool {
	return role == WorkspaceRoleOwner || role == WorkspaceRoleAdmin || role == WorkspaceRoleEditor
}

type EntitlementSubjectType string

const (
	EntitlementSubjectWorkspace EntitlementSubjectType = "workspace"
	EntitlementSubjectUser      EntitlementSubjectType = "user"
)

type EntitlementResourceType string

const (
	EntitlementResourceSkillPackage          EntitlementResourceType = "skill_package"
	EntitlementResourcePlatformKnowledgeBase EntitlementResourceType = "platform_knowledge_base"
	EntitlementResourceConnectorCapability   EntitlementResourceType = "connector_capability"
	EntitlementResourceCampaignFeature       EntitlementResourceType = "campaign_feature"
	EntitlementResourceCreatorMarketplace    EntitlementResourceType = "creator_marketplace"
)

type Entitlement struct {
	ID           string                  `json:"id"`
	ProductLine  ProductLine             `json:"productLine"`
	SubjectType  EntitlementSubjectType  `json:"subjectType"`
	SubjectID    string                  `json:"subjectId"`
	WorkspaceID  string                  `json:"workspaceId,omitempty"`
	ResourceType EntitlementResourceType `json:"resourceType"`
	ResourceID   string                  `json:"resourceId"`
	Status       EntitlementStatus       `json:"status"`
	StartsAt     *time.Time              `json:"startsAt,omitempty"`
	ExpiresAt    *time.Time              `json:"expiresAt,omitempty"`
	Source       string                  `json:"source,omitempty"`
	Metadata     map[string]any          `json:"metadata"`
	CreatedAt    time.Time               `json:"createdAt"`
	UpdatedAt    time.Time               `json:"updatedAt"`
}

type EntitlementStatus string

const (
	EntitlementStatusPending  EntitlementStatus = "pending"
	EntitlementStatusActive   EntitlementStatus = "active"
	EntitlementStatusExpired  EntitlementStatus = "expired"
	EntitlementStatusRevoked  EntitlementStatus = "revoked"
	EntitlementStatusCanceled EntitlementStatus = "canceled"
)

func (status EntitlementStatus) IsUsable() bool {
	return status == EntitlementStatusActive
}

type ReviewStatus string

const (
	ReviewStatusDraft            ReviewStatus = "draft"
	ReviewStatusPending          ReviewStatus = "pending"
	ReviewStatusApproved         ReviewStatus = "approved"
	ReviewStatusRejected         ReviewStatus = "rejected"
	ReviewStatusChangesRequested ReviewStatus = "changes_requested"
	ReviewStatusCanceled         ReviewStatus = "canceled"
)

func (status ReviewStatus) IsTerminal() bool {
	return status == ReviewStatusApproved || status == ReviewStatusRejected || status == ReviewStatusCanceled
}

type ReviewTargetType string

const (
	ReviewTargetContent              ReviewTargetType = "content"
	ReviewTargetCampaign             ReviewTargetType = "campaign"
	ReviewTargetCreatorDeliverable   ReviewTargetType = "creator_deliverable"
	ReviewTargetSkillPackageVersion  ReviewTargetType = "skill_package_version"
	ReviewTargetPublishJob           ReviewTargetType = "publish_job"
	ReviewTargetCommercialCompliance ReviewTargetType = "commercial_compliance"
)

type ReviewRecord struct {
	ID              string           `json:"id"`
	ProductLine     ProductLine      `json:"productLine"`
	WorkspaceID     string           `json:"workspaceId,omitempty"`
	TargetType      ReviewTargetType `json:"targetType"`
	TargetID        string           `json:"targetId"`
	Status          ReviewStatus     `json:"status"`
	RequestedByID   string           `json:"requestedById,omitempty"`
	ReviewedByID    string           `json:"reviewedById,omitempty"`
	DecisionMessage string           `json:"decisionMessage,omitempty"`
	Evidence        map[string]any   `json:"evidence"`
	Metadata        map[string]any   `json:"metadata"`
	CreatedAt       time.Time        `json:"createdAt"`
	UpdatedAt       time.Time        `json:"updatedAt"`
}

type CampaignStatus string

const (
	CampaignStatusDraft     CampaignStatus = "draft"
	CampaignStatusPlanned   CampaignStatus = "planned"
	CampaignStatusActive    CampaignStatus = "active"
	CampaignStatusPaused    CampaignStatus = "paused"
	CampaignStatusCompleted CampaignStatus = "completed"
	CampaignStatusCanceled  CampaignStatus = "canceled"
)

type CampaignRef struct {
	ID          string         `json:"id"`
	WorkspaceID string         `json:"workspaceId"`
	Name        string         `json:"name"`
	Status      CampaignStatus `json:"status"`
}

type SkillPackageStatus string

const (
	SkillPackageStatusDraft      SkillPackageStatus = "draft"
	SkillPackageStatusSubmitted  SkillPackageStatus = "submitted"
	SkillPackageStatusApproved   SkillPackageStatus = "approved"
	SkillPackageStatusPublished  SkillPackageStatus = "published"
	SkillPackageStatusRejected   SkillPackageStatus = "rejected"
	SkillPackageStatusDeprecated SkillPackageStatus = "deprecated"
)

type SkillPackageRef struct {
	ID        string             `json:"id"`
	VersionID string             `json:"versionId,omitempty"`
	Name      string             `json:"name"`
	Status    SkillPackageStatus `json:"status"`
	Owner     ResourceRef        `json:"owner"`
}

type CreatorCollaborationStatus string

const (
	CreatorCollaborationStatusDraft              CreatorCollaborationStatus = "draft"
	CreatorCollaborationStatusBriefing           CreatorCollaborationStatus = "briefing"
	CreatorCollaborationStatusAwaitingAcceptance CreatorCollaborationStatus = "awaiting_acceptance"
	CreatorCollaborationStatusInProgress         CreatorCollaborationStatus = "in_progress"
	CreatorCollaborationStatusReviewing          CreatorCollaborationStatus = "reviewing"
	CreatorCollaborationStatusPublished          CreatorCollaborationStatus = "published"
	CreatorCollaborationStatusSettled            CreatorCollaborationStatus = "settled"
	CreatorCollaborationStatusCanceled           CreatorCollaborationStatus = "canceled"
)

type CreatorRef struct {
	ID             string   `json:"id"`
	DisplayName    string   `json:"displayName"`
	Verticals      []string `json:"verticals"`
	Verification   string   `json:"verification,omitempty"`
	PrimaryChannel string   `json:"primaryChannel,omitempty"`
}

type CommercialEvidenceType string

const (
	CommercialEvidenceDisclosure     CommercialEvidenceType = "disclosure"
	CommercialEvidenceUsageRight     CommercialEvidenceType = "usage_right"
	CommercialEvidenceApprovalRecord CommercialEvidenceType = "approval_record"
	CommercialEvidencePublication    CommercialEvidenceType = "publication"
	CommercialEvidenceSettlement     CommercialEvidenceType = "settlement"
)

type CommercialEvidence struct {
	ID           string                 `json:"id"`
	ProductLine  ProductLine            `json:"productLine"`
	WorkspaceID  string                 `json:"workspaceId,omitempty"`
	ResourceType ResourceType           `json:"resourceType"`
	ResourceID   string                 `json:"resourceId"`
	Type         CommercialEvidenceType `json:"type"`
	Title        string                 `json:"title"`
	Payload      map[string]any         `json:"payload"`
	CapturedBy   ResourceRef            `json:"capturedBy"`
	CreatedAt    time.Time              `json:"createdAt"`
}

type AuditActorType string

const (
	AuditActorUser     AuditActorType = "user"
	AuditActorPlatform AuditActorType = "platform"
	AuditActorSystem   AuditActorType = "system"
)

type AuditEvent struct {
	ID           string         `json:"id"`
	ProductLine  ProductLine    `json:"productLine"`
	WorkspaceID  string         `json:"workspaceId,omitempty"`
	ActorType    AuditActorType `json:"actorType"`
	ActorID      string         `json:"actorId,omitempty"`
	Action       string         `json:"action"`
	ResourceType ResourceType   `json:"resourceType"`
	ResourceID   string         `json:"resourceId,omitempty"`
	Metadata     map[string]any `json:"metadata"`
}

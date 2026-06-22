export type User = {
  id: string;
  name: string;
  email: string;
  isPlatformAdmin: boolean;
  subscriptionTier: 'free' | 'vip';
  subscriptionPlanId: 'free' | 'vip' | string;
  subscriptionStatus: 'active' | 'inactive' | 'expired' | 'canceled';
  subscriptionExpiresAt?: string;
  monthlyTokenBudgetCents: number;
  monthlyTokenUsedCents: number;
  monthlyTokenInputUsed: number;
  monthlyTokenOutputUsed: number;
  subscriptionCurrentPeriod: string;
  onboardingCompleted: boolean;
  onboardingCompletedAt?: string;
  createdAt: string;
};

export type SubscriptionPlan = {
  id: 'free' | 'vip' | string;
  name: string;
  tier: 'free' | 'vip';
  priceCents: number;
  currency: string;
  monthlyTokenBudgetCents: number;
  inputTokenPricePer1k: number;
  outputTokenPricePer1k: number;
  enabled: boolean;
};

export type WorkspaceType = 'personal' | 'company';

export type Workspace = {
  id: string;
  name: string;
  type: WorkspaceType;
  plan: string;
  status: string;
  industry: string;
  language: string;
  tone: string;
  createdAt: string;
};

export type KnowledgeBase = {
  id: string;
  workspaceId: string;
  name: string;
  description: string;
  itemCount: number;
  updatedAt: string;
};

export type KnowledgeItem = {
  id: string;
  knowledgeBaseIds: string[];
  workspaceId: string;
  type: string;
  title: string;
  content: string;
  enabled: boolean;
  updatedAt: string;
};

export type PlatformKnowledgeBase = {
  id: string;
  name: string;
  description: string;
  category: string;
  priceCents: number;
  currency: string;
  marketplaceListed: boolean;
  itemCount: number;
  updatedAt: string;
};

export type PlatformKnowledgeItem = {
  id: string;
  knowledgeBaseIds: string[];
  type: string;
  title: string;
  content: string;
  enabled: boolean;
  updatedAt: string;
};

export type MediaPlatform = {
  id: string;
  name: string;
  type: string;
  enabled: boolean;
  supportsArticle: boolean;
  supportsImage: boolean;
  supportsScheduling: boolean;
  credentialFields: string[];
};

export type MediaAccount = {
  id: string;
  workspaceId: string;
  platformId: string;
  name: string;
  externalId: string;
  loginMethod: string;
  credentialMeta?: Record<string, string>;
  status: string;
  expiresAt?: string;
  lastCheckedAt: string;
};

export type CreatorVerificationState = 'unverified' | 'pending' | 'verified' | 'rejected';

export type CreatorAvailabilityStatus = 'available' | 'limited' | 'unavailable';

export type Creator = {
  id: string;
  displayName: string;
  legalName?: string;
  bio: string;
  avatarUrl: string;
  contactEmail?: string;
  verticals: string[];
  audienceAttributes: Record<string, string>;
  basePriceCents: number;
  currency: string;
  availabilityStatus: CreatorAvailabilityStatus;
  collaborationPolicy: string;
  verificationState: CreatorVerificationState;
  brandSafetyLevel: string;
  createdAt: string;
  updatedAt: string;
};

export type CreatorMediaAccount = {
  id: string;
  creatorId: string;
  platformId: string;
  platformName: string;
  handle: string;
  profileUrl: string;
  followerCount: number;
  averageEngagementRate: number;
  verticals: string[];
  audienceAttributes: Record<string, string>;
  accountAccessMode: 'creator_operated' | 'agency_authorized' | 'public_profile' | string;
  verified: boolean;
  createdAt: string;
  updatedAt: string;
};

export type CreatorShortlist = {
  id: string;
  workspaceId: string;
  creatorId: string;
  name: string;
  fitScore: number;
  qualificationStatus: 'watching' | 'qualified' | 'rejected' | 'ordered' | string;
  brandSafetyLevel: string;
  brandSafetyNotes: string;
  operatorNotes: string;
  createdByUserId: string;
  createdAt: string;
  updatedAt: string;
};

export type CreatorCampaignBriefStatus = 'draft' | 'active' | 'archived';

export type CreatorCampaignBrief = {
  id: string;
  workspaceId: string;
  title: string;
  objective: string;
  productName: string;
  targetAudience: string;
  platformTargets: string[];
  deliverableRequirements: string[];
  disclosureRequirements: string[];
  prohibitedClaims: string[];
  authorizationScope: string;
  contentUsageRights: string;
  reviewWindowHours: number;
  deadlineAt?: string;
  budgetCents: number;
  currency: string;
  status: CreatorCampaignBriefStatus;
  createdByUserId: string;
  createdAt: string;
  updatedAt: string;
};

export type CreatorOrderStatus =
  | 'proposed'
  | 'accepted'
  | 'in_progress'
  | 'submitted'
  | 'approved'
  | 'published'
  | 'completed'
  | 'canceled'
  | 'disputed';

export type CreatorOrder = {
  id: string;
  workspaceId: string;
  briefId: string;
  creatorId: string;
  status: CreatorOrderStatus;
  priceCents: number;
  depositCents: number;
  serviceFeeCents: number;
  currency: string;
  disclosureRequirements: string[];
  deliverableRequirements: string[];
  authorizationScope: string;
  contentUsageRights: string;
  dueAt?: string;
  acceptedAt?: string;
  completedAt?: string;
  lastMessage: string;
  createdByUserId: string;
  createdAt: string;
  updatedAt: string;
};

export type CreatorDeliverableStatus = 'submitted' | 'revision_requested' | 'approved' | 'rejected' | 'published';

export type CreatorDeliverable = {
  id: string;
  workspaceId: string;
  orderId: string;
  creatorId: string;
  type: string;
  title: string;
  content: string;
  assetUrls: string[];
  status: CreatorDeliverableStatus;
  externalUrl: string;
  publicationProofUrl: string;
  publicationProofNote: string;
  reviewFeedback: string;
  revision: number;
  submittedAt: string;
  reviewedAt?: string;
  publishedAt?: string;
  createdAt: string;
  updatedAt: string;
};

export type CreatorSettlementStatus = 'pending' | 'invoiced' | 'payable' | 'paid' | 'refunded' | 'disputed' | 'canceled';

export type CreatorSettlement = {
  id: string;
  workspaceId: string;
  orderId: string;
  creatorId: string;
  status: CreatorSettlementStatus;
  priceCents: number;
  depositCents: number;
  serviceFeeCents: number;
  creatorPayoutCents: number;
  currency: string;
  invoiceId: string;
  dueAt?: string;
  paidAt?: string;
  createdAt: string;
  updatedAt: string;
};

export type CreatorComplianceEvidenceType =
  | 'ad_disclosure'
  | 'authorization_record'
  | 'usage_rights'
  | 'review_log'
  | 'publication_proof'
  | 'dispute_record';

export type CreatorComplianceEvidence = {
  id: string;
  workspaceId: string;
  orderId: string;
  deliverableId: string;
  creatorId: string;
  evidenceType: CreatorComplianceEvidenceType;
  disclosureText: string;
  authorizationScope: string;
  contentUsageRights: string;
  externalUrl: string;
  fileUrl: string;
  notes: string;
  createdByUserId: string;
  createdAt: string;
};

export type CreatorDetail = {
  creator: Creator;
  mediaAccounts: CreatorMediaAccount[];
  shortlists: CreatorShortlist[];
};

export type ContentStatus = 'draft' | 'review' | 'approved' | 'scheduled' | 'published' | 'failed' | 'archived';

export type Content = {
  id: string;
  workspaceId: string;
  knowledgeBaseId: string;
  title: string;
  summary: string;
  body: string;
  keywords: string[];
  status: ContentStatus;
  author: string;
  source: string;
  updatedAt: string;
};

export type GenerationPipelinePlan = {
  inputAnalysis: boolean;
  contentPlan: boolean;
  qualityCheck: boolean;
  rewriteRounds: number;
};

export type GenerationPipelineSettings = {
  free: GenerationPipelinePlan;
  vip: GenerationPipelinePlan;
};

export type GenerationTraceStep = {
  id: string;
  label: string;
  status: 'succeeded' | 'failed' | 'skipped' | string;
  summary: string;
  details: string[];
  warnings: string[];
};

export type GenerationTrace = {
  subscriptionTier: string;
  pipeline: GenerationPipelinePlan;
  steps: GenerationTraceStep[];
  warnings: string[];
  retrievedKnowledgeIds: string[];
};

export type GenerateContentResponse = {
  content: Content;
  trace: GenerationTrace;
};

export type PublishScheduleFrequency = 'once' | 'daily' | 'weekly' | 'monthly';

export type PublishSchedule = {
  id: string;
  workspaceId: string;
  name: string;
  contentId: string;
  mediaAccountId: string;
  frequency: PublishScheduleFrequency;
  nextRunAt: string;
  enabled: boolean;
  createdAt: string;
};

export type PublishJobStatus = 'queued' | 'running' | 'manual_pending' | 'succeeded' | 'failed' | 'retrying';

export type PublishJob = {
  id: string;
  workspaceId: string;
  scheduleId: string;
  contentId: string;
  mediaAccountId: string;
  status: PublishJobStatus;
  scheduledAt: string;
  externalUrl: string;
  lastMessage: string;
};

export type PreparedPostCopyBlock = {
  label: string;
  value: string;
};

export type PreparedPost = {
  mode: string;
  platformType: string;
  platformName: string;
  publishFormatId: string;
  publishMode: string;
  title: string;
  body: string;
  hashtags: string[];
  copyBlocks: PreparedPostCopyBlock[];
  checklist: string[];
  warnings: string[];
  characterCount: number;
  preparedAt: string;
};

export type PublishResult = {
  status: string;
  message: string;
  externalId: string;
  externalUrl: string;
  rawResponse: Record<string, unknown>;
};

export type Overview = {
  workspaceId: string;
  knowledgeBaseCount: number;
  mediaAccountCount: number;
  contentCount: number;
  scheduleCount: number;
  publishJobCount: number;
  draftCount: number;
  queuedJobs: number;
  failedJobs: number;
};

export type LoginResponse = {
  token: string;
  user: User;
  workspaces: Workspace[];
};

export type RegisterPayload = {
  name: string;
  email: string;
  password: string;
  workspaceName?: string;
};

export type CompleteOnboardingPayload = {
  workspaceId: string;
  industry: string;
  tones: string[];
  subscriptionPlanId?: string;
  skipSubscription?: boolean;
};

export type CompleteOnboardingResponse = {
  user: User;
  workspace: Workspace;
};

export type WorkspaceData = {
  user: User;
  workspaces: Workspace[];
  overview: Overview;
  knowledgeBases: KnowledgeBase[];
  knowledgeItems: KnowledgeItem[];
  mediaPlatforms: MediaPlatform[];
  mediaAccounts: MediaAccount[];
  contents: Content[];
  publishSchedules: PublishSchedule[];
  publishJobs: PublishJob[];
};

export type CreateKnowledgeBasePayload = {
  name: string;
  description: string;
};

export type CreateKnowledgeItemPayload = {
  knowledgeBaseId?: string;
  knowledgeBaseIds?: string[];
  type: string;
  title: string;
  content: string;
};

export type FormatKnowledgeContentPayload = {
  type: string;
  title?: string;
  content: string;
};

export type FormatKnowledgeContentResponse = {
  content: string;
  provider: string;
  model: string;
  fallback?: boolean;
  fallbackError?: string;
  tokenUsage: {
    inputTokens: number;
    outputTokens: number;
    totalTokens: number;
  };
};

export type AssignKnowledgeItemsToBasesPayload = {
  knowledgeItemIds: string[];
  knowledgeBaseIds: string[];
};

export type CreatePlatformKnowledgeBasePayload = {
  name: string;
  description: string;
  category: string;
  priceCents: number;
  currency: string;
  marketplaceListed: boolean;
};

export type CreatePlatformKnowledgeItemPayload = {
  knowledgeBaseId?: string;
  knowledgeBaseIds?: string[];
  type: string;
  title: string;
  content: string;
  enabled: boolean;
};

export type CreateMediaAccountPayload = {
  platformId: string;
  name: string;
  externalId: string;
  loginMethod?: string;
  phoneNumber?: string;
};

export type CreateCreatorShortlistPayload = {
  creatorId: string;
  name?: string;
  fitScore?: number;
  qualificationStatus?: string;
  brandSafetyLevel?: string;
  brandSafetyNotes?: string;
  operatorNotes?: string;
};

export type CreateCreatorCampaignBriefPayload = {
  title: string;
  objective?: string;
  productName?: string;
  targetAudience?: string;
  platformTargets?: string[];
  deliverableRequirements?: string[];
  disclosureRequirements?: string[];
  prohibitedClaims?: string[];
  authorizationScope?: string;
  contentUsageRights?: string;
  reviewWindowHours?: number;
  deadlineAt?: string;
  budgetCents?: number;
  currency?: string;
  status?: CreatorCampaignBriefStatus;
};

export type CreateCreatorOrderPayload = {
  briefId: string;
  creatorId: string;
  priceCents?: number;
  depositCents?: number;
  serviceFeeCents?: number;
  currency?: string;
  disclosureRequirements?: string[];
  deliverableRequirements?: string[];
  authorizationScope?: string;
  contentUsageRights?: string;
  dueAt?: string;
  lastMessage?: string;
};

export type CreateCreatorOrderResponse = {
  order: CreatorOrder;
  settlement: CreatorSettlement;
};

export type SubmitCreatorDeliverablePayload = {
  type?: string;
  title?: string;
  content?: string;
  assetUrls?: string[];
};

export type ReviewCreatorDeliverablePayload = {
  decision: 'approve' | 'request_revision' | 'reject';
  feedback?: string;
};

export type RecordCreatorPublicationProofPayload = {
  externalUrl: string;
  publicationProofUrl?: string;
  publicationProofNote?: string;
  disclosureText: string;
  notes?: string;
  publishedAt?: string;
};

export type RecordCreatorPublicationProofResponse = {
  deliverable: CreatorDeliverable;
  order: CreatorOrder;
  settlement: CreatorSettlement;
  evidence: CreatorComplianceEvidence;
};

export type StartMediaAccountBrowserLoginPayload = Record<string, never>;

export type StartMediaAccountBrowserLoginResponse = {
  account: MediaAccount;
  expiresAt: string;
  mode?: string;
  qrScreenshotData: string;
  qrLoginUrl: string;
  sessionId: string;
  browserProfile: string;
  stateFile: string;
};

export type CompleteMediaAccountBrowserLoginPayload = {
  sessionId: string;
};

export type GenerateContentPayload = {
  keywords: string[];
  keywordPrompt?: string;
  contentType: string;
  knowledgeBaseId?: string;
  knowledgeBaseIds?: string[];
  publishFormatId?: string;
};

export type CreateContentPayload = {
  title: string;
  summary: string;
  body: string;
  author: string;
  knowledgeBaseId: string;
  keywords: string[];
};

export type CreatePublishSchedulePayload = {
  name: string;
  contentId: string;
  mediaAccountId: string;
  frequency: PublishScheduleFrequency;
  nextRunAt: string;
};

export type PreparePublishPayload = {
  contentId: string;
  mediaAccountId: string;
  publishFormatId?: string;
  assetPaths?: string[];
  runNow?: boolean;
};

export type PreparePublishResponse = {
  job: PublishJob;
  preparedPost: PreparedPost;
  publishResult?: PublishResult;
};

export type ConfirmPublishPayload = {
  externalUrl: string;
  message?: string;
};

export type RunPublishJobPayload = {
  assetPaths?: string[];
  preparedPost?: PreparedPost;
};

export type RunPublishJobResponse = {
  job: PublishJob;
  preparedPost: PreparedPost;
  publishResult: PublishResult;
};

export type CreateMediaPlatformPayload = {
  name: string;
  type: string;
  enabled: boolean;
  supportsArticle: boolean;
  supportsImage: boolean;
  supportsScheduling: boolean;
  credentialFields: string[];
};

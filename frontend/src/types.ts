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

export type BrandAssetStatus = 'active' | 'archived';

export type BrandAsset = {
  id: string;
  workspaceId: string;
  type: string;
  name: string;
  description: string;
  content: string;
  channels: string[];
  tags: string[];
  source: string;
  status: BrandAssetStatus;
  metadata: Record<string, string>;
  createdAt: string;
  updatedAt: string;
};

export type BrandGuardrail = {
  id: string;
  workspaceId: string;
  assetId: string;
  name: string;
  category: string;
  channel: string;
  sourceType: string;
  sourceId: string;
  severity: string;
  rules: string[];
  action: string;
  enabled: boolean;
  createdAt: string;
  updatedAt: string;
};

export type ApprovalWorkflowStatus = 'draft' | 'active' | 'completed' | 'canceled';

export type ApprovalStage = {
  name: string;
  approverRole: string;
  requiredApprovals: number;
};

export type ApprovalWorkflow = {
  id: string;
  workspaceId: string;
  resourceType: string;
  resourceId: string;
  name: string;
  status: ApprovalWorkflowStatus;
  stages: ApprovalStage[];
  createdByUserId: string;
  createdAt: string;
  updatedAt: string;
};

export type ApprovalTaskStatus = 'pending' | 'approved' | 'rejected' | 'skipped' | 'canceled';

export type ApprovalTask = {
  id: string;
  workspaceId: string;
  workflowId: string;
  resourceType: string;
  resourceId: string;
  stageName: string;
  assigneeUserId: string;
  assigneeRole: string;
  status: ApprovalTaskStatus;
  decision: string;
  comment: string;
  processedByUserId: string;
  dueAt?: string;
  processedAt?: string;
  createdAt: string;
  updatedAt: string;
};

export type ComplianceFinding = {
  id: string;
  checkId: string;
  workspaceId: string;
  severity: string;
  category: string;
  evidence: string;
  finding: string;
  action: string;
  sourceType: string;
  sourceId: string;
  createdAt: string;
};

export type ComplianceCheckStatus = 'queued' | 'running' | 'completed' | 'failed';

export type ComplianceCheck = {
  id: string;
  workspaceId: string;
  resourceType: string;
  resourceId: string;
  channel: string;
  status: ComplianceCheckStatus;
  riskLevel: string;
  summary: string;
  findings: ComplianceFinding[];
  createdByUserId: string;
  createdAt: string;
  updatedAt: string;
};

export type AgencyClientRelation = {
  id: string;
  agencyWorkspaceId: string;
  clientWorkspaceId: string;
  clientName: string;
  status: string;
  scopes: string[];
  notes: string;
  createdAt: string;
  updatedAt: string;
};

export type ReportPackage = {
  id: string;
  workspaceId: string;
  name: string;
  reportType: string;
  audience: string;
  periodStart: string;
  periodEnd: string;
  status: string;
  sections: string[];
  metrics: Record<string, unknown>;
  summary: string;
  generatedByUserId: string;
  createdAt: string;
  updatedAt: string;
};

export type StrategyRecommendation = {
  id: string;
  workspaceId: string;
  sourceType: string;
  recommendationType: string;
  title: string;
  rationale: string;
  evidence: string[];
  action: string;
  confidence: number;
  status: string;
  createdAt: string;
  updatedAt: string;
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

export type CreateBrandAssetPayload = {
  type?: string;
  name: string;
  description?: string;
  content?: string;
  channels?: string[];
  tags?: string[];
  source?: string;
  status?: BrandAssetStatus;
  metadata?: Record<string, string>;
};

export type CreateBrandGuardrailPayload = {
  assetId?: string;
  name: string;
  category?: string;
  channel?: string;
  sourceType?: string;
  sourceId?: string;
  severity?: string;
  rules: string[];
  action?: string;
  enabled?: boolean;
};

export type CreateApprovalWorkflowPayload = {
  resourceType?: string;
  resourceId?: string;
  name: string;
  status?: 'draft' | 'active';
  stages: ApprovalStage[];
};

export type CreateApprovalWorkflowResponse = {
  workflow: ApprovalWorkflow;
  tasks: ApprovalTask[];
};

export type ProcessApprovalTaskPayload = {
  decision: 'approve' | 'reject' | 'skip' | 'cancel' | string;
  comment?: string;
};

export type SubmitComplianceCheckPayload = {
  resourceType?: string;
  resourceId?: string;
  channel?: string;
  title?: string;
  content?: string;
};

export type CreateAgencyClientRelationPayload = {
  clientWorkspaceId: string;
  clientName?: string;
  status?: string;
  scopes?: string[];
  notes?: string;
};

export type GenerateReportPackagePayload = {
  name?: string;
  reportType?: string;
  audience?: string;
  periodStart?: string;
  periodEnd?: string;
  sections?: string[];
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

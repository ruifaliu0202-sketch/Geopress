import type {
  AgencyClientRelation,
  AssignKnowledgeItemsToBasesPayload,
  ApprovalTask,
  ApprovalWorkflow,
  BrandAsset,
  BrandGuardrail,
  CompleteOnboardingPayload,
  CompleteOnboardingResponse,
  ComplianceCheck,
  Content,
  ContentMetric,
  CreateAgencyClientRelationPayload,
  CreateApprovalWorkflowPayload,
  CreateApprovalWorkflowResponse,
  CreateBrandAssetPayload,
  CreateBrandGuardrailPayload,
  CreateContentPayload,
  CreateKnowledgeBasePayload,
  CreateKnowledgeItemPayload,
  CreateMediaAccountPayload,
  CreatePublishSchedulePayload,
  GenerateReportPackagePayload,
  FormatKnowledgeContentPayload,
  FormatKnowledgeContentResponse,
  GenerateContentPayload,
  GenerateContentResponse,
  InstalledSkillPackage,
  KnowledgeBase,
  KnowledgeItem,
  LoginResponse,
  MediaAccount,
  MediaAccountMatrixItem,
  MediaAccountMetricSnapshot,
  MediaAccountSyncJob,
  MediaPlatform,
  Overview,
  ConfirmPublishPayload,
  PreparePublishPayload,
  PreparePublishResponse,
  RequestMediaAccountSyncPayload,
  ProcessApprovalTaskPayload,
  ReportPackage,
  RunPublishJobPayload,
  RunPublishJobResponse,
  SkillPackageMarketplaceItem,
  SkillPackageUsageMetric,
  StartMediaAccountBrowserLoginPayload,
  StartMediaAccountBrowserLoginResponse,
  CompleteMediaAccountBrowserLoginPayload,
  CreateCreatorCampaignBriefPayload,
  CreateCreatorOrderPayload,
  CreateCreatorOrderResponse,
  CreateCreatorShortlistPayload,
  StrategyRecommendation,
  SubmitComplianceCheckPayload,
  PublishJob,
  PublishSchedule,
  Campaign,
  CampaignCalendarItem,
  CampaignReportSummary,
  CreateCampaignPayload,
  UpdateCampaignPayload,
  CreateCampaignCalendarItemPayload,
  Creator,
  CreatorCampaignBrief,
  CreatorComplianceEvidence,
  CreatorDeliverable,
  CreatorDetail,
  CreatorOrder,
  CreatorSettlement,
  CreatorShortlist,
  RecordCreatorPublicationProofPayload,
  RecordCreatorPublicationProofResponse,
  RegisterPayload,
  ReviewCreatorDeliverablePayload,
  SubscriptionPlan,
  SubmitCreatorDeliverablePayload,
  User,
  WorkspaceSkillEntitlement,
  Workspace,
  WorkspaceData,
} from './types';

const API_BASE_URL = import.meta.env.VITE_API_BASE_URL ?? '/api';

type ListResponse<T> = {
  items: T[];
};

export class ApiRequestError extends Error {
  status: number;

  constructor(message: string, status: number) {
    super(message);
    this.name = 'ApiRequestError';
    this.status = status;
  }
}

type MeResponse = {
  user: User;
  workspaces: Workspace[];
};

async function request<T>(path: string, token?: string, workspaceId?: string, init?: RequestInit): Promise<T> {
  const headers = new Headers(init?.headers);
  if (init?.body && !headers.has('Content-Type')) {
    headers.set('Content-Type', 'application/json');
  }
  if (token) {
    headers.set('Authorization', `Bearer ${token}`);
  }
  if (workspaceId) {
    headers.set('X-Workspace-ID', workspaceId);
  }

  const response = await fetch(`${API_BASE_URL}${path}`, {
    ...init,
    headers,
  });

  if (!response.ok) {
    const message = await response.text();
    let errorMessage = message;
    try {
      const data = JSON.parse(message) as { error?: string };
      if (data.error) {
        errorMessage = data.error;
      }
    } catch {
      errorMessage = message;
    }
    throw new ApiRequestError(errorMessage || `API request failed: ${response.status}`, response.status);
  }

  return response.json() as Promise<T>;
}

export async function login(email: string, password: string): Promise<LoginResponse> {
  return request<LoginResponse>('/auth/login', undefined, undefined, {
    method: 'POST',
    body: JSON.stringify({ email, password }),
  });
}

export async function registerUser(payload: RegisterPayload): Promise<LoginResponse> {
  return request<LoginResponse>('/auth/register', undefined, undefined, {
    method: 'POST',
    body: JSON.stringify(payload),
  });
}

export async function fetchSubscriptionPlans(token: string, workspaceId: string): Promise<SubscriptionPlan[]> {
  const response = await request<ListResponse<SubscriptionPlan>>('/subscription-plans', token, workspaceId);
  return response.items;
}

export async function completeOnboarding(
  token: string,
  workspaceId: string,
  payload: CompleteOnboardingPayload,
): Promise<CompleteOnboardingResponse> {
  return request<CompleteOnboardingResponse>('/onboarding/complete', token, workspaceId, {
    method: 'POST',
    body: JSON.stringify(payload),
  });
}

export async function fetchWorkspace(token: string, workspaceId: string): Promise<WorkspaceData> {
  const [me, overview, knowledgeBases, knowledgeItems, mediaPlatforms, mediaAccounts, contents, schedules, jobs] =
    await Promise.all([
      request<MeResponse>('/me', token, workspaceId),
      request<Overview>('/overview', token, workspaceId),
      request<ListResponse<KnowledgeBase>>('/knowledge-bases', token, workspaceId),
      request<ListResponse<KnowledgeItem>>('/knowledge-items', token, workspaceId),
      request<ListResponse<MediaPlatform>>('/media-platforms', token, workspaceId),
      request<ListResponse<MediaAccount>>('/media-accounts', token, workspaceId),
      request<ListResponse<Content>>('/contents', token, workspaceId),
      request<ListResponse<PublishSchedule>>('/publish-schedules', token, workspaceId),
      request<ListResponse<PublishJob>>('/publish-jobs', token, workspaceId),
    ]);

  return {
    user: me.user,
    workspaces: me.workspaces,
    overview,
    knowledgeBases: knowledgeBases.items,
    knowledgeItems: knowledgeItems.items,
    mediaPlatforms: mediaPlatforms.items,
    mediaAccounts: mediaAccounts.items,
    contents: contents.items,
    publishSchedules: schedules.items,
    publishJobs: jobs.items,
  };
}

export async function createKnowledgeBase(
  token: string,
  workspaceId: string,
  payload: CreateKnowledgeBasePayload,
): Promise<KnowledgeBase> {
  return request<KnowledgeBase>('/knowledge-bases', token, workspaceId, {
    method: 'POST',
    body: JSON.stringify(payload),
  });
}

export async function createKnowledgeItem(
  token: string,
  workspaceId: string,
  payload: CreateKnowledgeItemPayload,
): Promise<KnowledgeItem> {
  return request<KnowledgeItem>('/knowledge-items', token, workspaceId, {
    method: 'POST',
    body: JSON.stringify(payload),
  });
}

export async function formatKnowledgeContent(
  token: string,
  workspaceId: string,
  payload: FormatKnowledgeContentPayload,
): Promise<FormatKnowledgeContentResponse> {
  return request<FormatKnowledgeContentResponse>('/knowledge-items/format', token, workspaceId, {
    method: 'POST',
    body: JSON.stringify(payload),
  });
}

export async function assignKnowledgeItemsToBases(
  token: string,
  workspaceId: string,
  payload: AssignKnowledgeItemsToBasesPayload,
): Promise<ListResponse<KnowledgeItem>> {
  return request<ListResponse<KnowledgeItem>>('/knowledge-items/assign-bases', token, workspaceId, {
    method: 'POST',
    body: JSON.stringify(payload),
  });
}

export async function fetchSkillPackageMarketplace(
  token: string,
  workspaceId: string,
): Promise<SkillPackageMarketplaceItem[]> {
  const response = await request<ListResponse<SkillPackageMarketplaceItem>>('/skill-packages/marketplace', token, workspaceId);
  return response.items;
}

export async function fetchInstalledSkillPackages(
  token: string,
  workspaceId: string,
): Promise<InstalledSkillPackage[]> {
  const response = await request<ListResponse<InstalledSkillPackage>>('/skill-packages/installed', token, workspaceId);
  return response.items;
}

export async function installSkillPackage(
  token: string,
  workspaceId: string,
  packageId: string,
  payload: { versionId?: string; seats?: number } = {},
): Promise<WorkspaceSkillEntitlement> {
  return request<WorkspaceSkillEntitlement>(`/skill-package-entitlements/${packageId}/install`, token, workspaceId, {
    method: 'POST',
    body: JSON.stringify(payload),
  });
}

export async function purchaseSkillPackage(
  token: string,
  workspaceId: string,
  packageId: string,
  payload: { versionId?: string; seats?: number } = {},
): Promise<WorkspaceSkillEntitlement> {
  return request<WorkspaceSkillEntitlement>(`/skill-package-entitlements/${packageId}/purchase`, token, workspaceId, {
    method: 'POST',
    body: JSON.stringify(payload),
  });
}

export async function subscribeSkillPackage(
  token: string,
  workspaceId: string,
  packageId: string,
  payload: { versionId?: string; seats?: number } = {},
): Promise<WorkspaceSkillEntitlement> {
  return request<WorkspaceSkillEntitlement>(`/skill-package-entitlements/${packageId}/subscribe`, token, workspaceId, {
    method: 'POST',
    body: JSON.stringify(payload),
  });
}

export async function fetchSkillPackageUsage(
  token: string,
  workspaceId: string,
  packageId?: string,
): Promise<SkillPackageUsageMetric[]> {
  const query = packageId ? `?packageId=${encodeURIComponent(packageId)}` : '';
  const response = await request<ListResponse<SkillPackageUsageMetric>>(`/skill-packages/usage${query}`, token, workspaceId);
  return response.items;
}

export async function createMediaAccount(
  token: string,
  workspaceId: string,
  payload: CreateMediaAccountPayload,
): Promise<MediaAccount> {
  return request<MediaAccount>('/media-accounts', token, workspaceId, {
    method: 'POST',
    body: JSON.stringify(payload),
  });
}

export async function fetchCreators(token: string, workspaceId: string): Promise<Creator[]> {
  const response = await request<ListResponse<Creator>>('/creators', token, workspaceId);
  return response.items;
}

export async function fetchCreator(token: string, workspaceId: string, creatorId: string): Promise<CreatorDetail> {
  return request<CreatorDetail>(`/creators/${creatorId}`, token, workspaceId);
}

export async function fetchCreatorShortlists(token: string, workspaceId: string): Promise<CreatorShortlist[]> {
  const response = await request<ListResponse<CreatorShortlist>>('/creator-shortlists', token, workspaceId);
  return response.items;
}

export async function createCreatorShortlist(
  token: string,
  workspaceId: string,
  payload: CreateCreatorShortlistPayload,
): Promise<CreatorShortlist> {
  return request<CreatorShortlist>('/creator-shortlists', token, workspaceId, {
    method: 'POST',
    body: JSON.stringify(payload),
  });
}

export async function fetchCreatorCampaignBriefs(token: string, workspaceId: string): Promise<CreatorCampaignBrief[]> {
  const response = await request<ListResponse<CreatorCampaignBrief>>('/creator-briefs', token, workspaceId);
  return response.items;
}

export async function createCreatorCampaignBrief(
  token: string,
  workspaceId: string,
  payload: CreateCreatorCampaignBriefPayload,
): Promise<CreatorCampaignBrief> {
  return request<CreatorCampaignBrief>('/creator-briefs', token, workspaceId, {
    method: 'POST',
    body: JSON.stringify(payload),
  });
}

export async function fetchCreatorOrders(token: string, workspaceId: string): Promise<CreatorOrder[]> {
  const response = await request<ListResponse<CreatorOrder>>('/creator-orders', token, workspaceId);
  return response.items;
}

export async function createCreatorOrder(
  token: string,
  workspaceId: string,
  payload: CreateCreatorOrderPayload,
): Promise<CreateCreatorOrderResponse> {
  return request<CreateCreatorOrderResponse>('/creator-orders', token, workspaceId, {
    method: 'POST',
    body: JSON.stringify(payload),
  });
}

export async function fetchCreatorDeliverables(token: string, workspaceId: string): Promise<CreatorDeliverable[]> {
  const response = await request<ListResponse<CreatorDeliverable>>('/creator-deliverables', token, workspaceId);
  return response.items;
}

export async function submitCreatorDeliverable(
  token: string,
  workspaceId: string,
  orderId: string,
  payload: SubmitCreatorDeliverablePayload,
): Promise<CreatorDeliverable> {
  return request<CreatorDeliverable>(`/creator-orders/${orderId}/deliverables`, token, workspaceId, {
    method: 'POST',
    body: JSON.stringify(payload),
  });
}

export async function reviewCreatorDeliverable(
  token: string,
  workspaceId: string,
  deliverableId: string,
  payload: ReviewCreatorDeliverablePayload,
): Promise<CreatorDeliverable> {
  return request<CreatorDeliverable>(`/creator-deliverables/${deliverableId}/review`, token, workspaceId, {
    method: 'POST',
    body: JSON.stringify(payload),
  });
}

export async function recordCreatorPublicationProof(
  token: string,
  workspaceId: string,
  deliverableId: string,
  payload: RecordCreatorPublicationProofPayload,
): Promise<RecordCreatorPublicationProofResponse> {
  return request<RecordCreatorPublicationProofResponse>(
    `/creator-deliverables/${deliverableId}/publication-proof`,
    token,
    workspaceId,
    {
      method: 'POST',
      body: JSON.stringify(payload),
    },
  );
}

export async function fetchCreatorSettlements(token: string, workspaceId: string): Promise<CreatorSettlement[]> {
  const response = await request<ListResponse<CreatorSettlement>>('/creator-settlements', token, workspaceId);
  return response.items;
}

export async function fetchCreatorComplianceEvidence(
  token: string,
  workspaceId: string,
): Promise<CreatorComplianceEvidence[]> {
  const response = await request<ListResponse<CreatorComplianceEvidence>>('/creator-compliance-evidence', token, workspaceId);
  return response.items;
}

export async function startMediaAccountBrowserLogin(
  token: string,
  workspaceId: string,
  accountId: string,
  payload: StartMediaAccountBrowserLoginPayload,
): Promise<StartMediaAccountBrowserLoginResponse> {
  return request<StartMediaAccountBrowserLoginResponse>(`/media-accounts/${accountId}/browser-login/start`, token, workspaceId, {
    method: 'POST',
    body: JSON.stringify(payload),
  });
}

export async function completeMediaAccountBrowserLogin(
  token: string,
  workspaceId: string,
  accountId: string,
  payload: CompleteMediaAccountBrowserLoginPayload,
): Promise<MediaAccount> {
  return request<MediaAccount>(`/media-accounts/${accountId}/browser-login/complete`, token, workspaceId, {
    method: 'POST',
    body: JSON.stringify(payload),
  });
}

export async function fetchMediaAccountMatrix(
  token: string,
  workspaceId: string,
): Promise<MediaAccountMatrixItem[]> {
  const response = await request<ListResponse<MediaAccountMatrixItem>>('/media-account-matrix', token, workspaceId);
  return response.items;
}

export async function fetchMediaAccountMatrixItem(
  token: string,
  workspaceId: string,
  accountId: string,
): Promise<MediaAccountMatrixItem> {
  return request<MediaAccountMatrixItem>(`/media-account-matrix/${accountId}`, token, workspaceId);
}

export async function fetchMediaAccountMetricSnapshots(
  token: string,
  workspaceId: string,
  accountId: string,
  limit = 90,
): Promise<MediaAccountMetricSnapshot[]> {
  const response = await request<ListResponse<MediaAccountMetricSnapshot>>(
    `/media-account-matrix/${accountId}/metric-snapshots?limit=${limit}`,
    token,
    workspaceId,
  );
  return response.items;
}

export async function fetchContentMetrics(
  token: string,
  workspaceId: string,
  params: { mediaAccountId?: string; contentId?: string; limit?: number } = {},
): Promise<ContentMetric[]> {
  const query = new URLSearchParams();
  if (params.mediaAccountId) {
    query.set('mediaAccountId', params.mediaAccountId);
  }
  if (params.contentId) {
    query.set('contentId', params.contentId);
  }
  if (params.limit) {
    query.set('limit', String(params.limit));
  }
  const suffix = query.toString() ? `?${query.toString()}` : '';
  const response = await request<ListResponse<ContentMetric>>(`/content-metrics${suffix}`, token, workspaceId);
  return response.items;
}

export async function requestMediaAccountSync(
  token: string,
  workspaceId: string,
  accountId: string,
  payload: RequestMediaAccountSyncPayload = {},
): Promise<MediaAccountSyncJob> {
  return request<MediaAccountSyncJob>(`/media-account-matrix/${accountId}/sync-jobs`, token, workspaceId, {
    method: 'POST',
    body: JSON.stringify(payload),
  });
}

export async function createContent(
  token: string,
  workspaceId: string,
  payload: CreateContentPayload,
): Promise<Content> {
  return request<Content>('/contents', token, workspaceId, {
    method: 'POST',
    body: JSON.stringify(payload),
  });
}

export async function generateContent(
  token: string,
  workspaceId: string,
  payload: GenerateContentPayload,
): Promise<GenerateContentResponse> {
  const response = await request<GenerateContentResponse | Content>('/contents/generate', token, workspaceId, {
    method: 'POST',
    body: JSON.stringify(payload),
  });
  if ('content' in response && 'trace' in response) {
    return response;
  }
  return {
    content: response,
    trace: {
      subscriptionTier: '',
      pipeline: { inputAnalysis: false, contentPlan: false, qualityCheck: false, rewriteRounds: 0 },
      steps: [],
      warnings: [],
      retrievedKnowledgeIds: [],
    },
  };
}

export async function createPublishSchedule(
  token: string,
  workspaceId: string,
  payload: CreatePublishSchedulePayload,
): Promise<{ schedule: PublishSchedule; job: PublishJob }> {
  return request<{ schedule: PublishSchedule; job: PublishJob }>('/publish-schedules', token, workspaceId, {
    method: 'POST',
    body: JSON.stringify(payload),
  });
}

export async function fetchCampaigns(token: string, workspaceId: string): Promise<Campaign[]> {
  const response = await request<ListResponse<Campaign>>('/campaigns', token, workspaceId);
  return response.items;
}

export async function createCampaign(
  token: string,
  workspaceId: string,
  payload: CreateCampaignPayload,
): Promise<Campaign> {
  return request<Campaign>('/campaigns', token, workspaceId, {
    method: 'POST',
    body: JSON.stringify(payload),
  });
}

export async function updateCampaign(
  token: string,
  workspaceId: string,
  campaignId: string,
  payload: UpdateCampaignPayload,
): Promise<Campaign> {
  return request<Campaign>(`/campaigns/${campaignId}`, token, workspaceId, {
    method: 'PUT',
    body: JSON.stringify(payload),
  });
}

export async function fetchCampaignCalendarItems(
  token: string,
  workspaceId: string,
  campaignId: string,
): Promise<CampaignCalendarItem[]> {
  const response = await request<ListResponse<CampaignCalendarItem>>(
    `/campaigns/${campaignId}/calendar-items`,
    token,
    workspaceId,
  );
  return response.items;
}

export async function createCampaignCalendarItem(
  token: string,
  workspaceId: string,
  campaignId: string,
  payload: CreateCampaignCalendarItemPayload,
): Promise<CampaignCalendarItem> {
  return request<CampaignCalendarItem>(`/campaigns/${campaignId}/calendar-items`, token, workspaceId, {
    method: 'POST',
    body: JSON.stringify(payload),
  });
}

export async function fetchCampaignReportSummary(
  token: string,
  workspaceId: string,
  campaignId: string,
): Promise<CampaignReportSummary> {
  return request<CampaignReportSummary>(`/campaigns/${campaignId}/report-summary`, token, workspaceId);
}

export async function preparePublish(
  token: string,
  workspaceId: string,
  payload: PreparePublishPayload,
): Promise<PreparePublishResponse> {
  return request<PreparePublishResponse>('/publish/prepare', token, workspaceId, {
    method: 'POST',
    body: JSON.stringify(payload),
  });
}

export async function confirmPublishJob(
  token: string,
  workspaceId: string,
  jobId: string,
  payload: ConfirmPublishPayload,
): Promise<PublishJob> {
  return request<PublishJob>(`/publish-jobs/${jobId}/confirm`, token, workspaceId, {
    method: 'POST',
    body: JSON.stringify(payload),
  });
}

export async function runPublishJob(
  token: string,
  workspaceId: string,
  jobId: string,
  payload: RunPublishJobPayload,
): Promise<RunPublishJobResponse> {
  return request<RunPublishJobResponse>(`/publish-jobs/${jobId}/run`, token, workspaceId, {
    method: 'POST',
    body: JSON.stringify(payload),
  });
}

export async function fetchBrandAssets(token: string, workspaceId: string): Promise<BrandAsset[]> {
  const response = await request<ListResponse<BrandAsset>>('/brand-assets', token, workspaceId);
  return response.items;
}

export async function createBrandAsset(
  token: string,
  workspaceId: string,
  payload: CreateBrandAssetPayload,
): Promise<BrandAsset> {
  return request<BrandAsset>('/brand-assets', token, workspaceId, {
    method: 'POST',
    body: JSON.stringify(payload),
  });
}

export async function updateBrandAsset(
  token: string,
  workspaceId: string,
  assetId: string,
  payload: CreateBrandAssetPayload,
): Promise<BrandAsset> {
  return request<BrandAsset>(`/brand-assets/${assetId}`, token, workspaceId, {
    method: 'PUT',
    body: JSON.stringify(payload),
  });
}

export async function archiveBrandAsset(token: string, workspaceId: string, assetId: string): Promise<BrandAsset> {
  return request<BrandAsset>(`/brand-assets/${assetId}`, token, workspaceId, {
    method: 'DELETE',
  });
}

export async function fetchBrandGuardrails(token: string, workspaceId: string): Promise<BrandGuardrail[]> {
  const response = await request<ListResponse<BrandGuardrail>>('/brand-guardrails', token, workspaceId);
  return response.items;
}

export async function createBrandGuardrail(
  token: string,
  workspaceId: string,
  payload: CreateBrandGuardrailPayload,
): Promise<BrandGuardrail> {
  return request<BrandGuardrail>('/brand-guardrails', token, workspaceId, {
    method: 'POST',
    body: JSON.stringify(payload),
  });
}

export async function fetchApprovalWorkflows(token: string, workspaceId: string): Promise<ApprovalWorkflow[]> {
  const response = await request<ListResponse<ApprovalWorkflow>>('/approval-workflows', token, workspaceId);
  return response.items;
}

export async function createApprovalWorkflow(
  token: string,
  workspaceId: string,
  payload: CreateApprovalWorkflowPayload,
): Promise<CreateApprovalWorkflowResponse> {
  return request<CreateApprovalWorkflowResponse>('/approval-workflows', token, workspaceId, {
    method: 'POST',
    body: JSON.stringify(payload),
  });
}

export async function fetchApprovalTasks(token: string, workspaceId: string): Promise<ApprovalTask[]> {
  const response = await request<ListResponse<ApprovalTask>>('/approval-tasks', token, workspaceId);
  return response.items;
}

export async function processApprovalTask(
  token: string,
  workspaceId: string,
  taskId: string,
  payload: ProcessApprovalTaskPayload,
): Promise<ApprovalTask> {
  return request<ApprovalTask>(`/approval-tasks/${taskId}/process`, token, workspaceId, {
    method: 'POST',
    body: JSON.stringify(payload),
  });
}

export async function fetchComplianceChecks(token: string, workspaceId: string): Promise<ComplianceCheck[]> {
  const response = await request<ListResponse<ComplianceCheck>>('/compliance-checks', token, workspaceId);
  return response.items;
}

export async function submitComplianceCheck(
  token: string,
  workspaceId: string,
  payload: SubmitComplianceCheckPayload,
): Promise<ComplianceCheck> {
  return request<ComplianceCheck>('/compliance-checks', token, workspaceId, {
    method: 'POST',
    body: JSON.stringify(payload),
  });
}

export async function fetchAgencyClientRelations(
  token: string,
  workspaceId: string,
): Promise<AgencyClientRelation[]> {
  const response = await request<ListResponse<AgencyClientRelation>>('/agency-client-relations', token, workspaceId);
  return response.items;
}

export async function createAgencyClientRelation(
  token: string,
  workspaceId: string,
  payload: CreateAgencyClientRelationPayload,
): Promise<AgencyClientRelation> {
  return request<AgencyClientRelation>('/agency-client-relations', token, workspaceId, {
    method: 'POST',
    body: JSON.stringify(payload),
  });
}

export async function fetchReportPackages(token: string, workspaceId: string): Promise<ReportPackage[]> {
  const response = await request<ListResponse<ReportPackage>>('/report-packages', token, workspaceId);
  return response.items;
}

export async function generateReportPackage(
  token: string,
  workspaceId: string,
  payload: GenerateReportPackagePayload,
): Promise<ReportPackage> {
  return request<ReportPackage>('/report-packages/generate', token, workspaceId, {
    method: 'POST',
    body: JSON.stringify(payload),
  });
}

export async function fetchStrategyRecommendations(
  token: string,
  workspaceId: string,
): Promise<StrategyRecommendation[]> {
  const response = await request<ListResponse<StrategyRecommendation>>('/strategy-recommendations', token, workspaceId);
  return response.items;
}

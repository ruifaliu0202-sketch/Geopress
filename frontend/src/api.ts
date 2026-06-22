import type {
  AssignKnowledgeItemsToBasesPayload,
  CompleteOnboardingPayload,
  CompleteOnboardingResponse,
  Content,
  ContentMetric,
  CreateContentPayload,
  CreateKnowledgeBasePayload,
  CreateKnowledgeItemPayload,
  CreateMediaAccountPayload,
  CreatePublishSchedulePayload,
  FormatKnowledgeContentPayload,
  FormatKnowledgeContentResponse,
  GenerateContentPayload,
  GenerateContentResponse,
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
  RunPublishJobPayload,
  RunPublishJobResponse,
  StartMediaAccountBrowserLoginPayload,
  StartMediaAccountBrowserLoginResponse,
  CompleteMediaAccountBrowserLoginPayload,
  PublishJob,
  PublishSchedule,
  RegisterPayload,
  SubscriptionPlan,
  User,
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

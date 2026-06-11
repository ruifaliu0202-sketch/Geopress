import type {
  Content,
  CreateContentPayload,
  CreateKnowledgeBasePayload,
  CreateKnowledgeItemPayload,
  CreateMediaAccountPayload,
  CreatePublishSchedulePayload,
  GenerateContentPayload,
  KnowledgeBase,
  KnowledgeItem,
  LoginResponse,
  MediaAccount,
  MediaPlatform,
  Overview,
  PublishJob,
  PublishSchedule,
  User,
  Workspace,
  WorkspaceData,
} from './types';

const API_BASE_URL = import.meta.env.VITE_API_BASE_URL ?? '/api';

type ListResponse<T> = {
  items: T[];
};

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
    throw new Error(`API request failed: ${response.status}`);
  }

  return response.json() as Promise<T>;
}

export async function login(email: string, password: string): Promise<LoginResponse> {
  return request<LoginResponse>('/auth/login', undefined, undefined, {
    method: 'POST',
    body: JSON.stringify({ email, password }),
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
): Promise<Content> {
  return request<Content>('/contents/generate', token, workspaceId, {
    method: 'POST',
    body: JSON.stringify(payload),
  });
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

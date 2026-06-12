import type { DataProvider, Identifier, RaRecord } from 'react-admin';
import type { CreateMediaPlatformPayload } from '../types';

const API_BASE_URL = import.meta.env.VITE_API_BASE_URL ?? '/api';

export type AdminAIConfig = {
  provider: 'mock' | 'openai';
  openAIBaseUrl: string;
  openAIModel: string;
  requestTimeoutSeconds: number;
  apiKeyConfigured: boolean;
  apiKeyPreview: string;
};

export type UpdateAdminAIConfigPayload = {
  provider: 'mock' | 'openai';
  openAIBaseUrl: string;
  openAIModel: string;
  openAIAPIKey?: string;
  requestTimeoutSeconds: number;
  clearAPIKey?: boolean;
};

type ListResponse<T> = {
  items: T[];
};

const resourcePath: Record<string, string> = {
  users: '/admin/users',
  workspaces: '/admin/workspaces',
  members: '/admin/workspace-members',
  mediaPlatforms: '/admin/media-platforms',
  mediaAccounts: '/admin/media-accounts',
};

async function request<T>(token: string, path: string, init?: RequestInit): Promise<T> {
  const headers = new Headers(init?.headers);
  headers.set('Authorization', `Bearer ${token}`);
  if (init?.body && !headers.has('Content-Type')) {
    headers.set('Content-Type', 'application/json');
  }

  const response = await fetch(`${API_BASE_URL}${path}`, { ...init, headers });
  if (!response.ok) {
    throw new Error(`Admin API request failed: ${response.status}`);
  }
  return response.json() as Promise<T>;
}

export async function fetchAdminAIConfig(token: string): Promise<AdminAIConfig> {
  return request<AdminAIConfig>(token, '/admin/ai-config');
}

export async function updateAdminAIConfig(token: string, payload: UpdateAdminAIConfigPayload): Promise<AdminAIConfig> {
  return request<AdminAIConfig>(token, '/admin/ai-config', {
    method: 'PUT',
    body: JSON.stringify(payload),
  });
}

type AdminRecord = RaRecord & Record<string, unknown>;

function withAdminIds(resource: string, rows: Array<Record<string, unknown>>): AdminRecord[] {
  return rows.map((row) => {
    if (row.id) {
      return row as AdminRecord;
    }
    if (resource === 'members') {
      return {
        ...row,
        id: `${String(row.workspaceId)}:${String(row.userId)}`,
      } as AdminRecord;
    }
    return row as AdminRecord;
  });
}

function sortRows<T extends AdminRecord>(rows: T[], field?: string, order?: string) {
  if (!field) {
    return rows;
  }

  return [...rows].sort((left, right) => {
    const leftValue = String((left as Record<string, unknown>)[field] ?? '');
    const rightValue = String((right as Record<string, unknown>)[field] ?? '');
    return order === 'DESC' ? rightValue.localeCompare(leftValue) : leftValue.localeCompare(rightValue);
  });
}

export function createAdminDataProvider(token: string): DataProvider {
  return {
    async getList(resource, params) {
      const path = resourcePath[resource];
      if (!path) {
        throw new Error(`Unsupported admin resource: ${resource}`);
      }

      const response = await request<ListResponse<Record<string, unknown>>>(token, path);
      const rows = withAdminIds(resource, response.items);
      const sorted = sortRows(rows, params.sort?.field, params.sort?.order);
      const page = params.pagination?.page ?? 1;
      const perPage = params.pagination?.perPage ?? 25;
      const start = (page - 1) * perPage;
      const end = start + perPage;

      return {
        data: sorted.slice(start, end) as never,
        total: sorted.length,
      };
    },

    async getOne(resource, params) {
      const path = resourcePath[resource];
      if (!path) {
        throw new Error(`Unsupported admin resource: ${resource}`);
      }

      const response = await request<ListResponse<Record<string, unknown>>>(token, path);
      const item = withAdminIds(resource, response.items).find((row) => row.id === params.id);
      if (!item) {
        throw new Error(`Record not found: ${String(params.id)}`);
      }
      return { data: item as never };
    },

    async create(resource, params) {
      if (resource !== 'mediaPlatforms') {
        throw new Error(`Create is not supported for ${resource}`);
      }

      const data = params.data as CreateMediaPlatformPayload;
      const created = await request<AdminRecord>(token, '/admin/media-platforms', {
        method: 'POST',
        body: JSON.stringify({
          ...data,
          credentialFields: normalizeCredentialFields(data.credentialFields),
        }),
      });
      return { data: created as never };
    },

    async update() {
      throw new Error('Update is not implemented yet');
    },

    async updateMany() {
      throw new Error('Bulk update is not implemented yet');
    },

    async delete() {
      throw new Error('Delete is not implemented yet');
    },

    async deleteMany() {
      throw new Error('Bulk delete is not implemented yet');
    },

    async getMany(resource, params) {
      const response = await this.getList(resource, {
        pagination: { page: 1, perPage: 1000 },
        sort: { field: 'id', order: 'ASC' },
        filter: {},
      });
      const ids = params.ids.map((id: Identifier) => String(id));
      return {
        data: response.data.filter((row) => ids.includes(String(row.id))),
      };
    },

    async getManyReference(resource, params) {
      const response = await this.getList(resource, params);
      return response;
    },
  };
}

function normalizeCredentialFields(value: unknown) {
  if (Array.isArray(value)) {
    return value.map(String).map((item) => item.trim()).filter(Boolean);
  }
  return String(value ?? '')
    .split(/[,，\n]/)
    .map((item) => item.trim())
    .filter(Boolean);
}

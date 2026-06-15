import type { KnowledgeBase, MediaAccount, MediaPlatform, User } from '../types';

export function formatSubscription(user: User | null) {
  if (!user) {
    return '-';
  }
  const tier = user.subscriptionTier === 'vip' ? 'VIP' : 'Free';
  const statusMap: Record<User['subscriptionStatus'], string> = {
    active: '有效',
    inactive: '未激活',
    expired: '已过期',
    canceled: '已取消',
  };
  const status = statusMap[user.subscriptionStatus] ?? user.subscriptionStatus;
  if (user.monthlyTokenBudgetCents > 0) {
    const remaining = Math.max(0, user.monthlyTokenBudgetCents - user.monthlyTokenUsedCents);
    return `${tier} / ${status} / ${formatMoney(remaining, 'USD')} 剩余`;
  }
  return `${tier} / ${status}`;
}

export function formatMoney(cents: number, currency: string) {
  return `${(Number(cents || 0) / 100).toFixed(0)} ${currency || 'USD'}`;
}

export function knowledgeBaseName(items: KnowledgeBase[], id: string) {
  return items.find((item) => item.id === id)?.name ?? id;
}

export function knowledgeBaseNames(items: KnowledgeBase[], ids: string[]) {
  if (ids.length === 0) {
    return '-';
  }
  return ids.map((id) => knowledgeBaseName(items, id)).join(', ');
}

export function uniqueValues(values: string[]) {
  return Array.from(new Set(values.filter(Boolean)));
}

export function platformName(items: MediaPlatform[], id: string) {
  return items.find((item) => item.id === id)?.name ?? id;
}

export function platformType(items: MediaPlatform[], id: string) {
  return items.find((item) => item.id === id)?.type ?? '';
}

export function supportsBrowserLogin(platformTypeValue?: string) {
  return platformTypeValue === 'xiaohongshu';
}

export function loginMethodLabel(value?: string) {
  if (value === 'qr') {
    return '二维码登录';
  }
  if (value === 'phone') {
    return '手机号登录';
  }
  if (value === 'manual' || !value) {
    return '手动授权';
  }
  return value;
}

export function mediaAccountStatusLabel(value: string) {
  if (value === 'connected') {
    return '已连接';
  }
  if (value === 'pending_login') {
    return '待登录';
  }
  if (value === 'qr_waiting') {
    return '等待扫码';
  }
  return '需处理';
}

export function mediaAccountStatusColor(value: string): 'default' | 'success' | 'warning' {
  if (value === 'connected') {
    return 'success';
  }
  if (value === 'pending_login' || value === 'qr_waiting') {
    return 'warning';
  }
  return 'default';
}

export function credentialFieldLabel(value: string) {
  const labels: Record<string, string> = {
    accessToken: '访问令牌',
    appId: 'App ID',
    appSecret: 'App Secret',
    applicationPassword: '应用密码',
    nickname: '昵称',
    phoneNumber: '手机号',
    profileUrl: '主页链接',
    qrLogin: '二维码登录',
    siteUrl: '站点地址',
    username: '用户名',
  };
  return labels[value] ?? value;
}

export function accountName(items: MediaAccount[], id: string) {
  return items.find((item) => item.id === id)?.name ?? id;
}

export function contentName(items: Array<{ id: string; title: string }>, id: string) {
  return items.find((item) => item.id === id)?.title ?? id;
}

export function splitKeywords(value: string) {
  return value
    .split(/[,，;；\n]/)
    .map((item) => item.trim())
    .map((item) => item.replace(/^[-*]\s*/, '').trim())
    .filter(Boolean);
}

export function splitGenerationKeywords(value: string) {
  if (!isMarkdownPrompt(value)) {
    return splitKeywords(value);
  }

  const lines = value.split('\n');
  const items: string[] = [];
  let inCoreThemes = false;
  for (const rawLine of lines) {
    const line = rawLine.trim();
    if (line.startsWith('## ')) {
      inCoreThemes = line.includes('核心主题');
      continue;
    }
    if (inCoreThemes && /^[-*]\s+/.test(line)) {
      items.push(line.replace(/^[-*]\s+/, '').trim());
    }
  }
  return items.length > 0 ? items : splitKeywords(value);
}

export function isMarkdownPrompt(value: string) {
  return /^##\s+生成目标/m.test(value) || /^##\s+核心主题/m.test(value);
}

export function formatDate(value: string) {
  return new Intl.DateTimeFormat('zh-CN', {
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
  }).format(new Date(value));
}

export function defaultScheduleInputValue() {
  const nextHour = new Date();
  nextHour.setHours(nextHour.getHours() + 1, 0, 0, 0);
  const timezoneOffset = nextHour.getTimezoneOffset() * 60000;
  return new Date(nextHour.getTime() - timezoneOffset).toISOString().slice(0, 16);
}

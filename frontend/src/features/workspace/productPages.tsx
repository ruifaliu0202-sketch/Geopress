import type { ReactNode } from 'react';
import { useCallback, useEffect, useState } from 'react';
import {
  Alert,
  Box,
  Button,
  Card,
  CardContent,
  Chip,
  CircularProgress,
  Divider,
  FormControl,
  Grid,
  InputLabel,
  MenuItem,
  Select,
  Stack,
  Tab,
  Tabs,
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TableRow,
  TextField,
  Typography,
} from '@mui/material';
import AddIcon from '@mui/icons-material/Add';
import AutoAwesomeOutlinedIcon from '@mui/icons-material/AutoAwesomeOutlined';
import AutorenewIcon from '@mui/icons-material/Autorenew';
import CheckCircleOutlineIcon from '@mui/icons-material/CheckCircleOutline';
import FactCheckOutlinedIcon from '@mui/icons-material/FactCheckOutlined';
import LibraryBooksOutlinedIcon from '@mui/icons-material/LibraryBooksOutlined';
import LoginOutlinedIcon from '@mui/icons-material/LoginOutlined';
import PlaylistAddCheckOutlinedIcon from '@mui/icons-material/PlaylistAddCheckOutlined';
import PriceCheckOutlinedIcon from '@mui/icons-material/PriceCheckOutlined';
import VisibilityOutlinedIcon from '@mui/icons-material/VisibilityOutlined';
import {
  createAgencyClientRelation,
  createApprovalWorkflow,
  createBrandAsset,
  createBrandGuardrail,
  createCampaign,
  createCampaignCalendarItem,
  createCreatorCampaignBrief,
  createCreatorOrder,
  createCreatorShortlist,
  fetchAgencyClientRelations,
  fetchApprovalTasks,
  fetchApprovalWorkflows,
  fetchBrandAssets,
  fetchBrandGuardrails,
  fetchCampaignCalendarItems,
  fetchCampaignReportSummary,
  fetchCampaigns,
  fetchComplianceChecks,
  fetchContentMetrics,
  fetchCreators,
  fetchCreatorCampaignBriefs,
  fetchCreatorDeliverables,
  fetchCreatorOrders,
  fetchCreatorSettlements,
  fetchCreatorShortlists,
  fetchInstalledSkillPackages,
  fetchMediaAccountMatrix,
  fetchReportPackages,
  fetchSkillPackageMarketplace,
  fetchSkillPackageUsage,
  fetchStrategyRecommendations,
  generateReportPackage,
  installSkillPackage,
  processApprovalTask,
  purchaseSkillPackage,
  requestMediaAccountSync,
  submitComplianceCheck,
} from '../../api';
import { InfoRow, MetricCard, Section } from '../../components/common';
import type {
  AgencyClientRelation,
  ApprovalTask,
  ApprovalWorkflow,
  BrandAsset,
  BrandGuardrail,
  Campaign,
  CampaignCalendarItem,
  CampaignReportSummary,
  ComplianceCheck,
  ContentMetric,
  Creator,
  CreatorCampaignBrief,
  CreatorDeliverable,
  CreatorOrder,
  CreatorSettlement,
  CreatorShortlist,
  InstalledSkillPackage,
  MediaAccountMatrixItem,
  ReportPackage,
  SkillPackageMarketplaceItem,
  SkillPackageUsageMetric,
  StrategyRecommendation,
  WorkspaceData,
} from '../../types';
import { accountName, contentName, formatDate, formatMoney, loginMethodLabel, mediaAccountStatusLabel, platformName, splitKeywords } from '../../utils/formatters';

type ProductPageProps = {
  token: string;
  workspaceId: string;
  data: WorkspaceData;
  onChanged: () => void;
};

type AsyncState<T> = {
  items: T;
  loading: boolean;
  error: string | null;
};

const emptyState = { loading: false, error: null };

function errorMessage(err: unknown, fallback: string) {
  return err instanceof Error ? err.message : fallback;
}

function statusColor(value: string): 'default' | 'primary' | 'info' | 'success' | 'warning' | 'error' {
  if (['active', 'published', 'completed', 'approved', 'succeeded', 'generated', 'connected'].includes(value)) {
    return 'success';
  }
  if (['draft', 'planned', 'pending', 'queued', 'manual_pending', 'submitted'].includes(value)) {
    return 'warning';
  }
  if (['failed', 'rejected', 'canceled', 'archived'].includes(value)) {
    return value === 'failed' || value === 'rejected' ? 'error' : 'default';
  }
  return 'info';
}

function LoadingRow({ label }: { label: string }) {
  return (
    <Stack direction="row" spacing={1.25} alignItems="center" sx={{ minHeight: 72, py: 1 }}>
      <CircularProgress size={20} />
      <Typography color="text.secondary">{label}</Typography>
    </Stack>
  );
}

function EmptyText({ children }: { children: string }) {
  return (
    <Box
      sx={{
        minHeight: 72,
        display: 'flex',
        alignItems: 'center',
        border: '1px dashed',
        borderColor: 'divider',
        borderRadius: 1,
        bgcolor: 'action.hover',
        px: 2,
        py: 1.5,
        width: '100%',
        boxSizing: 'border-box',
      }}
    >
      <Typography color="text.secondary" sx={{ overflowWrap: 'anywhere' }}>
        {children}
      </Typography>
    </Box>
  );
}

function ProductTable({
  children,
  minWidth = 760,
  size = 'medium',
}: {
  children: ReactNode;
  minWidth?: number;
  size?: 'small' | 'medium';
}) {
  return (
    <TableContainer sx={{ width: '100%', overflowX: 'auto', WebkitOverflowScrolling: 'touch' }}>
      <Table
        size={size}
        sx={{
          minWidth,
          '& .MuiTableCell-root': {
            verticalAlign: 'top',
            overflowWrap: 'anywhere',
          },
          '& .MuiTableCell-head': {
            whiteSpace: 'nowrap',
          },
        }}
      >
        {children}
      </Table>
    </TableContainer>
  );
}

const wrappingTextSx = {
  minWidth: 0,
  overflowWrap: 'anywhere',
};

const secondaryTextSx = {
  ...wrappingTextSx,
  display: '-webkit-box',
  WebkitBoxOrient: 'vertical',
  WebkitLineClamp: 2,
  overflow: 'hidden',
} as const;

function commaValue(value: string) {
  return splitKeywords(value);
}

function todayInputValue() {
  return new Date().toISOString().slice(0, 10);
}

function nextMonthInputValue() {
  const value = new Date();
  value.setMonth(value.getMonth() + 1);
  return value.toISOString().slice(0, 10);
}

type MatrixPlatformKey = 'overview' | 'sohu' | 'xiaohongshu' | 'toutiao' | 'netease';
type MatrixAssetPlatformKey = Exclude<MatrixPlatformKey, 'overview'>;

type MatrixPlatformConfig = {
  key: MatrixAssetPlatformKey;
  label: string;
  iconLabel: string;
  description: string;
};

type MatrixAccountView = {
  id: string;
  platformId: string;
  platformKey: MatrixAssetPlatformKey;
  platformName: string;
  name: string;
  externalId: string;
  group: string;
  status: string;
  healthStatus: string;
  dataFreshness: string;
  loginMethod: string;
  followerCount: number;
  contentCount: number;
  readCount: number;
  engagementRate: number;
  metricCount: number;
  lastProfileSyncedAt: string;
  lastMetricsSyncedAt: string;
  lastSyncStatus: string;
  lastSyncMessage: string;
  warnings: string[];
};

type MatrixPublishBackflowView = {
  id: string;
  platformId: string;
  platformKey: MatrixAssetPlatformKey;
  platformName: string;
  accountId: string;
  accountName: string;
  contentTitle: string;
  status: string;
  capturedAt: string;
  readCount: number;
  likeCount: number;
  commentCount: number;
  externalUrl: string;
  sourceLabel: string;
};

type MatrixPlatformSummary = {
  accountCount: number;
  connectedCount: number;
  authIssueCount: number;
  staleCount: number;
  followerCount: number;
  readCount: number;
  publishMetricCount: number;
  issueCount: number;
};

const matrixPlatformConfigs: MatrixPlatformConfig[] = [
  {
    key: 'sohu',
    label: '搜狐号',
    iconLabel: '搜',
    description: '当前优先打通的媒体链路，承接手机号登录、发布任务和数据回流。',
  },
  {
    key: 'xiaohongshu',
    label: '小红书',
    iconLabel: '小',
    description: '纳管账号资源与账号状态，后续按可见数据补齐快照。',
  },
  {
    key: 'toutiao',
    label: '头条号',
    iconLabel: '头',
    description: '维护账号资产、发布任务和单篇内容阅读互动数据。',
  },
  {
    key: 'netease',
    label: '网易号',
    iconLabel: '网',
    description: '维护账号资产、发布任务和单篇内容阅读互动数据。',
  },
];

function compactNumber(value: number) {
  return new Intl.NumberFormat('zh-CN', {
    notation: value >= 10000 ? 'compact' : 'standard',
    maximumFractionDigits: 1,
  }).format(value);
}

function percentValue(value: number) {
  return `${(value * 100).toFixed(2)}%`;
}

function sohuStatusLabel(value: string) {
  const labels: Record<string, string> = {
    connected: '已连接',
    pending_login: '待登录',
    qr_waiting: '等待扫码',
    login_waiting: '登录中',
    expired: '已过期',
    needs_auth: '待授权',
    draft: '草稿',
    healthy: '健康',
    unknown: '未知',
    watching: '观察中',
    needs_authorization: '需授权',
    fresh: '最新',
    stale: '待刷新',
    missing: '无快照',
    published: '已发布',
    succeeded: '成功',
    queued: '排队中',
    running: '执行中',
    retrying: '重试中',
    failed: '失败',
    manual_pending: '待确认',
    done: '完成',
    active: '进行中',
    pending: '待接入',
    blocked: '受阻',
    high: '高',
    medium: '中',
    low: '低',
  };
  return labels[value] ?? value;
}

function sohuStatusColor(value: string): 'default' | 'primary' | 'info' | 'success' | 'warning' | 'error' {
  if (['connected', 'healthy', 'fresh', 'published', 'succeeded', 'done'].includes(value)) {
    return 'success';
  }
  if (['pending_login', 'qr_waiting', 'login_waiting', 'needs_auth', 'needs_authorization', 'stale', 'manual_pending', 'active', 'watching', 'queued', 'running', 'retrying'].includes(value)) {
    return 'warning';
  }
  if (['missing', 'blocked', 'failed', 'expired'].includes(value)) {
    return 'error';
  }
  if (['pending', 'draft'].includes(value)) {
    return 'info';
  }
  return 'default';
}

function platformConfigByKey(platformKey: MatrixAssetPlatformKey) {
  return matrixPlatformConfigs.find((platform) => platform.key === platformKey) ?? matrixPlatformConfigs[0];
}

function matrixDate(value?: string) {
  if (!value) {
    return '-';
  }
  const parsed = new Date(value);
  if (Number.isNaN(parsed.getTime())) {
    return value;
  }
  return formatDate(value);
}

function platformKeyForType(type: string): MatrixAssetPlatformKey | null {
  if (type === 'sohu' || type === 'xiaohongshu' || type === 'toutiao' || type === 'netease') {
    return type;
  }
  return null;
}

function fallbackPlatformKey(index: number): MatrixAssetPlatformKey {
  return matrixPlatformConfigs[index % matrixPlatformConfigs.length]?.key ?? 'sohu';
}

function matrixSummary(accounts: MatrixAccountView[], contentRows: MatrixPublishBackflowView[]): MatrixPlatformSummary {
  return {
    accountCount: accounts.length,
    connectedCount: accounts.filter((account) => account.status === 'connected').length,
    authIssueCount: accounts.filter((account) => account.status !== 'connected').length,
    staleCount: accounts.filter((account) => account.dataFreshness !== 'fresh').length,
    followerCount: accounts.reduce((sum, account) => sum + account.followerCount, 0),
    readCount: contentRows.reduce((sum, item) => sum + item.readCount, 0),
    publishMetricCount: contentRows.filter((item) => item.sourceLabel !== '发布任务').length,
    issueCount: accounts.filter((account) => account.warnings.length > 0 || account.status !== 'connected' || account.dataFreshness !== 'fresh').length,
  };
}

function matrixAccountViews(items: MediaAccountMatrixItem[], data: WorkspaceData): MatrixAccountView[] {
  return items.map((item, index) => {
    const platform = item.platform.id ? item.platform : data.mediaPlatforms.find((candidate) => candidate.id === item.account.platformId);
    const platformKey = platformKeyForType(platform?.type ?? '') ?? fallbackPlatformKey(index);
    const platformConfig = platformConfigByKey(platformKey);
    const latestSnapshot = item.latestSnapshot;
    const accountMetrics = data.publishJobs.filter((job) => job.mediaAccountId === item.account.id);
    return {
      id: item.account.id,
      platformId: item.account.platformId,
      platformKey,
      platformName: platform?.name || platformConfig.label,
      name: item.account.name,
      externalId: item.account.externalId,
      group: item.account.accountGroup || '未分组',
      status: item.account.status,
      healthStatus: item.account.healthStatus || 'unknown',
      dataFreshness: item.dataFreshness || latestSnapshot?.freshnessStatus || 'missing',
      loginMethod: item.account.loginMethod,
      followerCount: latestSnapshot?.followerCount ?? 0,
      contentCount: latestSnapshot?.contentCount ?? accountMetrics.length,
      readCount: 0,
      engagementRate: latestSnapshot?.engagementRate ?? 0,
      metricCount: item.contentMetricCount,
      lastProfileSyncedAt: matrixDate(item.account.lastProfileSyncedAt),
      lastMetricsSyncedAt: matrixDate(item.account.lastMetricsSyncedAt ?? latestSnapshot?.capturedAt),
      lastSyncStatus: item.latestSyncJob?.status || item.account.lastSyncStatus || '',
      lastSyncMessage: item.latestSyncJob?.errorMessage || item.account.lastSyncMessage || '',
      warnings: item.warnings ?? [],
    };
  });
}

function matrixPublishBackflowViews(metrics: ContentMetric[], accounts: MatrixAccountView[], data: WorkspaceData): MatrixPublishBackflowView[] {
  return metrics.map((item, index) => {
    const account = accounts.find((candidate) => candidate.id === item.mediaAccountId);
    const platform = data.mediaPlatforms.find((candidate) => candidate.id === item.platformId);
    const platformKey = account?.platformKey ?? platformKeyForType(platform?.type ?? '') ?? fallbackPlatformKey(index);
    const publishJob = data.publishJobs.find((job) => job.id === item.publishJobId);
    const readCount = item.viewCount || item.impressionCount;
    return {
      id: item.id,
      platformId: item.platformId,
      platformKey,
      platformName: account?.platformName || platform?.name || platformConfigByKey(platformKey).label,
      accountId: item.mediaAccountId,
      accountName: account?.name || accountName(data.mediaAccounts, item.mediaAccountId),
      contentTitle: contentName(data.contents, item.contentId),
      status: publishJob?.status || 'published',
      capturedAt: matrixDate(item.capturedAt || item.metricDate),
      readCount,
      likeCount: item.likeCount,
      commentCount: item.commentCount,
      externalUrl: item.externalUrl || publishJob?.externalUrl || '',
      sourceLabel: item.publishJobId ? '发布任务回流' : '手动指标',
    };
  });
}

function publishJobsWithoutMetrics(data: WorkspaceData, accounts: MatrixAccountView[], metrics: ContentMetric[]): MatrixPublishBackflowView[] {
  const metricJobIds = new Set(metrics.map((item) => item.publishJobId).filter(Boolean));
  return data.publishJobs
    .filter((job) => !metricJobIds.has(job.id))
    .slice(0, 20)
    .map((job, index) => {
      const account = accounts.find((item) => item.id === job.mediaAccountId);
      const platformKey = account?.platformKey ?? fallbackPlatformKey(index);
      return {
        id: `job-${job.id}`,
        platformId: account?.platformId || '',
        platformKey,
        platformName: account?.platformName || platformConfigByKey(platformKey).label,
        accountId: job.mediaAccountId,
        accountName: account?.name || accountName(data.mediaAccounts, job.mediaAccountId),
        contentTitle: contentName(data.contents, job.contentId),
        status: job.status,
        capturedAt: matrixDate(job.scheduledAt),
        readCount: 0,
        likeCount: 0,
        commentCount: 0,
        externalUrl: job.externalUrl,
        sourceLabel: '发布任务',
      };
    });
}

function withAccountReadTotals(accounts: MatrixAccountView[], contentRows: MatrixPublishBackflowView[]) {
  return accounts.map((account) => ({
    ...account,
    readCount: contentRows
      .filter((item) => item.accountId === account.id)
      .reduce((sum, item) => sum + item.readCount, 0),
  }));
}

function PlatformStatusCard({
  platform,
  active,
  summary,
  onSelect,
}: {
  platform: MatrixPlatformConfig;
  active: boolean;
  summary: MatrixPlatformSummary;
  onSelect: () => void;
}) {
  return (
    <Box
      component="button"
      type="button"
      onClick={onSelect}
      sx={{
        width: '100%',
        minHeight: 192,
        textAlign: 'left',
        border: '1px solid',
        borderColor: active ? 'primary.main' : 'divider',
        borderRadius: 1,
        bgcolor: active ? 'action.selected' : 'background.paper',
        p: 2,
        cursor: 'pointer',
        font: 'inherit',
        color: 'inherit',
        transition: 'border-color 120ms ease, background-color 120ms ease',
        '&:hover': {
          borderColor: 'primary.main',
          bgcolor: 'action.hover',
        },
      }}
    >
      <Stack spacing={1.5} sx={{ minWidth: 0 }}>
        <Stack direction="row" spacing={1.25} alignItems="center">
          <Box
            sx={{
              width: 42,
              height: 42,
              borderRadius: 1,
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'center',
              bgcolor: 'primary.main',
              color: 'primary.contrastText',
              fontWeight: 800,
              flexShrink: 0,
            }}
          >
            {platform.iconLabel}
          </Box>
          <Box sx={{ minWidth: 0 }}>
            <Typography fontWeight={800} sx={wrappingTextSx}>
              {platform.label}
            </Typography>
            <Typography variant="body2" color="text.secondary" sx={secondaryTextSx}>
              {platform.description}
            </Typography>
          </Box>
        </Stack>
        <Grid container spacing={1}>
          <Grid size={6}>
            <InfoRow label="账号" value={String(summary.accountCount)} />
          </Grid>
          <Grid size={6}>
            <InfoRow label="已连接" value={String(summary.connectedCount)} />
          </Grid>
          <Grid size={6}>
            <InfoRow label="待授权" value={String(summary.authIssueCount)} />
          </Grid>
          <Grid size={6}>
            <InfoRow label="待处理" value={String(summary.issueCount)} />
          </Grid>
        </Grid>
        <Stack direction="row" spacing={0.75} flexWrap="wrap" useFlexGap>
          <Chip size="small" label={`粉丝 ${compactNumber(summary.followerCount)}`} variant="outlined" />
          <Chip size="small" label={`阅读 ${compactNumber(summary.readCount)}`} variant="outlined" />
        </Stack>
      </Stack>
    </Box>
  );
}

function fallbackMatrixItems(data: WorkspaceData): MediaAccountMatrixItem[] {
  return data.mediaAccounts.map((account) => {
    const platform = data.mediaPlatforms.find((item) => item.id === account.platformId) ?? {
      id: account.platformId,
      name: platformName(data.mediaPlatforms, account.platformId),
      type: '',
      enabled: true,
      supportsArticle: false,
      supportsImage: false,
      supportsScheduling: false,
      credentialFields: [],
    };
    const dataFreshness = account.lastMetricsSyncedAt ? 'stale' : 'missing';
    return {
      account,
      platform,
      contentMetricCount: 0,
      dataFreshness,
      warnings: account.status === 'connected' ? [] : ['media account authorization is not connected'],
    };
  });
}

function PlatformOverviewTab({
  activePlatform,
  accounts,
  contentRows,
  summaries,
  onSelectPlatform,
}: {
  activePlatform: MatrixPlatformKey;
  accounts: MatrixAccountView[];
  contentRows: MatrixPublishBackflowView[];
  summaries: Record<MatrixAssetPlatformKey, MatrixPlatformSummary>;
  onSelectPlatform: (value: MatrixPlatformKey) => void;
}) {
  const totalSummary = matrixSummary(accounts, contentRows);
  const issueAccounts = accounts.filter((account) => account.status !== 'connected' || account.dataFreshness !== 'fresh' || account.warnings.length > 0);
  const recentRows = contentRows.slice(0, 8);

  return (
    <Stack spacing={3}>
      <Grid container spacing={2}>
        <MetricCard label="矩阵账号" value={totalSummary.accountCount} helper="个人/工作区纳管账号" />
        <MetricCard label="已连接" value={totalSummary.connectedCount} helper="账号状态为已连接" />
        <MetricCard label="平台总粉丝" value={totalSummary.followerCount} helper={`账号快照汇总，约 ${compactNumber(totalSummary.followerCount)}`} />
        <MetricCard label="单篇数据" value={totalSummary.publishMetricCount} helper="已回流的单篇内容指标" />
        <MetricCard label="待处理" value={totalSummary.issueCount} helper="授权、数据过期或告警" tone={totalSummary.issueCount > 0 ? 'error' : 'primary'} />
      </Grid>

      <Section title="平台状态">
        <Grid container spacing={1.5}>
          {matrixPlatformConfigs.map((platform) => (
            <Grid key={platform.key} size={{ xs: 12, md: 6, xl: 3 }}>
              <PlatformStatusCard platform={platform} active={activePlatform === platform.key} summary={summaries[platform.key]} onSelect={() => onSelectPlatform(platform.key)} />
            </Grid>
          ))}
        </Grid>
      </Section>

      <Grid container spacing={2}>
        <Grid size={{ xs: 12, lg: 5 }}>
          <Section title="待处理账号">
            {issueAccounts.length === 0 ? (
              <EmptyText>暂无待处理账号</EmptyText>
            ) : (
              <Stack spacing={1.25}>
                {issueAccounts.slice(0, 8).map((account) => (
                  <Box key={account.id} sx={{ border: '1px solid', borderColor: 'divider', borderRadius: 1, p: 1.5 }}>
                    <Stack spacing={1}>
                      <Stack direction="row" spacing={0.75} alignItems="center" flexWrap="wrap" useFlexGap>
                        <Chip size="small" label={account.platformName} />
                        <Chip size="small" label={mediaAccountStatusLabel(account.status)} color={sohuStatusColor(account.status)} />
                        <Chip size="small" label={sohuStatusLabel(account.dataFreshness)} color={sohuStatusColor(account.dataFreshness)} variant="outlined" />
                      </Stack>
                      <Typography fontWeight={700} sx={wrappingTextSx}>
                        {account.name}
                      </Typography>
                      <Typography variant="body2" color="text.secondary" sx={secondaryTextSx}>
                        {account.warnings[0] || account.lastSyncMessage || '等待补齐账号快照'}
                      </Typography>
                    </Stack>
                  </Box>
                ))}
              </Stack>
            )}
          </Section>
        </Grid>

        <Grid size={{ xs: 12, lg: 7 }}>
          <Section title="最近发布回流">
            {recentRows.length === 0 ? (
              <EmptyText>暂无发布任务回流数据</EmptyText>
            ) : (
              <ProductTable minWidth={820}>
                <TableHead>
                  <TableRow>
                    <TableCell>内容</TableCell>
                    <TableCell>平台</TableCell>
                    <TableCell>账号</TableCell>
                    <TableCell>状态</TableCell>
                    <TableCell>阅读</TableCell>
                    <TableCell>互动</TableCell>
                  </TableRow>
                </TableHead>
                <TableBody>
                  {recentRows.map((item) => (
                    <TableRow key={item.id} hover>
                      <TableCell sx={{ minWidth: 220 }}>
                        <Typography fontWeight={700} sx={secondaryTextSx}>
                          {item.contentTitle}
                        </Typography>
                        <Typography variant="body2" color="text.secondary">
                          {item.capturedAt} / {item.sourceLabel}
                        </Typography>
                      </TableCell>
                      <TableCell sx={{ minWidth: 110 }}>{item.platformName}</TableCell>
                      <TableCell sx={{ minWidth: 140 }}>{item.accountName}</TableCell>
                      <TableCell>
                        <Chip size="small" label={sohuStatusLabel(item.status)} color={sohuStatusColor(item.status)} />
                      </TableCell>
                      <TableCell>{compactNumber(item.readCount)}</TableCell>
                      <TableCell>{(item.likeCount + item.commentCount).toLocaleString()}</TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </ProductTable>
            )}
          </Section>
        </Grid>
      </Grid>
    </Stack>
  );
}

export function MediaMatrixView({ token, workspaceId, data, onChanged }: ProductPageProps) {
  const [activeTab, setActiveTab] = useState<MatrixPlatformKey>('overview');
  const [selectedAccountId, setSelectedAccountId] = useState('');
  const [groupFilter, setGroupFilter] = useState('all');
  const [statusFilter, setStatusFilter] = useState('all');
  const [matrixState, setMatrixState] = useState<AsyncState<MediaAccountMatrixItem[]>>({ ...emptyState, items: [] });
  const [contentMetricState, setContentMetricState] = useState<AsyncState<ContentMetric[]>>({ ...emptyState, items: [] });
  const [syncingId, setSyncingId] = useState('');
  const [syncError, setSyncError] = useState<string | null>(null);

  const load = useCallback(async () => {
    setMatrixState((current) => ({ ...current, loading: true, error: null }));
    setContentMetricState((current) => ({ ...current, loading: true, error: null }));
    try {
      const [matrixItems, contentMetrics] = await Promise.all([
        fetchMediaAccountMatrix(token, workspaceId),
        fetchContentMetrics(token, workspaceId, { limit: 200 }),
      ]);
      setMatrixState({ items: matrixItems, loading: false, error: null });
      setContentMetricState({ items: contentMetrics, loading: false, error: null });
      setSelectedAccountId((current) => current || matrixItems[0]?.account.id || data.mediaAccounts[0]?.id || '');
    } catch (err) {
      const message = errorMessage(err, '媒体矩阵加载失败');
      setMatrixState((current) => ({ ...current, loading: false, error: message }));
      setContentMetricState((current) => ({ ...current, loading: false, error: message }));
    }
  }, [data.mediaAccounts, token, workspaceId]);

  useEffect(() => {
    void load();
  }, [load]);

  const matrixItems = matrixState.items.length > 0 ? matrixState.items : fallbackMatrixItems(data);
  const baseAccounts = matrixAccountViews(matrixItems, data);
  const metricBackflowRows = matrixPublishBackflowViews(contentMetricState.items, baseAccounts, data);
  const allContentRows = [...metricBackflowRows, ...publishJobsWithoutMetrics(data, baseAccounts, contentMetricState.items)];
  const accounts = withAccountReadTotals(baseAccounts, allContentRows);
  const summaries = matrixPlatformConfigs.reduce(
    (current, platform) => ({
      ...current,
      [platform.key]: matrixSummary(
        accounts.filter((account) => account.platformKey === platform.key),
        allContentRows.filter((item) => item.platformKey === platform.key),
      ),
    }),
    {} as Record<MatrixAssetPlatformKey, MatrixPlatformSummary>,
  );

  const activePlatformKey = activeTab === 'overview' ? 'sohu' : activeTab;
  const activePlatform = platformConfigByKey(activePlatformKey);
  const platformAccounts = accounts.filter((account) => account.platformKey === activePlatform.key);
  const platformContentRows = allContentRows.filter((item) => item.platformKey === activePlatform.key);
  const activeSummary = matrixSummary(platformAccounts, platformContentRows);
  const accountGroups = Array.from(new Set(accounts.map((account) => account.group).filter(Boolean)));
  const filteredAccounts = platformAccounts.filter((account) => {
    const groupOK = groupFilter === 'all' || account.group === groupFilter;
    const statusOK = statusFilter === 'all' || account.status === statusFilter;
    return groupOK && statusOK;
  });
  const selectedAccount =
    platformAccounts.find((account) => account.id === selectedAccountId) ??
    platformAccounts[0] ??
    accounts.find((account) => account.id === selectedAccountId) ??
    accounts[0];

  const runSync = async (accountId: string, syncType: 'metrics' | 'content_metrics' | 'full' = 'full') => {
    setSyncingId(accountId);
    setSyncError(null);
    try {
      await requestMediaAccountSync(token, workspaceId, accountId, {
        syncType,
        idempotencyKey: `${syncType}:${accountId}:${new Date().toISOString().slice(0, 16)}`,
        requestPayload: { requestedFrom: 'media_matrix_page' },
      });
      await load();
      onChanged();
    } catch (err) {
      setSyncError(errorMessage(err, '同步任务创建失败'));
    } finally {
      setSyncingId('');
    }
  };

  const selectTab = (value: MatrixPlatformKey) => {
    setActiveTab(value);
    if (value !== 'overview') {
      const first = accounts.find((account) => account.platformKey === value);
      if (first) {
        setSelectedAccountId(first.id);
      }
    }
  };

  const renderPlatformTab = () => (
    <Stack spacing={3}>
      <Grid container spacing={2}>
        <MetricCard label={`${activePlatform.label}账号`} value={activeSummary.accountCount} helper="平台账号资产" />
        <MetricCard label="已连接" value={activeSummary.connectedCount} helper="账号授权状态正常" />
        <MetricCard label="平台粉丝" value={activeSummary.followerCount} helper={`账号快照汇总，约 ${compactNumber(activeSummary.followerCount)}`} />
        <MetricCard label="单篇数据" value={activeSummary.publishMetricCount} helper="已回流的单篇内容指标" />
        <MetricCard label="待处理" value={activeSummary.issueCount} helper="授权、同步或快照问题" tone={activeSummary.issueCount > 0 ? 'error' : 'primary'} />
      </Grid>

      <Grid container spacing={2}>
        <Grid size={{ xs: 12, xl: activePlatform.key === 'xiaohongshu' ? 12 : 8 }}>
          <Section
            title={`${activePlatform.label}账号资产`}
            action={
              <Stack direction={{ xs: 'column', sm: 'row' }} spacing={1}>
                <FormControl size="small" sx={{ minWidth: 132 }}>
                  <InputLabel>分组</InputLabel>
                  <Select label="分组" value={groupFilter} onChange={(event) => setGroupFilter(String(event.target.value))}>
                    <MenuItem value="all">全部</MenuItem>
                    {accountGroups.map((group) => (
                      <MenuItem key={group} value={group}>
                        {group}
                      </MenuItem>
                    ))}
                  </Select>
                </FormControl>
                <FormControl size="small" sx={{ minWidth: 132 }}>
                  <InputLabel>状态</InputLabel>
                  <Select label="状态" value={statusFilter} onChange={(event) => setStatusFilter(String(event.target.value))}>
                    <MenuItem value="all">全部</MenuItem>
                    <MenuItem value="connected">已连接</MenuItem>
                    <MenuItem value="pending_login">待登录</MenuItem>
                    <MenuItem value="qr_waiting">等待扫码</MenuItem>
                    <MenuItem value="expired">已过期</MenuItem>
                  </Select>
                </FormControl>
              </Stack>
            }
          >
            {matrixState.loading ? (
              <LoadingRow label="正在加载媒体矩阵账号" />
            ) : filteredAccounts.length === 0 ? (
              <EmptyText>当前平台暂无账号资产</EmptyText>
            ) : (
              <ProductTable minWidth={1020}>
                <TableHead>
                  <TableRow>
                    <TableCell>账号</TableCell>
                    <TableCell>平台 / 分组</TableCell>
                    <TableCell>粉丝 / 内容</TableCell>
                    <TableCell>阅读 / 单篇数据</TableCell>
                    <TableCell>状态</TableCell>
                    <TableCell>最近同步</TableCell>
                    <TableCell align="right">操作</TableCell>
                  </TableRow>
                </TableHead>
                <TableBody>
                  {filteredAccounts.map((account) => (
                    <TableRow
                      key={account.id}
                      hover
                      selected={account.id === selectedAccountId}
                      onClick={() => setSelectedAccountId(account.id)}
                      sx={{ cursor: 'pointer' }}
                    >
                    <TableCell sx={{ minWidth: 190 }}>
                      <Stack direction="row" spacing={1} alignItems="flex-start">
                        <Box
                          sx={{
                            width: 32,
                            height: 32,
                            borderRadius: 1,
                            display: 'flex',
                            alignItems: 'center',
                            justifyContent: 'center',
                            bgcolor: 'primary.main',
                            color: 'primary.contrastText',
                            fontWeight: 800,
                            flexShrink: 0,
                          }}
                        >
                          {platformConfigByKey(account.platformKey).iconLabel}
                        </Box>
                        <Box sx={{ minWidth: 0 }}>
                          <Typography fontWeight={800} sx={wrappingTextSx}>
                            {account.name}
                          </Typography>
                          <Typography variant="body2" color="text.secondary" sx={wrappingTextSx}>
                            {account.externalId || account.id}
                          </Typography>
                        </Box>
                      </Stack>
                    </TableCell>
                    <TableCell sx={{ minWidth: 150 }}>
                      <Typography variant="body2" fontWeight={700} sx={wrappingTextSx}>
                        {account.platformName}
                      </Typography>
                      <Typography variant="body2" color="text.secondary" sx={wrappingTextSx}>
                        {account.group}
                      </Typography>
                    </TableCell>
                    <TableCell sx={{ minWidth: 120 }}>
                      <Typography fontWeight={800}>{compactNumber(account.followerCount)}</Typography>
                      <Typography variant="body2" color="text.secondary">
                        {account.contentCount} 篇内容
                      </Typography>
                    </TableCell>
                    <TableCell sx={{ minWidth: 130 }}>
                      <Typography fontWeight={800}>{compactNumber(account.readCount)}</Typography>
                      <Typography variant="body2" color="text.secondary">
                        {account.metricCount} 条单篇数据
                      </Typography>
                    </TableCell>
                    <TableCell sx={{ minWidth: 128 }}>
                      <Stack spacing={0.75} alignItems="flex-start">
                        <Chip size="small" label={mediaAccountStatusLabel(account.status)} color={sohuStatusColor(account.status)} />
                        <Chip size="small" label={sohuStatusLabel(account.dataFreshness)} color={sohuStatusColor(account.dataFreshness)} variant="outlined" />
                      </Stack>
                    </TableCell>
                    <TableCell sx={{ minWidth: 160 }}>
                      <Typography variant="body2" color="text.secondary" sx={wrappingTextSx}>
                        指标：{account.lastMetricsSyncedAt}
                      </Typography>
                      <Typography variant="body2" color="text.secondary" sx={wrappingTextSx}>
                        主页：{account.lastProfileSyncedAt}
                      </Typography>
                    </TableCell>
                    <TableCell align="right" sx={{ minWidth: 190 }}>
                      <Stack direction="row" spacing={0.75} justifyContent="flex-end" flexWrap="wrap" useFlexGap>
                        {activePlatform.key !== 'xiaohongshu' && (
                          <Button size="small" startIcon={<VisibilityOutlinedIcon />} onClick={() => setSelectedAccountId(account.id)}>
                            详情
                          </Button>
                        )}
                        <Button size="small" startIcon={<AutorenewIcon />} onClick={() => void runSync(account.id)} disabled={syncingId === account.id}>
                          同步
                        </Button>
                      </Stack>
                    </TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </ProductTable>
            )}
          </Section>
        </Grid>

        {activePlatform.key !== 'xiaohongshu' && (
          <Grid size={{ xs: 12, xl: 4 }}>
            <Section title="账号详情">
              {selectedAccount ? (
                <Stack spacing={2}>
                  <Stack direction="row" spacing={1} flexWrap="wrap" useFlexGap>
                    <Chip label={selectedAccount.platformName} color="primary" />
                    <Chip label={selectedAccount.group} color="primary" variant="outlined" />
                    <Chip label={sohuStatusLabel(selectedAccount.healthStatus)} color={sohuStatusColor(selectedAccount.healthStatus)} />
                    <Chip label={loginMethodLabel(selectedAccount.loginMethod)} variant="outlined" />
                  </Stack>
                  <Stack spacing={1}>
                    <InfoRow label="账号" value={`${selectedAccount.name} / ${selectedAccount.externalId || selectedAccount.id}`} />
                    <InfoRow label="粉丝" value={compactNumber(selectedAccount.followerCount)} />
                    <InfoRow label="发布阅读" value={compactNumber(selectedAccount.readCount)} />
                    <InfoRow label="互动率" value={percentValue(selectedAccount.engagementRate)} />
                    <InfoRow label="最近主页同步" value={selectedAccount.lastProfileSyncedAt} />
                    <InfoRow label="最近指标同步" value={selectedAccount.lastMetricsSyncedAt} />
                    <InfoRow label="同步状态" value={selectedAccount.lastSyncStatus ? sohuStatusLabel(selectedAccount.lastSyncStatus) : '-'} />
                  </Stack>
                  {selectedAccount.warnings.length > 0 && (
                    <Alert severity="warning">
                      {selectedAccount.warnings.join('；')}
                    </Alert>
                  )}
                  <Button startIcon={<AutorenewIcon />} variant="contained" onClick={() => void runSync(selectedAccount.id)} disabled={syncingId === selectedAccount.id}>
                    创建数据同步任务
                  </Button>
                </Stack>
              ) : (
                <EmptyText>请选择一个媒体号账号</EmptyText>
              )}
            </Section>
          </Grid>
        )}
      </Grid>

      <Grid container spacing={2}>
        <Grid size={{ xs: 12 }}>
          <Section title="发布任务回流 / 单篇内容数据">
            {contentMetricState.loading ? (
              <LoadingRow label="正在加载发布回流数据" />
            ) : platformContentRows.length === 0 ? (
              <EmptyText>当前平台暂无单篇发布数据</EmptyText>
            ) : (
              <ProductTable minWidth={900}>
                <TableHead>
                  <TableRow>
                    <TableCell>内容</TableCell>
                    <TableCell>账号</TableCell>
                    <TableCell>状态</TableCell>
                    <TableCell>阅读</TableCell>
                    <TableCell>互动</TableCell>
                    <TableCell>采集时间</TableCell>
                    <TableCell>外部链接</TableCell>
                  </TableRow>
                </TableHead>
                <TableBody>
                  {platformContentRows.map((item) => (
                    <TableRow key={item.id} hover>
                      <TableCell sx={{ minWidth: 220 }}>
                        <Typography fontWeight={700} sx={secondaryTextSx}>
                          {item.contentTitle}
                        </Typography>
                        <Typography variant="body2" color="text.secondary">
                          {item.sourceLabel}
                        </Typography>
                      </TableCell>
                      <TableCell sx={{ minWidth: 140 }}>{item.accountName}</TableCell>
                      <TableCell>
                        <Chip size="small" label={sohuStatusLabel(item.status)} color={sohuStatusColor(item.status)} />
                      </TableCell>
                      <TableCell>{compactNumber(item.readCount)}</TableCell>
                      <TableCell>{(item.likeCount + item.commentCount).toLocaleString()}</TableCell>
                      <TableCell>{item.capturedAt}</TableCell>
                      <TableCell sx={{ minWidth: 220 }}>
                        <Typography variant="body2" sx={wrappingTextSx}>
                          {item.externalUrl || '-'}
                        </Typography>
                      </TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </ProductTable>
            )}
          </Section>
        </Grid>
      </Grid>
    </Stack>
  );

  return (
    <Stack spacing={3}>
      {(matrixState.error || contentMetricState.error || syncError) && (
        <Alert severity="warning">
          {matrixState.error || contentMetricState.error || syncError}
        </Alert>
      )}

      <Stack direction={{ xs: 'column', lg: 'row' }} spacing={2} alignItems={{ xs: 'stretch', lg: 'center' }} justifyContent="space-between">
        <Stack spacing={0.75} sx={{ minWidth: 0 }}>
          <Typography variant="h5" fontWeight={800} sx={wrappingTextSx}>
            媒体矩阵
          </Typography>
          <Typography color="text.secondary" sx={wrappingTextSx}>
            最小闭环聚焦账号资源和账号状态、发布任务回流、平台粉丝数量和阅读量。
          </Typography>
        </Stack>
        <Stack direction={{ xs: 'column', sm: 'row' }} spacing={1} flexWrap="wrap" useFlexGap>
          <Button startIcon={<AddIcon />} variant="contained">
            新增账号
          </Button>
          <Button startIcon={<LoginOutlinedIcon />} variant="outlined" disabled={!selectedAccount || selectedAccount.status === 'connected'}>
            账号授权
          </Button>
          <Button startIcon={<AutorenewIcon />} variant="outlined" onClick={() => selectedAccount && void runSync(selectedAccount.id)} disabled={!selectedAccount || syncingId === selectedAccount.id}>
            刷新数据
          </Button>
        </Stack>
      </Stack>

      <Box sx={{ borderBottom: 1, borderColor: 'divider' }}>
        <Tabs
          value={activeTab}
          onChange={(_, value: MatrixPlatformKey) => selectTab(value)}
          variant="scrollable"
          scrollButtons="auto"
          allowScrollButtonsMobile
        >
          <Tab value="overview" label="总览" />
          {matrixPlatformConfigs.map((platform) => (
            <Tab key={platform.key} value={platform.key} label={platform.label} />
          ))}
        </Tabs>
      </Box>

      {activeTab === 'overview' ? (
        <PlatformOverviewTab activePlatform={activeTab} accounts={accounts} contentRows={allContentRows} summaries={summaries} onSelectPlatform={selectTab} />
      ) : (
        renderPlatformTab()
      )}
    </Stack>
  );
}

export function CampaignsView({ token, workspaceId, data }: ProductPageProps) {
  const [state, setState] = useState<AsyncState<Campaign[]>>({ ...emptyState, items: [] });
  const [selectedCampaignId, setSelectedCampaignId] = useState('');
  const [calendarItems, setCalendarItems] = useState<CampaignCalendarItem[]>([]);
  const [report, setReport] = useState<CampaignReportSummary | null>(null);
  const [form, setForm] = useState({
    name: '',
    goal: '',
    products: '',
    targetAudiences: '',
    channels: '小红书',
    budgetCents: '0',
    contentQuota: '8',
  });
  const [calendarForm, setCalendarForm] = useState({ title: '', channel: 'xiaohongshu', mediaAccountId: '' });
  const [submitting, setSubmitting] = useState(false);

  const selectedCampaign = state.items.find((item) => item.id === selectedCampaignId) ?? null;

  const load = useCallback(async () => {
    setState((current) => ({ ...current, loading: true, error: null }));
    try {
      const campaigns = await fetchCampaigns(token, workspaceId);
      setState({ items: campaigns, loading: false, error: null });
      setSelectedCampaignId((current) => current || campaigns[0]?.id || '');
    } catch (err) {
      setState((current) => ({ ...current, loading: false, error: errorMessage(err, '战役加载失败') }));
    }
  }, [token, workspaceId]);

  useEffect(() => {
    void load();
  }, [load]);

  useEffect(() => {
    if (!selectedCampaignId) {
      setCalendarItems([]);
      setReport(null);
      return;
    }
    let mounted = true;
    Promise.all([
      fetchCampaignCalendarItems(token, workspaceId, selectedCampaignId),
      fetchCampaignReportSummary(token, workspaceId, selectedCampaignId),
    ])
      .then(([items, summary]) => {
        if (mounted) {
          setCalendarItems(items);
          setReport(summary);
        }
      })
      .catch((err) => {
        if (mounted) {
          setState((current) => ({ ...current, error: errorMessage(err, '战役明细加载失败') }));
        }
      });
    return () => {
      mounted = false;
    };
  }, [selectedCampaignId, token, workspaceId]);

  const submitCampaign = async () => {
    if (!form.name.trim()) {
      setState((current) => ({ ...current, error: '请填写战役名称' }));
      return;
    }
    setSubmitting(true);
    setState((current) => ({ ...current, error: null }));
    try {
      const created = await createCampaign(token, workspaceId, {
        name: form.name.trim(),
        goal: form.goal.trim(),
        products: commaValue(form.products),
        targetAudiences: commaValue(form.targetAudiences),
        channels: commaValue(form.channels),
        budgetCents: Number(form.budgetCents || 0),
        contentQuota: Number(form.contentQuota || 0),
        status: 'planned',
        mediaAccountIds: data.mediaAccounts.map((account) => account.id).slice(0, 3),
        successMetrics: ['impression', 'engagement', 'conversion'],
      });
      setForm({ name: '', goal: '', products: '', targetAudiences: '', channels: '小红书', budgetCents: '0', contentQuota: '8' });
      setSelectedCampaignId(created.id);
      await load();
    } catch (err) {
      setState((current) => ({ ...current, error: errorMessage(err, '战役创建失败') }));
    } finally {
      setSubmitting(false);
    }
  };

  const submitCalendarItem = async () => {
    if (!selectedCampaignId || !calendarForm.title.trim()) {
      setState((current) => ({ ...current, error: '请选择战役并填写排期标题' }));
      return;
    }
    setSubmitting(true);
    setState((current) => ({ ...current, error: null }));
    try {
      await createCampaignCalendarItem(token, workspaceId, selectedCampaignId, {
        title: calendarForm.title.trim(),
        channel: calendarForm.channel,
        mediaAccountId: calendarForm.mediaAccountId,
        contentType: 'xiaohongshu_long_article',
        status: 'planned',
        approvalRequired: true,
        approvalStatus: 'pending',
      });
      setCalendarForm({ title: '', channel: 'xiaohongshu', mediaAccountId: '' });
      const [items, summary] = await Promise.all([
        fetchCampaignCalendarItems(token, workspaceId, selectedCampaignId),
        fetchCampaignReportSummary(token, workspaceId, selectedCampaignId),
      ]);
      setCalendarItems(items);
      setReport(summary);
    } catch (err) {
      setState((current) => ({ ...current, error: errorMessage(err, '排期创建失败') }));
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <Stack spacing={3}>
      <Grid container spacing={2}>
        <MetricCard label="战役" value={state.items.length} helper="跨账号内容项目" />
        <MetricCard label="排期" value={calendarItems.length} helper="选中战役日历项" />
        <MetricCard label="已发布" value={report?.publishedItemCount ?? 0} helper="选中战役发布产出" />
        <MetricCard label="异常" value={report?.failedItemCount ?? 0} helper="选中战役失败项" tone={(report?.failedItemCount ?? 0) > 0 ? 'error' : 'primary'} />
      </Grid>

      {state.error && <Alert severity="error">{state.error}</Alert>}

      <Grid container spacing={2}>
        <Grid size={{ xs: 12, lg: 4 }}>
          <Section title="新建战役">
            <Stack spacing={1.5}>
              <TextField size="small" label="名称" value={form.name} onChange={(event) => setForm((current) => ({ ...current, name: event.target.value }))} />
              <TextField size="small" label="目标" value={form.goal} onChange={(event) => setForm((current) => ({ ...current, goal: event.target.value }))} multiline minRows={2} />
              <TextField size="small" label="产品" value={form.products} onChange={(event) => setForm((current) => ({ ...current, products: event.target.value }))} placeholder="逗号分隔" />
              <TextField size="small" label="目标受众" value={form.targetAudiences} onChange={(event) => setForm((current) => ({ ...current, targetAudiences: event.target.value }))} placeholder="逗号分隔" />
              <TextField size="small" label="渠道" value={form.channels} onChange={(event) => setForm((current) => ({ ...current, channels: event.target.value }))} />
              <Stack direction={{ xs: 'column', sm: 'row' }} spacing={1}>
                <TextField size="small" label="预算分" value={form.budgetCents} onChange={(event) => setForm((current) => ({ ...current, budgetCents: event.target.value }))} type="number" />
                <TextField size="small" label="内容配额" value={form.contentQuota} onChange={(event) => setForm((current) => ({ ...current, contentQuota: event.target.value }))} type="number" />
              </Stack>
              <Button startIcon={<AddIcon />} variant="contained" onClick={submitCampaign} disabled={submitting}>
                创建战役
              </Button>
            </Stack>
          </Section>
        </Grid>

        <Grid size={{ xs: 12, lg: 8 }}>
          <Section
            title="战役列表"
            action={
              <Button startIcon={<AutorenewIcon />} onClick={load} disabled={state.loading}>
                刷新
              </Button>
            }
          >
            {state.loading ? (
              <LoadingRow label="正在加载战役" />
            ) : state.items.length === 0 ? (
              <EmptyText>暂无战役</EmptyText>
            ) : (
              <ProductTable minWidth={780}>
                <TableHead>
                  <TableRow>
                    <TableCell>名称</TableCell>
                    <TableCell>目标</TableCell>
                    <TableCell>状态</TableCell>
                    <TableCell>渠道</TableCell>
                    <TableCell>预算</TableCell>
                  </TableRow>
                </TableHead>
                <TableBody>
                  {state.items.map((campaign) => (
                    <TableRow
                      key={campaign.id}
                      hover
                      selected={campaign.id === selectedCampaignId}
                      onClick={() => setSelectedCampaignId(campaign.id)}
                      sx={{ cursor: 'pointer' }}
                    >
                      <TableCell sx={{ minWidth: 180 }}>
                        <Typography fontWeight={700} sx={wrappingTextSx}>
                          {campaign.name}
                        </Typography>
                        <Typography variant="body2" color="text.secondary" sx={secondaryTextSx}>
                          {campaign.description || campaign.id}
                        </Typography>
                      </TableCell>
                      <TableCell sx={{ minWidth: 180 }}>{campaign.goal || '-'}</TableCell>
                      <TableCell>
                        <Chip size="small" label={campaign.status} color={statusColor(campaign.status)} />
                      </TableCell>
                      <TableCell sx={{ minWidth: 140 }}>{campaign.channels.join(', ') || '-'}</TableCell>
                      <TableCell>{formatMoney(campaign.budgetCents, campaign.currency)}</TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </ProductTable>
            )}
          </Section>
        </Grid>
      </Grid>

      <Grid container spacing={2}>
        <Grid size={{ xs: 12, lg: 4 }}>
          <Section title="新增排期">
            <Stack spacing={1.5}>
              <Typography color="text.secondary" sx={wrappingTextSx}>
                {selectedCampaign ? selectedCampaign.name : '请选择战役'}
              </Typography>
              <TextField size="small" label="排期标题" value={calendarForm.title} onChange={(event) => setCalendarForm((current) => ({ ...current, title: event.target.value }))} />
              <TextField size="small" label="渠道" value={calendarForm.channel} onChange={(event) => setCalendarForm((current) => ({ ...current, channel: event.target.value }))} />
              <FormControl size="small" fullWidth>
                <InputLabel>媒体账号</InputLabel>
                <Select label="媒体账号" value={calendarForm.mediaAccountId} onChange={(event) => setCalendarForm((current) => ({ ...current, mediaAccountId: String(event.target.value) }))}>
                  <MenuItem value="">暂不指定</MenuItem>
                  {data.mediaAccounts.map((account) => (
                    <MenuItem key={account.id} value={account.id}>
                      {account.name}
                    </MenuItem>
                  ))}
                </Select>
              </FormControl>
              <Button startIcon={<PlaylistAddCheckOutlinedIcon />} variant="contained" onClick={submitCalendarItem} disabled={submitting || !selectedCampaignId}>
                创建排期
              </Button>
            </Stack>
          </Section>
        </Grid>
        <Grid size={{ xs: 12, lg: 8 }}>
          <Section title="战役排期">
            {calendarItems.length === 0 ? (
              <EmptyText>暂无排期</EmptyText>
            ) : (
              <ProductTable minWidth={760}>
                <TableHead>
                  <TableRow>
                    <TableCell>标题</TableCell>
                    <TableCell>账号</TableCell>
                    <TableCell>状态</TableCell>
                    <TableCell>审核</TableCell>
                    <TableCell>发布时间窗</TableCell>
                  </TableRow>
                </TableHead>
                <TableBody>
                  {calendarItems.map((item) => (
                    <TableRow key={item.id} hover>
                      <TableCell sx={{ minWidth: 180 }}>
                        <Typography fontWeight={700} sx={wrappingTextSx}>
                          {item.title}
                        </Typography>
                        <Typography variant="body2" color="text.secondary" sx={secondaryTextSx}>
                          {item.brief || item.contentType}
                        </Typography>
                      </TableCell>
                      <TableCell sx={{ minWidth: 140 }}>{accountName(data.mediaAccounts, item.mediaAccountId)}</TableCell>
                      <TableCell>
                        <Chip size="small" label={item.status} color={statusColor(item.status)} />
                      </TableCell>
                      <TableCell>{item.approvalRequired ? item.approvalStatus || 'required' : '无需审核'}</TableCell>
                      <TableCell>{item.publishWindowStartAt ? formatDate(item.publishWindowStartAt) : '-'}</TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </ProductTable>
            )}
          </Section>
        </Grid>
      </Grid>
    </Stack>
  );
}

export function CreatorsView({ token, workspaceId }: ProductPageProps) {
  const [state, setState] = useState<
    AsyncState<{
      creators: Creator[];
      shortlists: CreatorShortlist[];
      briefs: CreatorCampaignBrief[];
      orders: CreatorOrder[];
      deliverables: CreatorDeliverable[];
      settlements: CreatorSettlement[];
    }>
  >({
    ...emptyState,
    items: { creators: [], shortlists: [], briefs: [], orders: [], deliverables: [], settlements: [] },
  });
  const [selectedCreatorId, setSelectedCreatorId] = useState('');
  const [briefForm, setBriefForm] = useState({ title: '', objective: '', budgetCents: '100000' });
  const [submitting, setSubmitting] = useState(false);

  const load = useCallback(async () => {
    setState((current) => ({ ...current, loading: true, error: null }));
    try {
      const [creators, shortlists, briefs, orders, deliverables, settlements] = await Promise.all([
        fetchCreators(token, workspaceId),
        fetchCreatorShortlists(token, workspaceId),
        fetchCreatorCampaignBriefs(token, workspaceId),
        fetchCreatorOrders(token, workspaceId),
        fetchCreatorDeliverables(token, workspaceId),
        fetchCreatorSettlements(token, workspaceId),
      ]);
      setState({ items: { creators, shortlists, briefs, orders, deliverables, settlements }, loading: false, error: null });
      setSelectedCreatorId((current) => current || creators[0]?.id || '');
    } catch (err) {
      setState((current) => ({ ...current, loading: false, error: errorMessage(err, '达人合作数据加载失败') }));
    }
  }, [token, workspaceId]);

  useEffect(() => {
    void load();
  }, [load]);

  const selectedCreator = state.items.creators.find((item) => item.id === selectedCreatorId) ?? null;

  const addShortlist = async (creator: Creator) => {
    setSubmitting(true);
    setState((current) => ({ ...current, error: null }));
    try {
      await createCreatorShortlist(token, workspaceId, {
        creatorId: creator.id,
        name: '默认候选池',
        fitScore: 80,
        qualificationStatus: 'qualified',
        brandSafetyLevel: creator.brandSafetyLevel || 'medium',
      });
      await load();
    } catch (err) {
      setState((current) => ({ ...current, error: errorMessage(err, '加入候选失败') }));
    } finally {
      setSubmitting(false);
    }
  };

  const createBrief = async () => {
    if (!briefForm.title.trim()) {
      setState((current) => ({ ...current, error: '请填写 brief 标题' }));
      return;
    }
    setSubmitting(true);
    try {
      await createCreatorCampaignBrief(token, workspaceId, {
        title: briefForm.title.trim(),
        objective: briefForm.objective.trim(),
        platformTargets: ['小红书'],
        deliverableRequirements: ['1 篇图文笔记'],
        disclosureRequirements: ['正文需明确品牌合作'],
        prohibitedClaims: ['不得承诺增长效果'],
        authorizationScope: '达人自行发布，品牌不得登录达人账号',
        contentUsageRights: '品牌可在自有渠道二次使用 90 天',
        budgetCents: Number(briefForm.budgetCents || 0),
        currency: 'CNY',
        status: 'active',
      });
      setBriefForm({ title: '', objective: '', budgetCents: '100000' });
      await load();
    } catch (err) {
      setState((current) => ({ ...current, error: errorMessage(err, 'Brief 创建失败') }));
    } finally {
      setSubmitting(false);
    }
  };

  const createOrder = async () => {
    const brief = state.items.briefs[0];
    if (!selectedCreator || !brief) {
      setState((current) => ({ ...current, error: '请先选择达人并创建 brief' }));
      return;
    }
    setSubmitting(true);
    try {
      await createCreatorOrder(token, workspaceId, {
        briefId: brief.id,
        creatorId: selectedCreator.id,
        depositCents: Math.round(brief.budgetCents * 0.3),
        lastMessage: '请按 brief 提交初稿',
      });
      await load();
    } catch (err) {
      setState((current) => ({ ...current, error: errorMessage(err, '订单创建失败') }));
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <Stack spacing={3}>
      <Grid container spacing={2}>
        <MetricCard label="达人库" value={state.items.creators.length} helper="可合作达人资源" />
        <MetricCard label="候选" value={state.items.shortlists.length} helper="已加入工作区候选池" />
        <MetricCard label="订单" value={state.items.orders.length} helper="合作履约记录" />
        <MetricCard label="待结算" value={state.items.settlements.filter((item) => item.status !== 'paid').length} helper="财务待处理" />
      </Grid>
      {state.error && <Alert severity="error">{state.error}</Alert>}

      <Grid container spacing={2}>
        <Grid size={{ xs: 12, lg: 8 }}>
          <Section
            title="达人库"
            action={
              <Button startIcon={<AutorenewIcon />} onClick={load} disabled={state.loading}>
                刷新
              </Button>
            }
          >
            {state.loading ? (
              <LoadingRow label="正在加载达人库" />
            ) : state.items.creators.length === 0 ? (
              <EmptyText>暂无达人数据</EmptyText>
            ) : (
              <ProductTable minWidth={800}>
                <TableHead>
                  <TableRow>
                    <TableCell>达人</TableCell>
                    <TableCell>领域</TableCell>
                    <TableCell>价格</TableCell>
                    <TableCell>状态</TableCell>
                    <TableCell align="right">操作</TableCell>
                  </TableRow>
                </TableHead>
                <TableBody>
                  {state.items.creators.map((creator) => (
                    <TableRow
                      key={creator.id}
                      hover
                      selected={creator.id === selectedCreatorId}
                      onClick={() => setSelectedCreatorId(creator.id)}
                      sx={{ cursor: 'pointer' }}
                    >
                      <TableCell sx={{ minWidth: 220 }}>
                        <Typography fontWeight={700} sx={wrappingTextSx}>
                          {creator.displayName}
                        </Typography>
                        <Typography variant="body2" color="text.secondary" sx={secondaryTextSx}>
                          {creator.bio}
                        </Typography>
                      </TableCell>
                      <TableCell sx={{ minWidth: 140 }}>{creator.verticals.join(', ') || '-'}</TableCell>
                      <TableCell>{formatMoney(creator.basePriceCents, creator.currency)}</TableCell>
                      <TableCell>
                        <Chip size="small" label={`${creator.verificationState}/${creator.availabilityStatus}`} color={statusColor(creator.verificationState)} />
                      </TableCell>
                      <TableCell align="right">
                        <Button
                          size="small"
                          onClick={(event) => {
                            event.stopPropagation();
                            void addShortlist(creator);
                          }}
                          disabled={submitting}
                          sx={{ whiteSpace: 'nowrap' }}
                        >
                          加候选
                        </Button>
                      </TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </ProductTable>
            )}
          </Section>
        </Grid>
        <Grid size={{ xs: 12, lg: 4 }}>
          <Section title="合作动作">
            <Stack spacing={1.5}>
              <InfoRow label="当前达人" value={selectedCreator?.displayName ?? '-'} />
              <TextField size="small" label="Brief 标题" value={briefForm.title} onChange={(event) => setBriefForm((current) => ({ ...current, title: event.target.value }))} />
              <TextField size="small" label="目标" value={briefForm.objective} onChange={(event) => setBriefForm((current) => ({ ...current, objective: event.target.value }))} multiline minRows={2} />
              <TextField size="small" label="预算分" type="number" value={briefForm.budgetCents} onChange={(event) => setBriefForm((current) => ({ ...current, budgetCents: event.target.value }))} />
              <Button startIcon={<LibraryBooksOutlinedIcon />} variant="outlined" onClick={createBrief} disabled={submitting}>
                创建 Brief
              </Button>
              <Button startIcon={<PriceCheckOutlinedIcon />} variant="contained" onClick={createOrder} disabled={submitting || !selectedCreator}>
                下合作单
              </Button>
            </Stack>
          </Section>
        </Grid>
      </Grid>

      <Grid container spacing={2}>
        <Grid size={{ xs: 12, lg: 6 }}>
          <Section title="合作订单">
            {state.items.orders.length === 0 ? (
              <EmptyText>暂无订单</EmptyText>
            ) : (
              <ProductTable minWidth={620}>
                <TableHead>
                  <TableRow>
                    <TableCell>达人</TableCell>
                    <TableCell>状态</TableCell>
                    <TableCell>金额</TableCell>
                    <TableCell>消息</TableCell>
                  </TableRow>
                </TableHead>
                <TableBody>
                  {state.items.orders.map((order) => (
                    <TableRow key={order.id} hover>
                      <TableCell sx={{ minWidth: 160 }}>
                        {state.items.creators.find((creator) => creator.id === order.creatorId)?.displayName ?? order.creatorId}
                      </TableCell>
                      <TableCell>
                        <Chip size="small" label={order.status} color={statusColor(order.status)} />
                      </TableCell>
                      <TableCell>{formatMoney(order.priceCents, order.currency)}</TableCell>
                      <TableCell sx={{ minWidth: 180 }}>{order.lastMessage || '-'}</TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </ProductTable>
            )}
          </Section>
        </Grid>
        <Grid size={{ xs: 12, lg: 6 }}>
          <Section title="交付与结算">
            <Stack spacing={2}>
              <Stack direction="row" spacing={1} flexWrap="wrap" useFlexGap sx={{ minWidth: 0 }}>
                {state.items.deliverables.map((item) => (
                  <Chip key={item.id} label={`${item.title || item.id}: ${item.status}`} color={statusColor(item.status)} />
                ))}
                {state.items.deliverables.length === 0 && <EmptyText>暂无交付物</EmptyText>}
              </Stack>
              <Divider />
              <Stack spacing={1}>
                {state.items.settlements.map((item) => (
                  <InfoRow key={item.id} label={item.orderId} value={`${item.status} / ${formatMoney(item.creatorPayoutCents, item.currency)}`} />
                ))}
                {state.items.settlements.length === 0 && <EmptyText>暂无结算记录</EmptyText>}
              </Stack>
            </Stack>
          </Section>
        </Grid>
      </Grid>
    </Stack>
  );
}

export function SkillPackagesView({ token, workspaceId }: ProductPageProps) {
  const [state, setState] = useState<
    AsyncState<{
      marketplace: SkillPackageMarketplaceItem[];
      installed: InstalledSkillPackage[];
      usage: SkillPackageUsageMetric[];
    }>
  >({ ...emptyState, items: { marketplace: [], installed: [], usage: [] } });
  const [installingId, setInstallingId] = useState('');

  const load = useCallback(async () => {
    setState((current) => ({ ...current, loading: true, error: null }));
    try {
      const [marketplace, installed, usage] = await Promise.all([
        fetchSkillPackageMarketplace(token, workspaceId),
        fetchInstalledSkillPackages(token, workspaceId),
        fetchSkillPackageUsage(token, workspaceId),
      ]);
      setState({ items: { marketplace, installed, usage }, loading: false, error: null });
    } catch (err) {
      setState((current) => ({ ...current, loading: false, error: errorMessage(err, '技能包数据加载失败') }));
    }
  }, [token, workspaceId]);

  useEffect(() => {
    void load();
  }, [load]);

  const install = async (item: SkillPackageMarketplaceItem, paid: boolean) => {
    setInstallingId(item.package.id);
    setState((current) => ({ ...current, error: null }));
    try {
      if (paid) {
        await purchaseSkillPackage(token, workspaceId, item.package.id, { versionId: item.version?.id });
      } else {
        await installSkillPackage(token, workspaceId, item.package.id, { versionId: item.version?.id });
      }
      await load();
    } catch (err) {
      setState((current) => ({ ...current, error: errorMessage(err, '技能包安装失败') }));
    } finally {
      setInstallingId('');
    }
  };

  return (
    <Stack spacing={3}>
      <Grid container spacing={2}>
        <MetricCard label="市场技能包" value={state.items.marketplace.length} helper="平台已发布创作能力" />
        <MetricCard label="已安装" value={state.items.installed.length} helper="当前工作区可用于 AI 生成" />
        <MetricCard label="调用次数" value={state.items.usage.reduce((sum, item) => sum + item.count, 0)} helper="技能包使用记录" />
        <MetricCard label="生成记录" value={state.items.usage.filter((item) => item.metricType === 'generation').length} helper="接入生成链路次数" />
      </Grid>
      {state.error && <Alert severity="error">{state.error}</Alert>}

      <Section
        title="创作技能包市场"
        action={
          <Button startIcon={<AutorenewIcon />} onClick={load} disabled={state.loading}>
            刷新
          </Button>
        }
      >
        {state.loading ? (
          <LoadingRow label="正在加载技能包" />
        ) : state.items.marketplace.length === 0 ? (
          <EmptyText>暂无已发布技能包。请在平台后台创建并发布技能包。</EmptyText>
        ) : (
          <Grid container spacing={2}>
            {state.items.marketplace.map((item) => (
              <Grid key={item.package.id} size={{ xs: 12, md: 6, xl: 4 }}>
                <Card variant="outlined" sx={{ height: '100%' }}>
                  <CardContent sx={{ height: '100%', minHeight: 244 }}>
                    <Stack spacing={1.5} sx={{ height: '100%' }}>
                      <Stack direction={{ xs: 'column', sm: 'row' }} justifyContent="space-between" spacing={1} alignItems={{ xs: 'flex-start', sm: 'center' }}>
                        <Typography variant="h3" sx={wrappingTextSx}>
                          {item.package.name}
                        </Typography>
                        <Chip size="small" label={item.installed ? '已安装' : item.package.category || '技能'} color={item.installed ? 'success' : 'info'} />
                      </Stack>
                      <Typography color="text.secondary" sx={{ ...secondaryTextSx, minHeight: 48 }}>
                        {item.package.description || '-'}
                      </Typography>
                      <Stack direction="row" spacing={1} flexWrap="wrap" useFlexGap>
                        <Chip size="small" label={item.package.targetPlatform || '通用平台'} />
                        <Chip size="small" label={item.package.targetIndustry || '通用行业'} />
                        <Chip size="small" label={formatMoney(item.package.priceCents, item.package.currency)} />
                      </Stack>
                      <Typography variant="body2" color="text.secondary" sx={wrappingTextSx}>
                        版本 {item.version?.version ?? '-'} / 作者 {item.package.authorName || item.package.authorId || '-'}
                      </Typography>
                      <Stack direction={{ xs: 'column', sm: 'row' }} spacing={1} sx={{ mt: 'auto' }}>
                        <Button variant="contained" disabled={item.installed || installingId === item.package.id} onClick={() => install(item, false)}>
                          试用安装
                        </Button>
                        <Button variant="outlined" disabled={item.installed || installingId === item.package.id} onClick={() => install(item, true)}>
                          购买
                        </Button>
                      </Stack>
                    </Stack>
                  </CardContent>
                </Card>
              </Grid>
            ))}
          </Grid>
        )}
      </Section>

      <Section title="已安装技能包">
        {state.items.installed.length === 0 ? (
          <EmptyText>暂无已安装技能包</EmptyText>
        ) : (
          <ProductTable minWidth={720}>
            <TableHead>
              <TableRow>
                <TableCell>技能包</TableCell>
                <TableCell>版本</TableCell>
                <TableCell>来源</TableCell>
                <TableCell>席位</TableCell>
                <TableCell>到期</TableCell>
              </TableRow>
            </TableHead>
            <TableBody>
              {state.items.installed.map((item) => (
                <TableRow key={item.entitlement.id} hover>
                  <TableCell sx={{ minWidth: 180 }}>{item.package?.name ?? item.entitlement.packageId}</TableCell>
                  <TableCell sx={{ minWidth: 140 }}>{item.version?.version ?? item.entitlement.versionId}</TableCell>
                  <TableCell>
                    <Chip size="small" label={item.entitlement.source} color={statusColor(item.entitlement.status)} />
                  </TableCell>
                  <TableCell>{item.entitlement.seats}</TableCell>
                  <TableCell>{item.entitlement.expiresAt ? formatDate(item.entitlement.expiresAt) : '-'}</TableCell>
                </TableRow>
              ))}
            </TableBody>
          </ProductTable>
        )}
      </Section>
    </Stack>
  );
}

export function BrandComplianceView({ token, workspaceId, data }: ProductPageProps) {
  const [state, setState] = useState<
    AsyncState<{
      assets: BrandAsset[];
      guardrails: BrandGuardrail[];
      workflows: ApprovalWorkflow[];
      tasks: ApprovalTask[];
      checks: ComplianceCheck[];
      relations: AgencyClientRelation[];
      reports: ReportPackage[];
      recommendations: StrategyRecommendation[];
    }>
  >({
    ...emptyState,
    items: {
      assets: [],
      guardrails: [],
      workflows: [],
      tasks: [],
      checks: [],
      relations: [],
      reports: [],
      recommendations: [],
    },
  });
  const [assetForm, setAssetForm] = useState({ name: '', type: 'forbidden_phrase', content: '', channels: 'xiaohongshu', tags: 'compliance' });
  const [checkForm, setCheckForm] = useState({ title: '', content: '', channel: 'xiaohongshu' });
  const [submitting, setSubmitting] = useState(false);

  const load = useCallback(async () => {
    setState((current) => ({ ...current, loading: true, error: null }));
    try {
      const [assets, guardrails, workflows, tasks, checks, relations, reports, recommendations] = await Promise.all([
        fetchBrandAssets(token, workspaceId),
        fetchBrandGuardrails(token, workspaceId),
        fetchApprovalWorkflows(token, workspaceId),
        fetchApprovalTasks(token, workspaceId),
        fetchComplianceChecks(token, workspaceId),
        fetchAgencyClientRelations(token, workspaceId),
        fetchReportPackages(token, workspaceId),
        fetchStrategyRecommendations(token, workspaceId),
      ]);
      setState({
        items: { assets, guardrails, workflows, tasks, checks, relations, reports, recommendations },
        loading: false,
        error: null,
      });
    } catch (err) {
      setState((current) => ({ ...current, loading: false, error: errorMessage(err, '品牌合规数据加载失败') }));
    }
  }, [token, workspaceId]);

  useEffect(() => {
    void load();
  }, [load]);

  const createAssetAndGuardrail = async () => {
    if (!assetForm.name.trim()) {
      setState((current) => ({ ...current, error: '请填写品牌资产名称' }));
      return;
    }
    setSubmitting(true);
    try {
      const asset = await createBrandAsset(token, workspaceId, {
        name: assetForm.name.trim(),
        type: assetForm.type,
        content: assetForm.content.trim(),
        channels: commaValue(assetForm.channels),
        tags: commaValue(assetForm.tags),
        source: 'manual',
      });
      if (asset.content.trim()) {
        await createBrandGuardrail(token, workspaceId, {
          assetId: asset.id,
          name: `${asset.name} 守则`,
          category: 'claim_risk',
          channel: asset.channels[0] || 'xiaohongshu',
          severity: 'high',
          rules: [asset.content],
          action: '提交复核',
        });
      }
      setAssetForm({ name: '', type: 'forbidden_phrase', content: '', channels: 'xiaohongshu', tags: 'compliance' });
      await load();
    } catch (err) {
      setState((current) => ({ ...current, error: errorMessage(err, '品牌资产创建失败') }));
    } finally {
      setSubmitting(false);
    }
  };

  const submitCheck = async () => {
    if (!checkForm.title.trim() && !checkForm.content.trim()) {
      setState((current) => ({ ...current, error: '请填写待检查内容' }));
      return;
    }
    setSubmitting(true);
    try {
      await submitComplianceCheck(token, workspaceId, {
        resourceType: 'content',
        channel: checkForm.channel,
        title: checkForm.title,
        content: checkForm.content,
      });
      setCheckForm({ title: '', content: '', channel: 'xiaohongshu' });
      await load();
    } catch (err) {
      setState((current) => ({ ...current, error: errorMessage(err, '合规检查提交失败') }));
    } finally {
      setSubmitting(false);
    }
  };

  const createWorkflow = async () => {
    setSubmitting(true);
    try {
      await createApprovalWorkflow(token, workspaceId, {
        resourceType: 'content',
        resourceId: data.contents[0]?.id || 'manual_resource',
        name: '品牌法务双审',
        status: 'active',
        stages: [
          { name: '品牌审核', approverRole: 'brand_manager', requiredApprovals: 1 },
          { name: '法务审核', approverRole: 'legal', requiredApprovals: 1 },
        ],
      });
      await load();
    } catch (err) {
      setState((current) => ({ ...current, error: errorMessage(err, '审批流创建失败') }));
    } finally {
      setSubmitting(false);
    }
  };

  const approveTask = async (task: ApprovalTask) => {
    setSubmitting(true);
    try {
      await processApprovalTask(token, workspaceId, task.id, { decision: 'approve', comment: '页面快速通过' });
      await load();
    } catch (err) {
      setState((current) => ({ ...current, error: errorMessage(err, '审批任务处理失败') }));
    } finally {
      setSubmitting(false);
    }
  };

  const generateReport = async () => {
    setSubmitting(true);
    try {
      await generateReportPackage(token, workspaceId, {
        name: '经营与合规报告',
        reportType: 'monthly',
        audience: 'workspace_operator',
        periodStart: todayInputValue(),
        periodEnd: nextMonthInputValue(),
        sections: ['content_delivery', 'compliance_risks', 'media_matrix'],
      });
      await load();
    } catch (err) {
      setState((current) => ({ ...current, error: errorMessage(err, '报告生成失败') }));
    } finally {
      setSubmitting(false);
    }
  };

  const createRelation = async () => {
    setSubmitting(true);
    try {
      await createAgencyClientRelation(token, workspaceId, {
        clientWorkspaceId: data.workspaces.find((workspace) => workspace.id !== workspaceId)?.id || workspaceId,
        clientName: '示例客户',
        status: 'active',
        scopes: ['content_review', 'reporting'],
        notes: '用于代理服务交付与月报归档',
      });
      await load();
    } catch (err) {
      setState((current) => ({ ...current, error: errorMessage(err, '客户关系创建失败') }));
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <Stack spacing={3}>
      <Grid container spacing={2}>
        <MetricCard label="品牌资产" value={state.items.assets.length} helper="表达、禁用词和素材规范" />
        <MetricCard label="守则" value={state.items.guardrails.length} helper="合规自动检查规则" />
        <MetricCard label="待审批" value={state.items.tasks.filter((item) => item.status === 'pending').length} helper="品牌/法务任务" />
        <MetricCard label="高风险检查" value={state.items.checks.filter((item) => item.riskLevel === 'high').length} helper="需人工复核" tone={state.items.checks.some((item) => item.riskLevel === 'high') ? 'error' : 'primary'} />
      </Grid>
      {state.error && <Alert severity="error">{state.error}</Alert>}

      <Grid container spacing={2}>
        <Grid size={{ xs: 12, lg: 4 }}>
          <Section title="品牌资产与守则">
            <Stack spacing={1.5}>
              <TextField size="small" label="名称" value={assetForm.name} onChange={(event) => setAssetForm((current) => ({ ...current, name: event.target.value }))} />
              <TextField size="small" label="类型" value={assetForm.type} onChange={(event) => setAssetForm((current) => ({ ...current, type: event.target.value }))} />
              <TextField size="small" label="内容/禁用表达" value={assetForm.content} onChange={(event) => setAssetForm((current) => ({ ...current, content: event.target.value }))} multiline minRows={2} />
              <TextField size="small" label="渠道" value={assetForm.channels} onChange={(event) => setAssetForm((current) => ({ ...current, channels: event.target.value }))} />
              <TextField size="small" label="标签" value={assetForm.tags} onChange={(event) => setAssetForm((current) => ({ ...current, tags: event.target.value }))} />
              <Button startIcon={<FactCheckOutlinedIcon />} variant="contained" onClick={createAssetAndGuardrail} disabled={submitting}>
                创建资产和守则
              </Button>
            </Stack>
          </Section>
        </Grid>
        <Grid size={{ xs: 12, lg: 8 }}>
          <Section
            title="品牌资产列表"
            action={
              <Button startIcon={<AutorenewIcon />} onClick={load} disabled={state.loading}>
                刷新
              </Button>
            }
          >
            {state.loading ? (
              <LoadingRow label="正在加载品牌合规数据" />
            ) : state.items.assets.length === 0 ? (
              <EmptyText>暂无品牌资产</EmptyText>
            ) : (
              <ProductTable minWidth={720}>
                <TableHead>
                  <TableRow>
                    <TableCell>资产</TableCell>
                    <TableCell>渠道</TableCell>
                    <TableCell>标签</TableCell>
                    <TableCell>状态</TableCell>
                  </TableRow>
                </TableHead>
                <TableBody>
                  {state.items.assets.map((asset) => (
                    <TableRow key={asset.id} hover>
                      <TableCell sx={{ minWidth: 220 }}>
                        <Typography fontWeight={700} sx={wrappingTextSx}>
                          {asset.name}
                        </Typography>
                        <Typography variant="body2" color="text.secondary" sx={secondaryTextSx}>
                          {asset.content || asset.description || asset.type}
                        </Typography>
                      </TableCell>
                      <TableCell sx={{ minWidth: 140 }}>{asset.channels.join(', ') || '-'}</TableCell>
                      <TableCell sx={{ minWidth: 140 }}>{asset.tags.join(', ') || '-'}</TableCell>
                      <TableCell>
                        <Chip size="small" label={asset.status} color={statusColor(asset.status)} />
                      </TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </ProductTable>
            )}
          </Section>
        </Grid>
      </Grid>

      <Grid container spacing={2}>
        <Grid size={{ xs: 12, lg: 4 }}>
          <Section title="合规检查">
            <Stack spacing={1.5}>
              <TextField size="small" label="标题" value={checkForm.title} onChange={(event) => setCheckForm((current) => ({ ...current, title: event.target.value }))} />
              <TextField size="small" label="渠道" value={checkForm.channel} onChange={(event) => setCheckForm((current) => ({ ...current, channel: event.target.value }))} />
              <TextField size="small" label="待检查内容" value={checkForm.content} onChange={(event) => setCheckForm((current) => ({ ...current, content: event.target.value }))} multiline minRows={4} />
              <Button startIcon={<CheckCircleOutlineIcon />} variant="contained" onClick={submitCheck} disabled={submitting}>
                提交检查
              </Button>
              <Button startIcon={<PlaylistAddCheckOutlinedIcon />} variant="outlined" onClick={createWorkflow} disabled={submitting}>
                创建审批流
              </Button>
            </Stack>
          </Section>
        </Grid>
        <Grid size={{ xs: 12, lg: 8 }}>
          <Section title="检查与审批">
            <Stack spacing={2}>
              {state.items.checks.length === 0 ? (
                <EmptyText>暂无合规检查</EmptyText>
              ) : (
                <ProductTable minWidth={680}>
                  <TableHead>
                    <TableRow>
                      <TableCell>检查</TableCell>
                      <TableCell>风险</TableCell>
                      <TableCell>发现</TableCell>
                      <TableCell>时间</TableCell>
                    </TableRow>
                  </TableHead>
                  <TableBody>
                    {state.items.checks.map((check) => (
                      <TableRow key={check.id} hover>
                        <TableCell sx={{ minWidth: 220 }}>
                          <Typography fontWeight={700} sx={wrappingTextSx}>
                            {check.summary || check.resourceType}
                          </Typography>
                          <Typography variant="body2" color="text.secondary" sx={wrappingTextSx}>
                            {check.channel}
                          </Typography>
                        </TableCell>
                        <TableCell>
                          <Chip size="small" label={check.riskLevel || check.status} color={check.riskLevel === 'high' ? 'error' : statusColor(check.status)} />
                        </TableCell>
                        <TableCell>{check.findings.length}</TableCell>
                        <TableCell>{formatDate(check.createdAt)}</TableCell>
                      </TableRow>
                    ))}
                  </TableBody>
                </ProductTable>
              )}
              <Divider />
              <Stack spacing={1}>
                {state.items.tasks.map((task) => (
                  <Stack
                    key={task.id}
                    direction={{ xs: 'column', md: 'row' }}
                    spacing={1}
                    alignItems={{ xs: 'stretch', md: 'center' }}
                    justifyContent="space-between"
                    sx={{ border: '1px solid', borderColor: 'divider', borderRadius: 1, p: 1.25, minWidth: 0 }}
                  >
                    <Box sx={{ minWidth: 0 }}>
                      <Typography fontWeight={700} sx={wrappingTextSx}>
                        {task.stageName}
                      </Typography>
                      <Typography variant="body2" color="text.secondary" sx={wrappingTextSx}>
                        {task.resourceType}/{task.resourceId}
                      </Typography>
                    </Box>
                    <Stack direction="row" spacing={1} alignItems="center" justifyContent={{ xs: 'space-between', md: 'flex-end' }}>
                      <Chip size="small" label={task.status} color={statusColor(task.status)} />
                      <Button size="small" disabled={task.status !== 'pending' || submitting} onClick={() => approveTask(task)}>
                        通过
                      </Button>
                    </Stack>
                  </Stack>
                ))}
                {state.items.tasks.length === 0 && <EmptyText>暂无审批任务</EmptyText>}
              </Stack>
            </Stack>
          </Section>
        </Grid>
      </Grid>

      <Grid container spacing={2}>
        <Grid size={{ xs: 12, lg: 6 }}>
          <Section
            title="报告包"
            action={
              <Button startIcon={<AutoAwesomeOutlinedIcon />} variant="contained" onClick={generateReport} disabled={submitting}>
                生成报告
              </Button>
            }
          >
            {state.items.reports.length === 0 ? (
              <EmptyText>暂无报告</EmptyText>
            ) : (
              <Stack spacing={1.25}>
                {state.items.reports.map((report) => (
                  <Card key={report.id} variant="outlined">
                    <CardContent>
                      <Stack spacing={1}>
                        <Stack direction={{ xs: 'column', sm: 'row' }} justifyContent="space-between" spacing={1} alignItems={{ xs: 'flex-start', sm: 'center' }}>
                          <Typography variant="h3" sx={wrappingTextSx}>
                            {report.name}
                          </Typography>
                          <Chip size="small" label={report.status} color={statusColor(report.status)} />
                        </Stack>
                        <Typography color="text.secondary" sx={secondaryTextSx}>
                          {report.summary || report.reportType}
                        </Typography>
                        <Typography variant="body2" color="text.secondary" sx={wrappingTextSx}>
                          {report.sections.join(', ')}
                        </Typography>
                      </Stack>
                    </CardContent>
                  </Card>
                ))}
              </Stack>
            )}
          </Section>
        </Grid>
        <Grid size={{ xs: 12, lg: 6 }}>
          <Section
            title="客户关系与策略建议"
            action={
              <Button startIcon={<AddIcon />} onClick={createRelation} disabled={submitting}>
                创建客户关系
              </Button>
            }
          >
            <Stack spacing={2}>
              <Stack spacing={1}>
                {state.items.relations.map((relation) => (
                  <InfoRow key={relation.id} label={relation.clientName || relation.clientWorkspaceId} value={`${relation.status} / ${relation.scopes.join(', ')}`} />
                ))}
                {state.items.relations.length === 0 && <EmptyText>暂无客户关系</EmptyText>}
              </Stack>
              <Divider />
              <Stack spacing={1.25}>
                {state.items.recommendations.map((item) => (
                  <Card key={item.id} variant="outlined">
                    <CardContent>
                      <Typography variant="h3" sx={wrappingTextSx}>
                        {item.title}
                      </Typography>
                      <Typography color="text.secondary" sx={{ ...secondaryTextSx, mt: 0.75 }}>
                        {item.rationale}
                      </Typography>
                      <Typography variant="body2" sx={{ ...wrappingTextSx, mt: 1 }}>
                        {item.action}
                      </Typography>
                    </CardContent>
                  </Card>
                ))}
                {state.items.recommendations.length === 0 && <EmptyText>暂无策略建议</EmptyText>}
              </Stack>
            </Stack>
          </Section>
        </Grid>
      </Grid>
    </Stack>
  );
}

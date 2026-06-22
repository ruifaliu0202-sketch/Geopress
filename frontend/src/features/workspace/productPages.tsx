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
  Table,
  TableBody,
  TableCell,
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
import PlaylistAddCheckOutlinedIcon from '@mui/icons-material/PlaylistAddCheckOutlined';
import PriceCheckOutlinedIcon from '@mui/icons-material/PriceCheckOutlined';
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
import { accountName, formatDate, formatMoney, splitKeywords } from '../../utils/formatters';

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
    <Stack direction="row" spacing={1.25} alignItems="center">
      <CircularProgress size={20} />
      <Typography color="text.secondary">{label}</Typography>
    </Stack>
  );
}

function EmptyText({ children }: { children: string }) {
  return <Typography color="text.secondary">{children}</Typography>;
}

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

export function MediaMatrixView({ token, workspaceId, data }: ProductPageProps) {
  const [state, setState] = useState<AsyncState<{ matrix: MediaAccountMatrixItem[]; contentMetrics: ContentMetric[] }>>({
    ...emptyState,
    items: { matrix: [], contentMetrics: [] },
  });
  const [syncingAccountId, setSyncingAccountId] = useState('');

  const load = useCallback(async () => {
    setState((current) => ({ ...current, loading: true, error: null }));
    try {
      const [matrix, contentMetrics] = await Promise.all([
        fetchMediaAccountMatrix(token, workspaceId),
        fetchContentMetrics(token, workspaceId, { limit: 20 }),
      ]);
      setState({ items: { matrix, contentMetrics }, loading: false, error: null });
    } catch (err) {
      setState((current) => ({ ...current, loading: false, error: errorMessage(err, '媒体矩阵加载失败') }));
    }
  }, [token, workspaceId]);

  useEffect(() => {
    void load();
  }, [load]);

  const requestSync = async (accountId: string) => {
    setSyncingAccountId(accountId);
    setState((current) => ({ ...current, error: null }));
    try {
      await requestMediaAccountSync(token, workspaceId, accountId, { syncType: 'full' });
      await load();
    } catch (err) {
      setState((current) => ({ ...current, error: errorMessage(err, '同步任务创建失败') }));
    } finally {
      setSyncingAccountId('');
    }
  };

  const connectedCount = state.items.matrix.filter((item) => item.account.status === 'connected').length;
  const staleCount = state.items.matrix.filter((item) => item.dataFreshness !== 'fresh').length;

  return (
    <Stack spacing={3}>
      <Grid container spacing={2}>
        <MetricCard label="矩阵账号" value={state.items.matrix.length} helper="纳入运营视图的媒体号" />
        <MetricCard label="已连接" value={connectedCount} helper="可执行登录态任务" />
        <MetricCard label="待刷新" value={staleCount} helper="指标新鲜度需关注" tone={staleCount > 0 ? 'error' : 'primary'} />
        <MetricCard label="内容指标" value={state.items.contentMetrics.length} helper="最近采集的内容数据" />
      </Grid>

      <Section
        title="媒体号矩阵"
        action={
          <Button startIcon={<AutorenewIcon />} variant="outlined" onClick={load} disabled={state.loading}>
            刷新
          </Button>
        }
      >
        <Stack spacing={2}>
          {state.error && <Alert severity="error">{state.error}</Alert>}
          {state.loading ? (
            <LoadingRow label="正在加载媒体矩阵" />
          ) : state.items.matrix.length === 0 ? (
            <EmptyText>暂无媒体号矩阵数据</EmptyText>
          ) : (
            <Table>
              <TableHead>
                <TableRow>
                  <TableCell>账号</TableCell>
                  <TableCell>平台</TableCell>
                  <TableCell>定位</TableCell>
                  <TableCell>粉丝 / 互动率</TableCell>
                  <TableCell>健康状态</TableCell>
                  <TableCell>同步状态</TableCell>
                  <TableCell align="right">操作</TableCell>
                </TableRow>
              </TableHead>
              <TableBody>
                {state.items.matrix.map((item) => (
                  <TableRow key={item.account.id} hover>
                    <TableCell>
                      <Typography fontWeight={700}>{item.account.name}</Typography>
                      <Typography variant="body2" color="text.secondary">
                        {item.account.externalId || item.account.id}
                      </Typography>
                    </TableCell>
                    <TableCell>{item.platform.name}</TableCell>
                    <TableCell sx={{ maxWidth: 320 }}>
                      <Typography variant="body2">{item.account.positioning || item.account.persona || '-'}</Typography>
                      <Typography variant="body2" color="text.secondary">
                        {item.account.targetAudience || '-'}
                      </Typography>
                    </TableCell>
                    <TableCell>
                      {item.latestSnapshot ? (
                        <>
                          <Typography fontWeight={700}>{item.latestSnapshot.followerCount.toLocaleString()}</Typography>
                          <Typography variant="body2" color="text.secondary">
                            {(item.latestSnapshot.engagementRate * 100).toFixed(2)}%
                          </Typography>
                        </>
                      ) : (
                        '-'
                      )}
                    </TableCell>
                    <TableCell>
                      <Chip size="small" label={item.account.healthStatus || 'unknown'} color={statusColor(item.account.healthStatus)} />
                    </TableCell>
                    <TableCell>
                      <Stack spacing={0.5}>
                        <Chip size="small" label={item.account.lastSyncStatus || 'never_synced'} color={statusColor(item.account.lastSyncStatus)} />
                        {item.warnings.map((warning) => (
                          <Typography key={warning} variant="body2" color="text.secondary">
                            {warning}
                          </Typography>
                        ))}
                      </Stack>
                    </TableCell>
                    <TableCell align="right">
                      <Button
                        size="small"
                        startIcon={<AutorenewIcon />}
                        disabled={syncingAccountId === item.account.id}
                        onClick={() => requestSync(item.account.id)}
                      >
                        同步
                      </Button>
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          )}
        </Stack>
      </Section>

      <Section title="内容指标回流">
        {state.items.contentMetrics.length === 0 ? (
          <EmptyText>暂无内容指标</EmptyText>
        ) : (
          <Table>
            <TableHead>
              <TableRow>
                <TableCell>内容</TableCell>
                <TableCell>账号</TableCell>
                <TableCell>曝光 / 互动</TableCell>
                <TableCell>日期</TableCell>
                <TableCell>外部链接</TableCell>
              </TableRow>
            </TableHead>
            <TableBody>
              {state.items.contentMetrics.map((item) => (
                <TableRow key={item.id} hover>
                  <TableCell>{item.contentId}</TableCell>
                  <TableCell>{accountName(data.mediaAccounts, item.mediaAccountId)}</TableCell>
                  <TableCell>
                    {item.impressionCount.toLocaleString()} / {item.likeCount + item.commentCount + item.shareCount}
                  </TableCell>
                  <TableCell>{formatDate(item.metricDate)}</TableCell>
                  <TableCell>{item.externalUrl || '-'}</TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        )}
      </Section>
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
              <Stack direction="row" spacing={1}>
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
              <Table>
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
                      <TableCell>
                        <Typography fontWeight={700}>{campaign.name}</Typography>
                        <Typography variant="body2" color="text.secondary">
                          {campaign.description || campaign.id}
                        </Typography>
                      </TableCell>
                      <TableCell>{campaign.goal || '-'}</TableCell>
                      <TableCell>
                        <Chip size="small" label={campaign.status} color={statusColor(campaign.status)} />
                      </TableCell>
                      <TableCell>{campaign.channels.join(', ') || '-'}</TableCell>
                      <TableCell>{formatMoney(campaign.budgetCents, campaign.currency)}</TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
            )}
          </Section>
        </Grid>
      </Grid>

      <Grid container spacing={2}>
        <Grid size={{ xs: 12, lg: 4 }}>
          <Section title="新增排期">
            <Stack spacing={1.5}>
              <Typography color="text.secondary">{selectedCampaign ? selectedCampaign.name : '请选择战役'}</Typography>
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
              <Table>
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
                      <TableCell>
                        <Typography fontWeight={700}>{item.title}</Typography>
                        <Typography variant="body2" color="text.secondary">
                          {item.brief || item.contentType}
                        </Typography>
                      </TableCell>
                      <TableCell>{accountName(data.mediaAccounts, item.mediaAccountId)}</TableCell>
                      <TableCell>
                        <Chip size="small" label={item.status} color={statusColor(item.status)} />
                      </TableCell>
                      <TableCell>{item.approvalRequired ? item.approvalStatus || 'required' : '无需审核'}</TableCell>
                      <TableCell>{item.publishWindowStartAt ? formatDate(item.publishWindowStartAt) : '-'}</TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
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
            ) : (
              <Table>
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
                    <TableRow key={creator.id} hover selected={creator.id === selectedCreatorId} onClick={() => setSelectedCreatorId(creator.id)} sx={{ cursor: 'pointer' }}>
                      <TableCell>
                        <Typography fontWeight={700}>{creator.displayName}</Typography>
                        <Typography variant="body2" color="text.secondary">
                          {creator.bio}
                        </Typography>
                      </TableCell>
                      <TableCell>{creator.verticals.join(', ') || '-'}</TableCell>
                      <TableCell>{formatMoney(creator.basePriceCents, creator.currency)}</TableCell>
                      <TableCell>
                        <Chip size="small" label={`${creator.verificationState}/${creator.availabilityStatus}`} color={statusColor(creator.verificationState)} />
                      </TableCell>
                      <TableCell align="right">
                        <Button size="small" onClick={(event) => { event.stopPropagation(); void addShortlist(creator); }} disabled={submitting}>
                          加候选
                        </Button>
                      </TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
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
              <Table>
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
                      <TableCell>{state.items.creators.find((creator) => creator.id === order.creatorId)?.displayName ?? order.creatorId}</TableCell>
                      <TableCell>
                        <Chip size="small" label={order.status} color={statusColor(order.status)} />
                      </TableCell>
                      <TableCell>{formatMoney(order.priceCents, order.currency)}</TableCell>
                      <TableCell>{order.lastMessage || '-'}</TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
            )}
          </Section>
        </Grid>
        <Grid size={{ xs: 12, lg: 6 }}>
          <Section title="交付与结算">
            <Stack spacing={2}>
              <Stack direction="row" spacing={1} flexWrap="wrap" useFlexGap>
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
                <Card variant="outlined">
                  <CardContent>
                    <Stack spacing={1.5}>
                      <Stack direction="row" justifyContent="space-between" spacing={2}>
                        <Typography variant="h3">{item.package.name}</Typography>
                        <Chip size="small" label={item.installed ? '已安装' : item.package.category || '技能'} color={item.installed ? 'success' : 'info'} />
                      </Stack>
                      <Typography color="text.secondary" sx={{ minHeight: 48 }}>
                        {item.package.description || '-'}
                      </Typography>
                      <Stack direction="row" spacing={1} flexWrap="wrap" useFlexGap>
                        <Chip size="small" label={item.package.targetPlatform || '通用平台'} />
                        <Chip size="small" label={item.package.targetIndustry || '通用行业'} />
                        <Chip size="small" label={formatMoney(item.package.priceCents, item.package.currency)} />
                      </Stack>
                      <Typography variant="body2" color="text.secondary">
                        版本 {item.version?.version ?? '-'} / 作者 {item.package.authorName || item.package.authorId || '-'}
                      </Typography>
                      <Stack direction="row" spacing={1}>
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
          <Table>
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
                  <TableCell>{item.package?.name ?? item.entitlement.packageId}</TableCell>
                  <TableCell>{item.version?.version ?? item.entitlement.versionId}</TableCell>
                  <TableCell>
                    <Chip size="small" label={item.entitlement.source} color={statusColor(item.entitlement.status)} />
                  </TableCell>
                  <TableCell>{item.entitlement.seats}</TableCell>
                  <TableCell>{item.entitlement.expiresAt ? formatDate(item.entitlement.expiresAt) : '-'}</TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
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
              <Table>
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
                      <TableCell>
                        <Typography fontWeight={700}>{asset.name}</Typography>
                        <Typography variant="body2" color="text.secondary">
                          {asset.content || asset.description || asset.type}
                        </Typography>
                      </TableCell>
                      <TableCell>{asset.channels.join(', ') || '-'}</TableCell>
                      <TableCell>{asset.tags.join(', ') || '-'}</TableCell>
                      <TableCell>
                        <Chip size="small" label={asset.status} color={statusColor(asset.status)} />
                      </TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
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
              <Table>
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
                      <TableCell>
                        <Typography fontWeight={700}>{check.summary || check.resourceType}</Typography>
                        <Typography variant="body2" color="text.secondary">
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
              </Table>
              <Divider />
              <Stack spacing={1}>
                {state.items.tasks.map((task) => (
                  <Stack key={task.id} direction={{ xs: 'column', md: 'row' }} spacing={1} alignItems={{ xs: 'stretch', md: 'center' }} justifyContent="space-between">
                    <Box>
                      <Typography fontWeight={700}>{task.stageName}</Typography>
                      <Typography variant="body2" color="text.secondary">
                        {task.resourceType}/{task.resourceId}
                      </Typography>
                    </Box>
                    <Stack direction="row" spacing={1} alignItems="center">
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
                        <Stack direction="row" justifyContent="space-between">
                          <Typography variant="h3">{report.name}</Typography>
                          <Chip size="small" label={report.status} color={statusColor(report.status)} />
                        </Stack>
                        <Typography color="text.secondary">{report.summary || report.reportType}</Typography>
                        <Typography variant="body2" color="text.secondary">
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
                      <Typography variant="h3">{item.title}</Typography>
                      <Typography color="text.secondary" sx={{ mt: 0.75 }}>
                        {item.rationale}
                      </Typography>
                      <Typography variant="body2" sx={{ mt: 1 }}>
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

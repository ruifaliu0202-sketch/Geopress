import { type ReactNode, useEffect, useMemo, useRef, useState } from 'react';
import {
  Alert,
  Box,
  Button,
  Card,
  CardContent,
  Checkbox,
  Chip,
  CircularProgress,
  Dialog,
  DialogActions,
  DialogContent,
  DialogTitle,
  Divider,
  FormControl,
  Grid,
  IconButton,
  InputLabel,
  ListItemText,
  MenuItem,
  Paper,
  Select,
  Stack,
  TextField,
  Tooltip,
  Typography,
} from '@mui/material';
import AddIcon from '@mui/icons-material/Add';
import ChevronLeftIcon from '@mui/icons-material/ChevronLeft';
import ChevronRightIcon from '@mui/icons-material/ChevronRight';
import LoginOutlinedIcon from '@mui/icons-material/LoginOutlined';
import PsychologyAltOutlinedIcon from '@mui/icons-material/PsychologyAltOutlined';
import ScheduleOutlinedIcon from '@mui/icons-material/ScheduleOutlined';
import DeleteOutlineIcon from '@mui/icons-material/DeleteOutline';
import type { DialogKey, ViewKey } from '../../appTypes';
import {
  completeOnboarding,
  enhanceKnowledgeAsset,
  fetchKnowledgeAssetChunks,
  fetchKnowledgeAssetTasks,
  fetchSubscriptionPlans,
  login,
  registerUser,
  retryKnowledgeAssetProcessing,
  trashKnowledgeAsset,
  trashKnowledgeBase,
  updateKnowledgeAssetBases,
} from '../../api';
import { InfoRow, MetricCard, Section, VIPFeatureButton } from '../../components/common';
import { selectedSurfaceSx } from '../../components/surfaceStyles';
import {
  ContentTable,
  JobsTable,
  KnowledgeAssetsTable,
  MediaAccountsTable,
  MediaPlatformTable,
  SchedulesTable,
} from '../../components/dataTables';
import {
  BrandComplianceView,
  CampaignsView,
  CreatorsView,
  MediaMatrixView,
  SkillPackagesView,
} from './productPages';
import type {
  Content,
  KnowledgeAsset,
  KnowledgeBase,
  KnowledgeChunk,
  KnowledgeProcessingTask,
  SubscriptionPlan,
  User,
  Workspace,
  WorkspaceData,
} from '../../types';
import {
  formatDate,
  formatMoney,
  formatSubscription,
  knowledgeBaseName,
  knowledgeBaseNames,
} from '../../utils/formatters';

const onboardingIndustries = ['本地生活', 'B2B SaaS', '教育培训', '美业医美', '电商零售', '企业服务', '个人创作者', '其他'];
const onboardingTones = ['专业', '清晰', '克制', '亲和', '犀利', '种草感', '可信', '实用'];
const knowledgePackageWidth = 280;
const knowledgePackageGap = 12;
type StatusChipColor = 'default' | 'info' | 'warning' | 'success' | 'error';
type StatusChipOption = { label: string; color: StatusChipColor };
type KnowledgeAssetDetailTarget = 'detail' | 'tips' | 'chunks';

const knowledgeAssetStatusMap: Record<string, StatusChipOption> = {
  pending: { label: '待处理', color: 'warning' },
  processing: { label: '处理中', color: 'info' },
  ready: { label: '可用', color: 'success' },
  failed: { label: '失败', color: 'error' },
  archived: { label: '已归档', color: 'default' },
};

const aiEnhancementStatusMap: Record<string, StatusChipOption> = {
  disabled: { label: '未启用', color: 'default' },
  pending: { label: '待增强', color: 'warning' },
  processing: { label: '增强中', color: 'info' },
  succeeded: { label: '增强成功', color: 'success' },
  failed: { label: '增强失败', color: 'error' },
  skipped: { label: '已跳过', color: 'default' },
};

const knowledgeTaskTypeMap: Record<string, string> = {
  extract: '内容提取',
  extract_retry: '重新提取',
  ai_enhance: 'AI 增强',
};

const knowledgeTaskStatusMap: Record<string, StatusChipOption> = {
  pending: { label: '待处理', color: 'warning' },
  queued: { label: '排队中', color: 'warning' },
  running: { label: '处理中', color: 'info' },
  succeeded: { label: '成功', color: 'success' },
  failed: { label: '失败', color: 'error' },
  canceled: { label: '已取消', color: 'default' },
};

const embeddingStatusMap: Record<string, StatusChipOption> = {
  pending: { label: '待向量化', color: 'warning' },
  processing: { label: '向量化中', color: 'info' },
  ready: { label: '向量可用', color: 'success' },
  failed: { label: '向量失败', color: 'error' },
  skipped: { label: '已跳过', color: 'default' },
};

function statusChipOption(status: string | undefined, map: Record<string, StatusChipOption>, fallback = '未知') {
  const key = status?.trim() ?? '';
  return map[key] ?? { label: key || fallback, color: 'default' as const };
}

function taskTypeLabel(taskType: string | undefined) {
  const key = taskType?.trim() ?? '';
  return knowledgeTaskTypeMap[key] ?? (key || '处理任务');
}

function StatusChip({ option }: { option: StatusChipOption }) {
  return <Chip size="small" label={option.label} color={option.color} variant={option.color === 'default' ? 'outlined' : 'filled'} />;
}

function DetailInfoRow({ label, children }: { label: string; children: ReactNode }) {
  return (
    <Stack
      direction={{ xs: 'column', sm: 'row' }}
      justifyContent="space-between"
      alignItems={{ xs: 'flex-start', sm: 'center' }}
      spacing={{ xs: 0.5, sm: 2 }}
      sx={{ py: 1.25, minWidth: 0 }}
    >
      <Typography color="text.secondary" sx={{ flexShrink: 0 }}>
        {label}
      </Typography>
      <Box sx={{ textAlign: { xs: 'left', sm: 'right' }, overflowWrap: 'anywhere', minWidth: 0 }}>{children}</Box>
    </Stack>
  );
}

export function LoginView({ onLogin }: { onLogin: (result: Awaited<ReturnType<typeof login>>) => void }) {
  const [mode, setMode] = useState<'login' | 'register'>('login');
  const [name, setName] = useState('');
  const [email, setEmail] = useState('demo@geopress.local');
  const [password, setPassword] = useState('demo');
  const [workspaceName, setWorkspaceName] = useState('');
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const isRegister = mode === 'register';

  const handleSubmit = async () => {
    setSubmitting(true);
    setError(null);
    try {
      const result = isRegister
        ? await registerUser({
            name,
            email,
            password,
            workspaceName: workspaceName || undefined,
          })
        : await login(email, password);
      onLogin(result);
    } catch (err) {
      setError(err instanceof Error ? err.message : isRegister ? '注册失败' : '登录失败');
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <Box sx={{ minHeight: '100vh', display: 'grid', placeItems: 'center', bgcolor: 'background.default', p: 2 }}>
      <Paper sx={{ width: '100%', maxWidth: 420, p: 3, border: '1px solid', borderColor: 'divider' }} elevation={0}>
        <Stack spacing={2.5}>
          <Box>
            <Typography variant="h1">Geopress</Typography>
            <Typography color="text.secondary" sx={{ mt: 1 }}>
              {isRegister ? '注册后会自动创建个人工作区。' : '登录后进入个人或公司工作区。'}
            </Typography>
          </Box>
          {error && <Alert severity="error">{error}</Alert>}
          {isRegister && <TextField label="姓名" value={name} onChange={(event) => setName(event.target.value)} fullWidth />}
          <TextField label="邮箱" value={email} onChange={(event) => setEmail(event.target.value)} fullWidth />
          <TextField
            label="密码"
            type="password"
            value={password}
            onChange={(event) => setPassword(event.target.value)}
            fullWidth
          />
          {isRegister && (
            <TextField
              label="工作区名称"
              value={workspaceName}
              onChange={(event) => setWorkspaceName(event.target.value)}
              placeholder="默认使用姓名创建个人工作区"
              fullWidth
            />
          )}
          <Button
            startIcon={isRegister ? <AddIcon /> : <LoginOutlinedIcon />}
            variant="contained"
            onClick={handleSubmit}
            disabled={submitting}
          >
            {isRegister ? '注册并进入' : '登录'}
          </Button>
          <Button
            variant="text"
            onClick={() => {
              setMode(isRegister ? 'login' : 'register');
              setError(null);
              if (!isRegister) {
                setEmail('');
                setPassword('');
              } else {
                setEmail('demo@geopress.local');
                setPassword('demo');
              }
            }}
          >
            {isRegister ? '已有账号，去登录' : '注册新账号'}
          </Button>
          <Typography variant="body2" color="text.secondary">
            Demo 账号：demo@geopress.local 或 growth@geopress.local，密码 demo。
          </Typography>
        </Stack>
      </Paper>
    </Box>
  );
}

export function OnboardingView({
  token,
  workspaceId,
  user,
  onComplete,
}: {
  token: string;
  workspaceId: string;
  user: User;
  onComplete: (result: Awaited<ReturnType<typeof completeOnboarding>>) => void;
}) {
  const [step, setStep] = useState(0);
  const [industry, setIndustry] = useState('');
  const [tones, setTones] = useState<string[]>([]);
  const [plans, setPlans] = useState<SubscriptionPlan[]>([]);
  const [selectedPlanId, setSelectedPlanId] = useState('vip');
  const [loadingPlans, setLoadingPlans] = useState(false);
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    let mounted = true;
    setLoadingPlans(true);
    fetchSubscriptionPlans(token, workspaceId)
      .then((items) => {
        if (!mounted) {
          return;
        }
        setPlans(items);
        const vip = items.find((item) => item.id === 'vip');
        setSelectedPlanId(vip?.id ?? items[0]?.id ?? 'free');
      })
      .catch((err: unknown) => {
        if (mounted) {
          setError(err instanceof Error ? err.message : '套餐加载失败');
        }
      })
      .finally(() => {
        if (mounted) {
          setLoadingPlans(false);
        }
      });
    return () => {
      mounted = false;
    };
  }, [token, workspaceId]);

  const toggleTone = (tone: string) => {
    setTones((current) => {
      if (current.includes(tone)) {
        return current.filter((item) => item !== tone);
      }
      return [...current, tone];
    });
  };

  const canContinue = step === 0 ? industry !== '' : step === 1 ? tones.length > 0 : true;
  const submit = async (skipSubscription = false) => {
    if (!industry || tones.length === 0) {
      setError('请选择行业和语气');
      return;
    }
    setSubmitting(true);
    setError(null);
    try {
      const result = await completeOnboarding(token, workspaceId, {
        workspaceId,
        industry,
        tones,
        subscriptionPlanId: selectedPlanId,
        skipSubscription,
      });
      onComplete(result);
    } catch (err) {
      setError(err instanceof Error ? err.message : '保存失败');
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <Box sx={{ minHeight: '100vh', bgcolor: 'background.default', p: { xs: 2, md: 4 } }}>
      <Box sx={{ maxWidth: 900, mx: 'auto' }}>
        <Paper sx={{ p: { xs: 2.5, md: 4 }, border: '1px solid', borderColor: 'divider' }} elevation={0}>
          <Stack spacing={3}>
            <Box>
              <Typography variant="h1">欢迎，{user.name}</Typography>
              <Typography color="text.secondary" sx={{ mt: 1 }}>
                {step + 1} / 3
              </Typography>
            </Box>
            {error && <Alert severity="error">{error}</Alert>}

            {step === 0 && (
              <Stack spacing={2}>
                <Typography variant="h2">选择行业</Typography>
                <Grid container spacing={1.25}>
                  {onboardingIndustries.map((item) => (
                    <Grid key={item} size={{ xs: 6, sm: 4 }}>
                      <Button
                        variant={industry === item ? 'contained' : 'outlined'}
                        onClick={() => setIndustry(item)}
                        fullWidth
                        sx={{ minHeight: 44 }}
                      >
                        {item}
                      </Button>
                    </Grid>
                  ))}
                </Grid>
              </Stack>
            )}

            {step === 1 && (
              <Stack spacing={2}>
                <Typography variant="h2">选择语气</Typography>
                <Stack direction="row" spacing={1} flexWrap="wrap" useFlexGap>
                  {onboardingTones.map((tone) => (
                    <Chip
                      key={tone}
                      label={tone}
                      color={tones.includes(tone) ? 'primary' : 'default'}
                      variant={tones.includes(tone) ? 'filled' : 'outlined'}
                      onClick={() => toggleTone(tone)}
                      sx={{ minHeight: 36, borderRadius: 1 }}
                    />
                  ))}
                </Stack>
              </Stack>
            )}

            {step === 2 && (
              <Stack spacing={2}>
                <Typography variant="h2">选择订阅</Typography>
                {loadingPlans ? (
                  <Stack direction="row" spacing={1.5} alignItems="center">
                    <CircularProgress size={22} />
                    <Typography color="text.secondary">正在读取订阅计划</Typography>
                  </Stack>
                ) : (
                  <Grid container spacing={1.5}>
                    {plans.map((plan) => (
                      <Grid key={plan.id} size={{ xs: 12, sm: 6 }}>
                        <Paper
                          elevation={0}
                          onClick={() => setSelectedPlanId(plan.id)}
                          sx={{
                            p: 2,
                            cursor: 'pointer',
                            border: '1px solid',
                            borderColor: selectedPlanId === plan.id ? 'primary.main' : 'divider',
                            bgcolor: selectedPlanId === plan.id ? 'action.selected' : 'background.paper',
                          }}
                        >
                          <Stack spacing={1}>
                            <Stack direction="row" alignItems="center" justifyContent="space-between">
                              <Typography variant="h2">{plan.name}</Typography>
                              <Chip size="small" label={formatMoney(plan.priceCents, plan.currency)} color={plan.id === 'vip' ? 'primary' : 'default'} />
                            </Stack>
                            <Typography color="text.secondary">
                              {plan.monthlyTokenBudgetCents > 0
                                ? `${formatMoney(plan.monthlyTokenBudgetCents, plan.currency)} AI Token 额度`
                                : '基础试用额度'}
                            </Typography>
                          </Stack>
                        </Paper>
                      </Grid>
                    ))}
                  </Grid>
                )}
              </Stack>
            )}

            <Stack direction={{ xs: 'column', sm: 'row' }} spacing={1} justifyContent="space-between">
              <Button disabled={step === 0 || submitting} onClick={() => setStep((value) => Math.max(0, value - 1))}>
                上一步
              </Button>
              <Stack direction={{ xs: 'column', sm: 'row' }} spacing={1}>
                {step === 2 && (
                  <Button variant="outlined" onClick={() => submit(true)} disabled={submitting}>
                    跳过
                  </Button>
                )}
                {step < 2 ? (
                  <Button variant="contained" onClick={() => setStep((value) => value + 1)} disabled={!canContinue || submitting}>
                    下一步
                  </Button>
                ) : (
                  <Button variant="contained" onClick={() => submit(false)} disabled={submitting || loadingPlans}>
                    完成
                  </Button>
                )}
              </Stack>
            </Stack>
          </Stack>
        </Paper>
      </Box>
    </Box>
  );
}

export function WorkspaceWorkbenchPanel({
  open,
  currentWorkspace,
  user,
  onToggle,
  onGenerate,
  onSchedule,
  onContent,
}: {
  open: boolean;
  currentWorkspace: Workspace | null;
  user: User | null;
  onToggle: () => void;
  onGenerate: () => void;
  onSchedule: () => void;
  onContent: () => void;
}) {
  return (
    <Paper
      elevation={0}
      sx={{
        width: { xs: '100%', md: open ? 300 : 64 },
        maxWidth: { xs: '100%', md: open ? 300 : 64 },
        flex: { xs: '0 0 auto', md: `0 0 ${open ? 300 : 64}px` },
        boxSizing: 'border-box',
        position: { md: 'sticky' },
        top: { md: 88 },
        p: open ? 2 : 1,
        border: '1px solid',
        borderColor: 'divider',
        borderRadius: 2,
        overflow: 'hidden',
        transition: (theme) =>
          theme.transitions.create(['width', 'flex-basis', 'padding'], {
            duration: theme.transitions.duration.shorter,
          }),
      }}
    >
      <Stack spacing={open ? 2 : 1} alignItems={open ? 'stretch' : 'center'}>
        <Stack direction="row" alignItems="center" justifyContent={open ? 'space-between' : 'center'} spacing={1}>
          {open && (
            <Box sx={{ minWidth: 0 }}>
              <Typography variant="h2">工作区工作台</Typography>
              <Typography color="text.secondary" sx={{ mt: 0.75 }}>
                {currentWorkspace ? currentWorkspace.name : '正在读取工作区信息'}
              </Typography>
            </Box>
          )}
          <Tooltip title={open ? '折叠工作台' : '展开工作台'}>
            <IconButton size="small" onClick={onToggle} aria-label={open ? '折叠工作台' : '展开工作台'}>
              {open ? <ChevronLeftIcon /> : <ChevronRightIcon />}
            </IconButton>
          </Tooltip>
        </Stack>

        {open ? (
          <>
            <Divider />
            <Stack spacing={1}>
              <InfoRow label="行业" value={currentWorkspace?.industry ?? '-'} />
              <InfoRow label="语气" value={currentWorkspace?.tone ?? '-'} />
              <InfoRow label="账号订阅" value={formatSubscription(user)} />
            </Stack>
            <Stack spacing={1}>
              <Button
                startIcon={<PsychologyAltOutlinedIcon />}
                variant="outlined"
                onClick={onGenerate}
                fullWidth
                data-tour-id="workbench-generate"
              >
                关键词生成
              </Button>
              <Button startIcon={<ScheduleOutlinedIcon />} variant="outlined" onClick={onSchedule} fullWidth data-tour-id="workbench-schedule">
                新建计划
              </Button>
              <Button startIcon={<AddIcon />} variant="contained" onClick={onContent} fullWidth data-tour-id="workbench-content">
                新建内容
              </Button>
            </Stack>
          </>
        ) : (
          <Stack spacing={1} alignItems="center">
            <Tooltip title="关键词生成">
              <IconButton size="small" onClick={onGenerate} aria-label="关键词生成" data-tour-id="workbench-generate">
                <PsychologyAltOutlinedIcon fontSize="small" />
              </IconButton>
            </Tooltip>
            <Tooltip title="新建计划">
              <IconButton size="small" onClick={onSchedule} aria-label="新建计划" data-tour-id="workbench-schedule">
                <ScheduleOutlinedIcon fontSize="small" />
              </IconButton>
            </Tooltip>
            <Tooltip title="新建内容">
              <IconButton size="small" color="primary" onClick={onContent} aria-label="新建内容" data-tour-id="workbench-content">
                <AddIcon fontSize="small" />
              </IconButton>
            </Tooltip>
          </Stack>
        )}
      </Stack>
    </Paper>
  );
}

export function ActiveView({
  view,
  token,
  workspaceId,
  workspace,
  currentWorkspace,
  openDialog,
  onChanged,
  onLoginMediaAccount,
  onPreparePublish,
}: {
  view: ViewKey;
  token: string;
  workspaceId: string;
  workspace: WorkspaceData;
  currentWorkspace: Workspace;
  openDialog: (dialog: DialogKey) => void;
  onChanged: () => void;
  onLoginMediaAccount: (accountId: string) => void;
  onPreparePublish: (contentId: string) => void;
}) {
  if (view === 'knowledge') {
    return <KnowledgeView token={token} workspaceId={workspaceId} data={workspace} openDialog={openDialog} onChanged={onChanged} />;
  }
  if (view === 'accounts') {
    return <AccountsView data={workspace} openDialog={openDialog} onLoginMediaAccount={onLoginMediaAccount} />;
  }
  if (view === 'mediaMatrix') {
    return <MediaMatrixView token={token} workspaceId={workspaceId} data={workspace} onChanged={onChanged} />;
  }
  if (view === 'campaigns') {
    return <CampaignsView token={token} workspaceId={workspaceId} data={workspace} onChanged={onChanged} />;
  }
  if (view === 'creators') {
    return <CreatorsView token={token} workspaceId={workspaceId} data={workspace} onChanged={onChanged} />;
  }
  if (view === 'skillPackages') {
    return <SkillPackagesView token={token} workspaceId={workspaceId} data={workspace} onChanged={onChanged} />;
  }
  if (view === 'brandCompliance') {
    return <BrandComplianceView token={token} workspaceId={workspaceId} data={workspace} onChanged={onChanged} />;
  }
  if (view === 'generate') {
    return <GenerateView data={workspace} openDialog={openDialog} />;
  }
  if (view === 'contents') {
    return <ContentsView contents={workspace.contents} openDialog={openDialog} onPreparePublish={onPreparePublish} />;
  }
  if (view === 'schedules') {
    return <SchedulesView data={workspace} openDialog={openDialog} />;
  }
  if (view === 'jobs') {
    return <JobsView data={workspace} />;
  }
  if (view === 'settings') {
    return <SettingsView workspace={currentWorkspace} user={workspace.user} />;
  }
  return <OverviewView data={workspace} openDialog={openDialog} />;
}

function OverviewView({ data, openDialog }: { data: WorkspaceData; openDialog: (dialog: DialogKey) => void }) {
  return (
    <Stack spacing={3}>
      <Grid container spacing={2}>
        <MetricCard label="知识库" value={data.overview.knowledgeBaseCount} helper="生成内容的上下文资产" />
        <MetricCard label="媒体账号" value={data.overview.mediaAccountCount} helper="当前工作区绑定账号" />
        <MetricCard label="内容草稿" value={data.overview.draftCount} helper="等待编辑或排程" />
        <MetricCard label="异常任务" value={data.overview.failedJobs} helper="需要人工处理或重试" tone="error" />
      </Grid>
      <Grid container spacing={2}>
        <Grid size={{ xs: 12, lg: 6 }}>
          <Section
            title="知识资产"
            action={
              <Button
                size="small"
                startIcon={<AddIcon />}
                onClick={() => openDialog('knowledgeAsset')}
                data-tour-id="overview-create-knowledge-asset"
              >
                创建资产
              </Button>
            }
          >
            <KnowledgeAssetsTable assets={data.knowledgeAssets.slice(0, 5)} bases={data.knowledgeBases} />
          </Section>
        </Grid>
        <Grid size={{ xs: 12, lg: 6 }}>
          <Section
            title="发布队列"
            action={<Button size="small" startIcon={<ScheduleOutlinedIcon />} onClick={() => openDialog('schedule')}>新建计划</Button>}
          >
            <JobsTable data={data} dense />
          </Section>
        </Grid>
      </Grid>
    </Stack>
  );
}

function KnowledgeView({
  token,
  workspaceId,
  data,
  openDialog,
  onChanged,
}: {
  token: string;
  workspaceId: string;
  data: WorkspaceData;
  openDialog: (dialog: DialogKey) => void;
  onChanged: () => void;
}) {
  const packageViewportRef = useRef<HTMLDivElement | null>(null);
  const [selectedBaseId, setSelectedBaseId] = useState('');
  const [packagePage, setPackagePage] = useState(0);
  const [visiblePackageCount, setVisiblePackageCount] = useState(1);
  const [assetSearch, setAssetSearch] = useState('');
  const [selectedAsset, setSelectedAsset] = useState<KnowledgeAsset | null>(null);
  const [selectedAssetIds, setSelectedAssetIds] = useState<string[]>([]);
  const [assetChunks, setAssetChunks] = useState<KnowledgeChunk[]>([]);
  const [assetTasks, setAssetTasks] = useState<KnowledgeProcessingTask[]>([]);
  const [assetDetailLoading, setAssetDetailLoading] = useState(false);
  const [assetDetailError, setAssetDetailError] = useState<string | null>(null);
  const [assetDetailTarget, setAssetDetailTarget] = useState<KnowledgeAssetDetailTarget>('detail');
  const [draggingBase, setDraggingBase] = useState<KnowledgeBase | null>(null);
  const [confirmTrashBase, setConfirmTrashBase] = useState<KnowledgeBase | null>(null);
  const [trashSubmitting, setTrashSubmitting] = useState(false);
  const [trashError, setTrashError] = useState<string | null>(null);
  const [removeConfirm, setRemoveConfirm] = useState<{ assets: KnowledgeAsset[] } | null>(null);
  const [removeSubmitting, setRemoveSubmitting] = useState(false);
  const [removeError, setRemoveError] = useState<string | null>(null);
  const [trashAssetConfirm, setTrashAssetConfirm] = useState<KnowledgeAsset | null>(null);
  const [trashAssetSubmitting, setTrashAssetSubmitting] = useState(false);
  const [trashAssetError, setTrashAssetError] = useState<string | null>(null);
  const [retryAssetConfirm, setRetryAssetConfirm] = useState<KnowledgeAsset | null>(null);
  const [retryAssetSubmitting, setRetryAssetSubmitting] = useState(false);
  const [retryAssetError, setRetryAssetError] = useState<string | null>(null);
  const [enhanceAssetConfirm, setEnhanceAssetConfirm] = useState<KnowledgeAsset | null>(null);
  const [enhanceAssetSubmitting, setEnhanceAssetSubmitting] = useState(false);
  const [enhanceAssetError, setEnhanceAssetError] = useState<string | null>(null);
  const totalPackagePages = Math.max(1, Math.ceil(data.knowledgeBases.length / visiblePackageCount));
  const packageStart = Math.min(packagePage, totalPackagePages - 1) * visiblePackageCount;
  const visiblePackages = data.knowledgeBases.slice(packageStart, packageStart + visiblePackageCount);
  const filteredAssets = useMemo(() => {
    const keyword = assetSearch.trim().toLowerCase();
    return data.knowledgeAssets.filter((asset) => {
      if (selectedBaseId && !asset.knowledgeBaseIds.includes(selectedBaseId)) {
        return false;
      }
      if (!keyword) {
        return true;
      }
      return [
        asset.title,
        asset.assetType,
        asset.originalFilename,
        asset.mimeType,
        asset.status,
        asset.aiEnhancementStatus,
        asset.errorMessage,
        knowledgeBaseNames(data.knowledgeBases, asset.knowledgeBaseIds, '未分类'),
      ]
        .join(' ')
        .toLowerCase()
        .includes(keyword);
    });
  }, [assetSearch, data.knowledgeAssets, data.knowledgeBases, selectedBaseId]);

  useEffect(() => {
    setSelectedAssetIds((current) => current.filter((id) => filteredAssets.some((asset) => asset.id === id)));
  }, [filteredAssets]);

  useEffect(() => {
    setSelectedAssetIds([]);
  }, [selectedBaseId]);
  useEffect(() => {
    const viewport = packageViewportRef.current;
    if (!viewport) {
      return undefined;
    }

    const updateVisibleCount = () => {
      const width = viewport.getBoundingClientRect().width;
      setVisiblePackageCount(Math.max(1, Math.floor((width + knowledgePackageGap) / (knowledgePackageWidth + knowledgePackageGap))));
    };

    updateVisibleCount();
    const observer = new ResizeObserver(updateVisibleCount);
    observer.observe(viewport);
    return () => observer.disconnect();
  }, []);

  useEffect(() => {
    setPackagePage((value) => Math.min(value, totalPackagePages - 1));
  }, [totalPackagePages]);

  const openAssetDetail = async (asset: KnowledgeAsset, target: KnowledgeAssetDetailTarget = 'detail') => {
    setSelectedAsset(asset);
    setAssetDetailTarget(target);
    setAssetChunks([]);
    setAssetTasks([]);
    setAssetDetailError(null);
    setAssetDetailLoading(true);
    try {
      const [chunks, tasks] = await Promise.all([
        fetchKnowledgeAssetChunks(token, workspaceId, asset.id),
        fetchKnowledgeAssetTasks(token, workspaceId, asset.id),
      ]);
      setAssetChunks(chunks);
      setAssetTasks(tasks);
    } catch (err) {
      setAssetDetailError(err instanceof Error ? err.message : '资产详情加载失败');
    } finally {
      setAssetDetailLoading(false);
    }
  };

  const handleAssetBasesUpdated = (asset: KnowledgeAsset) => {
    setSelectedAsset(asset);
    void openAssetDetail(asset, assetDetailTarget);
    onChanged();
  };

  const removeAssetsFromSelectedBase = async (assets: KnowledgeAsset[]) => {
    if (!selectedBaseId || assets.length === 0) {
      return;
    }
    setRemoveSubmitting(true);
    setRemoveError(null);
    try {
      await Promise.all(
        assets.map((asset) =>
          updateKnowledgeAssetBases(token, workspaceId, asset.id, {
            knowledgeBaseIds: asset.knowledgeBaseIds.filter((id) => id !== selectedBaseId),
          }),
        ),
      );
      setSelectedAssetIds([]);
      setRemoveConfirm(null);
      onChanged();
    } catch (err) {
      setRemoveError(err instanceof Error ? err.message : '移出知识库包失败');
    } finally {
      setRemoveSubmitting(false);
    }
  };

  const moveBaseToTrash = async () => {
    if (!confirmTrashBase) {
      return;
    }
    setTrashSubmitting(true);
    setTrashError(null);
    try {
      await trashKnowledgeBase(token, workspaceId, confirmTrashBase.id);
      if (selectedBaseId === confirmTrashBase.id) {
        setSelectedBaseId('');
      }
      setConfirmTrashBase(null);
      onChanged();
    } catch (err) {
      setTrashError(err instanceof Error ? err.message : '移入垃圾箱失败');
    } finally {
      setTrashSubmitting(false);
    }
  };

  const moveAssetToTrash = async () => {
    if (!trashAssetConfirm) {
      return;
    }
    setTrashAssetSubmitting(true);
    setTrashAssetError(null);
    try {
      await trashKnowledgeAsset(token, workspaceId, trashAssetConfirm.id);
      setTrashAssetConfirm(null);
      setSelectedAsset(null);
      onChanged();
    } catch (err) {
      setTrashAssetError(err instanceof Error ? err.message : '知识资产移入垃圾箱失败');
    } finally {
      setTrashAssetSubmitting(false);
    }
  };

  const retryAssetProcessing = async () => {
    if (!retryAssetConfirm) {
      return;
    }
    setRetryAssetSubmitting(true);
    setRetryAssetError(null);
    try {
      await retryKnowledgeAssetProcessing(token, workspaceId, retryAssetConfirm.id);
      setRetryAssetConfirm(null);
      setSelectedAsset(null);
      onChanged();
    } catch (err) {
      setRetryAssetError(err instanceof Error ? err.message : '知识资产重试失败');
    } finally {
      setRetryAssetSubmitting(false);
    }
  };

  const enhanceAsset = async () => {
    if (!enhanceAssetConfirm) {
      return;
    }
    setEnhanceAssetSubmitting(true);
    setEnhanceAssetError(null);
    try {
      await enhanceKnowledgeAsset(token, workspaceId, enhanceAssetConfirm.id);
      setEnhanceAssetConfirm(null);
      setSelectedAsset(null);
      onChanged();
    } catch (err) {
      setEnhanceAssetError(err instanceof Error ? err.message : 'AI 增强任务提交失败');
    } finally {
      setEnhanceAssetSubmitting(false);
    }
  };

  return (
    <>
      <Stack spacing={2.5}>
        <Section
          title="知识库包"
          action={
            <Stack direction="row" spacing={1}>
              <Button size="small" variant={selectedBaseId ? 'outlined' : 'contained'} onClick={() => setSelectedBaseId('')}>
                全部
              </Button>
              <Button size="small" startIcon={<AddIcon />} onClick={() => openDialog('knowledgeBase')} data-tour-id="knowledge-create-base">
                新建包
              </Button>
            </Stack>
          }
        >
          <Stack direction="row" alignItems="center" spacing={{ xs: 0.75, sm: 1.5 }} sx={{ minWidth: 0 }}>
            <IconButton
              aria-label="上一页知识库包"
              disabled={packagePage === 0 || data.knowledgeBases.length === 0}
              onClick={() => setPackagePage((value) => Math.max(0, value - 1))}
              sx={{ flexShrink: 0 }}
            >
              <ChevronLeftIcon />
            </IconButton>
            <Box ref={packageViewportRef} sx={{ flex: 1, minWidth: 0, overflow: 'hidden' }}>
              <Stack direction="row" spacing={1.5} sx={{ minHeight: 148 }}>
                {data.knowledgeBases.length === 0 && (
                  <Paper
                    elevation={0}
                    sx={{
                      width: '100%',
                      minHeight: 148,
                      display: 'flex',
                      alignItems: 'center',
                      border: '1px dashed',
                      borderColor: 'divider',
                      bgcolor: 'action.hover',
                      p: 2,
                    }}
                  >
                    <Typography color="text.secondary">暂无知识库包，请先新建包。</Typography>
                  </Paper>
                )}
                {visiblePackages.map((base) => (
                  <Card
                    key={base.id}
                    draggable
                    onDragStart={(event: React.DragEvent<HTMLDivElement>) => {
                      event.dataTransfer.effectAllowed = 'move';
                      event.dataTransfer.setData('text/plain', base.id);
                      setDraggingBase(base);
                    }}
                    onDragEnd={() => setDraggingBase(null)}
                    onClick={() => setSelectedBaseId((value) => (value === base.id ? '' : base.id))}
                    sx={[
                      {
                        width: knowledgePackageWidth,
                        flex: `0 0 ${knowledgePackageWidth}px`,
                        cursor: 'pointer',
                        border: '1px solid',
                      },
                      selectedSurfaceSx(selectedBaseId === base.id),
                    ]}
                  >
                    <CardContent sx={{ minHeight: 148 }}>
                      <Stack spacing={1.25} sx={{ height: '100%' }}>
                        <Stack direction="row" justifyContent="space-between" spacing={1} alignItems="flex-start">
                          <Typography variant="h3" sx={{ overflowWrap: 'anywhere' }}>{base.name}</Typography>
                          <Stack direction="row" spacing={0.5} alignItems="center" sx={{ flexShrink: 0 }}>
                            <Chip label={`${base.itemCount} 条`} size="small" color={selectedBaseId === base.id ? 'primary' : 'info'} />
                            <Tooltip title="移入垃圾箱">
                              <IconButton
                                size="small"
                                color="error"
                                aria-label={`移除 ${base.name}`}
                                onClick={(event) => {
                                  event.stopPropagation();
                                  setConfirmTrashBase(base);
                                  setTrashError(null);
                                }}
                              >
                                <DeleteOutlineIcon fontSize="small" />
                              </IconButton>
                            </Tooltip>
                          </Stack>
                        </Stack>
                        <Typography
                          color="text.secondary"
                          sx={{
                            minHeight: 44,
                            overflowWrap: 'anywhere',
                            display: '-webkit-box',
                            WebkitBoxOrient: 'vertical',
                            WebkitLineClamp: 2,
                            overflow: 'hidden',
                          }}
                        >
                          {base.description || '暂无说明'}
                        </Typography>
                        <Typography variant="body2" color="text.secondary" sx={{ mt: 'auto' }}>
                          更新于 {formatDate(base.updatedAt)}
                        </Typography>
                      </Stack>
                    </CardContent>
                  </Card>
                ))}
              </Stack>
            </Box>
            <IconButton
              aria-label="下一页知识库包"
              disabled={packagePage >= totalPackagePages - 1 || data.knowledgeBases.length === 0}
              onClick={() => setPackagePage((value) => Math.min(totalPackagePages - 1, value + 1))}
              sx={{ flexShrink: 0 }}
            >
              <ChevronRightIcon />
            </IconButton>
          </Stack>
        </Section>

        <Section
          title="知识资产"
          action={
            <Button startIcon={<AddIcon />} variant="contained" onClick={() => openDialog('knowledgeAsset')} data-tour-id="knowledge-create-asset">
              上传/创建资产
            </Button>
          }
        >
          <Stack spacing={2}>
            <Alert severity="info">
              支持 Word、文本、PDF、图片资产。图片和 PDF 使用 AI 视觉 OCR 解析，仅付费订阅可用；非付费订阅上传后会解析失败。
            </Alert>
            <Stack direction={{ xs: 'column', md: 'row' }} spacing={1.5} alignItems={{ xs: 'stretch', md: 'center' }}>
              <TextField
                size="small"
                label="检索资产"
                value={assetSearch}
                onChange={(event) => setAssetSearch(event.target.value)}
                fullWidth
              />
              {selectedBaseId && (
                <Chip
                  label={`筛选：${knowledgeBaseName(data.knowledgeBases, selectedBaseId)}`}
                  onDelete={() => setSelectedBaseId('')}
                  color="primary"
                />
              )}
              <Typography color="text.secondary" sx={{ whiteSpace: 'nowrap' }}>
                {filteredAssets.length} 个
              </Typography>
              {selectedBaseId && (
                <Button
                  color="warning"
                  variant="outlined"
                  disabled={selectedAssetIds.length === 0}
                  onClick={() => {
                    const assets = filteredAssets.filter((asset) => selectedAssetIds.includes(asset.id));
                    setRemoveError(null);
                    setRemoveConfirm({ assets });
                  }}
                >
                  批量移出包
                </Button>
              )}
              <Button
                color="error"
                variant="outlined"
                disabled={selectedAssetIds.length !== 1}
                onClick={() => {
                  const asset = filteredAssets.find((item) => item.id === selectedAssetIds[0]);
                  if (!asset) {
                    return;
                  }
                  setTrashAssetError(null);
                  setTrashAssetConfirm(asset);
                }}
              >
                删除选中
              </Button>
            </Stack>
            <KnowledgeAssetsTable
              assets={filteredAssets}
              bases={data.knowledgeBases}
              onOpenAsset={openAssetDetail}
              selectedIds={selectedAssetIds}
              onSelectedIdsChange={selectedBaseId ? setSelectedAssetIds : undefined}
              onRemoveFromBase={
                selectedBaseId
                  ? (asset) => {
                      setRemoveError(null);
                      setRemoveConfirm({ assets: [asset] });
                    }
                  : undefined
              }
              onRetryAsset={(asset) => {
                setRetryAssetError(null);
                setRetryAssetConfirm(asset);
              }}
              onEnhanceAsset={(asset) => {
                setEnhanceAssetError(null);
                setEnhanceAssetConfirm(asset);
              }}
              onTrashAsset={(asset) => {
                setTrashAssetError(null);
                setTrashAssetConfirm(asset);
              }}
            />
          </Stack>
        </Section>
      </Stack>
      {draggingBase && (
        <Box
          onDragOver={(event) => {
            event.preventDefault();
            event.dataTransfer.dropEffect = 'move';
          }}
          onDrop={(event) => {
            event.preventDefault();
            setConfirmTrashBase(draggingBase);
            setDraggingBase(null);
            setTrashError(null);
          }}
          sx={{
            position: 'fixed',
            inset: 0,
            zIndex: (theme) => theme.zIndex.modal - 1,
            bgcolor: 'rgba(16, 24, 20, 0.24)',
            display: 'flex',
            alignItems: 'flex-end',
            justifyContent: 'center',
            p: { xs: 2, md: 4 },
            pointerEvents: 'auto',
          }}
        >
          <Paper
            elevation={8}
            sx={{
              width: 'min(520px, 100%)',
              minHeight: 112,
              display: 'grid',
              placeItems: 'center',
              border: '2px dashed',
              borderColor: 'error.main',
              bgcolor: 'background.paper',
              color: 'error.main',
              p: 2,
            }}
          >
            <Stack spacing={0.75} alignItems="center">
              <DeleteOutlineIcon />
              <Typography fontWeight={800}>拖到这里移入垃圾箱</Typography>
              <Typography variant="body2" color="text.secondary">
                30 天内可在顶部垃圾箱恢复。
              </Typography>
            </Stack>
          </Paper>
        </Box>
      )}
      <Dialog open={Boolean(confirmTrashBase)} onClose={trashSubmitting ? undefined : () => setConfirmTrashBase(null)} maxWidth="xs" fullWidth>
        <DialogTitle>移入垃圾箱</DialogTitle>
        <DialogContent>
          <Stack spacing={1.5} sx={{ pt: 1 }}>
            {trashError && <Alert severity="error">{trashError}</Alert>}
            <Typography>
              确认将「{confirmTrashBase?.name}」移入垃圾箱？30 天内可以在顶部垃圾箱恢复。
            </Typography>
          </Stack>
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setConfirmTrashBase(null)} disabled={trashSubmitting}>
            取消
          </Button>
          <Button color="error" variant="contained" onClick={moveBaseToTrash} disabled={trashSubmitting}>
            {trashSubmitting ? '处理中' : '移入垃圾箱'}
          </Button>
        </DialogActions>
      </Dialog>
      <Dialog open={Boolean(removeConfirm)} onClose={removeSubmitting ? undefined : () => setRemoveConfirm(null)} maxWidth="xs" fullWidth>
        <DialogTitle>移出当前知识库包</DialogTitle>
        <DialogContent>
          <Stack spacing={1.5} sx={{ pt: 1 }}>
            {removeError && <Alert severity="error">{removeError}</Alert>}
            <Typography>
              确认将 {removeConfirm?.assets.length ?? 0} 个知识资产从「{knowledgeBaseName(data.knowledgeBases, selectedBaseId)}」移出？
            </Typography>
            <Typography variant="body2" color="text.secondary">
              资产本身不会删除，只是不再归属于当前知识库包。
            </Typography>
          </Stack>
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setRemoveConfirm(null)} disabled={removeSubmitting}>
            取消
          </Button>
          <Button
            color="warning"
            variant="contained"
            onClick={() => removeConfirm && void removeAssetsFromSelectedBase(removeConfirm.assets)}
            disabled={removeSubmitting}
          >
            {removeSubmitting ? '处理中' : '确认移出'}
          </Button>
        </DialogActions>
      </Dialog>
      <Dialog open={Boolean(trashAssetConfirm)} onClose={trashAssetSubmitting ? undefined : () => setTrashAssetConfirm(null)} maxWidth="xs" fullWidth>
        <DialogTitle>删除知识资产</DialogTitle>
        <DialogContent>
          <Stack spacing={1.5} sx={{ pt: 1 }}>
            {trashAssetError && <Alert severity="error">{trashAssetError}</Alert>}
            <Typography>
              确认将「{trashAssetConfirm?.title}」移入垃圾箱？30 天内可以在顶部垃圾箱恢复。
            </Typography>
          </Stack>
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setTrashAssetConfirm(null)} disabled={trashAssetSubmitting}>
            取消
          </Button>
          <Button color="error" variant="contained" onClick={moveAssetToTrash} disabled={trashAssetSubmitting}>
            {trashAssetSubmitting ? '处理中' : '移入垃圾箱'}
          </Button>
        </DialogActions>
      </Dialog>
      <Dialog open={Boolean(retryAssetConfirm)} onClose={retryAssetSubmitting ? undefined : () => setRetryAssetConfirm(null)} maxWidth="xs" fullWidth>
        <DialogTitle>重试知识资产</DialogTitle>
        <DialogContent>
          <Stack spacing={1.5} sx={{ pt: 1 }}>
            {retryAssetError && <Alert severity="error">{retryAssetError}</Alert>}
            <Typography>
              确认重新解析并拆分「{retryAssetConfirm?.title}」？现有片段会被清空后重新生成。
            </Typography>
          </Stack>
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setRetryAssetConfirm(null)} disabled={retryAssetSubmitting}>
            取消
          </Button>
          <Button variant="contained" onClick={retryAssetProcessing} disabled={retryAssetSubmitting}>
            {retryAssetSubmitting ? '处理中' : '确认重试'}
          </Button>
        </DialogActions>
      </Dialog>
      <Dialog open={Boolean(enhanceAssetConfirm)} onClose={enhanceAssetSubmitting ? undefined : () => setEnhanceAssetConfirm(null)} maxWidth="xs" fullWidth>
        <DialogTitle>AI 增强知识资产</DialogTitle>
        <DialogContent>
          <Stack spacing={1.5} sx={{ pt: 1 }}>
            {enhanceAssetError && <Alert severity="error">{enhanceAssetError}</Alert>}
            <Typography>
              确认对「{enhanceAssetConfirm?.title}」应用 AI 增强？增强成功后会替换当前知识片段；失败时保留现有片段。
            </Typography>
          </Stack>
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setEnhanceAssetConfirm(null)} disabled={enhanceAssetSubmitting}>
            取消
          </Button>
          <Button variant="contained" onClick={enhanceAsset} disabled={enhanceAssetSubmitting}>
            {enhanceAssetSubmitting ? '提交中' : '开始增强'}
          </Button>
        </DialogActions>
      </Dialog>
      <KnowledgeAssetDetailDialog
        token={token}
        workspaceId={workspaceId}
        asset={selectedAsset}
        bases={data.knowledgeBases}
        chunks={assetChunks}
        tasks={assetTasks}
        focusTarget={assetDetailTarget}
        loading={assetDetailLoading}
        error={assetDetailError}
        onBasesUpdated={handleAssetBasesUpdated}
        onRetryAsset={(asset) => {
          setRetryAssetError(null);
          setRetryAssetConfirm(asset);
        }}
        onEnhanceAsset={(asset) => {
          setEnhanceAssetError(null);
          setEnhanceAssetConfirm(asset);
        }}
        onTrashAsset={(asset) => {
          setTrashAssetError(null);
          setTrashAssetConfirm(asset);
        }}
        onClose={() => setSelectedAsset(null)}
      />
    </>
  );
}

function KnowledgeAssetDetailDialog({
  token,
  workspaceId,
  asset,
  bases,
  chunks,
  tasks,
  focusTarget,
  loading,
  error,
  onBasesUpdated,
  onRetryAsset,
  onEnhanceAsset,
  onTrashAsset,
  onClose,
}: {
  token: string;
  workspaceId: string;
  asset: KnowledgeAsset | null;
  bases: KnowledgeBase[];
  chunks: KnowledgeChunk[];
  tasks: KnowledgeProcessingTask[];
  focusTarget: KnowledgeAssetDetailTarget;
  loading: boolean;
  error: string | null;
  onBasesUpdated: (asset: KnowledgeAsset) => void;
  onRetryAsset: (asset: KnowledgeAsset) => void;
  onEnhanceAsset: (asset: KnowledgeAsset) => void;
  onTrashAsset: (asset: KnowledgeAsset) => void;
  onClose: () => void;
}) {
  const [knowledgeBaseIds, setKnowledgeBaseIds] = useState<string[]>([]);
  const [savingBases, setSavingBases] = useState(false);
  const [baseError, setBaseError] = useState<string | null>(null);
  const tipsRef = useRef<HTMLDivElement | null>(null);
  const chunksRef = useRef<HTMLDivElement | null>(null);

  useEffect(() => {
    if (asset) {
      setKnowledgeBaseIds(asset.knowledgeBaseIds ?? []);
      setBaseError(null);
    }
  }, [asset]);

  const saveBases = async () => {
    if (!asset) {
      return;
    }
    setSavingBases(true);
    setBaseError(null);
    try {
      const updatedAsset = await updateKnowledgeAssetBases(token, workspaceId, asset.id, { knowledgeBaseIds });
      onBasesUpdated(updatedAsset);
    } catch (err) {
      setBaseError(err instanceof Error ? err.message : '知识资产分类保存失败');
    } finally {
      setSavingBases(false);
    }
  };

  const basesChanged = Boolean(asset) && knowledgeBaseIds.join('|') !== (asset?.knowledgeBaseIds ?? []).join('|');
  const aiStatus = asset?.aiEnhancementStatus || (asset?.aiEnhancementEnabled ? 'pending' : 'disabled');
  const aiEnhancementRunning = Boolean(asset?.aiEnhancementEnabled && (aiStatus === 'pending' || aiStatus === 'processing'));
  const canEnhanceAsset = Boolean(
    asset &&
      asset.status === 'ready' &&
      !aiEnhancementRunning &&
      (!asset.aiEnhancementEnabled || aiStatus === 'disabled' || aiStatus === 'failed' || aiStatus === 'skipped'),
  );

  useEffect(() => {
    if (!asset || loading) {
      return;
    }
    const targetRef = focusTarget === 'chunks' ? chunksRef : focusTarget === 'tips' ? tipsRef : null;
    if (!targetRef?.current) {
      return;
    }
    window.setTimeout(() => {
      targetRef.current?.scrollIntoView({ behavior: 'smooth', block: 'start' });
    }, 80);
  }, [asset, focusTarget, loading]);

  return (
    <Dialog open={Boolean(asset)} onClose={loading || savingBases ? undefined : onClose} fullWidth maxWidth="md">
      <DialogTitle>{asset?.title ?? '知识资产详情'}</DialogTitle>
      <DialogContent>
        <Stack spacing={2} sx={{ pt: 1 }}>
          {error && <Alert severity="error">{error}</Alert>}
          {baseError && <Alert severity="error">{baseError}</Alert>}
          {asset && (
            <Grid container spacing={1.5}>
              <Grid size={{ xs: 12, sm: 6 }}>
                <DetailInfoRow label="状态">
                  <StatusChip option={statusChipOption(asset.status, knowledgeAssetStatusMap)} />
                </DetailInfoRow>
              </Grid>
              <Grid size={{ xs: 12, sm: 6 }}>
                <InfoRow label="进度" value={`${asset.progress}%`} />
              </Grid>
              <Grid size={{ xs: 12, sm: 6 }}>
                <DetailInfoRow label="AI 增强">
                  <StatusChip
                    option={
                      asset.aiEnhancementEnabled
                        ? statusChipOption(asset.aiEnhancementStatus, aiEnhancementStatusMap)
                        : aiEnhancementStatusMap.disabled
                    }
                  />
                </DetailInfoRow>
              </Grid>
              <Grid size={{ xs: 12, sm: 6 }}>
                <InfoRow label="更新时间" value={formatDate(asset.updatedAt)} />
              </Grid>
              <Grid size={{ xs: 12 }}>
                <InfoRow label="文件" value={asset.originalFilename || asset.mimeType || asset.assetType || '文本资产'} />
              </Grid>
              <Grid size={{ xs: 12 }}>
                <FormControl fullWidth disabled={savingBases || bases.length === 0}>
                  <InputLabel shrink>知识库包分类</InputLabel>
                  <Select
                    multiple
                    displayEmpty
                    label="知识库包分类"
                    value={knowledgeBaseIds}
                    onChange={(event) => {
                      const value = event.target.value;
                      setKnowledgeBaseIds(typeof value === 'string' ? value.split(',') : value);
                    }}
                    renderValue={(selected) => knowledgeBaseNames(bases, selected, '未分类资产')}
                  >
                    {bases.map((base) => (
                      <MenuItem key={base.id} value={base.id}>
                        <Checkbox checked={knowledgeBaseIds.includes(base.id)} />
                        <ListItemText primary={base.name} />
                      </MenuItem>
                    ))}
                  </Select>
                </FormControl>
              </Grid>
              {asset.errorMessage && (
                <Grid size={{ xs: 12 }}>
                  <Alert severity="error">{asset.errorMessage}</Alert>
                </Grid>
              )}
            </Grid>
          )}
          {loading ? (
            <Stack direction="row" spacing={1.5} alignItems="center">
              <CircularProgress size={18} />
              <Typography color="text.secondary">正在加载资产处理详情</Typography>
            </Stack>
          ) : (
            <>
              <Divider />
              <Box ref={tipsRef}>
                <Typography fontWeight={700} sx={{ mb: 1 }}>
                  处理任务
                </Typography>
                <Stack spacing={1}>
                  {tasks.length === 0 && <Typography color="text.secondary">暂无处理任务记录</Typography>}
                  {tasks.map((task) => (
                    <Paper key={task.id} variant="outlined" sx={{ p: 1.5 }}>
                      <Stack direction={{ xs: 'column', sm: 'row' }} spacing={1} justifyContent="space-between">
                        <Typography fontWeight={700}>{taskTypeLabel(task.taskType)}</Typography>
                        <Stack direction="row" spacing={1} alignItems="center" flexWrap="wrap" useFlexGap>
                          <StatusChip option={statusChipOption(task.status, knowledgeTaskStatusMap)} />
                          <Typography variant="body2" color="text.secondary">
                            {task.progress}%
                          </Typography>
                        </Stack>
                      </Stack>
                      {task.errorMessage && (
                        <Typography variant="body2" color="error" sx={{ mt: 1, overflowWrap: 'anywhere' }}>
                          {task.errorMessage}
                        </Typography>
                      )}
                    </Paper>
                  ))}
                </Stack>
              </Box>
              <Box ref={chunksRef}>
                <Typography fontWeight={700} sx={{ mb: 1 }}>
                  知识片段
                </Typography>
                <Stack spacing={1}>
                  {chunks.length === 0 && <Typography color="text.secondary">暂无可用片段</Typography>}
                  {chunks.slice(0, 8).map((chunk) => (
                    <Paper key={chunk.id} variant="outlined" sx={{ p: 1.5 }}>
                      <Stack spacing={0.75}>
                        <Stack direction={{ xs: 'column', sm: 'row' }} spacing={1} justifyContent="space-between">
                          <Typography fontWeight={700} sx={{ overflowWrap: 'anywhere' }}>
                            {chunk.title || `片段 ${chunk.chunkIndex + 1}`}
                          </Typography>
                          <StatusChip option={statusChipOption(chunk.embeddingStatus, embeddingStatusMap)} />
                        </Stack>
                        <Typography
                          variant="body2"
                          color="text.secondary"
                          sx={{
                            overflowWrap: 'anywhere',
                            display: '-webkit-box',
                            WebkitBoxOrient: 'vertical',
                            WebkitLineClamp: 3,
                            overflow: 'hidden',
                          }}
                        >
                          {chunk.summary || chunk.content}
                        </Typography>
                      </Stack>
                    </Paper>
                  ))}
                </Stack>
              </Box>
            </>
          )}
        </Stack>
      </DialogContent>
      <DialogActions>
        {asset && (
          <Button color="error" onClick={() => onTrashAsset(asset)} disabled={loading || savingBases}>
            删除
          </Button>
        )}
        {asset && asset.status === 'failed' && (
          <Button onClick={() => onRetryAsset(asset)} disabled={loading || savingBases}>
            重试
          </Button>
        )}
        {asset && (
          <VIPFeatureButton
            onClick={() => onEnhanceAsset(asset)}
            disabled={loading || savingBases || !canEnhanceAsset}
            selected={canEnhanceAsset}
            animateHighlight={canEnhanceAsset}
          >
            AI 增强
          </VIPFeatureButton>
        )}
        <Button onClick={onClose} disabled={loading || savingBases}>
          关闭
        </Button>
        <Button onClick={saveBases} disabled={loading || savingBases || !basesChanged} variant="contained">
          {savingBases ? '保存中' : '保存分类'}
        </Button>
      </DialogActions>
    </Dialog>
  );
}

function AccountsView({
  data,
  openDialog,
  onLoginMediaAccount,
}: {
  data: WorkspaceData;
  openDialog: (dialog: DialogKey) => void;
  onLoginMediaAccount: (accountId: string) => void;
}) {
  return (
    <Stack spacing={2}>
      <Section
        title="媒体平台能力"
        action={
          <Button startIcon={<AddIcon />} variant="contained" onClick={() => openDialog('mediaAccount')} data-tour-id="media-bind-account">
            绑定账号
          </Button>
        }
      >
        <MediaPlatformTable platforms={data.mediaPlatforms} />
      </Section>
      <Section title="已绑定媒体账号">
        <MediaAccountsTable accounts={data.mediaAccounts} platforms={data.mediaPlatforms} onLogin={onLoginMediaAccount} />
      </Section>
    </Stack>
  );
}

function GenerateView({ data, openDialog }: { data: WorkspaceData; openDialog: (dialog: DialogKey) => void }) {
  return (
    <Section
      title="关键词生成文章"
      action={
        <Button startIcon={<PsychologyAltOutlinedIcon />} variant="contained" onClick={() => openDialog('generate')} data-tour-id="generate-start">
          开始生成
        </Button>
      }
    >
      <Stack spacing={1.5}>
        <ContentTable contents={data.contents.filter((item) => item.source !== 'manual')} />
      </Stack>
    </Section>
  );
}

function ContentsView({
  contents,
  openDialog,
  onPreparePublish,
}: {
  contents: Content[];
  openDialog: (dialog: DialogKey) => void;
  onPreparePublish: (contentId: string) => void;
}) {
  return (
    <Section
      title="内容管理"
      action={<Button startIcon={<AddIcon />} variant="contained" onClick={() => openDialog('content')}>新建内容</Button>}
    >
      <Box data-tour-id="contents-view">
        <ContentTable contents={contents} onPreparePublish={onPreparePublish} />
      </Box>
    </Section>
  );
}

function SchedulesView({ data, openDialog }: { data: WorkspaceData; openDialog: (dialog: DialogKey) => void }) {
  return (
    <Section
      title="发布计划"
      action={<Button startIcon={<ScheduleOutlinedIcon />} variant="contained" onClick={() => openDialog('schedule')}>新建计划</Button>}
    >
      <SchedulesTable data={data} />
    </Section>
  );
}

function JobsView({ data }: { data: WorkspaceData }) {
  return (
    <Section title="发布任务">
      <Box data-tour-id="jobs-list">
        <JobsTable data={data} />
      </Box>
    </Section>
  );
}

function SettingsView({ workspace, user }: { workspace: Workspace; user: User }) {
  return (
    <Section title="工作区设置">
      <Grid container spacing={2}>
        <Grid size={{ xs: 12, md: 6 }}>
          <InfoRow label="当前用户" value={`${user.name} / ${user.email}`} />
          <InfoRow label="工作区 ID" value={workspace.id} />
          <InfoRow label="名称" value={workspace.name} />
          <InfoRow label="类型" value={workspace.type === 'company' ? '公司' : '个人'} />
        </Grid>
        <Grid size={{ xs: 12, md: 6 }}>
          <InfoRow label="账号订阅" value={formatSubscription(user)} />
          <InfoRow label="工作区方案" value={workspace.plan} />
          <InfoRow label="行业" value={workspace.industry} />
          <InfoRow label="语言" value={workspace.language} />
          <InfoRow label="默认语气" value={workspace.tone} />
        </Grid>
      </Grid>
    </Section>
  );
}

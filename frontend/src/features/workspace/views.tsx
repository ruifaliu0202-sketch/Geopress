import { useEffect, useMemo, useRef, useState } from 'react';
import {
  Alert,
  Box,
  Button,
  Card,
  CardContent,
  Checkbox,
  Chip,
  CircularProgress,
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
import type { DialogKey, ViewKey } from '../../appTypes';
import { assignKnowledgeItemsToBases, completeOnboarding, fetchSubscriptionPlans, login, registerUser } from '../../api';
import { InfoRow, MetricCard, Section } from '../../components/common';
import {
  ContentTable,
  JobsTable,
  KnowledgeItemsTable,
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
import type { Content, SubscriptionPlan, User, Workspace, WorkspaceData } from '../../types';
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
            title="知识库上下文"
            action={
              <Button
                size="small"
                startIcon={<AddIcon />}
                onClick={() => openDialog('knowledgeItem')}
                data-tour-id="overview-create-knowledge-item"
              >
                新增条目
              </Button>
            }
          >
            <KnowledgeItemsTable items={data.knowledgeItems.slice(0, 5)} bases={data.knowledgeBases} />
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
  const [search, setSearch] = useState('');
  const [selectedItemIds, setSelectedItemIds] = useState<string[]>([]);
  const [targetBaseIds, setTargetBaseIds] = useState<string[]>([]);
  const [assigning, setAssigning] = useState(false);
  const [assignError, setAssignError] = useState<string | null>(null);
  const totalPackagePages = Math.max(1, Math.ceil(data.knowledgeBases.length / visiblePackageCount));
  const packageStart = Math.min(packagePage, totalPackagePages - 1) * visiblePackageCount;
  const visiblePackages = data.knowledgeBases.slice(packageStart, packageStart + visiblePackageCount);
  const filteredItems = useMemo(() => {
    const keyword = search.trim().toLowerCase();
    return data.knowledgeItems.filter((item) => {
      if (selectedBaseId && !item.knowledgeBaseIds.includes(selectedBaseId)) {
        return false;
      }
      if (!keyword) {
        return true;
      }
      return [item.title, item.content, item.type, knowledgeBaseNames(data.knowledgeBases, item.knowledgeBaseIds)]
        .join(' ')
        .toLowerCase()
        .includes(keyword);
    });
  }, [data.knowledgeBases, data.knowledgeItems, search, selectedBaseId]);

  useEffect(() => {
    setSelectedItemIds((current) => current.filter((id) => filteredItems.some((item) => item.id === id)));
  }, [filteredItems]);

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

  useEffect(() => {
    setTargetBaseIds((current) => current.filter((id) => data.knowledgeBases.some((base) => base.id === id)));
  }, [data.knowledgeBases]);

  const submitAssign = async () => {
    if (selectedItemIds.length === 0 || targetBaseIds.length === 0) {
      setAssignError('请选择条目和目标知识库包');
      return;
    }
    setAssigning(true);
    setAssignError(null);
    try {
      await assignKnowledgeItemsToBases(token, workspaceId, {
        knowledgeItemIds: selectedItemIds,
        knowledgeBaseIds: targetBaseIds,
      });
      setSelectedItemIds([]);
      onChanged();
    } catch (err) {
      setAssignError(err instanceof Error ? err.message : '批量添加失败');
    } finally {
      setAssigning(false);
    }
  };

  return (
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
        <Stack direction="row" alignItems="center" spacing={1.5}>
          <IconButton
            aria-label="上一页知识库包"
            disabled={packagePage === 0}
            onClick={() => setPackagePage((value) => Math.max(0, value - 1))}
          >
            <ChevronLeftIcon />
          </IconButton>
          <Box ref={packageViewportRef} sx={{ flex: 1, overflow: 'hidden' }}>
            <Stack direction="row" spacing={1.5} sx={{ minHeight: 148 }}>
              {visiblePackages.map((base) => (
                <Card
                  key={base.id}
                  onClick={() => setSelectedBaseId((value) => (value === base.id ? '' : base.id))}
                  sx={{
                    width: knowledgePackageWidth,
                    flex: `0 0 ${knowledgePackageWidth}px`,
                    cursor: 'pointer',
                    border: '1px solid',
                    borderColor: selectedBaseId === base.id ? 'primary.main' : 'divider',
                    boxShadow: selectedBaseId === base.id ? 2 : 0,
                  }}
                >
                  <CardContent>
                    <Stack spacing={1.25}>
                      <Stack direction="row" justifyContent="space-between" spacing={2}>
                        <Typography variant="h3" sx={{ overflowWrap: 'anywhere' }}>{base.name}</Typography>
                        <Chip label={`${base.itemCount} 条`} size="small" color={selectedBaseId === base.id ? 'primary' : 'info'} />
                      </Stack>
                      <Typography color="text.secondary" sx={{ minHeight: 44 }}>
                        {base.description || '暂无说明'}
                      </Typography>
                      <Typography variant="body2" color="text.secondary">
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
            disabled={packagePage >= totalPackagePages - 1}
            onClick={() => setPackagePage((value) => Math.min(totalPackagePages - 1, value + 1))}
          >
            <ChevronRightIcon />
          </IconButton>
        </Stack>
      </Section>

      <Section
        title="知识条目检索"
        action={
          <Stack direction={{ xs: 'column', md: 'row' }} spacing={1} alignItems={{ xs: 'stretch', md: 'center' }}>
            <FormControl size="small" sx={{ minWidth: 220 }}>
              <InputLabel>批量加入知识库包</InputLabel>
              <Select
                multiple
                label="批量加入知识库包"
                value={targetBaseIds}
                onChange={(event) => {
                  const value = event.target.value;
                  setTargetBaseIds(typeof value === 'string' ? value.split(',') : value);
                }}
                renderValue={(selected) => knowledgeBaseNames(data.knowledgeBases, selected)}
              >
                {data.knowledgeBases.map((base) => (
                  <MenuItem key={base.id} value={base.id}>
                    <Checkbox checked={targetBaseIds.includes(base.id)} />
                    <ListItemText primary={base.name} />
                  </MenuItem>
                ))}
              </Select>
            </FormControl>
            <Button variant="outlined" disabled={assigning || selectedItemIds.length === 0 || targetBaseIds.length === 0} onClick={submitAssign}>
              加入选中条目
            </Button>
            <Button startIcon={<AddIcon />} variant="contained" onClick={() => openDialog('knowledgeItem')} data-tour-id="knowledge-create-item">
              新增条目
            </Button>
          </Stack>
        }
      >
        <Stack spacing={2}>
          {assignError && <Alert severity="error">{assignError}</Alert>}
          <Stack direction={{ xs: 'column', md: 'row' }} spacing={1.5} alignItems={{ xs: 'stretch', md: 'center' }}>
            <TextField
              size="small"
              label="检索条目"
              value={search}
              onChange={(event) => setSearch(event.target.value)}
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
              {filteredItems.length} 条
            </Typography>
          </Stack>
          <KnowledgeItemsTable
            items={filteredItems}
            bases={data.knowledgeBases}
            selectedIds={selectedItemIds}
            onSelectedIdsChange={setSelectedItemIds}
          />
        </Stack>
      </Section>
    </Stack>
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

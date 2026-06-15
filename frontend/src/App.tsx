import { useEffect, useMemo, useRef, useState, type ReactNode } from 'react';
import {
  Alert,
  Accordion,
  AccordionDetails,
  AccordionSummary,
  AppBar,
  Box,
  Button,
  Card,
  CardContent,
  Checkbox,
  Chip,
  CircularProgress,
  Container,
  Dialog,
  DialogActions,
  DialogContent,
  DialogTitle,
  Divider,
  Drawer,
  FormControl,
  Grid,
  IconButton,
  InputLabel,
  MenuItem,
  Link,
  List,
  ListItem,
  ListItemText,
  Paper,
  Select,
  Stack,
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableRow,
  TextField,
  Toolbar,
  Tooltip,
  Typography,
} from '@mui/material';
import AccountTreeOutlinedIcon from '@mui/icons-material/AccountTreeOutlined';
import AddIcon from '@mui/icons-material/Add';
import ArticleOutlinedIcon from '@mui/icons-material/ArticleOutlined';
import AutorenewIcon from '@mui/icons-material/Autorenew';
import ChevronLeftIcon from '@mui/icons-material/ChevronLeft';
import ChevronRightIcon from '@mui/icons-material/ChevronRight';
import CloseIcon from '@mui/icons-material/Close';
import ContentCopyIcon from '@mui/icons-material/ContentCopy';
import DashboardOutlinedIcon from '@mui/icons-material/DashboardOutlined';
import ExpandMoreIcon from '@mui/icons-material/ExpandMore';
import KeyOutlinedIcon from '@mui/icons-material/KeyOutlined';
import LoginOutlinedIcon from '@mui/icons-material/LoginOutlined';
import ManageAccountsOutlinedIcon from '@mui/icons-material/ManageAccountsOutlined';
import PsychologyAltOutlinedIcon from '@mui/icons-material/PsychologyAltOutlined';
import PublishOutlinedIcon from '@mui/icons-material/PublishOutlined';
import ScheduleOutlinedIcon from '@mui/icons-material/ScheduleOutlined';
import SettingsOutlinedIcon from '@mui/icons-material/SettingsOutlined';
import {
  completeOnboarding,
  createContent,
  createKnowledgeBase,
  createKnowledgeItem,
  createMediaAccount,
  createPublishSchedule,
  fetchWorkspace,
  fetchSubscriptionPlans,
  formatKnowledgeContent,
  generateContent,
  login,
  preparePublish,
  registerUser,
  runPublishJob,
  startMediaAccountBrowserLogin,
  completeMediaAccountBrowserLogin,
  assignKnowledgeItemsToBases,
} from './api';
import { AdminConsole } from './admin/AdminConsole';
import type {
  Content,
  ContentStatus,
  KnowledgeBase,
  KnowledgeItem,
  GenerationTrace,
  MediaAccount,
  MediaPlatform,
  PreparedPost,
  PreparePublishResponse,
  PublishJobStatus,
  PublishScheduleFrequency,
  SubscriptionPlan,
  User,
  Workspace,
  WorkspaceData,
} from './types';
import vipGoldIconUrl from './assets/vip-gold.png';

type ViewKey = 'overview' | 'knowledge' | 'accounts' | 'generate' | 'contents' | 'schedules' | 'jobs' | 'settings' | 'admin';
type DialogKey =
  | 'knowledgeBase'
  | 'knowledgeItem'
  | 'mediaAccount'
  | 'mediaAccountLogin'
  | 'content'
  | 'generate'
  | 'schedule'
  | 'publishPrepare'
  | null;

const navItems: Array<{ key: ViewKey; label: string; icon: ReactNode }> = [
  { key: 'overview', label: '概览', icon: <DashboardOutlinedIcon /> },
  { key: 'knowledge', label: '知识库', icon: <PsychologyAltOutlinedIcon /> },
  { key: 'accounts', label: '媒体账号', icon: <KeyOutlinedIcon /> },
  { key: 'generate', label: 'AI 生成', icon: <ArticleOutlinedIcon /> },
  { key: 'contents', label: '内容', icon: <AccountTreeOutlinedIcon /> },
  { key: 'schedules', label: '计划', icon: <ScheduleOutlinedIcon /> },
  { key: 'jobs', label: '任务', icon: <PublishOutlinedIcon /> },
  { key: 'settings', label: '工作区', icon: <SettingsOutlinedIcon /> },
];

const adminNavItem: { key: ViewKey; label: string; icon: ReactNode } = {
  key: 'admin',
  label: '平台后台',
  icon: <ManageAccountsOutlinedIcon />,
};

const contentStatusMap: Record<ContentStatus, { label: string; color: 'default' | 'info' | 'warning' | 'success' | 'error' }> = {
  draft: { label: '草稿', color: 'default' },
  review: { label: '待审核', color: 'warning' },
  approved: { label: '已通过', color: 'success' },
  scheduled: { label: '已排程', color: 'info' },
  published: { label: '已发布', color: 'success' },
  failed: { label: '失败', color: 'error' },
  archived: { label: '已归档', color: 'default' },
};

const jobStatusMap: Record<PublishJobStatus, { label: string; color: 'default' | 'info' | 'error' | 'success' | 'warning' }> = {
  queued: { label: '排队中', color: 'default' },
  running: { label: '发布中', color: 'info' },
  manual_pending: { label: '待人工发布', color: 'warning' },
  retrying: { label: '重试中', color: 'warning' },
  succeeded: { label: '成功', color: 'success' },
  failed: { label: '失败', color: 'error' },
};

const frequencyLabel: Record<PublishScheduleFrequency, string> = {
  once: '一次性',
  daily: '每天',
  weekly: '每周',
  monthly: '每月',
};

const knowledgePackageWidth = 280;
const knowledgePackageGap = 12;

const contentTypeOptions = [
  { value: 'xiaohongshu_long_article', label: '小红书长文' },
  { value: 'article', label: '通用长文章' },
  { value: 'brief', label: '短文' },
  { value: 'case_study', label: '案例稿' },
  { value: 'product_intro', label: '产品介绍' },
];

const onboardingIndustries = ['本地生活', 'B2B SaaS', '教育培训', '美业医美', '电商零售', '企业服务', '个人创作者', '其他'];
const onboardingTones = ['专业', '清晰', '克制', '亲和', '犀利', '种草感', '可信', '实用'];

function App() {
  const [token, setToken] = useState('');
  const [user, setUser] = useState<User | null>(null);
  const [workspaceId, setWorkspaceId] = useState('');
  const [activeView, setActiveView] = useState<ViewKey>('overview');
  const [workspace, setWorkspace] = useState<WorkspaceData | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [reloadKey, setReloadKey] = useState(0);
  const [dialog, setDialog] = useState<DialogKey>(null);
  const [selectedContentId, setSelectedContentId] = useState('');
  const [selectedMediaAccountId, setSelectedMediaAccountId] = useState('');
  const [workbenchOpen, setWorkbenchOpen] = useState(true);
  const [generationTrace, setGenerationTrace] = useState<GenerationTrace | null>(null);
  const [traceDrawerOpen, setTraceDrawerOpen] = useState(false);

  useEffect(() => {
    if (!token || !workspaceId) {
      return;
    }

    let mounted = true;
    setLoading(true);
    setError(null);
    fetchWorkspace(token, workspaceId)
      .then((data) => {
        if (mounted) {
          setWorkspace(data);
          setUser(data.user);
        }
      })
      .catch((err: unknown) => {
        if (mounted) {
          setError(err instanceof Error ? err.message : '加载失败');
        }
      })
      .finally(() => {
        if (mounted) {
          setLoading(false);
        }
      });

    return () => {
      mounted = false;
    };
  }, [reloadKey, token, workspaceId]);

  const currentWorkspace = useMemo(() => {
    return workspace?.workspaces.find((item) => item.id === workspaceId) ?? null;
  }, [workspace, workspaceId]);

  const refresh = () => setReloadKey((value) => value + 1);
  const visibleNavItems = user?.isPlatformAdmin ? [...navItems, adminNavItem] : navItems;

  if (!token) {
    return (
      <LoginView
        onLogin={(result) => {
          setToken(result.token);
          setUser(result.user);
          setWorkspaceId(result.workspaces[0]?.id ?? '');
          setWorkspace(null);
          setActiveView('overview');
        }}
      />
    );
  }

  if (user && !user.onboardingCompleted && workspaceId) {
    return (
      <OnboardingView
        token={token}
        workspaceId={workspaceId}
        user={user}
        onComplete={(result) => {
          setUser(result.user);
          setWorkspace(null);
          setReloadKey((value) => value + 1);
        }}
      />
    );
  }

  if (activeView === 'admin' && user?.isPlatformAdmin) {
    return <AdminConsole token={token} onBack={() => setActiveView('overview')} />;
  }

  return (
    <Box sx={{ minHeight: '100vh', bgcolor: 'background.default' }}>
      <AppBar
        position="sticky"
        color="inherit"
        elevation={0}
        sx={{ borderBottom: '1px solid', borderColor: 'divider' }}
      >
        <Toolbar sx={{ gap: 2 }}>
          <Stack direction="row" alignItems="center" spacing={1.25} sx={{ minWidth: 230 }}>
            <Box
              sx={{
                display: 'grid',
                placeItems: 'center',
                width: 34,
                height: 34,
                borderRadius: 1,
                bgcolor: 'primary.main',
                color: 'primary.contrastText',
                fontWeight: 800,
              }}
            >
              G
            </Box>
            <Box>
              <Typography variant="h3" sx={{ lineHeight: 1 }}>
                Geopress
              </Typography>
              <Typography variant="caption" color="text.secondary">
                {user ? `${user.name} / ${user.email}` : '内容自动发布平台'}
              </Typography>
            </Box>
          </Stack>

          <Stack
            direction="row"
            spacing={0.75}
            sx={{ flex: 1, display: { xs: 'none', lg: 'flex' }, alignItems: 'center' }}
          >
            {visibleNavItems.map((item) => (
              <Button
                key={item.key}
                startIcon={item.icon}
                variant={activeView === item.key ? 'contained' : 'text'}
                color={activeView === item.key ? 'primary' : 'inherit'}
                onClick={() => setActiveView(item.key)}
              >
                {item.label}
              </Button>
            ))}
          </Stack>

          <FormControl size="small" sx={{ minWidth: { xs: 160, sm: 240 } }}>
            <InputLabel id="workspace-select-label">工作区</InputLabel>
            <Select
              labelId="workspace-select-label"
              label="工作区"
              value={workspaceId}
              onChange={(event) => setWorkspaceId(String(event.target.value))}
            >
              {(workspace?.workspaces ?? []).map((item) => (
                <MenuItem key={item.id} value={item.id}>
                  {item.name}
                </MenuItem>
              ))}
              {!workspace && workspaceId && <MenuItem value={workspaceId}>{workspaceId}</MenuItem>}
            </Select>
          </FormControl>

          <Tooltip title="刷新数据">
            <span>
              <IconButton disabled={loading} onClick={refresh}>
                <AutorenewIcon />
              </IconButton>
            </span>
          </Tooltip>
        </Toolbar>
      </AppBar>

      <Container maxWidth="xl" sx={{ py: 3 }}>
        <Box
          sx={{
            display: 'flex',
            flexDirection: { xs: 'column', md: 'row' },
            alignItems: 'flex-start',
            gap: 3,
          }}
        >
          <WorkspaceWorkbenchPanel
            open={workbenchOpen}
            currentWorkspace={currentWorkspace}
            user={user}
            onToggle={() => setWorkbenchOpen((value) => !value)}
            onGenerate={() => setDialog('generate')}
            onSchedule={() => setDialog('schedule')}
            onContent={() => setDialog('content')}
          />

          <Box sx={{ flex: 1, minWidth: 0, width: '100%' }}>
            <Stack spacing={3}>
              <Stack direction="row" spacing={1} sx={{ display: { xs: 'flex', lg: 'none' }, overflowX: 'auto', pb: 0.5 }}>
                {visibleNavItems.map((item) => (
                  <Button
                    key={item.key}
                    startIcon={item.icon}
                    variant={activeView === item.key ? 'contained' : 'outlined'}
                    onClick={() => setActiveView(item.key)}
                    sx={{ whiteSpace: 'nowrap' }}
                  >
                    {item.label}
                  </Button>
                ))}
              </Stack>

              {error && <Alert severity="error">{error}</Alert>}
              {loading && (
                <Stack direction="row" alignItems="center" spacing={1.5}>
                  <CircularProgress size={22} />
                  <Typography color="text.secondary">正在加载工作区数据</Typography>
                </Stack>
              )}
              {!loading && workspace && currentWorkspace && (
                <ActiveView
                  view={activeView}
                  token={token}
                  workspaceId={workspaceId}
                  workspace={workspace}
                  currentWorkspace={currentWorkspace}
                  openDialog={setDialog}
                  onChanged={refresh}
                  onLoginMediaAccount={(accountId) => {
                    setSelectedMediaAccountId(accountId);
                    setDialog('mediaAccountLogin');
                  }}
                  onPreparePublish={(contentId) => {
                    setSelectedContentId(contentId);
                    setDialog('publishPrepare');
                  }}
                />
              )}
            </Stack>
          </Box>
        </Box>
      </Container>

      {workspace && (
        <WorkspaceDialogs
          dialog={dialog}
          token={token}
          workspaceId={workspaceId}
          data={workspace}
          selectedContentId={selectedContentId}
          selectedMediaAccountId={selectedMediaAccountId}
          onClose={() => {
            setDialog(null);
            setSelectedMediaAccountId('');
          }}
          onCreated={(nextView) => {
            setDialog(null);
            setSelectedMediaAccountId('');
            if (nextView) {
              setActiveView(nextView);
            }
            refresh();
          }}
          onGeneratedTrace={(trace) => {
            setGenerationTrace(trace);
            setTraceDrawerOpen(true);
          }}
        />
      )}
      <GenerationTraceDrawer open={traceDrawerOpen} trace={generationTrace} onClose={() => setTraceDrawerOpen(false)} />
    </Box>
  );
}

function LoginView({ onLogin }: { onLogin: (result: Awaited<ReturnType<typeof login>>) => void }) {
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

function OnboardingView({
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
      <Container maxWidth="md">
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
      </Container>
    </Box>
  );
}

function WorkspaceWorkbenchPanel({
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
              <Button startIcon={<PsychologyAltOutlinedIcon />} variant="outlined" onClick={onGenerate} fullWidth>
                关键词生成
              </Button>
              <Button startIcon={<ScheduleOutlinedIcon />} variant="outlined" onClick={onSchedule} fullWidth>
                新建计划
              </Button>
              <Button startIcon={<AddIcon />} variant="contained" onClick={onContent} fullWidth>
                新建内容
              </Button>
            </Stack>
          </>
        ) : (
          <Stack spacing={1} alignItems="center">
            <Tooltip title="关键词生成">
              <IconButton size="small" onClick={onGenerate} aria-label="关键词生成">
                <PsychologyAltOutlinedIcon fontSize="small" />
              </IconButton>
            </Tooltip>
            <Tooltip title="新建计划">
              <IconButton size="small" onClick={onSchedule} aria-label="新建计划">
                <ScheduleOutlinedIcon fontSize="small" />
              </IconButton>
            </Tooltip>
            <Tooltip title="新建内容">
              <IconButton size="small" color="primary" onClick={onContent} aria-label="新建内容">
                <AddIcon fontSize="small" />
              </IconButton>
            </Tooltip>
          </Stack>
        )}
      </Stack>
    </Paper>
  );
}

function ActiveView({
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
            action={<Button size="small" startIcon={<AddIcon />} onClick={() => openDialog('knowledgeItem')}>新增条目</Button>}
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
            <Button size="small" startIcon={<AddIcon />} onClick={() => openDialog('knowledgeBase')}>
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
            <Button startIcon={<AddIcon />} variant="contained" onClick={() => openDialog('knowledgeItem')}>
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
        action={<Button startIcon={<AddIcon />} variant="contained" onClick={() => openDialog('mediaAccount')}>绑定账号</Button>}
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
      action={<Button startIcon={<PsychologyAltOutlinedIcon />} variant="contained" onClick={() => openDialog('generate')}>开始生成</Button>}
    >
      <Stack spacing={1.5}>
        <Typography color="text.secondary">
          生成时只接收关键词和知识库包上下文，内容类型由系统模板控制写作技能、发布格式和结构化输出边界，结果先保存为草稿。
        </Typography>
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
      <ContentTable contents={contents} onPreparePublish={onPreparePublish} />
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
      <JobsTable data={data} />
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

function WorkspaceDialogs({
  dialog,
  token,
  workspaceId,
  data,
  selectedContentId,
  selectedMediaAccountId,
  onClose,
  onCreated,
  onGeneratedTrace,
}: {
  dialog: DialogKey;
  token: string;
  workspaceId: string;
  data: WorkspaceData;
  selectedContentId: string;
  selectedMediaAccountId: string;
  onClose: () => void;
  onCreated: (nextView?: ViewKey) => void;
  onGeneratedTrace: (trace: GenerationTrace) => void;
}) {
  const selectedMediaAccount = data.mediaAccounts.find((account) => account.id === selectedMediaAccountId) ?? null;

  return (
    <>
      <KnowledgeBaseDialog
        open={dialog === 'knowledgeBase'}
        token={token}
        workspaceId={workspaceId}
        onClose={onClose}
        onCreated={() => onCreated('knowledge')}
      />
      <KnowledgeItemDialog
        open={dialog === 'knowledgeItem'}
        token={token}
        workspaceId={workspaceId}
        bases={data.knowledgeBases}
        user={data.user}
        onClose={onClose}
        onCreated={() => onCreated('knowledge')}
      />
      <MediaAccountDialog
        open={dialog === 'mediaAccount'}
        token={token}
        workspaceId={workspaceId}
        platforms={data.mediaPlatforms}
        onClose={onClose}
        onCreated={() => onCreated('accounts')}
      />
      <MediaAccountLoginDialog
        open={dialog === 'mediaAccountLogin'}
        token={token}
        workspaceId={workspaceId}
        account={selectedMediaAccount}
        platform={selectedMediaAccount ? data.mediaPlatforms.find((platform) => platform.id === selectedMediaAccount.platformId) ?? null : null}
        onClose={onClose}
        onCreated={() => onCreated('accounts')}
      />
      <ContentDialog
        open={dialog === 'content'}
        token={token}
        workspaceId={workspaceId}
        bases={data.knowledgeBases}
        onClose={onClose}
        onCreated={() => onCreated('contents')}
      />
      <GenerateDialog
        open={dialog === 'generate'}
        token={token}
        workspaceId={workspaceId}
        bases={data.knowledgeBases}
        user={data.user}
        onClose={onClose}
        onCreated={() => onCreated('contents')}
        onGeneratedTrace={onGeneratedTrace}
      />
      <ScheduleDialog
        open={dialog === 'schedule'}
        token={token}
        workspaceId={workspaceId}
        contents={data.contents}
        accounts={data.mediaAccounts}
        platforms={data.mediaPlatforms}
        onClose={onClose}
        onCreated={() => onCreated('schedules')}
      />
      <PublishPrepareDialog
        open={dialog === 'publishPrepare'}
        token={token}
        workspaceId={workspaceId}
        data={data}
        selectedContentId={selectedContentId}
        onClose={onClose}
        onCreated={() => onCreated('jobs')}
      />
    </>
  );
}

function KnowledgeBaseDialog(props: DialogBaseProps) {
  const [name, setName] = useState('');
  const [description, setDescription] = useState('');
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const submit = async () => {
    if (!name.trim()) {
      setError('请填写知识库名称');
      return;
    }
    setSubmitting(true);
    setError(null);
    try {
      await createKnowledgeBase(props.token, props.workspaceId, { name: name.trim(), description: description.trim() });
      setName('');
      setDescription('');
      props.onCreated();
    } catch (err) {
      setError(err instanceof Error ? err.message : '创建失败');
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <FormDialog title="新建知识库" open={props.open} error={error} submitting={submitting} onClose={props.onClose} onSubmit={submit}>
      <TextField label="名称" value={name} onChange={(event) => setName(event.target.value)} fullWidth required />
      <TextField label="描述" value={description} onChange={(event) => setDescription(event.target.value)} fullWidth multiline minRows={3} />
    </FormDialog>
  );
}

function KnowledgeItemDialog({
  bases,
  user,
  ...props
}: DialogBaseProps & {
  bases: KnowledgeBase[];
  user: User;
}) {
  const [knowledgeBaseIds, setKnowledgeBaseIds] = useState<string[]>([]);
  const [type, setType] = useState('brand');
  const [title, setTitle] = useState('');
  const [content, setContent] = useState('');
  const [submitting, setSubmitting] = useState(false);
  const [formatting, setFormatting] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const isFormatAvailable = user.subscriptionTier === 'vip' && user.subscriptionStatus === 'active';

  useEffect(() => {
    if (props.open) {
      setKnowledgeBaseIds(bases[0]?.id ? [bases[0].id] : []);
      setError(null);
    }
  }, [bases, props.open]);

  const submit = async () => {
    if (knowledgeBaseIds.length === 0 || !title.trim() || !content.trim()) {
      setError('请选择知识库并填写标题和内容');
      return;
    }
    setSubmitting(true);
    setError(null);
    try {
      await createKnowledgeItem(props.token, props.workspaceId, {
        knowledgeBaseIds,
        type,
        title: title.trim(),
        content: content.trim(),
      });
      setTitle('');
      setContent('');
      props.onCreated();
    } catch (err) {
      setError(err instanceof Error ? err.message : '创建失败');
    } finally {
      setSubmitting(false);
    }
  };

  const formatContent = async () => {
    if (!isFormatAvailable) {
      setError('格式化是 VIP 功能，请升级订阅后使用');
      return;
    }
    if (!content.trim()) {
      setError('请先填写内容，再使用格式化');
      return;
    }
    setFormatting(true);
    setError(null);
    try {
      const response = await formatKnowledgeContent(props.token, props.workspaceId, {
        type,
        title: title.trim(),
        content: content.trim(),
      });
      setContent(response.content);
    } catch (err) {
      setError(err instanceof Error ? err.message : '格式化失败');
    } finally {
      setFormatting(false);
    }
  };

  return (
    <FormDialog title="新增知识条目" open={props.open} error={error} submitting={submitting || formatting} onClose={props.onClose} onSubmit={submit}>
      <Alert severity="info">
        内容建议使用 Markdown 结构，例如标题、要点、适用场景、事实边界、禁用表达和待补充事项。结构越清晰，系统检索知识片段和 AI 生成草稿时越容易准确引用事实、减少误读和编造。散乱文本可以先填写，再用 VIP 格式化整理回内容栏。
      </Alert>
      <FormControl fullWidth disabled={bases.length === 0}>
        <InputLabel>知识库包</InputLabel>
        <Select
          multiple
          label="知识库包"
          value={knowledgeBaseIds}
          onChange={(event) => {
            const value = event.target.value;
            setKnowledgeBaseIds(typeof value === 'string' ? value.split(',') : value);
          }}
          renderValue={(selected) => knowledgeBaseNames(bases, selected)}
        >
          {bases.map((base) => (
            <MenuItem key={base.id} value={base.id}>
              <Checkbox checked={knowledgeBaseIds.includes(base.id)} />
              <ListItemText primary={base.name} />
            </MenuItem>
          ))}
        </Select>
      </FormControl>
      <SelectField
        label="类型"
        value={type}
        onChange={setType}
        items={[
          { value: 'brand', label: '品牌资料' },
          { value: 'product', label: '产品资料' },
          { value: 'case', label: '案例' },
          { value: 'faq', label: 'FAQ' },
          { value: 'style', label: '风格指南' },
          { value: 'audience', label: '目标受众' },
        ]}
      />
      <TextField label="标题" value={title} onChange={(event) => setTitle(event.target.value)} fullWidth required />
      <Stack direction={{ xs: 'column', sm: 'row' }} spacing={1} alignItems={{ xs: 'stretch', sm: 'center' }} justifyContent="space-between">
        <Typography fontWeight={700}>内容</Typography>
        <Tooltip title={isFormatAvailable ? '调用 AI 把散乱文本整理成适合检索的 Markdown' : 'VIP 用户可用 AI 格式化'}>
          <span>
            <Button
              variant="outlined"
              size="small"
              onClick={formatContent}
              disabled={formatting || submitting || !isFormatAvailable}
              startIcon={<Box component="img" src={vipGoldIconUrl} alt="VIP" sx={{ width: 30, height: 15, objectFit: 'contain' }} />}
            >
              {formatting ? '格式化中' : '格式化'}
            </Button>
          </span>
        </Tooltip>
      </Stack>
      <TextField label="内容" value={content} onChange={(event) => setContent(event.target.value)} fullWidth multiline minRows={4} required />
    </FormDialog>
  );
}

function MediaAccountDialog({
  platforms,
  ...props
}: DialogBaseProps & {
  platforms: MediaPlatform[];
}) {
  const enabledPlatforms = useMemo(() => platforms.filter((platform) => platform.enabled), [platforms]);
  const [platformId, setPlatformId] = useState('');
  const [name, setName] = useState('');
  const [externalId, setExternalId] = useState('');
  const [loginMethod, setLoginMethod] = useState('phone');
  const [phoneNumber, setPhoneNumber] = useState('');
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const selectedPlatform = enabledPlatforms.find((platform) => platform.id === platformId);
  const requiresPhone = selectedPlatform?.credentialFields.includes('phoneNumber') ?? false;
  const requiresQR = selectedPlatform?.credentialFields.includes('qrLogin') ?? false;

  useEffect(() => {
    if (props.open) {
      const defaultPlatform = enabledPlatforms.find((platform) => platform.type === 'xiaohongshu') ?? enabledPlatforms.find((platform) => platform.credentialFields.includes('phoneNumber')) ?? enabledPlatforms[0];
      setPlatformId(defaultPlatform?.id ?? '');
      setLoginMethod(defaultPlatform?.credentialFields.includes('qrLogin') ? 'qr' : defaultPlatform?.credentialFields.includes('phoneNumber') ? 'phone' : 'manual');
      setError(null);
    }
  }, [enabledPlatforms, props.open]);

  const updatePlatform = (value: string) => {
    const nextPlatform = enabledPlatforms.find((platform) => platform.id === value);
    setPlatformId(value);
    setLoginMethod(nextPlatform?.credentialFields.includes('qrLogin') ? 'qr' : nextPlatform?.credentialFields.includes('phoneNumber') ? 'phone' : 'manual');
  };

  const submit = async () => {
    if (!platformId || !name.trim()) {
      setError('请选择平台并填写账号名称');
      return;
    }
    if (loginMethod === 'phone' && !phoneNumber.trim()) {
      setError('请填写手机号');
      return;
    }
    setSubmitting(true);
    setError(null);
    try {
      await createMediaAccount(props.token, props.workspaceId, {
        platformId,
        name: name.trim(),
        externalId: externalId.trim(),
        loginMethod,
        phoneNumber: phoneNumber.trim(),
      });
      setName('');
      setExternalId('');
      setPhoneNumber('');
      props.onCreated();
    } catch (err) {
      setError(err instanceof Error ? err.message : '创建失败');
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <FormDialog title="绑定媒体账号" open={props.open} error={error} submitting={submitting} onClose={props.onClose} onSubmit={submit}>
      <SelectField label="媒体平台" value={platformId} onChange={updatePlatform} items={enabledPlatforms.map((platform) => ({ value: platform.id, label: platform.name }))} />
      <TextField label="账号名称" value={name} onChange={(event) => setName(event.target.value)} fullWidth required />
      <TextField label="外部账号标识" value={externalId} onChange={(event) => setExternalId(event.target.value)} fullWidth />
      {requiresQR ? (
        <Alert severity="info">小红书绑定将由服务端浏览器打开二维码登录页，扫码确认后保存服务端浏览器会话。</Alert>
      ) : requiresPhone ? (
        <Alert severity="info">本机发布平台仅支持手机号验证码登录。</Alert>
      ) : (
        <SelectField
          label="登录方式"
          value={loginMethod}
          onChange={setLoginMethod}
          items={[
            { value: 'phone', label: '手机号登录' },
            { value: 'manual', label: '手动授权' },
          ]}
        />
      )}
      {!requiresQR && (loginMethod === 'phone' || requiresPhone) && (
        <TextField
          label="登录手机号"
          value={phoneNumber}
          onChange={(event) => setPhoneNumber(event.target.value)}
          fullWidth
          required={loginMethod === 'phone'}
          placeholder="+86 13800000000"
        />
      )}
    </FormDialog>
  );
}

function MediaAccountLoginDialog({
  account,
  platform,
  ...props
}: DialogBaseProps & {
  account: MediaAccount | null;
  platform: MediaPlatform | null;
}) {
  const [sessionId, setSessionId] = useState('');
  const [qrScreenshotData, setQRScreenshotData] = useState('');
  const [qrLoginUrl, setQRLoginUrl] = useState('');
  const [stateFile, setStateFile] = useState('');
  const [sessionStarted, setSessionStarted] = useState(false);
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    if (props.open) {
      setSessionId('');
      setQRScreenshotData('');
      setQRLoginUrl('');
      setStateFile('');
      setSessionStarted(false);
      setError(null);
    }
  }, [account, props.open]);

  const startLogin = async () => {
    if (!account) {
      setError('请选择媒体账号');
      return;
    }
    setSubmitting(true);
    setError(null);
    try {
      const response = await startMediaAccountBrowserLogin(props.token, props.workspaceId, account.id, {});
      setSessionStarted(true);
      setSessionId(response.sessionId);
      setQRScreenshotData(response.qrScreenshotData);
      setQRLoginUrl(response.qrLoginUrl);
      setStateFile(response.stateFile);
    } catch (err) {
      setError(err instanceof Error ? err.message : '启动二维码登录失败');
    } finally {
      setSubmitting(false);
    }
  };

  const completeLogin = async () => {
    if (!account) {
      setError('请选择媒体账号');
      return;
    }
    if (!sessionId) {
      setError('请先启动二维码登录会话');
      return;
    }
    setSubmitting(true);
    setError(null);
    try {
      await completeMediaAccountBrowserLogin(props.token, props.workspaceId, account.id, {
        sessionId,
      });
      props.onCreated();
    } catch (err) {
      const message = err instanceof Error ? err.message : '登录绑定失败';
      setError(stateFile ? `${message}。可查看状态文件：${stateFile}` : message);
    } finally {
      setSubmitting(false);
    }
  };

  const canLogin = account && supportsBrowserLogin(platform?.type) && account.loginMethod === 'qr';

  return (
    <Dialog open={props.open} onClose={submitting ? undefined : props.onClose} fullWidth maxWidth="sm">
      <DialogTitle>{platform?.type === 'xiaohongshu' ? '小红书二维码登录绑定' : '浏览器登录绑定'}</DialogTitle>
      <DialogContent>
        <Stack spacing={2} sx={{ pt: 1 }}>
          {error && <Alert severity="error">{error}</Alert>}
          {!canLogin && <Alert severity="warning">当前账号不支持服务端浏览器二维码登录。</Alert>}
          {canLogin && platform?.type === 'xiaohongshu' && (
            <Alert severity="info">
              点击生成二维码后，请使用小红书 App 扫码并确认登录。确认后返回这里完成绑定。
            </Alert>
          )}
          {account && (
            <Paper variant="outlined" sx={{ p: 1.5, borderRadius: 1 }}>
              <InfoRow label="媒体账号" value={account.name} />
              <InfoRow label="平台" value={platform?.name ?? account.platformId} />
              <InfoRow label="状态" value={mediaAccountStatusLabel(account.status)} />
            </Paper>
          )}
          <Stack direction={{ xs: 'column', sm: 'row' }} spacing={1}>
            <Button variant="outlined" onClick={startLogin} disabled={!canLogin || submitting} sx={{ minWidth: 160 }}>
              生成二维码
            </Button>
            <Button variant="contained" onClick={completeLogin} disabled={!canLogin || submitting || !sessionStarted}>
              我已扫码确认
            </Button>
          </Stack>
          {qrScreenshotData && (
            <Paper variant="outlined" sx={{ p: 2, borderRadius: 1, display: 'grid', justifyItems: 'center', gap: 1 }}>
              <Box component="img" src={qrScreenshotData} alt="小红书登录二维码" sx={{ width: 240, height: 240 }} />
              <Typography variant="caption" color="text.secondary" sx={{ overflowWrap: 'anywhere' }}>
                {sessionId}
              </Typography>
              {stateFile && (
                <Typography variant="caption" color="text.secondary" sx={{ overflowWrap: 'anywhere' }}>
                  {stateFile}
                </Typography>
              )}
              <Link href={qrLoginUrl} target="_blank" rel="noreferrer" variant="body2">
                打开登录链接
              </Link>
            </Paper>
          )}
        </Stack>
      </DialogContent>
      <DialogActions>
        <Button onClick={props.onClose} disabled={submitting}>
          取消
        </Button>
        <Button onClick={completeLogin} disabled={!canLogin || submitting || !sessionStarted} variant="contained">
          完成绑定
        </Button>
      </DialogActions>
    </Dialog>
  );
}

function ContentDialog({
  bases,
  ...props
}: DialogBaseProps & {
  bases: KnowledgeBase[];
}) {
  const [title, setTitle] = useState('');
  const [summary, setSummary] = useState('');
  const [body, setBody] = useState('');
  const [keywords, setKeywords] = useState('');
  const [knowledgeBaseId, setKnowledgeBaseId] = useState('');
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    if (props.open) {
      setKnowledgeBaseId(bases[0]?.id ?? '');
      setError(null);
    }
  }, [bases, props.open]);

  const submit = async () => {
    if (!title.trim()) {
      setError('请填写标题');
      return;
    }
    setSubmitting(true);
    setError(null);
    try {
      await createContent(props.token, props.workspaceId, {
        title: title.trim(),
        summary: summary.trim(),
        body: body.trim(),
        author: 'Current User',
        knowledgeBaseId,
        keywords: splitKeywords(keywords),
      });
      setTitle('');
      setSummary('');
      setBody('');
      setKeywords('');
      props.onCreated();
    } catch (err) {
      setError(err instanceof Error ? err.message : '创建失败');
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <FormDialog title="新建内容" open={props.open} error={error} submitting={submitting} onClose={props.onClose} onSubmit={submit}>
      <SelectField label="关联知识库" value={knowledgeBaseId} onChange={setKnowledgeBaseId} items={bases.map((base) => ({ value: base.id, label: base.name }))} />
      <TextField label="标题" value={title} onChange={(event) => setTitle(event.target.value)} fullWidth required />
      <TextField label="摘要" value={summary} onChange={(event) => setSummary(event.target.value)} fullWidth />
      <TextField label="关键词" value={keywords} onChange={(event) => setKeywords(event.target.value)} fullWidth />
      <TextField label="正文" value={body} onChange={(event) => setBody(event.target.value)} fullWidth multiline minRows={5} />
    </FormDialog>
  );
}

function GenerateDialog({
  bases,
  user,
  onGeneratedTrace,
  ...props
}: DialogBaseProps & {
  bases: KnowledgeBase[];
  user: User;
  onGeneratedTrace: (trace: GenerationTrace) => void;
}) {
  const [keywords, setKeywords] = useState('内容营销, 增长');
  const [contentType, setContentType] = useState('xiaohongshu_long_article');
  const [knowledgeBaseIds, setKnowledgeBaseIds] = useState<string[]>([]);
  const [submitting, setSubmitting] = useState(false);
  const [formatting, setFormatting] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const keywordItems = useMemo(() => splitKeywords(keywords), [keywords]);
  const isFormatAvailable = user.subscriptionTier === 'vip' && user.subscriptionStatus === 'active';

  useEffect(() => {
    if (props.open) {
      setKnowledgeBaseIds(bases.map((base) => base.id));
      setError(null);
    }
  }, [bases, props.open]);

  const formatKeywords = async () => {
    if (!isFormatAvailable) {
      setError('格式化是 VIP 功能，请升级订阅后使用');
      return;
    }
    if (!keywords.trim()) {
      setError('请先填写关键词和素材');
      return;
    }
    setFormatting(true);
    setError(null);
    try {
      const response = await formatKnowledgeContent(props.token, props.workspaceId, {
        type: 'generation_keywords',
        title: '发布内容关键词',
        content: keywords.trim(),
      });
      setKeywords(response.content);
    } catch (err) {
      setError(err instanceof Error ? err.message : '格式化失败');
    } finally {
      setFormatting(false);
    }
  };

  const submit = async () => {
    const values = keywordItems;
    if (values.length === 0) {
      setError('请至少填写一个关键词');
      return;
    }
    setSubmitting(true);
    setError(null);
    try {
      const response = await generateContent(props.token, props.workspaceId, { keywords: values, contentType, knowledgeBaseIds, publishFormatId: contentType });
      onGeneratedTrace(response.trace);
      props.onCreated();
    } catch (err) {
      setError(err instanceof Error ? err.message : '生成失败');
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <Dialog open={props.open} onClose={submitting || formatting ? undefined : props.onClose} fullWidth maxWidth="md">
      <DialogTitle>关键词生成发布内容</DialogTitle>
      <DialogContent>
        <Stack spacing={2} sx={{ pt: 1 }}>
          {error && <Alert severity="error">{error}</Alert>}
          <FormControl fullWidth>
            <InputLabel>知识库包上下文</InputLabel>
            <Select
              multiple
              label="知识库包上下文"
              value={knowledgeBaseIds}
              onChange={(event) => {
                const value = event.target.value;
                setKnowledgeBaseIds(typeof value === 'string' ? value.split(',') : value);
              }}
              renderValue={(selected) => (selected.length === 0 ? '全部知识库包' : knowledgeBaseNames(bases, selected))}
            >
              {bases.map((base) => (
                <MenuItem key={base.id} value={base.id}>
                  <Checkbox checked={knowledgeBaseIds.includes(base.id)} />
                  <ListItemText primary={base.name} />
                </MenuItem>
              ))}
            </Select>
          </FormControl>
          <SelectField
            label="内容类型"
            value={contentType}
            onChange={setContentType}
            items={contentTypeOptions}
          />
          <Stack direction={{ xs: 'column', sm: 'row' }} spacing={1} alignItems={{ xs: 'stretch', sm: 'center' }} justifyContent="space-between">
            <Typography fontWeight={700}>关键词与素材</Typography>
            <Tooltip title={isFormatAvailable ? '调用 AI 把散乱关键词整理成可检索的文本条目' : 'VIP 用户可用 AI 格式化'}>
              <span>
                <Button
                  variant="outlined"
                  size="small"
                  onClick={formatKeywords}
                  disabled={formatting || submitting || !isFormatAvailable}
                  startIcon={<Box component="img" src={vipGoldIconUrl} alt="VIP" sx={{ width: 30, height: 15, objectFit: 'contain' }} />}
                >
                  {formatting ? '格式化中' : '格式化'}
                </Button>
              </span>
            </Tooltip>
          </Stack>
          <TextField
            label="关键词"
            value={keywords}
            onChange={(event) => setKeywords(event.target.value)}
            fullWidth
            required
            multiline
            minRows={4}
            helperText="每行、逗号或分号分隔一个关键词/素材点"
          />
          <Paper variant="outlined" sx={{ p: 1.5, borderRadius: 1 }}>
            <Stack spacing={1}>
              <Typography variant="body2" color="text.secondary">
                已识别关键词
              </Typography>
              {keywordItems.length === 0 ? (
                <Typography color="text.secondary">暂无关键词</Typography>
              ) : (
                <Stack spacing={0.75}>
                  {keywordItems.map((item) => (
                    <Typography key={item} variant="body2" sx={{ overflowWrap: 'anywhere' }}>
                      {item}
                    </Typography>
                  ))}
                </Stack>
              )}
            </Stack>
          </Paper>
        </Stack>
      </DialogContent>
      <DialogActions>
        <Button onClick={props.onClose} disabled={submitting || formatting}>
          取消
        </Button>
        <Button onClick={submit} disabled={submitting || formatting} variant="contained">
          {submitting ? '生成中' : '确认'}
        </Button>
      </DialogActions>
    </Dialog>
  );
}

function ScheduleDialog({
  contents,
  accounts,
  platforms,
  ...props
}: DialogBaseProps & {
  contents: Content[];
  accounts: MediaAccount[];
  platforms: MediaPlatform[];
}) {
  const [name, setName] = useState('');
  const [contentId, setContentId] = useState('');
  const [mediaAccountId, setMediaAccountId] = useState('');
  const [frequency, setFrequency] = useState<PublishScheduleFrequency>('once');
  const [nextRunAt, setNextRunAt] = useState(defaultScheduleInputValue());
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    if (props.open) {
      setContentId(contents[0]?.id ?? '');
      setMediaAccountId(accounts[0]?.id ?? '');
      setNextRunAt(defaultScheduleInputValue());
      setError(null);
    }
  }, [accounts, contents, props.open]);

  const submit = async () => {
    if (!contentId || !mediaAccountId || !nextRunAt) {
      setError('请选择内容、媒体账号和计划时间');
      return;
    }
    setSubmitting(true);
    setError(null);
    try {
      await createPublishSchedule(props.token, props.workspaceId, {
        name: name.trim(),
        contentId,
        mediaAccountId,
        frequency,
        nextRunAt: new Date(nextRunAt).toISOString(),
      });
      setName('');
      props.onCreated();
    } catch (err) {
      setError(err instanceof Error ? err.message : '创建失败');
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <FormDialog title="新建发布计划" open={props.open} error={error} submitting={submitting} onClose={props.onClose} onSubmit={submit}>
      <TextField label="计划名称" value={name} onChange={(event) => setName(event.target.value)} fullWidth />
      <SelectField label="内容" value={contentId} onChange={setContentId} items={contents.map((content) => ({ value: content.id, label: content.title }))} />
      <SelectField
        label="媒体账号"
        value={mediaAccountId}
        onChange={setMediaAccountId}
        items={accounts.map((account) => ({ value: account.id, label: `${account.name} / ${platformName(platforms, account.platformId)}` }))}
      />
      <SelectField
        label="频率"
        value={frequency}
        onChange={(value) => setFrequency(value as PublishScheduleFrequency)}
        items={[
          { value: 'once', label: '一次性' },
          { value: 'daily', label: '每天' },
          { value: 'weekly', label: '每周' },
          { value: 'monthly', label: '每月' },
        ]}
      />
      <TextField
        label="下次执行"
        type="datetime-local"
        value={nextRunAt}
        onChange={(event) => setNextRunAt(event.target.value)}
        InputLabelProps={{ shrink: true }}
        fullWidth
      />
    </FormDialog>
  );
}

function PublishPrepareDialog({
  data,
  selectedContentId,
  ...props
}: DialogBaseProps & {
  data: WorkspaceData;
  selectedContentId: string;
}) {
  const xiaohongshuAccounts = useMemo(
    () => data.mediaAccounts.filter((account) => platformType(data.mediaPlatforms, account.platformId) === 'xiaohongshu' && account.status === 'connected'),
    [data.mediaAccounts, data.mediaPlatforms],
  );
  const [contentId, setContentId] = useState('');
  const [mediaAccountId, setMediaAccountId] = useState('');
  const [prepared, setPrepared] = useState<PreparePublishResponse | null>(null);
  const [publishResult, setPublishResult] = useState<PreparePublishResponse['publishResult']>(undefined);
  const [publishTitle, setPublishTitle] = useState('');
  const [publishBody, setPublishBody] = useState('');
  const [copiedLabel, setCopiedLabel] = useState('');
  const [submitting, setSubmitting] = useState(false);
  const [running, setRunning] = useState(false);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    if (props.open) {
      const requestedContent = data.contents.find((content) => content.id === selectedContentId);
      setContentId(requestedContent?.id ?? data.contents[0]?.id ?? '');
      setMediaAccountId(xiaohongshuAccounts[0]?.id ?? '');
      setPrepared(null);
      setPublishResult(undefined);
      setPublishTitle('');
      setPublishBody('');
      setCopiedLabel('');
      setError(null);
    }
  }, [data.contents, props.open, selectedContentId, xiaohongshuAccounts]);

  const selectedContent = data.contents.find((content) => content.id === contentId);

  const handlePrepare = async () => {
    if (!contentId || !mediaAccountId) {
      setError('请选择内容和小红书账号');
      return;
    }
    setSubmitting(true);
    setError(null);
    try {
      const result = await preparePublish(props.token, props.workspaceId, { contentId, mediaAccountId, publishFormatId: 'xiaohongshu_long_article' });
      setPrepared(result);
      setPublishResult(result.publishResult);
      setPublishTitle(result.preparedPost.title);
      setPublishBody(result.preparedPost.body);
    } catch (err) {
      setError(err instanceof Error ? err.message : '发布准备失败');
    } finally {
      setSubmitting(false);
    }
  };

  const handleRun = async () => {
    if (!prepared) {
      return;
    }
    setRunning(true);
    setError(null);
    try {
      const preparedPost: PreparedPost = {
        ...prepared.preparedPost,
        title: publishTitle.trim(),
        body: publishBody.trim(),
        characterCount: publishBody.trim().length,
        publishFormatId: 'xiaohongshu_long_article',
        publishMode: 'long_article',
      };
      const result = await runPublishJob(props.token, props.workspaceId, prepared.job.id, {
        preparedPost,
      });
      setPrepared({ job: result.job, preparedPost: result.preparedPost, publishResult: result.publishResult });
      setPublishResult(result.publishResult);
      props.onCreated();
    } catch (err) {
      setError(err instanceof Error ? err.message : '小红书浏览器发布失败');
    } finally {
      setRunning(false);
    }
  };

  const handleCopy = async (block: { label: string; value: string }) => {
    try {
      await navigator.clipboard.writeText(block.value);
      setCopiedLabel(block.label);
    } catch {
      setCopiedLabel('');
      setError('复制失败，请手动选中文案复制');
    }
  };

  const busy = submitting || running;
  const previewPost = prepared
    ? {
        ...prepared.preparedPost,
        title: publishTitle,
        body: publishBody,
        characterCount: publishBody.length,
        copyBlocks: [
          { label: '长文标题', value: publishTitle },
          { label: '长文正文', value: publishBody },
        ],
      }
    : null;

  return (
    <Dialog open={props.open} onClose={busy ? undefined : props.onClose} fullWidth maxWidth="md">
      <DialogTitle>发布小红书长文</DialogTitle>
      <DialogContent>
        <Stack spacing={2} sx={{ pt: 1 }}>
          {error && <Alert severity="error">{error}</Alert>}
          {xiaohongshuAccounts.length === 0 && (
            <Alert severity="warning">当前工作区还没有绑定小红书账号。</Alert>
          )}
          <Grid container spacing={2}>
            <Grid size={{ xs: 12, md: 7 }}>
              <SelectField
                label="内容"
                value={contentId}
                onChange={(value) => {
                  setContentId(value);
                  setPrepared(null);
                }}
                items={data.contents.map((content) => ({ value: content.id, label: content.title }))}
              />
            </Grid>
            <Grid size={{ xs: 12, md: 5 }}>
              <SelectField
                label="小红书账号"
                value={mediaAccountId}
                onChange={(value) => {
                  setMediaAccountId(value);
                  setPrepared(null);
                }}
                items={xiaohongshuAccounts.map((account) => ({ value: account.id, label: account.name }))}
              />
            </Grid>
          </Grid>

          {selectedContent && (
            <Paper variant="outlined" sx={{ p: 1.5, borderRadius: 1 }}>
              <Stack spacing={0.5}>
                <Typography fontWeight={700}>{selectedContent.title}</Typography>
                <Typography variant="body2" color="text.secondary">
                  {selectedContent.summary || selectedContent.body.slice(0, 120)}
                </Typography>
              </Stack>
            </Paper>
          )}

          {previewPost && <PreparedPostPanel post={previewPost} copiedLabel={copiedLabel} onCopy={handleCopy} />}

          {prepared && (
            <Stack spacing={1.5}>
              <TextField
                label="小红书长文标题"
                value={publishTitle}
                onChange={(event) => setPublishTitle(event.target.value.slice(0, 64))}
                helperText={`${publishTitle.length}/64`}
                fullWidth
              />
              <TextField
                label="小红书长文正文"
                value={publishBody}
                onChange={(event) => setPublishBody(event.target.value)}
                helperText={`${publishBody.length} 字。确认后后台会用已登录浏览器打开小红书长文编辑器并点击发布。`}
                fullWidth
                multiline
                minRows={10}
              />
              {publishResult && (
                <Alert severity={publishResult.status === 'published' ? 'success' : 'info'}>
                  {publishResult.message}
                  {publishResult.externalId ? ` 笔记 ID：${publishResult.externalId}` : ''}
                </Alert>
              )}
            </Stack>
          )}
        </Stack>
      </DialogContent>
      <DialogActions>
        <Button onClick={props.onClose} disabled={busy}>
          取消
        </Button>
        {!prepared && (
          <Button onClick={handlePrepare} disabled={busy || xiaohongshuAccounts.length === 0} variant="contained">
            生成发布包
          </Button>
        )}
        {prepared && (
          <Button onClick={handleRun} disabled={busy || !publishTitle.trim() || !publishBody.trim()} variant="contained">
            确认发布
          </Button>
        )}
      </DialogActions>
    </Dialog>
  );
}

function PreparedPostPanel({
  post,
  copiedLabel,
  onCopy,
}: {
  post: PreparedPost;
  copiedLabel: string;
  onCopy: (block: { label: string; value: string }) => void;
}) {
  return (
    <Stack spacing={2}>
      {post.warnings.map((warning) => (
        <Alert key={warning} severity="info">
          {warning}
        </Alert>
      ))}
      <Grid container spacing={2}>
        <Grid size={{ xs: 12, md: 7 }}>
          <Stack spacing={1.25}>
            {post.copyBlocks.map((block) => (
              <Paper key={block.label} variant="outlined" sx={{ p: 1.5, borderRadius: 1 }}>
                <Stack direction="row" justifyContent="space-between" alignItems="center" spacing={1.5}>
                  <Typography fontWeight={700}>{block.label}</Typography>
                  <Tooltip title={`复制${block.label}`}>
                    <span>
                      <IconButton size="small" onClick={() => onCopy(block)}>
                        <ContentCopyIcon fontSize="small" />
                      </IconButton>
                    </span>
                  </Tooltip>
                </Stack>
                <Typography
                  variant="body2"
                  sx={{ mt: 1, whiteSpace: 'pre-wrap', overflowWrap: 'anywhere', maxHeight: 220, overflowY: 'auto' }}
                >
                  {block.value}
                </Typography>
                {copiedLabel === block.label && (
                  <Typography variant="caption" color="success.main">
                    已复制
                  </Typography>
                )}
              </Paper>
            ))}
          </Stack>
        </Grid>
        <Grid size={{ xs: 12, md: 5 }}>
          <Paper variant="outlined" sx={{ p: 1.5, borderRadius: 1 }}>
            <Stack spacing={1}>
              <Typography fontWeight={700}>发布检查</Typography>
              <List dense disablePadding>
                {post.checklist.map((item) => (
                  <ListItem key={item} disableGutters>
                    <ListItemText primary={item} />
                  </ListItem>
                ))}
              </List>
              <Divider />
              <InfoRow label="平台" value={post.platformName} />
              <InfoRow label="字数" value={`${post.characterCount}`} />
              <InfoRow label="话题" value={post.hashtags.join(' ') || '未生成'} />
            </Stack>
          </Paper>
        </Grid>
      </Grid>
    </Stack>
  );
}

function GenerationTraceDrawer({
  open,
  trace,
  onClose,
}: {
  open: boolean;
  trace: GenerationTrace | null;
  onClose: () => void;
}) {
  return (
    <Drawer anchor="right" open={open} onClose={onClose}>
      <Box sx={{ width: { xs: '100vw', sm: 420 }, maxWidth: '100vw', p: 2 }}>
        <Stack spacing={2}>
          <Stack direction="row" alignItems="center" justifyContent="space-between" spacing={1}>
            <Box>
              <Typography variant="h2">思考过程</Typography>
              <Typography variant="body2" color="text.secondary">
                {trace ? `${trace.subscriptionTier || 'free'} 链路 / ${trace.steps.length} 个阶段` : '暂无生成记录'}
              </Typography>
            </Box>
            <IconButton onClick={onClose} aria-label="关闭思考过程">
              <CloseIcon />
            </IconButton>
          </Stack>
          {trace?.warnings?.map((warning) => (
            <Alert key={warning} severity="warning">
              {warning}
            </Alert>
          ))}
          {!trace || trace.steps.length === 0 ? (
            <Typography color="text.secondary">生成后会显示本次使用的链路、知识检索、计划、校验和重写结果。</Typography>
          ) : (
            <Stack spacing={1}>
              {trace.steps.map((step, index) => (
                <Accordion key={`${step.id}-${index}`} defaultExpanded={index < 2} disableGutters>
                  <AccordionSummary expandIcon={<ExpandMoreIcon />}>
                    <Stack direction="row" spacing={1} alignItems="center" sx={{ minWidth: 0 }}>
                      <Chip size="small" label={traceStepStatusLabel(step.status)} color={traceStepStatusColor(step.status)} />
                      <Typography fontWeight={700} sx={{ overflowWrap: 'anywhere' }}>
                        {step.label}
                      </Typography>
                    </Stack>
                  </AccordionSummary>
                  <AccordionDetails>
                    <Stack spacing={1}>
                      <Typography variant="body2">{step.summary}</Typography>
                      {step.details.map((item) => (
                        <Typography key={item} variant="body2" color="text.secondary" sx={{ overflowWrap: 'anywhere' }}>
                          {item}
                        </Typography>
                      ))}
                      {step.warnings.map((warning) => (
                        <Alert key={warning} severity="warning">
                          {warning}
                        </Alert>
                      ))}
                    </Stack>
                  </AccordionDetails>
                </Accordion>
              ))}
            </Stack>
          )}
        </Stack>
      </Box>
    </Drawer>
  );
}

function traceStepStatusLabel(status: string) {
  if (status === 'succeeded') {
    return '完成';
  }
  if (status === 'failed') {
    return '失败';
  }
  if (status === 'skipped') {
    return '跳过';
  }
  return status;
}

function traceStepStatusColor(status: string): 'default' | 'success' | 'error' | 'warning' {
  if (status === 'succeeded') {
    return 'success';
  }
  if (status === 'failed') {
    return 'error';
  }
  if (status === 'skipped') {
    return 'warning';
  }
  return 'default';
}

type DialogBaseProps = {
  open: boolean;
  token: string;
  workspaceId: string;
  onClose: () => void;
  onCreated: () => void;
};

function FormDialog({
  title,
  open,
  error,
  submitting,
  children,
  onClose,
  onSubmit,
}: {
  title: string;
  open: boolean;
  error: string | null;
  submitting: boolean;
  children: ReactNode;
  onClose: () => void;
  onSubmit: () => void;
}) {
  return (
    <Dialog open={open} onClose={submitting ? undefined : onClose} fullWidth maxWidth="sm">
      <DialogTitle>{title}</DialogTitle>
      <DialogContent>
        <Stack spacing={2} sx={{ pt: 1 }}>
          {error && <Alert severity="error">{error}</Alert>}
          {children}
        </Stack>
      </DialogContent>
      <DialogActions>
        <Button onClick={onClose} disabled={submitting}>
          取消
        </Button>
        <Button onClick={onSubmit} disabled={submitting} variant="contained">
          确认
        </Button>
      </DialogActions>
    </Dialog>
  );
}

function SelectField({
  label,
  value,
  items,
  onChange,
}: {
  label: string;
  value: string;
  items: Array<{ value: string; label: string }>;
  onChange: (value: string) => void;
}) {
  return (
    <FormControl fullWidth disabled={items.length === 0}>
      <InputLabel>{label}</InputLabel>
      <Select label={label} value={value} onChange={(event) => onChange(String(event.target.value))}>
        {items.map((item) => (
          <MenuItem key={item.value} value={item.value}>
            {item.label}
          </MenuItem>
        ))}
      </Select>
    </FormControl>
  );
}

function MetricCard({
  label,
  value,
  helper,
  tone = 'primary',
}: {
  label: string;
  value: number;
  helper: string;
  tone?: 'primary' | 'error';
}) {
  return (
    <Grid size={{ xs: 12, sm: 6, lg: 3 }}>
      <Card>
        <CardContent>
          <Typography variant="body2" color="text.secondary">
            {label}
          </Typography>
          <Typography variant="h1" color={tone === 'error' ? 'error.main' : 'text.primary'} sx={{ mt: 1 }}>
            {value}
          </Typography>
          <Typography variant="body2" color="text.secondary" sx={{ mt: 1 }}>
            {helper}
          </Typography>
        </CardContent>
      </Card>
    </Grid>
  );
}

function Section({ title, action, children }: { title: string; action?: ReactNode; children: ReactNode }) {
  return (
    <Paper elevation={0} sx={{ border: '1px solid', borderColor: 'divider', borderRadius: 2, overflow: 'hidden' }}>
      <Stack direction="row" alignItems="center" justifyContent="space-between" spacing={2} sx={{ px: 2, py: 1.5 }}>
        <Typography variant="h3">{title}</Typography>
        {action}
      </Stack>
      <Divider />
      <Box sx={{ p: 2, overflowX: 'auto' }}>{children}</Box>
    </Paper>
  );
}

function KnowledgeItemsTable({
  items,
  bases,
  selectedIds = [],
  onSelectedIdsChange,
}: {
  items: KnowledgeItem[];
  bases: KnowledgeBase[];
  selectedIds?: string[];
  onSelectedIdsChange?: (ids: string[]) => void;
}) {
  const selectable = Boolean(onSelectedIdsChange);
  const allSelected = selectable && items.length > 0 && items.every((item) => selectedIds.includes(item.id));
  const toggleAll = (checked: boolean) => {
    if (!onSelectedIdsChange) {
      return;
    }
    if (checked) {
      onSelectedIdsChange(uniqueValues([...selectedIds, ...items.map((item) => item.id)]));
    } else {
      onSelectedIdsChange(selectedIds.filter((id) => !items.some((item) => item.id === id)));
    }
  };
  const toggleItem = (itemId: string, checked: boolean) => {
    if (!onSelectedIdsChange) {
      return;
    }
    onSelectedIdsChange(checked ? uniqueValues([...selectedIds, itemId]) : selectedIds.filter((id) => id !== itemId));
  };

  return (
    <Table>
      <TableHead>
        <TableRow>
          {selectable && (
            <TableCell padding="checkbox">
              <Checkbox
                checked={allSelected}
                indeterminate={selectedIds.length > 0 && !allSelected}
                onChange={(event) => toggleAll(event.target.checked)}
              />
            </TableCell>
          )}
          <TableCell>标题</TableCell>
          <TableCell>知识库包</TableCell>
          <TableCell>类型</TableCell>
          <TableCell>状态</TableCell>
          <TableCell>更新时间</TableCell>
        </TableRow>
      </TableHead>
      <TableBody>
        {items.map((item) => (
          <TableRow key={item.id} hover>
            {selectable && (
              <TableCell padding="checkbox">
                <Checkbox
                  checked={selectedIds.includes(item.id)}
                  onChange={(event) => toggleItem(item.id, event.target.checked)}
                />
              </TableCell>
            )}
            <TableCell>
              <Typography fontWeight={700}>{item.title}</Typography>
              <Typography variant="body2" color="text.secondary" sx={{ maxWidth: 520 }}>
                {item.content}
              </Typography>
            </TableCell>
            <TableCell>{knowledgeBaseNames(bases, item.knowledgeBaseIds)}</TableCell>
            <TableCell>{item.type}</TableCell>
            <TableCell>
              <Chip size="small" label={item.enabled ? '启用' : '停用'} color={item.enabled ? 'success' : 'default'} />
            </TableCell>
            <TableCell>{formatDate(item.updatedAt)}</TableCell>
          </TableRow>
        ))}
      </TableBody>
    </Table>
  );
}

function MediaPlatformTable({ platforms }: { platforms: MediaPlatform[] }) {
  return (
    <Table>
      <TableHead>
        <TableRow>
          <TableCell>平台</TableCell>
          <TableCell>类型</TableCell>
          <TableCell>能力</TableCell>
          <TableCell>状态</TableCell>
          <TableCell>凭证字段</TableCell>
        </TableRow>
      </TableHead>
      <TableBody>
        {platforms.map((platform) => (
          <TableRow key={platform.id} hover>
            <TableCell>{platform.name}</TableCell>
            <TableCell>{platform.type}</TableCell>
            <TableCell>
              <Stack direction="row" spacing={0.5}>
                {platform.supportsArticle && <Chip size="small" label="文章" />}
                {platform.supportsImage && <Chip size="small" label="图片" />}
                {platform.supportsScheduling && <Chip size="small" label="定时" />}
              </Stack>
            </TableCell>
            <TableCell>
              <Chip size="small" label={platform.enabled ? '启用' : '停用'} color={platform.enabled ? 'success' : 'default'} />
            </TableCell>
            <TableCell>{platform.credentialFields.map(credentialFieldLabel).join(', ')}</TableCell>
          </TableRow>
        ))}
      </TableBody>
    </Table>
  );
}

function MediaAccountsTable({
  accounts,
  platforms,
  onLogin,
}: {
  accounts: MediaAccount[];
  platforms: MediaPlatform[];
  onLogin?: (accountId: string) => void;
}) {
  return (
    <Table>
      <TableHead>
        <TableRow>
          <TableCell>账号</TableCell>
          <TableCell>平台</TableCell>
          <TableCell>登录方式</TableCell>
          <TableCell>登录凭证</TableCell>
          <TableCell>外部标识</TableCell>
          <TableCell>状态</TableCell>
          <TableCell>检查时间</TableCell>
          {onLogin && <TableCell align="right">操作</TableCell>}
        </TableRow>
      </TableHead>
      <TableBody>
        {accounts.map((account) => {
          const platform = platforms.find((item) => item.id === account.platformId);
          const canLogin = supportsBrowserLogin(platform?.type) && account.loginMethod === 'qr' && account.status !== 'connected';
          return (
            <TableRow key={account.id} hover>
              <TableCell>{account.name}</TableCell>
              <TableCell>{platform?.name ?? account.platformId}</TableCell>
              <TableCell>{loginMethodLabel(account.loginMethod)}</TableCell>
              <TableCell>{account.loginMethod === 'qr' ? '服务端二维码' : account.credentialMeta?.phoneNumber ?? '-'}</TableCell>
              <TableCell>{account.externalId}</TableCell>
              <TableCell>
                <Chip size="small" label={mediaAccountStatusLabel(account.status)} color={mediaAccountStatusColor(account.status)} />
              </TableCell>
              <TableCell>{formatDate(account.lastCheckedAt)}</TableCell>
              {onLogin && (
                <TableCell align="right">
                  {canLogin ? (
                    <Button size="small" startIcon={<LoginOutlinedIcon />} onClick={() => onLogin(account.id)}>
                      登录绑定
                    </Button>
                  ) : (
                    '-'
                  )}
                </TableCell>
              )}
            </TableRow>
          );
        })}
      </TableBody>
    </Table>
  );
}

function ContentTable({
  contents,
  onPreparePublish,
}: {
  contents: Content[];
  onPreparePublish?: (contentId: string) => void;
}) {
  return (
    <Table>
      <TableHead>
        <TableRow>
          <TableCell>标题</TableCell>
          <TableCell>关键词</TableCell>
          <TableCell>状态</TableCell>
          <TableCell>来源</TableCell>
          <TableCell>更新时间</TableCell>
          {onPreparePublish && <TableCell align="right">操作</TableCell>}
        </TableRow>
      </TableHead>
      <TableBody>
        {contents.map((content) => {
          const status = contentStatusMap[content.status];
          return (
            <TableRow key={content.id} hover>
              <TableCell>
                <Typography fontWeight={700}>{content.title}</Typography>
                <Typography variant="body2" color="text.secondary">
                  {content.summary}
                </Typography>
              </TableCell>
              <TableCell>{content.keywords.join(', ')}</TableCell>
              <TableCell>
                <Chip size="small" label={status.label} color={status.color} />
              </TableCell>
              <TableCell>{content.source}</TableCell>
              <TableCell>{formatDate(content.updatedAt)}</TableCell>
              {onPreparePublish && (
                <TableCell align="right">
                  <Button size="small" startIcon={<PublishOutlinedIcon />} onClick={() => onPreparePublish(content.id)}>
                    小红书发布
                  </Button>
                </TableCell>
              )}
            </TableRow>
          );
        })}
      </TableBody>
    </Table>
  );
}

function SchedulesTable({ data }: { data: WorkspaceData }) {
  return (
    <Table>
      <TableHead>
        <TableRow>
          <TableCell>计划</TableCell>
          <TableCell>内容</TableCell>
          <TableCell>媒体账号</TableCell>
          <TableCell>频率</TableCell>
          <TableCell>下次执行</TableCell>
          <TableCell>状态</TableCell>
        </TableRow>
      </TableHead>
      <TableBody>
        {data.publishSchedules.map((schedule) => (
          <TableRow key={schedule.id} hover>
            <TableCell>{schedule.name}</TableCell>
            <TableCell>{contentName(data.contents, schedule.contentId)}</TableCell>
            <TableCell>{accountName(data.mediaAccounts, schedule.mediaAccountId)}</TableCell>
            <TableCell>{frequencyLabel[schedule.frequency]}</TableCell>
            <TableCell>{formatDate(schedule.nextRunAt)}</TableCell>
            <TableCell>
              <Chip size="small" label={schedule.enabled ? '启用' : '暂停'} color={schedule.enabled ? 'success' : 'default'} />
            </TableCell>
          </TableRow>
        ))}
      </TableBody>
    </Table>
  );
}

function JobsTable({ data, dense = false }: { data: WorkspaceData; dense?: boolean }) {
  return (
    <Table size={dense ? 'small' : 'medium'}>
      <TableHead>
        <TableRow>
          <TableCell>内容</TableCell>
          <TableCell>媒体账号</TableCell>
          <TableCell>状态</TableCell>
          <TableCell>计划时间</TableCell>
          <TableCell>消息</TableCell>
          <TableCell>结果</TableCell>
        </TableRow>
      </TableHead>
      <TableBody>
        {data.publishJobs.map((job) => {
          const status = jobStatusMap[job.status];
          return (
            <TableRow key={job.id} hover>
              <TableCell>{contentName(data.contents, job.contentId)}</TableCell>
              <TableCell>{accountName(data.mediaAccounts, job.mediaAccountId)}</TableCell>
              <TableCell>
                <Chip size="small" label={status.label} color={status.color} />
              </TableCell>
              <TableCell>{formatDate(job.scheduledAt)}</TableCell>
              <TableCell>{job.lastMessage}</TableCell>
              <TableCell>
                {job.externalUrl ? (
                  <Link href={job.externalUrl} target="_blank" rel="noreferrer">
                    查看
                  </Link>
                ) : (
                  '-'
                )}
              </TableCell>
            </TableRow>
          );
        })}
      </TableBody>
    </Table>
  );
}

function InfoRow({ label, value }: { label: string; value: string }) {
  return (
    <Stack direction="row" justifyContent="space-between" spacing={2} sx={{ py: 1.25 }}>
      <Typography color="text.secondary">{label}</Typography>
      <Typography fontWeight={700} sx={{ textAlign: 'right', overflowWrap: 'anywhere' }}>
        {value}
      </Typography>
    </Stack>
  );
}

function formatSubscription(user: User | null) {
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

function formatMoney(cents: number, currency: string) {
  return `${(Number(cents || 0) / 100).toFixed(0)} ${currency || 'USD'}`;
}

function knowledgeBaseName(items: KnowledgeBase[], id: string) {
  return items.find((item) => item.id === id)?.name ?? id;
}

function knowledgeBaseNames(items: KnowledgeBase[], ids: string[]) {
  if (ids.length === 0) {
    return '-';
  }
  return ids.map((id) => knowledgeBaseName(items, id)).join(', ');
}

function uniqueValues(values: string[]) {
  return Array.from(new Set(values.filter(Boolean)));
}

function platformName(items: MediaPlatform[], id: string) {
  return items.find((item) => item.id === id)?.name ?? id;
}

function platformType(items: MediaPlatform[], id: string) {
  return items.find((item) => item.id === id)?.type ?? '';
}

function supportsBrowserLogin(platformTypeValue?: string) {
  return platformTypeValue === 'xiaohongshu';
}

function loginMethodLabel(value?: string) {
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

function mediaAccountStatusLabel(value: string) {
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

function mediaAccountStatusColor(value: string): 'default' | 'success' | 'warning' {
  if (value === 'connected') {
    return 'success';
  }
  if (value === 'pending_login' || value === 'qr_waiting') {
    return 'warning';
  }
  return 'default';
}

function credentialFieldLabel(value: string) {
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

function accountName(items: MediaAccount[], id: string) {
  return items.find((item) => item.id === id)?.name ?? id;
}

function contentName(items: Content[], id: string) {
  return items.find((item) => item.id === id)?.title ?? id;
}

function splitKeywords(value: string) {
  return value
    .split(/[,，;；\n]/)
    .map((item) => item.trim())
    .map((item) => item.replace(/^[-*]\s*/, '').trim())
    .filter(Boolean);
}


function formatDate(value: string) {
  return new Intl.DateTimeFormat('zh-CN', {
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
  }).format(new Date(value));
}

function defaultScheduleInputValue() {
  const nextHour = new Date();
  nextHour.setHours(nextHour.getHours() + 1, 0, 0, 0);
  const timezoneOffset = nextHour.getTimezoneOffset() * 60000;
  return new Date(nextHour.getTime() - timezoneOffset).toISOString().slice(0, 16);
}

export default App;

import { useCallback, useEffect, useMemo, useState } from 'react';
import {
  Alert,
  AppBar,
  Box,
  Button,
  CircularProgress,
  Container,
  FormControl,
  IconButton,
  InputLabel,
  MenuItem,
  Select,
  Stack,
  Toolbar,
  Tooltip,
  Typography,
} from '@mui/material';
import AccountTreeOutlinedIcon from '@mui/icons-material/AccountTreeOutlined';
import ArticleOutlinedIcon from '@mui/icons-material/ArticleOutlined';
import AutorenewIcon from '@mui/icons-material/Autorenew';
import DashboardOutlinedIcon from '@mui/icons-material/DashboardOutlined';
import HelpOutlineIcon from '@mui/icons-material/HelpOutline';
import KeyOutlinedIcon from '@mui/icons-material/KeyOutlined';
import LogoutOutlinedIcon from '@mui/icons-material/LogoutOutlined';
import ManageAccountsOutlinedIcon from '@mui/icons-material/ManageAccountsOutlined';
import CampaignOutlinedIcon from '@mui/icons-material/CampaignOutlined';
import GavelOutlinedIcon from '@mui/icons-material/GavelOutlined';
import HubOutlinedIcon from '@mui/icons-material/HubOutlined';
import LocalMallOutlinedIcon from '@mui/icons-material/LocalMallOutlined';
import GroupsOutlinedIcon from '@mui/icons-material/GroupsOutlined';
import PsychologyAltOutlinedIcon from '@mui/icons-material/PsychologyAltOutlined';
import PublishOutlinedIcon from '@mui/icons-material/PublishOutlined';
import ScheduleOutlinedIcon from '@mui/icons-material/ScheduleOutlined';
import SettingsOutlinedIcon from '@mui/icons-material/SettingsOutlined';
import { ApiRequestError, fetchWorkspace, login } from './api';
import { AdminConsole } from './admin/AdminConsole';
import type { DialogKey, NavItem, ViewKey } from './appTypes';
import { AIThinkingOverlay } from './components/AIThinkingOverlay';
import {
  createFormattingThinkingRunner,
  failThinkingStep,
  generationThinkingSteps,
  initialThinkingState,
} from './components/aiThinkingModel';
import { OnboardingTour } from './components/OnboardingTour';
import type { OnboardingTourStep } from './components/OnboardingTour';
import { WorkspaceDialogs } from './features/workspace/dialogs';
import {
  ActiveView,
  LoginView,
  OnboardingView,
  WorkspaceWorkbenchPanel,
} from './features/workspace/views';
import type { GenerationTrace, User, WorkspaceData } from './types';

const navItems: NavItem[] = [
  { key: 'overview', label: '概览', icon: <DashboardOutlinedIcon /> },
  { key: 'knowledge', label: '知识库', icon: <PsychologyAltOutlinedIcon /> },
  { key: 'accounts', label: '媒体账号', icon: <KeyOutlinedIcon /> },
  { key: 'mediaMatrix', label: '媒体矩阵', icon: <HubOutlinedIcon /> },
  { key: 'campaigns', label: '战役', icon: <CampaignOutlinedIcon /> },
  { key: 'creators', label: '达人合作', icon: <GroupsOutlinedIcon /> },
  { key: 'skillPackages', label: '技能包', icon: <LocalMallOutlinedIcon /> },
  { key: 'brandCompliance', label: '合规报告', icon: <GavelOutlinedIcon /> },
  { key: 'generate', label: 'AI 生成', icon: <ArticleOutlinedIcon /> },
  { key: 'contents', label: '内容', icon: <AccountTreeOutlinedIcon /> },
  { key: 'schedules', label: '计划', icon: <ScheduleOutlinedIcon /> },
  { key: 'jobs', label: '任务', icon: <PublishOutlinedIcon /> },
  { key: 'settings', label: '工作区', icon: <SettingsOutlinedIcon /> },
];

const adminNavItem: NavItem = {
  key: 'admin',
  label: '平台后台',
  icon: <ManageAccountsOutlinedIcon />,
};

const authStorageKey = 'geopress.workspaceAuth';
const workspaceTourStorageKey = 'geopress.workspaceTourCompleted';

const workspaceTourSteps: OnboardingTourStep[] = [
  {
    id: 'workspace',
    title: 'Step 1：选择工作区',
    targetId: 'workspace-select',
    placement: 'bottom',
    content: '先确认当前工作区。知识库、媒体账号、生成内容和发布任务都会按工作区隔离。',
  },
  {
    id: 'knowledge',
    title: 'Step 2：维护知识库',
    targetId: 'nav-knowledge',
    fallbackTargetId: 'mobile-nav-knowledge',
    placement: 'bottom',
    content: '进入知识库，先创建知识库包，再创建品牌、产品、语气、禁忌等引导条目。AI 生成时会从这些条目里检索上下文。',
  },
  {
    id: 'knowledge-base',
    title: 'Step 3：创建知识库包',
    targetId: 'knowledge-create-base',
    placement: 'bottom',
    content: '知识库包相当于一组可复用技能包。后续生成内容时可以选择多个包组合使用。',
  },
  {
    id: 'knowledge-item',
    title: 'Step 4：创建引导条目',
    targetId: 'knowledge-create-item',
    fallbackTargetId: 'overview-create-knowledge-item',
    placement: 'bottom',
    content: '引导条目是最小知识资产，适合记录产品卖点、用户画像、表达风格、素材事实和输出限制。',
  },
  {
    id: 'accounts',
    title: 'Step 5：连接小红书',
    targetId: 'nav-accounts',
    fallbackTargetId: 'mobile-nav-accounts',
    placement: 'bottom',
    content: '进入媒体账号，绑定小红书账号。系统会通过服务端浏览器打开二维码登录页，扫码后保存登录态。',
  },
  {
    id: 'account-bind',
    title: 'Step 6：绑定媒体账号',
    targetId: 'media-bind-account',
    placement: 'bottom',
    content: '点击绑定账号，选择小红书平台。账号创建后，在账号列表里继续扫码完成连接。',
  },
  {
    id: 'generate',
    title: 'Step 7：关键词生成',
    targetId: 'generate-start',
    fallbackTargetId: 'workbench-generate',
    placement: 'bottom',
    content: '进入 AI 生成，输入关键词并选择知识库包。系统会按后台配置的生成链路产出草稿和 Thinking 过程。',
  },
  {
    id: 'publish',
    title: 'Step 8：创建发布任务',
    targetId: 'content-publish-action',
    fallbackTargetId: 'contents-view',
    placement: 'bottom',
    content: '生成草稿后到内容页点击小红书发布，生成发布包，确认标题和正文后执行发布任务。',
  },
  {
    id: 'jobs',
    title: 'Step 9：确认发布结果',
    targetId: 'jobs-list',
    fallbackTargetId: 'mobile-nav-jobs',
    placement: 'bottom',
    content: '最后到任务页查看发布状态。若平台需要人工确认，可以在发布流程里补充外部链接并确认结果。',
  },
];

const workspaceTourStepViews: Partial<Record<string, ViewKey>> = {
  workspace: 'overview',
  knowledge: 'knowledge',
  'knowledge-base': 'knowledge',
  'knowledge-item': 'knowledge',
  accounts: 'accounts',
  'account-bind': 'accounts',
  generate: 'generate',
  publish: 'contents',
  jobs: 'jobs',
};

type StoredAuth = {
  token: string;
  user: User;
  workspaceId: string;
};

function readStoredAuth(): StoredAuth | null {
  try {
    const raw = window.localStorage.getItem(authStorageKey);
    if (!raw) {
      return null;
    }
    const parsed = JSON.parse(raw) as Partial<StoredAuth>;
    if (!parsed.token || !parsed.user || !parsed.workspaceId) {
      return null;
    }
    return {
      token: parsed.token,
      user: parsed.user,
      workspaceId: parsed.workspaceId,
    };
  } catch {
    return null;
  }
}

function writeStoredAuth(auth: StoredAuth) {
  try {
    window.localStorage.setItem(authStorageKey, JSON.stringify(auth));
  } catch {
    // Ignore storage failures so private-mode browsers can still use in-memory auth.
  }
}

function clearStoredAuth() {
  try {
    window.localStorage.removeItem(authStorageKey);
  } catch {
    // Ignore storage failures.
  }
}

function App() {
  const [initialAuth] = useState<StoredAuth | null>(() => readStoredAuth());
  const [token, setToken] = useState(initialAuth?.token ?? '');
  const [user, setUser] = useState<User | null>(initialAuth?.user ?? null);
  const [workspaceId, setWorkspaceId] = useState(initialAuth?.workspaceId ?? '');
  const [activeView, setActiveView] = useState<ViewKey>('overview');
  const [workspace, setWorkspace] = useState<WorkspaceData | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [reloadKey, setReloadKey] = useState(0);
  const [dialog, setDialog] = useState<DialogKey>(null);
  const [selectedContentId, setSelectedContentId] = useState('');
  const [selectedMediaAccountId, setSelectedMediaAccountId] = useState('');
  const [workbenchOpen, setWorkbenchOpen] = useState(true);
  const [thinking, setThinking] = useState(initialThinkingState);
  const [tourOpen, setTourOpen] = useState(false);
  const [tourStepIndex, setTourStepIndex] = useState(0);
  const [tourAutoStarted, setTourAutoStarted] = useState(false);

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
          const nextWorkspaceId = data.workspaces.some((item) => item.id === workspaceId)
            ? workspaceId
            : data.workspaces[0]?.id ?? '';
          setWorkspace(data);
          setUser(data.user);
          if (nextWorkspaceId && nextWorkspaceId !== workspaceId) {
            setWorkspaceId(nextWorkspaceId);
          }
          if (nextWorkspaceId) {
            writeStoredAuth({ token, user: data.user, workspaceId: nextWorkspaceId });
          }
        }
      })
      .catch((err: unknown) => {
        if (mounted) {
          if (err instanceof ApiRequestError && err.status === 401) {
            clearStoredAuth();
            setToken('');
            setUser(null);
            setWorkspaceId('');
            setWorkspace(null);
            setActiveView('overview');
            return;
          }
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
  const runFormattingThinking = useMemo(() => createFormattingThinkingRunner(setThinking), []);
  const setTourStep = useCallback((nextStepIndex: number) => {
    const clampedStepIndex = Math.min(Math.max(nextStepIndex, 0), workspaceTourSteps.length - 1);
    const stepView = workspaceTourStepViews[workspaceTourSteps[clampedStepIndex]?.id];
    if (stepView) {
      setActiveView(stepView);
    }
    setTourStepIndex(clampedStepIndex);
  }, []);
  const startWorkspaceTour = () => {
    setTourAutoStarted(true);
    setTourStep(0);
    setTourOpen(true);
  };
  const rememberWorkspaceTourSeen = () => {
    try {
      window.localStorage.setItem(workspaceTourStorageKey, 'true');
    } catch {
      // Ignore storage failures; the tour can still be started manually.
    }
  };
  const closeWorkspaceTour = () => {
    rememberWorkspaceTourSeen();
    setTourAutoStarted(true);
    setTourOpen(false);
  };
  const completeWorkspaceTour = () => {
    rememberWorkspaceTourSeen();
    setTourAutoStarted(true);
    setTourOpen(false);
  };

  useEffect(() => {
    if (!token || !workspace || activeView === 'admin' || tourAutoStarted) {
      return;
    }
    try {
      if (window.localStorage.getItem(workspaceTourStorageKey) === 'true') {
        setTourAutoStarted(true);
        return;
      }
    } catch {
      // If storage is unavailable, show the tour once for the current page load.
    }
    setTourAutoStarted(true);
    setTourStep(0);
    setTourOpen(true);
  }, [activeView, setTourStep, token, tourAutoStarted, workspace]);

  const handleLogout = () => {
    clearStoredAuth();
    setToken('');
    setUser(null);
    setWorkspaceId('');
    setWorkspace(null);
    setActiveView('overview');
    setDialog(null);
    setThinking(initialThinkingState);
    setTourOpen(false);
    setTourStepIndex(0);
    setTourAutoStarted(false);
  };
  const startGenerationThinking = () => {
    setThinking({
      open: true,
      blocking: true,
      title: 'AI 生文思考过程',
      subtitle: '正在按订阅链路生成发布草稿',
      steps: generationThinkingSteps(),
      trace: null,
    });
  };
  const failThinking = (message: string) => failThinkingStep(setThinking, message);
  const showGenerationTrace = (trace: GenerationTrace) => {
    setThinking({
      open: true,
      blocking: false,
      title: 'AI 生文思考过程',
      subtitle: `${trace.subscriptionTier || 'free'} 链路 / ${trace.steps.length} 个阶段`,
      steps: [],
      trace,
    });
  };
  const closeThinking = () => {
    setThinking((current) => ({ ...current, open: false, blocking: false }));
  };

  if (!token) {
    return (
      <LoginView
        onLogin={(result: Awaited<ReturnType<typeof login>>) => {
          const nextWorkspaceId = result.workspaces[0]?.id ?? '';
          setToken(result.token);
          setUser(result.user);
          setWorkspaceId(nextWorkspaceId);
          setWorkspace(null);
          setActiveView('overview');
          if (nextWorkspaceId) {
            writeStoredAuth({ token: result.token, user: result.user, workspaceId: nextWorkspaceId });
          }
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
          writeStoredAuth({ token, user: result.user, workspaceId });
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
                data-tour-id={`nav-${item.key}`}
              >
                {item.label}
              </Button>
            ))}
          </Stack>

          <FormControl size="small" sx={{ minWidth: { xs: 160, sm: 240 } }} data-tour-id="workspace-select">
            <InputLabel id="workspace-select-label">工作区</InputLabel>
            <Select
              labelId="workspace-select-label"
              label="工作区"
              value={workspaceId}
              onChange={(event) => {
                const nextWorkspaceId = String(event.target.value);
                setWorkspaceId(nextWorkspaceId);
                if (user) {
                  writeStoredAuth({ token, user, workspaceId: nextWorkspaceId });
                }
              }}
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
          <Tooltip title="教学引导">
            <IconButton onClick={startWorkspaceTour} data-tour-id="tour-start">
              <HelpOutlineIcon />
            </IconButton>
          </Tooltip>
          <Tooltip title="退出登录">
            <IconButton onClick={handleLogout}>
              <LogoutOutlinedIcon />
            </IconButton>
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
                    data-tour-id={`mobile-nav-${item.key}`}
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
          onStartGenerationThinking={startGenerationThinking}
          onGeneratedTrace={showGenerationTrace}
          runFormattingThinking={runFormattingThinking}
          onThinkingFailed={failThinking}
        />
      )}
      <AIThinkingOverlay state={thinking} onClose={closeThinking} />
      <OnboardingTour
        open={tourOpen && activeView !== 'admin'}
        steps={workspaceTourSteps}
        stepIndex={tourStepIndex}
        onStepChange={setTourStep}
        onClose={closeWorkspaceTour}
        onFinish={completeWorkspaceTour}
      />
    </Box>
  );
}

export default App;

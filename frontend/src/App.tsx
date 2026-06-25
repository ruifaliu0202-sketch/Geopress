import { useCallback, useEffect, useMemo, useState } from 'react';
import {
  Alert,
  Box,
  Button,
  CircularProgress,
  Dialog,
  DialogActions,
  DialogContent,
  DialogTitle,
  FormControl,
  IconButton,
  InputLabel,
  MenuItem,
  Radio,
  Select,
  Stack,
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
import DeleteOutlineIcon from '@mui/icons-material/DeleteOutline';
import RestoreFromTrashOutlinedIcon from '@mui/icons-material/RestoreFromTrashOutlined';
import TuneOutlinedIcon from '@mui/icons-material/TuneOutlined';
import {
  ApiRequestError,
  deleteKnowledgeAssetForever,
  deleteKnowledgeBaseForever,
  fetchKnowledgeTrash,
  fetchWorkspace,
  login,
  purgeExpiredKnowledgeTrash,
  restoreKnowledgeAsset,
  restoreKnowledgeBase,
} from './api';
import { AdminConsole } from './admin/AdminConsole';
import type { DialogKey, NavItem, ViewKey } from './appTypes';
import { AIThinkingOverlay } from './components/AIThinkingOverlay';
import {
  failThinkingStep,
  generationThinkingSteps,
  initialThinkingState,
} from './components/aiThinkingModel';
import { FloatingWorkspaceAssistant } from './components/assistant';
import { InfoRow, ProductSurface } from './components/common';
import { WorkspaceShell } from './components/layout/WorkspaceShell';
import { OnboardingTour } from './components/OnboardingTour';
import type { OnboardingTourStep } from './components/OnboardingTour';
import { WorkspaceDialogs } from './features/workspace/dialogs';
import {
  ActiveView,
  LoginView,
  OnboardingView,
} from './features/workspace/views';
import type { GenerationTrace, User, WorkspaceData } from './types';
import {
  themePresets,
  type ThemePreference,
} from './theme';

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
    fallbackTargetId: 'mobile-nav-menu',
    placement: 'bottom',
    content: '进入知识库，先创建知识库包，再上传文件或创建文本资产。AI 生成时会从知识资产片段里检索上下文。',
  },
  {
    id: 'knowledge-base',
    title: 'Step 3：创建知识库包',
    targetId: 'knowledge-create-base',
    placement: 'bottom',
    content: '知识库包相当于一组可复用技能包。后续生成内容时可以选择多个包组合使用。',
  },
  {
    id: 'knowledge-asset',
    title: 'Step 4：创建知识资产',
    targetId: 'knowledge-create-asset',
    fallbackTargetId: 'overview-create-knowledge-asset',
    placement: 'bottom',
    content: '知识资产可以是上传文件或文本资产，适合记录产品卖点、用户画像、表达风格、素材事实和输出限制。',
  },
  {
    id: 'accounts',
    title: 'Step 5：连接小红书',
    targetId: 'nav-accounts',
    fallbackTargetId: 'mobile-nav-menu',
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
    fallbackTargetId: 'assistant-generate',
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
    fallbackTargetId: 'mobile-nav-menu',
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

function PersonalSettingsDialog({
  open,
  user,
  themePreference,
  onThemePreferenceChange,
  onClose,
}: {
  open: boolean;
  user: User | null;
  themePreference: ThemePreference;
  onThemePreferenceChange: (value: ThemePreference) => void;
  onClose: () => void;
}) {
  return (
    <Dialog open={open} onClose={onClose} fullWidth maxWidth="sm">
      <DialogTitle>个人设置</DialogTitle>
      <DialogContent>
        <Stack spacing={2.5} sx={{ pt: 1 }}>
          <Stack spacing={0.5}>
            <Typography fontWeight={700}>{user?.name ?? '用户'}</Typography>
            <Typography color="text.secondary">{user?.email ?? ''}</Typography>
          </Stack>
          <Stack spacing={1}>
            <Typography fontWeight={700}>颜色主题</Typography>
            <Stack spacing={1}>
              {(Object.keys(themePresets) as ThemePreference[]).map((key) => {
                const preset = themePresets[key];
                const selected = themePreference === key;
                return (
                  <Box
                    key={key}
                    role="button"
                    tabIndex={0}
                    onClick={() => onThemePreferenceChange(key)}
                    onKeyDown={(event) => {
                      if (event.key === 'Enter' || event.key === ' ') {
                        event.preventDefault();
                        onThemePreferenceChange(key);
                      }
                    }}
                    sx={{
                      display: 'grid',
                      gridTemplateColumns: 'auto 1fr auto',
                      alignItems: 'center',
                      gap: 1.25,
                      p: 1.25,
                      border: '1px solid',
                      borderColor: selected ? 'primary.main' : 'divider',
                      borderRadius: 1,
                      cursor: 'pointer',
                      bgcolor: selected ? 'action.selected' : 'background.paper',
                    }}
                  >
                    <Radio checked={selected} size="small" />
                    <Stack spacing={0.25} sx={{ minWidth: 0 }}>
                      <Typography fontWeight={700}>{preset.label}</Typography>
                      <Typography variant="body2" color="text.secondary">
                        偏好会保存在本机浏览器。
                      </Typography>
                    </Stack>
                    <Stack direction="row" spacing={0.5}>
                      {[preset.primary, preset.secondary, preset.background].map((color) => (
                        <Box
                          key={color}
                          sx={{
                            width: 22,
                            height: 22,
                            borderRadius: 0.75,
                            bgcolor: color,
                            border: '1px solid',
                            borderColor: 'divider',
                          }}
                        />
                      ))}
                    </Stack>
                  </Box>
                );
              })}
            </Stack>
          </Stack>
        </Stack>
      </DialogContent>
      <DialogActions>
        <Button onClick={onClose}>关闭</Button>
      </DialogActions>
    </Dialog>
  );
}

function TrashDialog({
  open,
  token,
  workspaceId,
  onClose,
  onChanged,
}: {
  open: boolean;
  token: string;
  workspaceId: string;
  onClose: () => void;
  onChanged: () => void;
}) {
  const [trash, setTrash] = useState<{ knowledgeBases: WorkspaceData['knowledgeBases']; knowledgeAssets: WorkspaceData['knowledgeAssets'] }>({
    knowledgeBases: [],
    knowledgeAssets: [],
  });
  const [loading, setLoading] = useState(false);
  const [busyId, setBusyId] = useState('');
  const [error, setError] = useState<string | null>(null);
  const [confirmDelete, setConfirmDelete] = useState<{ type: 'base' | 'asset'; id: string; title: string } | null>(null);

  const loadTrash = useCallback(async () => {
    if (!open || !token || !workspaceId) {
      return;
    }
    setLoading(true);
    setError(null);
    try {
      const result = await fetchKnowledgeTrash(token, workspaceId);
      setTrash(result);
    } catch (err) {
      setError(err instanceof Error ? err.message : '垃圾箱加载失败');
    } finally {
      setLoading(false);
    }
  }, [open, token, workspaceId]);

  useEffect(() => {
    void loadTrash();
  }, [loadTrash]);

  const runAction = async (id: string, action: () => Promise<unknown>) => {
    setBusyId(id);
    setError(null);
    try {
      await action();
      await loadTrash();
      onChanged();
    } catch (err) {
      setError(err instanceof Error ? err.message : '操作失败');
    } finally {
      setBusyId('');
    }
  };

  const empty = trash.knowledgeBases.length === 0 && trash.knowledgeAssets.length === 0;

  return (
    <>
      <Dialog open={open} onClose={loading || busyId ? undefined : onClose} fullWidth maxWidth="md">
        <DialogTitle>垃圾箱</DialogTitle>
        <DialogContent>
          <Stack spacing={2} sx={{ pt: 1 }}>
            <Alert severity="info">移入垃圾箱的知识库包和知识资产会保留 30 天，期间可以恢复；到期后会自动清理。</Alert>
            {error && <Alert severity="error">{error}</Alert>}
            {loading ? (
              <Stack direction="row" spacing={1.5} alignItems="center">
                <CircularProgress size={20} />
                <Typography color="text.secondary">正在读取垃圾箱</Typography>
              </Stack>
            ) : empty ? (
              <Typography color="text.secondary">垃圾箱为空。</Typography>
            ) : (
              <Stack spacing={2}>
                <TrashSection
                  title="知识库包"
                  items={trash.knowledgeBases.map((item) => ({
                    id: item.id,
                    title: item.name,
                    subtitle: `${item.itemCount} 个资产 / ${item.deleteExpiresAt ? `到期 ${new Date(item.deleteExpiresAt).toLocaleDateString()}` : '30 天后清理'}`,
                  }))}
                  busyId={busyId}
                  onRestore={(id) => runAction(id, () => restoreKnowledgeBase(token, workspaceId, id))}
                  onDelete={(item) => setConfirmDelete({ type: 'base', id: item.id, title: item.title })}
                />
                <TrashSection
                  title="知识资产"
                  items={trash.knowledgeAssets.map((item) => ({
                    id: item.id,
                    title: item.title,
                    subtitle: `${item.assetType || item.mimeType || 'asset'} / ${item.deleteExpiresAt ? `到期 ${new Date(item.deleteExpiresAt).toLocaleDateString()}` : '30 天后清理'}`,
                  }))}
                  busyId={busyId}
                  onRestore={(id) => runAction(id, () => restoreKnowledgeAsset(token, workspaceId, id))}
                  onDelete={(item) => setConfirmDelete({ type: 'asset', id: item.id, title: item.title })}
                />
              </Stack>
            )}
          </Stack>
        </DialogContent>
        <DialogActions>
          <Button
            startIcon={<RestoreFromTrashOutlinedIcon />}
            onClick={() => runAction('purge-expired', () => purgeExpiredKnowledgeTrash(token, workspaceId))}
            disabled={loading || Boolean(busyId)}
          >
            清理过期
          </Button>
          <Button onClick={onClose} disabled={loading || Boolean(busyId)}>
            关闭
          </Button>
        </DialogActions>
      </Dialog>

      <Dialog open={Boolean(confirmDelete)} onClose={() => setConfirmDelete(null)} maxWidth="xs" fullWidth>
        <DialogTitle>彻底删除</DialogTitle>
        <DialogContent>
          <Typography>
            确认彻底删除「{confirmDelete?.title}」？删除后不能从垃圾箱恢复。
          </Typography>
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setConfirmDelete(null)}>取消</Button>
          <Button
            color="error"
            variant="contained"
            onClick={() => {
              const target = confirmDelete;
              if (!target) {
                return;
              }
              setConfirmDelete(null);
              void runAction(
                target.id,
                target.type === 'base'
                  ? () => deleteKnowledgeBaseForever(token, workspaceId, target.id)
                  : () => deleteKnowledgeAssetForever(token, workspaceId, target.id),
              );
            }}
          >
            彻底删除
          </Button>
        </DialogActions>
      </Dialog>
    </>
  );
}

function TrashSection({
  title,
  items,
  busyId,
  onRestore,
  onDelete,
}: {
  title: string;
  items: { id: string; title: string; subtitle: string }[];
  busyId: string;
  onRestore: (id: string) => void;
  onDelete: (item: { id: string; title: string; subtitle: string }) => void;
}) {
  return (
    <Stack spacing={1}>
      <Typography fontWeight={700}>{title}</Typography>
      {items.length === 0 ? (
        <Typography color="text.secondary">暂无{title}</Typography>
      ) : (
        <Stack spacing={1}>
          {items.map((item) => (
            <Box
              key={item.id}
              sx={{
                display: 'grid',
                gridTemplateColumns: { xs: '1fr', sm: '1fr auto' },
                gap: 1,
                alignItems: 'center',
                p: 1.25,
                border: '1px solid',
                borderColor: 'divider',
                borderRadius: 1,
                bgcolor: 'background.paper',
              }}
            >
              <Stack spacing={0.25} sx={{ minWidth: 0 }}>
                <Typography fontWeight={700} sx={{ overflowWrap: 'anywhere' }}>
                  {item.title}
                </Typography>
                <Typography variant="body2" color="text.secondary" sx={{ overflowWrap: 'anywhere' }}>
                  {item.subtitle}
                </Typography>
              </Stack>
              <Stack direction="row" spacing={1} justifyContent="flex-end">
                <Button size="small" onClick={() => onRestore(item.id)} disabled={Boolean(busyId)}>
                  恢复
                </Button>
                <Button size="small" color="error" onClick={() => onDelete(item)} disabled={Boolean(busyId)}>
                  彻底删除
                </Button>
              </Stack>
            </Box>
          ))}
        </Stack>
      )}
    </Stack>
  );
}

function App({
  themePreference,
  onThemePreferenceChange,
}: {
  themePreference: ThemePreference;
  onThemePreferenceChange: (value: ThemePreference) => void;
}) {
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
  const [thinking, setThinking] = useState(initialThinkingState);
  const [tourOpen, setTourOpen] = useState(false);
  const [tourStepIndex, setTourStepIndex] = useState(0);
  const [tourAutoStarted, setTourAutoStarted] = useState(false);
  const [trashOpen, setTrashOpen] = useState(false);
  const [settingsOpen, setSettingsOpen] = useState(false);

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
    <WorkspaceShell
      activeView={activeView}
      navItems={visibleNavItems}
      onNavigate={setActiveView}
      topShortcuts={
        <>
          <FormControl
            size="small"
            sx={{ width: { xs: 156, sm: 240 }, flex: '0 1 auto' }}
            data-tour-id="workspace-select"
          >
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
              <IconButton disabled={loading} onClick={refresh} aria-label="刷新数据">
                <AutorenewIcon />
              </IconButton>
            </span>
          </Tooltip>
          <Tooltip title="教学引导">
            <IconButton onClick={startWorkspaceTour} aria-label="教学引导" data-tour-id="tour-start">
              <HelpOutlineIcon />
            </IconButton>
          </Tooltip>
          <Tooltip title="垃圾箱">
            <IconButton onClick={() => setTrashOpen(true)} aria-label="垃圾箱">
              <DeleteOutlineIcon />
            </IconButton>
          </Tooltip>
          <Tooltip title="个人设置">
            <IconButton onClick={() => setSettingsOpen(true)} aria-label="个人设置">
              <TuneOutlinedIcon />
            </IconButton>
          </Tooltip>
          {user?.isPlatformAdmin && (
            <Tooltip title="平台后台">
              <IconButton onClick={() => setActiveView('admin')} aria-label="平台后台">
                <ManageAccountsOutlinedIcon />
              </IconButton>
            </Tooltip>
          )}
          <Stack sx={{ display: { xs: 'none', md: 'flex' }, minWidth: 0, maxWidth: 220 }}>
            <Typography variant="body2" fontWeight={700} noWrap>
              {user?.name ?? '用户'}
            </Typography>
            <Typography variant="caption" color="text.secondary" noWrap>
              {user?.email ?? ''}
            </Typography>
          </Stack>
          <Tooltip title="退出登录">
            <IconButton onClick={handleLogout} aria-label="退出登录">
              <LogoutOutlinedIcon />
            </IconButton>
          </Tooltip>
        </>
      }
      rightContext={
        <Stack spacing={2} sx={{ position: { lg: 'sticky' }, top: { lg: 88 } }}>
          <ProductSurface tone="cream" padded>
            <Stack spacing={1}>
              <Typography variant="h3">工作区上下文</Typography>
              <InfoRow label="名称" value={currentWorkspace?.name ?? '-'} />
              <InfoRow label="行业" value={currentWorkspace?.industry ?? '-'} />
              <InfoRow label="语气" value={currentWorkspace?.tone ?? '-'} />
              <InfoRow label="方案" value={currentWorkspace?.plan ?? '-'} />
            </Stack>
          </ProductSurface>
          <ProductSurface tone="sage" padded>
            <Stack spacing={0.75}>
              <Typography variant="h3">AI 工作台</Typography>
              <Typography color="text.secondary">
                右下角 Corgi 助手已接入生成、知识库、账号绑定、发布计划和引导动作。
              </Typography>
            </Stack>
          </ProductSurface>
        </Stack>
      }
    >
      <Stack spacing={3}>
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
          onThinkingFailed={failThinking}
        />
      )}
      <TrashDialog
        open={trashOpen}
        token={token}
        workspaceId={workspaceId}
        onClose={() => setTrashOpen(false)}
        onChanged={refresh}
      />
      <PersonalSettingsDialog
        open={settingsOpen}
        user={user}
        themePreference={themePreference}
        onThemePreferenceChange={onThemePreferenceChange}
        onClose={() => setSettingsOpen(false)}
      />
      <AIThinkingOverlay state={thinking} onClose={closeThinking} />
      {workspace && (
        <FloatingWorkspaceAssistant
          workspace={currentWorkspace}
          user={user}
          state={{ loading, error, online: !error }}
          actionCallbacks={{
            generateContent: () => setDialog('generate'),
            createKnowledgeBase: () => setDialog('knowledgeBase'),
            createKnowledgeAsset: () => setDialog('knowledgeAsset'),
            bindMediaAccount: () => setDialog('mediaAccount'),
            createSchedule: () => setDialog('schedule'),
            openOnboardingGuide: () => startWorkspaceTour(),
            refreshWorkspace: () => refresh(),
          }}
        />
      )}
      <OnboardingTour
        open={tourOpen && activeView !== 'admin'}
        steps={workspaceTourSteps}
        stepIndex={tourStepIndex}
        onStepChange={setTourStep}
        onClose={closeWorkspaceTour}
        onFinish={completeWorkspaceTour}
      />
    </WorkspaceShell>
  );
}

export default App;

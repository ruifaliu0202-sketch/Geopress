import { useEffect, useMemo, useState, type ReactNode } from 'react';
import {
  Alert,
  AppBar,
  Box,
  Button,
  Card,
  CardContent,
  Chip,
  CircularProgress,
  Container,
  Dialog,
  DialogActions,
  DialogContent,
  DialogTitle,
  Divider,
  FormControl,
  Grid,
  IconButton,
  InputLabel,
  MenuItem,
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
import DashboardOutlinedIcon from '@mui/icons-material/DashboardOutlined';
import KeyOutlinedIcon from '@mui/icons-material/KeyOutlined';
import LoginOutlinedIcon from '@mui/icons-material/LoginOutlined';
import ManageAccountsOutlinedIcon from '@mui/icons-material/ManageAccountsOutlined';
import PsychologyAltOutlinedIcon from '@mui/icons-material/PsychologyAltOutlined';
import PublishOutlinedIcon from '@mui/icons-material/PublishOutlined';
import ScheduleOutlinedIcon from '@mui/icons-material/ScheduleOutlined';
import SettingsOutlinedIcon from '@mui/icons-material/SettingsOutlined';
import {
  createContent,
  createKnowledgeBase,
  createKnowledgeItem,
  createMediaAccount,
  createPublishSchedule,
  fetchWorkspace,
  generateContent,
  login,
} from './api';
import { AdminConsole } from './admin/AdminConsole';
import type {
  Content,
  ContentStatus,
  KnowledgeBase,
  KnowledgeItem,
  MediaAccount,
  MediaPlatform,
  PublishJobStatus,
  PublishScheduleFrequency,
  User,
  Workspace,
  WorkspaceData,
} from './types';

type ViewKey = 'overview' | 'knowledge' | 'accounts' | 'generate' | 'contents' | 'schedules' | 'jobs' | 'settings' | 'admin';
type DialogKey =
  | 'knowledgeBase'
  | 'knowledgeItem'
  | 'mediaAccount'
  | 'content'
  | 'generate'
  | 'schedule'
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
        <Stack spacing={3}>
          <Paper
            elevation={0}
            sx={{ p: { xs: 2, md: 3 }, border: '1px solid', borderColor: 'divider', borderRadius: 2 }}
          >
            <Stack
              direction={{ xs: 'column', md: 'row' }}
              spacing={2}
              alignItems={{ xs: 'flex-start', md: 'center' }}
              justifyContent="space-between"
            >
              <Box>
                <Typography variant="h1">工作区工作台</Typography>
                <Typography color="text.secondary" sx={{ mt: 0.75 }}>
                  {currentWorkspace
                    ? `${currentWorkspace.name} / ${currentWorkspace.industry} / ${currentWorkspace.tone}`
                    : '正在读取工作区信息'}
                </Typography>
              </Box>
              <Stack direction="row" spacing={1} flexWrap="wrap" useFlexGap>
                <Button startIcon={<PsychologyAltOutlinedIcon />} variant="outlined" onClick={() => setDialog('generate')}>
                  关键词生成
                </Button>
                <Button startIcon={<ScheduleOutlinedIcon />} variant="outlined" onClick={() => setDialog('schedule')}>
                  新建计划
                </Button>
                <Button startIcon={<AddIcon />} variant="contained" onClick={() => setDialog('content')}>
                  新建内容
                </Button>
              </Stack>
            </Stack>
          </Paper>

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
              workspace={workspace}
              currentWorkspace={currentWorkspace}
              openDialog={setDialog}
            />
          )}
        </Stack>
      </Container>

      {workspace && (
        <WorkspaceDialogs
          dialog={dialog}
          token={token}
          workspaceId={workspaceId}
          data={workspace}
          onClose={() => setDialog(null)}
          onCreated={(nextView) => {
            setDialog(null);
            if (nextView) {
              setActiveView(nextView);
            }
            refresh();
          }}
        />
      )}
    </Box>
  );
}

function LoginView({ onLogin }: { onLogin: (result: Awaited<ReturnType<typeof login>>) => void }) {
  const [email, setEmail] = useState('demo@geopress.local');
  const [password, setPassword] = useState('demo');
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const handleLogin = async () => {
    setSubmitting(true);
    setError(null);
    try {
      const result = await login(email, password);
      onLogin(result);
    } catch (err) {
      setError(err instanceof Error ? err.message : '登录失败');
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
              登录后进入个人或公司工作区。
            </Typography>
          </Box>
          {error && <Alert severity="error">{error}</Alert>}
          <TextField label="邮箱" value={email} onChange={(event) => setEmail(event.target.value)} fullWidth />
          <TextField
            label="密码"
            type="password"
            value={password}
            onChange={(event) => setPassword(event.target.value)}
            fullWidth
          />
          <Button startIcon={<LoginOutlinedIcon />} variant="contained" onClick={handleLogin} disabled={submitting}>
            登录
          </Button>
          <Typography variant="body2" color="text.secondary">
            Demo 账号：demo@geopress.local 或 growth@geopress.local，任意密码。
          </Typography>
        </Stack>
      </Paper>
    </Box>
  );
}

function ActiveView({
  view,
  workspace,
  currentWorkspace,
  openDialog,
}: {
  view: ViewKey;
  workspace: WorkspaceData;
  currentWorkspace: Workspace;
  openDialog: (dialog: DialogKey) => void;
}) {
  if (view === 'knowledge') {
    return <KnowledgeView data={workspace} openDialog={openDialog} />;
  }
  if (view === 'accounts') {
    return <AccountsView data={workspace} openDialog={openDialog} />;
  }
  if (view === 'generate') {
    return <GenerateView data={workspace} openDialog={openDialog} />;
  }
  if (view === 'contents') {
    return <ContentsView contents={workspace.contents} openDialog={openDialog} />;
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

function KnowledgeView({ data, openDialog }: { data: WorkspaceData; openDialog: (dialog: DialogKey) => void }) {
  return (
    <Stack spacing={2}>
      <Grid container spacing={2}>
        {data.knowledgeBases.map((base) => (
          <Grid key={base.id} size={{ xs: 12, md: 6, lg: 4 }}>
            <Card>
              <CardContent>
                <Stack spacing={1.25}>
                  <Stack direction="row" justifyContent="space-between" spacing={2}>
                    <Typography variant="h3">{base.name}</Typography>
                    <Chip label={`${base.itemCount} 条`} size="small" color="info" />
                  </Stack>
                  <Typography color="text.secondary">{base.description}</Typography>
                  <Typography variant="body2" color="text.secondary">
                    更新于 {formatDate(base.updatedAt)}
                  </Typography>
                </Stack>
              </CardContent>
            </Card>
          </Grid>
        ))}
      </Grid>
      <Section
        title="知识条目"
        action={
          <Stack direction="row" spacing={1}>
            <Button startIcon={<AddIcon />} onClick={() => openDialog('knowledgeBase')}>
              新建知识库
            </Button>
            <Button startIcon={<AddIcon />} variant="contained" onClick={() => openDialog('knowledgeItem')}>
              新增条目
            </Button>
          </Stack>
        }
      >
        <KnowledgeItemsTable items={data.knowledgeItems} bases={data.knowledgeBases} />
      </Section>
    </Stack>
  );
}

function AccountsView({ data, openDialog }: { data: WorkspaceData; openDialog: (dialog: DialogKey) => void }) {
  return (
    <Stack spacing={2}>
      <Section
        title="媒体平台能力"
        action={<Button startIcon={<AddIcon />} variant="contained" onClick={() => openDialog('mediaAccount')}>绑定账号</Button>}
      >
        <MediaPlatformTable platforms={data.mediaPlatforms} />
      </Section>
      <Section title="已绑定媒体账号">
        <MediaAccountsTable accounts={data.mediaAccounts} platforms={data.mediaPlatforms} />
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
          现阶段使用 mock AI Provider。生成时会读取当前工作区知识库条目，先打通上下文选择、草稿生成和后续排程链路。
        </Typography>
        <ContentTable contents={data.contents.filter((item) => item.source !== 'manual')} />
      </Stack>
    </Section>
  );
}

function ContentsView({ contents, openDialog }: { contents: Content[]; openDialog: (dialog: DialogKey) => void }) {
  return (
    <Section
      title="内容管理"
      action={<Button startIcon={<AddIcon />} variant="contained" onClick={() => openDialog('content')}>新建内容</Button>}
    >
      <ContentTable contents={contents} />
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
          <InfoRow label="套餐" value={workspace.plan} />
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
  onClose,
  onCreated,
}: {
  dialog: DialogKey;
  token: string;
  workspaceId: string;
  data: WorkspaceData;
  onClose: () => void;
  onCreated: (nextView?: ViewKey) => void;
}) {
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
        onClose={onClose}
        onCreated={() => onCreated('contents')}
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
  ...props
}: DialogBaseProps & {
  bases: KnowledgeBase[];
}) {
  const [knowledgeBaseId, setKnowledgeBaseId] = useState('');
  const [type, setType] = useState('brand');
  const [title, setTitle] = useState('');
  const [content, setContent] = useState('');
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    if (props.open) {
      setKnowledgeBaseId(bases[0]?.id ?? '');
      setError(null);
    }
  }, [bases, props.open]);

  const submit = async () => {
    if (!knowledgeBaseId || !title.trim() || !content.trim()) {
      setError('请选择知识库并填写标题和内容');
      return;
    }
    setSubmitting(true);
    setError(null);
    try {
      await createKnowledgeItem(props.token, props.workspaceId, {
        knowledgeBaseId,
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

  return (
    <FormDialog title="新增知识条目" open={props.open} error={error} submitting={submitting} onClose={props.onClose} onSubmit={submit}>
      <SelectField label="知识库" value={knowledgeBaseId} onChange={setKnowledgeBaseId} items={bases.map((base) => ({ value: base.id, label: base.name }))} />
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
  const enabledPlatforms = platforms.filter((platform) => platform.enabled);
  const [platformId, setPlatformId] = useState('');
  const [name, setName] = useState('');
  const [externalId, setExternalId] = useState('');
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    if (props.open) {
      setPlatformId(enabledPlatforms[0]?.id ?? '');
      setError(null);
    }
  }, [enabledPlatforms, props.open]);

  const submit = async () => {
    if (!platformId || !name.trim()) {
      setError('请选择平台并填写账号名称');
      return;
    }
    setSubmitting(true);
    setError(null);
    try {
      await createMediaAccount(props.token, props.workspaceId, {
        platformId,
        name: name.trim(),
        externalId: externalId.trim(),
      });
      setName('');
      setExternalId('');
      props.onCreated();
    } catch (err) {
      setError(err instanceof Error ? err.message : '创建失败');
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <FormDialog title="绑定媒体账号" open={props.open} error={error} submitting={submitting} onClose={props.onClose} onSubmit={submit}>
      <SelectField label="媒体平台" value={platformId} onChange={setPlatformId} items={enabledPlatforms.map((platform) => ({ value: platform.id, label: platform.name }))} />
      <TextField label="账号名称" value={name} onChange={(event) => setName(event.target.value)} fullWidth required />
      <TextField label="外部账号标识" value={externalId} onChange={(event) => setExternalId(event.target.value)} fullWidth />
    </FormDialog>
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
  ...props
}: DialogBaseProps & {
  bases: KnowledgeBase[];
}) {
  const [keywords, setKeywords] = useState('内容营销, 增长');
  const [contentType, setContentType] = useState('article');
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
    const values = splitKeywords(keywords);
    if (values.length === 0) {
      setError('请至少填写一个关键词');
      return;
    }
    setSubmitting(true);
    setError(null);
    try {
      await generateContent(props.token, props.workspaceId, { keywords: values, contentType, knowledgeBaseId });
      props.onCreated();
    } catch (err) {
      setError(err instanceof Error ? err.message : '生成失败');
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <FormDialog title="关键词生成文章" open={props.open} error={error} submitting={submitting} onClose={props.onClose} onSubmit={submit}>
      <SelectField label="知识库上下文" value={knowledgeBaseId} onChange={setKnowledgeBaseId} items={bases.map((base) => ({ value: base.id, label: base.name }))} />
      <SelectField
        label="内容类型"
        value={contentType}
        onChange={setContentType}
        items={[
          { value: 'article', label: '长文章' },
          { value: 'brief', label: '短文' },
          { value: 'case_study', label: '案例稿' },
          { value: 'product_intro', label: '产品介绍' },
        ]}
      />
      <TextField label="关键词" value={keywords} onChange={(event) => setKeywords(event.target.value)} fullWidth required />
    </FormDialog>
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

function KnowledgeItemsTable({ items, bases }: { items: KnowledgeItem[]; bases: KnowledgeBase[] }) {
  return (
    <Table>
      <TableHead>
        <TableRow>
          <TableCell>标题</TableCell>
          <TableCell>知识库</TableCell>
          <TableCell>类型</TableCell>
          <TableCell>状态</TableCell>
          <TableCell>更新时间</TableCell>
        </TableRow>
      </TableHead>
      <TableBody>
        {items.map((item) => (
          <TableRow key={item.id} hover>
            <TableCell>
              <Typography fontWeight={700}>{item.title}</Typography>
              <Typography variant="body2" color="text.secondary" sx={{ maxWidth: 520 }}>
                {item.content}
              </Typography>
            </TableCell>
            <TableCell>{knowledgeBaseName(bases, item.knowledgeBaseId)}</TableCell>
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
            <TableCell>{platform.credentialFields.join(', ')}</TableCell>
          </TableRow>
        ))}
      </TableBody>
    </Table>
  );
}

function MediaAccountsTable({ accounts, platforms }: { accounts: MediaAccount[]; platforms: MediaPlatform[] }) {
  return (
    <Table>
      <TableHead>
        <TableRow>
          <TableCell>账号</TableCell>
          <TableCell>平台</TableCell>
          <TableCell>外部标识</TableCell>
          <TableCell>状态</TableCell>
          <TableCell>检查时间</TableCell>
        </TableRow>
      </TableHead>
      <TableBody>
        {accounts.map((account) => (
          <TableRow key={account.id} hover>
            <TableCell>{account.name}</TableCell>
            <TableCell>{platformName(platforms, account.platformId)}</TableCell>
            <TableCell>{account.externalId}</TableCell>
            <TableCell>
              <Chip size="small" label={account.status === 'connected' ? '已连接' : '需处理'} color={account.status === 'connected' ? 'success' : 'warning'} />
            </TableCell>
            <TableCell>{formatDate(account.lastCheckedAt)}</TableCell>
          </TableRow>
        ))}
      </TableBody>
    </Table>
  );
}

function ContentTable({ contents }: { contents: Content[] }) {
  return (
    <Table>
      <TableHead>
        <TableRow>
          <TableCell>标题</TableCell>
          <TableCell>关键词</TableCell>
          <TableCell>状态</TableCell>
          <TableCell>来源</TableCell>
          <TableCell>更新时间</TableCell>
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

function knowledgeBaseName(items: KnowledgeBase[], id: string) {
  return items.find((item) => item.id === id)?.name ?? id;
}

function platformName(items: MediaPlatform[], id: string) {
  return items.find((item) => item.id === id)?.name ?? id;
}

function accountName(items: MediaAccount[], id: string) {
  return items.find((item) => item.id === id)?.name ?? id;
}

function contentName(items: Content[], id: string) {
  return items.find((item) => item.id === id)?.title ?? id;
}

function splitKeywords(value: string) {
  return value
    .split(/[,，\n]/)
    .map((item) => item.trim())
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

import { useEffect, useMemo, useState } from 'react';
import {
  Alert,
  Box,
  Button,
  Checkbox,
  Dialog,
  DialogActions,
  DialogContent,
  DialogTitle,
  Divider,
  FormControl,
  Grid,
  IconButton,
  InputLabel,
  Link,
  List,
  ListItem,
  ListItemText,
  MenuItem,
  Paper,
  Select,
  Stack,
  TextField,
  Tooltip,
  Typography,
} from '@mui/material';
import ContentCopyIcon from '@mui/icons-material/ContentCopy';
import type { DialogKey, ViewKey } from '../../appTypes';
import {
  completeMediaAccountBrowserLogin,
  createContent,
  createKnowledgeBase,
  createKnowledgeItem,
  createMediaAccount,
  createPublishSchedule,
  fetchInstalledSkillPackages,
  formatKnowledgeContent,
  generateContent,
  preparePublish,
  runPublishJob,
  startMediaAccountBrowserLogin,
} from '../../api';
import type { RunFormattingThinking } from '../../components/aiThinkingModel';
import { DialogBaseProps, FormDialog, InfoRow, SelectField } from '../../components/common';
import type {
  Content,
  GenerationTrace,
  InstalledSkillPackage,
  KnowledgeBase,
  MediaAccount,
  MediaPlatform,
  PreparedPost,
  PreparePublishResponse,
  PublishScheduleFrequency,
  User,
  WorkspaceData,
} from '../../types';
import {
  defaultScheduleInputValue,
  isMarkdownPrompt,
  knowledgeBaseNames,
  platformName,
  platformType,
  mediaAccountStatusLabel,
  splitGenerationKeywords,
  splitKeywords,
  supportsBrowserLogin,
} from '../../utils/formatters';
import vipGoldIconUrl from '../../assets/vip-gold.png';

const contentTypeOptions = [
  { value: 'xiaohongshu_long_article', label: '小红书长文' },
  { value: 'article', label: '通用长文章' },
  { value: 'brief', label: '短文' },
  { value: 'case_study', label: '案例稿' },
  { value: 'product_intro', label: '产品介绍' },
];

export function WorkspaceDialogs({
  dialog,
  token,
  workspaceId,
  data,
  selectedContentId,
  selectedMediaAccountId,
  onClose,
  onCreated,
  onStartGenerationThinking,
  onGeneratedTrace,
  runFormattingThinking,
  onThinkingFailed,
}: {
  dialog: DialogKey;
  token: string;
  workspaceId: string;
  data: WorkspaceData;
  selectedContentId: string;
  selectedMediaAccountId: string;
  onClose: () => void;
  onCreated: (nextView?: ViewKey) => void;
  onStartGenerationThinking: () => void;
  onGeneratedTrace: (trace: GenerationTrace) => void;
  runFormattingThinking: RunFormattingThinking;
  onThinkingFailed: (message: string) => void;
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
        runFormattingThinking={runFormattingThinking}
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
        onStartGenerationThinking={onStartGenerationThinking}
        onGeneratedTrace={onGeneratedTrace}
        runFormattingThinking={runFormattingThinking}
        onThinkingFailed={onThinkingFailed}
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
  runFormattingThinking,
  ...props
}: DialogBaseProps & {
  bases: KnowledgeBase[];
  user: User;
  runFormattingThinking: RunFormattingThinking;
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
      await runFormattingThinking({
        subtitle: '正在把知识条目整理成可检索 Markdown',
        request: () =>
          formatKnowledgeContent(props.token, props.workspaceId, {
            type,
            title: title.trim(),
            content: content.trim(),
          }),
        onSuccess: (response) => {
          setContent(response.content);
          if (response.fallback) {
            setError(`AI 格式化不可用，已使用 mock 降级：${response.fallbackError || 'provider failed'}`);
          }
        },
        onFailure: setError,
      });
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
  onStartGenerationThinking,
  onGeneratedTrace,
  runFormattingThinking,
  onThinkingFailed,
  ...props
}: DialogBaseProps & {
  bases: KnowledgeBase[];
  user: User;
  onStartGenerationThinking: () => void;
  onGeneratedTrace: (trace: GenerationTrace) => void;
  runFormattingThinking: RunFormattingThinking;
  onThinkingFailed: (message: string) => void;
}) {
  const [keywords, setKeywords] = useState('内容营销, 增长');
  const [contentType, setContentType] = useState('xiaohongshu_long_article');
  const [knowledgeBaseIds, setKnowledgeBaseIds] = useState<string[]>([]);
  const [installedSkillPackages, setInstalledSkillPackages] = useState<InstalledSkillPackage[]>([]);
  const [skillPackageVersionId, setSkillPackageVersionId] = useState('');
  const [loadingSkillPackages, setLoadingSkillPackages] = useState(false);
  const [submitting, setSubmitting] = useState(false);
  const [formatting, setFormatting] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const keywordItems = useMemo(() => splitGenerationKeywords(keywords), [keywords]);
  const isFormatAvailable = user.subscriptionTier === 'vip' && user.subscriptionStatus === 'active';

  useEffect(() => {
    if (props.open) {
      setKnowledgeBaseIds(bases.map((base) => base.id));
      setError(null);
    }
  }, [bases, props.open]);

  useEffect(() => {
    if (!props.open) {
      return;
    }
    let mounted = true;
    setLoadingSkillPackages(true);
    fetchInstalledSkillPackages(props.token, props.workspaceId)
      .then((items) => {
        if (!mounted) {
          return;
        }
        setInstalledSkillPackages(items);
        setSkillPackageVersionId((current) =>
          current && items.some((item) => item.entitlement.versionId === current) ? current : '',
        );
      })
      .catch(() => {
        if (!mounted) {
          return;
        }
        setInstalledSkillPackages([]);
        setSkillPackageVersionId('');
      })
      .finally(() => {
        if (mounted) {
          setLoadingSkillPackages(false);
        }
      });
    return () => {
      mounted = false;
    };
  }, [props.open, props.token, props.workspaceId]);

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
      const sourceKeywords = isMarkdownPrompt(keywords) ? splitGenerationKeywords(keywords).join('\n') : keywords.trim();
      await runFormattingThinking({
        subtitle: '正在把关键词整理成生成提示词',
        request: () =>
          formatKnowledgeContent(props.token, props.workspaceId, {
            type: 'generation_keywords',
            title: '发布内容关键词',
            content: sourceKeywords,
          }),
        onSuccess: (response) => {
          setKeywords(response.content);
          if (response.fallback) {
            setError(`AI 格式化不可用，已使用 mock 降级：${response.fallbackError || 'provider failed'}`);
          }
        },
        onFailure: setError,
      });
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
    onStartGenerationThinking();
    try {
      const keywordPrompt = isMarkdownPrompt(keywords) ? keywords.trim() : undefined;
      const response = await generateContent(props.token, props.workspaceId, {
        keywords: values,
        keywordPrompt,
        contentType,
        knowledgeBaseIds,
        publishFormatId: contentType,
        skillPackageVersionId: skillPackageVersionId || undefined,
      });
      onGeneratedTrace(response.trace);
      props.onCreated();
    } catch (err) {
      const message = err instanceof Error ? err.message : '生成失败';
      setError(message);
      onThinkingFailed(message);
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
          <SelectField
            label="创作技能包"
            value={skillPackageVersionId}
            onChange={setSkillPackageVersionId}
            items={[
              { value: '', label: loadingSkillPackages ? '正在加载技能包' : '不使用技能包' },
              ...installedSkillPackages.map((item) => ({
                value: item.entitlement.versionId,
                label: `${item.package?.name ?? item.entitlement.packageId}${item.version?.version ? ` / ${item.version.version}` : ''}`,
              })),
            ]}
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
          />
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

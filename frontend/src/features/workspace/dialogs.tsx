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
import UploadFileOutlinedIcon from '@mui/icons-material/UploadFileOutlined';
import type { DialogKey, ViewKey } from '../../appTypes';
import {
  completeMediaAccountBrowserLogin,
  confirmPublishJob,
  createContent,
  createKnowledgeAssetFromText,
  createKnowledgeBase,
  createMediaAccount,
  createPublishSchedule,
  fetchInstalledSkillPackages,
  fetchMediaAccountAuthStatus,
  generateContent,
  preparePublish,
  runPublishJob,
  sendMediaAccountAuthAction,
  startMediaAccountBrowserLogin,
  startMediaAccountAuth,
  uploadKnowledgeAsset,
} from '../../api';
import { DialogBaseProps, FormDialog, InfoRow, SelectField, VIPFeatureButton } from '../../components/common';
import { WorkflowDrawer } from '../../components/WorkflowDrawer';
import type { WorkflowState, WorkflowStep } from '../../components/workflowModel';
import type {
  Content,
  GenerationTrace,
  InstalledSkillPackage,
  KnowledgeBase,
  MediaAccount,
  MediaAccountAuthState,
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
  mediaAccountStatusLabel,
  splitGenerationKeywords,
  splitKeywords,
  supportsBrowserLogin,
  supportsInteractiveLogin,
} from '../../utils/formatters';

const contentTypeOptions = [
  { value: 'xiaohongshu_long_article', label: '小红书长文' },
  { value: 'article', label: '通用长文章' },
  { value: 'brief', label: '短文' },
  { value: 'case_study', label: '案例稿' },
  { value: 'product_intro', label: '产品介绍' },
];

function knowledgeAssetFilename(title: string) {
  const normalized = title.trim().replace(/[\\/:*?"<>|]/g, '-');
  return `${normalized || 'knowledge-asset'}.md`;
}

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
      <KnowledgeAssetDialog
        open={dialog === 'knowledgeAsset'}
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
        onClose={onClose}
        onCreated={() => onCreated('contents')}
        onStartGenerationThinking={onStartGenerationThinking}
        onGeneratedTrace={onGeneratedTrace}
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

function KnowledgeAssetDialog({
  bases,
  user,
  ...props
}: DialogBaseProps & {
  bases: KnowledgeBase[];
  user: User;
}) {
  const [mode, setMode] = useState<'upload' | 'text'>('upload');
  const [knowledgeBaseIds, setKnowledgeBaseIds] = useState<string[]>([]);
  const [title, setTitle] = useState('');
  const [text, setText] = useState('');
  const [file, setFile] = useState<File | null>(null);
  const [aiEnhancementEnabled, setAIEnhancementEnabled] = useState(false);
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const isVIP = user.subscriptionTier === 'vip' && user.subscriptionStatus === 'active';
  const selectedFileIsOCRTarget = mode === 'upload' && file ? isOCRTargetFile(file) : false;

  useEffect(() => {
    if (props.open) {
      setMode('upload');
      setKnowledgeBaseIds([]);
      setTitle('');
      setText('');
      setFile(null);
      setAIEnhancementEnabled(false);
      setError(null);
    }
  }, [bases, props.open]);

  const submit = async () => {
    if (mode === 'upload' && !file) {
      setError('请选择要上传的文件');
      return;
    }
    if (mode === 'text' && (!title.trim() || !text.trim())) {
      setError('请填写标题和文本内容');
      return;
    }
    setSubmitting(true);
    setError(null);
    try {
      if (mode === 'upload') {
        await uploadKnowledgeAsset(props.token, props.workspaceId, {
          file: file as File,
          title: title.trim() || undefined,
          knowledgeBaseIds,
          aiEnhancementEnabled: isVIP && aiEnhancementEnabled,
          mimeType: file?.type,
          assetType: 'upload',
        });
      } else {
        await createKnowledgeAssetFromText(props.token, props.workspaceId, {
          title: title.trim(),
          text: text.trim(),
          knowledgeBaseIds,
          aiEnhancementEnabled: isVIP && aiEnhancementEnabled,
          mimeType: 'text/markdown',
          originalFilename: knowledgeAssetFilename(title),
          assetType: 'text',
        });
      }
      props.onCreated();
    } catch (err) {
      setError(err instanceof Error ? err.message : '知识资产创建失败');
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <FormDialog title="新增知识资产" open={props.open} error={error} submitting={submitting} onClose={props.onClose} onSubmit={submit}>
      <Alert severity="info">
        支持上传 Word、文本、PDF、图片，也可以直接粘贴文本创建资产。图片和 PDF 会使用 AI 视觉 OCR 解析。
      </Alert>
      <Stack direction={{ xs: 'column', sm: 'row' }} spacing={1}>
        <Button variant={mode === 'upload' ? 'contained' : 'outlined'} onClick={() => setMode('upload')} disabled={submitting}>
          文件上传
        </Button>
        <Button variant={mode === 'text' ? 'contained' : 'outlined'} onClick={() => setMode('text')} disabled={submitting}>
          文本创建
        </Button>
      </Stack>
      <FormControl fullWidth disabled={bases.length === 0 || submitting}>
        <InputLabel shrink>知识库包（可选）</InputLabel>
        <Select
          multiple
          displayEmpty
          label="知识库包（可选）"
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
      <TextField
        label={mode === 'upload' ? '标题（可选）' : '标题'}
        value={title}
        onChange={(event) => setTitle(event.target.value)}
        fullWidth
        required={mode === 'text'}
      />
      {mode === 'upload' ? (
        <Stack spacing={1}>
          <Button component="label" variant="outlined" startIcon={<UploadFileOutlinedIcon />} disabled={submitting}>
            选择文件
            <input
              hidden
              type="file"
              accept=".doc,.docx,.txt,.md,.markdown,.pdf,image/*,text/plain,text/markdown,application/pdf,application/msword,application/vnd.openxmlformats-officedocument.wordprocessingml.document"
              onChange={(event) => {
                setFile(event.target.files?.[0] ?? null);
              }}
            />
          </Button>
          <Typography variant="body2" color="text.secondary" sx={{ overflowWrap: 'anywhere' }}>
            {file ? `${file.name} / ${Math.ceil(file.size / 1024)} KB` : '未选择文件'}
          </Typography>
          {selectedFileIsOCRTarget && (
            <Alert severity={isVIP ? 'info' : 'warning'}>
              图片和 PDF 解析需要付费订阅；当前账号{isVIP ? '可使用 AI 视觉 OCR。' : '不是付费订阅，上传后知识资产解析会失败。'}
            </Alert>
          )}
        </Stack>
      ) : (
        <TextField
          label="文本内容"
          value={text}
          onChange={(event) => setText(event.target.value)}
          fullWidth
          multiline
          minRows={6}
          required
        />
      )}
      <Stack direction={{ xs: 'column', sm: 'row' }} spacing={1.25} alignItems={{ xs: 'stretch', sm: 'center' }}>
        <Tooltip title={isVIP ? '使用 AI 生成标题、摘要、标签和结构化 Markdown，不替代原始资产。' : 'AI 增强仅 VIP 订阅可用。'}>
          <span>
            <VIPFeatureButton
              type="button"
              selected={aiEnhancementEnabled && isVIP}
              disabled={!isVIP || submitting}
              onClick={() => setAIEnhancementEnabled((value) => !value)}
            >
              AI增强
            </VIPFeatureButton>
          </span>
        </Tooltip>
      </Stack>
      {!isVIP && (
        <Alert severity="warning">
          当前订阅不可使用 AI 增强，请升级 VIP 后启用。
        </Alert>
      )}
    </FormDialog>
  );
}

function isOCRTargetFile(file: File) {
  const name = file.name.toLowerCase();
  const mimeType = file.type.toLowerCase();
  return mimeType.startsWith('image/') || mimeType === 'application/pdf' || /\.(png|jpe?g|webp|gif|bmp|tiff?|heic|pdf)$/.test(name);
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
  const selectedPlatformName = selectedPlatform?.name ?? '媒体平台';

  useEffect(() => {
    if (props.open) {
      const defaultPlatform = enabledPlatforms.find((platform) => platform.credentialFields.includes('qrLogin')) ?? enabledPlatforms.find((platform) => platform.credentialFields.includes('phoneNumber')) ?? enabledPlatforms[0];
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
        <Alert severity="info">{selectedPlatformName}绑定将由服务端浏览器打开二维码登录页，扫码确认后保存服务端浏览器会话。</Alert>
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
  const [authState, setAuthState] = useState<MediaAccountAuthState | null>(null);
  const [phoneNumber, setPhoneNumber] = useState('');
  const [captchaCode, setCaptchaCode] = useState('');
  const [smsCode, setSMSCode] = useState('');
  const [sessionStarted, setSessionStarted] = useState(false);
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    if (props.open) {
      setSessionId('');
      setQRScreenshotData('');
      setQRLoginUrl('');
      setStateFile('');
      setAuthState(null);
      setPhoneNumber(account?.credentialMeta?.phoneNumber ?? '');
      setCaptchaCode('');
      setSMSCode('');
      setSessionStarted(false);
      setError(null);
    }
  }, [account, props.open]);

  const platformNameValue = platform?.name ?? '媒体平台';

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

  const startInteractiveLogin = async () => {
    if (!account) {
      setError('请选择媒体账号');
      return;
    }
    setSubmitting(true);
    setError(null);
    try {
      const response = await startMediaAccountAuth(props.token, props.workspaceId, account.id);
      setSessionStarted(true);
      setSessionId(response.sessionId);
      setStateFile(response.stateFile);
      setAuthState(response.state);
    } catch (err) {
      setError(err instanceof Error ? err.message : '启动手机号登录失败');
    } finally {
      setSubmitting(false);
    }
  };

  const refreshInteractiveLogin = async () => {
    if (!account || !sessionId) {
      return;
    }
    try {
      const response = await fetchMediaAccountAuthStatus(props.token, props.workspaceId, account.id);
      setAuthState(response.state);
      if (response.account.status === 'connected') {
        props.onCreated();
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : '读取登录状态失败');
    }
  };

  const sendInteractiveAction = async (action: string) => {
    if (!account || !sessionId) {
      setError('请先启动登录会话');
      return;
    }
    setSubmitting(true);
    setError(null);
    try {
      const response = await sendMediaAccountAuthAction(props.token, props.workspaceId, account.id, {
        sessionId,
        action,
        phoneNumber: phoneNumber.trim(),
        captchaCode: captchaCode.trim(),
        smsCode: smsCode.trim(),
      });
      setAuthState(response.state);
      if (response.account.status === 'connected') {
        props.onCreated();
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : '提交登录信息失败');
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

  const canQRLogin = Boolean(account && supportsBrowserLogin(platform?.type) && account.loginMethod === 'qr');
  const canInteractiveLogin = Boolean(account && supportsInteractiveLogin(platform) && account.loginMethod === 'phone');
  const canLogin = canQRLogin || canInteractiveLogin;
  const qrWorkflow = mediaAccountQRWorkflow(platformNameValue, qrScreenshotData, sessionStarted, submitting);
  const qrWorkflowOpen = canQRLogin && (submitting || sessionStarted || Boolean(qrScreenshotData));
  const closeQRWorkflow = () => {
    setQRScreenshotData('');
    setQRLoginUrl('');
    setStateFile('');
    setSessionId('');
    setSessionStarted(false);
  };
  const interactiveWorkflow = mediaAccountAuthWorkflow(platformNameValue, authState, sessionStarted, submitting);
  const closeInteractiveWorkflow = () => {
    setAuthState(null);
    setSessionStarted(false);
    setSessionId('');
    setCaptchaCode('');
    setSMSCode('');
  };
  const interactiveWorkflowOpen = canInteractiveLogin && Boolean(authState || sessionStarted);

  return (
    <>
      <Dialog open={props.open} onClose={submitting ? undefined : props.onClose} fullWidth maxWidth="sm">
        <DialogTitle>{platformNameValue}登录绑定</DialogTitle>
        <DialogContent>
          <Stack spacing={2} sx={{ pt: 1 }}>
            {error && <Alert severity="error">{error}</Alert>}
            {!canLogin && <Alert severity="warning">当前账号不支持服务端浏览器登录绑定。</Alert>}
            {canLogin && (
              <Alert severity="info">
                {canInteractiveLogin
                  ? `点击开始登录后，右侧工作流会接管${platformNameValue}手机号登录。验证码由你输入，系统不会保存验证码。`
                  : `点击生成二维码后，请使用${platformNameValue}支持的 App 扫码并确认登录。确认后返回这里完成绑定。`}
              </Alert>
            )}
            {account && (
              <Paper variant="outlined" sx={{ p: 1.5, borderRadius: 1 }}>
                <InfoRow label="媒体账号" value={account.name} />
                <InfoRow label="平台" value={platform?.name ?? account.platformId} />
                <InfoRow label="状态" value={mediaAccountStatusLabel(account.status)} />
              </Paper>
            )}
            {canQRLogin && (
              <Stack direction={{ xs: 'column', sm: 'row' }} spacing={1}>
                <Button variant="outlined" onClick={startLogin} disabled={submitting} sx={{ minWidth: 160 }}>
                  生成二维码
                </Button>
                <Button variant="contained" onClick={completeLogin} disabled={submitting || !sessionStarted}>
                  我已扫码确认
                </Button>
              </Stack>
            )}
            {canInteractiveLogin && (
              <Stack direction={{ xs: 'column', sm: 'row' }} spacing={1}>
                <Button variant="outlined" onClick={startInteractiveLogin} disabled={submitting} sx={{ minWidth: 160 }}>
                  开始手机号登录
                </Button>
                <Button variant="text" onClick={refreshInteractiveLogin} disabled={submitting || !sessionStarted}>
                  打开/刷新工作流
                </Button>
              </Stack>
            )}
          </Stack>
        </DialogContent>
        <DialogActions>
          <Button onClick={props.onClose} disabled={submitting}>
            取消
          </Button>
          {canQRLogin && (
            <Button onClick={completeLogin} disabled={submitting || !sessionStarted} variant="contained">
              完成绑定
            </Button>
          )}
          {canInteractiveLogin && (
            <Button onClick={startInteractiveLogin} disabled={submitting} variant="contained">
              {sessionStarted ? '刷新工作流' : '开始登录'}
            </Button>
          )}
        </DialogActions>
      </Dialog>
      <WorkflowDrawer
        state={{
          ...qrWorkflow,
          open: qrWorkflowOpen,
        }}
        onClose={closeQRWorkflow}
        emptyText="二维码登录会显示服务端浏览器、二维码截图和扫码确认步骤。"
      >
        <QRLoginWorkflowControls
          platformName={platformNameValue}
          qrScreenshotData={qrScreenshotData}
          qrLoginUrl={qrLoginUrl}
          sessionId={sessionId}
          stateFile={stateFile}
          submitting={submitting}
          onGenerate={startLogin}
          onComplete={completeLogin}
        />
      </WorkflowDrawer>
      <WorkflowDrawer
        state={{
          ...interactiveWorkflow,
          open: interactiveWorkflowOpen,
        }}
        onClose={closeInteractiveWorkflow}
        emptyText="手机号登录会显示服务端浏览器、验证码和登录态检测步骤。"
      >
        <SohuPhoneLoginWorkflowControls
          authState={authState}
          platformName={platformNameValue}
          phoneNumber={phoneNumber}
          captchaCode={captchaCode}
          smsCode={smsCode}
          submitting={submitting}
          onPhoneNumberChange={setPhoneNumber}
          onCaptchaCodeChange={setCaptchaCode}
          onSMSCodeChange={setSMSCode}
          onSubmitPhone={() => sendInteractiveAction('submit_phone')}
          onRefreshCaptcha={() => sendInteractiveAction('refresh_captcha')}
          onSubmitCaptcha={() => sendInteractiveAction('submit_captcha')}
          onSubmitSMSCode={() => sendInteractiveAction('submit_sms_code')}
          onRefreshStatus={refreshInteractiveLogin}
        />
      </WorkflowDrawer>
    </>
  );
}

function interactiveLoginStatusLabel(value: string) {
  const labels: Record<string, string> = {
    starting: '启动浏览器',
    phone_required: '等待手机号',
    captcha_required: '等待图形验证码',
    sms_code_required: '等待短信验证码',
    manual_intervention_required: '需要人工验证',
    profile_in_use: '浏览器会话占用',
    action_failed: '操作失败',
    debug_screenshot_ready: '调试截图已生成',
    connected: '已连接',
    expired: '已超时',
  };
  return labels[value] ?? value;
}

function mediaAccountQRWorkflow(platformName: string, qrScreenshotData: string, sessionStarted: boolean, submitting: boolean): WorkflowState {
  return {
    open: submitting || sessionStarted || Boolean(qrScreenshotData),
    blocking: submitting,
    title: `${platformName}二维码登录工作流`,
    subtitle: qrScreenshotData ? '二维码已生成，等待扫码确认' : '服务端浏览器正在生成登录二维码',
    steps: mediaAccountQRWorkflowSteps(qrScreenshotData, submitting),
    warnings: [],
  };
}

function mediaAccountQRWorkflowSteps(qrScreenshotData: string, submitting: boolean): WorkflowStep[] {
  const hasQR = Boolean(qrScreenshotData);
  const items: Array<{ id: string; label: string; summary: string }> = [
    { id: 'start_browser', label: '启动服务端浏览器', summary: '为当前媒体账号打开持久化浏览器 profile。' },
    { id: 'open_login', label: '打开登录页', summary: '访问平台登录页并等待页面稳定。' },
    { id: 'capture_qr', label: '截取登录二维码', summary: '从页面中定位二维码区域并生成可展示截图。' },
    { id: 'confirm_login', label: '等待扫码确认', summary: '用户扫码确认后，服务端检测平台登录态。' },
  ];
  const activeIndex = hasQR ? 3 : submitting ? 2 : 0;
  return items.map((item, index) => ({
    ...item,
    status: hasQR && index < 3 ? 'succeeded' : index === activeIndex ? 'running' : index < activeIndex ? 'succeeded' : 'pending',
    details: [],
    warnings: [],
  }));
}

function QRLoginWorkflowControls({
  platformName,
  qrScreenshotData,
  qrLoginUrl,
  sessionId,
  stateFile,
  submitting,
  onGenerate,
  onComplete,
}: {
  platformName: string;
  qrScreenshotData: string;
  qrLoginUrl: string;
  sessionId: string;
  stateFile: string;
  submitting: boolean;
  onGenerate: () => void;
  onComplete: () => void;
}) {
  return (
    <Paper variant="outlined" sx={{ p: 1.5, borderRadius: 1 }}>
      <Stack spacing={1.4} alignItems="stretch">
        {qrScreenshotData ? (
          <Box component="img" src={qrScreenshotData} alt={`${platformName}登录二维码`} sx={{ width: 240, height: 240, alignSelf: 'center' }} />
        ) : (
          <Alert severity="info">正在等待服务端浏览器返回二维码截图。</Alert>
        )}
        <Stack direction={{ xs: 'column', sm: 'row' }} spacing={1}>
          <Button variant="outlined" onClick={onGenerate} disabled={submitting}>
            重新生成
          </Button>
          <Button variant="contained" onClick={onComplete} disabled={submitting || !sessionId}>
            我已扫码确认
          </Button>
        </Stack>
        {sessionId && (
          <Typography variant="caption" color="text.secondary" sx={{ overflowWrap: 'anywhere' }}>
            {sessionId}
          </Typography>
        )}
        {stateFile && (
          <Typography variant="caption" color="text.secondary" sx={{ overflowWrap: 'anywhere' }}>
            {stateFile}
          </Typography>
        )}
        {qrLoginUrl && (
          <Link href={qrLoginUrl} target="_blank" rel="noreferrer" variant="body2">
            打开登录链接
          </Link>
        )}
      </Stack>
    </Paper>
  );
}

function mediaAccountAuthWorkflow(platformName: string, authState: MediaAccountAuthState | null, sessionStarted: boolean, submitting: boolean): WorkflowState {
  const steps = mediaAccountAuthWorkflowSteps(authState, submitting);
  return {
    open: sessionStarted || Boolean(authState),
    blocking: submitting,
    title: `${platformName}登录工作流`,
    subtitle: authState?.message || (sessionStarted ? '服务端浏览器正在执行登录步骤' : '等待启动登录会话'),
    steps,
    warnings: authState?.warnings ?? [],
  };
}

function mediaAccountAuthWorkflowSteps(authState: MediaAccountAuthState | null, submitting: boolean): WorkflowStep[] {
  const status = authState?.status ?? 'starting';
  const base: Array<{ id: string; label: string; summary: string }> = [
    { id: 'start_browser', label: '启动服务端浏览器', summary: '打开平台登录页并保持浏览器 profile。' },
    { id: 'submit_phone', label: '提交手机号', summary: '等待用户输入手机号后写入页面。' },
    { id: 'captcha', label: '读取图形验证码', summary: '从平台页面截取图形验证码并等待用户输入。' },
    { id: 'sms', label: '输入短信验证码', summary: '提交图形验证码并等待用户输入手机收到的验证码。' },
    { id: 'confirm_login', label: '确认登录态', summary: '提交短信验证码并检测平台登录状态。' },
  ];
  const activeIndex = mediaAccountAuthActiveStepIndex(status);
  return base.map((step, index) => {
    const failed = status === 'action_failed' || status === 'expired';
    const intervention = status === 'manual_intervention_required' || status === 'profile_in_use';
    return {
      ...step,
      status:
        status === 'connected'
          ? 'succeeded'
          : failed && index === activeIndex
            ? 'failed'
            : intervention && index === activeIndex
              ? 'running'
              : index < activeIndex
                ? 'succeeded'
                : index === activeIndex
                  ? submitting ? 'running' : 'running'
                  : 'pending',
      details: index === activeIndex && authState?.message ? [authState.message] : [],
      warnings: index === activeIndex && failed && authState?.message ? [authState.message] : [],
    };
  });
}

function mediaAccountAuthActiveStepIndex(status: string) {
  if (status === 'starting' || status === 'profile_in_use') {
    return 0;
  }
  if (status === 'phone_required') {
    return 1;
  }
  if (status === 'captcha_required') {
    return 2;
  }
  if (status === 'sms_code_required') {
    return 3;
  }
  if (status === 'manual_intervention_required' || status === 'action_failed' || status === 'expired') {
    return 4;
  }
  if (status === 'connected') {
    return 4;
  }
  return 0;
}

function SohuPhoneLoginWorkflowControls({
  authState,
  platformName,
  phoneNumber,
  captchaCode,
  smsCode,
  submitting,
  onPhoneNumberChange,
  onCaptchaCodeChange,
  onSMSCodeChange,
  onSubmitPhone,
  onRefreshCaptcha,
  onSubmitCaptcha,
  onSubmitSMSCode,
  onRefreshStatus,
}: {
  authState: MediaAccountAuthState | null;
  platformName: string;
  phoneNumber: string;
  captchaCode: string;
  smsCode: string;
  submitting: boolean;
  onPhoneNumberChange: (value: string) => void;
  onCaptchaCodeChange: (value: string) => void;
  onSMSCodeChange: (value: string) => void;
  onSubmitPhone: () => void;
  onRefreshCaptcha: () => void;
  onSubmitCaptcha: () => void;
  onSubmitSMSCode: () => void;
  onRefreshStatus: () => void;
}) {
  if (!authState) {
    return <Alert severity="info">点击开始登录后，服务端浏览器会进入手机号登录流程。</Alert>;
  }

  return (
    <Paper variant="outlined" sx={{ p: 1.5, borderRadius: 1 }}>
      <Stack spacing={1.4}>
        <InfoRow label="当前步骤" value={interactiveLoginStatusLabel(authState.status)} />
        <Typography variant="body2" color="text.secondary">
          {authState.message}
        </Typography>
        {authState.status === 'phone_required' && (
          <Stack spacing={1}>
            <TextField label="手机号" value={phoneNumber} onChange={(event) => onPhoneNumberChange(event.target.value)} fullWidth />
            <Button variant="contained" onClick={onSubmitPhone} disabled={submitting || !phoneNumber.trim()}>
              下一步
            </Button>
          </Stack>
        )}
        {authState.status === 'captcha_required' && (
          <Stack spacing={1}>
            {authState.captchaScreenshotData ? (
              <Box
                component="img"
                src={authState.captchaScreenshotData}
                alt={`${platformName}图形验证码`}
                sx={{ width: 176, height: 70, objectFit: 'contain', border: '1px solid', borderColor: 'divider', borderRadius: 1 }}
              />
            ) : (
              <Alert severity="warning">暂未读取到图形验证码，请刷新验证码或继续检测。</Alert>
            )}
            <TextField label="图形验证码" value={captchaCode} onChange={(event) => onCaptchaCodeChange(event.target.value)} fullWidth />
            <Stack direction={{ xs: 'column', sm: 'row' }} spacing={1}>
              <Button variant="outlined" onClick={onRefreshCaptcha} disabled={submitting}>
                刷新
              </Button>
              <Button variant="contained" onClick={onSubmitCaptcha} disabled={submitting || !captchaCode.trim()}>
                发送短信
              </Button>
            </Stack>
          </Stack>
        )}
        {authState.status === 'sms_code_required' && (
          <Stack spacing={1}>
            <TextField label="短信验证码" value={smsCode} onChange={(event) => onSMSCodeChange(event.target.value)} fullWidth />
            <Button variant="contained" onClick={onSubmitSMSCode} disabled={submitting || !smsCode.trim()}>
              完成登录
            </Button>
          </Stack>
        )}
        {authState.status === 'manual_intervention_required' && (
          <Alert severity="warning">检测到滑块、风控或登录态未确认。请在服务端浏览器窗口完成验证，然后继续检测。</Alert>
        )}
        {authState.status === 'profile_in_use' && (
          <Alert severity="warning">该账号已有浏览器会话正在运行。请回到已有登录窗口继续操作，或关闭占用后重新开始。</Alert>
        )}
        <Button variant="text" onClick={onRefreshStatus} disabled={submitting}>
          继续检测
        </Button>
        <Typography variant="caption" color="text.secondary" sx={{ overflowWrap: 'anywhere' }}>
          {authState.stateFile}
        </Typography>
      </Stack>
    </Paper>
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
  onStartGenerationThinking,
  onGeneratedTrace,
  onThinkingFailed,
  ...props
}: DialogBaseProps & {
  bases: KnowledgeBase[];
  onStartGenerationThinking: () => void;
  onGeneratedTrace: (trace: GenerationTrace) => void;
  onThinkingFailed: (message: string) => void;
}) {
  const [keywords, setKeywords] = useState('内容营销, 增长');
  const [contentType, setContentType] = useState('xiaohongshu_long_article');
  const [knowledgeBaseIds, setKnowledgeBaseIds] = useState<string[]>([]);
  const [installedSkillPackages, setInstalledSkillPackages] = useState<InstalledSkillPackage[]>([]);
  const [skillPackageVersionId, setSkillPackageVersionId] = useState('');
  const [loadingSkillPackages, setLoadingSkillPackages] = useState(false);
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const keywordItems = useMemo(() => splitGenerationKeywords(keywords), [keywords]);

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
    <Dialog open={props.open} onClose={submitting ? undefined : props.onClose} fullWidth maxWidth="md">
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
          <Typography fontWeight={700}>关键词与素材</Typography>
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
        <Button onClick={props.onClose} disabled={submitting}>
          取消
        </Button>
        <Button onClick={submit} disabled={submitting} variant="contained">
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
  const connectedAccounts = useMemo(
    () => data.mediaAccounts.filter((account) => account.status === 'connected'),
    [data.mediaAccounts],
  );
  const [contentId, setContentId] = useState('');
  const [mediaAccountId, setMediaAccountId] = useState('');
  const [prepared, setPrepared] = useState<PreparePublishResponse | null>(null);
  const [publishResult, setPublishResult] = useState<PreparePublishResponse['publishResult']>(undefined);
  const [publishTitle, setPublishTitle] = useState('');
  const [publishBody, setPublishBody] = useState('');
  const [externalUrl, setExternalUrl] = useState('');
  const [confirmationMessage, setConfirmationMessage] = useState('');
  const [copiedLabel, setCopiedLabel] = useState('');
  const [submitting, setSubmitting] = useState(false);
  const [running, setRunning] = useState(false);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    if (props.open) {
      const requestedContent = data.contents.find((content) => content.id === selectedContentId);
      setContentId(requestedContent?.id ?? data.contents[0]?.id ?? '');
      setMediaAccountId(connectedAccounts[0]?.id ?? '');
      setPrepared(null);
      setPublishResult(undefined);
      setPublishTitle('');
      setPublishBody('');
      setExternalUrl('');
      setConfirmationMessage('');
      setCopiedLabel('');
      setError(null);
    }
  }, [connectedAccounts, data.contents, props.open, selectedContentId]);

  const selectedContent = data.contents.find((content) => content.id === contentId);
  const selectedAccount = connectedAccounts.find((account) => account.id === mediaAccountId);
  const selectedPlatform = selectedAccount ? data.mediaPlatforms.find((platform) => platform.id === selectedAccount.platformId) : undefined;
  const selectedPlatformType = selectedPlatform?.type ?? '';
  const canAutoPublish = prepared?.preparedPost.platformType === 'xiaohongshu';
  const dialogTitle = canAutoPublish || selectedPlatformType === 'xiaohongshu' ? '发布小红书长文' : '准备媒体平台发布包';

  const handlePrepare = async () => {
    if (!contentId || !mediaAccountId) {
      setError('请选择内容和媒体账号');
      return;
    }
    setSubmitting(true);
    setError(null);
    try {
      const result = await preparePublish(props.token, props.workspaceId, {
        contentId,
        mediaAccountId,
        publishFormatId: selectedPlatformType === 'xiaohongshu' ? 'xiaohongshu_long_article' : 'article',
      });
      setPrepared(result);
      setPublishResult(result.publishResult);
      setPublishTitle(result.preparedPost.title);
      setPublishBody(result.preparedPost.body);
      setExternalUrl('');
      setConfirmationMessage('');
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

  const handleConfirmManualPublish = async () => {
    if (!prepared) {
      return;
    }
    if (!externalUrl.trim()) {
      setError('请填写发布后的外部链接');
      return;
    }
    setRunning(true);
    setError(null);
    try {
      const job = await confirmPublishJob(props.token, props.workspaceId, prepared.job.id, {
        externalUrl: externalUrl.trim(),
        message: confirmationMessage.trim() || '已人工确认发布完成。',
      });
      setPrepared({ ...prepared, job });
      setPublishResult({
        status: 'published',
        message: job.lastMessage || '已人工确认发布完成。',
        externalId: '',
        externalUrl: job.externalUrl,
        rawResponse: {},
      });
      props.onCreated();
    } catch (err) {
      setError(err instanceof Error ? err.message : '人工确认失败');
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
        copyBlocks: prepared.preparedPost.copyBlocks.map((block) => {
          if (block.label.includes('标题')) {
            return { ...block, value: publishTitle };
          }
          if (block.label.includes('正文')) {
            return { ...block, value: publishBody };
          }
          return block;
        }),
      }
    : null;

  return (
    <Dialog open={props.open} onClose={busy ? undefined : props.onClose} fullWidth maxWidth="md">
      <DialogTitle>{dialogTitle}</DialogTitle>
      <DialogContent>
        <Stack spacing={2} sx={{ pt: 1 }}>
          {error && <Alert severity="error">{error}</Alert>}
          {connectedAccounts.length === 0 && <Alert severity="warning">当前工作区还没有已连接的媒体账号。</Alert>}
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
                label="媒体账号"
                value={mediaAccountId}
                onChange={(value) => {
                  setMediaAccountId(value);
                  setPrepared(null);
                }}
                items={connectedAccounts.map((account) => ({ value: account.id, label: `${account.name} / ${platformName(data.mediaPlatforms, account.platformId)}` }))}
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
                label={`${prepared.preparedPost.platformName}标题`}
                value={publishTitle}
                onChange={(event) => setPublishTitle(event.target.value.slice(0, canAutoPublish ? 64 : 80))}
                helperText={`${publishTitle.length}/${canAutoPublish ? 64 : 80}`}
                fullWidth
              />
              <TextField
                label={`${prepared.preparedPost.platformName}正文`}
                value={publishBody}
                onChange={(event) => setPublishBody(event.target.value)}
                helperText={canAutoPublish ? `${publishBody.length} 字。确认后后台会用已登录浏览器打开小红书长文编辑器并点击发布。` : `${publishBody.length} 字。复制发布后填写外部链接完成任务确认。`}
                fullWidth
                multiline
                minRows={10}
              />
              {!canAutoPublish && (
                <Stack spacing={1.5}>
                  <TextField
                    label="发布后的外部链接"
                    value={externalUrl}
                    onChange={(event) => setExternalUrl(event.target.value)}
                    placeholder="https://..."
                    fullWidth
                  />
                  <TextField
                    label="确认说明"
                    value={confirmationMessage}
                    onChange={(event) => setConfirmationMessage(event.target.value)}
                    placeholder="已人工发布并核对链接"
                    fullWidth
                  />
                </Stack>
              )}
              {publishResult && (
                <Alert severity={publishResult.status === 'published' ? 'success' : 'info'}>
                  {publishResult.message}
                  {publishResult.externalId ? ` 笔记 ID：${publishResult.externalId}` : ''}
                  {publishResult.externalUrl ? ` 链接：${publishResult.externalUrl}` : ''}
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
          <Button onClick={handlePrepare} disabled={busy || connectedAccounts.length === 0} variant="contained">
            生成发布包
          </Button>
        )}
        {prepared && canAutoPublish && (
          <Button onClick={handleRun} disabled={busy || !publishTitle.trim() || !publishBody.trim()} variant="contained">
            确认发布
          </Button>
        )}
        {prepared && !canAutoPublish && (
          <Button onClick={handleConfirmManualPublish} disabled={busy || !externalUrl.trim()} variant="contained">
            确认人工发布
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

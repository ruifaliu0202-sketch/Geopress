import { useEffect, useMemo, useState } from 'react';
import {
  Admin,
  BooleanField,
  BooleanInput,
  ChipField,
  Create,
  CustomRoutes,
  Datagrid,
  DateField,
  DateTimeInput,
  Edit,
  EditButton,
  FunctionField,
  Layout,
  List,
  Menu,
  NumberInput,
  ReferenceArrayField,
  ReferenceArrayInput,
  ReferenceField,
  required,
  Resource,
  SelectArrayInput,
  SelectInput,
  SimpleForm,
  SingleFieldList,
  TextField,
  TextInput,
  useGetList,
  type CreateProps,
  type EditProps,
  type ListProps,
} from 'react-admin';
import { Alert, Box, Button, Card, CardContent, Divider, FormControlLabel, Grid, MenuItem, Stack, Switch, TextField as MuiTextField, Typography } from '@mui/material';
import { Route } from 'react-router-dom';
import ArrowBackIcon from '@mui/icons-material/ArrowBack';
import ConnectedTvOutlinedIcon from '@mui/icons-material/ConnectedTvOutlined';
import AutoAwesomeOutlinedIcon from '@mui/icons-material/AutoAwesomeOutlined';
import GroupOutlinedIcon from '@mui/icons-material/GroupOutlined';
import LanOutlinedIcon from '@mui/icons-material/LanOutlined';
import ManageAccountsOutlinedIcon from '@mui/icons-material/ManageAccountsOutlined';
import Inventory2OutlinedIcon from '@mui/icons-material/Inventory2Outlined';
import ArticleOutlinedIcon from '@mui/icons-material/ArticleOutlined';
import WorkspacesOutlinedIcon from '@mui/icons-material/WorkspacesOutlined';
import { createAdminDataProvider, fetchAdminAIConfig, updateAdminAIConfig, type AdminAIConfig } from './dataProvider';
import type { GenerationPipelinePlan, GenerationPipelineSettings } from '../types';

export function AdminConsole({ token, onBack }: { token: string; onBack: () => void }) {
  const dataProvider = useMemo(() => createAdminDataProvider(token), [token]);

  return (
    <Box sx={{ minHeight: '100vh', bgcolor: 'background.default' }}>
      <Box sx={{ position: 'fixed', top: 12, right: 16, zIndex: 1500 }}>
        <Button startIcon={<ArrowBackIcon />} variant="contained" onClick={onBack}>
          返回工作台
        </Button>
      </Box>
      <Admin
        dataProvider={dataProvider}
        dashboard={AdminDashboard}
        layout={AdminLayout}
        title="Geopress 平台后台"
        disableTelemetry
      >
        <CustomRoutes>
          <Route path="/ai-config" element={<AIConfigPage token={token} />} />
        </CustomRoutes>
        <Resource name="platformKnowledgeBases" options={{ label: '平台知识库' }} list={PlatformKnowledgeBaseList} create={PlatformKnowledgeBaseCreate} edit={PlatformKnowledgeBaseEdit} icon={Inventory2OutlinedIcon} />
        <Resource name="platformKnowledgeItems" options={{ label: '平台知识条目' }} list={PlatformKnowledgeItemList} create={PlatformKnowledgeItemCreate} edit={PlatformKnowledgeItemEdit} icon={ArticleOutlinedIcon} />
        <Resource name="mediaPlatforms" options={{ label: '媒体平台' }} list={MediaPlatformList} edit={MediaPlatformEdit} icon={LanOutlinedIcon} />
        <Resource name="users" options={{ label: '用户' }} list={UserList} edit={UserEdit} icon={GroupOutlinedIcon} />
        <Resource name="workspaces" options={{ label: '工作区' }} list={WorkspaceList} icon={WorkspacesOutlinedIcon} />
        <Resource name="members" options={{ label: '成员' }} list={MemberList} icon={ManageAccountsOutlinedIcon} />
        <Resource name="mediaAccounts" options={{ label: '租户账号' }} list={MediaAccountList} icon={ConnectedTvOutlinedIcon} />
      </Admin>
    </Box>
  );
}

function AdminLayout(props: { children: React.ReactNode }) {
  return <Layout {...props} menu={AdminMenu} />;
}

function AdminMenu() {
  return (
    <Menu>
      <Menu.DashboardItem />
      <Menu.Item to="/ai-config" primaryText="AI 配置" leftIcon={<AutoAwesomeOutlinedIcon />} />
      <Menu.ResourceItem name="platformKnowledgeBases" />
      <Menu.ResourceItem name="platformKnowledgeItems" />
      <Menu.ResourceItem name="mediaPlatforms" />
      <Menu.ResourceItem name="users" />
      <Menu.ResourceItem name="workspaces" />
      <Menu.ResourceItem name="members" />
      <Menu.ResourceItem name="mediaAccounts" />
    </Menu>
  );
}

function AdminDashboard() {
  const users = useGetList('users', { pagination: { page: 1, perPage: 1 } });
  const workspaces = useGetList('workspaces', { pagination: { page: 1, perPage: 1 } });
  const platformKnowledgeBases = useGetList('platformKnowledgeBases', { pagination: { page: 1, perPage: 1 } });
  const platformKnowledgeItems = useGetList('platformKnowledgeItems', { pagination: { page: 1, perPage: 1 } });
  const platforms = useGetList('mediaPlatforms', { pagination: { page: 1, perPage: 1 } });
  const accounts = useGetList('mediaAccounts', { pagination: { page: 1, perPage: 1 } });

  return (
    <Box sx={{ p: 3 }}>
      <Stack spacing={3}>
        <Box>
          <Typography variant="h1">平台管理后台</Typography>
          <Typography color="text.secondary" sx={{ mt: 1 }}>
            维护平台知识市场、全局媒体渠道、账号体系、工作区和租户账号资源。
          </Typography>
        </Box>
        <Grid container spacing={2}>
          <AdminMetric label="用户" value={users.total ?? 0} />
          <AdminMetric label="工作区" value={workspaces.total ?? 0} />
          <AdminMetric label="平台知识库" value={platformKnowledgeBases.total ?? 0} />
          <AdminMetric label="平台知识条目" value={platformKnowledgeItems.total ?? 0} />
          <AdminMetric label="媒体渠道" value={platforms.total ?? 0} />
          <AdminMetric label="租户账号" value={accounts.total ?? 0} />
        </Grid>
      </Stack>
    </Box>
  );
}

function AIConfigPage({ token }: { token: string }) {
  const [config, setConfig] = useState<AdminAIConfig | null>(null);
  const [provider, setProvider] = useState<'mock' | 'openai'>('mock');
  const [openAIBaseUrl, setOpenAIBaseUrl] = useState('https://api.openai.com/v1');
  const [openAIModel, setOpenAIModel] = useState('gpt-5.5');
  const [openAIAPIKey, setOpenAIAPIKey] = useState('');
  const [requestTimeoutSeconds, setRequestTimeoutSeconds] = useState(45);
  const [generationPipeline, setGenerationPipeline] = useState<GenerationPipelineSettings>(defaultGenerationPipelineSettings);
  const [clearAPIKey, setClearAPIKey] = useState(false);
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [notice, setNotice] = useState<string | null>(null);

  useEffect(() => {
    let active = true;
    setLoading(true);
    fetchAdminAIConfig(token)
      .then((next) => {
        if (!active) {
          return;
        }
        setConfig(next);
        setProvider(next.provider);
        setOpenAIBaseUrl(next.openAIBaseUrl);
        setOpenAIModel(next.openAIModel);
        setRequestTimeoutSeconds(next.requestTimeoutSeconds);
        setGenerationPipeline(normalizeGenerationPipelineSettings(next.generationPipeline));
        setOpenAIAPIKey('');
        setClearAPIKey(false);
        setError(null);
      })
      .catch((err) => {
        if (active) {
          setError(err instanceof Error ? err.message : '加载 AI 配置失败');
        }
      })
      .finally(() => {
        if (active) {
          setLoading(false);
        }
      });
    return () => {
      active = false;
    };
  }, [token]);

  const submit = async () => {
    if (provider === 'openai' && !openAIBaseUrl.trim()) {
      setError('请填写 OpenAI Base URL');
      return;
    }
    if (provider === 'openai' && !openAIModel.trim()) {
      setError('请填写模型名称');
      return;
    }
    setSaving(true);
    setError(null);
    setNotice(null);
    try {
      const updated = await updateAdminAIConfig(token, {
        provider,
        openAIBaseUrl: openAIBaseUrl.trim(),
        openAIModel: openAIModel.trim(),
        openAIAPIKey: openAIAPIKey.trim() || undefined,
        requestTimeoutSeconds,
        clearAPIKey,
        generationPipeline,
      });
      setConfig(updated);
      setProvider(updated.provider);
      setOpenAIBaseUrl(updated.openAIBaseUrl);
      setOpenAIModel(updated.openAIModel);
      setRequestTimeoutSeconds(updated.requestTimeoutSeconds);
      setGenerationPipeline(normalizeGenerationPipelineSettings(updated.generationPipeline));
      setOpenAIAPIKey('');
      setClearAPIKey(false);
      setNotice('AI 配置已更新');
    } catch (err) {
      setError(err instanceof Error ? err.message : '保存 AI 配置失败');
    } finally {
      setSaving(false);
    }
  };

  return (
    <Box sx={{ p: 3 }}>
      <Stack spacing={3} sx={{ maxWidth: 820 }}>
        <Box>
          <Typography variant="h1">AI 配置</Typography>
          <Typography color="text.secondary" sx={{ mt: 1 }}>
            配置平台级 AI Provider、模型和 API Key。密钥保存后不会回显。
          </Typography>
        </Box>

        {error && <Alert severity="error">{error}</Alert>}
        {notice && <Alert severity="success">{notice}</Alert>}

        <Card>
          <CardContent>
            <Stack spacing={2.25}>
              <MuiTextField select label="Provider" value={provider} onChange={(event) => setProvider(event.target.value as 'mock' | 'openai')} fullWidth disabled={loading || saving}>
                <MenuItem value="mock">Mock Provider</MenuItem>
                <MenuItem value="openai">OpenAI</MenuItem>
              </MuiTextField>

              <MuiTextField label="OpenAI Base URL" value={openAIBaseUrl} onChange={(event) => setOpenAIBaseUrl(event.target.value)} fullWidth disabled={loading || saving || provider !== 'openai'} />
              <MuiTextField label="OpenAI Model" value={openAIModel} onChange={(event) => setOpenAIModel(event.target.value)} fullWidth disabled={loading || saving || provider !== 'openai'} />
              <MuiTextField
                label="OpenAI API Key"
                type="password"
                value={openAIAPIKey}
                onChange={(event) => setOpenAIAPIKey(event.target.value)}
                helperText={config?.apiKeyConfigured ? `当前已配置：${config.apiKeyPreview}` : '当前未配置密钥'}
                fullWidth
                disabled={loading || saving || provider !== 'openai' || clearAPIKey}
              />
              <MuiTextField
                label="请求超时秒数"
                type="number"
                value={requestTimeoutSeconds}
                onChange={(event) => setRequestTimeoutSeconds(Number(event.target.value))}
                inputProps={{ min: 1, max: 180 }}
                fullWidth
                disabled={loading || saving}
              />
              <FormControlLabel
                control={<Switch checked={clearAPIKey} onChange={(event) => setClearAPIKey(event.target.checked)} disabled={loading || saving} />}
                label="清除已保存的 API Key"
              />
              <Divider />
              <Typography fontWeight={700}>生成链路</Typography>
              <PipelinePlanEditor
                label="Free"
                value={generationPipeline.free}
                disabled={loading || saving}
                onChange={(next) => setGenerationPipeline((current) => ({ ...current, free: next }))}
              />
              <PipelinePlanEditor
                label="VIP"
                value={generationPipeline.vip}
                disabled={loading || saving}
                onChange={(next) => setGenerationPipeline((current) => ({ ...current, vip: next }))}
              />
              <Stack direction="row" spacing={1.5}>
                <Button variant="contained" onClick={submit} disabled={loading || saving}>
                  保存配置
                </Button>
              </Stack>
            </Stack>
          </CardContent>
        </Card>
      </Stack>
    </Box>
  );
}

const defaultGenerationPipelineSettings: GenerationPipelineSettings = {
  free: { inputAnalysis: true, contentPlan: false, qualityCheck: false, rewriteRounds: 0 },
  vip: { inputAnalysis: true, contentPlan: true, qualityCheck: true, rewriteRounds: 1 },
};

function normalizeGenerationPipelineSettings(value?: GenerationPipelineSettings): GenerationPipelineSettings {
  return {
    free: { ...defaultGenerationPipelineSettings.free, ...(value?.free ?? {}) },
    vip: { ...defaultGenerationPipelineSettings.vip, ...(value?.vip ?? {}) },
  };
}

function PipelinePlanEditor({
  label,
  value,
  disabled,
  onChange,
}: {
  label: string;
  value: GenerationPipelinePlan;
  disabled: boolean;
  onChange: (value: GenerationPipelinePlan) => void;
}) {
  return (
    <Box sx={{ border: '1px solid', borderColor: 'divider', borderRadius: 1, p: 1.5 }}>
      <Stack spacing={1.25}>
        <Typography fontWeight={700}>{label}</Typography>
        <Stack direction={{ xs: 'column', sm: 'row' }} spacing={1.5}>
          <FormControlLabel
            control={<Switch checked={value.inputAnalysis} onChange={(event) => onChange({ ...value, inputAnalysis: event.target.checked })} disabled={disabled} />}
            label="输入分析"
          />
          <FormControlLabel
            control={<Switch checked={value.contentPlan} onChange={(event) => onChange({ ...value, contentPlan: event.target.checked })} disabled={disabled} />}
            label="创作计划"
          />
          <FormControlLabel
            control={<Switch checked={value.qualityCheck} onChange={(event) => onChange({ ...value, qualityCheck: event.target.checked })} disabled={disabled} />}
            label="质量检查"
          />
        </Stack>
        <MuiTextField
          label="自动重写轮次"
          type="number"
          value={value.rewriteRounds}
          onChange={(event) => onChange({ ...value, rewriteRounds: Math.max(0, Math.min(3, Number(event.target.value))) })}
          inputProps={{ min: 0, max: 3 }}
          fullWidth
          disabled={disabled}
        />
      </Stack>
    </Box>
  );
}

function AdminMetric({ label, value }: { label: string; value: number }) {
  return (
    <Grid size={{ xs: 12, sm: 6, lg: 3 }}>
      <Card>
        <CardContent>
          <Typography color="text.secondary">{label}</Typography>
          <Typography variant="h1" sx={{ mt: 1 }}>
            {value}
          </Typography>
        </CardContent>
      </Card>
    </Grid>
  );
}

function PlatformKnowledgeBaseList(props: ListProps) {
  return (
    <List {...props} perPage={25} sort={{ field: 'name', order: 'ASC' }}>
      <Datagrid rowClick="edit" bulkActionButtons={false}>
        <TextField source="name" label="知识库名称" />
        <TextField source="category" label="分类" />
        <FunctionField label="价格" render={(record) => formatPrice(record?.priceCents, record?.currency)} />
        <BooleanField source="marketplaceListed" label="市场上架" />
        <TextField source="itemCount" label="条目数" />
        <DateField source="updatedAt" label="更新时间" showTime />
        <EditButton label="编辑" />
      </Datagrid>
    </List>
  );
}

function PlatformKnowledgeBaseCreate(props: CreateProps) {
  return (
    <Create {...props}>
      <PlatformKnowledgeBaseForm />
    </Create>
  );
}

function PlatformKnowledgeBaseEdit(props: EditProps) {
  return (
    <Edit {...props} mutationMode="pessimistic">
      <PlatformKnowledgeBaseForm />
    </Edit>
  );
}

function PlatformKnowledgeBaseForm() {
  return (
    <SimpleForm defaultValues={{ category: 'general', currency: 'CNY', priceCents: 0, marketplaceListed: false }}>
      <TextInput source="name" label="知识库名称" validate={required()} fullWidth />
      <TextInput source="description" label="说明" multiline minRows={3} fullWidth />
      <TextInput source="category" label="分类" validate={required()} helperText="例如 小红书、SEO、本地生活、合规" fullWidth />
      <NumberInput source="priceCents" label="价格（分）" min={0} step={100} fullWidth />
      <TextInput source="currency" label="币种" helperText="例如 CNY、USD" fullWidth />
      <BooleanInput source="marketplaceListed" label="上架到市场" />
    </SimpleForm>
  );
}

function PlatformKnowledgeItemList(props: ListProps) {
  return (
    <List {...props} perPage={25} sort={{ field: 'updatedAt', order: 'DESC' }}>
      <Datagrid rowClick="edit" bulkActionButtons={false}>
        <TextField source="title" label="条目标题" />
        <ReferenceArrayField source="knowledgeBaseIds" reference="platformKnowledgeBases" label="知识库包">
          <SingleFieldList linkType={false}>
            <ChipField source="name" />
          </SingleFieldList>
        </ReferenceArrayField>
        <ChipField source="type" label="类型" />
        <BooleanField source="enabled" label="启用" />
        <DateField source="updatedAt" label="更新时间" showTime />
        <EditButton label="编辑" />
      </Datagrid>
    </List>
  );
}

function PlatformKnowledgeItemCreate(props: CreateProps) {
  return (
    <Create {...props}>
      <PlatformKnowledgeItemForm />
    </Create>
  );
}

function PlatformKnowledgeItemEdit(props: EditProps) {
  return (
    <Edit {...props} mutationMode="pessimistic">
      <PlatformKnowledgeItemForm />
    </Edit>
  );
}

function PlatformKnowledgeItemForm() {
  return (
    <SimpleForm defaultValues={{ type: 'note', enabled: true }}>
      <ReferenceArrayInput source="knowledgeBaseIds" reference="platformKnowledgeBases" label="所属平台知识库包" perPage={100}>
        <SelectArrayInput optionText="name" validate={required()} fullWidth />
      </ReferenceArrayInput>
      <TextInput source="type" label="条目类型" validate={required()} helperText="例如 structure、template、compliance、style" fullWidth />
      <TextInput source="title" label="标题" validate={required()} fullWidth />
      <TextInput source="content" label="内容" validate={required()} multiline minRows={8} fullWidth />
      <BooleanInput source="enabled" label="启用" />
    </SimpleForm>
  );
}

function MediaPlatformList(props: ListProps) {
  return (
    <List {...props} perPage={25} sort={{ field: 'name', order: 'ASC' }}>
      <Datagrid rowClick="edit" bulkActionButtons={false}>
        <TextField source="name" label="渠道名称" />
        <TextField source="type" label="类型" />
        <BooleanField source="enabled" label="启用" />
        <BooleanField source="supportsArticle" label="文章" />
        <BooleanField source="supportsImage" label="图片" />
        <BooleanField source="supportsScheduling" label="定时" />
        <FunctionField label="凭证字段" render={(record) => (record.credentialFields ?? []).join(', ')} />
        <EditButton label="编辑" />
      </Datagrid>
    </List>
  );
}

function MediaPlatformEdit(props: EditProps) {
  return (
    <Edit {...props} mutationMode="pessimistic">
      <MediaPlatformForm />
    </Edit>
  );
}

function MediaPlatformForm() {
  return (
    <SimpleForm defaultValues={{ name: '网易号', type: 'netease', enabled: true, supportsArticle: true, supportsImage: true, supportsScheduling: false, credentialFields: ['qrLogin'] }}>
      <TextInput source="name" label="平台名称" validate={required()} fullWidth />
      <TextInput source="type" label="平台类型" validate={required()} helperText="支持 xiaohongshu、netease、toutiao、sohu" fullWidth />
      <TextInput
        source="credentialFields"
        label="凭证字段"
        format={formatCredentialFields}
        parse={parseCredentialFields}
        helperText="用英文逗号分隔，例如 qrLogin"
        fullWidth
      />
      <BooleanInput source="enabled" label="启用" />
      <BooleanInput source="supportsArticle" label="支持文章" />
      <BooleanInput source="supportsImage" label="支持图片" />
      <BooleanInput source="supportsScheduling" label="支持定时发布" />
    </SimpleForm>
  );
}

function formatCredentialFields(value: unknown) {
  return Array.isArray(value) ? value.join(', ') : String(value ?? '');
}

function parseCredentialFields(value: unknown) {
  if (Array.isArray(value)) {
    return value.map(String).map((item) => item.trim()).filter(Boolean);
  }
  return String(value ?? '')
    .split(/[,，\n]/)
    .map((item) => item.trim())
    .filter(Boolean);
}

function formatPrice(priceCents: unknown, currency: unknown) {
  const value = Number(priceCents ?? 0) / 100;
  const currencyCode = String(currency ?? 'CNY') || 'CNY';
  return `${value.toFixed(2)} ${currencyCode}`;
}

function UserList(props: ListProps) {
  return (
    <List {...props} perPage={25} sort={{ field: 'email', order: 'ASC' }}>
      <Datagrid rowClick="edit" bulkActionButtons={false}>
        <TextField source="name" label="姓名" />
        <TextField source="email" label="邮箱" />
        <TextField source="subscriptionTier" label="订阅等级" />
        <TextField source="subscriptionStatus" label="订阅状态" />
        <DateField source="subscriptionExpiresAt" label="订阅到期" showTime />
        <BooleanField source="isPlatformAdmin" label="平台管理员" />
        <DateField source="createdAt" label="创建时间" showTime />
        <EditButton label="编辑订阅" />
      </Datagrid>
    </List>
  );
}

function UserEdit(props: EditProps) {
  return (
    <Edit {...props} mutationMode="pessimistic">
      <SimpleForm>
        <TextInput source="name" label="姓名" disabled fullWidth />
        <TextInput source="email" label="邮箱" disabled fullWidth />
        <SelectInput
          source="subscriptionTier"
          label="订阅等级"
          choices={[
            { id: 'free', name: 'free' },
            { id: 'vip', name: 'vip' },
          ]}
          validate={required()}
          fullWidth
        />
        <SelectInput
          source="subscriptionStatus"
          label="订阅状态"
          choices={[
            { id: 'active', name: 'active' },
            { id: 'inactive', name: 'inactive' },
            { id: 'expired', name: 'expired' },
            { id: 'canceled', name: 'canceled' },
          ]}
          validate={required()}
          fullWidth
        />
        <DateTimeInput source="subscriptionExpiresAt" label="订阅到期" fullWidth />
      </SimpleForm>
    </Edit>
  );
}

function WorkspaceList(props: ListProps) {
  return (
    <List {...props} perPage={25} sort={{ field: 'name', order: 'ASC' }}>
      <Datagrid rowClick={false} bulkActionButtons={false}>
        <TextField source="name" label="名称" />
        <ChipField source="type" label="类型" />
        <TextField source="plan" label="工作区方案" />
        <TextField source="industry" label="行业" />
        <TextField source="language" label="语言" />
        <DateField source="createdAt" label="创建时间" showTime />
      </Datagrid>
    </List>
  );
}

function MemberList(props: ListProps) {
  return (
    <List {...props} perPage={25} sort={{ field: 'workspaceId', order: 'ASC' }}>
      <Datagrid rowClick={false} bulkActionButtons={false}>
        <ReferenceField source="workspaceId" reference="workspaces" label="工作区">
          <TextField source="name" />
        </ReferenceField>
        <ReferenceField source="userId" reference="users" label="用户">
          <TextField source="email" />
        </ReferenceField>
        <ChipField source="role" label="角色" />
      </Datagrid>
    </List>
  );
}

function MediaAccountList(props: ListProps) {
  return (
    <List {...props} perPage={25} sort={{ field: 'name', order: 'ASC' }}>
      <Datagrid rowClick={false} bulkActionButtons={false}>
        <TextField source="name" label="账号名称" />
        <ReferenceField source="workspaceId" reference="workspaces" label="工作区">
          <TextField source="name" />
        </ReferenceField>
        <ReferenceField source="platformId" reference="mediaPlatforms" label="渠道">
          <TextField source="name" />
        </ReferenceField>
        <FunctionField label="登录方式" render={(record) => loginMethodLabel(record?.loginMethod)} />
        <FunctionField label="登录凭证" render={(record) => accountLoginCredential(record)} />
        <TextField source="externalId" label="外部标识" />
        <ChipField source="status" label="状态" />
        <DateField source="lastCheckedAt" label="检查时间" showTime />
      </Datagrid>
    </List>
  );
}

function loginMethodLabel(value: unknown) {
  if (value === 'qr') {
    return '二维码登录';
  }
  if (value === 'phone') {
    return '手机号登录';
  }
  if (value === 'manual' || !value) {
    return '手动授权';
  }
  return String(value);
}

function accountLoginCredential(record: Record<string, unknown> | undefined) {
  if (record?.loginMethod === 'qr') {
    return '服务端二维码';
  }
  const meta = record?.credentialMeta;
  if (!meta || typeof meta !== 'object' || !('phoneNumber' in meta)) {
    return '-';
  }
  return String((meta as Record<string, unknown>).phoneNumber || '-');
}

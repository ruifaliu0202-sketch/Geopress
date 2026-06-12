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
  FunctionField,
  Layout,
  List,
  Menu,
  ReferenceField,
  required,
  Resource,
  SimpleForm,
  TextField,
  TextInput,
  useGetList,
  type CreateProps,
  type ListProps,
} from 'react-admin';
import { Alert, Box, Button, Card, CardContent, FormControlLabel, Grid, MenuItem, Stack, Switch, TextField as MuiTextField, Typography } from '@mui/material';
import { Route } from 'react-router-dom';
import ArrowBackIcon from '@mui/icons-material/ArrowBack';
import ConnectedTvOutlinedIcon from '@mui/icons-material/ConnectedTvOutlined';
import AutoAwesomeOutlinedIcon from '@mui/icons-material/AutoAwesomeOutlined';
import GroupOutlinedIcon from '@mui/icons-material/GroupOutlined';
import LanOutlinedIcon from '@mui/icons-material/LanOutlined';
import ManageAccountsOutlinedIcon from '@mui/icons-material/ManageAccountsOutlined';
import WorkspacesOutlinedIcon from '@mui/icons-material/WorkspacesOutlined';
import { createAdminDataProvider, fetchAdminAIConfig, updateAdminAIConfig, type AdminAIConfig } from './dataProvider';

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
        <Resource name="mediaPlatforms" options={{ label: '媒体渠道' }} list={MediaPlatformList} create={MediaPlatformCreate} icon={LanOutlinedIcon} />
        <Resource name="users" options={{ label: '用户' }} list={UserList} icon={GroupOutlinedIcon} />
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
  const platforms = useGetList('mediaPlatforms', { pagination: { page: 1, perPage: 1 } });
  const accounts = useGetList('mediaAccounts', { pagination: { page: 1, perPage: 1 } });

  return (
    <Box sx={{ p: 3 }}>
      <Stack spacing={3}>
        <Box>
          <Typography variant="h1">平台管理后台</Typography>
          <Typography color="text.secondary" sx={{ mt: 1 }}>
            维护全局媒体渠道、账号体系、工作区和租户账号资源。
          </Typography>
        </Box>
        <Grid container spacing={2}>
          <AdminMetric label="用户" value={users.total ?? 0} />
          <AdminMetric label="工作区" value={workspaces.total ?? 0} />
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
      });
      setConfig(updated);
      setProvider(updated.provider);
      setOpenAIBaseUrl(updated.openAIBaseUrl);
      setOpenAIModel(updated.openAIModel);
      setRequestTimeoutSeconds(updated.requestTimeoutSeconds);
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

function MediaPlatformList(props: ListProps) {
  return (
    <List {...props} perPage={25} sort={{ field: 'name', order: 'ASC' }}>
      <Datagrid rowClick={false} bulkActionButtons={false}>
        <TextField source="name" label="渠道名称" />
        <TextField source="type" label="类型" />
        <BooleanField source="enabled" label="启用" />
        <BooleanField source="supportsArticle" label="文章" />
        <BooleanField source="supportsImage" label="图片" />
        <BooleanField source="supportsScheduling" label="定时" />
        <FunctionField label="凭证字段" render={(record) => (record.credentialFields ?? []).join(', ')} />
      </Datagrid>
    </List>
  );
}

function MediaPlatformCreate(props: CreateProps) {
  return (
    <Create {...props}>
      <SimpleForm defaultValues={{ enabled: true, supportsArticle: true, supportsImage: true, supportsScheduling: false }}>
        <TextInput source="name" label="渠道名称" validate={required()} fullWidth />
        <TextInput source="type" label="渠道类型" validate={required()} fullWidth />
        <TextInput source="credentialFields" label="凭证字段" helperText="用英文逗号分隔，例如 accessToken,appSecret" fullWidth />
        <BooleanInput source="enabled" label="启用" />
        <BooleanInput source="supportsArticle" label="支持文章" />
        <BooleanInput source="supportsImage" label="支持图片" />
        <BooleanInput source="supportsScheduling" label="支持定时发布" />
      </SimpleForm>
    </Create>
  );
}

function UserList(props: ListProps) {
  return (
    <List {...props} perPage={25} sort={{ field: 'email', order: 'ASC' }}>
      <Datagrid rowClick={false} bulkActionButtons={false}>
        <TextField source="name" label="姓名" />
        <TextField source="email" label="邮箱" />
        <BooleanField source="isPlatformAdmin" label="平台管理员" />
        <DateField source="createdAt" label="创建时间" showTime />
      </Datagrid>
    </List>
  );
}

function WorkspaceList(props: ListProps) {
  return (
    <List {...props} perPage={25} sort={{ field: 'name', order: 'ASC' }}>
      <Datagrid rowClick={false} bulkActionButtons={false}>
        <TextField source="name" label="名称" />
        <ChipField source="type" label="类型" />
        <TextField source="plan" label="套餐" />
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

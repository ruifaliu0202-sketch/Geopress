import { useMemo } from 'react';
import {
  Admin,
  BooleanField,
  BooleanInput,
  ChipField,
  Create,
  Datagrid,
  DateField,
  FunctionField,
  List,
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
import { Box, Button, Card, CardContent, Grid, Stack, Typography } from '@mui/material';
import ArrowBackIcon from '@mui/icons-material/ArrowBack';
import ConnectedTvOutlinedIcon from '@mui/icons-material/ConnectedTvOutlined';
import GroupOutlinedIcon from '@mui/icons-material/GroupOutlined';
import LanOutlinedIcon from '@mui/icons-material/LanOutlined';
import ManageAccountsOutlinedIcon from '@mui/icons-material/ManageAccountsOutlined';
import WorkspacesOutlinedIcon from '@mui/icons-material/WorkspacesOutlined';
import { createAdminDataProvider } from './dataProvider';

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
        title="Geopress 平台后台"
        disableTelemetry
      >
        <Resource name="mediaPlatforms" options={{ label: '媒体渠道' }} list={MediaPlatformList} create={MediaPlatformCreate} icon={LanOutlinedIcon} />
        <Resource name="users" options={{ label: '用户' }} list={UserList} icon={GroupOutlinedIcon} />
        <Resource name="workspaces" options={{ label: '工作区' }} list={WorkspaceList} icon={WorkspacesOutlinedIcon} />
        <Resource name="members" options={{ label: '成员' }} list={MemberList} icon={ManageAccountsOutlinedIcon} />
        <Resource name="mediaAccounts" options={{ label: '租户账号' }} list={MediaAccountList} icon={ConnectedTvOutlinedIcon} />
      </Admin>
    </Box>
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
        <TextField source="externalId" label="外部标识" />
        <ChipField source="status" label="状态" />
        <DateField source="lastCheckedAt" label="检查时间" showTime />
      </Datagrid>
    </List>
  );
}

import type { ReactNode } from 'react';
import {
  Button,
  Checkbox,
  Chip,
  Link,
  Stack,
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TableRow,
  Typography,
} from '@mui/material';
import LoginOutlinedIcon from '@mui/icons-material/LoginOutlined';
import PublishOutlinedIcon from '@mui/icons-material/PublishOutlined';
import type {
  Content,
  ContentStatus,
  KnowledgeBase,
  KnowledgeItem,
  MediaAccount,
  MediaPlatform,
  PublishJobStatus,
  PublishScheduleFrequency,
  WorkspaceData,
} from '../types';
import {
  accountName,
  contentName,
  credentialFieldLabel,
  formatDate,
  knowledgeBaseNames,
  loginMethodLabel,
  mediaAccountStatusColor,
  mediaAccountStatusLabel,
  supportsBrowserLogin,
  uniqueValues,
} from '../utils/formatters';

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

const wrappingTextSx = {
  minWidth: 0,
  overflowWrap: 'anywhere',
};

const secondaryTextSx = {
  ...wrappingTextSx,
  display: '-webkit-box',
  WebkitBoxOrient: 'vertical',
  WebkitLineClamp: 2,
  overflow: 'hidden',
} as const;

function DataTableFrame({
  children,
  minWidth = 760,
  size = 'medium',
}: {
  children: ReactNode;
  minWidth?: number;
  size?: 'small' | 'medium';
}) {
  return (
    <TableContainer sx={{ width: '100%', overflowX: 'auto', WebkitOverflowScrolling: 'touch' }}>
      <Table
        size={size}
        sx={{
          minWidth,
          '& .MuiTableCell-root': {
            verticalAlign: 'top',
            overflowWrap: 'anywhere',
          },
          '& .MuiTableCell-head': {
            whiteSpace: 'nowrap',
          },
        }}
      >
        {children}
      </Table>
    </TableContainer>
  );
}

function EmptyTableRow({ colSpan, label }: { colSpan: number; label: string }) {
  return (
    <TableRow>
      <TableCell colSpan={colSpan} sx={{ py: 4, textAlign: 'center' }}>
        <Typography color="text.secondary">{label}</Typography>
      </TableCell>
    </TableRow>
  );
}

export function KnowledgeItemsTable({
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
    <DataTableFrame minWidth={selectable ? 820 : 760}>
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
        {items.length === 0 && <EmptyTableRow colSpan={selectable ? 6 : 5} label="暂无知识条目" />}
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
              <Typography fontWeight={700} sx={wrappingTextSx}>
                {item.title}
              </Typography>
              <Typography variant="body2" color="text.secondary" sx={{ ...secondaryTextSx, maxWidth: 520 }}>
                {item.content}
              </Typography>
            </TableCell>
            <TableCell sx={{ minWidth: 140 }}>{knowledgeBaseNames(bases, item.knowledgeBaseIds)}</TableCell>
            <TableCell sx={{ minWidth: 96 }}>{item.type}</TableCell>
            <TableCell>
              <Chip size="small" label={item.enabled ? '启用' : '停用'} color={item.enabled ? 'success' : 'default'} />
            </TableCell>
            <TableCell>{formatDate(item.updatedAt)}</TableCell>
          </TableRow>
        ))}
      </TableBody>
    </DataTableFrame>
  );
}

export function MediaPlatformTable({ platforms }: { platforms: MediaPlatform[] }) {
  return (
    <DataTableFrame minWidth={720}>
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
        {platforms.length === 0 && <EmptyTableRow colSpan={5} label="暂无可绑定媒体平台" />}
        {platforms.map((platform) => (
          <TableRow key={platform.id} hover>
            <TableCell sx={{ minWidth: 120 }}>{platform.name}</TableCell>
            <TableCell sx={{ minWidth: 112 }}>{platform.type}</TableCell>
            <TableCell>
              <Stack direction="row" spacing={0.5} flexWrap="wrap" useFlexGap>
                {platform.supportsArticle && <Chip size="small" label="文章" />}
                {platform.supportsImage && <Chip size="small" label="图片" />}
                {platform.supportsScheduling && <Chip size="small" label="定时" />}
              </Stack>
            </TableCell>
            <TableCell>
              <Chip size="small" label={platform.enabled ? '启用' : '停用'} color={platform.enabled ? 'success' : 'default'} />
            </TableCell>
            <TableCell sx={{ minWidth: 180 }}>{platform.credentialFields.map(credentialFieldLabel).join(', ')}</TableCell>
          </TableRow>
        ))}
      </TableBody>
    </DataTableFrame>
  );
}

export function MediaAccountsTable({
  accounts,
  platforms,
  onLogin,
}: {
  accounts: MediaAccount[];
  platforms: MediaPlatform[];
  onLogin?: (accountId: string) => void;
}) {
  return (
    <DataTableFrame minWidth={onLogin ? 920 : 840}>
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
        {accounts.length === 0 && <EmptyTableRow colSpan={onLogin ? 8 : 7} label="暂无绑定媒体账号" />}
        {accounts.map((account) => {
          const platform = platforms.find((item) => item.id === account.platformId);
          const canLogin = supportsBrowserLogin(platform?.type) && account.loginMethod === 'qr' && account.status !== 'connected';
          return (
            <TableRow key={account.id} hover>
              <TableCell sx={{ minWidth: 140 }}>{account.name}</TableCell>
              <TableCell sx={{ minWidth: 120 }}>{platform?.name ?? account.platformId}</TableCell>
              <TableCell>{loginMethodLabel(account.loginMethod)}</TableCell>
              <TableCell sx={{ minWidth: 140 }}>{account.loginMethod === 'qr' ? '服务端二维码' : account.credentialMeta?.phoneNumber ?? '-'}</TableCell>
              <TableCell sx={{ minWidth: 128 }}>{account.externalId || '-'}</TableCell>
              <TableCell>
                <Chip size="small" label={mediaAccountStatusLabel(account.status)} color={mediaAccountStatusColor(account.status)} />
              </TableCell>
              <TableCell>{formatDate(account.lastCheckedAt)}</TableCell>
              {onLogin && (
                <TableCell align="right">
                  {canLogin ? (
                    <Button size="small" startIcon={<LoginOutlinedIcon />} onClick={() => onLogin(account.id)} sx={{ whiteSpace: 'nowrap' }}>
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
    </DataTableFrame>
  );
}

export function ContentTable({
  contents,
  onPreparePublish,
}: {
  contents: Content[];
  onPreparePublish?: (contentId: string) => void;
}) {
  return (
    <DataTableFrame minWidth={onPreparePublish ? 840 : 760}>
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
        {contents.length === 0 && <EmptyTableRow colSpan={onPreparePublish ? 6 : 5} label="暂无内容" />}
        {contents.map((content) => {
          const status = contentStatusMap[content.status];
          return (
            <TableRow key={content.id} hover>
              <TableCell>
                <Typography fontWeight={700} sx={wrappingTextSx}>
                  {content.title}
                </Typography>
                <Typography variant="body2" color="text.secondary" sx={secondaryTextSx}>
                  {content.summary}
                </Typography>
              </TableCell>
              <TableCell sx={{ minWidth: 160 }}>{content.keywords.join(', ')}</TableCell>
              <TableCell>
                <Chip size="small" label={status.label} color={status.color} />
              </TableCell>
              <TableCell>{content.source}</TableCell>
              <TableCell>{formatDate(content.updatedAt)}</TableCell>
              {onPreparePublish && (
                <TableCell align="right">
                  <Button
                    size="small"
                    startIcon={<PublishOutlinedIcon />}
                    onClick={() => onPreparePublish(content.id)}
                    data-tour-id="content-publish-action"
                    sx={{ whiteSpace: 'nowrap' }}
                  >
                    小红书发布
                  </Button>
                </TableCell>
              )}
            </TableRow>
          );
        })}
      </TableBody>
    </DataTableFrame>
  );
}

export function SchedulesTable({ data }: { data: WorkspaceData }) {
  return (
    <DataTableFrame minWidth={800}>
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
        {data.publishSchedules.length === 0 && <EmptyTableRow colSpan={6} label="暂无发布计划" />}
        {data.publishSchedules.map((schedule) => (
          <TableRow key={schedule.id} hover>
            <TableCell sx={{ minWidth: 140 }}>{schedule.name}</TableCell>
            <TableCell sx={{ minWidth: 160 }}>{contentName(data.contents, schedule.contentId)}</TableCell>
            <TableCell sx={{ minWidth: 140 }}>{accountName(data.mediaAccounts, schedule.mediaAccountId)}</TableCell>
            <TableCell>{frequencyLabel[schedule.frequency]}</TableCell>
            <TableCell>{formatDate(schedule.nextRunAt)}</TableCell>
            <TableCell>
              <Chip size="small" label={schedule.enabled ? '启用' : '暂停'} color={schedule.enabled ? 'success' : 'default'} />
            </TableCell>
          </TableRow>
        ))}
      </TableBody>
    </DataTableFrame>
  );
}

export function JobsTable({ data, dense = false }: { data: WorkspaceData; dense?: boolean }) {
  return (
    <DataTableFrame minWidth={dense ? 720 : 860} size={dense ? 'small' : 'medium'}>
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
        {data.publishJobs.length === 0 && <EmptyTableRow colSpan={6} label="暂无发布任务" />}
        {data.publishJobs.map((job) => {
          const status = jobStatusMap[job.status];
          return (
            <TableRow key={job.id} hover>
              <TableCell sx={{ minWidth: 160 }}>{contentName(data.contents, job.contentId)}</TableCell>
              <TableCell sx={{ minWidth: 140 }}>{accountName(data.mediaAccounts, job.mediaAccountId)}</TableCell>
              <TableCell>
                <Chip size="small" label={status.label} color={status.color} />
              </TableCell>
              <TableCell>{formatDate(job.scheduledAt)}</TableCell>
              <TableCell sx={{ minWidth: 160 }}>{job.lastMessage || '-'}</TableCell>
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
    </DataTableFrame>
  );
}

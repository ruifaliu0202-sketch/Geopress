import {
  Button,
  Checkbox,
  Chip,
  Link,
  Stack,
  Table,
  TableBody,
  TableCell,
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

export function MediaPlatformTable({ platforms }: { platforms: MediaPlatform[] }) {
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

export function ContentTable({
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
                  <Button
                    size="small"
                    startIcon={<PublishOutlinedIcon />}
                    onClick={() => onPreparePublish(content.id)}
                    data-tour-id="content-publish-action"
                  >
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

export function SchedulesTable({ data }: { data: WorkspaceData }) {
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

export function JobsTable({ data, dense = false }: { data: WorkspaceData; dense?: boolean }) {
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

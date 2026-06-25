import type { ReactNode } from 'react';
import {
  Button,
  Chip,
  Checkbox,
  IconButton,
  LinearProgress,
  Link,
  Box,
  Stack,
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TableRow,
  Tooltip,
  Typography,
} from '@mui/material';
import DeleteOutlineIcon from '@mui/icons-material/DeleteOutline';
import FormatListBulletedOutlinedIcon from '@mui/icons-material/FormatListBulletedOutlined';
import LoginOutlinedIcon from '@mui/icons-material/LoginOutlined';
import PublishOutlinedIcon from '@mui/icons-material/PublishOutlined';
import RemoveCircleOutlineOutlinedIcon from '@mui/icons-material/RemoveCircleOutlineOutlined';
import ReplayOutlinedIcon from '@mui/icons-material/ReplayOutlined';
import { VIPFeatureButton } from './common';
import type {
  Content,
  ContentStatus,
  KnowledgeBase,
  KnowledgeAsset,
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
  supportsInteractiveLogin,
  supportsBrowserLogin,
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

const assetStatusMap: Record<string, { label: string; color: 'default' | 'info' | 'error' | 'success' | 'warning' }> = {
  processing: { label: '处理中', color: 'info' },
  ready: { label: '可用', color: 'success' },
  failed: { label: '失败', color: 'error' },
  pending: { label: '待处理', color: 'warning' },
};

const aiEnhancementStatusMap: Record<string, { label: string; color: 'default' | 'info' | 'error' | 'success' | 'warning' }> = {
  disabled: { label: '未启用', color: 'default' },
  pending: { label: '待增强', color: 'warning' },
  processing: { label: '增强中', color: 'info' },
  succeeded: { label: '已增强', color: 'success' },
  ready: { label: '已增强', color: 'success' },
  failed: { label: '增强失败', color: 'error' },
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

const stickyActionCellSx = {
  position: 'sticky',
  right: 0,
  zIndex: 2,
  bgcolor: 'background.paper',
  boxShadow: '-8px 0 12px rgba(15, 23, 42, 0.06)',
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

function normalizeProgress(value: number) {
  if (!Number.isFinite(value)) {
    return 0;
  }
  return Math.max(0, Math.min(100, value));
}

function assetStatusLabel(status: string) {
  return assetStatusMap[status] ?? { label: status || '未知', color: 'default' as const };
}

function aiEnhancementLabel(status: string, enabled: boolean) {
  if (!enabled) {
    return aiEnhancementStatusMap.disabled;
  }
  return aiEnhancementStatusMap[status] ?? { label: status || '待增强', color: 'default' as const };
}

function assetTypeLabel(asset: KnowledgeAsset) {
  const type = asset.assetType || asset.mimeType || 'text';
  const filename = asset.originalFilename?.trim();
  if (filename && filename !== asset.title) {
    return `${type} / ${filename}`;
  }
  return type;
}

function errorSummary(message: string) {
  if (!message) {
    return '无';
  }
  return message.length > 72 ? `${message.slice(0, 72)}...` : message;
}

type KnowledgeAssetActionTarget = 'detail' | 'tips' | 'chunks';

export function KnowledgeAssetsTable({
  assets,
  bases,
  onOpenAsset,
  selectedIds = [],
  onSelectedIdsChange,
  onRemoveFromBase,
  onRetryAsset,
  onEnhanceAsset,
  onTrashAsset,
}: {
  assets: KnowledgeAsset[];
  bases: KnowledgeBase[];
  onOpenAsset?: (asset: KnowledgeAsset, target?: KnowledgeAssetActionTarget) => void;
  selectedIds?: string[];
  onSelectedIdsChange?: (ids: string[]) => void;
  onRemoveFromBase?: (asset: KnowledgeAsset) => void;
  onRetryAsset?: (asset: KnowledgeAsset) => void;
  onEnhanceAsset?: (asset: KnowledgeAsset) => void;
  onTrashAsset?: (asset: KnowledgeAsset) => void;
}) {
  const selectable = Boolean(onSelectedIdsChange);
  const hasActions = Boolean(onOpenAsset || onRemoveFromBase || onRetryAsset || onEnhanceAsset || onTrashAsset);
  const selectedSet = new Set(selectedIds);
  const allSelected = selectable && assets.length > 0 && assets.every((asset) => selectedSet.has(asset.id));
  const someSelected = selectable && assets.some((asset) => selectedSet.has(asset.id)) && !allSelected;
  const columnCount = 8 + (selectable ? 1 : 0) + (hasActions ? 1 : 0);

  return (
    <DataTableFrame minWidth={selectable || hasActions ? 1120 : 960}>
      <TableHead>
        <TableRow>
          {selectable && (
            <TableCell padding="checkbox">
              <Checkbox
                size="small"
                checked={allSelected}
                indeterminate={someSelected}
                onChange={(event) => {
                  const currentVisibleIds = new Set(assets.map((asset) => asset.id));
                  if (event.target.checked) {
                    onSelectedIdsChange?.([...new Set([...selectedIds, ...assets.map((asset) => asset.id)])]);
                    return;
                  }
                  onSelectedIdsChange?.(selectedIds.filter((id) => !currentVisibleIds.has(id)));
                }}
                inputProps={{ 'aria-label': '选择全部知识资产' }}
              />
            </TableCell>
          )}
          <TableCell>标题</TableCell>
          <TableCell>所属知识库</TableCell>
          <TableCell>类型/文件名</TableCell>
          <TableCell>状态</TableCell>
          <TableCell>进度</TableCell>
          <TableCell>AI 增强</TableCell>
          <TableCell>更新时间</TableCell>
          <TableCell>错误信息</TableCell>
          {hasActions && <TableCell align="center" sx={{ ...stickyActionCellSx, zIndex: 3, minWidth: 172, width: 172 }}>操作</TableCell>}
        </TableRow>
      </TableHead>
      <TableBody>
        {assets.length === 0 && <EmptyTableRow colSpan={columnCount} label="暂无知识资产" />}
        {assets.map((asset) => {
          const status = assetStatusLabel(asset.status);
          const enhancement = aiEnhancementLabel(asset.aiEnhancementStatus, asset.aiEnhancementEnabled);
          const progress = normalizeProgress(asset.progress);
          const canRetry = asset.status === 'failed';
          const aiStatus = asset.aiEnhancementStatus || (asset.aiEnhancementEnabled ? 'pending' : 'disabled');
          const aiEnhancementRunning = asset.aiEnhancementEnabled && (aiStatus === 'pending' || aiStatus === 'processing');
          const canEnhance =
            asset.status === 'ready' &&
            onEnhanceAsset &&
            !aiEnhancementRunning &&
            (!asset.aiEnhancementEnabled || aiStatus === 'disabled' || aiStatus === 'failed' || aiStatus === 'skipped');
          return (
            <TableRow
              key={asset.id}
              hover={Boolean(onOpenAsset)}
              onClick={onOpenAsset ? () => onOpenAsset(asset) : undefined}
              sx={{ cursor: onOpenAsset ? 'pointer' : 'default' }}
            >
              {selectable && (
                <TableCell padding="checkbox" onClick={(event) => event.stopPropagation()}>
                  <Checkbox
                    size="small"
                    checked={selectedSet.has(asset.id)}
                    onChange={(event) => {
                      if (event.target.checked) {
                        onSelectedIdsChange?.([...selectedIds, asset.id]);
                        return;
                      }
                      onSelectedIdsChange?.(selectedIds.filter((id) => id !== asset.id));
                    }}
                    inputProps={{ 'aria-label': `选择 ${asset.title}` }}
                  />
                </TableCell>
              )}
              <TableCell sx={{ minWidth: 150, maxWidth: 190 }}>
                <Typography fontWeight={700} sx={{ ...wrappingTextSx, maxWidth: 180 }}>
                  {asset.title}
                </Typography>
                <Typography variant="body2" color="text.secondary" sx={{ ...secondaryTextSx, maxWidth: 180 }}>
                  {asset.id}
                </Typography>
              </TableCell>
              <TableCell sx={{ minWidth: 124, maxWidth: 150 }}>
                <Box sx={{ ...secondaryTextSx, maxWidth: 150 }}>{knowledgeBaseNames(bases, asset.knowledgeBaseIds, '未分类')}</Box>
              </TableCell>
              <TableCell sx={{ minWidth: 132, maxWidth: 160 }}>
                <Box sx={{ ...secondaryTextSx, maxWidth: 160 }}>{assetTypeLabel(asset)}</Box>
              </TableCell>
              <TableCell>
                <Chip size="small" label={status.label} color={status.color} />
              </TableCell>
              <TableCell sx={{ minWidth: 110 }}>
                <Stack spacing={0.75}>
                  <LinearProgress variant="determinate" value={progress} color={asset.status === 'failed' ? 'error' : 'primary'} />
                  <Typography variant="body2" color="text.secondary">
                    {progress}%
                  </Typography>
                </Stack>
              </TableCell>
              <TableCell>
                <Chip size="small" label={enhancement.label} color={enhancement.color} />
              </TableCell>
              <TableCell sx={{ minWidth: 116 }}>{formatDate(asset.updatedAt)}</TableCell>
              <TableCell sx={{ minWidth: 150, maxWidth: 180 }}>
                <Box sx={{ ...secondaryTextSx, maxWidth: 180 }}>{errorSummary(asset.errorMessage)}</Box>
              </TableCell>
              {hasActions && (
                <TableCell onClick={(event) => event.stopPropagation()} align="center" sx={{ ...stickyActionCellSx, minWidth: 172, width: 172 }}>
                  <Stack direction="row" spacing={0.25} justifyContent="center" flexWrap="nowrap">
                    {onOpenAsset && (
                      <Tooltip title="查看知识片段">
                        <IconButton size="small" color="info" aria-label={`查看 ${asset.title} 知识片段`} onClick={() => onOpenAsset(asset, 'chunks')}>
                          <FormatListBulletedOutlinedIcon fontSize="small" />
                        </IconButton>
                      </Tooltip>
                    )}
                    {onRetryAsset && canRetry && (
                      <Tooltip title="重试">
                        <IconButton size="small" aria-label={`重试 ${asset.title}`} onClick={() => onRetryAsset(asset)}>
                          <ReplayOutlinedIcon fontSize="small" />
                        </IconButton>
                      </Tooltip>
                    )}
                    {onEnhanceAsset && (
                      <Tooltip title={aiEnhancementRunning ? 'AI 增强处理中' : canEnhance ? '应用 AI 增强' : '当前状态不可 AI 增强'}>
                        <span>
                          <VIPFeatureButton
                            animateHighlight={canEnhance}
                            selected={canEnhance}
                            size="small"
                            disabled={!canEnhance}
                            aria-label={`AI 增强 ${asset.title}`}
                            onClick={() => onEnhanceAsset(asset)}
                            sx={{ minHeight: 30, px: 1.05, fontSize: 12, whiteSpace: 'nowrap' }}
                          >
                            AI增强
                          </VIPFeatureButton>
                        </span>
                      </Tooltip>
                    )}
                    {onRemoveFromBase && (
                      <Tooltip title="移出当前知识库包">
                        <IconButton size="small" color="warning" aria-label={`移出 ${asset.title}`} onClick={() => onRemoveFromBase(asset)}>
                          <RemoveCircleOutlineOutlinedIcon fontSize="small" />
                        </IconButton>
                      </Tooltip>
                    )}
                    {onTrashAsset && (
                      <Tooltip title="移入垃圾箱">
                        <IconButton size="small" color="error" aria-label={`删除 ${asset.title}`} onClick={() => onTrashAsset(asset)}>
                          <DeleteOutlineIcon fontSize="small" />
                        </IconButton>
                      </Tooltip>
                    )}
                  </Stack>
                </TableCell>
              )}
            </TableRow>
          );
        })}
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
          const canLogin =
            ((supportsBrowserLogin(platform?.type) && account.loginMethod === 'qr') ||
              (supportsInteractiveLogin(platform) && account.loginMethod === 'phone')) &&
            account.status !== 'connected';
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

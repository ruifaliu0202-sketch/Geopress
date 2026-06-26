import type { ReactNode } from 'react';
import { useCallback, useEffect, useState } from 'react';
import {
  Alert,
  Box,
  Button,
  Card,
  CardContent,
  Chip,
  CircularProgress,
  Divider,
  FormControl,
  Grid,
  InputLabel,
  MenuItem,
  Select,
  Stack,
  Tab,
  Tabs,
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TableRow,
  TextField,
  Typography,
} from '@mui/material';
import AddIcon from '@mui/icons-material/Add';
import AutoAwesomeOutlinedIcon from '@mui/icons-material/AutoAwesomeOutlined';
import AutorenewIcon from '@mui/icons-material/Autorenew';
import CheckCircleOutlineIcon from '@mui/icons-material/CheckCircleOutline';
import EditNoteOutlinedIcon from '@mui/icons-material/EditNoteOutlined';
import FactCheckOutlinedIcon from '@mui/icons-material/FactCheckOutlined';
import LibraryBooksOutlinedIcon from '@mui/icons-material/LibraryBooksOutlined';
import LoginOutlinedIcon from '@mui/icons-material/LoginOutlined';
import PlaylistAddCheckOutlinedIcon from '@mui/icons-material/PlaylistAddCheckOutlined';
import PriceCheckOutlinedIcon from '@mui/icons-material/PriceCheckOutlined';
import PublishOutlinedIcon from '@mui/icons-material/PublishOutlined';
import VisibilityOutlinedIcon from '@mui/icons-material/VisibilityOutlined';
import {
  createAgencyClientRelation,
  createApprovalWorkflow,
  createBrandAsset,
  createBrandGuardrail,
  createCampaign,
  createCampaignCalendarItem,
  createCreatorCampaignBrief,
  createCreatorOrder,
  createCreatorShortlist,
  fetchAgencyClientRelations,
  fetchApprovalTasks,
  fetchApprovalWorkflows,
  fetchBrandAssets,
  fetchBrandGuardrails,
  fetchCampaignCalendarItems,
  fetchCampaignReportSummary,
  fetchCampaigns,
  fetchComplianceChecks,
  fetchCreators,
  fetchCreatorCampaignBriefs,
  fetchCreatorDeliverables,
  fetchCreatorOrders,
  fetchCreatorSettlements,
  fetchCreatorShortlists,
  fetchInstalledSkillPackages,
  fetchReportPackages,
  fetchSkillPackageMarketplace,
  fetchSkillPackageUsage,
  fetchStrategyRecommendations,
  generateReportPackage,
  installSkillPackage,
  processApprovalTask,
  purchaseSkillPackage,
  submitComplianceCheck,
} from '../../api';
import { InfoRow, MetricCard, Section } from '../../components/common';
import type {
  AgencyClientRelation,
  ApprovalTask,
  ApprovalWorkflow,
  BrandAsset,
  BrandGuardrail,
  Campaign,
  CampaignCalendarItem,
  CampaignReportSummary,
  ComplianceCheck,
  Creator,
  CreatorCampaignBrief,
  CreatorDeliverable,
  CreatorOrder,
  CreatorSettlement,
  CreatorShortlist,
  InstalledSkillPackage,
  ReportPackage,
  SkillPackageMarketplaceItem,
  SkillPackageUsageMetric,
  StrategyRecommendation,
  WorkspaceData,
} from '../../types';
import { accountName, formatDate, formatMoney, splitKeywords } from '../../utils/formatters';

type ProductPageProps = {
  token: string;
  workspaceId: string;
  data: WorkspaceData;
  onChanged: () => void;
};

type AsyncState<T> = {
  items: T;
  loading: boolean;
  error: string | null;
};

const emptyState = { loading: false, error: null };

function errorMessage(err: unknown, fallback: string) {
  return err instanceof Error ? err.message : fallback;
}

function statusColor(value: string): 'default' | 'primary' | 'info' | 'success' | 'warning' | 'error' {
  if (['active', 'published', 'completed', 'approved', 'succeeded', 'generated', 'connected'].includes(value)) {
    return 'success';
  }
  if (['draft', 'planned', 'pending', 'queued', 'manual_pending', 'submitted'].includes(value)) {
    return 'warning';
  }
  if (['failed', 'rejected', 'canceled', 'archived'].includes(value)) {
    return value === 'failed' || value === 'rejected' ? 'error' : 'default';
  }
  return 'info';
}

function LoadingRow({ label }: { label: string }) {
  return (
    <Stack direction="row" spacing={1.25} alignItems="center" sx={{ minHeight: 72, py: 1 }}>
      <CircularProgress size={20} />
      <Typography color="text.secondary">{label}</Typography>
    </Stack>
  );
}

function EmptyText({ children }: { children: string }) {
  return (
    <Box
      sx={{
        minHeight: 72,
        display: 'flex',
        alignItems: 'center',
        border: '1px dashed',
        borderColor: 'divider',
        borderRadius: 1,
        bgcolor: 'action.hover',
        px: 2,
        py: 1.5,
        width: '100%',
        boxSizing: 'border-box',
      }}
    >
      <Typography color="text.secondary" sx={{ overflowWrap: 'anywhere' }}>
        {children}
      </Typography>
    </Box>
  );
}

function ProductTable({
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

function commaValue(value: string) {
  return splitKeywords(value);
}

function todayInputValue() {
  return new Date().toISOString().slice(0, 10);
}

function nextMonthInputValue() {
  const value = new Date();
  value.setMonth(value.getMonth() + 1);
  return value.toISOString().slice(0, 10);
}

type SohuAccountStatus = 'connected' | 'needs_auth' | 'draft';

type SohuMatrixAccount = {
  id: string;
  name: string;
  handle: string;
  group: string;
  owner: string;
  role: string;
  status: SohuAccountStatus;
  healthStatus: string;
  dataFreshness: string;
  persona: string;
  positioning: string;
  targetAudience: string;
  phoneMasked: string;
  homepageUrl: string;
  lastProfileSyncedAt: string;
  lastMetricsSyncedAt: string;
  nextAction: string;
  contentCategories: string[];
  metrics: {
    followers: number;
    contentCount: number;
    totalReads: number;
    totalLikes: number;
    comments: number;
    engagementRate: number;
  };
  recentPublish: {
    title: string;
    status: string;
    publishedAt: string;
    reads: number;
    interactions: number;
  };
  warnings: string[];
};

type SohuFlowStep = {
  id: string;
  label: string;
  status: 'done' | 'active' | 'pending' | 'blocked';
  detail: string;
};

type SohuContentMetricRow = {
  id: string;
  title: string;
  accountId: string;
  status: string;
  publishedAt: string;
  reads: number;
  likes: number;
  comments: number;
  externalUrl: string;
};

const sohuMatrixAccounts: SohuMatrixAccount[] = [
  {
    id: 'sohu_main',
    name: '品牌增长观察',
    handle: 'sohu_growth_daily',
    group: '品牌主号',
    owner: '运营一组',
    role: '主发布号',
    status: 'connected',
    healthStatus: 'healthy',
    dataFreshness: 'fresh',
    persona: '行业观察者',
    positioning: '营销增长、品牌内容方法论',
    targetAudience: '品牌市场负责人、中小企业主、内容运营',
    phoneMasked: '138****3921',
    homepageUrl: 'mp.sohu.com/profile/growth-daily',
    lastProfileSyncedAt: '2026-06-25 18:20',
    lastMetricsSyncedAt: '2026-06-25 18:28',
    nextAction: '补录昨日内容阅读数据',
    contentCategories: ['品牌增长', '投放复盘', '行业观察'],
    metrics: {
      followers: 42860,
      contentCount: 286,
      totalReads: 1842000,
      totalLikes: 23610,
      comments: 3260,
      engagementRate: 0.0146,
    },
    recentPublish: {
      title: '从种草到转化：内容团队如何复盘投放效率',
      status: 'published',
      publishedAt: '2026-06-25 10:30',
      reads: 18420,
      interactions: 426,
    },
    warnings: [],
  },
  {
    id: 'sohu_lab',
    name: '产品实验室',
    handle: 'product_lab_sohu',
    group: '垂类子号',
    owner: '产品内容组',
    role: '垂类承接',
    status: 'connected',
    healthStatus: 'watching',
    dataFreshness: 'stale',
    persona: '产品经理视角',
    positioning: '工具评测、SaaS 选型、内容自动化',
    targetAudience: '产品经理、创业团队、内容负责人',
    phoneMasked: '186****5072',
    homepageUrl: 'mp.sohu.com/profile/product-lab',
    lastProfileSyncedAt: '2026-06-22 09:15',
    lastMetricsSyncedAt: '2026-06-22 09:18',
    nextAction: '确认后台数据入口字段',
    contentCategories: ['SaaS', '效率工具', '案例拆解'],
    metrics: {
      followers: 16890,
      contentCount: 132,
      totalReads: 718000,
      totalLikes: 9240,
      comments: 980,
      engagementRate: 0.0121,
    },
    recentPublish: {
      title: '内容工作台的 5 个核心指标应该怎么设计',
      status: 'manual_pending',
      publishedAt: '2026-06-24 16:10',
      reads: 9620,
      interactions: 188,
    },
    warnings: ['指标快照超过 72 小时'],
  },
  {
    id: 'sohu_test',
    name: '搜狐号测试账号',
    handle: 'geo_test_sohu',
    group: '测试号',
    owner: '技术验证',
    role: '链路验证',
    status: 'needs_auth',
    healthStatus: 'needs_authorization',
    dataFreshness: 'missing',
    persona: '测试账号',
    positioning: '登录、发文、回流验证',
    targetAudience: '内部测试',
    phoneMasked: '159****8846',
    homepageUrl: 'mp.sohu.com/profile/test',
    lastProfileSyncedAt: '-',
    lastMetricsSyncedAt: '-',
    nextAction: '完成手机号验证码登录',
    contentCategories: ['测试发布'],
    metrics: {
      followers: 0,
      contentCount: 0,
      totalReads: 0,
      totalLikes: 0,
      comments: 0,
      engagementRate: 0,
    },
    recentPublish: {
      title: '暂无发布记录',
      status: 'draft',
      publishedAt: '-',
      reads: 0,
      interactions: 0,
    },
    warnings: ['未完成搜狐号授权'],
  },
];

const sohuFlowSteps: SohuFlowStep[] = [
  { id: 'account', label: '账号建档', status: 'done', detail: '记录手机号、主页、分组、负责人和定位' },
  { id: 'auth', label: '登录授权', status: 'active', detail: '复用搜狐号 phone/SMS 浏览器登录流程' },
  { id: 'snapshot', label: '数据快照', status: 'pending', detail: '录入后台可见粉丝、内容、阅读和互动数据' },
  { id: 'publish', label: '发布任务', status: 'pending', detail: '选择内容和搜狐号账号执行浏览器发布' },
  { id: 'metrics', label: '结果回流', status: 'pending', detail: '按内容 URL 补录阅读、点赞、评论等指标' },
];

const sohuContentMetrics: SohuContentMetricRow[] = [
  {
    id: 'content_sohu_1',
    title: '从种草到转化：内容团队如何复盘投放效率',
    accountId: 'sohu_main',
    status: 'published',
    publishedAt: '2026-06-25 10:30',
    reads: 18420,
    likes: 312,
    comments: 74,
    externalUrl: 'https://mp.sohu.com/a/884201928_121001',
  },
  {
    id: 'content_sohu_2',
    title: '内容工作台的 5 个核心指标应该怎么设计',
    accountId: 'sohu_lab',
    status: 'manual_pending',
    publishedAt: '2026-06-24 16:10',
    reads: 9620,
    likes: 141,
    comments: 47,
    externalUrl: 'https://mp.sohu.com/a/884111320_121001',
  },
  {
    id: 'content_sohu_3',
    title: '搜狐号链路测试：图文发布与结果确认',
    accountId: 'sohu_test',
    status: 'draft',
    publishedAt: '-',
    reads: 0,
    likes: 0,
    comments: 0,
    externalUrl: '',
  },
];

const sohuVisibleFieldGroups = [
  {
    title: '账号主页',
    fields: ['昵称', '头像', '搜狐号/主页地址', '简介', '认证状态', '粉丝数'],
  },
  {
    title: '后台首页',
    fields: ['账号状态', '系统通知', '昨日阅读', '新增粉丝', '待处理事项'],
  },
  {
    title: '内容管理',
    fields: ['文章标题', '发布时间', '审核状态', '外部链接', '阅读', '点赞', '评论'],
  },
  {
    title: '数据分析',
    fields: ['总阅读', '推荐量', '互动量', '粉丝趋势', '单篇内容趋势'],
  },
];

const sohuRoadmapItems = [
  '第一步：人工录入账号快照和单篇内容指标',
  '第二步：浏览器辅助读取当前页面可见字段',
  '第三步：同步任务队列化，沉淀采集日志和失败原因',
  '第四步：按账号定位、内容类型和发布时间做复盘报表',
];

type MatrixPlatformKey = 'overview' | 'sohu' | 'xiaohongshu' | 'toutiao' | 'netease';
type MatrixAssetPlatformKey = Exclude<MatrixPlatformKey, 'overview'>;

type MatrixPlatformConfig = {
  key: MatrixAssetPlatformKey;
  label: string;
  iconLabel: string;
  description: string;
  authMode: string;
  dataMode: string;
  primaryAction: string;
  flowSteps: SohuFlowStep[];
  visibleFieldGroups: Array<{ title: string; fields: string[] }>;
  roadmapItems: string[];
};

type MediaMatrixAccount = SohuMatrixAccount & {
  platformKey: MatrixAssetPlatformKey;
  platformName: string;
  platformIcon: string;
  loginMode: string;
  dataSource: string;
  capabilitySummary: string;
};

type MediaMatrixContentMetricRow = SohuContentMetricRow & {
  platformKey: MatrixAssetPlatformKey;
  platformName: string;
};

type MediaMatrixTodoItem = {
  id: string;
  platformKey: MatrixAssetPlatformKey;
  accountId: string;
  title: string;
  priority: 'high' | 'medium' | 'low';
  status: string;
  dueAt: string;
};

const defaultPlatformFlowSteps: SohuFlowStep[] = [
  { id: 'account', label: '账号建档', status: 'done', detail: '记录主页、负责人、分组、定位和账号能力' },
  { id: 'auth', label: '授权方式', status: 'pending', detail: '确认平台支持扫码、手机号、手动或 API 授权' },
  { id: 'snapshot', label: '数据快照', status: 'pending', detail: '先用人工录入沉淀账号级可视数据' },
  { id: 'publish', label: '发布任务', status: 'pending', detail: '接入平台发布能力或保留人工确认入口' },
  { id: 'metrics', label: '结果回流', status: 'pending', detail: '按内容链接补录阅读、互动和外部状态' },
];

const matrixPlatformConfigs: MatrixPlatformConfig[] = [
  {
    key: 'sohu',
    label: '搜狐号',
    iconLabel: '搜',
    description: '优先打通手机号登录、文章发布和可视数据快照。',
    authMode: '手机号验证码',
    dataMode: '人工录入 / 浏览器辅助',
    primaryAction: '发起搜狐号登录',
    flowSteps: sohuFlowSteps,
    visibleFieldGroups: sohuVisibleFieldGroups,
    roadmapItems: sohuRoadmapItems,
  },
  {
    key: 'xiaohongshu',
    label: '小红书',
    iconLabel: '小',
    description: '先维护账号资产和公开主页快照，后续再评估授权能力。',
    authMode: '扫码登录 / 手动维护',
    dataMode: '公开可见快照',
    primaryAction: '录入主页快照',
    flowSteps: defaultPlatformFlowSteps,
    visibleFieldGroups: [
      { title: '公开主页', fields: ['昵称', '小红书号', '头像', '简介', '粉丝', '获赞与收藏', '笔记数'] },
      { title: '发布记录', fields: ['笔记标题', '发布时间', '链接', '互动摘要', '人工备注'] },
      { title: '运营风险', fields: ['账号异常', '内容审核', '数据过期', '登录状态'] },
    ],
    roadmapItems: ['维护主页快照', '接入二维码登录状态', '记录笔记发布结果', '评估官方授权或浏览器辅助采集'],
  },
  {
    key: 'toutiao',
    label: '头条号',
    iconLabel: '头',
    description: '管理图文发布、内容链接和内容级阅读互动数据。',
    authMode: '扫码登录',
    dataMode: '人工录入 / 浏览器辅助',
    primaryAction: '发起头条号登录',
    flowSteps: defaultPlatformFlowSteps,
    visibleFieldGroups: [
      { title: '账号主页', fields: ['头条号名称', '头像', '认证信息', '粉丝', '介绍'] },
      { title: '内容管理', fields: ['标题', '状态', '发布时间', '阅读', '评论', '收益提示'] },
      { title: '数据概览', fields: ['推荐量', '阅读量', '粉丝变化', '互动量'] },
    ],
    roadmapItems: ['复用浏览器登录', '记录文章发布状态', '补录内容数据', '沉淀收益/推荐指标字段'],
  },
  {
    key: 'netease',
    label: '网易号',
    iconLabel: '网',
    description: '维护账号档案、发布状态和内容数据回填。',
    authMode: '扫码登录',
    dataMode: '人工录入 / 浏览器辅助',
    primaryAction: '发起网易号登录',
    flowSteps: defaultPlatformFlowSteps,
    visibleFieldGroups: [
      { title: '账号主页', fields: ['网易号名称', '账号 ID', '头像', '简介', '粉丝'] },
      { title: '内容列表', fields: ['标题', '审核状态', '发布时间', '阅读', '评论'] },
      { title: '运营状态', fields: ['登录状态', '数据更新时间', '异常提示'] },
    ],
    roadmapItems: ['复用二维码登录', '接入文章发布', '记录外部 URL', '扩展内容指标回流'],
  },
];

const additionalMatrixAccounts: MediaMatrixAccount[] = [
  {
    id: 'xhs_brand',
    platformKey: 'xiaohongshu',
    platformName: '小红书',
    platformIcon: '小',
    loginMode: '扫码登录',
    dataSource: '公开主页快照',
    capabilitySummary: '主页档案、发布排期、人工数据快照',
    name: '品牌灵感研究所',
    handle: 'RED_growth_lab',
    group: '品牌主号',
    owner: '内容增长组',
    role: '种草主阵地',
    status: 'connected',
    healthStatus: 'watching',
    dataFreshness: 'stale',
    persona: '品牌研究员',
    positioning: '品牌案例、生活方式场景、产品使用灵感',
    targetAudience: '消费品牌主理人、年轻职场女性',
    phoneMasked: '扫码登录',
    homepageUrl: 'xiaohongshu.com/user/profile/red-growth-lab',
    lastProfileSyncedAt: '2026-06-23 20:12',
    lastMetricsSyncedAt: '2026-06-23 20:18',
    nextAction: '补录最新主页展示数据',
    contentCategories: ['生活方式', '品牌案例', '产品场景'],
    metrics: {
      followers: 62800,
      contentCount: 94,
      totalReads: 1260000,
      totalLikes: 58200,
      comments: 4210,
      engagementRate: 0.0495,
    },
    recentPublish: {
      title: '新品上市前，品牌内容如何提前种草',
      status: 'published',
      publishedAt: '2026-06-24 19:40',
      reads: 41200,
      interactions: 2360,
    },
    warnings: ['主页数据快照超过 48 小时'],
  },
  {
    id: 'xhs_store',
    platformKey: 'xiaohongshu',
    platformName: '小红书',
    platformIcon: '小',
    loginMode: '手动维护',
    dataSource: '人工录入',
    capabilitySummary: '账号档案、内容排期、发布结果确认',
    name: '门店探访日记',
    handle: 'store_visit_notes',
    group: '区域子号',
    owner: '区域运营',
    role: '本地内容承接',
    status: 'draft',
    healthStatus: 'watching',
    dataFreshness: 'missing',
    persona: '探店记录员',
    positioning: '门店体验、活动预告、用户故事',
    targetAudience: '本地消费者、会员用户',
    phoneMasked: '手动维护',
    homepageUrl: 'xiaohongshu.com/user/profile/store-visit',
    lastProfileSyncedAt: '-',
    lastMetricsSyncedAt: '-',
    nextAction: '确认账号主页和负责人',
    contentCategories: ['门店活动', '用户故事'],
    metrics: {
      followers: 8200,
      contentCount: 36,
      totalReads: 210000,
      totalLikes: 8600,
      comments: 510,
      engagementRate: 0.0434,
    },
    recentPublish: {
      title: '周末门店体验活动复盘',
      status: 'draft',
      publishedAt: '-',
      reads: 0,
      interactions: 0,
    },
    warnings: ['账号档案未完善'],
  },
  {
    id: 'toutiao_main',
    platformKey: 'toutiao',
    platformName: '头条号',
    platformIcon: '头',
    loginMode: '扫码登录',
    dataSource: '人工录入 / 浏览器辅助',
    capabilitySummary: '文章发布、外链回填、阅读互动录入',
    name: '增长方法周刊',
    handle: 'growth_weekly',
    group: '品牌主号',
    owner: '运营一组',
    role: '资讯长文分发',
    status: 'connected',
    healthStatus: 'healthy',
    dataFreshness: 'fresh',
    persona: '行业编辑',
    positioning: '增长策略、行业观察、工具方法',
    targetAudience: '市场负责人、企业经营者',
    phoneMasked: '扫码登录',
    homepageUrl: 'mp.toutiao.com/profile/growth-weekly',
    lastProfileSyncedAt: '2026-06-25 16:00',
    lastMetricsSyncedAt: '2026-06-25 16:05',
    nextAction: '检查待发布文章审核状态',
    contentCategories: ['行业资讯', '增长策略', '工具方法'],
    metrics: {
      followers: 73500,
      contentCount: 412,
      totalReads: 2860000,
      totalLikes: 31800,
      comments: 5840,
      engagementRate: 0.0131,
    },
    recentPublish: {
      title: 'AI 内容团队如何搭建选题流水线',
      status: 'published',
      publishedAt: '2026-06-25 09:00',
      reads: 36600,
      interactions: 740,
    },
    warnings: [],
  },
  {
    id: 'toutiao_test',
    platformKey: 'toutiao',
    platformName: '头条号',
    platformIcon: '头',
    loginMode: '扫码登录',
    dataSource: '人工录入',
    capabilitySummary: '发布链路验证',
    name: '头条链路测试号',
    handle: 'tt_geo_test',
    group: '测试号',
    owner: '技术验证',
    role: '链路验证',
    status: 'needs_auth',
    healthStatus: 'needs_authorization',
    dataFreshness: 'missing',
    persona: '测试账号',
    positioning: '登录、发布、确认链接',
    targetAudience: '内部测试',
    phoneMasked: '扫码登录',
    homepageUrl: 'mp.toutiao.com/profile/test',
    lastProfileSyncedAt: '-',
    lastMetricsSyncedAt: '-',
    nextAction: '重新扫码授权',
    contentCategories: ['测试发布'],
    metrics: {
      followers: 0,
      contentCount: 0,
      totalReads: 0,
      totalLikes: 0,
      comments: 0,
      engagementRate: 0,
    },
    recentPublish: {
      title: '暂无发布记录',
      status: 'draft',
      publishedAt: '-',
      reads: 0,
      interactions: 0,
    },
    warnings: ['未完成授权'],
  },
  {
    id: 'netease_main',
    platformKey: 'netease',
    platformName: '网易号',
    platformIcon: '网',
    loginMode: '扫码登录',
    dataSource: '人工录入 / 浏览器辅助',
    capabilitySummary: '账号档案、文章发布、结果确认',
    name: '企业内容观察',
    handle: 'netease_content_ops',
    group: '品牌主号',
    owner: '品牌内容组',
    role: '资讯分发',
    status: 'connected',
    healthStatus: 'healthy',
    dataFreshness: 'fresh',
    persona: '企业内容编辑',
    positioning: '企业内容运营、案例观察、行业观点',
    targetAudience: '企业市场部、内容团队',
    phoneMasked: '扫码登录',
    homepageUrl: 'mp.163.com/profile/content-ops',
    lastProfileSyncedAt: '2026-06-25 14:35',
    lastMetricsSyncedAt: '2026-06-25 14:40',
    nextAction: '录入今天发布文章外链',
    contentCategories: ['行业观点', '企业内容', '案例复盘'],
    metrics: {
      followers: 28400,
      contentCount: 178,
      totalReads: 932000,
      totalLikes: 11800,
      comments: 1420,
      engagementRate: 0.0142,
    },
    recentPublish: {
      title: '企业内容团队怎样做跨平台分发',
      status: 'manual_pending',
      publishedAt: '2026-06-25 11:20',
      reads: 12800,
      interactions: 190,
    },
    warnings: ['发布结果 URL 待人工确认'],
  },
];

const mediaMatrixAccounts: MediaMatrixAccount[] = [
  ...sohuMatrixAccounts.map((account) => ({
    ...account,
    platformKey: 'sohu' as const,
    platformName: '搜狐号',
    platformIcon: '搜',
    loginMode: '手机号验证码',
    dataSource: '人工录入 / 浏览器辅助',
    capabilitySummary: '手机号登录、文章发布、结果回填',
  })),
  ...additionalMatrixAccounts,
];

const mediaMatrixContentMetrics: MediaMatrixContentMetricRow[] = [
  ...sohuContentMetrics.map((item) => ({
    ...item,
    platformKey: 'sohu' as const,
    platformName: '搜狐号',
  })),
  {
    id: 'content_xhs_1',
    platformKey: 'xiaohongshu',
    platformName: '小红书',
    title: '新品上市前，品牌内容如何提前种草',
    accountId: 'xhs_brand',
    status: 'published',
    publishedAt: '2026-06-24 19:40',
    reads: 41200,
    likes: 1980,
    comments: 380,
    externalUrl: 'https://www.xiaohongshu.com/explore/mock-note',
  },
  {
    id: 'content_toutiao_1',
    platformKey: 'toutiao',
    platformName: '头条号',
    title: 'AI 内容团队如何搭建选题流水线',
    accountId: 'toutiao_main',
    status: 'published',
    publishedAt: '2026-06-25 09:00',
    reads: 36600,
    likes: 520,
    comments: 220,
    externalUrl: 'https://www.toutiao.com/article/mock',
  },
  {
    id: 'content_netease_1',
    platformKey: 'netease',
    platformName: '网易号',
    title: '企业内容团队怎样做跨平台分发',
    accountId: 'netease_main',
    status: 'manual_pending',
    publishedAt: '2026-06-25 11:20',
    reads: 12800,
    likes: 142,
    comments: 48,
    externalUrl: '',
  },
];

const mediaMatrixTodos: MediaMatrixTodoItem[] = [
  { id: 'todo_sohu_auth', platformKey: 'sohu', accountId: 'sohu_test', title: '搜狐号测试账号需要完成手机号验证码登录', priority: 'high', status: 'needs_auth', dueAt: '今天' },
  { id: 'todo_xhs_snapshot', platformKey: 'xiaohongshu', accountId: 'xhs_brand', title: '小红书品牌主号主页数据超过 48 小时未更新', priority: 'medium', status: 'stale', dueAt: '今天' },
  { id: 'todo_netease_url', platformKey: 'netease', accountId: 'netease_main', title: '网易号发布结果 URL 待人工确认', priority: 'medium', status: 'manual_pending', dueAt: '今天' },
  { id: 'todo_tt_auth', platformKey: 'toutiao', accountId: 'toutiao_test', title: '头条链路测试号授权失效，需要重新扫码', priority: 'high', status: 'needs_auth', dueAt: '明天' },
];

function compactNumber(value: number) {
  return new Intl.NumberFormat('zh-CN', {
    notation: value >= 10000 ? 'compact' : 'standard',
    maximumFractionDigits: 1,
  }).format(value);
}

function percentValue(value: number) {
  return `${(value * 100).toFixed(2)}%`;
}

function sohuStatusLabel(value: string) {
  const labels: Record<string, string> = {
    connected: '已连接',
    needs_auth: '待授权',
    draft: '草稿',
    healthy: '健康',
    watching: '观察中',
    needs_authorization: '需授权',
    fresh: '最新',
    stale: '待刷新',
    missing: '无快照',
    published: '已发布',
    manual_pending: '待确认',
    done: '完成',
    active: '进行中',
    pending: '待接入',
    blocked: '受阻',
    high: '高',
    medium: '中',
    low: '低',
  };
  return labels[value] ?? value;
}

function sohuStatusColor(value: string): 'default' | 'primary' | 'info' | 'success' | 'warning' | 'error' {
  if (['connected', 'healthy', 'fresh', 'published', 'done'].includes(value)) {
    return 'success';
  }
  if (['needs_auth', 'needs_authorization', 'stale', 'manual_pending', 'active', 'watching'].includes(value)) {
    return 'warning';
  }
  if (['missing', 'blocked'].includes(value)) {
    return 'error';
  }
  if (['pending', 'draft'].includes(value)) {
    return 'info';
  }
  return 'default';
}

function platformConfigByKey(platformKey: MatrixAssetPlatformKey) {
  return matrixPlatformConfigs.find((platform) => platform.key === platformKey) ?? matrixPlatformConfigs[0];
}

function matrixAccountsByPlatform(platformKey: MatrixAssetPlatformKey) {
  return mediaMatrixAccounts.filter((account) => account.platformKey === platformKey);
}

function matrixContentByPlatform(platformKey: MatrixAssetPlatformKey) {
  return mediaMatrixContentMetrics.filter((item) => item.platformKey === platformKey);
}

function matrixAccountName(accountId: string) {
  return mediaMatrixAccounts.find((account) => account.id === accountId)?.name ?? accountId;
}

function platformSummary(platform: MatrixPlatformConfig) {
  const accounts = matrixAccountsByPlatform(platform.key);
  const content = matrixContentByPlatform(platform.key);
  const todos = mediaMatrixTodos.filter((item) => item.platformKey === platform.key);
  return {
    accountCount: accounts.length,
    connectedCount: accounts.filter((account) => account.status === 'connected').length,
    authIssueCount: accounts.filter((account) => account.status === 'needs_auth').length,
    staleCount: accounts.filter((account) => account.dataFreshness !== 'fresh').length,
    followerCount: accounts.reduce((sum, account) => sum + account.metrics.followers, 0),
    publishCount: content.filter((item) => item.status === 'published').length,
    issueCount: todos.length,
  };
}

function MatrixWorkflowStep({ step, index }: { step: SohuFlowStep; index: number }) {
  return (
    <Box
      sx={{
        display: 'grid',
        gridTemplateColumns: '32px minmax(0, 1fr)',
        gap: 1.25,
        minHeight: 94,
        border: '1px solid',
        borderColor: step.status === 'active' ? 'warning.light' : 'divider',
        borderRadius: 1,
        p: 1.5,
        bgcolor: step.status === 'active' ? 'warning.50' : 'background.paper',
      }}
    >
      <Box
        sx={{
          width: 32,
          height: 32,
          borderRadius: '50%',
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'center',
          bgcolor: `${sohuStatusColor(step.status)}.main`,
          color: 'common.white',
          fontWeight: 700,
        }}
      >
        {index + 1}
      </Box>
      <Stack spacing={0.75} sx={{ minWidth: 0 }}>
        <Stack direction="row" spacing={1} alignItems="center" flexWrap="wrap" useFlexGap>
          <Typography fontWeight={700} sx={wrappingTextSx}>
            {step.label}
          </Typography>
          <Chip size="small" label={sohuStatusLabel(step.status)} color={sohuStatusColor(step.status)} />
        </Stack>
        <Typography variant="body2" color="text.secondary" sx={secondaryTextSx}>
          {step.detail}
        </Typography>
      </Stack>
    </Box>
  );
}

function PlatformStatusCard({ platform, active, onSelect }: { platform: MatrixPlatformConfig; active: boolean; onSelect: () => void }) {
  const summary = platformSummary(platform);
  return (
    <Box
      component="button"
      type="button"
      onClick={onSelect}
      sx={{
        width: '100%',
        minHeight: 192,
        textAlign: 'left',
        border: '1px solid',
        borderColor: active ? 'primary.main' : 'divider',
        borderRadius: 1,
        bgcolor: active ? 'action.selected' : 'background.paper',
        p: 2,
        cursor: 'pointer',
        font: 'inherit',
        color: 'inherit',
        transition: 'border-color 120ms ease, background-color 120ms ease',
        '&:hover': {
          borderColor: 'primary.main',
          bgcolor: 'action.hover',
        },
      }}
    >
      <Stack spacing={1.5} sx={{ minWidth: 0 }}>
        <Stack direction="row" spacing={1.25} alignItems="center">
          <Box
            sx={{
              width: 42,
              height: 42,
              borderRadius: 1,
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'center',
              bgcolor: 'primary.main',
              color: 'primary.contrastText',
              fontWeight: 800,
              flexShrink: 0,
            }}
          >
            {platform.iconLabel}
          </Box>
          <Box sx={{ minWidth: 0 }}>
            <Typography fontWeight={800} sx={wrappingTextSx}>
              {platform.label}
            </Typography>
            <Typography variant="body2" color="text.secondary" sx={secondaryTextSx}>
              {platform.description}
            </Typography>
          </Box>
        </Stack>
        <Grid container spacing={1}>
          <Grid size={6}>
            <InfoRow label="账号" value={String(summary.accountCount)} />
          </Grid>
          <Grid size={6}>
            <InfoRow label="已连接" value={String(summary.connectedCount)} />
          </Grid>
          <Grid size={6}>
            <InfoRow label="待授权" value={String(summary.authIssueCount)} />
          </Grid>
          <Grid size={6}>
            <InfoRow label="待处理" value={String(summary.issueCount)} />
          </Grid>
        </Grid>
        <Stack direction="row" spacing={0.75} flexWrap="wrap" useFlexGap>
          <Chip size="small" label={platform.authMode} variant="outlined" />
          <Chip size="small" label={platform.dataMode} variant="outlined" />
        </Stack>
      </Stack>
    </Box>
  );
}

function PlatformOverviewTab({ activePlatform, onSelectPlatform }: { activePlatform: MatrixPlatformKey; onSelectPlatform: (value: MatrixPlatformKey) => void }) {
  const totalAccounts = mediaMatrixAccounts.length;
  const connectedCount = mediaMatrixAccounts.filter((account) => account.status === 'connected').length;
  const totalFollowers = mediaMatrixAccounts.reduce((sum, account) => sum + account.metrics.followers, 0);
  const publishedCount = mediaMatrixContentMetrics.filter((item) => item.status === 'published').length;
  const staleCount = mediaMatrixAccounts.filter((account) => account.dataFreshness !== 'fresh').length;
  const manualPendingCount = mediaMatrixContentMetrics.filter((item) => item.status === 'manual_pending').length;

  return (
    <Stack spacing={3}>
      <Grid container spacing={2}>
        <MetricCard label="矩阵账号" value={totalAccounts} helper="所有平台纳管账号" />
        <MetricCard label="已连接" value={connectedCount} helper="可进入发布或后台查看" />
        <MetricCard label="总粉丝" value={totalFollowers} helper={`最近快照汇总，约 ${compactNumber(totalFollowers)}`} />
        <MetricCard label="近 7 天发布" value={publishedCount} helper="mock 发布回流记录" />
        <MetricCard label="待刷新" value={staleCount} helper="快照缺失或过期" tone={staleCount > 0 ? 'error' : 'primary'} />
        <MetricCard label="待确认" value={manualPendingCount} helper="发布 URL 或指标待补录" tone={manualPendingCount > 0 ? 'error' : 'primary'} />
      </Grid>

      <Section title="平台状态">
        <Grid container spacing={1.5}>
          {matrixPlatformConfigs.map((platform) => (
            <Grid key={platform.key} size={{ xs: 12, md: 6, xl: 3 }}>
              <PlatformStatusCard platform={platform} active={activePlatform === platform.key} onSelect={() => onSelectPlatform(platform.key)} />
            </Grid>
          ))}
        </Grid>
      </Section>

      <Grid container spacing={2}>
        <Grid size={{ xs: 12, lg: 5 }}>
          <Section title="待处理事项">
            <Stack spacing={1.25}>
              {mediaMatrixTodos.map((todo) => {
                const platform = platformConfigByKey(todo.platformKey);
                return (
                  <Box key={todo.id} sx={{ border: '1px solid', borderColor: 'divider', borderRadius: 1, p: 1.5 }}>
                    <Stack spacing={1}>
                      <Stack direction="row" spacing={0.75} alignItems="center" flexWrap="wrap" useFlexGap>
                        <Chip size="small" label={platform.label} />
                        <Chip size="small" label={`优先级 ${sohuStatusLabel(todo.priority)}`} color={todo.priority === 'high' ? 'error' : 'warning'} variant="outlined" />
                        <Chip size="small" label={sohuStatusLabel(todo.status)} color={sohuStatusColor(todo.status)} />
                      </Stack>
                      <Typography fontWeight={700} sx={wrappingTextSx}>
                        {todo.title}
                      </Typography>
                      <Typography variant="body2" color="text.secondary">
                        {matrixAccountName(todo.accountId)} / {todo.dueAt}
                      </Typography>
                    </Stack>
                  </Box>
                );
              })}
            </Stack>
          </Section>
        </Grid>

        <Grid size={{ xs: 12, lg: 7 }}>
          <Section title="最近发布回流">
            <ProductTable minWidth={820}>
              <TableHead>
                <TableRow>
                  <TableCell>内容</TableCell>
                  <TableCell>平台</TableCell>
                  <TableCell>账号</TableCell>
                  <TableCell>状态</TableCell>
                  <TableCell>阅读</TableCell>
                  <TableCell>互动</TableCell>
                </TableRow>
              </TableHead>
              <TableBody>
                {mediaMatrixContentMetrics.map((item) => (
                  <TableRow key={item.id} hover>
                    <TableCell sx={{ minWidth: 220 }}>
                      <Typography fontWeight={700} sx={secondaryTextSx}>
                        {item.title}
                      </Typography>
                      <Typography variant="body2" color="text.secondary">
                        {item.publishedAt}
                      </Typography>
                    </TableCell>
                    <TableCell sx={{ minWidth: 110 }}>{item.platformName}</TableCell>
                    <TableCell sx={{ minWidth: 140 }}>{matrixAccountName(item.accountId)}</TableCell>
                    <TableCell>
                      <Chip size="small" label={sohuStatusLabel(item.status)} color={sohuStatusColor(item.status)} />
                    </TableCell>
                    <TableCell>{compactNumber(item.reads)}</TableCell>
                    <TableCell>{(item.likes + item.comments).toLocaleString()}</TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </ProductTable>
          </Section>
        </Grid>
      </Grid>
    </Stack>
  );
}

export function MediaMatrixView({ data }: ProductPageProps) {
  const [activeTab, setActiveTab] = useState<MatrixPlatformKey>('overview');
  const [selectedAccountId, setSelectedAccountId] = useState(mediaMatrixAccounts[0]?.id ?? '');
  const [groupFilter, setGroupFilter] = useState('all');
  const [statusFilter, setStatusFilter] = useState('all');
  const [snapshotForm, setSnapshotForm] = useState({
    followerCount: '42860',
    contentCount: '286',
    totalReads: '1842000',
    note: '后台首页 + 内容管理页人工录入',
  });
  const [savedAt, setSavedAt] = useState('');

  const activePlatformKey = activeTab === 'overview' ? 'sohu' : activeTab;
  const activePlatform = platformConfigByKey(activePlatformKey);
  const platformAccounts = matrixAccountsByPlatform(activePlatform.key);
  const platformContentMetrics = matrixContentByPlatform(activePlatform.key);
  const existingPlatformAccountCount = data.mediaAccounts.filter((account) => {
    const platform = data.mediaPlatforms.find((item) => item.id === account.platformId);
    return platform?.type === activePlatform.key || account.platformId === `plt_${activePlatform.key}`;
  }).length;
  const filteredAccounts = platformAccounts.filter((account) => {
    const groupOK = groupFilter === 'all' || account.group === groupFilter;
    const statusOK = statusFilter === 'all' || account.status === statusFilter;
    return groupOK && statusOK;
  });
  const selectedAccount =
    platformAccounts.find((account) => account.id === selectedAccountId) ??
    platformAccounts[0] ??
    mediaMatrixAccounts.find((account) => account.id === selectedAccountId) ??
    mediaMatrixAccounts[0];
  const connectedCount = platformAccounts.filter((account) => account.status === 'connected').length;
  const staleCount = platformAccounts.filter((account) => account.dataFreshness !== 'fresh').length;
  const totalFollowers = platformAccounts.reduce((sum, account) => sum + account.metrics.followers, 0);
  const totalReads = platformAccounts.reduce((sum, account) => sum + account.metrics.totalReads, 0);
  const publishedCount = platformContentMetrics.filter((item) => item.status === 'published').length;

  const submitMockSnapshot = () => {
    setSavedAt(new Date().toLocaleString('zh-CN', { hour12: false }));
  };

  const selectTab = (value: MatrixPlatformKey) => {
    setActiveTab(value);
    if (value !== 'overview') {
      const first = matrixAccountsByPlatform(value)[0];
      if (first) {
        setSelectedAccountId(first.id);
      }
    }
  };

  const renderPlatformTab = () => (
    <Stack spacing={3}>
      <Alert severity="info">
        当前为{activePlatform.label}资产视图 mock 数据。工作区已有{activePlatform.label}账号：{existingPlatformAccountCount} 个。
      </Alert>

      <Grid container spacing={2}>
        <MetricCard label={`${activePlatform.label}账号`} value={platformAccounts.length} helper="纳管账号资产" />
        <MetricCard label="已连接" value={connectedCount} helper="可进入发布和后台查看" />
        <MetricCard label="总粉丝" value={totalFollowers} helper={`最近账号快照汇总，约 ${compactNumber(totalFollowers)}`} />
        <MetricCard label="总阅读" value={totalReads} helper={`mock 可视数据累计，约 ${compactNumber(totalReads)}`} />
        <MetricCard label="已发布" value={publishedCount} helper="已回填外部 URL" />
        <MetricCard label="待刷新" value={staleCount} helper="快照缺失或过期" tone={staleCount > 0 ? 'error' : 'primary'} />
      </Grid>

      <Section title={`${activePlatform.label}功能链路`}>
        <Grid container spacing={1.5}>
          {activePlatform.flowSteps.map((step, index) => (
            <Grid key={step.id} size={{ xs: 12, md: 6, xl: 2.4 }}>
              <MatrixWorkflowStep step={step} index={index} />
            </Grid>
          ))}
        </Grid>
      </Section>

      <Grid container spacing={2}>
        <Grid size={{ xs: 12, xl: 8 }}>
          <Section
            title={`${activePlatform.label}账号资产`}
            action={
              <Stack direction={{ xs: 'column', sm: 'row' }} spacing={1}>
                <FormControl size="small" sx={{ minWidth: 132 }}>
                  <InputLabel>分组</InputLabel>
                  <Select label="分组" value={groupFilter} onChange={(event) => setGroupFilter(String(event.target.value))}>
                    <MenuItem value="all">全部</MenuItem>
                    <MenuItem value="品牌主号">品牌主号</MenuItem>
                    <MenuItem value="垂类子号">垂类子号</MenuItem>
                    <MenuItem value="区域子号">区域子号</MenuItem>
                    <MenuItem value="测试号">测试号</MenuItem>
                  </Select>
                </FormControl>
                <FormControl size="small" sx={{ minWidth: 132 }}>
                  <InputLabel>状态</InputLabel>
                  <Select label="状态" value={statusFilter} onChange={(event) => setStatusFilter(String(event.target.value))}>
                    <MenuItem value="all">全部</MenuItem>
                    <MenuItem value="connected">已连接</MenuItem>
                    <MenuItem value="needs_auth">待授权</MenuItem>
                    <MenuItem value="draft">草稿</MenuItem>
                  </Select>
                </FormControl>
              </Stack>
            }
          >
            <ProductTable minWidth={1080}>
              <TableHead>
                <TableRow>
                  <TableCell>账号</TableCell>
                  <TableCell>角色 / 负责人</TableCell>
                  <TableCell>定位</TableCell>
                  <TableCell>粉丝 / 内容</TableCell>
                  <TableCell>阅读 / 互动</TableCell>
                  <TableCell>状态</TableCell>
                  <TableCell>最近发布</TableCell>
                  <TableCell align="right">操作</TableCell>
                </TableRow>
              </TableHead>
              <TableBody>
                {filteredAccounts.map((account) => (
                  <TableRow
                    key={account.id}
                    hover
                    selected={account.id === selectedAccountId}
                    onClick={() => setSelectedAccountId(account.id)}
                    sx={{ cursor: 'pointer' }}
                  >
                    <TableCell sx={{ minWidth: 190 }}>
                      <Stack direction="row" spacing={1} alignItems="flex-start">
                        <Box
                          sx={{
                            width: 32,
                            height: 32,
                            borderRadius: 1,
                            display: 'flex',
                            alignItems: 'center',
                            justifyContent: 'center',
                            bgcolor: 'primary.main',
                            color: 'primary.contrastText',
                            fontWeight: 800,
                            flexShrink: 0,
                          }}
                        >
                          {account.platformIcon}
                        </Box>
                        <Box sx={{ minWidth: 0 }}>
                          <Typography fontWeight={800} sx={wrappingTextSx}>
                            {account.name}
                          </Typography>
                          <Typography variant="body2" color="text.secondary" sx={wrappingTextSx}>
                            {account.handle}
                          </Typography>
                          <Typography variant="body2" color="text.secondary" sx={wrappingTextSx}>
                            {account.homepageUrl}
                          </Typography>
                        </Box>
                      </Stack>
                    </TableCell>
                    <TableCell sx={{ minWidth: 150 }}>
                      <Typography variant="body2" fontWeight={700} sx={wrappingTextSx}>
                        {account.group}
                      </Typography>
                      <Typography variant="body2" color="text.secondary" sx={wrappingTextSx}>
                        {account.role} / {account.owner}
                      </Typography>
                    </TableCell>
                    <TableCell sx={{ minWidth: 220, maxWidth: 320 }}>
                      <Typography variant="body2" sx={secondaryTextSx}>
                        {account.positioning}
                      </Typography>
                      <Typography variant="body2" color="text.secondary" sx={secondaryTextSx}>
                        {account.targetAudience}
                      </Typography>
                    </TableCell>
                    <TableCell sx={{ minWidth: 120 }}>
                      <Typography fontWeight={800}>{compactNumber(account.metrics.followers)}</Typography>
                      <Typography variant="body2" color="text.secondary">
                        {account.metrics.contentCount} 篇内容
                      </Typography>
                    </TableCell>
                    <TableCell sx={{ minWidth: 130 }}>
                      <Typography fontWeight={800}>{compactNumber(account.metrics.totalReads)}</Typography>
                      <Typography variant="body2" color="text.secondary">
                        {percentValue(account.metrics.engagementRate)}
                      </Typography>
                    </TableCell>
                    <TableCell sx={{ minWidth: 128 }}>
                      <Stack spacing={0.75} alignItems="flex-start">
                        <Chip size="small" label={sohuStatusLabel(account.status)} color={sohuStatusColor(account.status)} />
                        <Chip size="small" label={sohuStatusLabel(account.dataFreshness)} color={sohuStatusColor(account.dataFreshness)} variant="outlined" />
                      </Stack>
                    </TableCell>
                    <TableCell sx={{ minWidth: 220, maxWidth: 280 }}>
                      <Typography variant="body2" fontWeight={700} sx={secondaryTextSx}>
                        {account.recentPublish.title}
                      </Typography>
                      <Typography variant="body2" color="text.secondary" sx={wrappingTextSx}>
                        {account.recentPublish.publishedAt}
                      </Typography>
                    </TableCell>
                    <TableCell align="right" sx={{ minWidth: 210 }}>
                      <Stack direction="row" spacing={0.75} justifyContent="flex-end" flexWrap="wrap" useFlexGap>
                        <Button size="small" startIcon={<VisibilityOutlinedIcon />} onClick={() => setSelectedAccountId(account.id)}>
                          详情
                        </Button>
                        <Button size="small" startIcon={<PublishOutlinedIcon />}>
                          发文
                        </Button>
                      </Stack>
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </ProductTable>
          </Section>
        </Grid>

        <Grid size={{ xs: 12, xl: 4 }}>
          <Section title="账号详情">
            {selectedAccount ? (
              <Stack spacing={2}>
                <Stack direction="row" spacing={1} flexWrap="wrap" useFlexGap>
                  <Chip label={selectedAccount.platformName} color="primary" />
                  <Chip label={selectedAccount.group} color="primary" variant="outlined" />
                  <Chip label={sohuStatusLabel(selectedAccount.healthStatus)} color={sohuStatusColor(selectedAccount.healthStatus)} />
                  <Chip label={selectedAccount.loginMode} variant="outlined" />
                </Stack>
                <Stack spacing={1}>
                  <InfoRow label="账号" value={`${selectedAccount.name} / ${selectedAccount.handle}`} />
                  <InfoRow label="人设" value={selectedAccount.persona} />
                  <InfoRow label="内容方向" value={selectedAccount.contentCategories.join('、')} />
                  <InfoRow label="能力" value={selectedAccount.capabilitySummary} />
                  <InfoRow label="数据来源" value={selectedAccount.dataSource} />
                  <InfoRow label="最近主页同步" value={selectedAccount.lastProfileSyncedAt} />
                  <InfoRow label="最近指标同步" value={selectedAccount.lastMetricsSyncedAt} />
                  <InfoRow label="下一步" value={selectedAccount.nextAction} />
                </Stack>
                {selectedAccount.warnings.length > 0 && (
                  <Alert severity="warning">
                    {selectedAccount.warnings.join('；')}
                  </Alert>
                )}
                <Divider />
                <Stack spacing={1}>
                  <Typography fontWeight={800}>数据快照录入</Typography>
                  <Stack direction={{ xs: 'column', sm: 'row' }} spacing={1}>
                    <TextField
                      size="small"
                      label="粉丝数"
                      value={snapshotForm.followerCount}
                      onChange={(event) => setSnapshotForm((current) => ({ ...current, followerCount: event.target.value }))}
                    />
                    <TextField
                      size="small"
                      label="内容数"
                      value={snapshotForm.contentCount}
                      onChange={(event) => setSnapshotForm((current) => ({ ...current, contentCount: event.target.value }))}
                    />
                  </Stack>
                  <TextField
                    size="small"
                    label="总阅读"
                    value={snapshotForm.totalReads}
                    onChange={(event) => setSnapshotForm((current) => ({ ...current, totalReads: event.target.value }))}
                  />
                  <TextField
                    size="small"
                    label="来源备注"
                    value={snapshotForm.note}
                    onChange={(event) => setSnapshotForm((current) => ({ ...current, note: event.target.value }))}
                    multiline
                    minRows={2}
                  />
                  <Button startIcon={<EditNoteOutlinedIcon />} variant="contained" onClick={submitMockSnapshot}>
                    暂存快照
                  </Button>
                  {savedAt && (
                    <Typography variant="body2" color="text.secondary">
                      最近暂存：{savedAt}
                    </Typography>
                  )}
                </Stack>
              </Stack>
            ) : (
              <EmptyText>请选择一个媒体号账号</EmptyText>
            )}
          </Section>
        </Grid>
      </Grid>

      <Grid container spacing={2}>
        <Grid size={{ xs: 12, lg: 7 }}>
          <Section title="发布与内容指标回流">
            <ProductTable minWidth={820}>
              <TableHead>
                <TableRow>
                  <TableCell>内容</TableCell>
                  <TableCell>账号</TableCell>
                  <TableCell>状态</TableCell>
                  <TableCell>阅读</TableCell>
                  <TableCell>互动</TableCell>
                  <TableCell>外部链接</TableCell>
                </TableRow>
              </TableHead>
              <TableBody>
                {platformContentMetrics.map((item) => (
                  <TableRow key={item.id} hover>
                    <TableCell sx={{ minWidth: 220 }}>
                      <Typography fontWeight={700} sx={secondaryTextSx}>
                        {item.title}
                      </Typography>
                      <Typography variant="body2" color="text.secondary">
                        {item.publishedAt}
                      </Typography>
                    </TableCell>
                    <TableCell sx={{ minWidth: 140 }}>{matrixAccountName(item.accountId)}</TableCell>
                    <TableCell>
                      <Chip size="small" label={sohuStatusLabel(item.status)} color={sohuStatusColor(item.status)} />
                    </TableCell>
                    <TableCell>{compactNumber(item.reads)}</TableCell>
                    <TableCell>{(item.likes + item.comments).toLocaleString()}</TableCell>
                    <TableCell sx={{ minWidth: 220 }}>
                      <Typography variant="body2" sx={wrappingTextSx}>
                        {item.externalUrl || '-'}
                      </Typography>
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </ProductTable>
          </Section>
        </Grid>

        <Grid size={{ xs: 12, lg: 5 }}>
          <Section title="可视字段清单">
            <Stack spacing={1.5}>
              {activePlatform.visibleFieldGroups.map((group) => (
                <Box key={group.title} sx={{ border: '1px solid', borderColor: 'divider', borderRadius: 1, p: 1.5 }}>
                  <Typography fontWeight={800} sx={{ mb: 1 }}>
                    {group.title}
                  </Typography>
                  <Stack direction="row" spacing={0.75} flexWrap="wrap" useFlexGap>
                    {group.fields.map((field) => (
                      <Chip key={field} size="small" label={field} variant="outlined" />
                    ))}
                  </Stack>
                </Box>
              ))}
            </Stack>
          </Section>
        </Grid>
      </Grid>

      <Section title="后续迭代规划">
        <Grid container spacing={1.5}>
          {activePlatform.roadmapItems.map((item, index) => (
            <Grid key={item} size={{ xs: 12, md: 6, xl: 3 }}>
              <Box sx={{ border: '1px solid', borderColor: 'divider', borderRadius: 1, p: 1.5, minHeight: 96 }}>
                <Stack spacing={1}>
                  <Chip label={`P${index + 1}`} color={index === 0 ? 'primary' : 'default'} sx={{ alignSelf: 'flex-start' }} />
                  <Typography variant="body2" sx={wrappingTextSx}>
                    {item}
                  </Typography>
                </Stack>
              </Box>
            </Grid>
          ))}
        </Grid>
      </Section>
    </Stack>
  );

  return (
    <Stack spacing={3}>
      <Stack direction={{ xs: 'column', lg: 'row' }} spacing={2} alignItems={{ xs: 'stretch', lg: 'center' }} justifyContent="space-between">
        <Stack spacing={0.75} sx={{ minWidth: 0 }}>
          <Typography variant="h5" fontWeight={800} sx={wrappingTextSx}>
            媒体矩阵
          </Typography>
          <Typography color="text.secondary" sx={wrappingTextSx}>
            总览看所有平台健康度，平台页维护具体媒体号资产、登录、发布和数据快照。
          </Typography>
        </Stack>
        <Stack direction={{ xs: 'column', sm: 'row' }} spacing={1} flexWrap="wrap" useFlexGap>
          <Button startIcon={<AddIcon />} variant="contained">
            新增账号
          </Button>
          <Button startIcon={<LoginOutlinedIcon />} variant="outlined">
            发起授权
          </Button>
          <Button startIcon={<EditNoteOutlinedIcon />} variant="outlined">
            录入快照
          </Button>
        </Stack>
      </Stack>

      <Box sx={{ borderBottom: 1, borderColor: 'divider' }}>
        <Tabs
          value={activeTab}
          onChange={(_, value: MatrixPlatformKey) => selectTab(value)}
          variant="scrollable"
          scrollButtons="auto"
          allowScrollButtonsMobile
        >
          <Tab value="overview" label="总览" />
          {matrixPlatformConfigs.map((platform) => (
            <Tab key={platform.key} value={platform.key} label={platform.label} />
          ))}
        </Tabs>
      </Box>

      {activeTab === 'overview' ? <PlatformOverviewTab activePlatform={activeTab} onSelectPlatform={selectTab} /> : renderPlatformTab()}
    </Stack>
  );
}

export function CampaignsView({ token, workspaceId, data }: ProductPageProps) {
  const [state, setState] = useState<AsyncState<Campaign[]>>({ ...emptyState, items: [] });
  const [selectedCampaignId, setSelectedCampaignId] = useState('');
  const [calendarItems, setCalendarItems] = useState<CampaignCalendarItem[]>([]);
  const [report, setReport] = useState<CampaignReportSummary | null>(null);
  const [form, setForm] = useState({
    name: '',
    goal: '',
    products: '',
    targetAudiences: '',
    channels: '小红书',
    budgetCents: '0',
    contentQuota: '8',
  });
  const [calendarForm, setCalendarForm] = useState({ title: '', channel: 'xiaohongshu', mediaAccountId: '' });
  const [submitting, setSubmitting] = useState(false);

  const selectedCampaign = state.items.find((item) => item.id === selectedCampaignId) ?? null;

  const load = useCallback(async () => {
    setState((current) => ({ ...current, loading: true, error: null }));
    try {
      const campaigns = await fetchCampaigns(token, workspaceId);
      setState({ items: campaigns, loading: false, error: null });
      setSelectedCampaignId((current) => current || campaigns[0]?.id || '');
    } catch (err) {
      setState((current) => ({ ...current, loading: false, error: errorMessage(err, '战役加载失败') }));
    }
  }, [token, workspaceId]);

  useEffect(() => {
    void load();
  }, [load]);

  useEffect(() => {
    if (!selectedCampaignId) {
      setCalendarItems([]);
      setReport(null);
      return;
    }
    let mounted = true;
    Promise.all([
      fetchCampaignCalendarItems(token, workspaceId, selectedCampaignId),
      fetchCampaignReportSummary(token, workspaceId, selectedCampaignId),
    ])
      .then(([items, summary]) => {
        if (mounted) {
          setCalendarItems(items);
          setReport(summary);
        }
      })
      .catch((err) => {
        if (mounted) {
          setState((current) => ({ ...current, error: errorMessage(err, '战役明细加载失败') }));
        }
      });
    return () => {
      mounted = false;
    };
  }, [selectedCampaignId, token, workspaceId]);

  const submitCampaign = async () => {
    if (!form.name.trim()) {
      setState((current) => ({ ...current, error: '请填写战役名称' }));
      return;
    }
    setSubmitting(true);
    setState((current) => ({ ...current, error: null }));
    try {
      const created = await createCampaign(token, workspaceId, {
        name: form.name.trim(),
        goal: form.goal.trim(),
        products: commaValue(form.products),
        targetAudiences: commaValue(form.targetAudiences),
        channels: commaValue(form.channels),
        budgetCents: Number(form.budgetCents || 0),
        contentQuota: Number(form.contentQuota || 0),
        status: 'planned',
        mediaAccountIds: data.mediaAccounts.map((account) => account.id).slice(0, 3),
        successMetrics: ['impression', 'engagement', 'conversion'],
      });
      setForm({ name: '', goal: '', products: '', targetAudiences: '', channels: '小红书', budgetCents: '0', contentQuota: '8' });
      setSelectedCampaignId(created.id);
      await load();
    } catch (err) {
      setState((current) => ({ ...current, error: errorMessage(err, '战役创建失败') }));
    } finally {
      setSubmitting(false);
    }
  };

  const submitCalendarItem = async () => {
    if (!selectedCampaignId || !calendarForm.title.trim()) {
      setState((current) => ({ ...current, error: '请选择战役并填写排期标题' }));
      return;
    }
    setSubmitting(true);
    setState((current) => ({ ...current, error: null }));
    try {
      await createCampaignCalendarItem(token, workspaceId, selectedCampaignId, {
        title: calendarForm.title.trim(),
        channel: calendarForm.channel,
        mediaAccountId: calendarForm.mediaAccountId,
        contentType: 'xiaohongshu_long_article',
        status: 'planned',
        approvalRequired: true,
        approvalStatus: 'pending',
      });
      setCalendarForm({ title: '', channel: 'xiaohongshu', mediaAccountId: '' });
      const [items, summary] = await Promise.all([
        fetchCampaignCalendarItems(token, workspaceId, selectedCampaignId),
        fetchCampaignReportSummary(token, workspaceId, selectedCampaignId),
      ]);
      setCalendarItems(items);
      setReport(summary);
    } catch (err) {
      setState((current) => ({ ...current, error: errorMessage(err, '排期创建失败') }));
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <Stack spacing={3}>
      <Grid container spacing={2}>
        <MetricCard label="战役" value={state.items.length} helper="跨账号内容项目" />
        <MetricCard label="排期" value={calendarItems.length} helper="选中战役日历项" />
        <MetricCard label="已发布" value={report?.publishedItemCount ?? 0} helper="选中战役发布产出" />
        <MetricCard label="异常" value={report?.failedItemCount ?? 0} helper="选中战役失败项" tone={(report?.failedItemCount ?? 0) > 0 ? 'error' : 'primary'} />
      </Grid>

      {state.error && <Alert severity="error">{state.error}</Alert>}

      <Grid container spacing={2}>
        <Grid size={{ xs: 12, lg: 4 }}>
          <Section title="新建战役">
            <Stack spacing={1.5}>
              <TextField size="small" label="名称" value={form.name} onChange={(event) => setForm((current) => ({ ...current, name: event.target.value }))} />
              <TextField size="small" label="目标" value={form.goal} onChange={(event) => setForm((current) => ({ ...current, goal: event.target.value }))} multiline minRows={2} />
              <TextField size="small" label="产品" value={form.products} onChange={(event) => setForm((current) => ({ ...current, products: event.target.value }))} placeholder="逗号分隔" />
              <TextField size="small" label="目标受众" value={form.targetAudiences} onChange={(event) => setForm((current) => ({ ...current, targetAudiences: event.target.value }))} placeholder="逗号分隔" />
              <TextField size="small" label="渠道" value={form.channels} onChange={(event) => setForm((current) => ({ ...current, channels: event.target.value }))} />
              <Stack direction={{ xs: 'column', sm: 'row' }} spacing={1}>
                <TextField size="small" label="预算分" value={form.budgetCents} onChange={(event) => setForm((current) => ({ ...current, budgetCents: event.target.value }))} type="number" />
                <TextField size="small" label="内容配额" value={form.contentQuota} onChange={(event) => setForm((current) => ({ ...current, contentQuota: event.target.value }))} type="number" />
              </Stack>
              <Button startIcon={<AddIcon />} variant="contained" onClick={submitCampaign} disabled={submitting}>
                创建战役
              </Button>
            </Stack>
          </Section>
        </Grid>

        <Grid size={{ xs: 12, lg: 8 }}>
          <Section
            title="战役列表"
            action={
              <Button startIcon={<AutorenewIcon />} onClick={load} disabled={state.loading}>
                刷新
              </Button>
            }
          >
            {state.loading ? (
              <LoadingRow label="正在加载战役" />
            ) : state.items.length === 0 ? (
              <EmptyText>暂无战役</EmptyText>
            ) : (
              <ProductTable minWidth={780}>
                <TableHead>
                  <TableRow>
                    <TableCell>名称</TableCell>
                    <TableCell>目标</TableCell>
                    <TableCell>状态</TableCell>
                    <TableCell>渠道</TableCell>
                    <TableCell>预算</TableCell>
                  </TableRow>
                </TableHead>
                <TableBody>
                  {state.items.map((campaign) => (
                    <TableRow
                      key={campaign.id}
                      hover
                      selected={campaign.id === selectedCampaignId}
                      onClick={() => setSelectedCampaignId(campaign.id)}
                      sx={{ cursor: 'pointer' }}
                    >
                      <TableCell sx={{ minWidth: 180 }}>
                        <Typography fontWeight={700} sx={wrappingTextSx}>
                          {campaign.name}
                        </Typography>
                        <Typography variant="body2" color="text.secondary" sx={secondaryTextSx}>
                          {campaign.description || campaign.id}
                        </Typography>
                      </TableCell>
                      <TableCell sx={{ minWidth: 180 }}>{campaign.goal || '-'}</TableCell>
                      <TableCell>
                        <Chip size="small" label={campaign.status} color={statusColor(campaign.status)} />
                      </TableCell>
                      <TableCell sx={{ minWidth: 140 }}>{campaign.channels.join(', ') || '-'}</TableCell>
                      <TableCell>{formatMoney(campaign.budgetCents, campaign.currency)}</TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </ProductTable>
            )}
          </Section>
        </Grid>
      </Grid>

      <Grid container spacing={2}>
        <Grid size={{ xs: 12, lg: 4 }}>
          <Section title="新增排期">
            <Stack spacing={1.5}>
              <Typography color="text.secondary" sx={wrappingTextSx}>
                {selectedCampaign ? selectedCampaign.name : '请选择战役'}
              </Typography>
              <TextField size="small" label="排期标题" value={calendarForm.title} onChange={(event) => setCalendarForm((current) => ({ ...current, title: event.target.value }))} />
              <TextField size="small" label="渠道" value={calendarForm.channel} onChange={(event) => setCalendarForm((current) => ({ ...current, channel: event.target.value }))} />
              <FormControl size="small" fullWidth>
                <InputLabel>媒体账号</InputLabel>
                <Select label="媒体账号" value={calendarForm.mediaAccountId} onChange={(event) => setCalendarForm((current) => ({ ...current, mediaAccountId: String(event.target.value) }))}>
                  <MenuItem value="">暂不指定</MenuItem>
                  {data.mediaAccounts.map((account) => (
                    <MenuItem key={account.id} value={account.id}>
                      {account.name}
                    </MenuItem>
                  ))}
                </Select>
              </FormControl>
              <Button startIcon={<PlaylistAddCheckOutlinedIcon />} variant="contained" onClick={submitCalendarItem} disabled={submitting || !selectedCampaignId}>
                创建排期
              </Button>
            </Stack>
          </Section>
        </Grid>
        <Grid size={{ xs: 12, lg: 8 }}>
          <Section title="战役排期">
            {calendarItems.length === 0 ? (
              <EmptyText>暂无排期</EmptyText>
            ) : (
              <ProductTable minWidth={760}>
                <TableHead>
                  <TableRow>
                    <TableCell>标题</TableCell>
                    <TableCell>账号</TableCell>
                    <TableCell>状态</TableCell>
                    <TableCell>审核</TableCell>
                    <TableCell>发布时间窗</TableCell>
                  </TableRow>
                </TableHead>
                <TableBody>
                  {calendarItems.map((item) => (
                    <TableRow key={item.id} hover>
                      <TableCell sx={{ minWidth: 180 }}>
                        <Typography fontWeight={700} sx={wrappingTextSx}>
                          {item.title}
                        </Typography>
                        <Typography variant="body2" color="text.secondary" sx={secondaryTextSx}>
                          {item.brief || item.contentType}
                        </Typography>
                      </TableCell>
                      <TableCell sx={{ minWidth: 140 }}>{accountName(data.mediaAccounts, item.mediaAccountId)}</TableCell>
                      <TableCell>
                        <Chip size="small" label={item.status} color={statusColor(item.status)} />
                      </TableCell>
                      <TableCell>{item.approvalRequired ? item.approvalStatus || 'required' : '无需审核'}</TableCell>
                      <TableCell>{item.publishWindowStartAt ? formatDate(item.publishWindowStartAt) : '-'}</TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </ProductTable>
            )}
          </Section>
        </Grid>
      </Grid>
    </Stack>
  );
}

export function CreatorsView({ token, workspaceId }: ProductPageProps) {
  const [state, setState] = useState<
    AsyncState<{
      creators: Creator[];
      shortlists: CreatorShortlist[];
      briefs: CreatorCampaignBrief[];
      orders: CreatorOrder[];
      deliverables: CreatorDeliverable[];
      settlements: CreatorSettlement[];
    }>
  >({
    ...emptyState,
    items: { creators: [], shortlists: [], briefs: [], orders: [], deliverables: [], settlements: [] },
  });
  const [selectedCreatorId, setSelectedCreatorId] = useState('');
  const [briefForm, setBriefForm] = useState({ title: '', objective: '', budgetCents: '100000' });
  const [submitting, setSubmitting] = useState(false);

  const load = useCallback(async () => {
    setState((current) => ({ ...current, loading: true, error: null }));
    try {
      const [creators, shortlists, briefs, orders, deliverables, settlements] = await Promise.all([
        fetchCreators(token, workspaceId),
        fetchCreatorShortlists(token, workspaceId),
        fetchCreatorCampaignBriefs(token, workspaceId),
        fetchCreatorOrders(token, workspaceId),
        fetchCreatorDeliverables(token, workspaceId),
        fetchCreatorSettlements(token, workspaceId),
      ]);
      setState({ items: { creators, shortlists, briefs, orders, deliverables, settlements }, loading: false, error: null });
      setSelectedCreatorId((current) => current || creators[0]?.id || '');
    } catch (err) {
      setState((current) => ({ ...current, loading: false, error: errorMessage(err, '达人合作数据加载失败') }));
    }
  }, [token, workspaceId]);

  useEffect(() => {
    void load();
  }, [load]);

  const selectedCreator = state.items.creators.find((item) => item.id === selectedCreatorId) ?? null;

  const addShortlist = async (creator: Creator) => {
    setSubmitting(true);
    setState((current) => ({ ...current, error: null }));
    try {
      await createCreatorShortlist(token, workspaceId, {
        creatorId: creator.id,
        name: '默认候选池',
        fitScore: 80,
        qualificationStatus: 'qualified',
        brandSafetyLevel: creator.brandSafetyLevel || 'medium',
      });
      await load();
    } catch (err) {
      setState((current) => ({ ...current, error: errorMessage(err, '加入候选失败') }));
    } finally {
      setSubmitting(false);
    }
  };

  const createBrief = async () => {
    if (!briefForm.title.trim()) {
      setState((current) => ({ ...current, error: '请填写 brief 标题' }));
      return;
    }
    setSubmitting(true);
    try {
      await createCreatorCampaignBrief(token, workspaceId, {
        title: briefForm.title.trim(),
        objective: briefForm.objective.trim(),
        platformTargets: ['小红书'],
        deliverableRequirements: ['1 篇图文笔记'],
        disclosureRequirements: ['正文需明确品牌合作'],
        prohibitedClaims: ['不得承诺增长效果'],
        authorizationScope: '达人自行发布，品牌不得登录达人账号',
        contentUsageRights: '品牌可在自有渠道二次使用 90 天',
        budgetCents: Number(briefForm.budgetCents || 0),
        currency: 'CNY',
        status: 'active',
      });
      setBriefForm({ title: '', objective: '', budgetCents: '100000' });
      await load();
    } catch (err) {
      setState((current) => ({ ...current, error: errorMessage(err, 'Brief 创建失败') }));
    } finally {
      setSubmitting(false);
    }
  };

  const createOrder = async () => {
    const brief = state.items.briefs[0];
    if (!selectedCreator || !brief) {
      setState((current) => ({ ...current, error: '请先选择达人并创建 brief' }));
      return;
    }
    setSubmitting(true);
    try {
      await createCreatorOrder(token, workspaceId, {
        briefId: brief.id,
        creatorId: selectedCreator.id,
        depositCents: Math.round(brief.budgetCents * 0.3),
        lastMessage: '请按 brief 提交初稿',
      });
      await load();
    } catch (err) {
      setState((current) => ({ ...current, error: errorMessage(err, '订单创建失败') }));
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <Stack spacing={3}>
      <Grid container spacing={2}>
        <MetricCard label="达人库" value={state.items.creators.length} helper="可合作达人资源" />
        <MetricCard label="候选" value={state.items.shortlists.length} helper="已加入工作区候选池" />
        <MetricCard label="订单" value={state.items.orders.length} helper="合作履约记录" />
        <MetricCard label="待结算" value={state.items.settlements.filter((item) => item.status !== 'paid').length} helper="财务待处理" />
      </Grid>
      {state.error && <Alert severity="error">{state.error}</Alert>}

      <Grid container spacing={2}>
        <Grid size={{ xs: 12, lg: 8 }}>
          <Section
            title="达人库"
            action={
              <Button startIcon={<AutorenewIcon />} onClick={load} disabled={state.loading}>
                刷新
              </Button>
            }
          >
            {state.loading ? (
              <LoadingRow label="正在加载达人库" />
            ) : state.items.creators.length === 0 ? (
              <EmptyText>暂无达人数据</EmptyText>
            ) : (
              <ProductTable minWidth={800}>
                <TableHead>
                  <TableRow>
                    <TableCell>达人</TableCell>
                    <TableCell>领域</TableCell>
                    <TableCell>价格</TableCell>
                    <TableCell>状态</TableCell>
                    <TableCell align="right">操作</TableCell>
                  </TableRow>
                </TableHead>
                <TableBody>
                  {state.items.creators.map((creator) => (
                    <TableRow
                      key={creator.id}
                      hover
                      selected={creator.id === selectedCreatorId}
                      onClick={() => setSelectedCreatorId(creator.id)}
                      sx={{ cursor: 'pointer' }}
                    >
                      <TableCell sx={{ minWidth: 220 }}>
                        <Typography fontWeight={700} sx={wrappingTextSx}>
                          {creator.displayName}
                        </Typography>
                        <Typography variant="body2" color="text.secondary" sx={secondaryTextSx}>
                          {creator.bio}
                        </Typography>
                      </TableCell>
                      <TableCell sx={{ minWidth: 140 }}>{creator.verticals.join(', ') || '-'}</TableCell>
                      <TableCell>{formatMoney(creator.basePriceCents, creator.currency)}</TableCell>
                      <TableCell>
                        <Chip size="small" label={`${creator.verificationState}/${creator.availabilityStatus}`} color={statusColor(creator.verificationState)} />
                      </TableCell>
                      <TableCell align="right">
                        <Button
                          size="small"
                          onClick={(event) => {
                            event.stopPropagation();
                            void addShortlist(creator);
                          }}
                          disabled={submitting}
                          sx={{ whiteSpace: 'nowrap' }}
                        >
                          加候选
                        </Button>
                      </TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </ProductTable>
            )}
          </Section>
        </Grid>
        <Grid size={{ xs: 12, lg: 4 }}>
          <Section title="合作动作">
            <Stack spacing={1.5}>
              <InfoRow label="当前达人" value={selectedCreator?.displayName ?? '-'} />
              <TextField size="small" label="Brief 标题" value={briefForm.title} onChange={(event) => setBriefForm((current) => ({ ...current, title: event.target.value }))} />
              <TextField size="small" label="目标" value={briefForm.objective} onChange={(event) => setBriefForm((current) => ({ ...current, objective: event.target.value }))} multiline minRows={2} />
              <TextField size="small" label="预算分" type="number" value={briefForm.budgetCents} onChange={(event) => setBriefForm((current) => ({ ...current, budgetCents: event.target.value }))} />
              <Button startIcon={<LibraryBooksOutlinedIcon />} variant="outlined" onClick={createBrief} disabled={submitting}>
                创建 Brief
              </Button>
              <Button startIcon={<PriceCheckOutlinedIcon />} variant="contained" onClick={createOrder} disabled={submitting || !selectedCreator}>
                下合作单
              </Button>
            </Stack>
          </Section>
        </Grid>
      </Grid>

      <Grid container spacing={2}>
        <Grid size={{ xs: 12, lg: 6 }}>
          <Section title="合作订单">
            {state.items.orders.length === 0 ? (
              <EmptyText>暂无订单</EmptyText>
            ) : (
              <ProductTable minWidth={620}>
                <TableHead>
                  <TableRow>
                    <TableCell>达人</TableCell>
                    <TableCell>状态</TableCell>
                    <TableCell>金额</TableCell>
                    <TableCell>消息</TableCell>
                  </TableRow>
                </TableHead>
                <TableBody>
                  {state.items.orders.map((order) => (
                    <TableRow key={order.id} hover>
                      <TableCell sx={{ minWidth: 160 }}>
                        {state.items.creators.find((creator) => creator.id === order.creatorId)?.displayName ?? order.creatorId}
                      </TableCell>
                      <TableCell>
                        <Chip size="small" label={order.status} color={statusColor(order.status)} />
                      </TableCell>
                      <TableCell>{formatMoney(order.priceCents, order.currency)}</TableCell>
                      <TableCell sx={{ minWidth: 180 }}>{order.lastMessage || '-'}</TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </ProductTable>
            )}
          </Section>
        </Grid>
        <Grid size={{ xs: 12, lg: 6 }}>
          <Section title="交付与结算">
            <Stack spacing={2}>
              <Stack direction="row" spacing={1} flexWrap="wrap" useFlexGap sx={{ minWidth: 0 }}>
                {state.items.deliverables.map((item) => (
                  <Chip key={item.id} label={`${item.title || item.id}: ${item.status}`} color={statusColor(item.status)} />
                ))}
                {state.items.deliverables.length === 0 && <EmptyText>暂无交付物</EmptyText>}
              </Stack>
              <Divider />
              <Stack spacing={1}>
                {state.items.settlements.map((item) => (
                  <InfoRow key={item.id} label={item.orderId} value={`${item.status} / ${formatMoney(item.creatorPayoutCents, item.currency)}`} />
                ))}
                {state.items.settlements.length === 0 && <EmptyText>暂无结算记录</EmptyText>}
              </Stack>
            </Stack>
          </Section>
        </Grid>
      </Grid>
    </Stack>
  );
}

export function SkillPackagesView({ token, workspaceId }: ProductPageProps) {
  const [state, setState] = useState<
    AsyncState<{
      marketplace: SkillPackageMarketplaceItem[];
      installed: InstalledSkillPackage[];
      usage: SkillPackageUsageMetric[];
    }>
  >({ ...emptyState, items: { marketplace: [], installed: [], usage: [] } });
  const [installingId, setInstallingId] = useState('');

  const load = useCallback(async () => {
    setState((current) => ({ ...current, loading: true, error: null }));
    try {
      const [marketplace, installed, usage] = await Promise.all([
        fetchSkillPackageMarketplace(token, workspaceId),
        fetchInstalledSkillPackages(token, workspaceId),
        fetchSkillPackageUsage(token, workspaceId),
      ]);
      setState({ items: { marketplace, installed, usage }, loading: false, error: null });
    } catch (err) {
      setState((current) => ({ ...current, loading: false, error: errorMessage(err, '技能包数据加载失败') }));
    }
  }, [token, workspaceId]);

  useEffect(() => {
    void load();
  }, [load]);

  const install = async (item: SkillPackageMarketplaceItem, paid: boolean) => {
    setInstallingId(item.package.id);
    setState((current) => ({ ...current, error: null }));
    try {
      if (paid) {
        await purchaseSkillPackage(token, workspaceId, item.package.id, { versionId: item.version?.id });
      } else {
        await installSkillPackage(token, workspaceId, item.package.id, { versionId: item.version?.id });
      }
      await load();
    } catch (err) {
      setState((current) => ({ ...current, error: errorMessage(err, '技能包安装失败') }));
    } finally {
      setInstallingId('');
    }
  };

  return (
    <Stack spacing={3}>
      <Grid container spacing={2}>
        <MetricCard label="市场技能包" value={state.items.marketplace.length} helper="平台已发布创作能力" />
        <MetricCard label="已安装" value={state.items.installed.length} helper="当前工作区可用于 AI 生成" />
        <MetricCard label="调用次数" value={state.items.usage.reduce((sum, item) => sum + item.count, 0)} helper="技能包使用记录" />
        <MetricCard label="生成记录" value={state.items.usage.filter((item) => item.metricType === 'generation').length} helper="接入生成链路次数" />
      </Grid>
      {state.error && <Alert severity="error">{state.error}</Alert>}

      <Section
        title="创作技能包市场"
        action={
          <Button startIcon={<AutorenewIcon />} onClick={load} disabled={state.loading}>
            刷新
          </Button>
        }
      >
        {state.loading ? (
          <LoadingRow label="正在加载技能包" />
        ) : state.items.marketplace.length === 0 ? (
          <EmptyText>暂无已发布技能包。请在平台后台创建并发布技能包。</EmptyText>
        ) : (
          <Grid container spacing={2}>
            {state.items.marketplace.map((item) => (
              <Grid key={item.package.id} size={{ xs: 12, md: 6, xl: 4 }}>
                <Card variant="outlined" sx={{ height: '100%' }}>
                  <CardContent sx={{ height: '100%', minHeight: 244 }}>
                    <Stack spacing={1.5} sx={{ height: '100%' }}>
                      <Stack direction={{ xs: 'column', sm: 'row' }} justifyContent="space-between" spacing={1} alignItems={{ xs: 'flex-start', sm: 'center' }}>
                        <Typography variant="h3" sx={wrappingTextSx}>
                          {item.package.name}
                        </Typography>
                        <Chip size="small" label={item.installed ? '已安装' : item.package.category || '技能'} color={item.installed ? 'success' : 'info'} />
                      </Stack>
                      <Typography color="text.secondary" sx={{ ...secondaryTextSx, minHeight: 48 }}>
                        {item.package.description || '-'}
                      </Typography>
                      <Stack direction="row" spacing={1} flexWrap="wrap" useFlexGap>
                        <Chip size="small" label={item.package.targetPlatform || '通用平台'} />
                        <Chip size="small" label={item.package.targetIndustry || '通用行业'} />
                        <Chip size="small" label={formatMoney(item.package.priceCents, item.package.currency)} />
                      </Stack>
                      <Typography variant="body2" color="text.secondary" sx={wrappingTextSx}>
                        版本 {item.version?.version ?? '-'} / 作者 {item.package.authorName || item.package.authorId || '-'}
                      </Typography>
                      <Stack direction={{ xs: 'column', sm: 'row' }} spacing={1} sx={{ mt: 'auto' }}>
                        <Button variant="contained" disabled={item.installed || installingId === item.package.id} onClick={() => install(item, false)}>
                          试用安装
                        </Button>
                        <Button variant="outlined" disabled={item.installed || installingId === item.package.id} onClick={() => install(item, true)}>
                          购买
                        </Button>
                      </Stack>
                    </Stack>
                  </CardContent>
                </Card>
              </Grid>
            ))}
          </Grid>
        )}
      </Section>

      <Section title="已安装技能包">
        {state.items.installed.length === 0 ? (
          <EmptyText>暂无已安装技能包</EmptyText>
        ) : (
          <ProductTable minWidth={720}>
            <TableHead>
              <TableRow>
                <TableCell>技能包</TableCell>
                <TableCell>版本</TableCell>
                <TableCell>来源</TableCell>
                <TableCell>席位</TableCell>
                <TableCell>到期</TableCell>
              </TableRow>
            </TableHead>
            <TableBody>
              {state.items.installed.map((item) => (
                <TableRow key={item.entitlement.id} hover>
                  <TableCell sx={{ minWidth: 180 }}>{item.package?.name ?? item.entitlement.packageId}</TableCell>
                  <TableCell sx={{ minWidth: 140 }}>{item.version?.version ?? item.entitlement.versionId}</TableCell>
                  <TableCell>
                    <Chip size="small" label={item.entitlement.source} color={statusColor(item.entitlement.status)} />
                  </TableCell>
                  <TableCell>{item.entitlement.seats}</TableCell>
                  <TableCell>{item.entitlement.expiresAt ? formatDate(item.entitlement.expiresAt) : '-'}</TableCell>
                </TableRow>
              ))}
            </TableBody>
          </ProductTable>
        )}
      </Section>
    </Stack>
  );
}

export function BrandComplianceView({ token, workspaceId, data }: ProductPageProps) {
  const [state, setState] = useState<
    AsyncState<{
      assets: BrandAsset[];
      guardrails: BrandGuardrail[];
      workflows: ApprovalWorkflow[];
      tasks: ApprovalTask[];
      checks: ComplianceCheck[];
      relations: AgencyClientRelation[];
      reports: ReportPackage[];
      recommendations: StrategyRecommendation[];
    }>
  >({
    ...emptyState,
    items: {
      assets: [],
      guardrails: [],
      workflows: [],
      tasks: [],
      checks: [],
      relations: [],
      reports: [],
      recommendations: [],
    },
  });
  const [assetForm, setAssetForm] = useState({ name: '', type: 'forbidden_phrase', content: '', channels: 'xiaohongshu', tags: 'compliance' });
  const [checkForm, setCheckForm] = useState({ title: '', content: '', channel: 'xiaohongshu' });
  const [submitting, setSubmitting] = useState(false);

  const load = useCallback(async () => {
    setState((current) => ({ ...current, loading: true, error: null }));
    try {
      const [assets, guardrails, workflows, tasks, checks, relations, reports, recommendations] = await Promise.all([
        fetchBrandAssets(token, workspaceId),
        fetchBrandGuardrails(token, workspaceId),
        fetchApprovalWorkflows(token, workspaceId),
        fetchApprovalTasks(token, workspaceId),
        fetchComplianceChecks(token, workspaceId),
        fetchAgencyClientRelations(token, workspaceId),
        fetchReportPackages(token, workspaceId),
        fetchStrategyRecommendations(token, workspaceId),
      ]);
      setState({
        items: { assets, guardrails, workflows, tasks, checks, relations, reports, recommendations },
        loading: false,
        error: null,
      });
    } catch (err) {
      setState((current) => ({ ...current, loading: false, error: errorMessage(err, '品牌合规数据加载失败') }));
    }
  }, [token, workspaceId]);

  useEffect(() => {
    void load();
  }, [load]);

  const createAssetAndGuardrail = async () => {
    if (!assetForm.name.trim()) {
      setState((current) => ({ ...current, error: '请填写品牌资产名称' }));
      return;
    }
    setSubmitting(true);
    try {
      const asset = await createBrandAsset(token, workspaceId, {
        name: assetForm.name.trim(),
        type: assetForm.type,
        content: assetForm.content.trim(),
        channels: commaValue(assetForm.channels),
        tags: commaValue(assetForm.tags),
        source: 'manual',
      });
      if (asset.content.trim()) {
        await createBrandGuardrail(token, workspaceId, {
          assetId: asset.id,
          name: `${asset.name} 守则`,
          category: 'claim_risk',
          channel: asset.channels[0] || 'xiaohongshu',
          severity: 'high',
          rules: [asset.content],
          action: '提交复核',
        });
      }
      setAssetForm({ name: '', type: 'forbidden_phrase', content: '', channels: 'xiaohongshu', tags: 'compliance' });
      await load();
    } catch (err) {
      setState((current) => ({ ...current, error: errorMessage(err, '品牌资产创建失败') }));
    } finally {
      setSubmitting(false);
    }
  };

  const submitCheck = async () => {
    if (!checkForm.title.trim() && !checkForm.content.trim()) {
      setState((current) => ({ ...current, error: '请填写待检查内容' }));
      return;
    }
    setSubmitting(true);
    try {
      await submitComplianceCheck(token, workspaceId, {
        resourceType: 'content',
        channel: checkForm.channel,
        title: checkForm.title,
        content: checkForm.content,
      });
      setCheckForm({ title: '', content: '', channel: 'xiaohongshu' });
      await load();
    } catch (err) {
      setState((current) => ({ ...current, error: errorMessage(err, '合规检查提交失败') }));
    } finally {
      setSubmitting(false);
    }
  };

  const createWorkflow = async () => {
    setSubmitting(true);
    try {
      await createApprovalWorkflow(token, workspaceId, {
        resourceType: 'content',
        resourceId: data.contents[0]?.id || 'manual_resource',
        name: '品牌法务双审',
        status: 'active',
        stages: [
          { name: '品牌审核', approverRole: 'brand_manager', requiredApprovals: 1 },
          { name: '法务审核', approverRole: 'legal', requiredApprovals: 1 },
        ],
      });
      await load();
    } catch (err) {
      setState((current) => ({ ...current, error: errorMessage(err, '审批流创建失败') }));
    } finally {
      setSubmitting(false);
    }
  };

  const approveTask = async (task: ApprovalTask) => {
    setSubmitting(true);
    try {
      await processApprovalTask(token, workspaceId, task.id, { decision: 'approve', comment: '页面快速通过' });
      await load();
    } catch (err) {
      setState((current) => ({ ...current, error: errorMessage(err, '审批任务处理失败') }));
    } finally {
      setSubmitting(false);
    }
  };

  const generateReport = async () => {
    setSubmitting(true);
    try {
      await generateReportPackage(token, workspaceId, {
        name: '经营与合规报告',
        reportType: 'monthly',
        audience: 'workspace_operator',
        periodStart: todayInputValue(),
        periodEnd: nextMonthInputValue(),
        sections: ['content_delivery', 'compliance_risks', 'media_matrix'],
      });
      await load();
    } catch (err) {
      setState((current) => ({ ...current, error: errorMessage(err, '报告生成失败') }));
    } finally {
      setSubmitting(false);
    }
  };

  const createRelation = async () => {
    setSubmitting(true);
    try {
      await createAgencyClientRelation(token, workspaceId, {
        clientWorkspaceId: data.workspaces.find((workspace) => workspace.id !== workspaceId)?.id || workspaceId,
        clientName: '示例客户',
        status: 'active',
        scopes: ['content_review', 'reporting'],
        notes: '用于代理服务交付与月报归档',
      });
      await load();
    } catch (err) {
      setState((current) => ({ ...current, error: errorMessage(err, '客户关系创建失败') }));
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <Stack spacing={3}>
      <Grid container spacing={2}>
        <MetricCard label="品牌资产" value={state.items.assets.length} helper="表达、禁用词和素材规范" />
        <MetricCard label="守则" value={state.items.guardrails.length} helper="合规自动检查规则" />
        <MetricCard label="待审批" value={state.items.tasks.filter((item) => item.status === 'pending').length} helper="品牌/法务任务" />
        <MetricCard label="高风险检查" value={state.items.checks.filter((item) => item.riskLevel === 'high').length} helper="需人工复核" tone={state.items.checks.some((item) => item.riskLevel === 'high') ? 'error' : 'primary'} />
      </Grid>
      {state.error && <Alert severity="error">{state.error}</Alert>}

      <Grid container spacing={2}>
        <Grid size={{ xs: 12, lg: 4 }}>
          <Section title="品牌资产与守则">
            <Stack spacing={1.5}>
              <TextField size="small" label="名称" value={assetForm.name} onChange={(event) => setAssetForm((current) => ({ ...current, name: event.target.value }))} />
              <TextField size="small" label="类型" value={assetForm.type} onChange={(event) => setAssetForm((current) => ({ ...current, type: event.target.value }))} />
              <TextField size="small" label="内容/禁用表达" value={assetForm.content} onChange={(event) => setAssetForm((current) => ({ ...current, content: event.target.value }))} multiline minRows={2} />
              <TextField size="small" label="渠道" value={assetForm.channels} onChange={(event) => setAssetForm((current) => ({ ...current, channels: event.target.value }))} />
              <TextField size="small" label="标签" value={assetForm.tags} onChange={(event) => setAssetForm((current) => ({ ...current, tags: event.target.value }))} />
              <Button startIcon={<FactCheckOutlinedIcon />} variant="contained" onClick={createAssetAndGuardrail} disabled={submitting}>
                创建资产和守则
              </Button>
            </Stack>
          </Section>
        </Grid>
        <Grid size={{ xs: 12, lg: 8 }}>
          <Section
            title="品牌资产列表"
            action={
              <Button startIcon={<AutorenewIcon />} onClick={load} disabled={state.loading}>
                刷新
              </Button>
            }
          >
            {state.loading ? (
              <LoadingRow label="正在加载品牌合规数据" />
            ) : state.items.assets.length === 0 ? (
              <EmptyText>暂无品牌资产</EmptyText>
            ) : (
              <ProductTable minWidth={720}>
                <TableHead>
                  <TableRow>
                    <TableCell>资产</TableCell>
                    <TableCell>渠道</TableCell>
                    <TableCell>标签</TableCell>
                    <TableCell>状态</TableCell>
                  </TableRow>
                </TableHead>
                <TableBody>
                  {state.items.assets.map((asset) => (
                    <TableRow key={asset.id} hover>
                      <TableCell sx={{ minWidth: 220 }}>
                        <Typography fontWeight={700} sx={wrappingTextSx}>
                          {asset.name}
                        </Typography>
                        <Typography variant="body2" color="text.secondary" sx={secondaryTextSx}>
                          {asset.content || asset.description || asset.type}
                        </Typography>
                      </TableCell>
                      <TableCell sx={{ minWidth: 140 }}>{asset.channels.join(', ') || '-'}</TableCell>
                      <TableCell sx={{ minWidth: 140 }}>{asset.tags.join(', ') || '-'}</TableCell>
                      <TableCell>
                        <Chip size="small" label={asset.status} color={statusColor(asset.status)} />
                      </TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </ProductTable>
            )}
          </Section>
        </Grid>
      </Grid>

      <Grid container spacing={2}>
        <Grid size={{ xs: 12, lg: 4 }}>
          <Section title="合规检查">
            <Stack spacing={1.5}>
              <TextField size="small" label="标题" value={checkForm.title} onChange={(event) => setCheckForm((current) => ({ ...current, title: event.target.value }))} />
              <TextField size="small" label="渠道" value={checkForm.channel} onChange={(event) => setCheckForm((current) => ({ ...current, channel: event.target.value }))} />
              <TextField size="small" label="待检查内容" value={checkForm.content} onChange={(event) => setCheckForm((current) => ({ ...current, content: event.target.value }))} multiline minRows={4} />
              <Button startIcon={<CheckCircleOutlineIcon />} variant="contained" onClick={submitCheck} disabled={submitting}>
                提交检查
              </Button>
              <Button startIcon={<PlaylistAddCheckOutlinedIcon />} variant="outlined" onClick={createWorkflow} disabled={submitting}>
                创建审批流
              </Button>
            </Stack>
          </Section>
        </Grid>
        <Grid size={{ xs: 12, lg: 8 }}>
          <Section title="检查与审批">
            <Stack spacing={2}>
              {state.items.checks.length === 0 ? (
                <EmptyText>暂无合规检查</EmptyText>
              ) : (
                <ProductTable minWidth={680}>
                  <TableHead>
                    <TableRow>
                      <TableCell>检查</TableCell>
                      <TableCell>风险</TableCell>
                      <TableCell>发现</TableCell>
                      <TableCell>时间</TableCell>
                    </TableRow>
                  </TableHead>
                  <TableBody>
                    {state.items.checks.map((check) => (
                      <TableRow key={check.id} hover>
                        <TableCell sx={{ minWidth: 220 }}>
                          <Typography fontWeight={700} sx={wrappingTextSx}>
                            {check.summary || check.resourceType}
                          </Typography>
                          <Typography variant="body2" color="text.secondary" sx={wrappingTextSx}>
                            {check.channel}
                          </Typography>
                        </TableCell>
                        <TableCell>
                          <Chip size="small" label={check.riskLevel || check.status} color={check.riskLevel === 'high' ? 'error' : statusColor(check.status)} />
                        </TableCell>
                        <TableCell>{check.findings.length}</TableCell>
                        <TableCell>{formatDate(check.createdAt)}</TableCell>
                      </TableRow>
                    ))}
                  </TableBody>
                </ProductTable>
              )}
              <Divider />
              <Stack spacing={1}>
                {state.items.tasks.map((task) => (
                  <Stack
                    key={task.id}
                    direction={{ xs: 'column', md: 'row' }}
                    spacing={1}
                    alignItems={{ xs: 'stretch', md: 'center' }}
                    justifyContent="space-between"
                    sx={{ border: '1px solid', borderColor: 'divider', borderRadius: 1, p: 1.25, minWidth: 0 }}
                  >
                    <Box sx={{ minWidth: 0 }}>
                      <Typography fontWeight={700} sx={wrappingTextSx}>
                        {task.stageName}
                      </Typography>
                      <Typography variant="body2" color="text.secondary" sx={wrappingTextSx}>
                        {task.resourceType}/{task.resourceId}
                      </Typography>
                    </Box>
                    <Stack direction="row" spacing={1} alignItems="center" justifyContent={{ xs: 'space-between', md: 'flex-end' }}>
                      <Chip size="small" label={task.status} color={statusColor(task.status)} />
                      <Button size="small" disabled={task.status !== 'pending' || submitting} onClick={() => approveTask(task)}>
                        通过
                      </Button>
                    </Stack>
                  </Stack>
                ))}
                {state.items.tasks.length === 0 && <EmptyText>暂无审批任务</EmptyText>}
              </Stack>
            </Stack>
          </Section>
        </Grid>
      </Grid>

      <Grid container spacing={2}>
        <Grid size={{ xs: 12, lg: 6 }}>
          <Section
            title="报告包"
            action={
              <Button startIcon={<AutoAwesomeOutlinedIcon />} variant="contained" onClick={generateReport} disabled={submitting}>
                生成报告
              </Button>
            }
          >
            {state.items.reports.length === 0 ? (
              <EmptyText>暂无报告</EmptyText>
            ) : (
              <Stack spacing={1.25}>
                {state.items.reports.map((report) => (
                  <Card key={report.id} variant="outlined">
                    <CardContent>
                      <Stack spacing={1}>
                        <Stack direction={{ xs: 'column', sm: 'row' }} justifyContent="space-between" spacing={1} alignItems={{ xs: 'flex-start', sm: 'center' }}>
                          <Typography variant="h3" sx={wrappingTextSx}>
                            {report.name}
                          </Typography>
                          <Chip size="small" label={report.status} color={statusColor(report.status)} />
                        </Stack>
                        <Typography color="text.secondary" sx={secondaryTextSx}>
                          {report.summary || report.reportType}
                        </Typography>
                        <Typography variant="body2" color="text.secondary" sx={wrappingTextSx}>
                          {report.sections.join(', ')}
                        </Typography>
                      </Stack>
                    </CardContent>
                  </Card>
                ))}
              </Stack>
            )}
          </Section>
        </Grid>
        <Grid size={{ xs: 12, lg: 6 }}>
          <Section
            title="客户关系与策略建议"
            action={
              <Button startIcon={<AddIcon />} onClick={createRelation} disabled={submitting}>
                创建客户关系
              </Button>
            }
          >
            <Stack spacing={2}>
              <Stack spacing={1}>
                {state.items.relations.map((relation) => (
                  <InfoRow key={relation.id} label={relation.clientName || relation.clientWorkspaceId} value={`${relation.status} / ${relation.scopes.join(', ')}`} />
                ))}
                {state.items.relations.length === 0 && <EmptyText>暂无客户关系</EmptyText>}
              </Stack>
              <Divider />
              <Stack spacing={1.25}>
                {state.items.recommendations.map((item) => (
                  <Card key={item.id} variant="outlined">
                    <CardContent>
                      <Typography variant="h3" sx={wrappingTextSx}>
                        {item.title}
                      </Typography>
                      <Typography color="text.secondary" sx={{ ...secondaryTextSx, mt: 0.75 }}>
                        {item.rationale}
                      </Typography>
                      <Typography variant="body2" sx={{ ...wrappingTextSx, mt: 1 }}>
                        {item.action}
                      </Typography>
                    </CardContent>
                  </Card>
                ))}
                {state.items.recommendations.length === 0 && <EmptyText>暂无策略建议</EmptyText>}
              </Stack>
            </Stack>
          </Section>
        </Grid>
      </Grid>
    </Stack>
  );
}

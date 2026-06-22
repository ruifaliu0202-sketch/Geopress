import AddIcon from '@mui/icons-material/Add';
import AutorenewIcon from '@mui/icons-material/Autorenew';
import HelpOutlineIcon from '@mui/icons-material/HelpOutline';
import KeyOutlinedIcon from '@mui/icons-material/KeyOutlined';
import PsychologyAltOutlinedIcon from '@mui/icons-material/PsychologyAltOutlined';
import ScheduleOutlinedIcon from '@mui/icons-material/ScheduleOutlined';
import type {
  WorkspaceAssistantActionCallbacks,
  WorkspaceAssistantActionDescriptor,
  WorkspaceAssistantActionId,
} from './types';

const actionDefaults: WorkspaceAssistantActionDescriptor[] = [
  {
    id: 'generateContent',
    label: '关键词生成内容',
    shortLabel: '生成',
    description: '根据关键词、知识库和内容类型创建可编辑草稿。',
    helper: '从关键词推进到草稿',
    icon: <PsychologyAltOutlinedIcon />,
    tone: 'primary',
    dataTourId: 'assistant-generate',
  },
  {
    id: 'createKnowledgeBase',
    label: '创建知识库包',
    shortLabel: '知识库包',
    description: '新增一组可复用的品牌、产品或行业知识包。',
    helper: '整理一组可复用知识',
    icon: <AddIcon />,
    tone: 'neutral',
    dataTourId: 'assistant-create-knowledge-base',
  },
  {
    id: 'createKnowledgeItem',
    label: '创建引导条目',
    shortLabel: '引导条目',
    description: '记录产品事实、表达风格、禁忌和素材说明。',
    helper: '补充事实和表达限制',
    icon: <AddIcon />,
    tone: 'neutral',
    dataTourId: 'assistant-create-knowledge-item',
  },
  {
    id: 'bindMediaAccount',
    label: '绑定媒体账号',
    shortLabel: '绑定账号',
    description: '连接小红书等媒体账号，为发布任务准备渠道。',
    helper: '连接内容发布渠道',
    icon: <KeyOutlinedIcon />,
    tone: 'success',
    dataTourId: 'assistant-bind-media-account',
  },
  {
    id: 'createSchedule',
    label: '创建发布计划',
    shortLabel: '发布计划',
    description: '把已准备内容排入发布节奏，跟踪计划和任务状态。',
    helper: '安排下一次发布',
    icon: <ScheduleOutlinedIcon />,
    tone: 'neutral',
    dataTourId: 'assistant-create-schedule',
  },
  {
    id: 'openOnboardingGuide',
    label: '打开教学引导',
    shortLabel: '引导',
    description: '重新播放工作区流程教学，检查关键配置是否齐全。',
    helper: '查看完整工作路径',
    icon: <HelpOutlineIcon />,
    tone: 'warning',
    dataTourId: 'assistant-open-guide',
  },
  {
    id: 'refreshWorkspace',
    label: '刷新工作区数据',
    shortLabel: '刷新',
    description: '重新同步当前工作区的知识、账号、内容和任务。',
    helper: '同步最新工作区状态',
    icon: <AutorenewIcon />,
    tone: 'neutral',
    dataTourId: 'assistant-refresh',
  },
];

export const workspaceAssistantActionIds = actionDefaults.map((action) => action.id);

export function createWorkspaceAssistantActions(
  callbacks: WorkspaceAssistantActionCallbacks = {},
  overrides: Partial<Record<WorkspaceAssistantActionId, Partial<WorkspaceAssistantActionDescriptor>>> = {},
): WorkspaceAssistantActionDescriptor[] {
  return actionDefaults.map((action) => {
    const onRun = callbacks[action.id];
    return {
      ...action,
      ...overrides[action.id],
      onRun,
      disabled: overrides[action.id]?.disabled ?? !onRun,
      disabledReason:
        overrides[action.id]?.disabledReason ??
        (!onRun ? '父组件暂未接入这个动作' : action.disabledReason),
    };
  });
}

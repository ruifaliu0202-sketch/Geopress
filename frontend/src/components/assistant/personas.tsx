import { CorgiPersonaPlaceholder } from './CorgiPersonaPlaceholder';
import type { WorkspaceAssistantPersona } from './types';

export const defaultCorgiAssistantPersona: WorkspaceAssistantPersona = {
  id: 'corgi-guide',
  name: 'Corgi',
  title: 'AI 助手',
  greeting: '我会帮你把知识、账号和发布计划串起来。',
  status: '待命中',
  asset: {
    kind: 'component',
    alt: '卡通柯基 AI 助手占位形象',
    node: <CorgiPersonaPlaceholder />,
  },
};

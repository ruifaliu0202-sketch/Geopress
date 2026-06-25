import type { AIThinkingState } from './aiThinkingModel';
import { traceStepsToThinkingSteps } from './aiThinkingModel';
import { WorkflowDrawer } from './WorkflowDrawer';

export function AIThinkingOverlay({ state, onClose }: { state: AIThinkingState; onClose: () => void }) {
  const steps = state.trace ? traceStepsToThinkingSteps(state.trace.steps) : state.steps;
  const warnings = state.trace?.warnings ?? [];
  return (
    <WorkflowDrawer
      state={{
        open: state.open,
        blocking: state.blocking,
        title: state.title || 'AI Thinking',
        subtitle: state.subtitle || (state.trace ? `${state.trace.subscriptionTier || 'free'} 链路 / ${steps.length} 个阶段` : '等待 AI 执行'),
        steps,
        warnings,
      }}
      onClose={onClose}
      emptyText="AI 执行时会显示当前链路、阶段状态和边界提示。"
    />
  );
}

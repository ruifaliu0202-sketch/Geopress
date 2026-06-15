import type { Dispatch, SetStateAction } from 'react';
import type { GenerationTrace } from '../types';

export type AIThinkingStepStatus = 'pending' | 'running' | 'succeeded' | 'failed' | 'skipped';

export type AIThinkingStep = {
  id: string;
  label: string;
  status: AIThinkingStepStatus;
  summary: string;
  details: string[];
  warnings: string[];
};

export type AIThinkingState = {
  open: boolean;
  blocking: boolean;
  title: string;
  subtitle: string;
  steps: AIThinkingStep[];
  trace: GenerationTrace | null;
};

export type FormattingThinkingResponse = {
  fallback?: boolean;
  fallbackError?: string;
};

export type FormattingThinkingRunOptions<T extends FormattingThinkingResponse> = {
  subtitle?: string;
  request: () => Promise<T>;
  onSuccess: (response: T) => void;
  onFailure?: (message: string) => void;
  durationMs?: number;
};

export type RunFormattingThinking = <T extends FormattingThinkingResponse>(options: FormattingThinkingRunOptions<T>) => Promise<T | null>;

export const defaultThinkingTraceDurationMs = 5000;

export const initialThinkingState: AIThinkingState = {
  open: false,
  blocking: false,
  title: 'AI Thinking',
  subtitle: '',
  steps: [],
  trace: null,
};

export function generationThinkingSteps(): AIThinkingStep[] {
  return createThinkingSteps([
    { id: 'input_analysis', label: '分析关键词', summary: '正在识别主题、受众和输入边界。' },
    { id: 'knowledge_retrieval', label: '检索知识库', summary: '正在查找可用于生成的知识片段。' },
    { id: 'content_plan', label: '规划内容结构', summary: '正在组织文章结构和发布格式。' },
    { id: 'draft_generation', label: '生成草稿', summary: '正在生成可审校的内容草稿。' },
    { id: 'quality_check', label: '校验发布边界', summary: '正在检查事实、格式和风险提示。' },
    { id: 'persist_draft', label: '保存草稿', summary: '正在写入内容库，等待人工审核。' },
  ]);
}

export function createFormattingThinkingRunner(setThinking: Dispatch<SetStateAction<AIThinkingState>>): RunFormattingThinking {
  return async ({
    subtitle = '正在格式化输入内容',
    request,
    onSuccess,
    onFailure,
    durationMs = defaultThinkingTraceDurationMs,
  }) => {
    const steps = formattingThinkingSteps();
    const stepDurations = thinkingStepDurations(steps.length, durationMs);
    const title = 'Thinking';
    setThinking({
      open: true,
      blocking: true,
      title,
      subtitle,
      steps: activateThinkingStep(steps, 0),
      trace: null,
    });

    const resultPromise = Promise.resolve()
      .then(request)
      .then((response) => ({ ok: true as const, response }))
      .catch((err: unknown) => ({ ok: false as const, err }));

    for (let index = 0; index < steps.length; index += 1) {
      setThinking((current) => ({
        ...current,
        open: true,
        blocking: true,
        title,
        subtitle,
        steps: activateThinkingStep(steps, index),
        trace: null,
      }));
      await wait(stepDurations[index] ?? 0);
    }

    const result = await resultPromise;
    if (result.ok) {
      const warning = result.response.fallback ? `真实 AI 不可用，已使用 mock 降级：${result.response.fallbackError || 'provider failed'}` : undefined;
      setThinking((current) => ({
        ...current,
        open: true,
        blocking: false,
        title,
        subtitle,
        steps: completeThinkingSteps(formattingThinkingSteps(result.response.fallback), warning),
        trace: null,
      }));
      onSuccess(result.response);
      return result.response;
    }

    const message = result.err instanceof Error ? result.err.message : '格式化失败';
    setThinking((current) => ({
      ...current,
      open: true,
      blocking: false,
      title,
      subtitle,
      steps: failCurrentThinkingStep(current.steps.length > 0 ? current.steps : steps, message),
      trace: null,
    }));
    onFailure?.(message);
    return null;
  };
}

export function failThinkingStep(setThinking: Dispatch<SetStateAction<AIThinkingState>>, message: string) {
  setThinking((current) => ({
    ...current,
    open: true,
    blocking: false,
    steps: current.steps.map((step) => (step.status === 'running' ? { ...step, status: 'failed', warnings: [message] } : step)),
  }));
}

export function traceStepsToThinkingSteps(steps: GenerationTrace['steps']): AIThinkingStep[] {
  return steps.map((step) => ({
    id: step.id,
    label: step.label,
    status: normalizeThinkingStatus(step.status),
    summary: step.summary,
    details: step.details,
    warnings: step.warnings,
  }));
}

export function currentRunningStep(steps: AIThinkingStep[]) {
  return steps.find((step) => step.status === 'running') ?? steps.find((step) => step.status === 'pending') ?? null;
}

export function traceStepStatusLabel(status: string) {
  if (status === 'running') {
    return '执行中';
  }
  if (status === 'pending') {
    return '等待中';
  }
  if (status === 'succeeded') {
    return '完成';
  }
  if (status === 'failed') {
    return '失败';
  }
  if (status === 'skipped') {
    return '跳过';
  }
  return status;
}

export function traceStepStatusColor(status: string): 'default' | 'success' | 'error' | 'warning' | 'info' {
  if (status === 'running') {
    return 'info';
  }
  if (status === 'succeeded') {
    return 'success';
  }
  if (status === 'failed') {
    return 'error';
  }
  if (status === 'skipped') {
    return 'warning';
  }
  return 'default';
}

function formattingThinkingSteps(fallback?: boolean): AIThinkingStep[] {
  return createThinkingSteps([
    { id: 'read_input', label: '读取输入', summary: '正在读取用户提供的关键词和素材。' },
    { id: 'extract_theme', label: '识别核心主题', summary: '正在提取可复用的主题词和素材点。' },
    { id: 'organize_goal', label: '整理生成目标', summary: '正在把零散输入整理成清晰目标。' },
    { id: 'build_markdown', label: '生成 Markdown', summary: '正在输出结构化提示词。' },
    { id: 'check_boundary', label: '校验事实边界', summary: '正在保留事实范围和待补充信息。' },
    { id: 'write_back', label: '回填内容栏', summary: fallback ? '真实 AI 不可用，已使用 mock 降级并回填。' : '格式化结果已回填内容栏。' },
  ]);
}

function createThinkingSteps(items: Array<{ id: string; label: string; summary: string; details?: string[] }>): AIThinkingStep[] {
  return items.map((item, index) => ({
    id: item.id,
    label: item.label,
    status: index === 0 ? 'running' : 'pending',
    summary: item.summary,
    details: item.details ?? [],
    warnings: [],
  }));
}

function completeThinkingSteps(steps: AIThinkingStep[], warning?: string): AIThinkingStep[] {
  return steps.map((step, index) => ({
    ...step,
    status: 'succeeded',
    warnings: warning && index === steps.length - 1 ? [warning] : step.warnings,
  }));
}

function thinkingStepDurations(stepCount: number, totalDurationMs = defaultThinkingTraceDurationMs): number[] {
  if (stepCount <= 0) {
    return [];
  }
  const baseDuration = Math.floor(totalDurationMs / stepCount);
  const remainder = totalDurationMs - baseDuration * stepCount;
  return Array.from({ length: stepCount }, (_, index) => baseDuration + (index < remainder ? 1 : 0));
}

function activateThinkingStep(steps: AIThinkingStep[], activeIndex: number): AIThinkingStep[] {
  return steps.map((step, index) => ({
    ...step,
    status: index < activeIndex ? 'succeeded' : index === activeIndex ? 'running' : 'pending',
    warnings: [],
  }));
}

function failCurrentThinkingStep(steps: AIThinkingStep[], message: string): AIThinkingStep[] {
  if (steps.length === 0) {
    return steps;
  }
  const runningIndex = steps.findIndex((step) => step.status === 'running');
  const failedIndex = runningIndex >= 0 ? runningIndex : steps.length - 1;
  return steps.map((step, index) => ({
    ...step,
    status: index === failedIndex ? 'failed' : step.status,
    warnings: index === failedIndex ? [message] : step.warnings,
  }));
}

function wait(ms: number) {
  return new Promise<void>((resolve) => {
    globalThis.setTimeout(resolve, ms);
  });
}

function normalizeThinkingStatus(status: string): AIThinkingStepStatus {
  if (status === 'succeeded' || status === 'failed' || status === 'skipped' || status === 'running' || status === 'pending') {
    return status;
  }
  return 'pending';
}

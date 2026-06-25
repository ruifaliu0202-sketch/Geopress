export type WorkflowStepStatus = 'pending' | 'running' | 'succeeded' | 'failed' | 'skipped';

export type WorkflowStep = {
  id: string;
  label: string;
  status: WorkflowStepStatus;
  summary: string;
  details: string[];
  warnings: string[];
};

export type WorkflowState = {
  open: boolean;
  blocking: boolean;
  title: string;
  subtitle: string;
  steps: WorkflowStep[];
  warnings?: string[];
};

export const initialWorkflowState: WorkflowState = {
  open: false,
  blocking: false,
  title: '',
  subtitle: '',
  steps: [],
  warnings: [],
};

export function createWorkflowSteps(items: Array<{ id: string; label: string; summary: string; details?: string[] }>): WorkflowStep[] {
  return items.map((item, index) => ({
    id: item.id,
    label: item.label,
    status: index === 0 ? 'running' : 'pending',
    summary: item.summary,
    details: item.details ?? [],
    warnings: [],
  }));
}

export function completeWorkflowSteps(steps: WorkflowStep[], warning?: string): WorkflowStep[] {
  return steps.map((step, index) => ({
    ...step,
    status: 'succeeded',
    warnings: warning && index === steps.length - 1 ? [warning] : step.warnings,
  }));
}

export function activateWorkflowStep(steps: WorkflowStep[], activeIndex: number): WorkflowStep[] {
  return steps.map((step, index) => ({
    ...step,
    status: index < activeIndex ? 'succeeded' : index === activeIndex ? 'running' : 'pending',
    warnings: [],
  }));
}

export function failCurrentWorkflowStep(steps: WorkflowStep[], message: string): WorkflowStep[] {
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

export function currentWorkflowStep(steps: WorkflowStep[]) {
  return steps.find((step) => step.status === 'running') ?? steps.find((step) => step.status === 'pending') ?? null;
}

export function workflowStepStatusLabel(status: string) {
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

export function workflowStepStatusColor(status: string): 'default' | 'success' | 'error' | 'warning' | 'info' {
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

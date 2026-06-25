import type { ReactNode } from 'react';
import type { SxProps, Theme } from '@mui/material/styles';
import type { User, Workspace } from '../../types';

export type WorkspaceAssistantActionId =
  | 'generateContent'
  | 'createKnowledgeBase'
  | 'createKnowledgeAsset'
  | 'bindMediaAccount'
  | 'createSchedule'
  | 'openOnboardingGuide'
  | 'refreshWorkspace';

export type WorkspaceAssistantActionTone = 'primary' | 'neutral' | 'success' | 'warning';

export type WorkspaceAssistantActionContext = {
  workspace: Workspace | null;
  user: User | null;
};

export type WorkspaceAssistantActionHandler = (
  context: WorkspaceAssistantActionContext,
) => void | Promise<void>;

export type WorkspaceAssistantActionDescriptor = {
  id: WorkspaceAssistantActionId;
  label: string;
  shortLabel?: string;
  description: string;
  helper?: string;
  icon: ReactNode;
  tone?: WorkspaceAssistantActionTone;
  disabled?: boolean;
  disabledReason?: string;
  onRun?: WorkspaceAssistantActionHandler;
  dataTourId?: string;
};

export type WorkspaceAssistantActionCallbacks = Partial<
  Record<WorkspaceAssistantActionId, WorkspaceAssistantActionHandler>
>;

export type WorkspaceAssistantPersonaAsset = {
  kind: 'image' | 'component';
  src?: string;
  alt: string;
  node?: ReactNode;
  sx?: SxProps<Theme>;
};

export type WorkspaceAssistantPersona = {
  id: string;
  name: string;
  title: string;
  greeting: string;
  status: string;
  asset: WorkspaceAssistantPersonaAsset;
};

export type WorkspaceAssistantState = {
  loading?: boolean;
  error?: string | null;
  online?: boolean;
};

export type WorkspaceAssistantProps = {
  workspace: Workspace | null;
  user: User | null;
  actions?: WorkspaceAssistantActionDescriptor[];
  actionCallbacks?: WorkspaceAssistantActionCallbacks;
  persona?: WorkspaceAssistantPersona;
  state?: WorkspaceAssistantState;
  defaultOpen?: boolean;
  open?: boolean;
  onOpenChange?: (open: boolean) => void;
  onActionRun?: (actionId: WorkspaceAssistantActionId) => void;
  anchor?: {
    vertical?: 'top' | 'bottom';
    horizontal?: 'left' | 'right';
  };
};

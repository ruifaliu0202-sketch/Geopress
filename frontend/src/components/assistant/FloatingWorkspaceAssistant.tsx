import { useMemo, useState } from 'react';
import {
  Avatar,
  Badge,
  Box,
  Button,
  Chip,
  CircularProgress,
  Collapse,
  Divider,
  IconButton,
  Paper,
  Stack,
  Tooltip,
  Typography,
} from '@mui/material';
import AutoAwesomeOutlinedIcon from '@mui/icons-material/AutoAwesomeOutlined';
import CloseIcon from '@mui/icons-material/Close';
import ExpandLessIcon from '@mui/icons-material/ExpandLess';
import SmartToyOutlinedIcon from '@mui/icons-material/SmartToyOutlined';
import { formatSubscription } from '../../utils/formatters';
import { createWorkspaceAssistantActions } from './actions';
import { defaultCorgiAssistantPersona } from './personas';
import type {
  WorkspaceAssistantActionDescriptor,
  WorkspaceAssistantActionTone,
  WorkspaceAssistantProps,
} from './types';

const toneColor: Record<WorkspaceAssistantActionTone, 'primary' | 'inherit' | 'success' | 'warning'> = {
  primary: 'primary',
  neutral: 'inherit',
  success: 'success',
  warning: 'warning',
};

function actionBorderColor(tone: WorkspaceAssistantActionTone | undefined) {
  if (tone === 'primary') {
    return 'primary.main';
  }
  if (tone === 'success') {
    return 'success.main';
  }
  if (tone === 'warning') {
    return 'warning.main';
  }
  return 'divider';
}

function AssistantPersonaAvatar({ persona }: Pick<WorkspaceAssistantProps, 'persona'>) {
  const activePersona = persona ?? defaultCorgiAssistantPersona;

  if (activePersona.asset.kind === 'image' && activePersona.asset.src) {
    return (
      <Avatar
        src={activePersona.asset.src}
        alt={activePersona.asset.alt}
        imgProps={{ draggable: false }}
        sx={{
          width: 74,
          height: 74,
          bgcolor: 'background.paper',
          border: '1px solid',
          borderColor: 'divider',
          boxShadow: '0 10px 24px rgba(15, 23, 42, 0.16)',
          ...activePersona.asset.sx,
        }}
      />
    );
  }

  return (
    <Avatar
      aria-label={activePersona.asset.alt}
      sx={{
        width: 74,
        height: 74,
        bgcolor: 'background.paper',
        border: '1px solid',
        borderColor: 'divider',
        boxShadow: '0 10px 24px rgba(15, 23, 42, 0.16)',
        ...activePersona.asset.sx,
      }}
    >
      {activePersona.asset.node ?? <SmartToyOutlinedIcon />}
    </Avatar>
  );
}

function AssistantActionButton({
  action,
  onRun,
}: {
  action: WorkspaceAssistantActionDescriptor;
  onRun: (action: WorkspaceAssistantActionDescriptor) => void;
}) {
  const tone = action.tone ?? 'neutral';
  const button = (
    <Button
      variant={tone === 'primary' ? 'contained' : 'outlined'}
      color={toneColor[tone]}
      startIcon={action.icon}
      onClick={() => onRun(action)}
      disabled={action.disabled}
      data-tour-id={action.dataTourId}
      sx={{
        justifyContent: 'flex-start',
        minHeight: 56,
        px: 1.5,
        py: 1,
        borderColor: actionBorderColor(action.tone),
        textAlign: 'left',
        '& .MuiButton-startIcon': {
          flex: '0 0 auto',
        },
      }}
      fullWidth
    >
      <Stack spacing={0.25} sx={{ minWidth: 0, alignItems: 'flex-start' }}>
        <Typography component="span" fontWeight={800} sx={{ lineHeight: 1.2, overflowWrap: 'anywhere' }}>
          {action.label}
        </Typography>
        <Typography
          component="span"
          variant="caption"
          color={tone === 'primary' ? 'primary.contrastText' : 'text.secondary'}
          sx={{ lineHeight: 1.25, whiteSpace: 'normal' }}
        >
          {action.helper ?? action.description}
        </Typography>
      </Stack>
    </Button>
  );

  if (!action.disabled || !action.disabledReason) {
    return button;
  }

  return (
    <Tooltip title={action.disabledReason} placement="left">
      <span>{button}</span>
    </Tooltip>
  );
}

export function FloatingWorkspaceAssistant({
  workspace,
  user,
  actions,
  actionCallbacks,
  persona = defaultCorgiAssistantPersona,
  state,
  defaultOpen = false,
  open,
  onOpenChange,
  onActionRun,
  anchor = { vertical: 'bottom', horizontal: 'right' },
}: WorkspaceAssistantProps) {
  const [internalOpen, setInternalOpen] = useState(defaultOpen);
  const expanded = open ?? internalOpen;
  const online = state?.online ?? true;
  const actionItems = useMemo(
    () => actions ?? createWorkspaceAssistantActions(actionCallbacks),
    [actionCallbacks, actions],
  );
  const connectedActions = actionItems.filter((action) => !action.disabled).length;

  const setExpanded = (nextOpen: boolean) => {
    if (open === undefined) {
      setInternalOpen(nextOpen);
    }
    onOpenChange?.(nextOpen);
  };

  const runAction = async (action: WorkspaceAssistantActionDescriptor) => {
    if (action.disabled || !action.onRun) {
      return;
    }
    onActionRun?.(action.id);
    await action.onRun({ workspace, user });
  };

  const horizontalOffset = anchor.horizontal === 'left' ? { left: { xs: 16, sm: 24 } } : { right: { xs: 16, sm: 24 } };
  const verticalOffset = anchor.vertical === 'top' ? { top: { xs: 82, sm: 92 } } : { bottom: { xs: 16, sm: 24 } };

  return (
    <Box
      sx={{
        position: 'fixed',
        zIndex: (theme) => theme.zIndex.drawer + 2,
        ...horizontalOffset,
        ...verticalOffset,
        width: { xs: 'calc(100vw - 32px)', sm: expanded ? 376 : 92 },
        maxWidth: { xs: 'calc(100vw - 32px)', sm: 376 },
        pointerEvents: 'none',
      }}
    >
      <Stack spacing={1.25} alignItems={anchor.horizontal === 'left' ? 'flex-start' : 'flex-end'}>
        <Collapse in={expanded} timeout="auto" unmountOnExit sx={{ width: '100%' }}>
          <Paper
            elevation={0}
            role="dialog"
            aria-label={`${persona.title}操作面板`}
            sx={{
              width: '100%',
              pointerEvents: 'auto',
              border: '1px solid',
              borderColor: 'divider',
              borderRadius: 2,
              overflow: 'hidden',
              boxShadow: '0 18px 48px rgba(15, 23, 42, 0.18)',
              bgcolor: 'background.paper',
            }}
          >
            <Stack spacing={0}>
              <Stack direction="row" alignItems="center" spacing={1.5} sx={{ p: 1.5 }}>
                <Badge
                  color={online ? 'success' : 'default'}
                  variant="dot"
                  overlap="circular"
                  anchorOrigin={{ vertical: 'bottom', horizontal: 'right' }}
                >
                  <AssistantPersonaAvatar persona={persona} />
                </Badge>
                <Box sx={{ minWidth: 0, flex: 1 }}>
                  <Stack direction="row" spacing={0.75} alignItems="center" sx={{ mb: 0.5, minWidth: 0 }}>
                    <Typography variant="h3" sx={{ lineHeight: 1.1, overflowWrap: 'anywhere' }}>
                      {persona.title}
                    </Typography>
                    <Chip size="small" label={persona.name} color="secondary" variant="outlined" />
                  </Stack>
                  <Typography color="text.secondary" sx={{ lineHeight: 1.35 }}>
                    {persona.greeting}
                  </Typography>
                </Box>
                <Tooltip title="收起 AI 助手">
                  <IconButton size="small" onClick={() => setExpanded(false)} aria-label="收起 AI 助手">
                    <CloseIcon fontSize="small" />
                  </IconButton>
                </Tooltip>
              </Stack>
              <Divider />
              <Stack spacing={1.25} sx={{ p: 1.5 }}>
                {state?.error && (
                  <Typography color="error.main" sx={{ fontWeight: 700, overflowWrap: 'anywhere' }}>
                    {state.error}
                  </Typography>
                )}
                <Stack direction="row" spacing={1} flexWrap="wrap" useFlexGap>
                  <Chip
                    icon={state?.loading ? <CircularProgress size={14} /> : <AutoAwesomeOutlinedIcon />}
                    label={state?.loading ? '同步中' : persona.status}
                    color={online ? 'primary' : 'default'}
                    variant="outlined"
                    size="small"
                  />
                  <Chip size="small" label={workspace?.name ?? '未选择工作区'} variant="outlined" />
                  <Chip size="small" label={formatSubscription(user)} variant="outlined" />
                </Stack>
                <Box
                  sx={{
                    display: 'grid',
                    gridTemplateColumns: { xs: '1fr', sm: '1fr 1fr' },
                    gap: 1,
                  }}
                >
                  {actionItems.map((action) => (
                    <AssistantActionButton key={action.id} action={action} onRun={runAction} />
                  ))}
                </Box>
                <Typography variant="caption" color="text.secondary">
                  {connectedActions > 0
                    ? `已准备 ${connectedActions} 个工作区动作`
                    : '等待接入工作区动作'}
                </Typography>
              </Stack>
            </Stack>
          </Paper>
        </Collapse>

        <Tooltip title={expanded ? 'AI 助手已展开' : '打开 AI 助手'} placement={anchor.horizontal === 'left' ? 'right' : 'left'}>
          <Button
            variant="contained"
            onClick={() => setExpanded(!expanded)}
            aria-expanded={expanded}
            aria-label={expanded ? '收起 AI 助手' : '打开 AI 助手'}
            startIcon={expanded ? <ExpandLessIcon /> : undefined}
            sx={{
              pointerEvents: 'auto',
              width: expanded ? 'auto' : 82,
              height: expanded ? 42 : 82,
              minWidth: expanded ? 0 : 82,
              borderRadius: expanded ? 2 : '50%',
              px: expanded ? 1.75 : 0,
              boxShadow: '0 14px 32px rgba(37, 99, 235, 0.28)',
              overflow: 'hidden',
            }}
            data-tour-id="floating-ai-assistant"
          >
            {expanded ? 'AI 助手' : <AssistantPersonaAvatar persona={persona} />}
          </Button>
        </Tooltip>
      </Stack>
    </Box>
  );
}

import type { ReactNode } from 'react';
import {
  Accordion,
  AccordionDetails,
  AccordionSummary,
  Alert,
  Backdrop,
  Box,
  Chip,
  CircularProgress,
  Drawer,
  IconButton,
  Paper,
  Stack,
  Typography,
} from '@mui/material';
import CloseIcon from '@mui/icons-material/Close';
import ExpandMoreIcon from '@mui/icons-material/ExpandMore';
import type { WorkflowState } from './workflowModel';
import {
  currentWorkflowStep,
  workflowStepStatusColor,
  workflowStepStatusLabel,
} from './workflowModel';

export function WorkflowDrawer({
  state,
  onClose,
  children,
  emptyText = '执行时会显示当前链路、阶段状态和边界提示。',
}: {
  state: WorkflowState;
  onClose: () => void;
  children?: ReactNode;
  emptyText?: string;
}) {
  const warnings = state.warnings ?? [];
  return (
    <>
      <Backdrop
        open={state.blocking}
        sx={{
          zIndex: (theme) => theme.zIndex.modal + 1,
          bgcolor: 'rgba(17, 24, 39, 0.34)',
          backdropFilter: 'blur(2px)',
        }}
      >
        <Paper elevation={4} sx={{ width: { xs: 'calc(100vw - 32px)', sm: 360 }, p: 2, borderRadius: 1 }}>
          <Stack spacing={1.5} alignItems="center">
            <CircularProgress size={32} />
            <Box sx={{ textAlign: 'center' }}>
              <Typography fontWeight={800}>{state.title}</Typography>
              <Typography variant="body2" color="text.secondary">
                {currentWorkflowStep(state.steps)?.label ?? state.subtitle}
              </Typography>
            </Box>
            <WorkflowScanBar />
          </Stack>
        </Paper>
      </Backdrop>
      <Drawer
        anchor="right"
        open={state.open}
        onClose={state.blocking ? undefined : onClose}
        sx={{ zIndex: (theme) => theme.zIndex.modal + 2 }}
      >
        <Box sx={{ width: { xs: '100vw', sm: 440 }, maxWidth: '100vw', p: 2 }}>
          <Stack spacing={2}>
            <Stack direction="row" alignItems="center" justifyContent="space-between" spacing={1}>
              <Box sx={{ minWidth: 0 }}>
                <Typography variant="h2" sx={{ overflowWrap: 'anywhere' }}>
                  {state.title || 'Workflow'}
                </Typography>
                <Typography variant="body2" color="text.secondary" sx={{ overflowWrap: 'anywhere' }}>
                  {state.subtitle || '等待执行'}
                </Typography>
              </Box>
              <IconButton onClick={onClose} aria-label="关闭工作流" disabled={state.blocking}>
                <CloseIcon />
              </IconButton>
            </Stack>
            {warnings.map((warning) => (
              <Alert key={warning} severity="warning">
                {warning}
              </Alert>
            ))}
            {state.steps.length === 0 ? (
              <Typography color="text.secondary">{emptyText}</Typography>
            ) : (
              <Stack spacing={1}>
                {state.steps.map((step, index) => (
                  <Accordion key={`${step.id}-${index}`} defaultExpanded={index < 2} disableGutters>
                    <AccordionSummary expandIcon={<ExpandMoreIcon />}>
                      <Stack direction="row" spacing={1.2} alignItems="center" sx={{ minWidth: 0 }}>
                        <WorkflowStatusDot status={step.status} />
                        <Typography fontWeight={700} sx={{ overflowWrap: 'anywhere' }}>
                          {step.label}
                        </Typography>
                        <Chip size="small" label={workflowStepStatusLabel(step.status)} color={workflowStepStatusColor(step.status)} />
                      </Stack>
                    </AccordionSummary>
                    <AccordionDetails>
                      <Stack spacing={1}>
                        <Typography variant="body2">{step.summary}</Typography>
                        {step.status === 'running' && <WorkflowScanBar compact />}
                        {step.details.map((item) => (
                          <Typography key={item} variant="body2" color="text.secondary" sx={{ overflowWrap: 'anywhere' }}>
                            {item}
                          </Typography>
                        ))}
                        {step.warnings.map((warning) => (
                          <Alert key={warning} severity="warning">
                            {warning}
                          </Alert>
                        ))}
                      </Stack>
                    </AccordionDetails>
                  </Accordion>
                ))}
              </Stack>
            )}
            {children}
          </Stack>
        </Box>
      </Drawer>
    </>
  );
}

function WorkflowScanBar({ compact = false }: { compact?: boolean }) {
  return (
    <Box sx={{ width: '100%', height: compact ? 3 : 4, borderRadius: 1, overflow: 'hidden', bgcolor: 'action.hover' }}>
      <Box
        sx={{
          width: compact ? '42%' : '45%',
          height: '100%',
          bgcolor: 'primary.main',
          animation: 'geopress-workflow-scan 1.2s ease-in-out infinite',
          '@keyframes geopress-workflow-scan': {
            '0%': { transform: 'translateX(-120%)' },
            '100%': { transform: 'translateX(260%)' },
          },
        }}
      />
    </Box>
  );
}

function WorkflowStatusDot({ status }: { status: string }) {
  const color = status === 'succeeded' ? 'success.main' : status === 'failed' ? 'error.main' : status === 'running' ? 'primary.main' : 'text.disabled';
  return (
    <Box
      sx={{
        width: 12,
        height: 12,
        borderRadius: '50%',
        bgcolor: color,
        flex: '0 0 auto',
        boxShadow: status === 'running' ? '0 0 0 0 rgba(25, 118, 210, 0.5)' : 'none',
        animation: status === 'running' ? 'geopress-workflow-pulse 1.25s ease-out infinite' : 'none',
        '@keyframes geopress-workflow-pulse': {
          '0%': { boxShadow: '0 0 0 0 rgba(25, 118, 210, 0.45)' },
          '100%': { boxShadow: '0 0 0 9px rgba(25, 118, 210, 0)' },
        },
      }}
    />
  );
}

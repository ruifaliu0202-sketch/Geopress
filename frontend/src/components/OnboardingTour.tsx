import { useEffect, useMemo, useState } from 'react';
import type { ReactNode } from 'react';
import {
  Box,
  Button,
  Chip,
  Divider,
  IconButton,
  Paper,
  Portal,
  Stack,
  Typography,
} from '@mui/material';
import ArrowBackIcon from '@mui/icons-material/ArrowBack';
import ArrowForwardIcon from '@mui/icons-material/ArrowForward';
import CloseIcon from '@mui/icons-material/Close';

type TourPlacement = 'top' | 'right' | 'bottom' | 'left' | 'center';

export type OnboardingTourStep = {
  id: string;
  title: string;
  content: ReactNode;
  targetId?: string;
  fallbackTargetId?: string;
  placement?: TourPlacement;
};

type Rect = {
  top: number;
  left: number;
  width: number;
  height: number;
};

const highlightPadding = 8;
const viewportPadding = 16;
const popoverGap = 18;

function clamp(value: number, min: number, max: number) {
  return Math.min(Math.max(value, min), max);
}

function findVisibleTarget(targetId?: string) {
  if (!targetId) {
    return null;
  }
  const candidates = Array.from(document.querySelectorAll<HTMLElement>(`[data-tour-id="${targetId}"]`));
  return (
    candidates.find((element) => {
      const rect = element.getBoundingClientRect();
      const style = window.getComputedStyle(element);
      return rect.width > 0 && rect.height > 0 && style.display !== 'none' && style.visibility !== 'hidden';
    }) ?? null
  );
}

function getPaddedRect(element: HTMLElement): Rect {
  const rect = element.getBoundingClientRect();
  return {
    top: Math.max(viewportPadding, rect.top - highlightPadding),
    left: Math.max(viewportPadding, rect.left - highlightPadding),
    width: rect.width + highlightPadding * 2,
    height: rect.height + highlightPadding * 2,
  };
}

function getPopoverPosition(rect: Rect | null, placement: TourPlacement = 'bottom') {
  const viewportWidth = window.innerWidth;
  const viewportHeight = window.innerHeight;
  const width = Math.min(460, viewportWidth - viewportPadding * 2);
  const maxLeft = viewportWidth - width - viewportPadding;
  const centerLeft = (viewportWidth - width) / 2;

  if (!rect || placement === 'center') {
    return {
      width,
      left: centerLeft,
      top: Math.max(viewportPadding, viewportHeight * 0.18),
    };
  }

  const preferredTop = {
    top: rect.top - popoverGap,
    right: rect.top + rect.height / 2,
    bottom: rect.top + rect.height + popoverGap,
    left: rect.top + rect.height / 2,
    center: viewportHeight * 0.18,
  }[placement];
  const preferredLeft = {
    top: rect.left + rect.width / 2 - width / 2,
    right: rect.left + rect.width + popoverGap,
    bottom: rect.left + rect.width / 2 - width / 2,
    left: rect.left - width - popoverGap,
    center: centerLeft,
  }[placement];

  const left = clamp(preferredLeft, viewportPadding, Math.max(viewportPadding, maxLeft));
  const top = placement === 'top'
    ? clamp(preferredTop - 260, viewportPadding, viewportHeight - viewportPadding - 260)
    : clamp(preferredTop, viewportPadding, viewportHeight - viewportPadding - 260);

  return { width, left, top };
}

export function OnboardingTour({
  open,
  steps,
  stepIndex,
  onStepChange,
  onClose,
  onFinish,
}: {
  open: boolean;
  steps: OnboardingTourStep[];
  stepIndex: number;
  onStepChange: (stepIndex: number) => void;
  onClose: () => void;
  onFinish: () => void;
}) {
  const step = steps[Math.min(stepIndex, Math.max(steps.length - 1, 0))];
  const [targetRect, setTargetRect] = useState<Rect | null>(null);
  const isLast = stepIndex >= steps.length - 1;
  const popoverPosition = useMemo(() => getPopoverPosition(targetRect, step?.placement), [step?.placement, targetRect]);

  useEffect(() => {
    if (!open || !step) {
      return undefined;
    }

    let updateTimer = 0;

    const updateTarget = () => {
      const target = findVisibleTarget(step.targetId) ?? findVisibleTarget(step.fallbackTargetId);
      setTargetRect(target ? getPaddedRect(target) : null);
    };

    const scrollToTarget = () => {
      const target = findVisibleTarget(step.targetId) ?? findVisibleTarget(step.fallbackTargetId);
      if (target) {
        target.scrollIntoView({ block: 'center', inline: 'center', behavior: 'smooth' });
      }
      updateTarget();
      updateTimer = window.setTimeout(updateTarget, 220);
    };

    const frame = window.requestAnimationFrame(scrollToTarget);
    window.addEventListener('resize', updateTarget);
    window.addEventListener('scroll', updateTarget, true);

    return () => {
      window.cancelAnimationFrame(frame);
      window.clearTimeout(updateTimer);
      window.removeEventListener('resize', updateTarget);
      window.removeEventListener('scroll', updateTarget, true);
    };
  }, [open, step]);

  useEffect(() => {
    if (!open) {
      return undefined;
    }

    const handleKeyDown = (event: KeyboardEvent) => {
      if (event.key === 'Escape') {
        event.preventDefault();
        onClose();
        return;
      }
      if (event.key === 'ArrowLeft') {
        event.preventDefault();
        onStepChange(Math.max(0, stepIndex - 1));
        return;
      }
      if (event.key === 'Enter' || event.key === 'ArrowRight') {
        event.preventDefault();
        if (isLast) {
          onFinish();
        } else {
          onStepChange(stepIndex + 1);
        }
      }
    };

    window.addEventListener('keydown', handleKeyDown);
    return () => window.removeEventListener('keydown', handleKeyDown);
  }, [isLast, onClose, onFinish, onStepChange, open, stepIndex]);

  if (!open || !step) {
    return null;
  }

  return (
    <Portal>
      <Box
        sx={{
          position: 'fixed',
          inset: 0,
          zIndex: (theme) => theme.zIndex.modal + 20,
          pointerEvents: 'none',
        }}
      >
        {targetRect ? (
          <Box
            sx={{
              position: 'fixed',
              top: targetRect.top,
              left: targetRect.left,
              width: targetRect.width,
              height: targetRect.height,
              borderRadius: 1.5,
              boxShadow: '0 0 0 9999px rgba(15, 23, 42, 0.62), 0 0 0 2px #14b8a6',
              transition: 'top 180ms ease, left 180ms ease, width 180ms ease, height 180ms ease',
            }}
          />
        ) : (
          <Box sx={{ position: 'fixed', inset: 0, bgcolor: 'rgba(15, 23, 42, 0.62)' }} />
        )}

        <Paper
          elevation={10}
          role="dialog"
          aria-modal="true"
          aria-labelledby="workspace-tour-title"
          sx={{
            position: 'fixed',
            top: popoverPosition.top,
            left: popoverPosition.left,
            width: popoverPosition.width,
            maxWidth: 'calc(100vw - 32px)',
            borderRadius: 2,
            overflow: 'hidden',
            pointerEvents: 'auto',
            transition: 'top 180ms ease, left 180ms ease',
          }}
        >
          <Stack spacing={0}>
            <Stack direction="row" alignItems="flex-start" justifyContent="space-between" spacing={2} sx={{ p: 3, pb: 1.5 }}>
              <Box>
                <Typography id="workspace-tour-title" variant="h2">
                  {step.title}
                </Typography>
              </Box>
              <IconButton size="small" onClick={onClose} aria-label="退出教学引导">
                <CloseIcon fontSize="small" />
              </IconButton>
            </Stack>

            <Box sx={{ px: 3, pb: 2 }}>
              <Typography component="div" color="text.secondary">
                {step.content}
              </Typography>
            </Box>

            <Divider />

            <Stack
              direction={{ xs: 'column', sm: 'row' }}
              alignItems={{ xs: 'stretch', sm: 'center' }}
              justifyContent="space-between"
              spacing={1.5}
              sx={{ p: 2 }}
            >
              <Stack direction="row" spacing={1} alignItems="center">
                <Typography variant="body2" color="text.secondary">
                  {stepIndex + 1} / {steps.length}
                </Typography>
                <Chip size="small" variant="outlined" label="Enter 下一步" />
                <Chip size="small" variant="outlined" label="ESC 退出" />
              </Stack>

              <Stack direction="row" spacing={1} justifyContent="flex-end">
                <Button
                  startIcon={<ArrowBackIcon />}
                  disabled={stepIndex === 0}
                  onClick={() => onStepChange(Math.max(0, stepIndex - 1))}
                >
                  上一步
                </Button>
                <Button
                  endIcon={isLast ? undefined : <ArrowForwardIcon />}
                  variant="contained"
                  onClick={() => {
                    if (isLast) {
                      onFinish();
                    } else {
                      onStepChange(stepIndex + 1);
                    }
                  }}
                >
                  {isLast ? '完成' : '下一步'}
                </Button>
              </Stack>
            </Stack>
          </Stack>
        </Paper>
      </Box>
    </Portal>
  );
}

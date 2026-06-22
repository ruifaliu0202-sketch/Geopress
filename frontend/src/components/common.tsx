import type { ReactNode } from 'react';
import {
  Alert,
  Box,
  Button,
  Dialog,
  DialogActions,
  DialogContent,
  DialogTitle,
  Divider,
  FormControl,
  Grid,
  InputLabel,
  MenuItem,
  Paper,
  Select,
  Stack,
  Typography,
} from '@mui/material';
import type { ButtonProps, PaperProps } from '@mui/material';
import { alpha } from '@mui/material/styles';
import type { SxProps, Theme } from '@mui/material/styles';
import { productThemeTokens } from '../theme';

export type DialogBaseProps = {
  open: boolean;
  token: string;
  workspaceId: string;
  onClose: () => void;
  onCreated: () => void;
};

export type ProductSurfaceTone = 'default' | 'cream' | 'sage' | 'white';
export type ProductSurfaceDimension = 'flat' | 'subtle' | 'raised';

export type ProductSurfaceProps = Omit<PaperProps, 'variant'> & {
  tone?: ProductSurfaceTone;
  dimension?: ProductSurfaceDimension;
  interactive?: boolean;
  padded?: boolean;
};

export type HighlightedActionButtonProps = ButtonProps & {
  animateHighlight?: boolean;
};

const { colors, shadows } = productThemeTokens;

function sxArray(sx?: SxProps<Theme>) {
  return Array.isArray(sx) ? sx : sx ? [sx] : [];
}

function productSurfaceSx({
  tone,
  dimension,
  interactive,
  padded,
}: Required<Pick<ProductSurfaceProps, 'tone' | 'dimension' | 'interactive' | 'padded'>>): SxProps<Theme> {
  return (theme) => {
    const backgroundByTone: Record<ProductSurfaceTone, string> = {
      default: theme.palette.background.paper,
      cream: colors.creamPaper,
      sage: alpha(colors.sage, 0.16),
      white: colors.creamRaised,
    };
    const borderByTone: Record<ProductSurfaceTone, string> = {
      default: theme.palette.divider,
      cream: alpha(colors.sageDeep, 0.14),
      sage: alpha(colors.sageDeep, 0.18),
      white: alpha(colors.sageDeep, 0.12),
    };
    const shadowByDimension: Record<ProductSurfaceDimension, string> = {
      flat: 'none',
      subtle: shadows.surface,
      raised: shadows.raised,
    };

    return {
      border: '1px solid',
      borderColor: borderByTone[tone],
      borderRadius: 2,
      backgroundColor: backgroundByTone[tone],
      backgroundImage:
        tone === 'sage'
          ? `linear-gradient(180deg, ${alpha(colors.sage, 0.2)} 0%, ${alpha(colors.creamPaper, 0.86)} 100%)`
          : 'none',
      boxShadow: shadowByDimension[dimension],
      ...(padded ? { p: { xs: 2, md: 2.5 } } : {}),
      ...(interactive
        ? {
            transition: theme.transitions.create(['border-color', 'box-shadow', 'transform'], {
              duration: theme.transitions.duration.shorter,
            }),
            '&:hover': {
              borderColor: alpha(colors.sageDeep, 0.3),
              boxShadow: dimension === 'raised' ? shadows.raised : shadows.surface,
              transform: 'translateY(-1px)',
            },
            '@media (prefers-reduced-motion: reduce)': {
              transition: 'border-color 120ms ease, box-shadow 120ms ease',
              '&:hover': {
                transform: 'none',
              },
            },
          }
        : {}),
    };
  };
}

function highlightedActionButtonSx(animateHighlight: boolean): SxProps<Theme> {
  return (theme) => ({
    '@keyframes geopressActionSweep': {
      '0%': {
        transform: 'translateX(-150%) skewX(-18deg)',
        opacity: 0,
      },
      '18%': {
        opacity: 0.85,
      },
      '48%, 100%': {
        transform: 'translateX(155%) skewX(-18deg)',
        opacity: 0,
      },
    },
    position: 'relative',
    isolation: 'isolate',
    overflow: 'hidden',
    minHeight: 44,
    px: 2.4,
    border: '1px solid',
    borderColor: alpha(colors.creamRaised, 0.42),
    color: colors.creamPaper,
    background: `linear-gradient(135deg, ${colors.sageDeep} 0%, #54704A 100%)`,
    boxShadow: shadows.action,
    '&:hover': {
      background: `linear-gradient(135deg, #2A3E33 0%, ${colors.sageText} 100%)`,
      boxShadow: '0 18px 38px rgba(52, 78, 65, 0.26)',
    },
    '&.Mui-focusVisible': {
      outline: `3px solid ${alpha(colors.sage, 0.45)}`,
      outlineOffset: 2,
    },
    '&.Mui-disabled': {
      borderColor: alpha(colors.sageDeep, 0.12),
      background: alpha(colors.sage, 0.28),
      color: alpha(colors.sageDeep, 0.45),
      boxShadow: 'none',
      '&::after': {
        display: 'none',
      },
    },
    '&::after': animateHighlight
      ? {
          content: '""',
          position: 'absolute',
          zIndex: 0,
          top: '-30%',
          bottom: '-30%',
          left: 0,
          width: '46%',
          background: `linear-gradient(90deg, transparent 0%, ${alpha(colors.creamRaised, 0.62)} 48%, transparent 100%)`,
          animation: 'geopressActionSweep 2.8s ease-in-out infinite',
          pointerEvents: 'none',
        }
      : undefined,
    '& > *': {
      position: 'relative',
      zIndex: 1,
    },
    '@media (prefers-reduced-motion: reduce)': {
      transition: 'background-color 120ms ease, border-color 120ms ease, box-shadow 120ms ease',
      '&::after': {
        animation: 'none',
        opacity: 0,
      },
    },
    [theme.breakpoints.down('sm')]: {
      minHeight: 42,
      px: 2,
    },
  });
}

export function ProductSurface({
  tone = 'default',
  dimension = 'subtle',
  interactive = false,
  padded = false,
  sx,
  children,
  ...paperProps
}: ProductSurfaceProps) {
  return (
    <Paper
      elevation={0}
      {...paperProps}
      sx={[productSurfaceSx({ tone, dimension, interactive, padded }), ...sxArray(sx)]}
    >
      {children}
    </Paper>
  );
}

export function HighlightedActionButton({
  animateHighlight = true,
  variant = 'contained',
  color = 'primary',
  disableElevation = true,
  sx,
  children,
  ...buttonProps
}: HighlightedActionButtonProps) {
  return (
    <Button
      variant={variant}
      color={color}
      disableElevation={disableElevation}
      {...buttonProps}
      sx={[highlightedActionButtonSx(animateHighlight), ...sxArray(sx)]}
    >
      {children}
    </Button>
  );
}

export function FormDialog({
  title,
  open,
  error,
  submitting,
  children,
  onClose,
  onSubmit,
}: {
  title: string;
  open: boolean;
  error: string | null;
  submitting: boolean;
  children: ReactNode;
  onClose: () => void;
  onSubmit: () => void;
}) {
  return (
    <Dialog open={open} onClose={submitting ? undefined : onClose} fullWidth maxWidth="sm">
      <DialogTitle>{title}</DialogTitle>
      <DialogContent>
        <Stack spacing={2} sx={{ pt: 1 }}>
          {error && <Alert severity="error">{error}</Alert>}
          {children}
        </Stack>
      </DialogContent>
      <DialogActions>
        <Button onClick={onClose} disabled={submitting}>
          取消
        </Button>
        <Button onClick={onSubmit} disabled={submitting} variant="contained">
          确认
        </Button>
      </DialogActions>
    </Dialog>
  );
}

export function SelectField({
  label,
  value,
  items,
  onChange,
}: {
  label: string;
  value: string;
  items: Array<{ value: string; label: string }>;
  onChange: (value: string) => void;
}) {
  return (
    <FormControl fullWidth disabled={items.length === 0}>
      <InputLabel>{label}</InputLabel>
      <Select label={label} value={value} onChange={(event) => onChange(String(event.target.value))}>
        {items.map((item) => (
          <MenuItem key={item.value} value={item.value}>
            {item.label}
          </MenuItem>
        ))}
      </Select>
    </FormControl>
  );
}

export function MetricCard({
  label,
  value,
  helper,
  tone = 'primary',
}: {
  label: string;
  value: number;
  helper: string;
  tone?: 'primary' | 'error';
}) {
  return (
    <Grid size={{ xs: 12, sm: 6, lg: 3 }}>
      <ProductSurface padded sx={{ height: '100%', minHeight: 132 }}>
        <Typography variant="body2" color="text.secondary">
          {label}
        </Typography>
        <Typography
          variant="h1"
          color={tone === 'error' ? 'error.main' : 'text.primary'}
          sx={{ mt: 1, overflowWrap: 'anywhere' }}
        >
          {value}
        </Typography>
        <Typography variant="body2" color="text.secondary" sx={{ mt: 1 }}>
          {helper}
        </Typography>
      </ProductSurface>
    </Grid>
  );
}

export function Section({ title, action, children }: { title: string; action?: ReactNode; children: ReactNode }) {
  return (
    <ProductSurface sx={{ overflow: 'hidden', minWidth: 0 }}>
      <Stack
        direction={{ xs: 'column', sm: 'row' }}
        alignItems={{ xs: 'stretch', sm: 'center' }}
        justifyContent="space-between"
        spacing={1.5}
        sx={{ px: { xs: 1.5, sm: 2 }, py: 1.5, minWidth: 0 }}
      >
        <Typography variant="h3" sx={{ minWidth: 0, overflowWrap: 'anywhere' }}>
          {title}
        </Typography>
        {action && (
          <Box
            sx={{
              display: 'flex',
              justifyContent: { xs: 'flex-start', sm: 'flex-end' },
              maxWidth: '100%',
              minWidth: 0,
              '& > .MuiStack-root': {
                flexWrap: 'wrap',
              },
              '& .MuiButton-root': {
                whiteSpace: 'nowrap',
              },
            }}
          >
            {action}
          </Box>
        )}
      </Stack>
      <Divider />
      <Box sx={{ p: { xs: 1.5, sm: 2 }, overflowX: 'auto', WebkitOverflowScrolling: 'touch', minWidth: 0 }}>
        {children}
      </Box>
    </ProductSurface>
  );
}

export function InfoRow({ label, value }: { label: string; value: string }) {
  return (
    <Stack
      direction={{ xs: 'column', sm: 'row' }}
      justifyContent="space-between"
      alignItems={{ xs: 'flex-start', sm: 'center' }}
      spacing={{ xs: 0.25, sm: 2 }}
      sx={{ py: 1.25, minWidth: 0 }}
    >
      <Typography color="text.secondary" sx={{ flexShrink: 0 }}>
        {label}
      </Typography>
      <Typography fontWeight={700} sx={{ textAlign: { xs: 'left', sm: 'right' }, overflowWrap: 'anywhere', minWidth: 0 }}>
        {value}
      </Typography>
    </Stack>
  );
}

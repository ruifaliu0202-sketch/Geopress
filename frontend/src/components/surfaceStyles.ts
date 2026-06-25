import { alpha } from '@mui/material/styles';
import type { Theme } from '@mui/material/styles';
import { productThemeTokens } from '../theme';

const { shadows } = productThemeTokens;

export function selectedSurfaceSx(selected: boolean) {
  return (theme: Theme) => ({
    borderColor: selected ? theme.palette.primary.main : theme.palette.divider,
    boxShadow: selected ? shadows.raised : shadows.surface,
    transform: selected ? 'translateY(-2px)' : 'none',
    backgroundColor: selected ? alpha(theme.palette.primary.light, 0.16) : theme.palette.background.paper,
    transition: theme.transitions.create(['border-color', 'box-shadow', 'transform', 'background-color'], {
      duration: theme.transitions.duration.shorter,
    }),
    '@media (prefers-reduced-motion: reduce)': {
      transition: 'border-color 120ms ease, box-shadow 120ms ease, background-color 120ms ease',
      transform: 'none',
    },
  });
}

import { alpha, createTheme } from '@mui/material/styles';

export type ThemePreference = 'sage' | 'ocean' | 'plum';

export const themePreferenceStorageKey = 'geopress.themePreference';

export const themePresets: Record<
  ThemePreference,
  {
    label: string;
    primary: string;
    primaryLight: string;
    primaryDark: string;
    secondary: string;
    background: string;
    paper: string;
    text: string;
    textSecondary: string;
    divider: string;
    info: string;
  }
> = {
  sage: {
    label: '鼠尾草绿',
    primary: '#344E41',
    primaryLight: '#9CAF88',
    primaryDark: '#26372D',
    secondary: '#7D946A',
    background: '#FFF8DE',
    paper: '#FFFDF4',
    text: '#344E41',
    textSecondary: '#61705F',
    divider: '#D8DEC9',
    info: '#2F6F73',
  },
  ocean: {
    label: '海盐蓝',
    primary: '#1D4E5F',
    primaryLight: '#8BB7BE',
    primaryDark: '#143541',
    secondary: '#A76E48',
    background: '#F6FAF8',
    paper: '#FFFFFF',
    text: '#1B3A42',
    textSecondary: '#587077',
    divider: '#D4E2E2',
    info: '#2F6F73',
  },
  plum: {
    label: '梅子灰',
    primary: '#5D3E57',
    primaryLight: '#B59BAF',
    primaryDark: '#432C3E',
    secondary: '#5F7D6F',
    background: '#FBF7F4',
    paper: '#FFFFFF',
    text: '#3F3440',
    textSecondary: '#756875',
    divider: '#E3D8DF',
    info: '#426C75',
  },
};

export const productThemeTokens = {
  colors: {
    cream: '#FFF8DE',
    creamPaper: '#FFFDF4',
    creamRaised: '#FFFFFF',
    sage: '#9CAF88',
    sageDeep: '#344E41',
    sageText: '#3A4A3F',
    sageSoft: '#EDF3E6',
    sageBorder: '#D8DEC9',
    tealInfo: '#2F6F73',
  },
  shadows: {
    surface: '0 10px 30px rgba(52, 78, 65, 0.08), 0 1px 2px rgba(52, 78, 65, 0.08)',
    raised: '0 18px 42px rgba(52, 78, 65, 0.12), 0 2px 6px rgba(52, 78, 65, 0.08)',
    action: '0 14px 34px rgba(52, 78, 65, 0.22)',
  },
};

const { colors, shadows } = productThemeTokens;

export function createAppTheme(preference: ThemePreference = 'sage') {
  const preset = themePresets[preference] ?? themePresets.sage;
  return createTheme({
  palette: {
    mode: 'light',
    primary: {
      main: preset.primary,
      light: preset.primaryLight,
      dark: preset.primaryDark,
      contrastText: preset.paper,
    },
    secondary: {
      main: preset.secondary,
      light: alpha(preset.secondary, 0.44),
      dark: preset.secondary,
      contrastText: preset.text,
    },
    success: {
      main: '#357A4A',
    },
    warning: {
      main: '#B07A2A',
    },
    error: {
      main: '#B54A45',
    },
    info: {
      main: preset.info,
    },
    background: {
      default: preset.background,
      paper: preset.paper,
    },
    text: {
      primary: preset.text,
      secondary: preset.textSecondary,
    },
    divider: preset.divider,
  },
  shape: {
    borderRadius: 8,
  },
  typography: {
    fontFamily:
      'Inter, ui-sans-serif, system-ui, -apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif',
    h1: {
      fontSize: '2rem',
      fontWeight: 700,
      letterSpacing: 0,
    },
    h2: {
      fontSize: '1.5rem',
      fontWeight: 700,
      letterSpacing: 0,
    },
    h3: {
      fontSize: '1.125rem',
      fontWeight: 700,
      letterSpacing: 0,
    },
    button: {
      textTransform: 'none',
      fontWeight: 700,
      letterSpacing: 0,
    },
  },
  components: {
    MuiCssBaseline: {
      styleOverrides: {
        body: {
          backgroundColor: preset.background,
          color: preset.text,
        },
        '::selection': {
          backgroundColor: alpha(preset.primaryLight, 0.35),
        },
      },
    },
    MuiAppBar: {
      styleOverrides: {
        root: {
          backgroundColor: preset.primary,
          boxShadow: `0 1px 0 ${alpha(preset.paper, 0.2)}, 0 12px 32px ${alpha(preset.primary, 0.18)}`,
        },
      },
    },
    MuiButton: {
      styleOverrides: {
        root: {
          borderRadius: 8,
          minHeight: 40,
          boxShadow: 'none',
          '&.MuiButton-sizeSmall': {
            minHeight: 34,
          },
        },
        containedPrimary: {
          backgroundColor: preset.primary,
          color: preset.paper,
          '&:hover': {
            backgroundColor: preset.primaryDark,
            boxShadow: shadows.action,
          },
        },
        outlinedPrimary: {
          borderColor: alpha(preset.primary, 0.32),
          color: preset.primary,
          '&:hover': {
            borderColor: alpha(preset.primary, 0.58),
            backgroundColor: alpha(preset.primaryLight, 0.14),
          },
        },
        textPrimary: {
          color: preset.primary,
          '&:hover': {
            backgroundColor: alpha(preset.primaryLight, 0.12),
          },
        },
      },
    },
    MuiCard: {
      styleOverrides: {
        root: {
          borderRadius: 8,
          border: `1px solid ${preset.divider}`,
          backgroundColor: preset.paper,
          backgroundImage: 'none',
          boxShadow: shadows.surface,
        },
      },
    },
    MuiPaper: {
      styleOverrides: {
        root: {
          backgroundImage: 'none',
        },
        elevation1: {
          boxShadow: shadows.surface,
        },
      },
    },
    MuiChip: {
      styleOverrides: {
        root: {
          borderRadius: 6,
          fontWeight: 700,
        },
        colorPrimary: {
          backgroundColor: alpha(preset.primaryLight, 0.28),
          color: preset.primary,
        },
      },
    },
    MuiOutlinedInput: {
      styleOverrides: {
        root: {
          backgroundColor: alpha(colors.creamRaised, 0.72),
          '& .MuiOutlinedInput-notchedOutline': {
            borderColor: alpha(preset.primary, 0.2),
          },
          '&:hover .MuiOutlinedInput-notchedOutline': {
            borderColor: alpha(preset.primary, 0.38),
          },
          '&.Mui-focused .MuiOutlinedInput-notchedOutline': {
            borderColor: preset.primary,
          },
        },
      },
    },
    MuiInputLabel: {
      styleOverrides: {
        root: {
          color: preset.textSecondary,
          '&.Mui-focused': {
            color: preset.primary,
          },
        },
      },
    },
    MuiTableCell: {
      styleOverrides: {
        root: {
          borderBottomColor: preset.divider,
        },
        head: {
          color: preset.text,
          fontWeight: 700,
          backgroundColor: alpha(preset.primaryLight, 0.12),
        },
      },
    },
    MuiLink: {
      styleOverrides: {
        root: {
          color: preset.primary,
        },
      },
    },
  },
});
}

export const theme = createAppTheme('sage');

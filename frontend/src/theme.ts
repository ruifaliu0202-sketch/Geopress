import { alpha, createTheme } from '@mui/material/styles';

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

export const theme = createTheme({
  palette: {
    mode: 'light',
    primary: {
      main: colors.sageDeep,
      light: colors.sage,
      dark: '#26372D',
      contrastText: colors.creamPaper,
    },
    secondary: {
      main: '#7D946A',
      light: '#C8D3B8',
      dark: '#586A4A',
      contrastText: '#1F2C24',
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
      main: colors.tealInfo,
    },
    background: {
      default: colors.cream,
      paper: colors.creamPaper,
    },
    text: {
      primary: colors.sageDeep,
      secondary: '#61705F',
    },
    divider: colors.sageBorder,
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
          backgroundColor: colors.cream,
          color: colors.sageDeep,
        },
        '::selection': {
          backgroundColor: alpha(colors.sage, 0.35),
        },
      },
    },
    MuiAppBar: {
      styleOverrides: {
        root: {
          backgroundColor: colors.sageDeep,
          boxShadow: '0 1px 0 rgba(255, 248, 222, 0.16), 0 12px 32px rgba(52, 78, 65, 0.18)',
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
          backgroundColor: colors.sageDeep,
          color: colors.creamPaper,
          '&:hover': {
            backgroundColor: '#26372D',
            boxShadow: shadows.action,
          },
        },
        outlinedPrimary: {
          borderColor: alpha(colors.sageDeep, 0.32),
          color: colors.sageDeep,
          '&:hover': {
            borderColor: alpha(colors.sageDeep, 0.58),
            backgroundColor: alpha(colors.sage, 0.14),
          },
        },
        textPrimary: {
          color: colors.sageDeep,
          '&:hover': {
            backgroundColor: alpha(colors.sage, 0.12),
          },
        },
      },
    },
    MuiCard: {
      styleOverrides: {
        root: {
          borderRadius: 8,
          border: `1px solid ${colors.sageBorder}`,
          backgroundColor: colors.creamPaper,
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
          backgroundColor: alpha(colors.sage, 0.28),
          color: colors.sageDeep,
        },
      },
    },
    MuiOutlinedInput: {
      styleOverrides: {
        root: {
          backgroundColor: alpha(colors.creamRaised, 0.72),
          '& .MuiOutlinedInput-notchedOutline': {
            borderColor: alpha(colors.sageDeep, 0.2),
          },
          '&:hover .MuiOutlinedInput-notchedOutline': {
            borderColor: alpha(colors.sageDeep, 0.38),
          },
          '&.Mui-focused .MuiOutlinedInput-notchedOutline': {
            borderColor: colors.sageDeep,
          },
        },
      },
    },
    MuiInputLabel: {
      styleOverrides: {
        root: {
          color: '#61705F',
          '&.Mui-focused': {
            color: colors.sageDeep,
          },
        },
      },
    },
    MuiTableCell: {
      styleOverrides: {
        root: {
          borderBottomColor: colors.sageBorder,
        },
        head: {
          color: colors.sageText,
          fontWeight: 700,
          backgroundColor: alpha(colors.sage, 0.12),
        },
      },
    },
    MuiLink: {
      styleOverrides: {
        root: {
          color: colors.sageDeep,
        },
      },
    },
  },
});

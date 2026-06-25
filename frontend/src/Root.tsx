import { useMemo, useState } from 'react';
import { CssBaseline, ThemeProvider } from '@mui/material';
import App from './App';
import {
  createAppTheme,
  themePreferenceStorageKey,
  themePresets,
  type ThemePreference,
} from './theme';

function readThemePreference(): ThemePreference {
  try {
    const value = window.localStorage.getItem(themePreferenceStorageKey) as ThemePreference | null;
    if (value && value in themePresets) {
      return value;
    }
  } catch {
    // Ignore localStorage failures.
  }
  return 'sage';
}

export function Root() {
  const [themePreference, setThemePreference] = useState<ThemePreference>(() => readThemePreference());
  const theme = useMemo(() => createAppTheme(themePreference), [themePreference]);

  const updateThemePreference = (value: ThemePreference) => {
    setThemePreference(value);
    try {
      window.localStorage.setItem(themePreferenceStorageKey, value);
    } catch {
      // Ignore localStorage failures.
    }
  };

  return (
    <ThemeProvider theme={theme}>
      <CssBaseline />
      <App themePreference={themePreference} onThemePreferenceChange={updateThemePreference} />
    </ThemeProvider>
  );
}

import { useCallback, useState } from 'react';

export type Theme = 'day' | 'night';

const STORAGE_KEY = 'mt-theme';

function readTheme(): Theme {
  // The inline script in index.html has already resolved the initial theme.
  return document.documentElement.dataset.theme === 'night' ? 'night' : 'day';
}

/** Day/night ("LAMP") theme switch, remembered per browser. */
export function useTheme() {
  const [theme, setThemeState] = useState<Theme>(readTheme);

  const setTheme = useCallback((next: Theme) => {
    document.documentElement.dataset.theme = next;
    try {
      localStorage.setItem(STORAGE_KEY, next);
    } catch {
      // storage unavailable (private mode): theme still applies for this session
    }
    setThemeState(next);
  }, []);

  return { theme, setTheme };
}

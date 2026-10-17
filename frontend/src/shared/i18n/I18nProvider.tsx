import { useCallback, useEffect, useState, type ReactNode } from 'react';
import { I18nContext, type Lang } from './context';
import { en } from './en';
import { ru, type Messages } from './ru';

const MESSAGES: Record<Lang, Messages> = { ru, en };
const STORAGE_KEY = 'mt-lang';

function initialLang(): Lang {
  try {
    const saved = localStorage.getItem(STORAGE_KEY);
    if (saved === 'ru' || saved === 'en') return saved;
  } catch {
    // storage unavailable (private mode): fall back to the browser language
  }
  return navigator.language.toLowerCase().startsWith('ru') ? 'ru' : 'en';
}

/** UI language (RU/EN), remembered per browser. */
export function I18nProvider({ children }: { children: ReactNode }) {
  const [lang, setLangState] = useState<Lang>(initialLang);

  useEffect(() => {
    document.documentElement.lang = lang;
  }, [lang]);

  const setLang = useCallback((next: Lang) => {
    try {
      localStorage.setItem(STORAGE_KEY, next);
    } catch {
      // storage unavailable: the language still applies for this session
    }
    setLangState(next);
  }, []);

  return (
    <I18nContext.Provider value={{ lang, t: MESSAGES[lang], setLang }}>
      {children}
    </I18nContext.Provider>
  );
}

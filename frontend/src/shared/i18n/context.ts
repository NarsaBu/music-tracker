import { createContext, useContext } from 'react';
import { ru, type Messages } from './ru';

export type Lang = 'ru' | 'en';

export const LANGS: readonly Lang[] = ['ru', 'en'];

export interface I18nValue {
  lang: Lang;
  t: Messages;
  setLang: (lang: Lang) => void;
}

// Without a provider (unit tests, Storybook) the UI is in Russian.
export const I18nContext = createContext<I18nValue>({ lang: 'ru', t: ru, setLang: () => {} });

export function useI18n(): I18nValue {
  return useContext(I18nContext);
}

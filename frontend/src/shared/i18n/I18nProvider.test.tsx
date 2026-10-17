import { act, renderHook } from '@testing-library/react';
import type { ReactNode } from 'react';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { en } from './en';
import { I18nProvider, useI18n } from './index';
import { ru } from './ru';

const wrapper = ({ children }: { children: ReactNode }) => <I18nProvider>{children}</I18nProvider>;

/** Every key of ru exists in en with the same kind of value, and vice versa. */
function shapeOf(v: unknown): unknown {
  if (Array.isArray(v)) return v.length;
  if (v && typeof v === 'object') {
    return Object.fromEntries(Object.entries(v).map(([k, x]) => [k, shapeOf(x)]));
  }
  return typeof v;
}

describe('i18n', () => {
  // Node's own localStorage shadows the jsdom one here: use a plain in-memory store.
  beforeEach(() => {
    const store = new Map<string, string>();
    vi.stubGlobal('localStorage', {
      getItem: (k: string) => store.get(k) ?? null,
      setItem: (k: string, v: string) => store.set(k, v),
    });
  });
  afterEach(() => vi.unstubAllGlobals());

  it('has the same keys in both languages', () => {
    expect(shapeOf(en)).toEqual(shapeOf(ru));
  });

  it('switches the language and remembers it', () => {
    localStorage.setItem('mt-lang', 'ru');
    const { result } = renderHook(() => useI18n(), { wrapper });
    expect(result.current.t.layout.nav.feed).toBe('Лента');

    act(() => result.current.setLang('en'));

    expect(result.current.lang).toBe('en');
    expect(result.current.t.layout.nav.feed).toBe('Feed');
    expect(localStorage.getItem('mt-lang')).toBe('en');
    expect(document.documentElement.lang).toBe('en');
  });
});

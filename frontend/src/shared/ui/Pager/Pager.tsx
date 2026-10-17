import { useState } from 'react';
import { useI18n } from '@/shared/i18n';
import { cx } from '@/shared/lib/cx';
import { pageItems } from '@/shared/lib/pagination';
import { PushButton } from '../PushButton/PushButton';
import styles from './Pager.module.css';

export interface PagerProps {
  /** 0-based. */
  page: number;
  pageCount: number;
  onPageChange: (page: number) => void;
  /** Accessible name of the navigation, e.g. "Страницы картотеки". */
  label: string;
  className?: string;
}

/** Page keys: first/previous, numbered pages with gaps, next/last and "go to page". */
export function Pager({ page, pageCount, onPageChange, label, className }: PagerProps) {
  const { t: all } = useI18n();
  const t = all.common.pager;
  const [jump, setJump] = useState('');
  if (pageCount <= 1) return null;

  const first = page === 0;
  const last = page >= pageCount - 1;
  const go = (p: number) => onPageChange(Math.min(pageCount - 1, Math.max(0, p)));

  return (
    <nav className={cx(styles.pager, className)} aria-label={label}>
      <div className={styles.keys}>
        <PushButton size="sm" disabled={first} onClick={() => go(0)} aria-label={t.first}>
          ⇤
        </PushButton>
        <PushButton size="sm" disabled={first} onClick={() => go(page - 1)} aria-label={t.prev}>
          ←
        </PushButton>
        {pageItems(page, pageCount).map((item, i) =>
          item === 'gap' ? (
            <span key={`gap-${i}`} className={styles.gap} aria-hidden>
              …
            </span>
          ) : (
            <PushButton
              key={item}
              size="sm"
              variant={item === page ? 'primary' : 'secondary'}
              className={styles.page}
              aria-current={item === page ? 'page' : undefined}
              aria-label={t.page(item + 1)}
              onClick={() => go(item)}
            >
              {item + 1}
            </PushButton>
          ),
        )}
        <PushButton size="sm" disabled={last} onClick={() => go(page + 1)} aria-label={t.next}>
          →
        </PushButton>
        <PushButton size="sm" disabled={last} onClick={() => go(pageCount - 1)} aria-label={t.last}>
          ⇥
        </PushButton>
      </div>
      <form
        className={styles.jump}
        // out-of-range numbers are clamped below instead of blocking the submit
        noValidate
        onSubmit={(e) => {
          e.preventDefault();
          const n = Number(jump);
          if (Number.isInteger(n) && n >= 1) go(n - 1);
          setJump('');
        }}
      >
        <input
          className={styles.jumpInput}
          type="number"
          inputMode="numeric"
          min={1}
          max={pageCount}
          aria-label={t.pageNumber}
          placeholder={`${page + 1}/${pageCount}`}
          value={jump}
          onChange={(e) => setJump(e.target.value)}
        />
        <PushButton size="sm" type="submit" disabled={!jump}>
          {t.goTo}
        </PushButton>
      </form>
    </nav>
  );
}

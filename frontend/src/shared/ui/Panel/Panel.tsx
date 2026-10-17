import type { ReactNode } from 'react';
import { cx } from '@/shared/lib/cx';
import styles from './Panel.module.css';

export interface PanelProps {
  /** Engraved caption in the top-left corner. */
  title?: string;
  /** Slot on the right side of the caption row (LEDs, small controls). */
  extra?: ReactNode;
  children: ReactNode;
  className?: string;
}

/** Brushed aluminium faceplate with screws in the corners. */
export function Panel({ title, extra, children, className }: PanelProps) {
  return (
    <section className={cx(styles.panel, className)} aria-label={title}>
      <span className={cx(styles.screw, styles.tl)} aria-hidden />
      <span className={cx(styles.screw, styles.tr)} aria-hidden />
      <span className={cx(styles.screw, styles.bl)} aria-hidden />
      <span className={cx(styles.screw, styles.br)} aria-hidden />
      {(title || extra) && (
        <header className={styles.header}>
          {title && <h2 className={styles.title}>{title}</h2>}
          {extra && <div className={styles.extra}>{extra}</div>}
        </header>
      )}
      <div className={styles.body}>{children}</div>
    </section>
  );
}

import { Children, type ReactNode } from 'react';
import { cx } from '@/shared/lib/cx';
import styles from './RecordShelf.module.css';

export interface RecordShelfProps {
  /** Accessible name of the list. */
  label: string;
  /** Records (usually RecordSleeve); each becomes a list item. */
  children: ReactNode;
  busy?: boolean;
  className?: string;
}

/** Wooden shelf with records standing on it: a responsive grid list. */
export function RecordShelf({ label, children, busy, className }: RecordShelfProps) {
  return (
    <ul className={cx(styles.shelf, className)} aria-label={label} aria-busy={busy || undefined}>
      {Children.map(children, (child) => (
        <li className={styles.slot}>{child}</li>
      ))}
    </ul>
  );
}

import type { ReactNode } from 'react';
import { cx } from '@/shared/lib/cx';
import styles from './LcdDisplay.module.css';

export interface LcdDisplayProps {
  children: ReactNode;
  /** Accessible name, e.g. "Последняя синхронизация". */
  label?: string;
  size?: 'sm' | 'md' | 'lg';
  className?: string;
}

/** Green backlit LCD behind glass, for counters, dates and short statuses. */
export function LcdDisplay({ children, label, size = 'md', className }: LcdDisplayProps) {
  return (
    <output className={cx(styles.lcd, styles[size], className)} aria-label={label}>
      {children}
    </output>
  );
}

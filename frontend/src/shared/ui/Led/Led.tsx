import { cx } from '@/shared/lib/cx';
import styles from './Led.module.css';

export type LedState = 'off' | 'on' | 'ok' | 'warn';

export interface LedProps {
  state: LedState;
  blink?: boolean;
  /**
   * Accessible meaning of the LED. Color is never the only carrier of meaning:
   * without a label the LED must be accompanied by visible text.
   */
  label?: string;
  className?: string;
}

export function Led({ state, blink = false, label, className }: LedProps) {
  return (
    <span
      className={cx(styles.led, styles[state], blink && state !== 'off' && styles.blink, className)}
      role={label ? 'img' : undefined}
      aria-label={label}
      aria-hidden={label ? undefined : true}
    />
  );
}

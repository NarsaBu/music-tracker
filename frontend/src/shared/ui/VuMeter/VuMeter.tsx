import type { CSSProperties } from 'react';
import { cx } from '@/shared/lib/cx';
import styles from './VuMeter.module.css';

export interface VuMeterProps {
  /** 0–100. */
  value: number;
  /** Accessible name, e.g. "Прогресс синхронизации". */
  label: string;
  className?: string;
}

/** Backlit VU meter with a swinging needle; semantically a progress bar. */
export function VuMeter({ value, label, className }: VuMeterProps) {
  const v = Math.min(100, Math.max(0, value));
  const style = { '--needle': `${(v / 100) * 100 - 50}deg` } as CSSProperties;

  return (
    <div className={cx(styles.meter, className)}>
      <progress className="srOnly" max={100} value={v} aria-label={label}>
        {v}%
      </progress>
      <div className={styles.face} style={style} aria-hidden>
        <span className={styles.scale} />
        <span className={styles.redZone} />
        <span className={styles.mark}>VU</span>
        <span className={styles.needle} />
        <span className={styles.pivot} />
      </div>
    </div>
  );
}

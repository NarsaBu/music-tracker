import type { CSSProperties } from 'react';
import { cx } from '@/shared/lib/cx';
import styles from './SignalMeter.module.css';

export interface SignalMeterProps {
  /** 0–100. */
  value: number;
  /** Accessible name, e.g. "Совпадение". */
  label: string;
  className?: string;
}

/** Tuner signal-strength gauge: a needle swinging over an arc scale. */
export function SignalMeter({ value, label, className }: SignalMeterProps) {
  const v = Math.min(100, Math.max(0, value));
  const style = { '--needle': `${(v / 100) * 90 - 45}deg` } as CSSProperties;

  return (
    <span className={cx(styles.root, className)}>
      <meter className="srOnly" min={0} max={100} low={50} high={85} optimum={100} value={v}>
        {v}
      </meter>
      <span className="srOnly">{`${label}: ${v}%`}</span>
      <span className={styles.gauge} style={style} aria-hidden>
        <span className={styles.scale} />
        <span className={styles.needle} />
      </span>
      <span className={styles.value} aria-hidden>
        {v}
      </span>
    </span>
  );
}

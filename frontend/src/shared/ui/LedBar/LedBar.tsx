import { cx } from '@/shared/lib/cx';
import styles from './LedBar.module.css';

export interface LedBarProps {
  /** 0–100. */
  value: number;
  /** Accessible name, e.g. "Прогресс синхронизации". */
  label: string;
  /** Text for screen readers instead of the bare percentage, e.g. "120 из 712". */
  valueText?: string;
  /** Work in progress: the leading segment blinks. */
  active?: boolean;
  segments?: number;
  className?: string;
}

const SCALE = [0, 25, 50, 75, 100];

/** Segmented LED level bar with a printed 0–100 scale; semantically a progress bar. */
export function LedBar({
  value,
  label,
  valueText,
  active = false,
  segments = 32,
  className,
}: LedBarProps) {
  const v = Math.min(100, Math.max(0, Math.round(value)));
  const lit = Math.round((v / 100) * segments);

  return (
    <div className={cx(styles.root, className)}>
      <div
        className={styles.bar}
        role="progressbar"
        aria-label={label}
        aria-valuemin={0}
        aria-valuemax={100}
        aria-valuenow={v}
        aria-valuetext={valueText}
      >
        {Array.from({ length: segments }, (_, i) => (
          <span
            key={i}
            className={cx(
              styles.segment,
              i < lit && styles.lit,
              // the top of the scale is amber, like a level meter near its peak
              i >= segments * 0.85 && styles.peak,
              active && i === lit - 1 && styles.leading,
            )}
          />
        ))}
      </div>
      <div className={styles.scale} aria-hidden>
        {SCALE.map((s) => (
          <span key={s}>{s}</span>
        ))}
      </div>
    </div>
  );
}

import { useId, type CSSProperties } from 'react';
import { cx } from '@/shared/lib/cx';
import { angleOf } from './geometry';
import styles from './RotarySelector.module.css';

export interface RotaryOption<T extends string> {
  value: T;
  label: string;
}

export interface RotarySelectorProps<T extends string> {
  label: string;
  options: ReadonlyArray<RotaryOption<T>>;
  value: T;
  onChange: (value: T) => void;
  disabled?: boolean;
  className?: string;
}

const MAX_OPTIONS = 7;

/**
 * Rotary source selector. Built on a native radio group, so arrow keys,
 * focus and screen readers work out of the box; clicking a mark selects it.
 */
export function RotarySelector<T extends string>({
  label,
  options,
  value,
  onChange,
  disabled,
  className,
}: RotarySelectorProps<T>) {
  const name = useId();
  if (import.meta.env.DEV && options.length > MAX_OPTIONS) {
    console.warn(`RotarySelector supports up to ${MAX_OPTIONS} options`);
  }
  const selected = Math.max(
    0,
    options.findIndex((o) => o.value === value),
  );

  return (
    <fieldset className={cx(styles.root, className)} disabled={disabled}>
      <legend className={styles.legend}>{label}</legend>
      <div className={styles.dial}>
        {options.map((o, i) => (
          <label
            key={o.value}
            className={cx(styles.mark, i === selected && styles.active)}
            style={{ '--angle': `${angleOf(i, options.length)}deg` } as CSSProperties}
          >
            <input
              type="radio"
              className="srOnly"
              name={name}
              value={o.value}
              checked={o.value === value}
              onChange={() => onChange(o.value)}
            />
            <span className={styles.markText}>{o.label}</span>
          </label>
        ))}
        <span
          className={styles.knob}
          style={{ '--angle': `${angleOf(selected, options.length)}deg` } as CSSProperties}
          aria-hidden
        >
          <span className={styles.pointer} />
        </span>
      </div>
    </fieldset>
  );
}

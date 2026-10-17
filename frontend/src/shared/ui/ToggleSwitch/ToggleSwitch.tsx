import { useId } from 'react';
import { cx } from '@/shared/lib/cx';
import { Led } from '../Led/Led';
import styles from './ToggleSwitch.module.css';

export interface ToggleSwitchProps {
  label: string;
  description?: string;
  checked: boolean;
  onChange: (checked: boolean) => void;
  disabled?: boolean;
  /** Shows an LED next to the label that mirrors the state. */
  withLed?: boolean;
  className?: string;
}

/** Bat-handle toggle switch. Built on a native checkbox with role="switch". */
export function ToggleSwitch({
  label,
  description,
  checked,
  onChange,
  disabled,
  withLed = true,
  className,
}: ToggleSwitchProps) {
  const id = useId();
  const descriptionId = description ? `${id}-desc` : undefined;

  return (
    <label className={cx(styles.root, disabled && styles.disabled, className)} htmlFor={id}>
      <input
        id={id}
        type="checkbox"
        role="switch"
        className={styles.input}
        checked={checked}
        disabled={disabled}
        aria-describedby={descriptionId}
        onChange={(e) => onChange(e.target.checked)}
      />
      <span className={styles.switch} aria-hidden>
        <span className={styles.mark}>on</span>
        <span className={styles.nut}>
          <span className={styles.lever} />
        </span>
        <span className={styles.mark}>off</span>
      </span>
      <span className={styles.text}>
        <span className={styles.label}>
          {withLed && <Led state={checked ? 'on' : 'off'} />}
          {label}
        </span>
        {description && (
          <span id={descriptionId} className={styles.description}>
            {description}
          </span>
        )}
      </span>
    </label>
  );
}

import { useId, type SelectHTMLAttributes } from 'react';
import { cx } from '@/shared/lib/cx';
import styles from './EngravedSelect.module.css';

export interface EngravedSelectOption {
  value: string;
  label: string;
}

export interface EngravedSelectProps extends Omit<
  SelectHTMLAttributes<HTMLSelectElement>,
  'id' | 'onChange'
> {
  label: string;
  options: readonly EngravedSelectOption[];
  value: string;
  onChange: (value: string) => void;
}

/** Native select recessed into the faceplate, styled like EngravedInput. */
export function EngravedSelect({
  label,
  options,
  value,
  onChange,
  className,
  ...rest
}: EngravedSelectProps) {
  const id = useId();
  return (
    <div className={cx(styles.field, className)}>
      <label className={styles.label} htmlFor={id}>
        {label}
      </label>
      <select
        id={id}
        className={styles.select}
        value={value}
        onChange={(e) => onChange(e.target.value)}
        {...rest}
      >
        {options.map((o) => (
          <option key={o.value} value={o.value}>
            {o.label}
          </option>
        ))}
      </select>
    </div>
  );
}

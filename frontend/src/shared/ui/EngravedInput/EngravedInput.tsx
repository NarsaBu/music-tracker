import { useId, type InputHTMLAttributes } from 'react';
import { cx } from '@/shared/lib/cx';
import styles from './EngravedInput.module.css';

export interface EngravedInputProps extends Omit<InputHTMLAttributes<HTMLInputElement>, 'id'> {
  label: string;
  hint?: string;
  error?: string;
}

/** Text field recessed into the faceplate, with an engraved caption. */
export function EngravedInput({ label, hint, error, className, ...rest }: EngravedInputProps) {
  const id = useId();
  const messageId = error || hint ? `${id}-msg` : undefined;

  return (
    <div className={cx(styles.field, className)}>
      <label className={styles.label} htmlFor={id}>
        {label}
      </label>
      <input
        id={id}
        className={cx(styles.input, error && styles.invalid)}
        aria-invalid={error ? true : undefined}
        aria-describedby={messageId}
        {...rest}
      />
      {(error || hint) && (
        <span id={messageId} className={error ? styles.error : styles.hint}>
          {error ?? hint}
        </span>
      )}
    </div>
  );
}

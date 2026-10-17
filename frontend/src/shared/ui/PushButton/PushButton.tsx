import type { ButtonHTMLAttributes } from 'react';
import { cx } from '@/shared/lib/cx';
import { Led } from '../Led/Led';
import styles from './PushButton.module.css';

export interface PushButtonProps extends ButtonHTMLAttributes<HTMLButtonElement> {
  /** primary — black plastic cap, secondary — aluminium cap. */
  variant?: 'primary' | 'secondary';
  /** Shows an LED in the cap; true lights it. */
  led?: boolean;
}

/** Mechanical push button: raised at rest, pressed in while active. */
export function PushButton({
  variant = 'secondary',
  led,
  type = 'button',
  className,
  children,
  ...rest
}: PushButtonProps) {
  return (
    <button type={type} className={cx(styles.button, styles[variant], className)} {...rest}>
      {led !== undefined && <Led state={led ? 'on' : 'off'} />}
      <span>{children}</span>
    </button>
  );
}

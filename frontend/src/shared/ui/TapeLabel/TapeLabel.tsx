import type { ReactNode } from 'react';
import { cx } from '@/shared/lib/cx';
import styles from './TapeLabel.module.css';

export interface TapeLabelProps {
  children: ReactNode;
  /** accent — red marker ink, for things that need attention (announcements). */
  tone?: 'plain' | 'accent';
  className?: string;
}

/** Strip of paper tape with a typed label: release type, "feat.", "soon". */
export function TapeLabel({ children, tone = 'plain', className }: TapeLabelProps) {
  return <span className={cx(styles.tape, styles[tone], className)}>{children}</span>;
}

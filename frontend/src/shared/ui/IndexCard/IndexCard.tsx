import type { CSSProperties, ReactNode } from 'react';
import { cx } from '@/shared/lib/cx';
import styles from './IndexCard.module.css';
import { tiltOf } from './tilt';

export interface IndexCardProps {
  title: string;
  /** Short text on the paper tab sticking out above the card (e.g. a country code). */
  tab?: string;
  /** Any stable string (an MBID): the card gets the same slight tilt on every render. */
  tiltSeed?: string;
  /** Slot on the right side of the title row (LEDs). */
  extra?: ReactNode;
  /** Bottom row for actions. */
  footer?: ReactNode;
  children?: ReactNode;
  className?: string;
}

/** Paper index card from a card catalogue drawer. */
export function IndexCard({
  title,
  tab,
  tiltSeed,
  extra,
  footer,
  children,
  className,
}: IndexCardProps) {
  const style = tiltSeed
    ? ({ '--card-tilt': `${tiltOf(tiltSeed)}deg` } as CSSProperties)
    : undefined;

  return (
    <article className={cx(styles.card, className)} style={style}>
      {tab && (
        <span className={styles.tab} aria-hidden>
          {tab}
        </span>
      )}
      <header className={styles.header}>
        <h3 className={styles.title}>{title}</h3>
        {extra && <div className={styles.extra}>{extra}</div>}
      </header>
      {children && <div className={styles.body}>{children}</div>}
      {footer && <footer className={styles.footer}>{footer}</footer>}
    </article>
  );
}

import { useState, type ReactNode } from 'react';
import { cx } from '@/shared/lib/cx';
import styles from './RecordSleeve.module.css';

export interface RecordSleeveProps {
  title: string;
  /** Second line, e.g. the artist credit. */
  subtitle?: string;
  /** Cover image; when missing or broken the sleeve shows a plain printed label. */
  coverUrl?: string;
  /** Small line under the titles, e.g. the release date. */
  meta?: ReactNode;
  /** Tape labels on the sleeve corner. */
  badges?: ReactNode;
  /** Makes the title a link (opens in a new tab). */
  href?: string;
  /** Dimmed sleeve, e.g. for history that is not news. */
  muted?: boolean;
  className?: string;
}

/** Vinyl record in its sleeve; on hover the disc slides out. */
export function RecordSleeve({
  title,
  subtitle,
  coverUrl,
  meta,
  badges,
  href,
  muted,
  className,
}: RecordSleeveProps) {
  const [broken, setBroken] = useState(false);
  const showCover = coverUrl && !broken;

  return (
    <article className={cx(styles.record, muted && styles.muted, className)}>
      <div className={styles.sleeve}>
        <span className={styles.disc} aria-hidden />
        {showCover ? (
          <img
            className={styles.cover}
            src={coverUrl}
            alt=""
            loading="lazy"
            decoding="async"
            onError={() => setBroken(true)}
          />
        ) : (
          <span className={styles.blank} aria-hidden>
            {title}
          </span>
        )}
        {badges && <div className={styles.badges}>{badges}</div>}
      </div>
      <h3 className={styles.title}>
        {href ? (
          <a href={href} target="_blank" rel="noreferrer">
            {title}
          </a>
        ) : (
          title
        )}
      </h3>
      {subtitle && <p className={styles.subtitle}>{subtitle}</p>}
      {meta && <p className={styles.meta}>{meta}</p>}
    </article>
  );
}

import { useId, useState, type DragEvent } from 'react';
import { cx } from '@/shared/lib/cx';
import styles from './CassetteDropzone.module.css';

export interface CassetteDropzoneProps {
  /** Engraved caption above the deck. */
  label: string;
  /** Text on the cassette label, e.g. the chosen file name. */
  tapeLabel?: string;
  hint?: string;
  /** File types for the picker, e.g. ".txt,.csv". */
  accept?: string;
  /** Spins the reels: the tape is playing (uploading). */
  busy?: boolean;
  disabled?: boolean;
  onFile: (file: File) => void;
  className?: string;
}

/**
 * Cassette slot of a tape deck: a native file input (click or keyboard opens the
 * picker) that also accepts a dropped file.
 */
export function CassetteDropzone({
  label,
  tapeLabel,
  hint,
  accept,
  busy = false,
  disabled = false,
  onFile,
  className,
}: CassetteDropzoneProps) {
  const id = useId();
  const captionId = `${id}-caption`;
  const hintId = hint ? `${id}-hint` : undefined;
  const [dragging, setDragging] = useState(false);

  const onDragOver = (e: DragEvent) => {
    if (disabled || !e.dataTransfer.types.includes('Files')) return;
    e.preventDefault();
    e.dataTransfer.dropEffect = 'copy';
    setDragging(true);
  };
  const onDrop = (e: DragEvent) => {
    e.preventDefault();
    setDragging(false);
    const file = e.dataTransfer.files[0];
    if (!disabled && file) onFile(file);
  };

  return (
    <div className={cx(styles.root, className)}>
      <label
        htmlFor={id}
        className={cx(styles.slot, dragging && styles.dragging, disabled && styles.disabled)}
        onDragOver={onDragOver}
        onDragLeave={() => setDragging(false)}
        onDrop={onDrop}
      >
        <span id={captionId} className={styles.caption}>
          {label}
        </span>
        <input
          id={id}
          type="file"
          className="srOnly"
          accept={accept}
          disabled={disabled}
          aria-labelledby={captionId}
          aria-describedby={hintId}
          onChange={(e) => {
            const file = e.target.files?.[0];
            // Reset so that choosing the same file again fires onChange.
            e.target.value = '';
            if (file) onFile(file);
          }}
        />
        <span className={cx(styles.cassette, busy && styles.busy)} aria-hidden>
          <span className={styles.tape}>{tapeLabel ?? 'C-90 · artists'}</span>
          <span className={styles.window}>
            <span className={styles.reel} />
            <span className={styles.reel} />
          </span>
        </span>
      </label>
      {hint && (
        <span id={hintId} className={styles.hint}>
          {hint}
        </span>
      )}
    </div>
  );
}

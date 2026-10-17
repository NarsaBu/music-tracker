const dateTimeFormats = new Map<string, Intl.DateTimeFormat>();

/** "17.10.2026, 06:03" (ru-RU) in the viewer's time zone. */
export function formatDateTime(iso: string, locale: string): string {
  let f = dateTimeFormats.get(locale);
  if (!f) {
    f = new Intl.DateTimeFormat(locale, {
      day: '2-digit',
      month: '2-digit',
      year: 'numeric',
      hour: '2-digit',
      minute: '2-digit',
    });
    dateTimeFormats.set(locale, f);
  }
  return f.format(new Date(iso));
}

/** Digits an LCD counter shows at least; larger numbers widen it. */
export const LCD_MIN_DIGITS = 3;

/** Tape-counter digits for LCDs: "042", "1234" — never cut, at least three digits. */
export function lcdCount(n: number, digits = LCD_MIN_DIGITS): string {
  return String(n).padStart(digits, '0');
}

/**
 * "done/total" counter with both halves as wide as the larger number, so the
 * display does not jump while counting: "042/715", "0042/1200".
 */
export function lcdPair(done: number, total: number): string {
  const digits = Math.max(LCD_MIN_DIGITS, String(done).length, String(total).length);
  return `${lcdCount(done, digits)}/${lcdCount(total, digits)}`;
}

/** Dashes of the minimum width, while a counter has no value yet. */
export const LCD_BLANK = '-'.repeat(LCD_MIN_DIGITS);

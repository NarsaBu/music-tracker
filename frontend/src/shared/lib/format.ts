const dateTime = new Intl.DateTimeFormat('ru-RU', {
  day: '2-digit',
  month: '2-digit',
  year: 'numeric',
  hour: '2-digit',
  minute: '2-digit',
});

/** "17.10.2026, 06:03" in the viewer's time zone. */
export function formatDateTime(iso: string): string {
  return dateTime.format(new Date(iso));
}

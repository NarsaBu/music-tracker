/**
 * Starts a browser download of a same-origin URL served as an attachment: the
 * browser streams it to disk with its own progress, nothing is held in memory.
 */
export function startDownload(url: string, filename: string): void {
  const a = document.createElement('a');
  a.href = url;
  a.download = filename;
  a.rel = 'noopener';
  document.body.append(a);
  a.click();
  a.remove();
}

export type PageItem = number | 'gap';

/**
 * Page buttons to show (0-based): the first and last pages, `around` pages on each
 * side of the current one, and 'gap' for the skipped runs. A gap of a single page
 * is shown as that page, so a gap always hides at least two.
 */
export function pageItems(current: number, count: number, around = 2): PageItem[] {
  const shown = new Set<number>([0, count - 1]);
  for (let p = current - around; p <= current + around; p++) shown.add(p);
  const pages = [...shown].filter((p) => p >= 0 && p < count).sort((a, b) => a - b);

  const out: PageItem[] = [];
  pages.forEach((p, i) => {
    const prev = pages[i - 1];
    if (prev !== undefined && p - prev === 2) out.push(prev + 1);
    else if (prev !== undefined && p - prev > 2) out.push('gap');
    out.push(p);
  });
  return out;
}

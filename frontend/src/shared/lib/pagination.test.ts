import { describe, expect, it } from 'vitest';
import { pageItems } from './pagination';

describe('pageItems', () => {
  it('shows every page of a short list', () => {
    expect(pageItems(0, 1)).toEqual([0]);
    expect(pageItems(2, 5)).toEqual([0, 1, 2, 3, 4]);
  });

  it('keeps the ends and the neighbourhood of the current page', () => {
    expect(pageItems(7, 20)).toEqual([0, 'gap', 5, 6, 7, 8, 9, 'gap', 19]);
    expect(pageItems(0, 20)).toEqual([0, 1, 2, 'gap', 19]);
    expect(pageItems(19, 20)).toEqual([0, 'gap', 17, 18, 19]);
  });

  it('never hides a single page behind a gap', () => {
    expect(pageItems(3, 20)).toEqual([0, 1, 2, 3, 4, 5, 'gap', 19]);
  });
});

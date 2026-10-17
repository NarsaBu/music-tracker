import { describe, expect, it } from 'vitest';
import { lcdCount, lcdPair } from './format';

describe('lcdCount', () => {
  it('pads to three digits and widens for larger numbers', () => {
    expect(lcdCount(7)).toBe('007');
    expect(lcdCount(832)).toBe('832');
    expect(lcdCount(1000)).toBe('1000');
    expect(lcdCount(12345)).toBe('12345');
  });
});

describe('lcdPair', () => {
  it('keeps both halves as wide as the larger number', () => {
    expect(lcdPair(4, 4)).toBe('004/004');
    expect(lcdPair(42, 1200)).toBe('0042/1200');
    expect(lcdPair(10_000, 10_000)).toBe('10000/10000');
  });
});

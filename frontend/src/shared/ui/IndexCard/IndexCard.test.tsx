import { render, screen } from '@testing-library/react';
import { IndexCard } from './IndexCard';
import { tiltOf } from './tilt';

describe('tiltOf', () => {
  it('is deterministic and bounded by ±0.6°', () => {
    const seeds = ['a', 'b', 'f6f2326f-6b25-4170-b89d-e235b25508e8', '', 'Nirvana'];
    for (const s of seeds) {
      expect(tiltOf(s)).toBe(tiltOf(s));
      expect(Math.abs(tiltOf(s))).toBeLessThanOrEqual(0.6);
    }
    expect(new Set(seeds.map(tiltOf)).size).toBeGreaterThan(1);
  });
});

describe('IndexCard', () => {
  it('renders an article with a heading and hides the decorative tab', () => {
    render(<IndexCard title="Sigur Rós" tab="IS" tiltSeed="x" />);
    const card = screen.getByRole('article');
    expect(screen.getByRole('heading', { name: 'Sigur Rós' })).toBeInTheDocument();
    expect(card.style.getPropertyValue('--card-tilt')).toMatch(/deg$/);
    expect(screen.getByText('IS')).toHaveAttribute('aria-hidden', 'true');
  });
});

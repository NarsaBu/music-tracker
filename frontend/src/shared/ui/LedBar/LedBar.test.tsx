import { render, screen } from '@testing-library/react';
import { LedBar } from './LedBar';

describe('LedBar', () => {
  it('is a labelled progress bar that lights segments by value', () => {
    const { container } = render(
      <LedBar value={50.4} label="Прогресс" valueText="1 из 2" segments={10} />,
    );
    const bar = screen.getByRole('progressbar', { name: 'Прогресс' });
    expect(bar).toHaveAttribute('aria-valuenow', '50');
    expect(bar).toHaveAttribute('aria-valuetext', '1 из 2');
    expect(container.querySelectorAll('[class*="lit"]')).toHaveLength(5);
  });

  it('clamps out-of-range values', () => {
    render(<LedBar value={130} label="Прогресс" />);
    expect(screen.getByRole('progressbar')).toHaveAttribute('aria-valuenow', '100');
  });
});

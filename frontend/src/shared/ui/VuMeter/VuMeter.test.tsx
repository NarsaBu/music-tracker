import { render, screen } from '@testing-library/react';
import { VuMeter } from './VuMeter';

describe('VuMeter', () => {
  it('is a labelled progress bar with a clamped value', () => {
    render(<VuMeter value={130} label="Прогресс" />);
    const bar = screen.getByRole('progressbar', { name: 'Прогресс' });
    expect(bar).toHaveAttribute('value', '100');
  });
});

import { render, screen } from '@testing-library/react';
import { SignalMeter } from './SignalMeter';

describe('SignalMeter', () => {
  it('exposes the value as a meter with a text alternative', () => {
    render(<SignalMeter value={87} label="Совпадение" />);
    expect(screen.getByRole('meter')).toHaveValue(87);
    expect(screen.getByText('Совпадение: 87%')).toBeInTheDocument();
  });

  it('clamps out-of-range values', () => {
    render(<SignalMeter value={140} label="Совпадение" />);
    expect(screen.getByRole('meter')).toHaveValue(100);
  });
});

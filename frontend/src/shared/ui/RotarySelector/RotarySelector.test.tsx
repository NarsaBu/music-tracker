import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { angleOf } from './geometry';
import { RotarySelector } from './RotarySelector';

const options = [
  { value: 'daily', label: 'Daily' },
  { value: 'weekly', label: 'Weekly' },
];

describe('RotarySelector', () => {
  it('renders an accessible radio group', () => {
    render(<RotarySelector label="Частота" options={options} value="daily" onChange={() => {}} />);
    expect(screen.getByRole('group', { name: 'Частота' })).toBeInTheDocument();
    expect(screen.getByRole('radio', { name: 'Daily' })).toBeChecked();
  });

  it('selects an option by clicking its mark', async () => {
    const onChange = vi.fn();
    render(<RotarySelector label="Частота" options={options} value="daily" onChange={onChange} />);

    await userEvent.click(screen.getByText('Weekly'));
    expect(onChange).toHaveBeenCalledWith('weekly');
  });

  it('spreads marks symmetrically', () => {
    expect(angleOf(0, 2)).toBe(-45);
    expect(angleOf(1, 2)).toBe(45);
    expect(angleOf(3, 7)).toBe(0);
    expect(angleOf(0, 1)).toBe(0);
  });
});

import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { EngravedSelect } from './EngravedSelect';

const options = [
  { value: '', label: 'Все' },
  { value: 'GB', label: 'GB' },
];

describe('EngravedSelect', () => {
  it('is a labelled select that reports the chosen value', async () => {
    const onChange = vi.fn();
    render(<EngravedSelect label="Страна" options={options} value="" onChange={onChange} />);

    const select = screen.getByRole('combobox', { name: 'Страна' });
    expect(select).toHaveValue('');
    await userEvent.selectOptions(select, 'GB');
    expect(onChange).toHaveBeenCalledWith('GB');
  });
});

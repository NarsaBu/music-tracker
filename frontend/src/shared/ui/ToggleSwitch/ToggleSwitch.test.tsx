import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { ToggleSwitch } from './ToggleSwitch';

describe('ToggleSwitch', () => {
  it('is exposed as an accessible switch', () => {
    render(<ToggleSwitch label="Album" checked onChange={() => {}} />);
    expect(screen.getByRole('switch', { name: /album/i })).toBeChecked();
  });

  it('reports the new state on click and keyboard', async () => {
    const onChange = vi.fn();
    render(<ToggleSwitch label="Album" checked={false} onChange={onChange} />);
    const sw = screen.getByRole('switch', { name: /album/i });

    await userEvent.click(sw);
    expect(onChange).toHaveBeenLastCalledWith(true);

    sw.focus();
    await userEvent.keyboard(' ');
    expect(onChange).toHaveBeenCalledTimes(2);
  });

  it('links the description', () => {
    render(
      <ToggleSwitch
        label="Featuring"
        description="Гостевые релизы"
        checked={false}
        onChange={() => {}}
      />,
    );
    expect(screen.getByRole('switch')).toHaveAccessibleDescription('Гостевые релизы');
  });
});

import { fireEvent, render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { CassetteDropzone } from './CassetteDropzone';

describe('CassetteDropzone', () => {
  it('passes a picked file on', async () => {
    const onFile = vi.fn();
    render(<CassetteDropzone label="Кассета" hint="txt или csv" accept=".txt" onFile={onFile} />);
    const input = screen.getByLabelText('Кассета');
    expect(input).toHaveAccessibleDescription('txt или csv');

    const file = new File(['Radiohead'], 'a.txt');
    await userEvent.upload(input, file);

    expect(onFile).toHaveBeenCalledWith(file);
  });

  it('accepts a dropped file unless disabled', () => {
    const onFile = vi.fn();
    const { rerender } = render(<CassetteDropzone label="Кассета" onFile={onFile} />);
    const slot = screen.getByText('Кассета').closest('label')!;
    const file = new File(['x'], 'a.csv');

    fireEvent.drop(slot, { dataTransfer: { files: [file], types: ['Files'] } });
    expect(onFile).toHaveBeenCalledWith(file);

    rerender(<CassetteDropzone label="Кассета" onFile={onFile} disabled />);
    fireEvent.drop(slot, { dataTransfer: { files: [file], types: ['Files'] } });
    expect(onFile).toHaveBeenCalledTimes(1);
    expect(screen.getByLabelText('Кассета')).toBeDisabled();
  });
});

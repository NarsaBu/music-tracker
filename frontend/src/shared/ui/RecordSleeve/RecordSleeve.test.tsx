import { fireEvent, render, screen } from '@testing-library/react';
import { RecordSleeve } from './RecordSleeve';

describe('RecordSleeve', () => {
  it('falls back to a printed sleeve when the cover fails to load', () => {
    const { container } = render(<RecordSleeve title="Mezzanine" coverUrl="https://x/cover.jpg" />);
    const img = container.querySelector('img');
    expect(img).toHaveAttribute('loading', 'lazy');

    fireEvent.error(img!);
    expect(container.querySelector('img')).toBeNull();
    expect(screen.getByRole('heading', { name: 'Mezzanine' })).toBeInTheDocument();
  });

  it('links the title to an external page', () => {
    render(<RecordSleeve title="Mezzanine" href="https://musicbrainz.org/x" />);
    expect(screen.getByRole('link', { name: 'Mezzanine' })).toHaveAttribute('target', '_blank');
  });
});

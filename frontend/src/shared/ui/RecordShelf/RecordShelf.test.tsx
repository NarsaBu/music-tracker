import { render, screen } from '@testing-library/react';
import { RecordShelf } from './RecordShelf';

describe('RecordShelf', () => {
  it('wraps every record into a list item', () => {
    render(
      <RecordShelf label="Релизы" busy>
        <span>A</span>
        <span>B</span>
      </RecordShelf>,
    );
    const list = screen.getByRole('list', { name: 'Релизы' });
    expect(list).toHaveAttribute('aria-busy', 'true');
    expect(screen.getAllByRole('listitem')).toHaveLength(2);
  });
});

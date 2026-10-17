import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { Pager } from './Pager';

describe('Pager', () => {
  it('renders nothing for a single page', () => {
    const { container } = render(
      <Pager page={0} pageCount={1} onPageChange={() => {}} label="Страницы" />,
    );
    expect(container).toBeEmptyDOMElement();
  });

  it('marks the current page and jumps to pages', async () => {
    const onPageChange = vi.fn();
    render(<Pager page={4} pageCount={20} onPageChange={onPageChange} label="Страницы" />);

    expect(screen.getByRole('button', { name: 'Страница 5' })).toHaveAttribute(
      'aria-current',
      'page',
    );
    await userEvent.click(screen.getByRole('button', { name: 'Последняя страница' }));
    expect(onPageChange).toHaveBeenLastCalledWith(19);
    await userEvent.click(screen.getByRole('button', { name: 'Страница 7' }));
    expect(onPageChange).toHaveBeenLastCalledWith(6);

    // out-of-range input is clamped to the last page
    await userEvent.type(screen.getByRole('spinbutton', { name: 'Номер страницы' }), '99');
    await userEvent.click(screen.getByRole('button', { name: 'Перейти' }));
    expect(onPageChange).toHaveBeenLastCalledWith(19);
  });
});

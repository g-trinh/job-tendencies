import { render, screen, fireEvent } from '@testing-library/react';
import { Pagination } from '../Pagination';

describe('Pagination', () => {
  it('renders the "Affichage X–Y sur Z offres" summary', () => {
    render(
      <Pagination
        page={2}
        pageSize={25}
        total={132}
        totalPages={6}
        onPageChange={() => {}}
        onPageSizeChange={() => {}}
      />,
    );

    // page 2, size 25 → rows 26-50
    expect(screen.getByText(/Affichage/)).toHaveTextContent(
      'Affichage 26–50 sur 132 offres',
    );
  });

  it('disables the "Précédent" button on the first page', () => {
    render(
      <Pagination
        page={1}
        pageSize={25}
        total={132}
        totalPages={6}
        onPageChange={() => {}}
        onPageSizeChange={() => {}}
      />,
    );

    expect(screen.getByRole('button', { name: /Précédent/ })).toBeDisabled();
    expect(screen.getByRole('button', { name: /Suivant/ })).not.toBeDisabled();
  });

  it('disables the "Suivant" button on the last page', () => {
    render(
      <Pagination
        page={6}
        pageSize={25}
        total={132}
        totalPages={6}
        onPageChange={() => {}}
        onPageSizeChange={() => {}}
      />,
    );

    expect(screen.getByRole('button', { name: /Suivant/ })).toBeDisabled();
    expect(screen.getByRole('button', { name: /Précédent/ })).not.toBeDisabled();
  });

  it('marks the active page with aria-current="page"', () => {
    render(
      <Pagination
        page={5}
        pageSize={25}
        total={288}
        totalPages={12}
        onPageChange={() => {}}
        onPageSizeChange={() => {}}
      />,
    );

    expect(
      screen.getByRole('button', { name: 'Page 5' }),
    ).toHaveAttribute('aria-current', 'page');
    expect(
      screen.getByRole('button', { name: 'Page 1' }),
    ).not.toHaveAttribute('aria-current');
  });

  it('renders an ellipsis on both sides of a windowed page cluster, with first/last always shown', () => {
    render(
      <Pagination
        page={5}
        pageSize={25}
        total={288}
        totalPages={12}
        onPageChange={() => {}}
        onPageSizeChange={() => {}}
      />,
    );

    // 1 … 4 [5] 6 … 12
    expect(screen.getByRole('button', { name: 'Page 1' })).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'Page 4' })).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'Page 5' })).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'Page 6' })).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'Page 12' })).toBeInTheDocument();
    expect(screen.queryByRole('button', { name: 'Page 2' })).not.toBeInTheDocument();
    const ellipses = document.querySelectorAll('.pagination__ellipsis');
    expect(ellipses).toHaveLength(2);
  });

  it('shows only a single active page and no ellipsis when there is one page', () => {
    render(
      <Pagination
        page={1}
        pageSize={25}
        total={12}
        totalPages={1}
        onPageChange={() => {}}
        onPageSizeChange={() => {}}
      />,
    );

    expect(screen.getByRole('button', { name: 'Page 1' })).toHaveAttribute(
      'aria-current',
      'page',
    );
    expect(document.querySelectorAll('.pagination__ellipsis')).toHaveLength(0);
    expect(screen.getByRole('button', { name: /Précédent/ })).toBeDisabled();
    expect(screen.getByRole('button', { name: /Suivant/ })).toBeDisabled();
  });

  it('calls onPageChange with the clicked page number', () => {
    const onPageChange = vi.fn();
    render(
      <Pagination
        page={5}
        pageSize={25}
        total={288}
        totalPages={12}
        onPageChange={onPageChange}
        onPageSizeChange={() => {}}
      />,
    );

    fireEvent.click(screen.getByRole('button', { name: 'Page 6' }));
    expect(onPageChange).toHaveBeenCalledWith(6);

    fireEvent.click(screen.getByRole('button', { name: /Suivant/ }));
    expect(onPageChange).toHaveBeenCalledWith(6);

    fireEvent.click(screen.getByRole('button', { name: /Précédent/ }));
    expect(onPageChange).toHaveBeenCalledWith(4);
  });

  it('calls onPageSizeChange when the page-size select changes', () => {
    const onPageSizeChange = vi.fn();
    render(
      <Pagination
        page={1}
        pageSize={25}
        total={132}
        totalPages={6}
        onPageChange={() => {}}
        onPageSizeChange={onPageSizeChange}
      />,
    );

    fireEvent.change(screen.getByLabelText('Offres par page'), {
      target: { value: '50' },
    });

    expect(onPageSizeChange).toHaveBeenCalledWith(50);
  });
});

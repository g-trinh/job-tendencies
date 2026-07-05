const PAGE_SIZE_OPTIONS = [10, 25, 50, 100] as const;

type PageToken = number | 'ellipsis-start' | 'ellipsis-end';

/**
 * Builds the page-number cluster: first and last page are always shown, plus a
 * window of one page on either side of `current`. Gaps become an ellipsis
 * token. Returns `[1]` when there is only one page (or none).
 */
function getPageWindow(current: number, totalPages: number): PageToken[] {
  if (totalPages <= 1) return [1];

  const delta = 1;
  const windowStart = Math.max(2, current - delta);
  const windowEnd = Math.min(totalPages - 1, current + delta);

  const pages: PageToken[] = [1];
  if (windowStart > 2) pages.push('ellipsis-start');
  for (let p = windowStart; p <= windowEnd; p++) pages.push(p);
  if (windowEnd < totalPages - 1) pages.push('ellipsis-end');
  pages.push(totalPages);

  return pages;
}

interface PaginationProps {
  page: number;
  pageSize: number;
  total: number;
  totalPages: number;
  onPageChange: (page: number) => void;
  onPageSizeChange: (pageSize: number) => void;
}

/**
 * Pagination bar for the jobs list (table/cards views — ADR-007). Shows the
 * "Affichage X–Y sur Z offres" summary, prev/next controls, a page-number
 * cluster with ellipsis, and the page-size select. Markup mirrors
 * `template/screens/job-browser.html`'s `.pagination` component.
 *
 * When `totalPages <= 1` the summary still renders but prev/next are disabled
 * and the page cluster shows only the single active page — the empty-list
 * case is handled by the caller (`JobsPage`), which does not render this
 * component when there are no jobs at all.
 */
function Pagination({
  page,
  pageSize,
  total,
  totalPages,
  onPageChange,
  onPageSizeChange,
}: PaginationProps) {
  const rangeStart = total === 0 ? 0 : (page - 1) * pageSize + 1;
  const rangeEnd = total === 0 ? 0 : Math.min(page * pageSize, total);
  const pages = getPageWindow(page, totalPages);

  return (
    <nav className="pagination" aria-label="Pagination">
      <p className="pagination__summary">
        Affichage{' '}
        <span className="num">
          {rangeStart}–{rangeEnd}
        </span>{' '}
        sur <span className="num">{total}</span> offres
      </p>
      <div className="pagination__controls">
        <button
          className="btn btn--secondary btn--sm"
          type="button"
          disabled={page <= 1}
          onClick={() => onPageChange(page - 1)}
        >
          <span aria-hidden="true">‹</span> Précédent
        </button>
        <ul className="pagination__pages">
          {pages.map((token, index) =>
            token === 'ellipsis-start' || token === 'ellipsis-end' ? (
              <li key={token}>
                <span className="pagination__ellipsis" aria-hidden="true">
                  …
                </span>
              </li>
            ) : (
              <li key={`page-${token}-${index}`}>
                <button
                  className="pagination__page"
                  type="button"
                  aria-current={token === page ? 'page' : undefined}
                  aria-label={`Page ${token}`}
                  onClick={() => onPageChange(token)}
                >
                  {token}
                </button>
              </li>
            ),
          )}
        </ul>
        <button
          className="btn btn--secondary btn--sm"
          type="button"
          disabled={totalPages <= 1 || page >= totalPages}
          onClick={() => onPageChange(page + 1)}
        >
          Suivant <span aria-hidden="true">›</span>
        </button>
        <span className="pagination__sep" aria-hidden="true" />
        <span className="pagination__size">
          <label className="sr-only" htmlFor="page-size">
            Offres par page
          </label>
          <select
            className="select select--sm"
            id="page-size"
            value={pageSize}
            onChange={(e) => onPageSizeChange(Number(e.target.value))}
          >
            {PAGE_SIZE_OPTIONS.map((size) => (
              <option key={size} value={size}>
                {size} / page
              </option>
            ))}
          </select>
        </span>
      </div>
    </nav>
  );
}

export { Pagination };

import { Link } from 'react-router-dom';
import { useJobs } from './useJobs';
import { KanbanBoard } from './KanbanBoard';

// Kanban stays fetch-all per ADR-007: the board only renders the tracked
// subset (non-null application_status), which stays comfortably under the
// page_size hard cap, so it requests the max page in one shot and never
// paginates per column.
const KANBAN_PAGE_SIZE = 100;

/**
 * Application kanban page at `/kanban`. Fetches all jobs for the active
 * profile (unfiltered) and hands them to `KanbanBoard`, which groups them by
 * `applicationStatus` client-side. Jobs not yet saved are excluded by the
 * board. Passes `includeExpired: true` so a tracked application whose job
 * posting has since expired still shows in its column — the board tracks
 * applications, not live postings, so expiry should not hide progress
 * (preserves pre-existing behaviour now that the server defaults to
 * excluding expired jobs).
 */
function KanbanPage() {
  const { data, isPending, isError } = useJobs(
    undefined,
    { page: 1, pageSize: KANBAN_PAGE_SIZE },
    true,
  );
  const jobs = data?.items;

  return (
    <>
      <header className="page__head">
        <div>
          <h1 className="page__title">Suivi des candidatures</h1>
          <nav aria-label="Navigation">
            <Link to="/">← Toutes les offres</Link>
          </nav>
        </div>
      </header>

      {isPending && <p>Chargement des candidatures…</p>}
      {isError && <p role="alert">Impossible de charger les candidatures.</p>}
      {jobs !== undefined && <KanbanBoard jobs={jobs} />}
    </>
  );
}

export { KanbanPage };

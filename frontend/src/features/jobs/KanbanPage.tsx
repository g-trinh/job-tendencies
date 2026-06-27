import { Link } from 'react-router-dom';
import { useJobs } from './useJobs';
import { KanbanBoard } from './KanbanBoard';

/**
 * Application kanban page at `/kanban`. Fetches all jobs for the active
 * profile (unfiltered) and hands them to `KanbanBoard`, which groups them by
 * `applicationStatus` client-side. Jobs not yet saved are excluded by the board.
 */
function KanbanPage() {
  const { data: jobs, isPending, isError } = useJobs();

  return (
    <main>
      <header>
        <h1>Suivi des candidatures</h1>
        <nav aria-label="Navigation">
          <Link to="/">← Toutes les offres</Link>
        </nav>
      </header>

      {isPending && <p>Chargement des candidatures…</p>}
      {isError && <p role="alert">Impossible de charger les candidatures.</p>}
      {jobs !== undefined && <KanbanBoard jobs={jobs} />}
    </main>
  );
}

export { KanbanPage };

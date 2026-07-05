import { usePipelineRunDetail } from './usePipelineRuns';
import { isTerminalStatus } from './types';

interface RunProgressProps {
  runId: string;
}

/** Live per-board progress for a single run, polled until a terminal status. */
function RunProgress({ runId }: RunProgressProps) {
  const { data: run, isPending, isError } = usePipelineRunDetail(runId);

  if (isPending) return <p className="muted">Chargement du suivi de l'exécution…</p>;
  if (isError)
    return (
      <div className="banner banner--danger" role="alert">
        Impossible de charger le suivi de l'exécution.
      </div>
    );
  if (!run) return null;

  const running = !isTerminalStatus(run.status);

  return (
    <section className="card" aria-label="Suivi de l'exécution">
      <div className="card__head">
        <h2 className="card__title">Suivi de l'exécution</h2>
        <span className={`badge ${running ? 'badge--brand' : 'badge--success'}`}>
          <span className="badge__dot"></span>
          Statut : {run.status}
          {running ? ' (en cours…)' : ''}
        </span>
      </div>
      <ul className="stack stack-4" aria-label="Progression par board">
        {run.boards.map((b) => (
          <li key={b.board_id} className="stack stack-2">
            <span className="text-sm medium">
              {b.board_id} — {b.status} — {b.pages_fetched} page(s),{' '}
              {b.listings_captured} offre(s)
              {b.error && <span role="alert"> — Erreur : {b.error}</span>}
            </span>
            <div className="progress">
              <div
                className={`progress__bar${
                  b.error ? ' progress__bar--danger' : ''
                }${isTerminalStatus(b.status) && !b.error ? ' progress__bar--success' : ''}`}
                style={{ width: isTerminalStatus(b.status) ? '100%' : '50%' }}
              ></div>
            </div>
          </li>
        ))}
      </ul>
      {run.boards.length === 0 && (
        <p className="muted">Aucune progression par board pour le moment.</p>
      )}
    </section>
  );
}

export { RunProgress };

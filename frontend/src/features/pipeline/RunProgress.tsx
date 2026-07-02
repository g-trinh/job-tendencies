import { usePipelineRunDetail } from './usePipelineRuns';
import { isTerminalStatus } from './types';

interface RunProgressProps {
  runId: string;
}

/** Live per-board progress for a single run, polled until a terminal status. */
function RunProgress({ runId }: RunProgressProps) {
  const { data: run, isPending, isError } = usePipelineRunDetail(runId);

  if (isPending) return <p>Chargement du suivi de l'exécution…</p>;
  if (isError) return <p role="alert">Impossible de charger le suivi de l'exécution.</p>;
  if (!run) return null;

  return (
    <section aria-label="Suivi de l'exécution">
      <h2>Suivi de l'exécution</h2>
      <p>
        Statut : {run.status}
        {isTerminalStatus(run.status) ? '' : ' (en cours…)'}
      </p>
      <ul aria-label="Progression par board">
        {run.boards.map((b) => (
          <li key={b.board_id}>
            {b.board_id} — {b.status} — {b.pages_fetched} page(s),{' '}
            {b.listings_captured} offre(s)
            {b.error && <span role="alert"> — Erreur : {b.error}</span>}
          </li>
        ))}
      </ul>
      {run.boards.length === 0 && <p>Aucune progression par board pour le moment.</p>}
    </section>
  );
}

export { RunProgress };

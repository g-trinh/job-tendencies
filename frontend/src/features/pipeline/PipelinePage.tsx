import { useState } from 'react';
import { usePipelineRunList, useTriggerPipelineRunMutation } from './usePipelineRuns';
import { RunProgress } from './RunProgress';

/**
 * Pipeline page at `/pipeline`: on-demand run trigger with live per-board
 * progress (polls `GET /api/pipeline/runs/{id}`, not the list endpoint —
 * see docs/plan/phase-6-api-contract.md), plus recent run history.
 */
function PipelinePage() {
  const [activeRunId, setActiveRunId] = useState<string | null>(null);
  const { mutate: trigger, isPending: isTriggering, isError: triggerFailed } =
    useTriggerPipelineRunMutation();
  const { data: runs, isPending, isError } = usePipelineRunList();

  return (
    <main>
      <h1>Pipeline de scraping</h1>

      <button
        type="button"
        disabled={isTriggering}
        onClick={() =>
          trigger(undefined, {
            onSuccess: (runId) => setActiveRunId(runId),
          })
        }
      >
        Lancer une exécution
      </button>
      {triggerFailed && (
        <p role="alert">Échec du déclenchement de l'exécution.</p>
      )}

      {activeRunId !== null && <RunProgress runId={activeRunId} />}

      <section aria-label="Historique des exécutions">
        <h2>Historique des exécutions</h2>
        {isPending && <p>Chargement de l'historique…</p>}
        {isError && <p role="alert">Impossible de charger l'historique.</p>}
        {runs !== undefined && runs.length === 0 && (
          <p>Aucune exécution pour le moment.</p>
        )}
        {runs !== undefined && runs.length > 0 && (
          <ul aria-label="Exécutions">
            {runs.map((run) => (
              <li key={run.run_id}>
                <button type="button" onClick={() => setActiveRunId(run.run_id)}>
                  {new Date(run.created_at).toLocaleString('fr-FR')} —{' '}
                  {run.status} ({run.trigger})
                </button>
              </li>
            ))}
          </ul>
        )}
      </section>
    </main>
  );
}

export { PipelinePage };

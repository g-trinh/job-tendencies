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
      <header className="page__head row-between">
        <div className="stack">
          <h1 className="page__title">Pipeline de scraping</h1>
          <p className="page__sub">
            Scraping → extraction LLM → scoring. Statut par source, mis à jour
            en direct.
          </p>
        </div>
        <button
          className="btn btn--primary"
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
      </header>

      <div className="stack stack-5">
        {triggerFailed && (
          <div className="banner banner--danger" role="alert">
            Échec du déclenchement de l'exécution.
          </div>
        )}

        {activeRunId !== null && <RunProgress runId={activeRunId} />}

        <section className="card" aria-label="Historique des exécutions">
          <div className="card__head">
            <h2 className="card__title">Historique des exécutions</h2>
          </div>
          {isPending && <p className="muted">Chargement de l'historique…</p>}
          {isError && (
            <div className="banner banner--danger" role="alert">
              Impossible de charger l'historique.
            </div>
          )}
          {runs !== undefined && runs.length === 0 && (
            <p className="muted">Aucune exécution pour le moment.</p>
          )}
          {runs !== undefined && runs.length > 0 && (
            <ul className="stack stack-2" aria-label="Exécutions">
              {runs.map((run) => (
                <li key={run.run_id}>
                  <button
                    className="btn btn--ghost btn--sm"
                    type="button"
                    onClick={() => setActiveRunId(run.run_id)}
                  >
                    {new Date(run.created_at).toLocaleString('fr-FR')} —{' '}
                    {run.status} ({run.trigger})
                  </button>
                </li>
              ))}
            </ul>
          )}
        </section>
      </div>
    </main>
  );
}

export { PipelinePage };

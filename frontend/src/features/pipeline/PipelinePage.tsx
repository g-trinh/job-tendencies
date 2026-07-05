import { useState } from 'react';
import { usePipelineRunList, useTriggerPipelineRunMutation } from './usePipelineRuns';
import { RunProgress } from './RunProgress';

/** Maps a raw run status to its badge modifier (template: pipeline history table). */
function statusBadgeClass(status: string): string {
  switch (status) {
    case 'completed':
      return 'badge--success';
    case 'failed':
      return 'badge--danger';
    case 'cancelled':
      return 'badge--warning';
    default:
      return 'badge--brand';
  }
}

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
    <>
      <header className="page__head">
        <div>
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
            <div className="table-wrap">
              <table className="table" aria-label="Exécutions">
                <thead>
                  <tr>
                    <th scope="col">Démarrée</th>
                    <th scope="col">Déclencheur</th>
                    <th scope="col">Statut</th>
                    <th scope="col">
                      <span className="sr-only">Actions</span>
                    </th>
                  </tr>
                </thead>
                <tbody>
                  {runs.map((run) => (
                    <tr key={run.run_id}>
                      <td className="num">
                        {new Date(run.created_at).toLocaleString('fr-FR')}
                      </td>
                      <td>
                        <span
                          className={
                            run.trigger === 'manual'
                              ? 'badge badge--brand'
                              : 'badge badge--neutral'
                          }
                        >
                          {run.trigger}
                        </span>
                      </td>
                      <td>
                        <span className={`badge ${statusBadgeClass(run.status)}`}>
                          {run.status}
                        </span>
                      </td>
                      <td className="row justify-end">
                        <button
                          className="btn btn--ghost btn--sm"
                          type="button"
                          onClick={() => setActiveRunId(run.run_id)}
                        >
                          Voir le suivi
                        </button>
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          )}
        </section>
      </div>
    </>
  );
}

export { PipelinePage };

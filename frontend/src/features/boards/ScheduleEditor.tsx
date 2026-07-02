import { useEffect, useState } from 'react';
import { useSchedule, useUpdateScheduleMutation } from './useSchedule';

/** Editor for the single global cron schedule that drives scrape ticks. */
function ScheduleEditor() {
  const { data, isPending, isError } = useSchedule();
  const [cron, setCron] = useState('');
  const { mutate, isPending: isSaving, isSuccess } =
    useUpdateScheduleMutation();

  useEffect(() => {
    if (data) setCron(data.cron);
  }, [data]);

  return (
    <section className="card" aria-label="Planification">
      <div className="card__head">
        <h2 className="card__title">Planification</h2>
      </div>
      {isPending && <p className="muted">Chargement de la planification…</p>}
      {isError && (
        <div className="banner banner--danger" role="alert">
          Impossible de charger la planification.
        </div>
      )}
      {data !== undefined && (
        <div className="stack stack-4">
          <div className="field">
            <label className="field__label" htmlFor="schedule-cron">
              Expression cron
            </label>
            <input
              className="input mono"
              id="schedule-cron"
              type="text"
              value={cron}
              onChange={(e) => setCron(e.target.value)}
            />
          </div>
          <button
            className="btn btn--primary"
            type="button"
            disabled={isSaving}
            onClick={() => mutate(cron)}
          >
            Enregistrer la planification
          </button>
          {isSuccess && (
            <span className="badge badge--success" role="status">
              Planification enregistrée.
            </span>
          )}
        </div>
      )}
    </section>
  );
}

export { ScheduleEditor };

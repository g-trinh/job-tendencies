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
    <section aria-label="Planification">
      <h2>Planification</h2>
      {isPending && <p>Chargement de la planification…</p>}
      {isError && (
        <p role="alert">Impossible de charger la planification.</p>
      )}
      {data !== undefined && (
        <>
          <div>
            <label htmlFor="schedule-cron">Expression cron</label>
            <input
              id="schedule-cron"
              type="text"
              value={cron}
              onChange={(e) => setCron(e.target.value)}
            />
          </div>
          <button
            type="button"
            disabled={isSaving}
            onClick={() => mutate(cron)}
          >
            Enregistrer la planification
          </button>
          {isSuccess && <p role="status">Planification enregistrée.</p>}
        </>
      )}
    </section>
  );
}

export { ScheduleEditor };

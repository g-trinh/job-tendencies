import { t } from '../../i18n/fr';
import { useApplicationMutation } from './useApplicationMutation';
import type { ApplicationStatus } from './types';

const ALL_STATUSES: ApplicationStatus[] = [
  'saved',
  'applied',
  'interview',
  'offer',
  'rejected',
];

interface ApplicationStatusSelectorProps {
  jobId: string;
  /** Current status for this profile+job pair; null if the job is not yet tracked. */
  currentStatus: ApplicationStatus | null;
}

/**
 * Dropdown that updates the application status for a job via
 * `PATCH /api/jobs/{id}/application`. Shows "Sauvegarder" as the initial
 * call-to-action when no status has been set. Disabled while a mutation is
 * in flight to prevent double-submission.
 */
function ApplicationStatusSelector({
  jobId,
  currentStatus,
}: ApplicationStatusSelectorProps) {
  const { mutate, isPending } = useApplicationMutation(jobId);

  function handleChange(e: React.ChangeEvent<HTMLSelectElement>) {
    const value = e.target.value as ApplicationStatus | '';
    if (value) {
      mutate(value as ApplicationStatus);
    }
  }

  return (
    <div>
      <label htmlFor={`app-status-${jobId}`}>
        {currentStatus ? 'Statut de candidature' : 'Suivre cette offre'}
      </label>
      <select
        id={`app-status-${jobId}`}
        value={currentStatus ?? ''}
        disabled={isPending}
        onChange={handleChange}
        aria-label={
          currentStatus ? 'Statut de candidature' : 'Suivre cette offre'
        }
      >
        {!currentStatus && <option value="">Sauvegarder cette offre</option>}
        {ALL_STATUSES.map((status) => (
          <option key={status} value={status}>
            {t(`application.status.${status}`)}
          </option>
        ))}
      </select>
      {isPending && <span aria-live="polite">Mise à jour…</span>}
    </div>
  );
}

export { ApplicationStatusSelector };

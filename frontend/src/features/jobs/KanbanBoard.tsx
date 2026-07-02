import { Link } from 'react-router-dom';
import { t } from '../../i18n/fr';
import { useApplicationMutation } from './useApplicationMutation';
import type { ApplicationStatus, JobSummary } from './types';

const KANBAN_COLUMNS: ApplicationStatus[] = [
  'saved',
  'applied',
  'interview',
  'offer',
  'rejected',
];

/** Next status in the forward direction, or null at the end. */
function nextStatus(current: ApplicationStatus): ApplicationStatus | null {
  const idx = KANBAN_COLUMNS.indexOf(current);
  return idx < KANBAN_COLUMNS.length - 1 ? KANBAN_COLUMNS[idx + 1] : null;
}

/** Previous status in the backward direction, or null at the start. */
function prevStatus(current: ApplicationStatus): ApplicationStatus | null {
  const idx = KANBAN_COLUMNS.indexOf(current);
  return idx > 0 ? KANBAN_COLUMNS[idx - 1] : null;
}

interface KanbanCardProps {
  job: JobSummary;
}

/** Movable card within a kanban column. */
function KanbanCard({ job }: KanbanCardProps) {
  const { mutate, isPending } = useApplicationMutation(job.id);
  const current = job.applicationStatus!;
  const prev = prevStatus(current);
  const next = nextStatus(current);

  return (
    <article aria-label={job.title || 'Offre sans titre'}>
      <h3>
        <Link to={`/jobs/${job.id}`}>{job.title || "Voir l'offre"}</Link>
      </h3>
      {(job.company || job.location) && (
        <p>{[job.company, job.location].filter(Boolean).join(' — ')}</p>
      )}
      {job.fitScore != null && <p>Pertinence : {job.fitScore}/100</p>}
      <div>
        {prev && (
          <button
            type="button"
            disabled={isPending}
            onClick={() => mutate(prev)}
            aria-label={`Reculer vers ${t(`application.status.${prev}`)}`}
          >
            ← {t(`application.status.${prev}`)}
          </button>
        )}
        {next && (
          <button
            type="button"
            disabled={isPending}
            onClick={() => mutate(next)}
            aria-label={`Avancer vers ${t(`application.status.${next}`)}`}
          >
            {t(`application.status.${next}`)} →
          </button>
        )}
      </div>
    </article>
  );
}

interface KanbanBoardProps {
  /** All jobs for the active profile; those with null applicationStatus are excluded. */
  jobs: JobSummary[];
}

/**
 * Kanban board grouped by application status. Jobs with `applicationStatus: null`
 * are not tracked yet and do not appear here; users save jobs from the list view.
 */
function KanbanBoard({ jobs }: KanbanBoardProps) {
  const tracked = jobs.filter((j) => j.applicationStatus !== null);

  return (
    <div role="region" aria-label="Kanban candidatures">
      {KANBAN_COLUMNS.map((status) => {
        const columnJobs = tracked.filter(
          (j) => j.applicationStatus === status,
        );
        return (
          <section key={status} aria-label={t(`application.status.${status}`)}>
            <h2>
              {t(`application.status.${status}`)}
              {columnJobs.length > 0 && (
                <span
                  aria-label={`${columnJobs.length} offre${columnJobs.length > 1 ? 's' : ''}`}
                >
                  {' '}
                  ({columnJobs.length})
                </span>
              )}
            </h2>
            {columnJobs.length === 0 ? (
              <p>Aucune offre dans cette colonne.</p>
            ) : (
              <ul aria-label={`Offres ${t(`application.status.${status}`)}`}>
                {columnJobs.map((job) => (
                  <li key={job.id}>
                    <KanbanCard job={job} />
                  </li>
                ))}
              </ul>
            )}
          </section>
        );
      })}
    </div>
  );
}

export { KanbanBoard };

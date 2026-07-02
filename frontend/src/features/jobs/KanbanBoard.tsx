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

/** Kanban stage dot colour token per column, per design_changes.md. */
const STAGE_TOKEN: Record<ApplicationStatus, string> = {
  saved: 'var(--color-stage-saved)',
  applied: 'var(--color-stage-applied)',
  interview: 'var(--color-stage-interview)',
  offer: 'var(--color-stage-offer)',
  rejected: 'var(--color-stage-rejected)',
};

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
    <article className="kanban__card" aria-label={job.title || 'Offre sans titre'}>
      <h3 className="text-sm">
        <Link to={`/jobs/${job.id}`}>{job.title || "Voir l'offre"}</Link>
      </h3>
      {(job.company || job.location) && (
        <p className="muted text-xs">
          {[job.company, job.location].filter(Boolean).join(' — ')}
        </p>
      )}
      {job.fitScore != null && <p>Pertinence : {job.fitScore}/100</p>}
      <div className="row">
        {prev && (
          <button
            className="btn btn--ghost btn--sm"
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
            className="btn btn--ghost btn--sm"
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
    <div className="kanban" role="region" aria-label="Kanban candidatures">
      {KANBAN_COLUMNS.map((status) => {
        const columnJobs = tracked.filter(
          (j) => j.applicationStatus === status,
        );
        return (
          <section
            className="kanban__col"
            key={status}
            aria-label={t(`application.status.${status}`)}
          >
            <div className="kanban__col-head">
              <h2 className="kanban__col-title">
                <span
                  className="kanban__stage-dot"
                  style={{ background: STAGE_TOKEN[status] }}
                />
                {t(`application.status.${status}`)}
              </h2>
              {columnJobs.length > 0 && (
                <span
                  className="kanban__count"
                  aria-label={`${columnJobs.length} offre${columnJobs.length > 1 ? 's' : ''}`}
                >
                  {columnJobs.length}
                </span>
              )}
            </div>
            {columnJobs.length === 0 ? (
              <p className="text-xs muted">Aucune offre dans cette colonne.</p>
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

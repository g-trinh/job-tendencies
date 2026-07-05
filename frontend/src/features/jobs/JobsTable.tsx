import { Link } from 'react-router-dom';
import { t } from '../../i18n/fr';
import type { JobSummary } from './types';

interface JobsTableProps {
  jobs: JobSummary[];
}

/** Formats a salary range compactly for the dense table view. */
function formatSalaryCompact(min: number | null, max: number | null): string {
  if (min === null && max === null) return '—';
  const k = (v: number) => `${Math.round(v / 1000)}k€`;
  if (min !== null && max !== null) return `${k(min)}–${k(max)}`;
  return k((min ?? max) as number);
}

/**
 * Dense table view of the jobs list. Complements the card view; toggled via
 * `ViewToggle`. Each row links to the job detail page.
 */
function JobsTable({ jobs }: JobsTableProps) {
  if (jobs.length === 0) {
    return <p>Aucune offre pour ce profil.</p>;
  }

  return (
    <div className="table-wrap">
    <table className="table">
      <thead>
        <tr>
          <th scope="col">Offre</th>
          <th scope="col">Entreprise</th>
          <th scope="col">Contrat</th>
          <th scope="col">Télétravail</th>
          <th scope="col" className="num">Salaire</th>
          <th scope="col" className="num">Pertinence</th>
          <th scope="col">Trouvée sur</th>
          <th scope="col">Candidature</th>
          <th scope="col">
            <span className="sr-only">Actions</span>
          </th>
        </tr>
      </thead>
      <tbody>
        {jobs.map((job) => (
          <tr
            key={job.id}
            className={job.expiredAt ? 'jobcard--expired' : undefined}
          >
            <td>
              <strong>
                <Link to={`/jobs/${job.id}`}>
                  {job.title || "Voir l'offre"}
                </Link>
              </strong>
              {job.expiredAt && (
                <span
                  className="badge badge--danger"
                  data-badge="expired"
                  aria-label="Offre expirée"
                >
                  {' '}
                  Expirée
                </span>
              )}
            </td>
            <td>
              {[job.company, job.location].filter(Boolean).join(' — ') || '—'}
            </td>
            <td>
              {job.contractType ? (
                <span className="badge badge--neutral">
                  {t(`job.contract.${job.contractType}`)}
                </span>
              ) : (
                '—'
              )}
            </td>
            <td>
              {job.remotePolicy ? (
                <span
                  className={
                    job.remotePolicy === 'full_remote'
                      ? 'badge badge--brand'
                      : 'badge badge--neutral'
                  }
                >
                  {t(`job.remote.${job.remotePolicy}`)}
                </span>
              ) : (
                '—'
              )}
            </td>
            <td className="num">
              {formatSalaryCompact(job.salaryMin, job.salaryMax)}
            </td>
            <td className="num">
              {job.fitScore != null ? `${job.fitScore}/100` : '—'}
            </td>
            <td className="text-xs muted">
              {job.sources.length > 0
                ? job.sources.map((s) => s.board_name).join(', ')
                : '—'}
            </td>
            <td>
              {job.applicationStatus
                ? t(`application.status.${job.applicationStatus}`)
                : '—'}
            </td>
            <td>
              {job.url ? (
                <a
                  className="btn btn--ghost btn--sm"
                  href={job.url}
                  target="_blank"
                  rel="noreferrer"
                >
                  Lien ↗
                </a>
              ) : (
                <span className="muted text-xs">—</span>
              )}
            </td>
          </tr>
        ))}
      </tbody>
    </table>
    </div>
  );
}

export { JobsTable };

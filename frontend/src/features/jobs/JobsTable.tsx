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
    <table>
      <thead>
        <tr>
          <th scope="col">Offre</th>
          <th scope="col">Entreprise</th>
          <th scope="col">Contrat</th>
          <th scope="col">Télétravail</th>
          <th scope="col">Salaire</th>
          <th scope="col">Pertinence</th>
          <th scope="col">Candidature</th>
        </tr>
      </thead>
      <tbody>
        {jobs.map((job) => (
          <tr key={job.id}>
            <td>
              <Link to={`/jobs/${job.id}`}>{job.title || "Voir l'offre"}</Link>
              {job.expiredAt && (
                <span data-badge="expired" aria-label="Offre expirée">
                  {' '}
                  Expirée
                </span>
              )}
            </td>
            <td>
              {[job.company, job.location].filter(Boolean).join(' — ') || '—'}
            </td>
            <td>
              {job.contractType ? t(`job.contract.${job.contractType}`) : '—'}
            </td>
            <td>
              {job.remotePolicy ? t(`job.remote.${job.remotePolicy}`) : '—'}
            </td>
            <td>{formatSalaryCompact(job.salaryMin, job.salaryMax)}</td>
            <td>{job.fitScore != null ? `${job.fitScore}/100` : '—'}</td>
            <td>
              {job.applicationStatus
                ? t(`application.status.${job.applicationStatus}`)
                : '—'}
            </td>
          </tr>
        ))}
      </tbody>
    </table>
  );
}

export { JobsTable };

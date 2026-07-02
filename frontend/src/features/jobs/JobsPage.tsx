import { useState } from 'react';
import { Link } from 'react-router-dom';
import { t } from '../../i18n/fr';
import { useJobs } from './useJobs';
import { JobFiltersBar } from './JobFiltersBar';
import { JobsTable } from './JobsTable';
import { ViewToggle, type View } from './ViewToggle';
import type { JobFilters, JobSummary } from './types';

/** Formats a salary range in whole euros, or a French placeholder when hidden. */
function formatSalary(min: number | null, max: number | null): string {
  if (min === null && max === null) {
    return 'Salaire non communiqué';
  }
  const euros = (value: number) => `${value.toLocaleString('fr-FR')} €`;
  if (min !== null && max !== null) {
    return `${euros(min)} – ${euros(max)}`;
  }
  return euros((min ?? max) as number);
}

/**
 * Presentational card for a single job. The heading links to the detail page;
 * an external link to the original posting is also provided when `url` is set.
 * `company`/`location` may be empty for HTML-fallback boards, so the company
 * line is conditional. Enum fields are skipped when empty (undetermined) rather
 * than rendering a raw key.
 */
function JobCard({ job }: { job: JobSummary }) {
  const companyLine = [job.company, job.location].filter(Boolean).join(' — ');

  return (
    <li>
      <article className={`card jobcard${job.expiredAt ? ' jobcard--expired' : ''}`}>
        <div className="row-between">
          <h2 className="jobcard__title">
            <Link to={`/jobs/${job.id}`}>{job.title || "Voir l'offre"}</Link>
          </h2>
          {job.fitScore != null && (
            <span className="fit-score" aria-hidden="true">
              {job.fitScore}
            </span>
          )}
        </div>
        {job.fitScore != null && <p>Pertinence : {job.fitScore}/100</p>}
        {job.expiredAt && (
          <p>
            <span
              className="badge badge--danger"
              data-badge="expired"
              aria-label="Offre expirée"
            >
              Expirée
            </span>
          </p>
        )}
        {job.url && (
          <a className="text-sm" href={job.url} target="_blank" rel="noreferrer">
            Offre originale
          </a>
        )}
        {!job.url && <p className="muted text-sm">Lien indisponible</p>}
        {companyLine !== '' && <p className="jobcard__company">{companyLine}</p>}
        <ul className="jobcard__meta" aria-label="Caractéristiques">
          {job.contractType && (
            <li className="badge badge--neutral">
              {t(`job.contract.${job.contractType}`)}
            </li>
          )}
          {job.remotePolicy && (
            <li className="badge badge--brand">
              {t(`job.remote.${job.remotePolicy}`)}
            </li>
          )}
          {job.seniority && (
            <li className="badge badge--neutral">
              {t(`job.seniority.${job.seniority}`)}
            </li>
          )}
          {job.workingDays && (
            <li className="badge badge--neutral">
              {t(`job.working_days.${job.workingDays}`)}
            </li>
          )}
        </ul>
        <p>{formatSalary(job.salaryMin, job.salaryMax)}</p>
        {job.skills.length > 0 && (
          <ul className="jobcard__meta" aria-label="Compétences">
            {job.skills.map((skill) => (
              <li key={skill} className="tag">
                {skill}
              </li>
            ))}
          </ul>
        )}
        <p className="text-xs muted">
          Compréhension : {job.understandingScore}/100
        </p>
        {job.applicationStatus && (
          <p>
            Candidature : {t(`application.status.${job.applicationStatus}`)}
          </p>
        )}
        {job.sources.length > 0 && (
          <p className="text-xs muted">
            Trouvé sur : {job.sources.map((s) => s.board_name).join(', ')}
          </p>
        )}
      </article>
    </li>
  );
}

/** Jobs list page, scoped to the active profile (see `useJobs`). */
function JobsPage() {
  const [filters, setFilters] = useState<JobFilters>({});
  const [view, setView] = useState<View>('card');
  // Hidden by default per job-browser/feature.md ("Job removed from board →
  // marked expired, data retained"). Client-side only — see JobFiltersBar.
  const [showExpired, setShowExpired] = useState(false);

  const { data: jobs, isPending, isError } = useJobs(filters);
  const visibleJobs = jobs?.filter(
    (job) => showExpired || job.expiredAt === null,
  );

  return (
    <main>
      <header className="page__head row-between">
        <h1 className="page__title">Offres</h1>
        <ViewToggle view={view} onChange={setView} />
      </header>
      <JobFiltersBar
        filters={filters}
        onChange={setFilters}
        showExpired={showExpired}
        onShowExpiredChange={setShowExpired}
      />
      {isPending && <p className="muted">Chargement des offres…</p>}
      {isError && (
        <div className="banner banner--danger" role="alert">
          Impossible de charger les offres.
        </div>
      )}
      {visibleJobs !== undefined && (
        <>
          {visibleJobs.length === 0 ? (
            <div className="state">
              <span className="state__title">Aucune offre pour ce profil.</span>
            </div>
          ) : view === 'table' ? (
            <JobsTable jobs={visibleJobs} />
          ) : (
            <ul className="grid-cards" aria-label="Offres">
              {visibleJobs.map((job) => (
                <JobCard key={job.id} job={job} />
              ))}
            </ul>
          )}
        </>
      )}
    </main>
  );
}

export { JobsPage };

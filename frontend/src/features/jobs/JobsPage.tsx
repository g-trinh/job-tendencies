import { useState, type CSSProperties } from 'react';
import { Link } from 'react-router-dom';
import { t } from '../../i18n/fr';
import { useJobs, DEFAULT_PAGE_SIZE } from './useJobs';
import { JobFiltersBar } from './JobFiltersBar';
import { JobsTable } from './JobsTable';
import { Pagination } from './Pagination';
import { ViewToggle, type View } from './ViewToggle';
import type { JobFilters, JobSummary } from './types';
import { useWidePage } from '../../components/AppShell';

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
          <div>
            <h2 className="jobcard__title">
              <Link to={`/jobs/${job.id}`}>{job.title || "Voir l'offre"}</Link>{' '}
              {job.expiredAt && (
                <span
                  className="badge badge--danger"
                  data-badge="expired"
                  aria-label="Offre expirée"
                >
                  Expirée
                </span>
              )}
            </h2>
            {companyLine !== '' && (
              <div className="jobcard__company">{companyLine}</div>
            )}
          </div>
          {job.fitScore != null && (
            <span
              className="fit-score"
              style={{ '--v': job.fitScore } as CSSProperties}
              aria-hidden="true"
            >
              {job.fitScore}
            </span>
          )}
        </div>
        {job.fitScore != null && (
          <p className="text-xs muted">
            Pertinence : <span className="num">{job.fitScore}/100</span>
          </p>
        )}
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
          <li className="badge badge--neutral num">
            {formatSalary(job.salaryMin, job.salaryMax)}
          </li>
        </ul>
        <div className="jobcard__meta">
          <span className="badge badge--neutral">
            Compréhension :{' '}
            <span className="num">{job.understandingScore}/100</span>
          </span>
          {job.applicationStatus && (
            <span className="badge badge--info">
              Candidature : {t(`application.status.${job.applicationStatus}`)}
            </span>
          )}
        </div>
        {job.skills.length > 0 && (
          <ul className="jobcard__meta" aria-label="Compétences">
            {job.skills.map((skill) => (
              <li key={skill} className="tag">
                {skill}
              </li>
            ))}
          </ul>
        )}
        <div className="row-between">
          {job.sources.length > 0 ? (
            <span className="text-xs muted">
              Trouvé sur : {job.sources.map((s) => s.board_name).join(', ')}
            </span>
          ) : (
            <span />
          )}
          {job.url ? (
            <a
              className="text-sm"
              href={job.url}
              target="_blank"
              rel="noreferrer"
            >
              Offre originale
            </a>
          ) : (
            <span className="muted text-sm">Lien indisponible</span>
          )}
        </div>
      </article>
    </li>
  );
}

/** Jobs list page, scoped to the active profile (see `useJobs`). */
function JobsPage() {
  const [filters, setFilters] = useState<JobFilters>({});
  const [view, setView] = useState<View>('table');
  // Hidden by default per job-browser/feature.md ("Job removed from board →
  // marked expired, data retained"). Sent as `include_expired` to `GET
  // /api/jobs` so the server filters in SQL — see JobFiltersBar and useJobs.
  const [showExpired, setShowExpired] = useState(false);
  // ADR-007 offset pagination — page/pageSize are local view state (single-user
  // app). Reset to page 1 whenever filters, sort, view, or expired-visibility
  // change so a stale page number is never applied to a different result set.
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(DEFAULT_PAGE_SIZE);

  function handleFiltersChange(next: JobFilters) {
    setFilters(next);
    setPage(1);
  }

  function handleViewChange(next: View) {
    setView(next);
    setPage(1);
  }

  function handlePageSizeChange(next: number) {
    setPageSize(next);
    setPage(1);
  }

  function handleShowExpiredChange(next: boolean) {
    setShowExpired(next);
    setPage(1);
  }

  const { data, isPending, isError } = useJobs(
    filters,
    { page, pageSize },
    showExpired,
  );
  const jobs = data?.items;

  useWidePage(true);

  return (
    <>
      <header className="page__head">
        <div>
          <h1 className="page__title">Offres</h1>
          {data !== undefined && (
            <p className="page__sub num">
              {data.total === 1
                ? '1 offre correspond au profil actif.'
                : `${data.total} offres correspondent au profil actif.`}
            </p>
          )}
        </div>
        <ViewToggle view={view} onChange={handleViewChange} />
      </header>
      <div className="layout-with-panel">
        <JobFiltersBar
          filters={filters}
          onChange={handleFiltersChange}
          showExpired={showExpired}
          onShowExpiredChange={handleShowExpiredChange}
        />
        <div className="stack stack-5">
          {isPending && <p className="muted">Chargement des offres…</p>}
          {isError && (
            <div className="banner banner--danger" role="alert">
              Impossible de charger les offres.
            </div>
          )}
          {jobs !== undefined && (
            <>
              {jobs.length === 0 ? (
                <div className="card">
                  <div className="state">
                    <span className="state__icon" aria-hidden="true">
                      🔍
                    </span>
                    <span className="state__title">
                      Aucune offre pour ce profil.
                    </span>
                  </div>
                </div>
              ) : view === 'table' ? (
                <JobsTable jobs={jobs} />
              ) : (
                <ul className="grid-cards" aria-label="Offres">
                  {jobs.map((job) => (
                    <JobCard key={job.id} job={job} />
                  ))}
                </ul>
              )}
            </>
          )}
          {data !== undefined && data.total > 0 && (
            <Pagination
              page={data.page}
              pageSize={data.pageSize}
              total={data.total}
              totalPages={data.totalPages}
              onPageChange={setPage}
              onPageSizeChange={handlePageSizeChange}
            />
          )}
        </div>
      </div>
    </>
  );
}

export { JobsPage };

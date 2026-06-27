import { t } from '../../i18n/fr';
import { useJobs } from './useJobs';
import type { JobSummary } from './types';

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
 * Presentational card for a single job. The heading links to the original
 * posting via `url`; `url` may be empty for boards that omit it, in which case
 * we render the title as plain text plus a "lien indisponible" notice rather
 * than a dead `<a href="">`. `company`/`location` may be empty for HTML-fallback
 * boards, so the company line is conditional. Enum fields are skipped when empty
 * (undetermined) rather than rendering a raw key.
 */
function JobCard({ job }: { job: JobSummary }) {
  const companyLine = [job.company, job.location].filter(Boolean).join(' — ');

  return (
    <li>
      <article>
        <h2>
          {job.url ? (
            <a href={job.url} target="_blank" rel="noreferrer">
              {job.title || "Voir l'offre"}
            </a>
          ) : (
            <span>{job.title || 'Offre sans titre'}</span>
          )}
        </h2>
        {!job.url && <p>Lien indisponible</p>}
        {companyLine !== '' && <p>{companyLine}</p>}
        <ul aria-label="Caractéristiques">
          {job.contractType && <li>{t(`job.contract.${job.contractType}`)}</li>}
          {job.remotePolicy && <li>{t(`job.remote.${job.remotePolicy}`)}</li>}
          {job.seniority && <li>{t(`job.seniority.${job.seniority}`)}</li>}
          {job.workingDays && <li>{t(`job.working_days.${job.workingDays}`)}</li>}
        </ul>
        <p>{formatSalary(job.salaryMin, job.salaryMax)}</p>
        {job.skills.length > 0 && (
          <ul aria-label="Compétences">
            {job.skills.map((skill) => (
              <li key={skill}>{skill}</li>
            ))}
          </ul>
        )}
        <p>Compréhension : {job.understandingScore}/100</p>
      </article>
    </li>
  );
}

/** Jobs list page, scoped to the active profile (see `useJobs`). */
function JobsPage() {
  const { data: jobs, isPending, isError } = useJobs();

  return (
    <main>
      <h1>Offres</h1>
      {isPending && <p>Chargement des offres…</p>}
      {isError && <p role="alert">Impossible de charger les offres.</p>}
      {jobs !== undefined &&
        (jobs.length === 0 ? (
          <p>Aucune offre pour ce profil.</p>
        ) : (
          <ul aria-label="Offres">
            {jobs.map((job) => (
              <JobCard key={job.id} job={job} />
            ))}
          </ul>
        ))}
    </main>
  );
}

export { JobsPage };

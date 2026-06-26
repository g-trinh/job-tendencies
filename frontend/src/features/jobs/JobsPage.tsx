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
 * posting (first source). `title`/`company`/`location` are optional today, so
 * the link falls back to a generic label and the company line is conditional.
 */
function JobCard({ job }: { job: JobSummary }) {
  const postingUrl = job.sources[0]?.sourceUrl;
  const companyLine = [job.company, job.location].filter(Boolean).join(' — ');

  return (
    <li>
      <article>
        <h2>
          {postingUrl !== undefined ? (
            <a href={postingUrl} target="_blank" rel="noreferrer">
              {job.title ?? "Voir l'offre"}
            </a>
          ) : (
            (job.title ?? 'Offre')
          )}
        </h2>
        {companyLine !== '' && <p>{companyLine}</p>}
        <ul aria-label="Caractéristiques">
          {job.contractType !== null && <li>{t(`job.contract.${job.contractType}`)}</li>}
          {job.remotePolicy !== null && <li>{t(`job.remote.${job.remotePolicy}`)}</li>}
          {job.seniority !== null && <li>{t(`job.seniority.${job.seniority}`)}</li>}
          {job.workingDays !== null && <li>{t(`job.working_days.${job.workingDays}`)}</li>}
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

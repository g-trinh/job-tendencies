import { useParams, Link } from 'react-router-dom';
import { t } from '../../i18n/fr';
import { useJobDetail } from './useJobDetail';
import { ConfidenceBadge } from './ConfidenceBadge';
import { ApplicationStatusSelector } from './ApplicationStatusSelector';

/** Formats a full salary range for the detail view. */
function formatSalary(min: number | null, max: number | null): string {
  if (min === null && max === null) return 'Salaire non communiqué';
  const euros = (v: number) => `${v.toLocaleString('fr-FR')} €`;
  if (min !== null && max !== null) return `${euros(min)} – ${euros(max)}`;
  return euros((min ?? max) as number);
}

/** Formats an ISO date string for display in French locale. */
function formatDate(iso: string): string {
  return new Date(iso).toLocaleDateString('fr-FR');
}

/** Human-readable field label for the confidence map keys. */
const FIELD_LABELS: Record<string, string> = {
  contract_type: 'Contrat',
  remote_policy: 'Télétravail',
  seniority: 'Séniorité',
  skills: 'Compétences',
  salary_min: 'Salaire min',
  salary_max: 'Salaire max',
  working_days: 'Jours travaillés',
};

/**
 * Full job detail page. Fetches `/api/jobs/:id` and renders the complete
 * structured data alongside per-field confidence badges from `field_confidence`
 * and a global `understanding_score` badge.
 *
 * Route: `/jobs/:id`
 */
function JobDetailPage() {
  const { id } = useParams<{ id: string }>();
  const { data: job, isPending, isError } = useJobDetail(id ?? '');

  return (
    <main>
      <nav aria-label="Fil d'Ariane">
        <Link to="/">← Retour aux offres</Link>
      </nav>

      {isPending && <p>Chargement de l'offre…</p>}
      {isError && <p role="alert">Impossible de charger cette offre.</p>}

      {job !== undefined && (
        <article>
          <header>
            <h1>{job.title || "Offre sans titre"}</h1>
            {job.expiredAt && (
              <p role="status">
                Cette offre a expiré le {formatDate(job.expiredAt)}.
              </p>
            )}
            {job.url && (
              <a href={job.url} target="_blank" rel="noreferrer">
                Voir l'offre originale
              </a>
            )}
          </header>

          {/* Identity */}
          {(job.company || job.location) && (
            <p>{[job.company, job.location].filter(Boolean).join(' — ')}</p>
          )}

          {/* Structured characteristics */}
          <ul aria-label="Caractéristiques">
            {job.contractType && <li>{t(`job.contract.${job.contractType}`)}</li>}
            {job.remotePolicy && <li>{t(`job.remote.${job.remotePolicy}`)}</li>}
            {job.seniority && <li>{t(`job.seniority.${job.seniority}`)}</li>}
            {job.workingDays && <li>{t(`job.working_days.${job.workingDays}`)}</li>}
          </ul>

          <p>{formatSalary(job.salaryMin, job.salaryMax)}</p>

          {job.skills.length > 0 && (
            <section aria-label="Compétences">
              <h2>Compétences</h2>
              <ul>
                {job.skills.map((skill) => (
                  <li key={skill}>{skill}</li>
                ))}
              </ul>
            </section>
          )}

          {/* Description */}
          {job.description && (
            <section aria-label="Description du poste">
              <h2>Description du poste</h2>
              <p>{job.description}</p>
            </section>
          )}

          {/* Confidence badges */}
          <section aria-label="Fiabilité de l'extraction">
            <h2>Fiabilité de l'extraction</h2>
            <p>
              Score de compréhension global :{' '}
              <span aria-label={`Score de compréhension : ${job.understandingScore}%`}>
                {job.understandingScore}/100
              </span>
            </p>
            {Object.keys(job.fieldConfidence).length > 0 && (
              <ul>
                {Object.entries(job.fieldConfidence).map(([field, score]) => (
                  <li key={field}>
                    <ConfidenceBadge
                      label={FIELD_LABELS[field] ?? field}
                      score={score}
                    />
                  </li>
                ))}
              </ul>
            )}
          </section>

          {/* Source boards */}
          {job.sources.length > 0 && (
            <section aria-label="Sources">
              <h2>Trouvé sur</h2>
              <ul>
                {job.sources.map((source) => (
                  <li key={source.board_id}>
                    <a href={source.source_url} target="_blank" rel="noreferrer">
                      {source.board_name}
                    </a>
                  </li>
                ))}
              </ul>
            </section>
          )}

          {/* Dates */}
          <section aria-label="Historique">
            <h2>Historique</h2>
            {job.firstSeen && <p>Première vue : {formatDate(job.firstSeen)}</p>}
            <p>Dernière vue : {formatDate(job.lastSeen)}</p>
          </section>

          {/* Scores */}
          {job.fitScore != null && (
            <p>Score de pertinence : {job.fitScore}/100</p>
          )}

          {/* Application status */}
          <section aria-label="Candidature">
            <h2>Candidature</h2>
            <ApplicationStatusSelector jobId={job.id} currentStatus={job.applicationStatus} />
          </section>
        </article>
      )}
    </main>
  );
}

export { JobDetailPage };

/**
 * French i18n dictionary for structured enum values.
 * Keys follow the pattern: "<domain>.<field>.<value>".
 * Raw scraped text is never translated — only structured enum fields go here.
 */
export const fr: Record<string, string> = {
  // Application kanban status
  'application.status.to_apply': 'À candidater',
  'application.status.applied': 'Candidature envoyée',
  'application.status.interview': 'Entretien',
  'application.status.offer': 'Offre reçue',
  'application.status.rejected': 'Refusé',
  'application.status.abandoned': 'Abandonné',

  // Job contract type
  'job.contract.cdi': 'CDI',
  'job.contract.cdd': 'CDD',
  'job.contract.freelance': 'Freelance',
  'job.contract.internship': 'Stage',
  'job.contract.apprenticeship': 'Alternance',

  // Remote policy
  'job.remote.none': 'Présentiel',
  'job.remote.partial': 'Hybride',
  'job.remote.full': 'Télétravail complet',
};

/**
 * Resolves a structured enum key to its French label.
 * Falls back to the key itself when no translation is registered,
 * so unknown keys degrade gracefully rather than crashing.
 */
export function t(key: string): string {
  return fr[key] ?? key;
}

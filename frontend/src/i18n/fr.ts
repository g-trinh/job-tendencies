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

  // Job contract type — values mirror kernel.ContractType.
  'job.contract.cdi': 'CDI',
  'job.contract.cdd': 'CDD',
  'job.contract.freelance': 'Freelance',
  'job.contract.interim': 'Intérim',

  // Remote policy — values mirror kernel.RemotePolicy.
  'job.remote.on_site': 'Présentiel',
  'job.remote.hybrid': 'Hybride',
  'job.remote.full_remote': 'Télétravail complet',

  // Seniority — values mirror kernel.Seniority.
  'job.seniority.entry': 'Débutant',
  'job.seniority.mid': 'Confirmé',
  'job.seniority.senior': 'Senior',
  'job.seniority.lead': 'Lead',
  'job.seniority.exec': 'Direction',

  // Working days — values mirror kernel.WorkingDays.
  'job.working_days.full_time': 'Temps plein',
  'job.working_days.part_time': 'Temps partiel',
  'job.working_days.four_day': 'Semaine de 4 jours',
};

/**
 * Resolves a structured enum key to its French label.
 * Falls back to the key itself when no translation is registered,
 * so unknown keys degrade gracefully rather than crashing.
 */
export function t(key: string): string {
  return fr[key] ?? key;
}

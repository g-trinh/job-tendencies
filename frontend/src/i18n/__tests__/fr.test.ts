import { t } from '../fr';

describe('t() — French i18n resolver', () => {
  it('resolves a known application-status enum key to French', () => {
    expect(t('application.status.applied')).toBe('Candidature envoyée');
  });

  it('resolves a known job-contract enum key to French', () => {
    expect(t('job.contract.cdi')).toBe('CDI');
  });

  it('resolves a known remote-policy enum key to French', () => {
    expect(t('job.remote.full')).toBe('Télétravail complet');
  });

  it('falls back to the key itself for unregistered keys', () => {
    expect(t('unknown.domain.value')).toBe('unknown.domain.value');
  });
});

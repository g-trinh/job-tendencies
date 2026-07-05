import type { JobSummaryDto, JobDetailDto, PagedJobsDto } from './types';

/**
 * Local fixture mirroring the `GET /api/jobs` contract. Used to stub the jobs
 * list until the backend endpoint is reachable, and as the canonical payload in
 * component tests. Shape is identical to the real wire DTO so swapping to the
 * live API requires no UI changes. The second job mimics an HTML-fallback board:
 * empty company/location and an undetermined ("") contract type.
 */
export const jobsFixture: JobSummaryDto[] = [
  {
    id: '11111111-1111-1111-1111-111111111111',
    title: 'Senior Backend Engineer (Go)',
    company: 'Alan',
    location: 'Paris',
    url: 'https://www.welcometothejungle.com/fr/companies/alan/jobs/senior-backend-engineer',
    contract_type: 'cdi',
    remote_policy: 'hybrid',
    seniority: 'senior',
    working_days: 'full_time',
    skills: ['Go', 'PostgreSQL', 'GCP'],
    salary_min: 65000,
    salary_max: 85000,
    understanding_score: 92,
    application_status: 'saved',
    fit_score: 87,
    sources: [
      {
        board_id: 'aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa',
        source_url:
          'https://www.welcometothejungle.com/fr/companies/alan/jobs/senior-backend-engineer',
        board_name: 'Welcome to the Jungle',
      },
    ],
    first_seen: '2026-06-20T10:00:00Z',
    expired_at: null,
  },
  {
    id: '22222222-2222-2222-2222-222222222222',
    title: 'Développeur Full-Stack',
    company: '',
    location: '',
    url: 'https://www.welcometothejungle.com/fr/companies/doctolib/jobs/developpeur-full-stack',
    contract_type: '',
    remote_policy: 'full_remote',
    seniority: 'mid',
    working_days: 'four_day',
    skills: ['TypeScript', 'React', 'Node.js'],
    salary_min: null,
    salary_max: null,
    understanding_score: 74,
    application_status: null,
    fit_score: null,
    sources: [
      {
        board_id: 'bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb',
        source_url:
          'https://www.welcometothejungle.com/fr/companies/doctolib/jobs/developpeur-full-stack',
        board_name: 'Welcome to the Jungle',
      },
    ],
    first_seen: '2026-06-22T14:30:00Z',
    expired_at: null,
  },
];

/**
 * List fixture including one expired job — used to test the "Expirée" badge and
 * the show/hide-expired toggle on `JobsPage`/`JobsTable`.
 */
export const jobsWithExpiredFixture: JobSummaryDto[] = [
  ...jobsFixture,
  {
    ...jobsFixture[0],
    id: '33333333-3333-3333-3333-333333333333',
    title: 'Lead Engineer (Go) — Expiré',
    expired_at: '2026-06-25T00:00:00Z',
  },
];

/** Fixture for `GET /api/jobs/{id}` — extends the first summary fixture. */
export const jobDetailFixture: JobDetailDto = {
  ...jobsFixture[0],
  description:
    'Nous recherchons un Senior Backend Engineer maîtrisant Go pour rejoindre notre équipe technique.',
  field_confidence: {
    contract_type: 95,
    remote_policy: 88,
    seniority: 90,
    skills: 85,
    salary_min: 70,
    salary_max: 70,
    working_days: 80,
  },
  contact_id: null,
  last_seen: '2026-06-27T08:00:00Z',
  expired_at: null,
};

/** Fixture for an expired job — used to test the expired state in detail view. */
export const expiredJobDetailFixture: JobDetailDto = {
  ...jobDetailFixture,
  id: '33333333-3333-3333-3333-333333333333',
  title: 'Lead Engineer (Go) — Expiré',
  expired_at: '2026-06-25T00:00:00Z',
};

/**
 * Wraps a list of job DTOs in the `GET /api/jobs` paginated envelope
 * (ADR-007). Defaults to a single full page (`page` 1, `page_size` 25, one
 * `total_pages`) — override to simulate multi-page responses in pagination
 * tests.
 */
export function toPagedJobsFixture(
  items: JobSummaryDto[],
  overrides: Partial<Omit<PagedJobsDto, 'items'>> = {},
): PagedJobsDto {
  const page = overrides.page ?? 1;
  const pageSize = overrides.page_size ?? 25;
  const total = overrides.total ?? items.length;
  const totalPages =
    overrides.total_pages ?? (total === 0 ? 0 : Math.ceil(total / pageSize));

  return {
    items,
    page,
    page_size: pageSize,
    total,
    total_pages: totalPages,
  };
}

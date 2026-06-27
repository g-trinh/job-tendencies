import type { JobSummaryDto } from './types';

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
  },
];

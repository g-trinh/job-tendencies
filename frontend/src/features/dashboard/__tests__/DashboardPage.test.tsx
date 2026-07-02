import { type ReactNode } from 'react';
import { render, screen } from '@testing-library/react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { MemoryRouter } from 'react-router-dom';
import MockAdapter from 'axios-mock-adapter';
import { apiClient, setActiveProfileId } from '../../../lib/apiClient';
import { ActiveProfileProvider } from '../../../context/ActiveProfileContext';
import { DashboardPage } from '../DashboardPage';

const ACTIVE_PROFILE_ID = 'profile-123';

function renderDashboardPage() {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  });
  function Wrapper({ children }: { children: ReactNode }) {
    return (
      <QueryClientProvider client={queryClient}>
        <ActiveProfileProvider>
          <MemoryRouter>{children}</MemoryRouter>
        </ActiveProfileProvider>
      </QueryClientProvider>
    );
  }
  return render(<DashboardPage />, { wrapper: Wrapper });
}

describe('DashboardPage', () => {
  let mock: MockAdapter;

  beforeEach(() => {
    mock = new MockAdapter(apiClient);
    setActiveProfileId(null);
    mock.onGet('/active-profile').reply(200, { id: ACTIVE_PROFILE_ID });
  });

  afterEach(() => {
    mock.restore();
  });

  it('renders stats cards from the API', async () => {
    mock.onGet('/dashboard/stats').reply(200, {
      total: 42,
      new_today: 3,
      new_this_week: 10,
      pct_remote: 55,
      avg_salary: 55000,
      top_contract_type: 'cdi',
    });
    mock.onGet('/dashboard/skills/frequency').reply(200, []);
    mock.onGet('/dashboard/skills/trend').reply(200, []);
    mock.onGet('/dashboard/matches').reply(200, []);

    renderDashboardPage();

    expect(await screen.findByText('42')).toBeInTheDocument();
    expect(screen.getByText('CDI')).toBeInTheDocument();
  });

  it('shows match alerts linking to the job detail page', async () => {
    mock.onGet('/dashboard/stats').reply(200, {
      total: 1,
      new_today: 0,
      new_this_week: 0,
      pct_remote: 0,
      avg_salary: null,
      top_contract_type: '',
    });
    mock.onGet('/dashboard/skills/frequency').reply(200, []);
    mock.onGet('/dashboard/skills/trend').reply(200, []);
    mock.onGet('/dashboard/matches').reply(200, [
      {
        id: 'job-1',
        title: 'Ingénieur Go',
        company: 'Acme',
        location: 'Paris',
        url: '',
        skills: ['Go'],
        remote_policy: 'hybrid',
        contract_type: 'cdi',
        salary_min: null,
        salary_max: null,
        weighted_score: 88,
        passes_dealbreakers: true,
      },
    ]);

    renderDashboardPage();

    expect(
      await screen.findByRole('link', { name: 'Ingénieur Go' }),
    ).toHaveAttribute('href', '/jobs/job-1');
  });

  it('shows an empty-state message when no skill frequency data is available', async () => {
    mock.onGet('/dashboard/stats').reply(200, {
      total: 0,
      new_today: 0,
      new_this_week: 0,
      pct_remote: 0,
      avg_salary: null,
      top_contract_type: '',
    });
    mock.onGet('/dashboard/skills/frequency').reply(200, []);
    mock.onGet('/dashboard/skills/trend').reply(200, []);
    mock.onGet('/dashboard/matches').reply(200, []);

    renderDashboardPage();

    expect(
      await screen.findByText('Aucune donnée de compétence disponible.'),
    ).toBeInTheDocument();
  });
});

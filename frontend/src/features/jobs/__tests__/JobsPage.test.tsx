import { type ReactNode } from 'react';
import { render, screen } from '@testing-library/react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import MockAdapter from 'axios-mock-adapter';
import { apiClient, setActiveProfileId } from '../../../lib/apiClient';
import { ActiveProfileProvider } from '../../../context/ActiveProfileContext';
import { JobsPage } from '../JobsPage';
import { jobsFixture } from '../fixtures';

const ACTIVE_PROFILE_ID = 'profile-123';

function renderJobsPage() {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  });

  function Wrapper({ children }: { children: ReactNode }) {
    return (
      <QueryClientProvider client={queryClient}>
        <ActiveProfileProvider>{children}</ActiveProfileProvider>
      </QueryClientProvider>
    );
  }

  return render(<JobsPage />, { wrapper: Wrapper });
}

describe('JobsPage', () => {
  let mock: MockAdapter;

  beforeEach(() => {
    mock = new MockAdapter(apiClient);
    setActiveProfileId(null);
    mock.onGet('/active-profile').reply(200, { id: ACTIVE_PROFILE_ID });
  });

  afterEach(() => {
    mock.restore();
  });

  // AC: the page lists jobs returned by the dev API
  it('lists the jobs returned by the API', async () => {
    mock.onGet('/jobs').reply(200, jobsFixture);

    renderJobsPage();

    expect(
      await screen.findByRole('link', { name: 'Senior Backend Engineer (Go)' }),
    ).toHaveAttribute(
      'href',
      'https://www.welcometothejungle.com/fr/companies/alan/jobs/senior-backend-engineer',
    );
    expect(
      screen.getByRole('link', { name: 'Développeur Full-Stack' }),
    ).toHaveAttribute(
      'href',
      'https://www.welcometothejungle.com/fr/companies/doctolib/jobs/developpeur-full-stack',
    );
  });

  // AC: structured enums shown in French
  it('renders structured enum fields in French', async () => {
    mock.onGet('/jobs').reply(200, jobsFixture);

    renderJobsPage();

    await screen.findByRole('link', { name: 'Senior Backend Engineer (Go)' });
    expect(screen.getByText('CDI')).toBeInTheDocument();
    expect(screen.getByText('Hybride')).toBeInTheDocument();
    expect(screen.getByText('Senior')).toBeInTheDocument();
    expect(screen.getByText('Temps plein')).toBeInTheDocument();
    expect(screen.getByText('Télétravail complet')).toBeInTheDocument();
    expect(screen.getByText('Semaine de 4 jours')).toBeInTheDocument();
  });

  // The second job's contract type is undetermined ("") — it must be omitted,
  // never rendered as the raw i18n key.
  it('omits an enum field that the extraction could not determine', async () => {
    mock.onGet('/jobs').reply(200, jobsFixture);

    renderJobsPage();

    await screen.findByRole('link', { name: 'Développeur Full-Stack' });
    expect(screen.queryByText(/job\.contract\./)).not.toBeInTheDocument();
  });

  // AC: the list is scoped to the active profile via X-Active-Profile
  it('scopes the request to the active profile via the X-Active-Profile header', async () => {
    let sentProfile: string | undefined;
    mock.onGet('/jobs').reply((config) => {
      sentProfile = (config.headers as Record<string, string>)['X-Active-Profile'];
      return [200, jobsFixture];
    });

    renderJobsPage();

    await screen.findByRole('link', { name: 'Senior Backend Engineer (Go)' });
    expect(sentProfile).toBe(ACTIVE_PROFILE_ID);
  });

  it('shows an empty-state message when no jobs match the profile', async () => {
    mock.onGet('/jobs').reply(200, []);

    renderJobsPage();

    expect(await screen.findByText('Aucune offre pour ce profil.')).toBeInTheDocument();
  });

  it('shows an error message when the jobs request fails', async () => {
    mock.onGet('/jobs').reply(500);

    renderJobsPage();

    expect(await screen.findByRole('alert')).toHaveTextContent(
      'Impossible de charger les offres.',
    );
  });
});

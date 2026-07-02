import { type ReactNode } from 'react';
import { render, screen, fireEvent } from '@testing-library/react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { MemoryRouter } from 'react-router-dom';
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
        <ActiveProfileProvider>
          <MemoryRouter>{children}</MemoryRouter>
        </ActiveProfileProvider>
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

    // Card view shows job titles as links to detail page
    expect(
      await screen.findByRole('link', { name: 'Senior Backend Engineer (Go)' }),
    ).toHaveAttribute('href', '/jobs/11111111-1111-1111-1111-111111111111');
    expect(
      screen.getByRole('link', { name: 'Développeur Full-Stack' }),
    ).toHaveAttribute('href', '/jobs/22222222-2222-2222-2222-222222222222');
  });

  // AC: structured enums shown in French
  it('renders structured enum fields in French', async () => {
    mock.onGet('/jobs').reply(200, jobsFixture);

    renderJobsPage();

    await screen.findByRole('link', { name: 'Senior Backend Engineer (Go)' });
    // Query within the characteristics lists (cards) to avoid matching filter <option> text.
    const charLists = screen.getAllByRole('list', { name: 'Caractéristiques' });
    const charText = charLists.map((el) => el.textContent).join('');
    expect(charText).toContain('CDI');
    expect(charText).toContain('Hybride');
    expect(charText).toContain('Senior');
    expect(charText).toContain('Temps plein');
    expect(charText).toContain('Télétravail complet');
    expect(charText).toContain('Semaine de 4 jours');
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
      sentProfile = (config.headers as Record<string, string>)[
        'X-Active-Profile'
      ];
      return [200, jobsFixture];
    });

    renderJobsPage();

    await screen.findByRole('link', { name: 'Senior Backend Engineer (Go)' });
    expect(sentProfile).toBe(ACTIVE_PROFILE_ID);
  });

  // A board may omit the posting URL — the card must show "Lien indisponible"
  // (no dead `<a href="">`), but still link to the detail page via the title.
  it('renders the title as a detail link and shows "Lien indisponible" when the posting URL is empty', async () => {
    const [job] = jobsFixture;
    mock.onGet('/jobs').reply(200, [{ ...job, url: '' }]);

    renderJobsPage();

    expect(await screen.findByText('Lien indisponible')).toBeInTheDocument();
    // Title still links to detail page (not the original posting)
    expect(screen.getByRole('link', { name: job.title })).toHaveAttribute(
      'href',
      `/jobs/${job.id}`,
    );
    // No "Offre originale" external link
    expect(
      screen.queryByRole('link', { name: 'Offre originale' }),
    ).not.toBeInTheDocument();
  });

  it('shows an empty-state message when no jobs match the profile', async () => {
    mock.onGet('/jobs').reply(200, []);

    renderJobsPage();

    expect(
      await screen.findByText('Aucune offre pour ce profil.'),
    ).toBeInTheDocument();
  });

  it('shows an error message when the jobs request fails', async () => {
    mock.onGet('/jobs').reply(500);

    renderJobsPage();

    expect(await screen.findByRole('alert')).toHaveTextContent(
      'Impossible de charger les offres.',
    );
  });

  // AC: application status shown in French on the card
  it('renders application status in French when present', async () => {
    mock.onGet('/jobs').reply(200, jobsFixture);

    renderJobsPage();

    await screen.findByRole('link', { name: 'Senior Backend Engineer (Go)' });
    // First fixture job has application_status: 'saved'
    expect(screen.getByText('Candidature : Sauvegardé')).toBeInTheDocument();
  });

  // AC: source board shown on the card
  it('renders the source board name on each card', async () => {
    mock.onGet('/jobs').reply(200, jobsFixture);

    renderJobsPage();

    await screen.findByRole('link', { name: 'Senior Backend Engineer (Go)' });
    expect(
      screen.getAllByText('Trouvé sur : Welcome to the Jungle'),
    ).toHaveLength(2);
  });

  // AC: fit score shown when present
  it('renders the fit score when the scoring pipeline has run', async () => {
    mock.onGet('/jobs').reply(200, jobsFixture);

    renderJobsPage();

    await screen.findByRole('link', { name: 'Senior Backend Engineer (Go)' });
    // First job has fit_score: 87
    expect(screen.getByText('Pertinence : 87/100')).toBeInTheDocument();
    // Second job has fit_score: null — score line must not appear
    // (only 1 "Pertinence" line total for the first job)
    expect(screen.getAllByText(/Pertinence :/)).toHaveLength(1);
  });

  // AC: view toggle switches between card and table layouts
  it('switches to table view when the Tableau button is pressed', async () => {
    mock.onGet('/jobs').reply(200, jobsFixture);

    renderJobsPage();

    await screen.findByRole('link', { name: 'Senior Backend Engineer (Go)' });

    // Initially in card view — jobs rendered as a list
    expect(screen.getByRole('list', { name: 'Offres' })).toBeInTheDocument();
    expect(screen.queryByRole('table')).not.toBeInTheDocument();

    fireEvent.click(screen.getByRole('button', { name: 'Tableau' }));

    expect(screen.getByRole('table')).toBeInTheDocument();
    expect(
      screen.queryByRole('list', { name: 'Offres' }),
    ).not.toBeInTheDocument();
  });

  it('switches back to card view when the Cartes button is pressed', async () => {
    mock.onGet('/jobs').reply(200, jobsFixture);

    renderJobsPage();

    await screen.findByRole('link', { name: 'Senior Backend Engineer (Go)' });

    fireEvent.click(screen.getByRole('button', { name: 'Tableau' }));
    expect(screen.getByRole('table')).toBeInTheDocument();

    fireEvent.click(screen.getByRole('button', { name: 'Cartes' }));

    expect(screen.getByRole('list', { name: 'Offres' })).toBeInTheDocument();
    expect(screen.queryByRole('table')).not.toBeInTheDocument();
  });

  // AC: filters are passed to the API as query params
  it('sends the remote_policy filter as a query param when selected', async () => {
    let sentParams: Record<string, unknown> = {};
    mock.onGet('/jobs').reply((config) => {
      sentParams = config.params as Record<string, unknown>;
      return [200, jobsFixture];
    });

    renderJobsPage();

    await screen.findByRole('link', { name: 'Senior Backend Engineer (Go)' });

    fireEvent.change(screen.getByLabelText('Télétravail'), {
      target: { value: 'hybrid' },
    });

    // Wait for the refetched request
    await screen.findByRole('link', { name: 'Senior Backend Engineer (Go)' });
    expect(sentParams['remote_policy']).toBe('hybrid');
  });
});

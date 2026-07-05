import { type ReactNode } from 'react';
import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { MemoryRouter } from 'react-router-dom';
import MockAdapter from 'axios-mock-adapter';
import { apiClient, setActiveProfileId } from '../../../lib/apiClient';
import { ActiveProfileProvider } from '../../../context/ActiveProfileContext';
import { JobsPage } from '../JobsPage';
import {
  jobsFixture,
  jobsWithExpiredFixture,
  toPagedJobsFixture,
} from '../fixtures';

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
    mock.onGet('/jobs').reply(200, toPagedJobsFixture(jobsFixture));

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
    mock.onGet('/jobs').reply(200, toPagedJobsFixture(jobsFixture));

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
    mock.onGet('/jobs').reply(200, toPagedJobsFixture(jobsFixture));

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
      return [200, toPagedJobsFixture(jobsFixture)];
    });

    renderJobsPage();

    await screen.findByRole('link', { name: 'Senior Backend Engineer (Go)' });
    expect(sentProfile).toBe(ACTIVE_PROFILE_ID);
  });

  // A board may omit the posting URL — the card must show "Lien indisponible"
  // (no dead `<a href="">`), but still link to the detail page via the title.
  it('renders the title as a detail link and shows "Lien indisponible" when the posting URL is empty', async () => {
    const [job] = jobsFixture;
    mock.onGet('/jobs').reply(200, toPagedJobsFixture([{ ...job, url: '' }]));

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
    mock.onGet('/jobs').reply(200, toPagedJobsFixture([]));

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
    mock.onGet('/jobs').reply(200, toPagedJobsFixture(jobsFixture));

    renderJobsPage();

    await screen.findByRole('link', { name: 'Senior Backend Engineer (Go)' });
    // First fixture job has application_status: 'saved'
    expect(screen.getByText('Candidature : Sauvegardé')).toBeInTheDocument();
  });

  // AC: source board shown on the card
  it('renders the source board name on each card', async () => {
    mock.onGet('/jobs').reply(200, toPagedJobsFixture(jobsFixture));

    renderJobsPage();

    await screen.findByRole('link', { name: 'Senior Backend Engineer (Go)' });
    expect(
      screen.getAllByText('Trouvé sur : Welcome to the Jungle'),
    ).toHaveLength(2);
  });

  // AC: fit score shown when present
  it('renders the fit score when the scoring pipeline has run', async () => {
    mock.onGet('/jobs').reply(200, toPagedJobsFixture(jobsFixture));

    renderJobsPage();

    await screen.findByRole('link', { name: 'Senior Backend Engineer (Go)' });
    // First job has fit_score: 87 (rendered as "Pertinence : " + a <span class="num">87/100</span>)
    expect(
      screen.getByText((_, element) => element?.textContent === 'Pertinence : 87/100'),
    ).toBeInTheDocument();
    // Second job has fit_score: null — score line must not appear
    // (only 1 "Pertinence" line total for the first job)
    expect(screen.getAllByText(/Pertinence :/)).toHaveLength(1);
  });

  // AC: view toggle switches between card and table layouts
  it('switches to table view when the Tableau button is pressed', async () => {
    mock.onGet('/jobs').reply(200, toPagedJobsFixture(jobsFixture));

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
    mock.onGet('/jobs').reply(200, toPagedJobsFixture(jobsFixture));

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
      return [200, toPagedJobsFixture(jobsFixture)];
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

  // AC: expired jobs are hidden by default (job-browser/feature.md edge case).
  // The server filters expired jobs in SQL when `include_expired` is absent —
  // simulate that by only returning the non-expired subset in that case.
  it('hides expired jobs by default and omits include_expired from the request', async () => {
    let sentParams: Record<string, unknown> = {};
    mock.onGet('/jobs').reply((config) => {
      sentParams = config.params as Record<string, unknown>;
      const includeExpired = Boolean(sentParams['include_expired']);
      const items = includeExpired
        ? jobsWithExpiredFixture
        : jobsFixture;
      return [200, toPagedJobsFixture(items)];
    });

    renderJobsPage();

    await screen.findByRole('link', { name: 'Senior Backend Engineer (Go)' });
    expect(
      screen.queryByRole('link', { name: 'Lead Engineer (Go) — Expiré' }),
    ).not.toBeInTheDocument();
    expect(sentParams['include_expired']).toBeUndefined();
  });

  // AC: a toggle reveals expired jobs, marked with an "Expirée" badge, and
  // sends `include_expired=true` so the server includes them in the paginated result.
  it('shows expired jobs with an "Expirée" badge once the toggle is checked', async () => {
    let sentParams: Record<string, unknown> = {};
    mock.onGet('/jobs').reply((config) => {
      sentParams = config.params as Record<string, unknown>;
      const includeExpired = Boolean(sentParams['include_expired']);
      const items = includeExpired
        ? jobsWithExpiredFixture
        : jobsFixture;
      return [200, toPagedJobsFixture(items)];
    });

    renderJobsPage();

    await screen.findByRole('link', { name: 'Senior Backend Engineer (Go)' });

    fireEvent.click(
      screen.getByRole('checkbox', { name: 'Afficher les offres expirées' }),
    );

    const expiredLink = await screen.findByRole('link', {
      name: 'Lead Engineer (Go) — Expiré',
    });
    expect(expiredLink).toBeInTheDocument();
    expect(screen.getByLabelText('Offre expirée')).toBeInTheDocument();
    expect(sentParams['include_expired']).toBe(true);
  });

  // AC: toggling expired visibility re-scopes the result set, so the page
  // resets to 1 (same reset pattern as filters/sort/view changes).
  it('resets to page 1 when the expired toggle is checked', async () => {
    const requestedPages: unknown[] = [];
    mock.onGet('/jobs').reply((config) => {
      const params = config.params as Record<string, unknown>;
      requestedPages.push(params['page']);
      return [
        200,
        toPagedJobsFixture(jobsFixture, {
          page: (params['page'] as number) ?? 1,
          total: 132,
        }),
      ];
    });

    renderJobsPage();

    await screen.findByRole('link', { name: 'Senior Backend Engineer (Go)' });

    fireEvent.click(screen.getByRole('button', { name: /Suivant/ }));
    await waitFor(() =>
      expect(screen.getByText(/Affichage/)).toHaveTextContent(
        'Affichage 26–50 sur 132 offres',
      ),
    );

    fireEvent.click(
      screen.getByRole('checkbox', { name: 'Afficher les offres expirées' }),
    );

    await waitFor(() =>
      expect(screen.getByText(/Affichage/)).toHaveTextContent(
        'Affichage 1–25 sur 132 offres',
      ),
    );
    expect(requestedPages).toEqual([1, 2, 1]);
  });

  // ADR-007: clicking "Suivant" fetches page 2 from the API
  it('fetches page 2 when "Suivant" is pressed', async () => {
    const requestedPages: unknown[] = [];
    mock.onGet('/jobs').reply((config) => {
      const params = config.params as Record<string, unknown>;
      requestedPages.push(params['page']);
      return [
        200,
        toPagedJobsFixture(jobsFixture, {
          page: (params['page'] as number) ?? 1,
          total: 132,
        }),
      ];
    });

    renderJobsPage();

    await screen.findByRole('link', { name: 'Senior Backend Engineer (Go)' });
    expect(screen.getByText(/Affichage/)).toHaveTextContent(
      'Affichage 1–25 sur 132 offres',
    );

    fireEvent.click(screen.getByRole('button', { name: /Suivant/ }));

    await waitFor(() =>
      expect(screen.getByText(/Affichage/)).toHaveTextContent(
        'Affichage 26–50 sur 132 offres',
      ),
    );
    expect(requestedPages).toEqual([1, 2]);
  });

  // ADR-007: changing a filter resets pagination to page 1
  it('resets to page 1 when a filter changes', async () => {
    const requestedPages: unknown[] = [];
    mock.onGet('/jobs').reply((config) => {
      const params = config.params as Record<string, unknown>;
      requestedPages.push(params['page']);
      return [
        200,
        toPagedJobsFixture(jobsFixture, {
          page: (params['page'] as number) ?? 1,
          total: 132,
        }),
      ];
    });

    renderJobsPage();

    await screen.findByRole('link', { name: 'Senior Backend Engineer (Go)' });

    fireEvent.click(screen.getByRole('button', { name: /Suivant/ }));
    await waitFor(() =>
      expect(screen.getByText(/Affichage/)).toHaveTextContent(
        'Affichage 26–50 sur 132 offres',
      ),
    );

    fireEvent.change(screen.getByLabelText('Télétravail'), {
      target: { value: 'hybrid' },
    });

    await waitFor(() =>
      expect(screen.getByText(/Affichage/)).toHaveTextContent(
        'Affichage 1–25 sur 132 offres',
      ),
    );
    expect(requestedPages).toEqual([1, 2, 1]);
  });
});

import { type ReactNode } from 'react';
import { render, screen, fireEvent } from '@testing-library/react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { MemoryRouter, Route, Routes } from 'react-router-dom';
import MockAdapter from 'axios-mock-adapter';
import { apiClient, setActiveProfileId } from '../../../lib/apiClient';
import { ActiveProfileProvider } from '../../../context/ActiveProfileContext';
import { JobDetailPage } from '../JobDetailPage';
import { jobDetailFixture, expiredJobDetailFixture } from '../fixtures';

const ACTIVE_PROFILE_ID = 'profile-123';
const JOB_ID = jobDetailFixture.id;

function renderDetailPage(jobId = JOB_ID) {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  });

  function Wrapper({ children }: { children: ReactNode }) {
    return (
      <QueryClientProvider client={queryClient}>
        <ActiveProfileProvider>
          <MemoryRouter initialEntries={[`/jobs/${jobId}`]}>
            <Routes>
              <Route path="/jobs/:id" element={<>{children}</>} />
            </Routes>
          </MemoryRouter>
        </ActiveProfileProvider>
      </QueryClientProvider>
    );
  }

  return render(<JobDetailPage />, { wrapper: Wrapper });
}

describe('JobDetailPage', () => {
  let mock: MockAdapter;

  beforeEach(() => {
    mock = new MockAdapter(apiClient);
    setActiveProfileId(null);
    mock.onGet('/active-profile').reply(200, { id: ACTIVE_PROFILE_ID });
  });

  afterEach(() => {
    mock.restore();
  });

  // AC: job detail returns all fields including confidence badges data
  it('renders the job title', async () => {
    mock.onGet(`/jobs/${JOB_ID}`).reply(200, jobDetailFixture);

    renderDetailPage();

    expect(
      await screen.findByRole('heading', {
        name: 'Senior Backend Engineer (Go)',
        level: 1,
      }),
    ).toBeInTheDocument();
  });

  it('renders the original posting link', async () => {
    mock.onGet(`/jobs/${JOB_ID}`).reply(200, jobDetailFixture);

    renderDetailPage();

    await screen.findByRole('heading', {
      name: 'Senior Backend Engineer (Go)',
      level: 1,
    });
    expect(
      screen.getByRole('link', { name: "Voir l'offre originale" }),
    ).toHaveAttribute('href', jobDetailFixture.url);
  });

  it('renders the job description', async () => {
    mock.onGet(`/jobs/${JOB_ID}`).reply(200, jobDetailFixture);

    renderDetailPage();

    await screen.findByRole('heading', {
      name: 'Senior Backend Engineer (Go)',
      level: 1,
    });
    expect(
      screen.getByText(
        'Nous recherchons un Senior Backend Engineer maîtrisant Go pour rejoindre notre équipe technique.',
      ),
    ).toBeInTheDocument();
  });

  // AC: confidence badges from field_confidence are shown
  it('renders per-field confidence badges', async () => {
    mock.onGet(`/jobs/${JOB_ID}`).reply(200, jobDetailFixture);

    renderDetailPage();

    await screen.findByRole('heading', {
      name: 'Senior Backend Engineer (Go)',
      level: 1,
    });
    // contract_type has confidence 95 → badge aria-label encodes tier and score
    expect(
      screen.getByLabelText('Contrat — confiance élevée (95%)'),
    ).toBeInTheDocument();
    // remote_policy confidence 88 → high tier
    expect(
      screen.getByLabelText('Télétravail — confiance élevée (88%)'),
    ).toBeInTheDocument();
  });

  // AC: understanding_score is shown
  it('renders the understanding score', async () => {
    mock.onGet(`/jobs/${JOB_ID}`).reply(200, jobDetailFixture);

    renderDetailPage();

    await screen.findByRole('heading', {
      name: 'Senior Backend Engineer (Go)',
      level: 1,
    });
    expect(
      screen.getByLabelText('Score de compréhension : 92%'),
    ).toBeInTheDocument();
  });

  // AC: source boards list shown
  it('renders the source board with a link', async () => {
    mock.onGet(`/jobs/${JOB_ID}`).reply(200, jobDetailFixture);

    renderDetailPage();

    await screen.findByRole('heading', {
      name: 'Senior Backend Engineer (Go)',
      level: 1,
    });
    expect(
      screen.getByRole('link', { name: 'Welcome to the Jungle' }),
    ).toHaveAttribute('href', jobDetailFixture.sources[0].source_url);
  });

  // AC: expired notice shown when expired_at is set
  it('renders an expiry notice for expired jobs', async () => {
    mock
      .onGet(`/jobs/${expiredJobDetailFixture.id}`)
      .reply(200, expiredJobDetailFixture);

    renderDetailPage(expiredJobDetailFixture.id);

    await screen.findByRole('heading', {
      name: expiredJobDetailFixture.title,
      level: 1,
    });
    expect(screen.getByRole('status')).toHaveTextContent(
      /Cette offre a expiré/,
    );
  });

  // AC: error state
  it('shows an error alert when the request fails', async () => {
    mock.onGet(`/jobs/${JOB_ID}`).reply(500);

    renderDetailPage();

    expect(await screen.findByRole('alert')).toHaveTextContent(
      'Impossible de charger cette offre.',
    );
  });

  // AC: back link to jobs list
  it('renders a link back to the jobs list', async () => {
    mock.onGet(`/jobs/${JOB_ID}`).reply(200, jobDetailFixture);

    renderDetailPage();

    await screen.findByRole('heading', {
      name: 'Senior Backend Engineer (Go)',
      level: 1,
    });
    expect(
      screen.getByRole('link', { name: '← Retour aux offres' }),
    ).toHaveAttribute('href', '/');
  });

  // AC: application status selector is shown
  it('renders the application status selector', async () => {
    mock.onGet(`/jobs/${JOB_ID}`).reply(200, jobDetailFixture);

    renderDetailPage();

    await screen.findByRole('heading', {
      name: 'Senior Backend Engineer (Go)',
      level: 1,
    });
    // Job has application_status: 'saved' → selector shows "Statut de candidature"
    expect(screen.getByLabelText('Statut de candidature')).toBeInTheDocument();
  });

  // AC: a re-extract action is available on the job detail view (P5-4)
  it('renders a re-extract button that queues re-extraction', async () => {
    mock.onGet(`/jobs/${JOB_ID}`).reply(200, jobDetailFixture);
    mock
      .onPost(`/jobs/${JOB_ID}/reextract`)
      .reply(202, { status: 're-extraction queued' });

    renderDetailPage();

    await screen.findByRole('heading', {
      name: 'Senior Backend Engineer (Go)',
      level: 1,
    });

    const button = screen.getByRole('button', {
      name: "Relancer l'extraction",
    });
    fireEvent.click(button);

    expect(await screen.findByText('Ré-extraction demandée avec succès.')).toBeInTheDocument();
    expect(mock.history.post).toHaveLength(1);
  });
});

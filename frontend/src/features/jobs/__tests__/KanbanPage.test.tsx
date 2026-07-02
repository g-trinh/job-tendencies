import { type ReactNode } from 'react';
import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { MemoryRouter } from 'react-router-dom';
import MockAdapter from 'axios-mock-adapter';
import { apiClient, setActiveProfileId } from '../../../lib/apiClient';
import { ActiveProfileProvider } from '../../../context/ActiveProfileContext';
import { KanbanPage } from '../KanbanPage';
import { jobsFixture } from '../fixtures';
import type { JobSummaryDto } from '../types';

const ACTIVE_PROFILE_ID = 'profile-123';

// A fixture job that is in 'applied' status
const appliedJob: JobSummaryDto = {
  ...jobsFixture[0],
  id: '33333333-3333-3333-3333-333333333333',
  title: 'Lead Go Engineer',
  application_status: 'applied',
};

function renderKanban() {
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

  return render(<KanbanPage />, { wrapper: Wrapper });
}

describe('KanbanPage', () => {
  let mock: MockAdapter;

  beforeEach(() => {
    mock = new MockAdapter(apiClient);
    setActiveProfileId(null);
    mock.onGet('/active-profile').reply(200, { id: ACTIVE_PROFILE_ID });
  });

  afterEach(() => {
    mock.restore();
  });

  // AC: kanban shows 5 columns for the full status lifecycle
  it('renders all five kanban columns after jobs load', async () => {
    mock.onGet('/jobs').reply(200, jobsFixture);

    renderKanban();

    // Wait for the kanban region which appears only after jobs load
    await screen.findByRole('region', { name: 'Kanban candidatures' });

    expect(
      screen.getByRole('region', { name: 'Sauvegardé' }),
    ).toBeInTheDocument();
    expect(
      screen.getByRole('region', { name: 'Candidature envoyée' }),
    ).toBeInTheDocument();
    expect(
      screen.getByRole('region', { name: 'Entretien' }),
    ).toBeInTheDocument();
    expect(
      screen.getByRole('region', { name: 'Offre reçue' }),
    ).toBeInTheDocument();
    expect(screen.getByRole('region', { name: 'Refusé' })).toBeInTheDocument();
  });

  // AC: jobs with non-null application status appear in the matching column
  it('places the saved job in the Sauvegardé column', async () => {
    mock.onGet('/jobs').reply(200, jobsFixture);

    renderKanban();

    await screen.findByRole('region', { name: 'Kanban candidatures' });

    // First fixture job has application_status: 'saved'
    const savedColumn = screen.getByRole('region', { name: 'Sauvegardé' });
    expect(savedColumn).toContainElement(
      screen.getByRole('link', { name: 'Senior Backend Engineer (Go)' }),
    );
  });

  // AC: jobs with null application_status do not appear in the kanban
  it('excludes jobs with no application status', async () => {
    mock.onGet('/jobs').reply(200, jobsFixture);

    renderKanban();

    await screen.findByRole('region', { name: 'Kanban candidatures' });

    // Second fixture job has application_status: null — must not appear
    expect(
      screen.queryByRole('link', { name: 'Développeur Full-Stack' }),
    ).not.toBeInTheDocument();
  });

  // AC: empty column shows a placeholder
  it('shows a placeholder in columns with no jobs', async () => {
    mock.onGet('/jobs').reply(200, jobsFixture);

    renderKanban();

    await screen.findByRole('region', { name: 'Kanban candidatures' });

    // 'applied' column has no jobs in the default fixture
    const appliedSection = screen.getByRole('region', {
      name: 'Candidature envoyée',
    });
    expect(appliedSection).toHaveTextContent(
      'Aucune offre dans cette colonne.',
    );
  });

  // AC: status transitions persist per profile+job via PATCH
  it('calls the application PATCH endpoint when advancing a job', async () => {
    mock.onGet('/jobs').reply(200, [appliedJob, ...jobsFixture]);
    let patchBody: unknown;
    mock.onPatch(`/jobs/${appliedJob.id}/application`).reply((config) => {
      patchBody = JSON.parse(config.data as string);
      return [200, { status: 'interview', updated_at: '2026-06-27T12:00:00Z' }];
    });
    // After mutation, re-fetch still returns the same list (testing the PATCH call, not the re-render)
    mock.onGet('/jobs').replyOnce(200, [appliedJob, ...jobsFixture]);

    renderKanban();

    await screen.findByRole('region', { name: 'Kanban candidatures' });
    expect(
      screen.getByRole('article', { name: 'Lead Go Engineer' }),
    ).toBeInTheDocument();

    fireEvent.click(
      screen.getByRole('button', { name: 'Avancer vers Entretien' }),
    );

    await waitFor(() => expect(patchBody).toEqual({ status: 'interview' }));
  });

  // AC: error state
  it('shows an error alert when the jobs request fails', async () => {
    mock.onGet('/jobs').reply(500);

    renderKanban();

    expect(await screen.findByRole('alert')).toHaveTextContent(
      'Impossible de charger les candidatures.',
    );
  });
});

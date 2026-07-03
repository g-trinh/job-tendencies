import { type ReactNode } from 'react';
import { render, screen, fireEvent } from '@testing-library/react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import MockAdapter from 'axios-mock-adapter';
import { apiClient } from '../../../lib/apiClient';
import { PipelinePage } from '../PipelinePage';

function renderPipelinePage() {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  });
  function Wrapper({ children }: { children: ReactNode }) {
    return (
      <QueryClientProvider client={queryClient}>
        {children}
      </QueryClientProvider>
    );
  }
  return render(<PipelinePage />, { wrapper: Wrapper });
}

describe('PipelinePage', () => {
  let mock: MockAdapter;

  beforeEach(() => {
    mock = new MockAdapter(apiClient);
  });

  afterEach(() => {
    mock.restore();
  });

  it('shows run history from the list endpoint', async () => {
    mock.onGet('/pipeline/runs').reply(200, {
      runs: [
        {
          run_id: 'run-1',
          profile_id: 'profile-1',
          trigger: 'manual',
          status: 'completed',
          created_at: '2026-07-01T10:00:00Z',
        },
      ],
    });

    renderPipelinePage();

    expect(
      await screen.findByText(/completed \(manual\)/),
    ).toBeInTheDocument();
  });

  it('triggers a run and polls the detail endpoint for per-board progress', async () => {
    mock.onGet('/pipeline/runs').reply(200, { runs: [] });
    mock.onPost('/pipeline/runs').reply(202, { run_id: 'run-2' });
    mock.onGet('/pipeline/runs/run-2').reply(200, {
      run_id: 'run-2',
      profile_id: 'profile-1',
      trigger: 'manual',
      status: 'running',
      created_at: '2026-07-02T10:00:00Z',
      boards: [
        {
          board_id: 'wttj',
          status: 'in_progress',
          pages_fetched: 2,
          listings_captured: 40,
        },
      ],
    });

    renderPipelinePage();

    fireEvent.click(
      screen.getByRole('button', { name: 'Lancer une exécution' }),
    );

    expect(
      await screen.findByText(/wttj — in_progress — 2 page/),
    ).toBeInTheDocument();
  });

  it('shows an empty-state message when there is no run history', async () => {
    mock.onGet('/pipeline/runs').reply(200, { runs: [] });

    renderPipelinePage();

    expect(
      await screen.findByText('Aucune exécution pour le moment.'),
    ).toBeInTheDocument();
  });
});

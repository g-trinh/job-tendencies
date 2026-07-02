import { type ReactNode } from 'react';
import { render, screen, fireEvent } from '@testing-library/react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import MockAdapter from 'axios-mock-adapter';
import { apiClient } from '../../../lib/apiClient';
import { ReextractButton } from '../ReextractButton';

const JOB_ID = '11111111-1111-1111-1111-111111111111';

function renderButton() {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  });

  function Wrapper({ children }: { children: ReactNode }) {
    return (
      <QueryClientProvider client={queryClient}>
        {children}
      </QueryClientProvider>
    );
  }

  return render(<ReextractButton jobId={JOB_ID} />, { wrapper: Wrapper });
}

describe('ReextractButton', () => {
  let mock: MockAdapter;

  beforeEach(() => {
    mock = new MockAdapter(apiClient);
  });

  afterEach(() => {
    mock.restore();
  });

  // AC: re-extraction can be triggered from the job detail view
  it('calls POST /api/jobs/{id}/reextract when clicked', async () => {
    mock
      .onPost(`/jobs/${JOB_ID}/reextract`)
      .reply(202, { status: 're-extraction queued' });

    renderButton();

    fireEvent.click(
      screen.getByRole('button', { name: "Relancer l'extraction" }),
    );

    await screen.findByText('Ré-extraction demandée avec succès.');
    expect(mock.history.post).toHaveLength(1);
  });

  // AC: the button gives pending feedback and disables to prevent double-submit
  it('disables the button and shows a pending message while the request is in flight', async () => {
    mock.onPost(`/jobs/${JOB_ID}/reextract`).reply(() => {
      return new Promise((resolve) =>
        setTimeout(() => resolve([202, { status: 're-extraction queued' }]), 20),
      );
    });

    renderButton();

    const button = screen.getByRole('button', {
      name: "Relancer l'extraction",
    });
    fireEvent.click(button);

    expect(
      await screen.findByText('Ré-extraction en cours…'),
    ).toBeInTheDocument();
    expect(button).toBeDisabled();

    await screen.findByText('Ré-extraction demandée avec succès.');
    expect(button).not.toBeDisabled();
  });

  // AC: an error state is shown, not a silent failure or a blocked UI
  it('shows an error message when the request fails', async () => {
    mock.onPost(`/jobs/${JOB_ID}/reextract`).reply(500);

    renderButton();

    fireEvent.click(
      screen.getByRole('button', { name: "Relancer l'extraction" }),
    );

    expect(await screen.findByRole('alert')).toHaveTextContent(
      "Impossible de relancer l'extraction.",
    );
  });
});

import { type ReactNode } from 'react';
import { render, screen, fireEvent } from '@testing-library/react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import MockAdapter from 'axios-mock-adapter';
import { apiClient } from '../../../lib/apiClient';
import { BoardsPage } from '../BoardsPage';
import type { BoardDto } from '../types';

const boardsFixture: BoardDto[] = [
  {
    id: 'wttj',
    name: 'Welcome to the Jungle',
    base_url: 'https://wttj.com',
    enabled: true,
    adapter: null,
  },
];

function renderBoardsPage() {
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
  return render(<BoardsPage />, { wrapper: Wrapper });
}

describe('BoardsPage', () => {
  let mock: MockAdapter;

  beforeEach(() => {
    mock = new MockAdapter(apiClient);
    mock.onGet('/schedule').reply(200, { cron: '0 * * * *' });
  });

  afterEach(() => {
    mock.restore();
  });

  it('lists boards with their enabled toggle', async () => {
    mock.onGet('/boards').reply(200, boardsFixture);

    renderBoardsPage();

    expect(
      await screen.findByRole('heading', { name: 'Welcome to the Jungle' }),
    ).toBeInTheDocument();
    expect(screen.getByLabelText('Activé')).toBeChecked();
  });

  it('warns when every board is disabled', async () => {
    mock.onGet('/boards').reply(200, [
      { ...boardsFixture[0], enabled: false },
    ]);

    renderBoardsPage();

    expect(
      await screen.findByText(
        'Tous les boards sont désactivés : aucune offre ne sera récupérée.',
      ),
    ).toBeInTheDocument();
  });

  it('shows an empty-state message when there are no boards', async () => {
    mock.onGet('/boards').reply(200, []);

    renderBoardsPage();

    expect(
      await screen.findByText(
        "Aucun board pour l'instant. Ajoutez-en un pour commencer.",
      ),
    ).toBeInTheDocument();
  });

  it('generates and approves an adapter draft', async () => {
    mock.onGet('/boards').reply(200, boardsFixture);
    mock.onPost('/boards/wttj/adapter/generate').reply(201, {
      id: 'adapter-1',
      status: 'draft',
      fetch_mode: 'html',
      version: 1,
      spec: { selectors: {} },
    });
    mock.onPost('/boards/wttj/adapter/approve').reply(200, {
      id: 'adapter-1',
      status: 'approved',
      fetch_mode: 'html',
      version: 1,
      spec: { selectors: {} },
    });

    renderBoardsPage();

    await screen.findByRole('heading', { name: 'Welcome to the Jungle' });

    fireEvent.change(
      screen.getByLabelText("Page d'exemple (HTML ou JSON de la page de recherche)"),
      { target: { value: '<html>example</html>' } },
    );
    fireEvent.click(
      screen.getByRole('button', { name: 'Générer un brouillon' }),
    );

    await screen.findByText('Aperçu du brouillon (version 1)');

    fireEvent.click(
      screen.getByRole('button', { name: "Approuver l'adaptateur" }),
    );

    expect(
      await screen.findByText('Adaptateur approuvé.'),
    ).toBeInTheDocument();
  });
});

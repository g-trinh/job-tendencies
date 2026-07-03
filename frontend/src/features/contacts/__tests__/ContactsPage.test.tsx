import { type ReactNode } from 'react';
import { render, screen, fireEvent } from '@testing-library/react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import MockAdapter from 'axios-mock-adapter';
import { apiClient } from '../../../lib/apiClient';
import { ContactsPage } from '../ContactsPage';
import type { ContactDto } from '../types';

const contactFixture: ContactDto = {
  id: 'contact-1',
  name: 'Jane Doe',
  company: 'Acme',
  email: 'jane@acme.com',
  linkedin_url: 'https://linkedin.com/in/jane',
  phone: '',
  notes: 'Recruteuse',
  tags: ['recruteur'],
  dedup_key: 'jane@acme.com',
};

function renderContactsPage() {
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
  return render(<ContactsPage />, { wrapper: Wrapper });
}

describe('ContactsPage', () => {
  let mock: MockAdapter;

  beforeEach(() => {
    mock = new MockAdapter(apiClient);
  });

  afterEach(() => {
    mock.restore();
  });

  it('lists contacts in a table', async () => {
    mock.onGet('/contacts').reply(200, [contactFixture]);

    renderContactsPage();

    expect(await screen.findByText('Jane Doe')).toBeInTheDocument();
    expect(screen.getByText('jane@acme.com')).toBeInTheDocument();
  });

  it('links to the CSV export endpoint', async () => {
    mock.onGet('/contacts').reply(200, []);

    renderContactsPage();

    expect(
      await screen.findByRole('link', { name: 'Exporter en CSV' }),
    ).toHaveAttribute('href', '/api/contacts/export.csv');
  });

  it('shows a merge confirmation when adding a contact upserts an existing one', async () => {
    mock.onGet('/contacts').reply(200, []);
    mock.onPost('/contacts').reply(200, contactFixture);

    renderContactsPage();

    await screen.findByText("Aucun contact pour l'instant.");

    fireEvent.change(screen.getByLabelText('Nom'), {
      target: { value: 'Jane Doe' },
    });
    fireEvent.click(screen.getByRole('button', { name: 'Enregistrer' }));

    expect(
      await screen.findByText(
        'Un contact existant a été mis à jour (même e-mail ou profil LinkedIn).',
      ),
    ).toBeInTheDocument();
  });

  it('shows an empty-state message when there are no contacts', async () => {
    mock.onGet('/contacts').reply(200, []);

    renderContactsPage();

    expect(
      await screen.findByText("Aucun contact pour l'instant."),
    ).toBeInTheDocument();
  });
});

import { type ReactNode } from 'react';
import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import MockAdapter from 'axios-mock-adapter';
import { apiClient } from '../../../lib/apiClient';
import { AuthProvider } from '../../../context/AuthContext';
import { LoginPage } from '../LoginPage';

const userFixture = { email: 'admin@example.com' };

function renderLoginPage() {
  const mock = new MockAdapter(apiClient);
  // AuthProvider bootstraps by calling GET /auth/me; stub as unauthenticated so
  // we can render LoginPage directly without interference.
  mock.onGet('/auth/me').reply(401);

  function Wrapper({ children }: { children: ReactNode }) {
    return <AuthProvider>{children}</AuthProvider>;
  }

  const utils = render(<LoginPage />, { wrapper: Wrapper });
  return { ...utils, mock };
}

describe('LoginPage — rendu', () => {
  afterEach(() => {
    // Restore the adapter to not leak state between tests
  });

  // AC: unauthenticated user sees the login screen
  it('affiche un titre et les champs en français', async () => {
    const { mock } = renderLoginPage();

    // Wait for auth bootstrap to settle
    await screen.findByRole('main', { name: 'Connexion' });

    expect(
      screen.getByRole('heading', { name: 'Connexion' }),
    ).toBeInTheDocument();
    expect(screen.getByLabelText('Adresse e-mail')).toBeInTheDocument();
    expect(screen.getByLabelText('Mot de passe')).toBeInTheDocument();
    expect(
      screen.getByRole('button', { name: 'Se connecter' }),
    ).toBeInTheDocument();

    mock.restore();
  });

  it('associe correctement les labels à leurs champs', async () => {
    const { mock } = renderLoginPage();
    await screen.findByRole('main', { name: 'Connexion' });

    const emailInput = screen.getByLabelText('Adresse e-mail');
    expect(emailInput).toHaveAttribute('type', 'email');
    expect(emailInput).toHaveAttribute('name', 'email');

    const passwordInput = screen.getByLabelText('Mot de passe');
    expect(passwordInput).toHaveAttribute('type', 'password');
    expect(passwordInput).toHaveAttribute('name', 'password');

    mock.restore();
  });
});

describe('LoginPage — soumission réussie', () => {
  // AC: valid login posts credentials and calls checkAuth
  it("envoie l'email et le mot de passe au serveur via POST /api/auth/login", async () => {
    const mock = new MockAdapter(apiClient);
    mock.onGet('/auth/me').replyOnce(401); // initial check: unauthenticated
    mock.onPost('/auth/login').replyOnce(200, {});
    mock.onGet('/auth/me').reply(200, userFixture); // after login: authenticated

    function Wrapper({ children }: { children: ReactNode }) {
      return <AuthProvider>{children}</AuthProvider>;
    }
    render(<LoginPage />, { wrapper: Wrapper });
    await screen.findByRole('main', { name: 'Connexion' });

    fireEvent.change(screen.getByLabelText('Adresse e-mail'), {
      target: { value: 'admin@example.com' },
    });
    fireEvent.change(screen.getByLabelText('Mot de passe'), {
      target: { value: 'secret123' },
    });
    fireEvent.click(screen.getByRole('button', { name: 'Se connecter' }));

    await waitFor(() => {
      expect(mock.history.post).toHaveLength(1);
    });

    const [loginRequest] = mock.history.post;
    expect(loginRequest.url).toBe('/auth/login');
    expect(JSON.parse(loginRequest.data as string)).toEqual({
      email: 'admin@example.com',
      password: 'secret123',
    });

    mock.restore();
  });

  it('désactive le bouton et affiche le label de chargement pendant la soumission', async () => {
    const mock = new MockAdapter(apiClient);
    mock.onGet('/auth/me').reply(401);
    // Delay login response so we can observe the submitting state
    mock.onPost('/auth/login').reply(() => new Promise(() => {}));

    function Wrapper({ children }: { children: ReactNode }) {
      return <AuthProvider>{children}</AuthProvider>;
    }
    render(<LoginPage />, { wrapper: Wrapper });
    await screen.findByRole('main', { name: 'Connexion' });

    fireEvent.change(screen.getByLabelText('Adresse e-mail'), {
      target: { value: 'admin@example.com' },
    });
    fireEvent.change(screen.getByLabelText('Mot de passe'), {
      target: { value: 'secret' },
    });
    fireEvent.click(screen.getByRole('button', { name: 'Se connecter' }));

    await waitFor(() => {
      expect(
        screen.getByRole('button', { name: 'Connexion en cours…' }),
      ).toBeDisabled();
    });

    mock.restore();
  });
});

describe('LoginPage — identifiants incorrects', () => {
  // AC: login form shows French error on 401
  it("affiche un message d'erreur en français quand les identifiants sont invalides", async () => {
    const mock = new MockAdapter(apiClient);
    mock.onGet('/auth/me').reply(401);
    mock.onPost('/auth/login').reply(401);

    function Wrapper({ children }: { children: ReactNode }) {
      return <AuthProvider>{children}</AuthProvider>;
    }
    render(<LoginPage />, { wrapper: Wrapper });
    await screen.findByRole('main', { name: 'Connexion' });

    fireEvent.change(screen.getByLabelText('Adresse e-mail'), {
      target: { value: 'wrong@example.com' },
    });
    fireEvent.change(screen.getByLabelText('Mot de passe'), {
      target: { value: 'badpassword' },
    });
    fireEvent.click(screen.getByRole('button', { name: 'Se connecter' }));

    expect(await screen.findByRole('alert')).toHaveTextContent(
      'Identifiants incorrects. Veuillez réessayer.',
    );

    mock.restore();
  });

  it("affiche un message d'erreur générique en cas d'erreur réseau", async () => {
    const mock = new MockAdapter(apiClient);
    mock.onGet('/auth/me').reply(401);
    mock.onPost('/auth/login').networkError();

    function Wrapper({ children }: { children: ReactNode }) {
      return <AuthProvider>{children}</AuthProvider>;
    }
    render(<LoginPage />, { wrapper: Wrapper });
    await screen.findByRole('main', { name: 'Connexion' });

    fireEvent.change(screen.getByLabelText('Adresse e-mail'), {
      target: { value: 'admin@example.com' },
    });
    fireEvent.change(screen.getByLabelText('Mot de passe'), {
      target: { value: 'secret' },
    });
    fireEvent.click(screen.getByRole('button', { name: 'Se connecter' }));

    expect(await screen.findByRole('alert')).toHaveTextContent(
      'Une erreur est survenue. Veuillez réessayer.',
    );

    mock.restore();
  });

  it('réactive le bouton après une erreur de soumission', async () => {
    const mock = new MockAdapter(apiClient);
    mock.onGet('/auth/me').reply(401);
    mock.onPost('/auth/login').reply(401);

    function Wrapper({ children }: { children: ReactNode }) {
      return <AuthProvider>{children}</AuthProvider>;
    }
    render(<LoginPage />, { wrapper: Wrapper });
    await screen.findByRole('main', { name: 'Connexion' });

    fireEvent.change(screen.getByLabelText('Adresse e-mail'), {
      target: { value: 'admin@example.com' },
    });
    fireEvent.change(screen.getByLabelText('Mot de passe'), {
      target: { value: 'bad' },
    });
    fireEvent.click(screen.getByRole('button', { name: 'Se connecter' }));

    // Wait for error to appear
    await screen.findByRole('alert');

    // Button should be enabled again
    expect(screen.getByRole('button', { name: 'Se connecter' })).toBeEnabled();

    mock.restore();
  });
});

import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import { MemoryRouter } from 'react-router-dom';
import MockAdapter from 'axios-mock-adapter';
import { apiClient, setOnUnauthorized } from '../../../lib/apiClient';
import { AuthProvider, useAuth } from '../../../context/AuthContext';
import { RequireAuth } from '../RequireAuth';

const userFixture = { email: 'admin@example.com' };

/**
 * A minimal protected component used as a stand-in for the full app.
 * Exposes a logout button and a "trigger 401" button to drive AC tests.
 */
function ProtectedApp() {
  const { logout } = useAuth();

  async function triggerGuardedCall() {
    // Simulates any guarded /api call returning 401 (e.g. session expired mid-use)
    try {
      await apiClient.get('/jobs');
    } catch {
      // The 401 interceptor in apiClient has already called _onUnauthorized;
      // the caught error here is intentionally swallowed in this test helper.
    }
  }

  return (
    <div>
      <h1>Application</h1>
      <button onClick={triggerGuardedCall}>Charger les offres</button>
      <button onClick={logout}>Se déconnecter</button>
    </div>
  );
}

function renderGuard() {
  return render(
    <AuthProvider>
      <MemoryRouter>
        <RequireAuth>
          <ProtectedApp />
        </RequireAuth>
      </MemoryRouter>
    </AuthProvider>,
  );
}

describe("RequireAuth — utilisateur non authentifié", () => {
  let mock: MockAdapter;

  beforeEach(() => {
    mock = new MockAdapter(apiClient);
  });

  afterEach(() => {
    mock.restore();
    setOnUnauthorized(null);
  });

  // AC: unauthenticated user sees the login screen
  it("affiche l'écran de connexion quand /auth/me retourne 401", async () => {
    mock.onGet('/auth/me').reply(401);

    renderGuard();

    // Wait for the auth check to complete and the login screen to appear
    await screen.findByRole('main', { name: 'Connexion' });

    expect(screen.getByRole('heading', { name: 'Connexion' })).toBeInTheDocument();
    expect(screen.queryByRole('heading', { name: 'Application' })).not.toBeInTheDocument();
  });

  it("n'affiche rien pendant la vérification de session en cours", () => {
    // Never resolves — simulates the loading state
    mock.onGet('/auth/me').reply(() => new Promise(() => {}));

    renderGuard();

    // Nothing should be rendered (no login, no app)
    expect(screen.queryByRole('main', { name: 'Connexion' })).not.toBeInTheDocument();
    expect(screen.queryByRole('heading', { name: 'Application' })).not.toBeInTheDocument();
  });
});

describe("RequireAuth — utilisateur authentifié", () => {
  let mock: MockAdapter;

  beforeEach(() => {
    mock = new MockAdapter(apiClient);
  });

  afterEach(() => {
    mock.restore();
    setOnUnauthorized(null);
  });

  // AC: valid session reveals the app
  it("affiche le contenu de l'application quand /auth/me retourne l'utilisateur", async () => {
    mock.onGet('/auth/me').reply(200, userFixture);

    renderGuard();

    await screen.findByRole('heading', { name: 'Application' });

    expect(screen.queryByRole('main', { name: 'Connexion' })).not.toBeInTheDocument();
  });
});

describe("RequireAuth — connexion réussie", () => {
  let mock: MockAdapter;

  beforeEach(() => {
    mock = new MockAdapter(apiClient);
  });

  afterEach(() => {
    mock.restore();
    setOnUnauthorized(null);
  });

  // AC: valid login reveals the app
  it("révèle l'application après une connexion valide", async () => {
    // First /auth/me call: unauthenticated → shows login screen
    mock.onGet('/auth/me').replyOnce(401);
    // POST /auth/login: success (cookie set by server, no token in body)
    mock.onPost('/auth/login').replyOnce(200, {});
    // Second /auth/me call (from checkAuth after login): authenticated
    mock.onGet('/auth/me').reply(200, userFixture);

    renderGuard();

    // 1. Login screen appears
    await screen.findByRole('main', { name: 'Connexion' });

    // 2. Fill and submit the form
    fireEvent.change(screen.getByLabelText('Adresse e-mail'), {
      target: { value: 'admin@example.com' },
    });
    fireEvent.change(screen.getByLabelText('Mot de passe'), {
      target: { value: 'secret123' },
    });
    fireEvent.click(screen.getByRole('button', { name: 'Se connecter' }));

    // 3. App content replaces the login screen
    await screen.findByRole('heading', { name: 'Application' });

    expect(screen.queryByRole('main', { name: 'Connexion' })).not.toBeInTheDocument();
  });
});

describe("RequireAuth — appel API non autorisé (AC2)", () => {
  let mock: MockAdapter;

  beforeEach(() => {
    mock = new MockAdapter(apiClient);
  });

  afterEach(() => {
    mock.restore();
    setOnUnauthorized(null);
  });

  // AC: 401 from any /api call redirects to login
  it("redirige vers la connexion quand un appel à une route protégée retourne 401", async () => {
    // Start authenticated
    mock.onGet('/auth/me').reply(200, userFixture);
    // The guarded /api/jobs call will return 401 (session expired mid-use)
    mock.onGet('/jobs').reply(401);

    renderGuard();

    // 1. App is shown
    await screen.findByRole('heading', { name: 'Application' });

    // 2. A guarded API call returns 401 — triggers the interceptor
    fireEvent.click(screen.getByRole('button', { name: 'Charger les offres' }));

    // 3. Login screen replaces the app
    await screen.findByRole('main', { name: 'Connexion' });

    expect(screen.queryByRole('heading', { name: 'Application' })).not.toBeInTheDocument();
  });
});

describe("RequireAuth — déconnexion", () => {
  let mock: MockAdapter;

  beforeEach(() => {
    mock = new MockAdapter(apiClient);
  });

  afterEach(() => {
    mock.restore();
    setOnUnauthorized(null);
  });

  it("appelle POST /auth/logout et affiche l'écran de connexion", async () => {
    mock.onGet('/auth/me').reply(200, userFixture);
    mock.onPost('/auth/logout').reply(200);

    renderGuard();

    // 1. App is shown
    await screen.findByRole('heading', { name: 'Application' });

    // 2. User clicks logout
    fireEvent.click(screen.getByRole('button', { name: 'Se déconnecter' }));

    // 3. Login screen replaces the app
    await screen.findByRole('main', { name: 'Connexion' });

    await waitFor(() => {
      expect(mock.history.post).toHaveLength(1);
      expect(mock.history.post[0].url).toBe('/auth/logout');
    });
  });

  it("affiche l'écran de connexion même si POST /auth/logout échoue", async () => {
    mock.onGet('/auth/me').reply(200, userFixture);
    mock.onPost('/auth/logout').networkError();

    renderGuard();

    await screen.findByRole('heading', { name: 'Application' });
    fireEvent.click(screen.getByRole('button', { name: 'Se déconnecter' }));

    // Login screen should still appear — logout clears local state regardless
    await screen.findByRole('main', { name: 'Connexion' });
  });
});

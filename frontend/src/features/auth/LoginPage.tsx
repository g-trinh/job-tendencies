import { useState, type FormEvent } from 'react';
import axios from 'axios';
import { apiClient } from '../../lib/apiClient';
import { useAuth } from '../../context/AuthContext';

function LoginPage() {
  const { checkAuth } = useAuth();
  const [error, setError] = useState<string | null>(null);
  const [isSubmitting, setIsSubmitting] = useState(false);

  async function handleSubmit(e: FormEvent<HTMLFormElement>) {
    e.preventDefault();
    setError(null);
    setIsSubmitting(true);

    const form = e.currentTarget;
    const email = (form.elements.namedItem('email') as HTMLInputElement).value;
    const password = (form.elements.namedItem('password') as HTMLInputElement).value;

    try {
      // Backend P4-BE-2 must implement POST /api/auth/login.
      // Expected contract: body { email, password } → 200 + sets httpOnly session
      // cookie. No token in the response body. 401 on bad credentials.
      await apiClient.post('/auth/login', { email, password });
      await checkAuth();
    } catch (err) {
      if (axios.isAxiosError(err) && err.response?.status === 401) {
        setError('Identifiants incorrects. Veuillez réessayer.');
      } else {
        setError('Une erreur est survenue. Veuillez réessayer.');
      }
    } finally {
      setIsSubmitting(false);
    }
  }

  return (
    <main aria-label="Connexion">
      <h1>Connexion</h1>
      <form onSubmit={handleSubmit} noValidate>
        <div>
          <label htmlFor="email">Adresse e-mail</label>
          <input
            id="email"
            name="email"
            type="email"
            required
            autoComplete="email"
            disabled={isSubmitting}
          />
        </div>
        <div>
          <label htmlFor="password">Mot de passe</label>
          <input
            id="password"
            name="password"
            type="password"
            required
            autoComplete="current-password"
            disabled={isSubmitting}
          />
        </div>
        {error !== null && <p role="alert">{error}</p>}
        <button type="submit" disabled={isSubmitting}>
          {isSubmitting ? 'Connexion en cours…' : 'Se connecter'}
        </button>
      </form>
    </main>
  );
}

export { LoginPage };

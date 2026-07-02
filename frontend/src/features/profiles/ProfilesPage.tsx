import { useState } from 'react';
import { useProfiles } from './useProfiles';
import { useCreateProfileMutation } from './useProfileMutations';
import { IdentityEditor } from './IdentityEditor';
import { ConditionsEditor } from './ConditionsEditor';
import { WeightsEditor } from './WeightsEditor';

/** Simple inline form to create a new profile. */
function CreateProfileForm() {
  const [name, setName] = useState('');
  const [location, setLocation] = useState('');
  const [keywords, setKeywords] = useState('');
  const { mutate, isPending, isSuccess } = useCreateProfileMutation();

  function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    mutate(
      {
        name,
        location,
        search_keywords: keywords
          .split(',')
          .map((k) => k.trim())
          .filter(Boolean),
      },
      {
        onSuccess: () => {
          setName('');
          setLocation('');
          setKeywords('');
        },
      },
    );
  }

  return (
    <form className="card" aria-label="Créer un profil" onSubmit={handleSubmit}>
      <div className="card__head">
        <h2 className="card__title">Créer un profil</h2>
      </div>
      <div className="stack stack-4">
        <div className="field">
          <label className="field__label" htmlFor="create-name">
            Nom
          </label>
          <input
            className="input"
            id="create-name"
            type="text"
            required
            value={name}
            onChange={(e) => setName(e.target.value)}
          />
        </div>
        <div className="field">
          <label className="field__label" htmlFor="create-location">
            Localisation
          </label>
          <input
            className="input"
            id="create-location"
            type="text"
            value={location}
            onChange={(e) => setLocation(e.target.value)}
          />
        </div>
        <div className="field">
          <label className="field__label" htmlFor="create-keywords">
            Mots-clés de recherche (séparés par des virgules)
          </label>
          <input
            className="input"
            id="create-keywords"
            type="text"
            value={keywords}
            onChange={(e) => setKeywords(e.target.value)}
          />
        </div>
        <div className="row">
          <button className="btn btn--primary" type="submit" disabled={isPending}>
            Créer
          </button>
          {isSuccess && (
            <span className="badge badge--success" role="status">
              Profil créé.
            </span>
          )}
        </div>
      </div>
    </form>
  );
}

/**
 * Profiles page at `/profiles`: profile list, and (for the active profile)
 * identity/skills, search conditions, and fit-score weights editors. The
 * profile switcher itself lives in the app shell topbar (see AppShell).
 */
function ProfilesPage() {
  const { data: profiles, isPending, isError } = useProfiles();
  const active = profiles?.find((p) => p.isActive);

  return (
    <main>
      <header className="page__head">
        <h1 className="page__title">Profils</h1>
        {active && (
          <p className="page__sub">
            Profil actif : <strong>{active.name}</strong>
          </p>
        )}
      </header>

      {isPending && <p className="muted">Chargement des profils…</p>}
      {isError && (
        <div className="banner banner--danger" role="alert">
          Impossible de charger les profils.
        </div>
      )}

      {profiles !== undefined && profiles.length === 0 && (
        <div className="state">
          <span className="state__title">
            Aucun profil pour l'instant. Créez-en un pour commencer.
          </span>
        </div>
      )}

      {active !== undefined && (
        <div className="stack stack-5">
          <IdentityEditor profile={active} />
          <ConditionsEditor
            profileId={active.id}
            conditions={active.conditions}
          />
          <WeightsEditor profileId={active.id} weights={active.weights} />
        </div>
      )}

      <CreateProfileForm />
    </main>
  );
}

export { ProfilesPage };

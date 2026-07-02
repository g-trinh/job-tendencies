import { useState } from 'react';
import { useProfiles } from './useProfiles';
import { useCreateProfileMutation } from './useProfileMutations';
import { ProfileSwitcher } from './ProfileSwitcher';
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
    <form aria-label="Créer un profil" onSubmit={handleSubmit}>
      <h2>Créer un profil</h2>
      <div>
        <label htmlFor="create-name">Nom</label>
        <input
          id="create-name"
          type="text"
          required
          value={name}
          onChange={(e) => setName(e.target.value)}
        />
      </div>
      <div>
        <label htmlFor="create-location">Localisation</label>
        <input
          id="create-location"
          type="text"
          value={location}
          onChange={(e) => setLocation(e.target.value)}
        />
      </div>
      <div>
        <label htmlFor="create-keywords">
          Mots-clés de recherche (séparés par des virgules)
        </label>
        <input
          id="create-keywords"
          type="text"
          value={keywords}
          onChange={(e) => setKeywords(e.target.value)}
        />
      </div>
      <button type="submit" disabled={isPending}>
        Créer
      </button>
      {isSuccess && <p role="status">Profil créé.</p>}
    </form>
  );
}

/**
 * Profiles page at `/profiles`: switcher, profile list, and (for the active
 * profile) identity/skills, search conditions, and fit-score weights editors.
 */
function ProfilesPage() {
  const { data: profiles, isPending, isError } = useProfiles();
  const active = profiles?.find((p) => p.isActive);

  return (
    <main>
      <h1>Profils</h1>
      <ProfileSwitcher />

      {isPending && <p>Chargement des profils…</p>}
      {isError && <p role="alert">Impossible de charger les profils.</p>}

      {profiles !== undefined && profiles.length === 0 && (
        <p>Aucun profil pour l'instant. Créez-en un pour commencer.</p>
      )}

      {active !== undefined && (
        <>
          <IdentityEditor profile={active} />
          <ConditionsEditor
            profileId={active.id}
            conditions={active.conditions}
          />
          <WeightsEditor profileId={active.id} weights={active.weights} />
        </>
      )}

      <CreateProfileForm />
    </main>
  );
}

export { ProfilesPage };

import { useActiveProfile } from '../../context/ActiveProfileContext';
import { useProfiles } from './useProfiles';

/** Dropdown to switch the globally active profile; re-scopes all server state. */
function ProfileSwitcher() {
  const { data: profiles } = useProfiles();
  const { activeProfileId, switchActiveProfile, isSwitching } =
    useActiveProfile();

  if (!profiles || profiles.length === 0) return null;

  return (
    <div>
      <label htmlFor="profile-switcher">Profil actif</label>
      <select
        id="profile-switcher"
        value={activeProfileId ?? ''}
        disabled={isSwitching}
        onChange={(e) => void switchActiveProfile(e.target.value)}
      >
        {profiles.map((p) => (
          <option key={p.id} value={p.id}>
            {p.name}
          </option>
        ))}
      </select>
    </div>
  );
}

export { ProfileSwitcher };

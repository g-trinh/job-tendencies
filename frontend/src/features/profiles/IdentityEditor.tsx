import { useRef, useState } from 'react';
import {
  useImportIdentityMutation,
  useUpdateIdentityMutation,
} from './useProfileMutations';
import type { Profile } from './types';
import type { Seniority } from '../jobs/types';
import { t } from '../../i18n/fr';

const SENIORITY_OPTIONS: Seniority[] = ['entry', 'mid', 'senior', 'lead', 'exec'];

interface IdentityEditorProps {
  profile: Profile;
}

/** Skills editor + seniority + PDF (LinkedIn export) import for a profile. */
function IdentityEditor({ profile }: IdentityEditorProps) {
  const [skillsInput, setSkillsInput] = useState(profile.skills.join(', '));
  const [seniority, setSeniority] = useState<string>(profile.seniority);
  const fileInputRef = useRef<HTMLInputElement>(null);

  const { mutate: updateIdentity, isPending: isSaving } =
    useUpdateIdentityMutation(profile.id);
  const {
    mutate: importPdf,
    isPending: isImporting,
    isError: importFailed,
  } = useImportIdentityMutation(profile.id);

  function handleSave() {
    const skills = skillsInput
      .split(',')
      .map((s) => s.trim())
      .filter(Boolean);
    updateIdentity({ skills, seniority });
  }

  function handleFileChange(e: React.ChangeEvent<HTMLInputElement>) {
    const file = e.target.files?.[0];
    if (file) importPdf(file);
  }

  return (
    <section aria-label="Identité et compétences">
      <h2>Identité et compétences</h2>

      <div>
        <label htmlFor="identity-pdf">Importer un CV LinkedIn (PDF)</label>
        <input
          id="identity-pdf"
          ref={fileInputRef}
          type="file"
          accept="application/pdf"
          onChange={handleFileChange}
          disabled={isImporting}
        />
        {isImporting && <p>Import en cours…</p>}
        {importFailed && (
          <p role="alert">
            Échec de l'import du PDF. Vérifiez le fichier et réessayez.
          </p>
        )}
      </div>

      <div>
        <label htmlFor="identity-skills">Compétences (séparées par des virgules)</label>
        <textarea
          id="identity-skills"
          value={skillsInput}
          onChange={(e) => setSkillsInput(e.target.value)}
        />
      </div>

      <div>
        <label htmlFor="identity-seniority">Séniorité</label>
        <select
          id="identity-seniority"
          value={seniority}
          onChange={(e) => setSeniority(e.target.value)}
        >
          <option value="">Non déterminée</option>
          {SENIORITY_OPTIONS.map((s) => (
            <option key={s} value={s}>
              {t(`job.seniority.${s}`)}
            </option>
          ))}
        </select>
      </div>

      {profile.rawExperience && (
        <div>
          <h3>Expérience brute extraite</h3>
          <p>{profile.rawExperience}</p>
        </div>
      )}

      <button type="button" disabled={isSaving} onClick={handleSave}>
        Enregistrer l'identité
      </button>
    </section>
  );
}

export { IdentityEditor };

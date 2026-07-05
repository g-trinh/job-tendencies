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
    <section className="card" aria-label="Identité et compétences">
      <div className="card__head">
        <h2 className="card__title">Identité et compétences</h2>
      </div>

      <div className="banner banner--info mbe-4">
        <span aria-hidden="true">ℹ️</span>
        <span>
          Importez votre export PDF LinkedIn pour extraire automatiquement
          compétences, expérience et séniorité.
        </span>
      </div>

      <div className="stack stack-4">
        <div className="row wrap">
          <label className="btn btn--secondary" htmlFor="identity-pdf">
            Importer un CV LinkedIn (PDF)
          </label>
          <input
            className="sr-only"
            id="identity-pdf"
            ref={fileInputRef}
            type="file"
            accept="application/pdf"
            onChange={handleFileChange}
            disabled={isImporting}
          />
          {isImporting && (
            <p className="skeleton skeleton--text skeleton--line-60" role="status">
              Import en cours…
            </p>
          )}
          {importFailed && (
            <div className="banner banner--danger" role="alert">
              Échec de l'import du PDF. Vérifiez le fichier et réessayez.
            </div>
          )}
        </div>

        <div className="field">
          <label className="field__label" htmlFor="identity-skills">
            Compétences (séparées par des virgules)
          </label>
          <textarea
            className="textarea"
            id="identity-skills"
            value={skillsInput}
            onChange={(e) => setSkillsInput(e.target.value)}
          />
        </div>

        <div className="field">
          <label className="field__label" htmlFor="identity-seniority">
            Séniorité
          </label>
          <select
            className="select"
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
          <div className="field">
            <span className="field__label">Expérience brute extraite</span>
            <p className="muted text-sm">{profile.rawExperience}</p>
          </div>
        )}

        <div className="row justify-end">
          <button
            className="btn btn--primary"
            type="button"
            disabled={isSaving}
            onClick={handleSave}
          >
            Enregistrer l'identité
          </button>
        </div>
      </div>
    </section>
  );
}

export { IdentityEditor };

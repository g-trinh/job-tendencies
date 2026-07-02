import { useState } from 'react';
import { useUpdateWeightsMutation } from './useProfileMutations';
import type { ProfileWeightsDto } from './types';

interface WeightsEditorProps {
  profileId: string;
  weights: ProfileWeightsDto;
}

const FIELDS: { key: keyof ProfileWeightsDto; label: string }[] = [
  { key: 'preferred_skills', label: 'Compétences préférées' },
  { key: 'salary', label: 'Salaire' },
  { key: 'location', label: 'Localisation' },
  { key: 'office_days', label: 'Jours de présence' },
  { key: 'working_days', label: 'Jours travaillés' },
];

/**
 * Sliders (0-100) for the five fit-score weights. Per profiles/feature.md, the
 * weights are a *soft* sum-to-100 constraint — the sum is not blocked, only
 * flagged — so the "Enregistrer" button stays enabled and a warning banner
 * appears instead when the total is not 100.
 */
function WeightsEditor({ profileId, weights }: WeightsEditorProps) {
  const [draft, setDraft] = useState<ProfileWeightsDto>(weights);
  const { mutate, isPending, isSuccess } = useUpdateWeightsMutation(profileId);

  const sum = FIELDS.reduce((acc, f) => acc + (draft[f.key] || 0), 0);
  const isBalanced = sum === 100;

  return (
    <section aria-label="Pondération du score de pertinence">
      <h2>Pondération du score de pertinence</h2>
      {FIELDS.map((f) => (
        <div key={f.key}>
          <label htmlFor={`weight-${f.key}`}>
            {f.label} : {draft[f.key]}%
          </label>
          <input
            id={`weight-${f.key}`}
            type="range"
            min={0}
            max={100}
            value={draft[f.key]}
            onChange={(e) =>
              setDraft({ ...draft, [f.key]: Number(e.target.value) })
            }
          />
        </div>
      ))}
      <p aria-live="polite">Total : {sum}%</p>
      {!isBalanced && (
        <p role="alert">
          La somme des pondérations n'est pas égale à 100 % (actuellement{' '}
          {sum}%). Vous pouvez tout de même enregistrer.
        </p>
      )}
      <button
        type="button"
        disabled={isPending}
        onClick={() => mutate(draft)}
      >
        Enregistrer
      </button>
      {isSuccess && <p role="status">Pondérations enregistrées.</p>}
    </section>
  );
}

export { WeightsEditor };

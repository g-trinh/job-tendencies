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
    <section className="card" aria-label="Pondération du score de pertinence">
      <div className="card__head">
        <h2 className="card__title">Pondération du score de pertinence</h2>
      </div>
      <div className="stack stack-4">
        {FIELDS.map((f) => (
          <div className="slider-row" key={f.key}>
            <label htmlFor={`weight-${f.key}`}>{f.label}</label>
            <input
              className="slider"
              id={`weight-${f.key}`}
              type="range"
              min={0}
              max={100}
              value={draft[f.key]}
              onChange={(e) =>
                setDraft({ ...draft, [f.key]: Number(e.target.value) })
              }
            />
            <span className="slider-row__val">{draft[f.key]}%</span>
          </div>
        ))}
      </div>
      <p aria-live="polite" className="sr-only">
        Total : {sum}%
      </p>
      <div className={`weights-sum ${isBalanced ? 'weights-sum--ok' : 'weights-sum--off'}`}>
        <span>Total : {sum}%</span>
        {!isBalanced && (
          <span role="alert">
            La somme des pondérations n'est pas égale à 100 % (actuellement{' '}
            {sum}%). Vous pouvez tout de même enregistrer.
          </span>
        )}
      </div>
      <button
        className="btn btn--primary"
        type="button"
        disabled={isPending}
        onClick={() => mutate(draft)}
      >
        Enregistrer
      </button>
      {isSuccess && (
        <span className="badge badge--success" role="status">
          Pondérations enregistrées.
        </span>
      )}
    </section>
  );
}

export { WeightsEditor };

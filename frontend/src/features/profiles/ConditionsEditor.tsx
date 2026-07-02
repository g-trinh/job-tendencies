import { useState } from 'react';
import { useUpdateConditionsMutation } from './useProfileMutations';
import type { ProfileConditionsDto } from './types';
import type { ContractType, RemotePolicy, WorkingDays } from '../jobs/types';

interface ConditionsEditorProps {
  profileId: string;
  conditions: ProfileConditionsDto;
}

/** Deal-breaker (hard) and preference (soft) search conditions for a profile. */
function ConditionsEditor({ profileId, conditions }: ConditionsEditorProps) {
  const [draft, setDraft] = useState<ProfileConditionsDto>(conditions);
  const { mutate, isPending, isSuccess } =
    useUpdateConditionsMutation(profileId);

  function set<K extends keyof ProfileConditionsDto>(
    key: K,
    value: ProfileConditionsDto[K],
  ) {
    setDraft({ ...draft, [key]: value });
  }

  return (
    <section className="card" aria-label="Conditions de recherche">
      <div className="card__head"><h2 className="card__title">Conditions de recherche</h2></div>

      <fieldset className="stack stack-4"><legend className="filter-group__title">Critères éliminatoires</legend>
      <div className="field">
        <label className="field__label" htmlFor="cond-contract">Type de contrat requis</label>
        <select
          className="select"
          id="cond-contract"
          value={draft.dealbreaker_contract_type ?? ''}
          onChange={(e) =>
            set(
              'dealbreaker_contract_type',
              (e.target.value as ContractType) || null,
            )
          }
        >
          <option value="">Indifférent</option>
          <option value="cdi">CDI</option>
          <option value="cdd">CDD</option>
          <option value="freelance">Freelance</option>
          <option value="interim">Intérim</option>
        </select>
      </div>
      <div className="field">
        <label className="field__label" htmlFor="cond-remote">Télétravail requis</label>
        <select
          className="select"
          id="cond-remote"
          value={draft.dealbreaker_remote_policy ?? ''}
          onChange={(e) =>
            set(
              'dealbreaker_remote_policy',
              (e.target.value as RemotePolicy) || null,
            )
          }
        >
          <option value="">Indifférent</option>
          <option value="on_site">Présentiel</option>
          <option value="hybrid">Hybride</option>
          <option value="full_remote">Télétravail complet</option>
        </select>
      </div>
      <div className="field">
        <label className="field__label" htmlFor="cond-salary-min">Salaire minimum (€)</label>
        <input
          className="input"
          id="cond-salary-min"
          type="number"
          min={0}
          value={draft.dealbreaker_salary_min ?? ''}
          onChange={(e) =>
            set(
              'dealbreaker_salary_min',
              e.target.value ? Number(e.target.value) : null,
            )
          }
        />
      </div>
      <div className="field">
        <label className="field__label" htmlFor="cond-required-skills">
          Compétences obligatoires (séparées par des virgules)
        </label>
        <input
          className="input"
          id="cond-required-skills"
          type="text"
          defaultValue={draft.dealbreaker_required_skills.join(', ')}
          onBlur={(e) =>
            set(
              'dealbreaker_required_skills',
              e.target.value
                .split(',')
                .map((s) => s.trim())
                .filter(Boolean),
            )
          }
        />
      </div>

      </fieldset><fieldset className="stack stack-4"><legend className="filter-group__title">Préférences</legend>
      <div className="field">
        <label className="field__label" htmlFor="cond-preferred-skills">
          Compétences préférées (séparées par des virgules)
        </label>
        <input
          className="input"
          id="cond-preferred-skills"
          type="text"
          defaultValue={draft.preferred_skills.join(', ')}
          onBlur={(e) =>
            set(
              'preferred_skills',
              e.target.value
                .split(',')
                .map((s) => s.trim())
                .filter(Boolean),
            )
          }
        />
      </div>
      <div className="field">
        <label className="field__label" htmlFor="cond-office-days">Jours de présence max</label>
        <input
          className="input"
          id="cond-office-days"
          type="number"
          min={0}
          max={7}
          value={draft.preferred_max_office_days ?? ''}
          onChange={(e) =>
            set(
              'preferred_max_office_days',
              e.target.value ? Number(e.target.value) : null,
            )
          }
        />
      </div>
      <div className="field">
        <label className="field__label" htmlFor="cond-location">Localisation préférée</label>
        <input
          className="input"
          id="cond-location"
          type="text"
          value={draft.preferred_location}
          onChange={(e) => set('preferred_location', e.target.value)}
        />
      </div>
      <div className="field">
        <label className="field__label" htmlFor="cond-working-days">Jours travaillés préférés</label>
        <select
          className="select"
          id="cond-working-days"
          value={draft.preferred_working_days}
          onChange={(e) =>
            set('preferred_working_days', e.target.value as WorkingDays | '')
          }
        >
          <option value="">Indifférent</option>
          <option value="full_time">Temps plein</option>
          <option value="part_time">Temps partiel</option>
          <option value="four_day">Semaine de 4 jours</option>
        </select>
      </div>

      </fieldset>
      <button className="btn btn--primary" type="button" disabled={isPending} onClick={() => mutate(draft)}>
        Enregistrer les conditions
      </button>
      {isSuccess && (
        <span className="badge badge--success" role="status">
          Conditions enregistrées.
        </span>
      )}
    </section>
  );
}

export { ConditionsEditor };

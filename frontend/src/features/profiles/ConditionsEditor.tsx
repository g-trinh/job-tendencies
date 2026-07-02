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
    <section aria-label="Conditions de recherche">
      <h2>Conditions de recherche</h2>

      <h3>Critères éliminatoires</h3>
      <div>
        <label htmlFor="cond-contract">Type de contrat requis</label>
        <select
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
      <div>
        <label htmlFor="cond-remote">Télétravail requis</label>
        <select
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
      <div>
        <label htmlFor="cond-salary-min">Salaire minimum (€)</label>
        <input
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
      <div>
        <label htmlFor="cond-required-skills">
          Compétences obligatoires (séparées par des virgules)
        </label>
        <input
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

      <h3>Préférences</h3>
      <div>
        <label htmlFor="cond-preferred-skills">
          Compétences préférées (séparées par des virgules)
        </label>
        <input
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
      <div>
        <label htmlFor="cond-office-days">Jours de présence max</label>
        <input
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
      <div>
        <label htmlFor="cond-location">Localisation préférée</label>
        <input
          id="cond-location"
          type="text"
          value={draft.preferred_location}
          onChange={(e) => set('preferred_location', e.target.value)}
        />
      </div>
      <div>
        <label htmlFor="cond-working-days">Jours travaillés préférés</label>
        <select
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

      <button type="button" disabled={isPending} onClick={() => mutate(draft)}>
        Enregistrer les conditions
      </button>
      {isSuccess && <p role="status">Conditions enregistrées.</p>}
    </section>
  );
}

export { ConditionsEditor };

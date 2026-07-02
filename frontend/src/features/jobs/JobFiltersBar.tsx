import type {
  ContractType,
  JobFilters,
  RemotePolicy,
  SortDir,
  SortField,
} from './types';

interface JobFiltersBarProps {
  filters: JobFilters;
  onChange: (filters: JobFilters) => void;
}

// Known boards seeded by P3-BO-1. Replace with a dynamic /api/boards fetch
// when that endpoint lands (board list endpoint deferred to a follow-up task).
const KNOWN_BOARDS = [
  { id: 'wttj', name: 'Welcome to the Jungle' },
  { id: 'indeed', name: 'Indeed' },
  { id: 'linkedin', name: 'LinkedIn' },
  { id: 'glassdoor', name: 'Glassdoor' },
];

/**
 * Controlled filter + sort bar for the jobs list. All filter state lives in the
 * parent (`JobsPage`); this component emits a new `JobFilters` object on every
 * change. Skills are entered as a comma-separated string and split on blur.
 */
function JobFiltersBar({ filters, onChange }: JobFiltersBarProps) {
  function set<K extends keyof JobFilters>(key: K, value: JobFilters[K]) {
    onChange({ ...filters, [key]: value });
  }

  function handleSkillsBlur(e: React.FocusEvent<HTMLInputElement>) {
    const raw = e.target.value.trim();
    const skills = raw
      ? raw
          .split(',')
          .map((s) => s.trim())
          .filter(Boolean)
      : [];
    set('skills', skills.length ? skills : undefined);
  }

  return (
    <section aria-label="Filtres et tri">
      <div>
        {/* Skills — comma-separated text input */}
        <label htmlFor="filter-skills">Compétences</label>
        <input
          id="filter-skills"
          type="text"
          placeholder="Go, React, …"
          defaultValue={filters.skills?.join(', ') ?? ''}
          onBlur={handleSkillsBlur}
        />
      </div>

      <div>
        <label htmlFor="filter-remote">Télétravail</label>
        <select
          id="filter-remote"
          value={filters.remote_policy ?? ''}
          onChange={(e) =>
            set(
              'remote_policy',
              (e.target.value as RemotePolicy | '') || undefined,
            )
          }
        >
          <option value="">Tous</option>
          <option value="on_site">Présentiel</option>
          <option value="hybrid">Hybride</option>
          <option value="full_remote">Télétravail complet</option>
        </select>
      </div>

      <div>
        <label htmlFor="filter-contract">Contrat</label>
        <select
          id="filter-contract"
          value={filters.contract_type ?? ''}
          onChange={(e) =>
            set(
              'contract_type',
              (e.target.value as ContractType | '') || undefined,
            )
          }
        >
          <option value="">Tous</option>
          <option value="cdi">CDI</option>
          <option value="cdd">CDD</option>
          <option value="freelance">Freelance</option>
          <option value="interim">Intérim</option>
        </select>
      </div>

      <div>
        <label htmlFor="filter-salary-min">Salaire min (€)</label>
        <input
          id="filter-salary-min"
          type="number"
          min={0}
          value={filters.salary_min ?? ''}
          onChange={(e) =>
            set(
              'salary_min',
              e.target.value ? parseInt(e.target.value, 10) : undefined,
            )
          }
        />
      </div>

      <div>
        <label htmlFor="filter-salary-max">Salaire max (€)</label>
        <input
          id="filter-salary-max"
          type="number"
          min={0}
          value={filters.salary_max ?? ''}
          onChange={(e) =>
            set(
              'salary_max',
              e.target.value ? parseInt(e.target.value, 10) : undefined,
            )
          }
        />
      </div>

      <div>
        <label htmlFor="filter-location">Localisation</label>
        <input
          id="filter-location"
          type="text"
          value={filters.location ?? ''}
          onChange={(e) => set('location', e.target.value || undefined)}
        />
      </div>

      <div>
        <label htmlFor="filter-board">Source</label>
        <select
          id="filter-board"
          value={filters.board_id ?? ''}
          onChange={(e) => set('board_id', e.target.value || undefined)}
        >
          <option value="">Toutes les sources</option>
          {KNOWN_BOARDS.map((b) => (
            <option key={b.id} value={b.id}>
              {b.name}
            </option>
          ))}
        </select>
      </div>

      <div>
        <label htmlFor="filter-since">Depuis</label>
        <input
          id="filter-since"
          type="date"
          value={filters.since ?? ''}
          onChange={(e) => set('since', e.target.value || undefined)}
        />
      </div>

      <div>
        <label htmlFor="filter-confidence">Confiance min (%)</label>
        <input
          id="filter-confidence"
          type="number"
          min={0}
          max={100}
          value={filters.confidence_min ?? ''}
          onChange={(e) =>
            set(
              'confidence_min',
              e.target.value ? parseInt(e.target.value, 10) : undefined,
            )
          }
        />
      </div>

      {/* Sort controls */}
      <div>
        <label htmlFor="sort-field">Trier par</label>
        <select
          id="sort-field"
          value={filters.sort ?? 'date'}
          onChange={(e) => set('sort', e.target.value as SortField)}
        >
          <option value="date">Date</option>
          <option value="fit">Pertinence</option>
          <option value="salary">Salaire</option>
        </select>
      </div>

      <div>
        <label htmlFor="sort-dir">Ordre</label>
        <select
          id="sort-dir"
          value={filters.sort_dir ?? 'desc'}
          onChange={(e) => set('sort_dir', e.target.value as SortDir)}
        >
          <option value="desc">Décroissant</option>
          <option value="asc">Croissant</option>
        </select>
      </div>
    </section>
  );
}

export { JobFiltersBar };

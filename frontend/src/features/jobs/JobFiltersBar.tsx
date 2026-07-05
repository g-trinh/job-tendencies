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
  /**
   * Whether expired jobs (job.expired_at set) are included in the list.
   * This is a client-side display toggle, not a `GET /api/jobs` query param —
   * the backend does not filter by expiry, so all jobs are fetched and this
   * flag decides what `JobsPage` renders. Defaults to hidden per
   * job-browser/feature.md ("Job removed from board → marked expired").
   */
  showExpired: boolean;
  onShowExpiredChange: (showExpired: boolean) => void;
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
function JobFiltersBar({
  filters,
  onChange,
  showExpired,
  onShowExpiredChange,
}: JobFiltersBarProps) {
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
    <aside className="card" aria-label="Filtres et tri">
      <h2 className="card__title mbe-4">Filtres</h2>
      <div className="field filter-group">
        {/* Skills — comma-separated text input */}
        <label className="field__label" htmlFor="filter-skills">
          Compétences
        </label>
        <input
          className="input"
          id="filter-skills"
          type="text"
          placeholder="Go, React, …"
          defaultValue={filters.skills?.join(', ') ?? ''}
          onBlur={handleSkillsBlur}
        />
      </div>

      <div className="field filter-group">
        <label className="field__label" htmlFor="filter-remote">
          Télétravail
        </label>
        <select
          className="select"
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

      <div className="field filter-group">
        <label className="field__label" htmlFor="filter-contract">
          Contrat
        </label>
        <select
          className="select"
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

      <div className="filter-group">
        <div className="filter-group__title">Salaire (€)</div>
        <div className="row">
          <label className="sr-only" htmlFor="filter-salary-min">
            Salaire min (€)
          </label>
          <input
            className="input"
            id="filter-salary-min"
            type="number"
            min={0}
            placeholder="min"
            value={filters.salary_min ?? ''}
            onChange={(e) =>
              set(
                'salary_min',
                e.target.value ? parseInt(e.target.value, 10) : undefined,
              )
            }
          />
          <span className="muted" aria-hidden="true">
            –
          </span>
          <label className="sr-only" htmlFor="filter-salary-max">
            Salaire max (€)
          </label>
          <input
            className="input"
            id="filter-salary-max"
            type="number"
            min={0}
            placeholder="max"
            value={filters.salary_max ?? ''}
            onChange={(e) =>
              set(
                'salary_max',
                e.target.value ? parseInt(e.target.value, 10) : undefined,
              )
            }
          />
        </div>
      </div>

      <div className="field filter-group">
        <label className="field__label" htmlFor="filter-location">
          Localisation
        </label>
        <input
          className="input"
          id="filter-location"
          type="text"
          value={filters.location ?? ''}
          onChange={(e) => set('location', e.target.value || undefined)}
        />
      </div>

      <div className="field filter-group">
        <label className="field__label" htmlFor="filter-board">
          Source
        </label>
        <select
          className="select"
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

      <div className="field filter-group">
        <label className="field__label" htmlFor="filter-since">
          Depuis
        </label>
        <input
          className="input"
          id="filter-since"
          type="date"
          value={filters.since ?? ''}
          onChange={(e) => set('since', e.target.value || undefined)}
        />
      </div>

      <div className="field filter-group">
        <label className="field__label" htmlFor="filter-confidence">
          Confiance min (%)
        </label>
        <input
          className="input"
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
      <div className="field filter-group">
        <label className="field__label" htmlFor="sort-field">
          Trier par
        </label>
        <select
          className="select"
          id="sort-field"
          value={filters.sort ?? 'date'}
          onChange={(e) => set('sort', e.target.value as SortField)}
        >
          {/* API contract: GET /api/jobs `sort` only accepts date|salary server-side. */}
          <option value="date">Date</option>
          <option value="salary">Salaire</option>
        </select>
      </div>

      <div className="field filter-group">
        <label className="field__label" htmlFor="sort-dir">
          Ordre
        </label>
        <select
          className="select"
          id="sort-dir"
          value={filters.sort_dir ?? 'desc'}
          onChange={(e) => set('sort_dir', e.target.value as SortDir)}
        >
          <option value="desc">Décroissant</option>
          <option value="asc">Croissant</option>
        </select>
      </div>

      <div className="filter-group filter-group--last">
        <label className="check" htmlFor="filter-show-expired">
          <input
            id="filter-show-expired"
            type="checkbox"
            checked={showExpired}
            onChange={(e) => onShowExpiredChange(e.target.checked)}
          />
          Afficher les offres expirées
        </label>
      </div>
    </aside>
  );
}

export { JobFiltersBar };

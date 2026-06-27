/**
 * API contract types for the job browser. These mirror the `GET /api/jobs`
 * response shape (snake_case, scoped to the active profile). Enum string values
 * are identical to the backend `kernel` package so the i18n dict can resolve
 * them directly via the `job.<field>.<value>` key pattern.
 *
 * Enum fields can be an empty string `""` when the extraction LLM could not
 * determine them — treated as "unknown" and not rendered (never `null`).
 */

export type ContractType = 'cdi' | 'cdd' | 'freelance' | 'interim';

export type RemotePolicy = 'on_site' | 'hybrid' | 'full_remote';

export type Seniority = 'entry' | 'mid' | 'senior' | 'lead' | 'exec';

export type WorkingDays = 'full_time' | 'part_time' | 'four_day';

/**
 * One row in the jobs list. Identity fields (`title`, `company`, `location`,
 * `url`) are captured verbatim from the search card during scraping and are
 * never translated; `company`/`location` may be empty for HTML-fallback boards.
 * Structured enum fields are rendered in French when present.
 */
export interface JobSummary {
  id: string;
  title: string;
  company: string;
  location: string;
  /** Link to the original posting; may be empty when the board omits it. */
  url: string;
  contractType: ContractType | '';
  remotePolicy: RemotePolicy | '';
  seniority: Seniority | '';
  workingDays: WorkingDays | '';
  skills: string[];
  /** Whole euros; null when the salary was not published. */
  salaryMin: number | null;
  salaryMax: number | null;
  /** Overall extraction parse-quality score, 0–100. */
  understandingScore: number;
}

/** Raw `GET /api/jobs` payload as returned by the backend (snake_case). */
export interface JobSummaryDto {
  id: string;
  title: string;
  company: string;
  location: string;
  url: string;
  contract_type: ContractType | '';
  remote_policy: RemotePolicy | '';
  seniority: Seniority | '';
  working_days: WorkingDays | '';
  skills: string[];
  salary_min: number | null;
  salary_max: number | null;
  understanding_score: number;
}

/** Maps a wire DTO to the camelCase domain shape used by the UI. */
export function toJobSummary(dto: JobSummaryDto): JobSummary {
  return {
    id: dto.id,
    title: dto.title,
    company: dto.company,
    location: dto.location,
    url: dto.url,
    contractType: dto.contract_type,
    remotePolicy: dto.remote_policy,
    seniority: dto.seniority,
    workingDays: dto.working_days,
    skills: dto.skills,
    salaryMin: dto.salary_min,
    salaryMax: dto.salary_max,
    understandingScore: dto.understanding_score,
  };
}

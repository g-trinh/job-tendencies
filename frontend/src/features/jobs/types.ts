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

/** Application kanban status — mirrors backend `kernel.ApplicationStatus`. */
export type ApplicationStatus =
  'saved' | 'applied' | 'interview' | 'offer' | 'rejected';

/** Sort field for the jobs list. */
export type SortField = 'date' | 'fit' | 'salary';

/** Sort direction for the jobs list. */
export type SortDir = 'asc' | 'desc';

/**
 * One source board for a job entry. A job may appear on multiple boards;
 * each yields one `JobSource`. `board_name` is the human-readable label
 * for the "found on:" display.
 */
export interface JobSource {
  board_id: string;
  source_url: string;
  board_name: string;
}

/**
 * Per-field extraction confidence map. Key is the field name (e.g. "contract_type"),
 * value is 0–100. Populated by the extraction LLM alongside `understanding_score`.
 */
export type FieldConfidence = Record<string, number>;

/**
 * Filter and sort state sent as query params to `GET /api/jobs`.
 * All fields are optional; omitted fields are not sent to the API.
 */
export interface JobFilters {
  skills?: string[];
  remote_policy?: RemotePolicy | '';
  contract_type?: ContractType | '';
  salary_min?: number | null;
  salary_max?: number | null;
  location?: string;
  board_id?: string;
  since?: string;
  confidence_min?: number | null;
  sort?: SortField;
  sort_dir?: SortDir;
}

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
  /** Current application status for this profile; null if not yet tracked. */
  applicationStatus: ApplicationStatus | null;
  /** Weighted fit score 0–100; null until the scoring pipeline runs. */
  fitScore: number | null;
  /** Source boards where this job was found. */
  sources: JobSource[];
  /** ISO-8601 date when this job was first scraped. */
  firstSeen: string | null;
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
  application_status: ApplicationStatus | null;
  fit_score: number | null;
  sources: JobSource[];
  first_seen: string | null;
}

/** Full job detail returned by `GET /api/jobs/{id}`. */
export interface JobDetail extends JobSummary {
  /** Full job description text (raw, as scraped). */
  description: string;
  /** Per-field extraction confidence map. */
  fieldConfidence: FieldConfidence;
  /** Linked recruiter contact id, if extracted. */
  contactId: string | null;
  /** ISO-8601 date when this job was last seen on its source board. */
  lastSeen: string;
  /** ISO-8601 date when this job was marked expired; null if still active. */
  expiredAt: string | null;
}

/** Raw `GET /api/jobs/{id}` payload (snake_case). */
export interface JobDetailDto extends JobSummaryDto {
  description: string;
  field_confidence: FieldConfidence;
  contact_id: string | null;
  last_seen: string;
  expired_at: string | null;
}

/** Maps a wire summary DTO to the camelCase domain shape used by the UI. */
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
    applicationStatus: dto.application_status,
    fitScore: dto.fit_score,
    sources: dto.sources,
    firstSeen: dto.first_seen,
  };
}

/** Maps a wire detail DTO to the camelCase domain shape used by the UI. */
export function toJobDetail(dto: JobDetailDto): JobDetail {
  return {
    ...toJobSummary(dto),
    description: dto.description,
    fieldConfidence: dto.field_confidence,
    contactId: dto.contact_id,
    lastSeen: dto.last_seen,
    expiredAt: dto.expired_at,
  };
}

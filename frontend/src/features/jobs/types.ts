/**
 * API contract types for the job browser. These mirror the `GET /api/jobs`
 * response shape (snake_case, scoped to the active profile). Enum string values
 * are identical to the backend `kernel` package so the i18n dict can resolve
 * them directly via the `job.<field>.<value>` key pattern.
 */

export type ContractType = 'cdi' | 'cdd' | 'freelance' | 'interim';

export type RemotePolicy = 'on_site' | 'hybrid' | 'full_remote';

export type Seniority = 'entry' | 'mid' | 'senior' | 'lead' | 'exec';

export type WorkingDays = 'full_time' | 'part_time' | 'four_day';

/** A board the job was found on, with the link to its original posting. */
export interface JobSource {
  sourceUrl: string;
}

/**
 * One row in the jobs list. Structured enum fields are rendered in French.
 * `title`/`company`/`location` are optional pending the backend decision on
 * capturing them from the search card (see design_changes); the UI renders
 * them when present and degrades gracefully when absent.
 */
export interface JobSummary {
  id: string;
  title?: string;
  company?: string;
  location?: string;
  contractType: ContractType | null;
  remotePolicy: RemotePolicy | null;
  seniority: Seniority | null;
  workingDays: WorkingDays | null;
  skills: string[];
  /** Whole euros; null when the salary was not published. */
  salaryMin: number | null;
  salaryMax: number | null;
  /** Overall extraction parse-quality score, 0–100. */
  understandingScore: number;
  /** Boards the job was found on; the first source backs the posting link. */
  sources: JobSource[];
}

/** Raw `GET /api/jobs` payload as returned by the backend (snake_case). */
export interface JobSummaryDto {
  id: string;
  title?: string;
  company?: string;
  location?: string;
  contract_type: ContractType | null;
  remote_policy: RemotePolicy | null;
  seniority: Seniority | null;
  working_days: WorkingDays | null;
  skills: string[];
  salary_min: number | null;
  salary_max: number | null;
  understanding_score: number;
  sources: { source_url: string }[];
}

/** Maps a wire DTO to the camelCase domain shape used by the UI. */
export function toJobSummary(dto: JobSummaryDto): JobSummary {
  return {
    id: dto.id,
    title: dto.title,
    company: dto.company,
    location: dto.location,
    contractType: dto.contract_type,
    remotePolicy: dto.remote_policy,
    seniority: dto.seniority,
    workingDays: dto.working_days,
    skills: dto.skills,
    salaryMin: dto.salary_min,
    salaryMax: dto.salary_max,
    understandingScore: dto.understanding_score,
    sources: dto.sources.map((s) => ({ sourceUrl: s.source_url })),
  };
}

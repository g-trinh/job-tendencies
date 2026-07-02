/**
 * API contract types for the Pipeline feature. Mirror
 * `backend/internal/handler/http/pipeline.go` response shapes.
 *
 * Per docs/plan/phase-6-api-contract.md: `GET /api/pipeline/runs` (list) has
 * NO per-board breakdown — only `GET /api/pipeline/runs/{id}` (detail) does.
 * Live per-board progress must poll the detail endpoint, not the list.
 */

export interface ScrapeRunSummaryDto {
  run_id: string;
  profile_id: string;
  trigger: string;
  status: string;
  created_at: string; // RFC3339
  finished_at?: string; // RFC3339, absent while running
}

export interface ScrapeRunBoardDto {
  board_id: string;
  status: string;
  pages_fetched: number;
  listings_captured: number;
  error?: string;
  started_at?: string;
  finished_at?: string;
}

export interface ScrapeRunDetailDto extends ScrapeRunSummaryDto {
  boards: ScrapeRunBoardDto[];
}

export interface ScrapeRunListResponseDto {
  runs: ScrapeRunSummaryDto[];
}

/** Terminal run statuses — polling stops once the run reaches one of these. */
export const TERMINAL_RUN_STATUSES = ['completed', 'failed', 'cancelled'];

export function isTerminalStatus(status: string): boolean {
  return TERMINAL_RUN_STATUSES.includes(status);
}

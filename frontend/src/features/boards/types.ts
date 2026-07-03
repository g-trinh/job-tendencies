/**
 * API contract types for the Boards feature. Mirror
 * `backend/internal/handler/http/boards.go` response/request shapes.
 */

export interface AdapterSpecDto {
  // Declarative scraper spec — shape owned by the LLM adapter generator.
  // Kept loose (unknown record) since the UI only needs to display it as
  // JSON for review, never to interpret individual fields.
  [key: string]: unknown;
}

export interface AdapterDto {
  id: string;
  status: 'draft' | 'approved' | string;
  fetch_mode: string;
  version: number;
  spec: AdapterSpecDto;
}

export interface BoardDto {
  id: string;
  name: string;
  base_url: string;
  enabled: boolean;
  adapter: AdapterDto | null;
}

export interface ScheduleDto {
  cron: string;
}

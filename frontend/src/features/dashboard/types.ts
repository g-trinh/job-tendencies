/**
 * API contract types for the Dashboard feature. Mirror
 * `backend/internal/handler/http/dashboard.go` response shapes.
 */

export interface SkillFrequencyDto {
  skill: string;
  count: number;
}

export interface SkillTrendDto {
  period: string; // ISO-8601
  skill: string;
  count: number;
}

export interface MatchDto {
  id: string;
  title: string;
  company: string;
  location: string;
  url: string;
  skills: string[];
  remote_policy: string;
  contract_type: string;
  salary_min: number | null;
  salary_max: number | null;
  weighted_score: number | null;
  passes_dealbreakers: boolean | null;
}

export interface StatsDto {
  total: number;
  new_today: number;
  new_this_week: number;
  pct_remote: number;
  avg_salary: number | null;
  top_contract_type: string;
}

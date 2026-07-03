/**
 * API contract types for the Profiles feature. Mirror `backend/internal/handler/http/profiles.go`
 * response/request shapes (snake_case on the wire). Enum values match `kernel` so the
 * i18n dict resolves them via `job.<field>.<value>` / `profile.<field>.<value>` keys.
 */
import type { ContractType, RemotePolicy, Seniority, WorkingDays } from '../jobs/types';

export interface ProfileConditionsDto {
  dealbreaker_contract_type: ContractType | null;
  dealbreaker_remote_policy: RemotePolicy | null;
  dealbreaker_salary_min: number | null;
  dealbreaker_required_skills: string[];
  preferred_skills: string[];
  preferred_max_office_days: number | null;
  preferred_location: string;
  preferred_working_days: WorkingDays | '';
}

export interface ProfileWeightsDto {
  preferred_skills: number;
  salary: number;
  location: number;
  office_days: number;
  working_days: number;
}

/** Raw `GET /api/profiles` / `GET /api/profiles/{id}` payload (snake_case). */
export interface ProfileDto {
  id: string;
  name: string;
  search_keywords: string[];
  location: string;
  is_active: boolean;
  skills: string[];
  seniority: Seniority | '';
  raw_experience: string;
  conditions: ProfileConditionsDto;
  weights: ProfileWeightsDto;
}

/** Domain shape used across the UI (camelCase). */
export interface Profile {
  id: string;
  name: string;
  searchKeywords: string[];
  location: string;
  isActive: boolean;
  skills: string[];
  seniority: Seniority | '';
  rawExperience: string;
  conditions: ProfileConditionsDto;
  weights: ProfileWeightsDto;
}

export function toProfile(dto: ProfileDto): Profile {
  return {
    id: dto.id,
    name: dto.name,
    searchKeywords: dto.search_keywords,
    location: dto.location,
    isActive: dto.is_active,
    skills: dto.skills,
    seniority: dto.seniority,
    rawExperience: dto.raw_experience,
    conditions: dto.conditions,
    weights: dto.weights,
  };
}

export const EMPTY_WEIGHTS: ProfileWeightsDto = {
  preferred_skills: 0,
  salary: 0,
  location: 0,
  office_days: 0,
  working_days: 0,
};

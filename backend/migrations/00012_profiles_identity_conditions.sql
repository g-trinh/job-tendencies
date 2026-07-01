-- +goose Up
-- Phase 3 profile extension: identity (skills, seniority), dealbreaker + preference
-- conditions, and fit-score weights are added as columns on the profile table.
-- This matches the infra/profiles flat-column projection; the ER diagram shows separate
-- entities (identity, profile_conditions, fit_weights) but the single-table layout is
-- chosen for simplicity while the user base is small (tech_debt.md).

ALTER TABLE profile
    -- Identity: extracted from LinkedIn PDF or manually maintained.
    ADD COLUMN skills    text[]  NOT NULL DEFAULT '{}',
    ADD COLUMN seniority text    NOT NULL DEFAULT '',

    -- Dealbreakers: hard filters that suppress a job from the browser.
    ADD COLUMN dealbreaker_contract_type    text,
    ADD COLUMN dealbreaker_remote_policy    text,
    ADD COLUMN dealbreaker_salary_min       bigint,
    ADD COLUMN dealbreaker_required_skills  text[] NOT NULL DEFAULT '{}',

    -- Preferences: soft inputs for the weighted fit score.
    ADD COLUMN preferred_skills            text[]  NOT NULL DEFAULT '{}',
    ADD COLUMN preferred_max_office_days   integer,
    ADD COLUMN preferred_location          text    NOT NULL DEFAULT '',
    ADD COLUMN preferred_working_days      text    NOT NULL DEFAULT '',

    -- Fit-score weights (integer percentages; soft components must sum to 100).
    ADD COLUMN weight_preferred_skills integer NOT NULL DEFAULT 40,
    ADD COLUMN weight_salary           integer NOT NULL DEFAULT 25,
    ADD COLUMN weight_location         integer NOT NULL DEFAULT 15,
    ADD COLUMN weight_office_days      integer NOT NULL DEFAULT 10,
    ADD COLUMN weight_working_days     integer NOT NULL DEFAULT 10;

-- +goose Down
ALTER TABLE profile
    DROP COLUMN weight_working_days,
    DROP COLUMN weight_office_days,
    DROP COLUMN weight_location,
    DROP COLUMN weight_salary,
    DROP COLUMN weight_preferred_skills,
    DROP COLUMN preferred_working_days,
    DROP COLUMN preferred_location,
    DROP COLUMN preferred_max_office_days,
    DROP COLUMN preferred_skills,
    DROP COLUMN dealbreaker_required_skills,
    DROP COLUMN dealbreaker_salary_min,
    DROP COLUMN dealbreaker_remote_policy,
    DROP COLUMN dealbreaker_contract_type,
    DROP COLUMN seniority,
    DROP COLUMN skills;

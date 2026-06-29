-- +goose Up
-- Scoring context: fit score per (job, profile) pair. Produced by the scoring
-- pipeline after extraction; consumed by the job-browser and dashboard.
-- passes_dealbreakers records the hard-filter gate result; weighted_score is the
-- 0-1 preference-weighted score; component_breakdown is per-component JSON.

CREATE TABLE job_score (
    job_id              text NOT NULL REFERENCES job(id) ON DELETE CASCADE,
    profile_id          text NOT NULL REFERENCES profile(id) ON DELETE CASCADE,
    passes_dealbreakers boolean          NOT NULL DEFAULT false,
    weighted_score      double precision NOT NULL DEFAULT 0,
    component_breakdown jsonb            NOT NULL DEFAULT '{}',
    scored_at           timestamptz      NOT NULL DEFAULT now(),
    PRIMARY KEY (job_id, profile_id)
);

CREATE INDEX job_score_profile ON job_score (profile_id);

-- +goose Down
DROP TABLE job_score;

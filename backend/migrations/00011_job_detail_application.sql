-- +goose Up
-- Phase 3 job detail: description is the full posting text, populated by the
-- extraction worker (P3-EX). Nullable until extracted.
ALTER TABLE job ADD COLUMN description text NOT NULL DEFAULT '';

-- job_application tracks a user's kanban state per (profile, job). Exactly one row
-- per (profile_id, job_id) pair; upsert on status change.
CREATE TABLE job_application (
    profile_id  text NOT NULL REFERENCES profile(id) ON DELETE CASCADE,
    job_id      text NOT NULL REFERENCES job(id) ON DELETE CASCADE,
    status      text NOT NULL,
    updated_at  timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (profile_id, job_id)
);

-- +goose Down
DROP TABLE job_application;
ALTER TABLE job DROP COLUMN description;

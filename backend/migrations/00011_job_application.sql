-- +goose Up
-- Adds description to job (populated by the extraction worker, P3-EX-*) and creates
-- the per-(profile, job) application kanban table used by the job browser.

ALTER TABLE job ADD COLUMN description text NOT NULL DEFAULT '';

CREATE TABLE job_application (
    profile_id text NOT NULL REFERENCES profile(id) ON DELETE CASCADE,
    job_id     text NOT NULL REFERENCES job(id) ON DELETE CASCADE,
    status     text NOT NULL,
    updated_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (profile_id, job_id)
);

-- +goose Down
DROP TABLE job_application;
ALTER TABLE job DROP COLUMN description;

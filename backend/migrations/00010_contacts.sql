-- +goose Up
-- Contacts context: recruiter records auto-populated from extraction and deduped
-- by email or LinkedIn URL. contact_id is then added to job as a nullable FK.

CREATE TABLE contact (
    id           text PRIMARY KEY DEFAULT gen_random_uuid()::text,
    name         text NOT NULL DEFAULT '',
    company      text NOT NULL DEFAULT '',
    email        text NOT NULL DEFAULT '',
    linkedin_url text NOT NULL DEFAULT '',
    phone        text NOT NULL DEFAULT '',
    notes        text NOT NULL DEFAULT '',
    tags         text[] NOT NULL DEFAULT '{}',
    -- dedup_key is the canonical identifier used for upsert: "email:<email>" when
    -- an email is present, otherwise "linkedin:<url>". Must be set before insert.
    dedup_key    text NOT NULL,
    created_at   timestamptz NOT NULL DEFAULT now(),
    updated_at   timestamptz NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX contact_dedup_key ON contact (dedup_key);

-- Link jobs to the recruiter contact that posted them.
ALTER TABLE job ADD COLUMN contact_id text REFERENCES contact(id) ON DELETE SET NULL;

-- +goose Down
ALTER TABLE job DROP COLUMN contact_id;
DROP TABLE contact;

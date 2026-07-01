-- +goose Up
-- Phase 3 job columns: fingerprint (cross-board dedup key) and expired_at.
-- contact_id is added in 00010 after the contact table exists.

-- fingerprint is the deterministic dedup key derived from the normalized
-- identity fields (title+company+location+salary). Nullable until computed by
-- the extraction worker (P3-EX-2).
ALTER TABLE job ADD COLUMN fingerprint text;
CREATE UNIQUE INDEX job_fingerprint ON job (fingerprint) WHERE fingerprint IS NOT NULL;

-- expired_at records when the listing was no longer found on the source board.
ALTER TABLE job ADD COLUMN expired_at timestamptz;

-- +goose Down
DROP INDEX job_fingerprint;
ALTER TABLE job DROP COLUMN fingerprint;
ALTER TABLE job DROP COLUMN expired_at;

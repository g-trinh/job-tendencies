-- +goose Up
-- Jobs schema: the structured Job aggregate produced by extraction and the JobSource
-- rows linking it to the raw listings it came from. Phase 2 creates one job per raw
-- listing (no dedup/merge/scoring); jobs are scoped to a profile via job_source →
-- raw_listing.profile_id. contact_id, fingerprint and expired_at are deferred.

CREATE TABLE job (
    id                  text PRIMARY KEY DEFAULT gen_random_uuid()::text,
    -- Identity facts captured verbatim from the search card (not LLM-extracted);
    -- canonical display fields and deterministic fingerprint inputs.
    title               text NOT NULL DEFAULT '',
    company             text NOT NULL DEFAULT '',
    location            text NOT NULL DEFAULT '',
    url                 text NOT NULL DEFAULT '',
    skills              text[] NOT NULL DEFAULT '{}',
    remote_policy       text NOT NULL DEFAULT '',
    office_days         integer NOT NULL DEFAULT 0,
    contract_type       text NOT NULL DEFAULT '',
    working_days        text NOT NULL DEFAULT '',
    salary_min          bigint,
    salary_max          bigint,
    seniority           text NOT NULL DEFAULT '',
    field_confidence    jsonb NOT NULL DEFAULT '{}',
    understanding_score integer NOT NULL DEFAULT 0,
    first_seen          timestamptz NOT NULL DEFAULT now(),
    last_seen           timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE job_source (
    job_id         text NOT NULL REFERENCES job(id) ON DELETE CASCADE,
    raw_listing_id text NOT NULL REFERENCES raw_listing(id) ON DELETE CASCADE,
    board_id       text NOT NULL REFERENCES board(id) ON DELETE CASCADE,
    source_url     text NOT NULL DEFAULT '',
    PRIMARY KEY (job_id, raw_listing_id)
);

CREATE INDEX job_source_raw_listing ON job_source (raw_listing_id);

-- +goose Down
DROP TABLE job_source;
DROP TABLE job;

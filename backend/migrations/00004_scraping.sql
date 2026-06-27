-- +goose Up
-- Scraping schema: verbatim raw listings (referenced into GCS) and the per-(board,
-- profile) high-water-mark that drives incremental crawling (pipeline.md §2).
-- Phase 2 omits scrape_run_board linkage on raw_listing; run tracking lands with the
-- pipeline-runs trigger (migration 00006) and is not required for the walking skeleton.

CREATE TABLE raw_listing (
    id                text PRIMARY KEY DEFAULT gen_random_uuid()::text,
    board_id          text NOT NULL REFERENCES board(id) ON DELETE CASCADE,
    profile_id        text NOT NULL REFERENCES profile(id) ON DELETE CASCADE,
    -- Identity facts captured verbatim from the search card (never LLM-inferred,
    -- never translated); the deterministic inputs to the job dedup fingerprint.
    title             text NOT NULL DEFAULT '',
    company           text NOT NULL DEFAULT '',
    location          text NOT NULL DEFAULT '',
    source_url        text NOT NULL,
    raw_ref           text NOT NULL,
    posted_at         timestamptz,
    content_hash      text NOT NULL,
    extraction_status text NOT NULL DEFAULT 'pending'
        CHECK (extraction_status IN ('pending', 'extracted')),
    captured_at       timestamptz NOT NULL DEFAULT now()
);

-- content_hash is the idempotency/dedup key within a board (pipeline.md §2 overlap skip).
CREATE UNIQUE INDEX raw_listing_board_content_hash
    ON raw_listing (board_id, content_hash);

CREATE TABLE scrape_high_water_mark (
    board_id         text NOT NULL REFERENCES board(id) ON DELETE CASCADE,
    profile_id       text NOT NULL REFERENCES profile(id) ON DELETE CASCADE,
    cursor_posted_at timestamptz NOT NULL,
    updated_at       timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (board_id, profile_id)
);

-- +goose Down
DROP TABLE scrape_high_water_mark;
DROP TABLE raw_listing;

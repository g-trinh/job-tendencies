-- +goose Up
-- Pipeline schema: a scrape_run records one pipeline execution. POST /api/pipeline/runs
-- inserts a queued on_demand run and publishes scrape.tick (pipeline.md §6). Per-board
-- progress (scrape_run_board) and run status transitions are deferred past Phase 2.

CREATE TABLE scrape_run (
    id         text PRIMARY KEY DEFAULT gen_random_uuid()::text,
    profile_id text NOT NULL REFERENCES profile(id) ON DELETE CASCADE,
    trigger    text NOT NULL CHECK (trigger IN ('on_demand', 'scheduled')),
    status     text NOT NULL DEFAULT 'queued',
    created_at timestamptz NOT NULL DEFAULT now()
);

-- +goose Down
DROP TABLE scrape_run;

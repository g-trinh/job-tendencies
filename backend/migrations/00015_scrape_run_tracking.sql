-- +goose Up
-- Scrape run tracking: add completion timestamp to scrape_run and create
-- scrape_run_board for per-board progress and counts (data-model.md Scraping,
-- P3-SCR-5). The scrape_run_board.status check mirrors the run lifecycle states.

ALTER TABLE scrape_run ADD COLUMN finished_at timestamptz;

CREATE TABLE scrape_run_board (
    id                text PRIMARY KEY DEFAULT gen_random_uuid()::text,
    run_id            text NOT NULL REFERENCES scrape_run(id) ON DELETE CASCADE,
    board_id          text NOT NULL REFERENCES board(id) ON DELETE CASCADE,
    status            text NOT NULL DEFAULT 'running'
        CHECK (status IN ('running', 'done', 'error')),
    pages_fetched     int  NOT NULL DEFAULT 0,
    listings_captured int  NOT NULL DEFAULT 0,
    error             text,
    started_at        timestamptz NOT NULL DEFAULT now(),
    finished_at       timestamptz,
    UNIQUE (run_id, board_id)
);

-- +goose Down
DROP TABLE scrape_run_board;
ALTER TABLE scrape_run DROP COLUMN finished_at;

-- +goose Up
-- schedule holds the single global cron expression applied to Cloud Scheduler.
-- There is exactly one row; it is upsert-managed at the application layer.
CREATE TABLE schedule (
    id         integer PRIMARY KEY DEFAULT 1 CHECK (id = 1),
    cron       text NOT NULL,
    updated_at timestamptz NOT NULL DEFAULT now()
);

-- +goose Down
DROP TABLE schedule;

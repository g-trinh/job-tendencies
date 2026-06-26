-- +goose Up
-- Board-manager schema: a job board source and its declarative scraping adapter.
-- Seeds Welcome to the Jungle (WTTJ) with a hand-written, approved json_api adapter
-- (Phase 2 skips LLM adapter generation). The adapter.spec mirrors the AdapterSpec
-- struct (internal/domain/llm) and is validated against AdapterSpec.Validate.

CREATE TABLE board (
    id       text PRIMARY KEY DEFAULT gen_random_uuid()::text,
    name     text NOT NULL,
    base_url text NOT NULL,
    enabled  boolean NOT NULL DEFAULT true
);

CREATE TABLE adapter (
    id         text PRIMARY KEY DEFAULT gen_random_uuid()::text,
    board_id   text NOT NULL REFERENCES board(id) ON DELETE CASCADE,
    status     text NOT NULL CHECK (status IN ('draft', 'approved')),
    fetch_mode text NOT NULL CHECK (fetch_mode IN ('json_api', 'html')),
    spec       jsonb NOT NULL,
    version    integer NOT NULL DEFAULT 1
);

-- At most one approved adapter per board.
CREATE UNIQUE INDEX adapter_one_approved_per_board
    ON adapter (board_id)
    WHERE status = 'approved';

INSERT INTO board (id, name, base_url, enabled) VALUES
    ('b0000000-0000-0000-0000-000000000001', 'Welcome to the Jungle',
     'https://www.welcometothejungle.com', true);

-- ASSUMPTION (Open Question #1): the WTTJ search endpoint + param names below are
-- the documented example from docs/architecture/pipeline.md §1, reverse-engineering
-- deferred to dev verification (P2-1). The slice is wired end-to-end against this
-- shape; the url_template / param_map / result paths are the single point to correct
-- once the live endpoint is confirmed. listing.fetch=use_search_payload keeps the
-- walking skeleton to a single HTTP call per page (no per-listing detail fetch).
INSERT INTO adapter (id, board_id, status, fetch_mode, spec, version) VALUES
    ('a0000000-0000-0000-0000-000000000001',
     'b0000000-0000-0000-0000-000000000001',
     'approved', 'json_api',
     '{
       "board": "welcometothejungle",
       "fetch_mode": "json_api",
       "search": {
         "url_template": "https://api.wttj.co/v2/search?query={keywords}&aroundQuery={location}&page={page}",
         "method": "GET",
         "param_map": {
           "keywords": "profile.search.keywords",
           "location": "profile.search.location"
         },
         "pagination": {"kind": "query_param", "param": "page", "start": 1},
         "result_node_path": "$.jobs[*]",
         "result_fields": {
           "listing_url": "$.url",
           "title": "$.name",
           "company": "$.organization.name",
           "location": "$.office.city",
           "posted_at": "$.published_at",
           "external_id": "$.id"
         }
       },
       "listing": {"fetch": "use_search_payload", "raw_capture": "full_response"},
       "incremental": {"cursor_field": "posted_at", "overlap_buffer": "36h", "safety_max_pages": 20}
     }'::jsonb,
     1);

-- +goose Down
DROP TABLE adapter;
DROP TABLE board;

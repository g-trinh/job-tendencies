-- +goose Up
-- Seed two public, no-auth, JSON job boards (Remotive, Arbeitnow) with correct,
-- pre-approved, enabled adapters so an on-demand pipeline run scrapes them
-- immediately, with no LLM adapter generation required. This unblocks scraping
-- after the previous LLM-generated WTTJ adapter hallucinated a non-existent host
-- (WTTJ itself is untouched here).
--
-- Endpoints verified live, public, no API key, GET JSON (see feature docs):
--   Remotive:  GET https://remotive.com/api/remote-jobs?search=go
--              jobs array at $.jobs; no real pagination, rate-limited ~4 calls/day,
--              so safety_max_pages=1 and the url_template carries no {page}
--              placeholder (kept byte-identical to the verified URL).
--   Arbeitnow: GET https://www.arbeitnow.com/api/job-board-api?page=1
--              jobs array at $.data; paginates via ?page=N.
--
-- Both boards ship enabled with an approved adapter (mirrors migration 00002's
-- hand-written WTTJ seed pattern); board/adapter are global, not profile-scoped
-- (see migration 00002/00004 schema).

INSERT INTO board (id, name, base_url, enabled) VALUES
    ('b0000000-0000-0000-0000-000000000005', 'Remotive',
     'https://remotive.com', true),
    ('b0000000-0000-0000-0000-000000000006', 'Arbeitnow',
     'https://www.arbeitnow.com', true);

INSERT INTO adapter (id, board_id, status, fetch_mode, spec, version) VALUES
    ('a0000000-0000-0000-0000-000000000002',
     'b0000000-0000-0000-0000-000000000005',
     'approved', 'json_api',
     '{
       "board": "remotive",
       "fetch_mode": "json_api",
       "search": {
         "url_template": "https://remotive.com/api/remote-jobs?search=go",
         "method": "GET",
         "param_map": {},
         "pagination": {"kind": "query_param", "param": "page", "start": 1},
         "result_node_path": "$.jobs",
         "result_fields": {
           "listing_url": "$.url",
           "title": "$.title",
           "company": "$.company_name",
           "location": "$.candidate_required_location",
           "posted_at": "$.publication_date",
           "external_id": "$.id"
         }
       },
       "listing": {"fetch": "use_search_payload", "raw_capture": "$"},
       "incremental": {"cursor_field": "posted_at", "overlap_buffer": "36h", "safety_max_pages": 1}
     }'::jsonb,
     1),
    ('a0000000-0000-0000-0000-000000000003',
     'b0000000-0000-0000-0000-000000000006',
     'approved', 'json_api',
     '{
       "board": "arbeitnow",
       "fetch_mode": "json_api",
       "search": {
         "url_template": "https://www.arbeitnow.com/api/job-board-api?page={page}",
         "method": "GET",
         "param_map": {},
         "pagination": {"kind": "query_param", "param": "page", "start": 1},
         "result_node_path": "$.data",
         "result_fields": {
           "listing_url": "$.url",
           "title": "$.title",
           "company": "$.company_name",
           "location": "$.location",
           "posted_at": "$.created_at",
           "external_id": "$.slug"
         }
       },
       "listing": {"fetch": "use_search_payload", "raw_capture": "$"},
       "incremental": {"cursor_field": "posted_at", "overlap_buffer": "36h", "safety_max_pages": 5}
     }'::jsonb,
     1);

-- +goose Down
DELETE FROM adapter
WHERE id IN (
    'a0000000-0000-0000-0000-000000000002',
    'a0000000-0000-0000-0000-000000000003'
);

DELETE FROM board
WHERE id IN (
    'b0000000-0000-0000-0000-000000000005',
    'b0000000-0000-0000-0000-000000000006'
);

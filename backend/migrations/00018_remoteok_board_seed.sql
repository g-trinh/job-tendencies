-- +goose Up
-- Seed RemoteOK as a third public, no-auth, JSON job board with a correct,
-- pre-approved, enabled adapter, following the exact pattern established in
-- migration 00017_public_board_seed.sql (global board, adapter.status='approved').
--
-- Endpoint verified live, public, no API key, GET JSON:
--   GET https://remoteok.com/api
--   Returns a JSON array at the document root (no wrapper key); no real
--   pagination, so safety_max_pages=1 and the url_template carries no {page}
--   placeholder (byte-identical to the verified URL), matching how 00017
--   handled Remotive's single-page case.
--
-- RemoteOK requires a descriptive User-Agent header on requests (see the
-- companion fetcher change in internal/infra/scraping/fetcher.go) — it
-- returns 403 to requests with no User-Agent at all. That is a fetcher-level
-- default applied to every board, not something this adapter spec controls.
--
-- The first element of the response array is a legal/disclaimer object
-- (e.g. {"legal": "..."}), not a job. It has no "id" field, unlike every
-- real job entry. result_node_path uses the gjson query filter "$.#(id!=)#"
-- to select only array elements with a non-empty "id", cleanly excluding the
-- legal notice without any bespoke parser change (see
-- internal/infra/scraping/fetcher.go normalizeJSONPath, which forwards this
-- path verbatim to gjson after stripping the "$." prefix).
INSERT INTO board (id, name, base_url, enabled) VALUES
    ('b0000000-0000-0000-0000-000000000007', 'RemoteOK',
     'https://remoteok.com', true);

INSERT INTO adapter (id, board_id, status, fetch_mode, spec, version) VALUES
    ('a0000000-0000-0000-0000-000000000004',
     'b0000000-0000-0000-0000-000000000007',
     'approved', 'json_api',
     '{
       "board": "remoteok",
       "fetch_mode": "json_api",
       "search": {
         "url_template": "https://remoteok.com/api",
         "method": "GET",
         "param_map": {},
         "pagination": {"kind": "query_param", "param": "page", "start": 1},
         "result_node_path": "$.#(id!=)#",
         "result_fields": {
           "listing_url": "$.url",
           "title": "$.position",
           "company": "$.company",
           "location": "$.location",
           "posted_at": "$.date",
           "external_id": "$.id"
         }
       },
       "listing": {"fetch": "use_search_payload", "raw_capture": "$"},
       "incremental": {"cursor_field": "posted_at", "overlap_buffer": "36h", "safety_max_pages": 1}
     }'::jsonb,
     1);

-- +goose Down
DELETE FROM adapter
WHERE id = 'a0000000-0000-0000-0000-000000000004';

DELETE FROM board
WHERE id = 'b0000000-0000-0000-0000-000000000007';

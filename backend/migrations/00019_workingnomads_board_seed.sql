-- +goose Up
-- Seed Working Nomads as a fourth public, no-auth, JSON job board with a correct,
-- pre-approved, enabled adapter, following the exact pattern established in
-- migration 00017_public_board_seed.sql (global board, adapter.status='approved')
-- and migration 00018_remoteok_board_seed.sql (single-page root-array response).
--
-- Endpoint verified live, public, no API key, GET JSON:
--   GET https://www.workingnomads.com/api/exposed_jobs/
-- Returns a JSON array at the document root; every element is a real job (no
-- legal/disclaimer element like RemoteOK's feed), so no filter is needed on
-- result_node_path — "$.@this" selects the root array itself. No real
-- pagination, so safety_max_pages=1 and the url_template carries no {page}
-- placeholder (byte-identical to the verified URL).
--
-- Working Nomads jobs carry no numeric/opaque "id" field, unlike Remotive,
-- Arbeitnow and RemoteOK. The job's own "url" is unique per listing and
-- stable across scrapes, so it is used as both listing_url and external_id
-- (the dedup/incremental key) here.
INSERT INTO board (id, name, base_url, enabled) VALUES
    ('b0000000-0000-0000-0000-000000000008', 'Working Nomads',
     'https://www.workingnomads.com', true);

INSERT INTO adapter (id, board_id, status, fetch_mode, spec, version) VALUES
    ('a0000000-0000-0000-0000-000000000005',
     'b0000000-0000-0000-0000-000000000008',
     'approved', 'json_api',
     '{
       "board": "workingnomads",
       "fetch_mode": "json_api",
       "search": {
         "url_template": "https://www.workingnomads.com/api/exposed_jobs/",
         "method": "GET",
         "param_map": {},
         "pagination": {"kind": "query_param", "param": "page", "start": 1},
         "result_node_path": "$.@this",
         "result_fields": {
           "listing_url": "$.url",
           "title": "$.title",
           "company": "$.company_name",
           "location": "$.location",
           "posted_at": "$.pub_date",
           "external_id": "$.url"
         }
       },
       "listing": {"fetch": "use_search_payload", "raw_capture": "$"},
       "incremental": {"cursor_field": "posted_at", "overlap_buffer": "36h", "safety_max_pages": 1}
     }'::jsonb,
     1);

-- +goose Down
DELETE FROM adapter
WHERE id = 'a0000000-0000-0000-0000-000000000005';

DELETE FROM board
WHERE id = 'b0000000-0000-0000-0000-000000000008';

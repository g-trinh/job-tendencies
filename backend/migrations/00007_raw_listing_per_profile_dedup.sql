-- +goose Up
-- The content-hash dedup key must include profile_id: two profiles can independently
-- capture the same listing from the same board, and each capture feeds its own
-- per-(board, profile) high-water-mark. The old (board_id, content_hash) unique index
-- let the first profile's capture block the second profile's. Widen it to
-- (board_id, profile_id, content_hash) so dedup is scoped per (board, profile), matching
-- the high-water-mark model (pipeline.md §2).

DROP INDEX raw_listing_board_content_hash;

CREATE UNIQUE INDEX raw_listing_board_profile_content_hash
    ON raw_listing (board_id, profile_id, content_hash);

-- +goose Down
DROP INDEX raw_listing_board_profile_content_hash;

CREATE UNIQUE INDEX raw_listing_board_content_hash
    ON raw_listing (board_id, content_hash);

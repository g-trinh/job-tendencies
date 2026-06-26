-- +goose Up
-- Profiles schema: the search persona that scopes scoped resources and supplies the
-- scraper's board-side filtering (keywords + location). Phase 2 seeds one default
-- active profile (hardcoded keywords/location); identity/conditions/weights deferred.

CREATE TABLE profile (
    id              text PRIMARY KEY DEFAULT gen_random_uuid()::text,
    name            text NOT NULL,
    search_keywords text[] NOT NULL DEFAULT '{}',
    location        text NOT NULL DEFAULT '',
    is_active       boolean NOT NULL DEFAULT false
);

-- Enforce "exactly one active profile at a time": at most one row with is_active.
CREATE UNIQUE INDEX profile_single_active
    ON profile ((is_active))
    WHERE is_active = true;

INSERT INTO profile (id, name, search_keywords, location, is_active) VALUES
    ('p0000000-0000-0000-0000-000000000001', 'Default',
     ARRAY['software engineer', 'backend'], 'Paris', true);

-- +goose Down
DROP TABLE profile;

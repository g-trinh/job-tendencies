-- +goose Up
-- Seed the three missing default boards: Indeed, LinkedIn, Glassdoor.
-- WTTJ was seeded in 00002. All four boards ship enabled with no adapter yet;
-- adapters are generated in P3-BO-2 and only approved adapters are crawled.

INSERT INTO board (id, name, base_url, enabled) VALUES
    ('b0000000-0000-0000-0000-000000000002', 'Indeed',
     'https://www.indeed.com', true),
    ('b0000000-0000-0000-0000-000000000003', 'LinkedIn',
     'https://www.linkedin.com/jobs', true),
    ('b0000000-0000-0000-0000-000000000004', 'Glassdoor',
     'https://www.glassdoor.com/Job', true);

-- +goose Down
DELETE FROM board
WHERE id IN (
    'b0000000-0000-0000-0000-000000000002',
    'b0000000-0000-0000-0000-000000000003',
    'b0000000-0000-0000-0000-000000000004'
);

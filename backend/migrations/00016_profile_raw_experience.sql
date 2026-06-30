-- +goose Up
-- Adds raw_experience to store the verbatim work experience block extracted from
-- a LinkedIn PDF import. Column is text to hold arbitrary length experience text.
ALTER TABLE profile ADD COLUMN raw_experience text NOT NULL DEFAULT '';

-- +goose Down
ALTER TABLE profile DROP COLUMN raw_experience;

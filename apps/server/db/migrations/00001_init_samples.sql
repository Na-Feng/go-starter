-- +goose Up
CREATE TABLE IF NOT EXISTS samples (
    id         TEXT PRIMARY KEY,
    title      TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- +goose Down
DROP TABLE IF EXISTS samples;

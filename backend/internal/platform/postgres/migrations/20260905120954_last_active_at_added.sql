-- internal/platform/postgres/migrations/00xx_last_active_at.sql

-- +goose Up
ALTER TABLE users ADD COLUMN IF NOT EXISTS last_active_at TIMESTAMPTZ;

-- +goose Down
ALTER TABLE users DROP COLUMN IF EXISTS last_active_at;
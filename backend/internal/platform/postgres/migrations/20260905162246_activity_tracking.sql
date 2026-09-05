-- internal/platform/postgres/migrations/00xx_activity_tracking.sql

-- +goose Up
ALTER TABLE borrow_requests ADD COLUMN IF NOT EXISTS updated_at TIMESTAMPTZ NOT NULL DEFAULT now();
ALTER TABLE borrow_requests ADD COLUMN IF NOT EXISTS accepted_at TIMESTAMPTZ;

-- +goose Down
ALTER TABLE borrow_requests DROP COLUMN IF EXISTS updated_at;
ALTER TABLE borrow_requests DROP COLUMN IF EXISTS accepted_at;
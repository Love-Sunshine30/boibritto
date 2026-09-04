-- internal/platform/postgres/migrations/00xx_handoff_confirmation.sql

-- +goose Up
ALTER TABLE borrow_requests ADD COLUMN IF NOT EXISTS owner_confirmed BOOLEAN NOT NULL DEFAULT false;
ALTER TABLE borrow_requests ADD COLUMN IF NOT EXISTS borrower_confirmed BOOLEAN NOT NULL DEFAULT false;

-- +goose Down
ALTER TABLE borrow_requests DROP COLUMN IF EXISTS owner_confirmed;
ALTER TABLE borrow_requests DROP COLUMN IF EXISTS borrower_confirmed;
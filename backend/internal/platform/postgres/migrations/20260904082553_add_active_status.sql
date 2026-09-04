-- internal/platform/postgres/migrations/00xx_add_active_status.sql

-- +goose Up
ALTER TABLE borrow_requests DROP CONSTRAINT chk_borrow_requests_status;
ALTER TABLE borrow_requests ADD CONSTRAINT chk_borrow_requests_status
  CHECK (status = ANY (ARRAY['pending', 'accepted', 'rejected', 'active', 'returned']));

-- +goose Down
ALTER TABLE borrow_requests DROP CONSTRAINT chk_borrow_requests_status;
ALTER TABLE borrow_requests ADD CONSTRAINT chk_borrow_requests_status
  CHECK (status = ANY (ARRAY['pending', 'accepted', 'rejected', 'returned']));
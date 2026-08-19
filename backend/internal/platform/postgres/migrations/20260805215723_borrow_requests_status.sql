-- +goose Up
-- +goose StatementBegin

-- 1. Remove the old boolean default value
ALTER TABLE borrow_requests 
  ALTER COLUMN status DROP DEFAULT;

-- 2. Convert boolean values to the new string status
--    (true -> 'accepted', false -> 'pending')
ALTER TABLE borrow_requests 
  ALTER COLUMN status TYPE VARCHAR(20) 
  USING (
    CASE 
      WHEN status = TRUE THEN 'accepted' 
      ELSE 'pending' 
    END
  );

-- 3. Set the new default and enforce allowed values
ALTER TABLE borrow_requests 
  ALTER COLUMN status SET DEFAULT 'pending',
  ADD CONSTRAINT chk_borrow_requests_status 
    CHECK (status IN ('pending', 'accepted', 'rejected'));

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

ALTER TABLE borrow_requests 
  DROP CONSTRAINT IF EXISTS chk_borrow_requests_status;

ALTER TABLE borrow_requests 
  ALTER COLUMN status TYPE BOOLEAN 
  USING (
    CASE 
      WHEN status = 'accepted' THEN TRUE 
      ELSE FALSE 
    END
  );

ALTER TABLE borrow_requests 
  ALTER COLUMN status SET DEFAULT FALSE;

-- +goose StatementEnd

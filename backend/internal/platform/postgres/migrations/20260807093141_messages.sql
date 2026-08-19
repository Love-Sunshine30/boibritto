-- +goose Up
-- +goose StatementBegin
CREATE TABLE messages (
  id SERIAL PRIMARY KEY,
  request_id INT NOT NULL REFERENCES borrow_requests(id) ON DELETE CASCADE,
  sender_id  INT NOT NULL REFERENCES users(id),
  body       TEXT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  read_at    TIMESTAMPTZ
);

CREATE INDEX idx_messages_request_created ON messages(request_id, created_at);
CREATE INDEX idx_messages_unread ON messages(request_id, sender_id, read_at) WHERE read_at IS NULL;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_messages_unread;
DROP INDEX IF EXISTS idx_messages_request_created;
DROP TABLE IF EXISTS messages;
-- +goose StatementEnd
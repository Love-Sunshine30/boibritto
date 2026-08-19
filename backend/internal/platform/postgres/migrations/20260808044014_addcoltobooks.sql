-- +goose Up
-- +goose StatementBegin
ALTER TABLE books ADD COLUMN cover_url TEXT;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE books ADD COLUMN isbn TEXT;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE books DROP COLUMN isbn;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE books DROP COLUMN cover_url;
-- +goose StatementEnd
-- internal/platform/postgres/migrations/00xx_book_forum_posts.sql

-- +goose Up
CREATE TABLE book_forum_posts (
    id         SERIAL PRIMARY KEY,
    book_id    INTEGER NOT NULL REFERENCES books(id) ON DELETE CASCADE,
    user_id    INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    body       TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_book_forum_posts_book_id ON book_forum_posts(book_id, created_at DESC);

-- +goose Down
DROP TABLE IF EXISTS book_forum_posts;
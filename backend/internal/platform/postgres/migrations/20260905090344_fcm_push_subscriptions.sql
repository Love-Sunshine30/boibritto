-- internal/platform/postgres/migrations/00xx_fcm_push_subscriptions.sql

-- +goose Up
-- Original table modeled Web Push (VAPID) subscriptions. Since we're
-- consolidating onto FCM for both Web and Android push (see architecture
-- notes: FCM covers both platforms, so a single push_subscriptions shape
-- covering both), drop the old shape and recreate for FCM tokens.
DROP TABLE IF EXISTS push_subscriptions;

CREATE TABLE push_subscriptions (
    id         SERIAL PRIMARY KEY,
    user_id    INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    platform   TEXT NOT NULL CHECK (platform IN ('web', 'android')),
    fcm_token  TEXT NOT NULL UNIQUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_push_subscriptions_user_id ON push_subscriptions(user_id);

-- +goose Down
DROP TABLE IF EXISTS push_subscriptions;
-- internal/platform/postgres/migrations/00xx_firebase_auth_users.sql

-- +goose Up
ALTER TABLE users ADD COLUMN IF NOT EXISTS firebase_uid TEXT UNIQUE;
ALTER TABLE users ADD COLUMN IF NOT EXISTS is_admin BOOLEAN NOT NULL DEFAULT false;

-- Firebase owns password verification now — this column is kept (not dropped)
-- only so cmd/migrate_users_to_firebase can still read existing bcrypt hashes
-- when importing pre-existing users into Firebase Auth. Drop it in a later
-- migration once that import has run and been verified.
ALTER TABLE users ALTER COLUMN password_hash DROP NOT NULL;

-- Firebase Auth has no concept of a WhatsApp number, so it can't be known at
-- JIT-provisioning time (first sign-in). Nullable here; enforced instead at
-- the application level via a required profile-completion step before a user
-- can list a book (see internal/profile and internal/books.Service).
ALTER TABLE users ALTER COLUMN whatsapp_number DROP NOT NULL;

-- +goose Down
ALTER TABLE users ALTER COLUMN whatsapp_number SET NOT NULL;
ALTER TABLE users ALTER COLUMN password_hash SET NOT NULL;
ALTER TABLE users DROP COLUMN IF EXISTS is_admin;
ALTER TABLE users DROP COLUMN IF EXISTS firebase_uid;
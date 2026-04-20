-- +goose Up
ALTER TABLE persons ADD COLUMN password_hash TEXT;
ALTER TABLE persons ADD COLUMN password_reset_token TEXT UNIQUE;
ALTER TABLE persons ADD COLUMN password_reset_expires_at TIMESTAMP WITH TIME ZONE;
CREATE TABLE sessions (
    id UUID PRIMARY KEY,
    person_id UUID NOT NULL REFERENCES persons(id) ON DELETE CASCADE,
    expires_at TIMESTAMP WITH TIME ZONE NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- +goose Down
DROP TABLE sessions;
ALTER TABLE persons DROP COLUMN password_hash, DROP COLUMN password_reset_token, DROP COLUMN password_reset_expires_at;

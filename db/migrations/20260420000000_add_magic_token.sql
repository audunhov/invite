-- +goose Up
ALTER TABLE invitees ADD COLUMN magic_token UUID NOT NULL DEFAULT gen_random_uuid();
CREATE UNIQUE INDEX idx_invitees_magic_token ON invitees(magic_token);

-- +goose Down
DROP INDEX idx_invitees_magic_token;
ALTER TABLE invitees DROP COLUMN magic_token;

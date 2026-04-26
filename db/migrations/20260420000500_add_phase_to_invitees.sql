-- +goose Up
ALTER TABLE invitees ADD COLUMN phase_id UUID REFERENCES invite_phases(id);

-- +goose Down
ALTER TABLE invitees DROP COLUMN phase_id;

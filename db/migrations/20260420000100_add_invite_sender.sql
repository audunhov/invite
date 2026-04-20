-- +goose Up
ALTER TABLE invites ADD COLUMN from_person_id UUID REFERENCES persons(id);

-- +goose Down
ALTER TABLE invites DROP COLUMN from_person_id;

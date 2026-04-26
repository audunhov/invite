-- +goose Up
CREATE TABLE tags (
    id UUID PRIMARY KEY,
    name TEXT NOT NULL UNIQUE,
    color TEXT NOT NULL DEFAULT '#6366f1',
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE TABLE invite_tags (
    invite_id UUID NOT NULL REFERENCES invites(id) ON DELETE CASCADE,
    tag_id UUID NOT NULL REFERENCES tags(id) ON DELETE CASCADE,
    PRIMARY KEY (invite_id, tag_id)
);

-- +goose Down
DROP TABLE invite_tags;
DROP TABLE tags;

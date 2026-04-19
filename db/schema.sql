CREATE TABLE persons (
    id UUID PRIMARY KEY,
    email TEXT UNIQUE NOT NULL,
    name TEXT NOT NULL
);

CREATE TABLE groups (
    id UUID PRIMARY KEY,
    name TEXT NOT NULL,
    description TEXT
);

CREATE TABLE group_members (
    id UUID PRIMARY KEY,
    contact_id UUID NOT NULL REFERENCES persons(id),
    group_id UUID NOT NULL REFERENCES groups(id)
);

CREATE TABLE invites (
    id UUID PRIMARY KEY,
    title TEXT NOT NULL,
    description TEXT,
    "from" TIMESTAMP WITH TIME ZONE NOT NULL,
    "to" TIMESTAMP WITH TIME ZONE,
    duration INTERVAL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL,
    status TEXT NOT NULL DEFAULT 'pending'
);

CREATE TABLE invitees (
    id UUID PRIMARY KEY,
    invite_id UUID NOT NULL REFERENCES invites(id),
    contact_id UUID NOT NULL REFERENCES persons(id),
    state TEXT NOT NULL DEFAULT 'pending',
    created_at TIMESTAMP WITH TIME ZONE NOT NULL,
    UNIQUE(invite_id, contact_id)
);

CREATE TABLE invite_phases (
    id UUID PRIMARY KEY,
    invite_id UUID NOT NULL REFERENCES invites(id),
    "order" INT NOT NULL,
    strategy_kind TEXT NOT NULL,
    strategy_config JSONB NOT NULL,
    UNIQUE(invite_id, "order")
);

CREATE TABLE invite_phase_state (
    phase_id UUID PRIMARY KEY REFERENCES invite_phases(id),
    status TEXT NOT NULL DEFAULT 'active',
    next_check_at TIMESTAMP WITH TIME ZONE,
    data JSONB NOT NULL DEFAULT '{}'
);

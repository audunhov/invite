-- name: GetPerson :one
SELECT * FROM persons WHERE id = $1;

-- name: CreatePerson :one
INSERT INTO persons (id, email, name, password_hash) VALUES ($1, $2, $3, $4) RETURNING *;

-- name: ListPersons :many
SELECT * FROM persons;

-- name: UpdatePerson :one
UPDATE persons
SET 
    email = COALESCE(sqlc.narg('email'), email),
    name = COALESCE(sqlc.narg('name'), name),
    password_hash = COALESCE(sqlc.narg('password_hash'), password_hash)
WHERE id = sqlc.arg('id')
RETURNING *;

-- name: DeletePerson :exec
DELETE FROM persons WHERE id = $1;

-- name: GetInvite :one
SELECT * FROM invites WHERE id = $1;

-- name: ListInvites :many
SELECT * FROM invites;

-- name: CreateInvite :one
INSERT INTO invites (id, title, description, "from", "to", duration, created_at, status, from_person_id)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
RETURNING *;

-- name: UpdateInvite :one
UPDATE invites
SET 
    title = COALESCE(sqlc.narg('title'), title),
    description = COALESCE(sqlc.narg('description'), description),
    "from" = COALESCE(sqlc.narg('from'), "from"),
    "to" = COALESCE(sqlc.narg('to'), "to"),
    duration = COALESCE(sqlc.narg('duration'), duration),
    status = COALESCE(sqlc.narg('status'), status),
    from_person_id = COALESCE(sqlc.narg('from_person_id'), from_person_id)
WHERE id = sqlc.arg('id')
RETURNING *;

-- name: DeleteInvite :exec
DELETE FROM invites WHERE id = $1;

-- name: GetGroup :one
SELECT * FROM groups WHERE id = $1;

-- name: ListGroups :many
SELECT * FROM groups;

-- name: CreateGroup :one
INSERT INTO groups (id, name, description)
VALUES ($1, $2, $3)
RETURNING *;

-- name: UpdateGroup :one
UPDATE groups
SET 
    name = COALESCE(sqlc.narg('name'), name),
    description = COALESCE(sqlc.narg('description'), description)
WHERE id = sqlc.arg('id')
RETURNING *;

-- name: DeleteGroup :exec
DELETE FROM groups WHERE id = $1;

-- name: ListGroupMembers :many
SELECT p.* FROM persons p
JOIN group_members gm ON p.id = gm.contact_id
WHERE gm.group_id = $1;

-- name: AddGroupMember :exec
INSERT INTO group_members (id, contact_id, group_id)
VALUES ($1, $2, $3);

-- name: RemoveGroupMember :exec
DELETE FROM group_members WHERE group_id = $1 AND contact_id = $2;

-- name: ListInvitePhases :many
SELECT * FROM invite_phases WHERE invite_id = $1 ORDER BY "order" ASC;

-- name: CreateInvitePhase :one
INSERT INTO invite_phases (id, invite_id, "order", strategy_kind, strategy_config)
VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: DeleteInvitePhase :exec
DELETE FROM invite_phases WHERE id = $1 AND invite_id = $2;

-- name: GetFirstInvitePhase :one
SELECT * FROM invite_phases 
WHERE invite_id = $1 
ORDER BY "order" ASC 
LIMIT 1;

-- name: GetInvitePhase :one
SELECT * FROM invite_phases WHERE id = $1;

-- name: GetInviteesStatus :many
SELECT p.id, p.email, p.name, i.created_at AS invited_at, i.state AS status, i.magic_token
FROM invitees i
JOIN persons p ON i.contact_id = p.id
WHERE i.invite_id = $1
ORDER BY i.created_at ASC;

-- name: GetActivePhaseForInvite :one
SELECT 
    p.id AS phase_id,
    p.invite_id,
    p."order",
    p.strategy_kind,
    p.strategy_config,
    s.status AS phase_status,
    s.next_check_at,
    s.data AS phase_data
FROM invite_phase_state s
JOIN invite_phases p ON s.phase_id = p.id
WHERE p.invite_id = $1 AND s.status = 'active';

-- name: GetActivePhasesToProcess :many
SELECT 
    p.id AS phase_id,
    p.invite_id,
    p."order",
    p.strategy_kind,
    p.strategy_config,
    s.status AS phase_status,
    s.next_check_at,
    s.data AS phase_data,
    i.title,
    i.description,
    i."from",
    i."to",
    i.duration,
    i.created_at,
    i.status AS invite_status,
    i.from_person_id
FROM invite_phase_state s
JOIN invite_phases p ON s.phase_id = p.id
JOIN invites i ON p.invite_id = i.id
WHERE s.status = 'active' 
  AND (s.next_check_at IS NULL OR s.next_check_at <= $1);

-- name: UpdatePhaseState :exec
UPDATE invite_phase_state
SET status = $2, next_check_at = $3, data = $4
WHERE phase_id = $1;

-- name: CreateInvitee :exec
INSERT INTO invitees (id, invite_id, contact_id, state, created_at, magic_token)
VALUES ($1, $2, $3, $4, $5, $6);

-- name: CreatePhaseState :exec
INSERT INTO invite_phase_state (phase_id, status, next_check_at, data)
VALUES ($1, $2, $3, $4);

-- name: GetInvitee :one
SELECT * FROM invitees WHERE id = $1;

-- name: GetInviteeByToken :one
SELECT i.*, inv.title, inv.description as invite_description, inv."from", inv."to"
FROM invitees i
JOIN invites inv ON i.invite_id = inv.id
WHERE i.magic_token = $1;

-- name: RespondToInvite :exec
UPDATE invitees SET state = $2 WHERE magic_token = $1;

-- name: ResolveRecipients :many
SELECT p.id, p.email, p.name
FROM persons p
WHERE p.id = ANY($1::uuid[])
OR p.id IN (
    SELECT contact_id FROM group_members WHERE group_id = ANY($1::uuid[])
);

-- name: GetPersonByEmail :one
SELECT * FROM persons WHERE email = $1;

-- name: GetPersonByResetToken :one
SELECT * FROM persons 
WHERE password_reset_token = $1 
  AND password_reset_expires_at > NOW();

-- name: UpdatePersonAuth :one
UPDATE persons
SET 
    password_hash = COALESCE(sqlc.narg('password_hash'), password_hash),
    password_reset_token = COALESCE(sqlc.narg('password_reset_token'), password_reset_token),
    password_reset_expires_at = COALESCE(sqlc.narg('password_reset_expires_at'), password_reset_expires_at)
WHERE id = sqlc.arg('id')
RETURNING *;

-- name: CreateSession :one
INSERT INTO sessions (id, person_id, expires_at, created_at)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: GetSession :one
SELECT s.*, p.email, p.name, p.password_hash
FROM sessions s
JOIN persons p ON s.person_id = p.id
WHERE s.id = $1 AND s.expires_at > NOW();

-- name: DeleteSession :exec
DELETE FROM sessions WHERE id = $1;

-- name: CountAdmins :one
SELECT COUNT(*) FROM persons WHERE password_hash IS NOT NULL;

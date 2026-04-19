-- name: GetPerson :one
SELECT * FROM persons WHERE id = $1;

-- name: CreatePerson :one
INSERT INTO persons (id, email, name) VALUES ($1, $2, $3) RETURNING *;

-- name: ListPersons :many
SELECT * FROM persons;

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
    i.status AS invite_status
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
INSERT INTO invitees (id, invite_id, contact_id, state, created_at)
VALUES ($1, $2, $3, $4, $5);

-- name: CreatePhaseState :exec
INSERT INTO invite_phase_state (phase_id, status, next_check_at, data)
VALUES ($1, $2, $3, $4);

-- name: GetInvitee :one
SELECT * FROM invitees WHERE id = $1;

-- name: ResolveRecipients :many
SELECT p.id, p.email, p.name
FROM persons p
WHERE p.id = ANY($1::uuid[])
OR p.id IN (
    SELECT contact_id FROM group_members WHERE group_id = ANY($1::uuid[])
);

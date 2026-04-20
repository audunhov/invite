-- +goose Up

-- 1. Invitees cascade delete
ALTER TABLE invitees DROP CONSTRAINT invitees_invite_id_fkey;
ALTER TABLE invitees ADD CONSTRAINT invitees_invite_id_fkey 
    FOREIGN KEY (invite_id) REFERENCES invites(id) ON DELETE CASCADE;

-- 2. Invite Phases cascade delete
ALTER TABLE invite_phases DROP CONSTRAINT invite_phases_invite_id_fkey;
ALTER TABLE invite_phases ADD CONSTRAINT invite_phases_invite_id_fkey 
    FOREIGN KEY (invite_id) REFERENCES invites(id) ON DELETE CASCADE;

-- 3. Invite Phase State cascade delete
ALTER TABLE invite_phase_state DROP CONSTRAINT invite_phase_state_phase_id_fkey;
ALTER TABLE invite_phase_state ADD CONSTRAINT invite_phase_state_phase_id_fkey 
    FOREIGN KEY (phase_id) REFERENCES invite_phases(id) ON DELETE CASCADE;


-- +goose Down
ALTER TABLE invite_phase_state DROP CONSTRAINT invite_phase_state_phase_id_fkey;
ALTER TABLE invite_phase_state ADD CONSTRAINT invite_phase_state_phase_id_fkey 
    FOREIGN KEY (phase_id) REFERENCES invite_phases(id);

ALTER TABLE invite_phases DROP CONSTRAINT invite_phases_invite_id_fkey;
ALTER TABLE invite_phases ADD CONSTRAINT invite_phases_invite_id_fkey 
    FOREIGN KEY (invite_id) REFERENCES invites(id);

ALTER TABLE invitees DROP CONSTRAINT invitees_invite_id_fkey;
ALTER TABLE invitees ADD CONSTRAINT invitees_invite_id_fkey 
    FOREIGN KEY (invite_id) REFERENCES invites(id);

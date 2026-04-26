# Streamlined Invite Wizard Design Spec

## Goal
Improve the user experience of creating a multi-phase invitation by consolidating the process into a single guided wizard and ensuring atomic data creation.

## Architecture

### API Update: Atomic Deep Create
Update `POST /invites` (NewInvite) to include an optional `phases` array.
- **NewInvite**:
    - ... existing fields ...
    - `tag_ids`: UUID[]
    - `phases`: Array of `NewInvitePhase`
- **Backend Logic**:
    - Open a database transaction.
    - Create the `invite` record.
    - Map `tag_ids` to `invite_tags`.
    - Iterate and create all `invite_phases`.
    - Commit or rollback on any failure.

### Frontend: Wizard UI
Refactor the Invite Modal into a 3-step wizard.
- **Step 1: Details**
- **Step 2: Phases** (List with "Add Phase" button that opens an inline form)
- **Step 3: Summary**
- **Buttons**: [Cancel] [Back] [Next/Create]

## Success Criteria
- Users can create a complete multi-phase invitation in a single flow.
- No "incomplete" invitations exist without phases.
- Improved validation (e.g., cannot proceed without at least one phase).

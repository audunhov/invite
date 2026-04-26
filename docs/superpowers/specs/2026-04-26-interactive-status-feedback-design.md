# Interactive Status Feedback Design Spec

## Goal
Provide granular, real-time visual feedback on the progress of multi-phase invitations directly within the invitation lists, improving the user's ability to monitor multiple processes at once.

## Architecture

### API Expansion: Invite Progress
Update `Invite` schema to include a `progress` object.
- `total_phases`: integer
- `active_phase_order`: integer (1-indexed)
- `total_accepted`: integer
- `total_invitees`: integer

### Backend Optimization
Update the `ListInvites` SQL query to aggregate phase and response counts.
- Count total phases per invite ID.
- Determine the current active phase order via `invite_phase_state`.
- Sum accepted responses and total recipients.

### Frontend: Micro-Timeline UI
Implement a new visual sub-component in `InvitesView.vue`.
- **Location**: Beneath the status badge in the Table and on the Card.
- **Visual**: A segmented horizontal bar.
- **Dynamic Styling**: CSS classes mapped to phase statuses (Completed, Active, Pending).

## Success Criteria
- Users can see the specific phase progress (e.g., 2 of 3) without opening a modal.
- Users have immediate visibility into response rates from the main list.
- The UI remains clean and uncluttered despite the additional data.

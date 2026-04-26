# Dashboard Visual Timeline Design Spec

## Goal
Provide a high-level visual overview of all active and recent invitations, showing their progress through multiple phases in a Gantt-style timeline.

## Architecture

### API Expansion
Update `DashboardStats` to include a `timeline` array.
- **TimelineInvite**:
    - `id`: UUID
    - `title`: string
    - `status`: string (pending, active, completed)
    - `phases`: Array of `TimelinePhase`
- **TimelinePhase**:
    - `order`: integer
    - `status`: string (pending, active, completed)
    - `accepted_count`: integer
    - `declined_count`: integer
    - `total_invitees`: integer

### Frontend Component
A new visualization in `DashboardView.vue` using Tailwind CSS for a custom Gantt-style track.
- Rows of invitations with their titles.
- A horizontal "track" per row.
- Tracks are segmented by phase count.
- Dynamic colors and tooltips for phase details.

## Success Criteria
- Users can see where each active invitation is in its phase sequence.
- Users can see response ratios per phase by hovering.
- Historical (recently completed) invites are visible for context.

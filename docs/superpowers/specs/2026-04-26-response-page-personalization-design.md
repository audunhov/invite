# Response Page Personalization Design Spec

## Goal
Transform the response page from a functional form into a professional, personalized invitation card to improve recipient engagement and acceptance rates.

## Architecture

### API Expansion: PublicInviteDetails
Update `PublicInviteDetails` schema.
- **Fields**:
    - `sender_name`: string (Full name of the person sending the invite)
    - `description`: string (Personal message/description from the invite)

### Backend Logic
Update `GetInviteeByToken` SQL query.
- Join `invitees` with `invites`.
- Join `invites` with `persons` (on `from_person_id`) to fetch the sender's name.

### Frontend: RespondView.vue
Redesign the layout using Approach 1 (Invite Card).
- Centered, shadowed card.
- "You're Invited by [Name]" header.
- Personal message block.
- Detailed date/time section.
- Clear action transitions (Accept/Decline -> Confirmation).

## Success Criteria
- Recipients see exactly who invited them.
- Recipients can read the full personal message provided in the invitation.
- The interface feels premium and "invitation-like."

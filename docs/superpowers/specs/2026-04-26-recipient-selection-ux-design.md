# Smart Recipient Selection UX Design Spec

## Goal
Improve the efficiency and clarity of selecting recipients (Persons and Groups) for invitation phases by adding real-time search and clear selection feedback.

## Architecture

### Frontend State & Logic
- **Search Query**: A new `recipientSearchQuery` string ref.
- **Computed: Unified Recipients**:
    - Combines `persons` (typed as 'person') and `groups` (typed as 'group').
    - Filters by the search query across name and email fields.
- **Computed: Selection Feedback**:
    - A list of objects representing currently selected items to render as removable chips.

### UI Components
- **Search Bar**: A Tailwind-styled input with a search icon.
- **Selected Tray**: A flexbox container of badges/chips for active selections.
- **Enhanced List**:
    - Sticky headers for 'Filtered Results' vs 'Everyone'.
    - Clear visual 'active' state for selected rows.

## Success Criteria
- Users can find a recipient in less than 2 seconds by typing.
- Users can clearly see the total count and names of selected recipients before confirming a phase.
- Group selections are visually distinct from individual selections.

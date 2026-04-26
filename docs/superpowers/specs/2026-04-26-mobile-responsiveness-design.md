# Mobile Responsiveness Design Spec

## Goal
Optimize the application for mobile devices by replacing complex, horizontal tables with readable vertical card layouts on small screens, ensuring a high-quality touch experience.

## Architecture

### Component Pattern
For each main list view (Invites, Persons, Groups):
1.  **Wrap Desktop Table**: Apply `hidden sm:block` to the current table container.
2.  **Add Mobile Card List**: Create a new container with `block sm:hidden` that iterates through the same data.
3.  **Card Structure**:
    - **Header**: High-priority information (Title/Name) + Status.
    - **Body**: Meta-data (Dates, Tags, Emails) using icons or small labels.
    - **Footer**: Action buttons refactored into a tapping-optimized grid.

### Touch Optimization
- Increase button padding to `py-3` on mobile.
- Use `grid-cols-2` for action groups to provide larger hit areas.
- Ensure modals take up `100%` width on mobile with consistent safe-area padding.

## Success Criteria
- No horizontal scrolling on screens down to 320px width.
- Critical actions (Start, Status, Edit) are reachable with a single thumb tap.
- Information hierarchy remains clear despite the reduced width.

# Tags Feature Design Spec

## Goal
Implement a configurable tagging system for invitations to group them by type (e.g., "Conference", "Debate") with visual color indicators.

## Architecture

### Database
- `tags` table: stores global tag definitions.
- `invite_tags` table: join table for many-to-many relationship between invites and tags.

### API (OpenAPI)
- `GET /tags`: List all tags.
- `POST /tags`: Create tag.
- `PATCH /tags/{id}`: Update tag.
- `DELETE /tags/{id}`: Delete tag.
- `Invite` object: Updated to include `tags` array.
- `NewInvite`/`UpdateInvite`: Updated to accept `tag_ids`.

### UI/UX
- **Tag Management**: Located in `SettingsView.vue`.
- **Tag Selection**: Multi-select in Invite Create/Edit modal.
- **Tag Display**: Colored badges in Invites list and Dashboard.
- **Safety**: Confirmation dialog when deleting a tag that is currently assigned to one or more invitations.

## Success Criteria
- Users can create, edit, and delete tags with custom colors.
- Users can assign multiple tags to an invitation.
- Deleting a used tag triggers a confirmation warning.
- Tags are visible in the main lists.

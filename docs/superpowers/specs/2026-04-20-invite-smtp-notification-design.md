# Design Spec: SMTP Integration and Dynamic Sender Notifications

## Overview
Implement an automated email notification system that sends invites to recipients using a dynamic "From" address based on the person who created the invite. The system will integrate with Google Workspace SMTP Relay to handle delivery.

## Goals
- Add SMTP configuration to the application.
- Track which `Person` is sending an `Invite`.
- Automatically send emails when an `Invitee` is created or activated.
- Provide a UI for selecting the sender from a list of available persons.

## Architecture

### 1. Data Model Changes
- **Table `invites`**: Add `from_person_id UUID` (Foreign Key to `persons.id`).
- **Migration**: `20260420000100_add_invite_sender.sql` to add the column and update existing rows if necessary.
- **OpenAPI**: Update `Invite` and `NewInvite` schemas to include `from_person_id`.

### 2. Backend Implementation (Go)
- **Configuration**:
  - `SMTP_HOST`: e.g., `smtp-relay.gmail.com`
  - `SMTP_PORT`: e.g., `587`
  - `SMTP_USER`: SMTP authentication username.
  - `SMTP_PASS`: SMTP authentication password (or App Password).
  - `SMTP_FROM_OVERRIDE`: (Optional) A fixed envelope sender if required by the relay.
- **Email Service**:
  - A new package or internal service to manage SMTP connections.
  - `SendInviteEmail(recipient Person, sender Person, invite Invite, magicToken UUID)`: Constructs and sends the email.
- **Logic Integration**:
  - Update `InvitePerson` in `main.go` to trigger the email sending in a background goroutine.
  - The email will include:
    - Subject: `Invite: {Invite.Title}`
    - Body: A simple text template with the description and the response link (`{BaseURL}/respond/{MagicToken}`).

### 3. Frontend Implementation (Vue)
- **InvitesView.vue**:
  - Add a "From" dropdown to the `InviteModal`.
  - Fetch all persons to populate the dropdown.
  - Default the selection to a "current user" (mocked as "Tom Cook"/`tom@example.com`).
  - Add a "System" person option (e.g., `system@example.com`).

## Success Criteria
- [ ] Creating an invite saves the `from_person_id`.
- [ ] When a "Sprint" or "Ladder" strategy triggers an invite, an email is sent to the recipient.
- [ ] The `From` header in the email matches the name and email of the selected `from_person_id`.
- [ ] The email body contains a valid link to the response page.

## Testing Strategy
- **Manual**: Use a tool like [MailHog](https://github.com/mailhog/MailHog) or a test Gmail account to verify delivery.
- **Unit**: Mock the SMTP sender to verify that `SendInviteEmail` is called with the correct parameters during strategy execution.

## Error Handling
- Log SMTP connection or delivery failures.
- Ensure that a failed email does not crash the orchestrator or API response (run in background).

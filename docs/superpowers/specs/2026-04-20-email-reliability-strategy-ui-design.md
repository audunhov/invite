# Design Spec: Email Reliability and Strategy UI

## Overview
Improve the reliability of email delivery through persistent logging and automated retries. Enhance the user experience by replacing raw JSON strategy configurations with human-readable summaries and exposing email delivery status in the UI.

## Goals
- Track all outgoing emails (Invites and Password Resets).
- Automatically retry failed email deliveries up to 3 times.
- Provide a manual retry option for persistent failures.
- Display concise, human-readable summaries of strategy configurations.
- Expose email delivery status to administrators.

## Architecture

### 1. Data Model Changes
- **Table `email_logs`**:
  - `id`: `UUID` (Primary Key).
  - `invitee_id`: `UUID` (Nullable, Foreign Key to `invitees.id` ON DELETE CASCADE).
  - `recipient_email`: `TEXT` (Not Null).
  - `subject`: `TEXT`.
  - `body`: `TEXT`.
  - `status`: `TEXT` (`pending`, `sent`, `failed`).
  - `error_message`: `TEXT`.
  - `attempts`: `INT` (Default 0).
  - `last_attempt_at`: `TIMESTAMP WITH TIME ZONE`.
  - `created_at`: `TIMESTAMP WITH TIME ZONE` (Default `NOW()`).

### 2. Backend Implementation (Go)

#### Email Service (`email/email.go`)
- Update `SendInvite` and `SendResetPasswordEmail` to:
  1. Create a `pending` log entry in `email_logs`.
  2. Attempt delivery.
  3. Update log status to `sent` on success or `failed` with error message on failure.
- Add `RetryEmail(ctx, logID)` method.

#### Orchestrator (`internal/app/orchestrator.go`)
- Add `ProcessFailedEmails(ctx)`:
  - Queries `email_logs` for `status = 'failed'` AND `attempts < 3`.
  - Re-triggers delivery for each.

#### API (`api/server.go`)
- Update `GetInviteStatus` to include email delivery status for each invitee.
- Add `POST /api/emails/{id}/retry` to trigger a manual retry.

### 3. Frontend Implementation (Vue 3)

#### Strategy Summaries (`InvitesView.vue`)
- Implement `formatStrategyConfig(phase)` helper:
  - **Ladder**: "Ladder: {count} persons, {timeout}m timeout"
  - **Sprint**: "Sprint: {count} recipients, {deadline} deadline"

#### Email Status UI (`InvitesView.vue` -> Status Modal)
- Add "Email Status" column to Recipient Details table.
- Show status badges:
  - `Sent` (Green)
  - `Retrying ({n}/3)` (Yellow)
  - `Failed` (Red, with tooltip for error)
- Show "Retry" button for `Failed` entries.

## Success Criteria
- [ ] Every email sent creates a log entry.
- [ ] Failed emails are automatically retried by the background worker.
- [ ] Admin can see why an email failed and trigger a manual retry.
- [ ] The "Phases" list shows readable summaries instead of JSON.

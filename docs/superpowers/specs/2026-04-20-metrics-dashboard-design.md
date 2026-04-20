# Design Spec: Metrics Dashboard

## Overview
Transform the application dashboard from a static landing page into a data-driven command center. The new dashboard will provide immediate visibility into active invitation processes, identify bottlenecks, and show a chronological feed of recent system activity.

## Goals
- Provide at-a-glance metrics for the last 30 days.
- Visualize "Active Bottlenecks" for all running invites.
- Show a unified "Recent Activity" feed derived from system events.
- Improve navigation to active processes.

## Architecture

### 1. API Extension (`api/openapi.yaml`)
- **New Endpoint**: `GET /api/dashboard/stats`.
- **Response Schema**:
  - `stats`: `active_invites_count`, `success_rate_percent`, `failed_emails_count`.
  - `bottlenecks`: Array of objects containing `invite_id`, `title`, `phase_order`, `strategy_kind`, `waiting_for_name`, and `active_since`.
  - `activity`: Array of objects containing `timestamp`, `type` (response, transition, error), and `message`.

### 2. Backend Implementation (Go)

#### Data Derivation (`internal/app/dashboard.go`)
- **Active Bottlenecks**:
  - Query active `invite_phase_state`.
  - For **Ladder**: Parse the state JSON to find the current person in the `List`.
  - For **Sprint**: List all pending invitees for that phase.
- **Recent Activity**:
  - Use `UNION ALL` or multiple queries to fetch the latest:
    - Invitee responses (Accepted/Declined).
    - Phase transitions (Phase State created/updated).
    - Email errors (from `email_logs`).
  - Sort by timestamp descending and limit to 20.

#### Controller (`api/server.go`)
- Implement the `GetDashboardStats` handler using the application service logic.

### 3. Frontend Implementation (Vue 3)

#### Layout (`DashboardView.vue`)
- **Top Row**: 3-4 Metric Cards (Big numbers, simple labels).
- **Middle Section**: "Active Bottlenecks" Grid.
  - Tailwind-styled cards with "Waiting for [Name]" prominently displayed.
  - Direct link to the Status modal of that invite.
- **Sidebar/Bottom Section**: "Recent Activity" Feed.
  - A clean vertical list with time-ago timestamps and icons per event type.

## Success Criteria
- [ ] Dashboard displays the correct number of active invites.
- [ ] Every active "Ladder" invite shows the specific person it is currently waiting on.
- [ ] Recent activity shows the latest invitee responses accurately.
- [ ] UI supports dark mode and matches existing aesthetics.

## Future Considerations
- Adjustable time range filters.
- Real-time updates via WebSockets or long polling.
- "Send Reminder" button directly from the bottleneck card.

# Metrics Dashboard Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Transform the dashboard into an actionable metrics center with bottleneck tracking and a recent activity feed.

**Architecture:**
- **Backend:** 
    - SQL queries for aggregating stats and recent events.
    - Application logic for parsing active phase state to identify specific bottlenecks.
- **Frontend:** 
    - Responsive card-based UI with Top Row stats.
    - Interactive "Active Bottlenecks" grid.
    - Chronological "Recent Activity" feed.

**Tech Stack:** Go, Vue 3, PostgreSQL (sqlc).

---

### Task 1: Database and API Extension

**Files:**
- Modify: `db/query.sql`
- Modify: `api/openapi.yaml`
- Modify: `api/api.gen.go` (via generation)
- Modify: `db/query.sql.go` (via generation)

- [ ] **Step 1: Implement Dashboard SQL Queries**
Add stats and activity queries to `db/query.sql`.
```sql
-- name: GetRecentActivity :many
(SELECT i.created_at as timestamp, 'invitee_response' as type, p.name || ' ' || i.state || ' invite "' || inv.title || '"' as message 
 FROM invitees i JOIN persons p ON i.contact_id = p.id JOIN invites inv ON i.invite_id = inv.id 
 WHERE i.state != 'pending' AND i.created_at > NOW() - INTERVAL '30 days')
UNION ALL
(SELECT created_at as timestamp, 'email_error' as type, 'Email failed to ' || recipient_email || ': ' || COALESCE(error_message, 'Unknown error') as message 
 FROM email_logs WHERE status = 'failed' AND created_at > NOW() - INTERVAL '30 days')
ORDER BY timestamp DESC LIMIT 20;

-- name: GetGlobalStats :one
SELECT 
    (SELECT COUNT(*) FROM invites WHERE status = 'active')::int as active_invites,
    (SELECT COUNT(*) FROM email_logs WHERE status = 'failed' AND created_at > NOW() - INTERVAL '30 days')::int as failed_emails,
    COALESCE((SELECT (COUNT(CASE WHEN state = 'accepted' THEN 1 END) * 100.0 / NULLIF(COUNT(*), 0))::float 
     FROM invitees WHERE created_at > NOW() - INTERVAL '30 days'), 0.0)::float as success_rate;
```

- [ ] **Step 2: Run sqlc generate**
Run: `sqlc generate`

- [ ] **Step 3: Update OpenAPI Specification**
Add `GET /api/dashboard/stats` and relevant schemas.
```yaml
    DashboardStats:
      type: object
      required: [stats, bottlenecks, activity]
      properties:
        stats:
          type: object
          properties:
            active_invites: { type: integer }
            failed_emails: { type: integer }
            success_rate: { type: number }
        bottlenecks:
          type: array
          items:
            type: object
            properties:
              invite_id: { type: string, format: uuid }
              title: { type: string }
              phase_order: { type: integer }
              strategy_kind: { type: string }
              waiting_for: { type: string }
              active_since: { type: string, format: date-time }
        activity:
          type: array
          items:
            type: object
            properties:
              timestamp: { type: string, format: date-time }
              type: { type: string }
              message: { type: string }
```

- [ ] **Step 4: Run go generate ./api/...**
Run: `go generate ./api/...`

- [ ] **Step 5: Commit**
```bash
git add db/ query.sql api/openapi.yaml
git commit -m "db/api: add dashboard stats support"
```

---

### Task 2: Backend Dashboard Service

**Files:**
- Create: `internal/app/dashboard.go`
- Modify: `api/server.go`

- [ ] **Step 1: Implement GetDashboardStats in App**
Create `internal/app/dashboard.go` with the logic to fetch stats, activity, and derive bottlenecks by parsing active phase JSON state.
- Handle Ladder: Extract current person name from `state.Data.index`.
- Handle Sprint: List all pending recipients for the phase.

- [ ] **Step 2: Add Handler to Server**
Implement `GetDashboardStats` in `api/server.go`.

- [ ] **Step 3: Commit**
```bash
git add internal/app/dashboard.go api/server.go
git commit -m "feat: implement dashboard stats service logic"
```

---

### Task 3: Frontend Dashboard UI

**Files:**
- Modify: `frontend/src/views/DashboardView.vue`

- [ ] **Step 1: Implement Dashboard Layout**
Replace the placeholder with a three-section layout:
1. **Summary Cards**: Active Invites, Success Rate, Failed Emails.
2. **Active Bottlenecks**: Cards showing "Waiting for [Name]" for every active invite.
3. **Recent Activity Feed**: A list showing the latest responses and errors.

- [ ] **Step 2: Add Logic and Auto-Refresh**
Fetch stats on mount and setup a 30-second polling interval.

- [ ] **Step 3: Commit**
```bash
git add frontend/src/views/DashboardView.vue
git commit -m "frontend: implement metrics dashboard UI"
```

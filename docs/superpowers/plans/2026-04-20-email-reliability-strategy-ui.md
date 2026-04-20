# Email Reliability and Strategy UI Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Implement email logging, automated retries, and human-readable strategy summaries.

**Architecture:**
- **Storage:** `email_logs` table for tracking delivery, attempts, and errors.
- **Worker:** Orchestrator task for automated retries of failed emails (up to 3 attempts).
- **API:** Updated `GetInviteStatus` to include email telemetry; new `RetryEmail` endpoint.
- **UI:** Human-readable strategy summaries and delivery status badges in the dashboard.

**Tech Stack:** Go (net/smtp), Vue 3, PostgreSQL (sqlc).

---

### Task 1: Database and SQL Updates

**Files:**
- Create: `db/migrations/20260420000400_add_email_logs.sql`
- Modify: `db/query.sql`
- Modify: `db/query.sql.go` (via generation)

- [ ] **Step 1: Create migration file**
```sql
-- +goose Up
CREATE TABLE email_logs (
    id UUID PRIMARY KEY,
    invitee_id UUID REFERENCES invitees(id) ON DELETE CASCADE,
    recipient_email TEXT NOT NULL,
    subject TEXT NOT NULL,
    body TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'pending',
    error_message TEXT,
    attempts INT NOT NULL DEFAULT 0,
    last_attempt_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- +goose Down
DROP TABLE email_logs;
```

- [ ] **Step 2: Run migration**
Run: `docker-compose exec app go run . migrate`

- [ ] **Step 3: Update SQL queries**
Add logging and retry queries to `db/query.sql`.
```sql
-- name: CreateEmailLog :one
INSERT INTO email_logs (id, invitee_id, recipient_email, subject, body, status, created_at)
VALUES ($1, $2, $3, $4, $5, $6, NOW())
RETURNING *;

-- name: UpdateEmailLogStatus :exec
UPDATE email_logs
SET status = $2, error_message = $3, attempts = attempts + 1, last_attempt_at = NOW()
WHERE id = $1;

-- name: GetFailedEmails :many
SELECT * FROM email_logs WHERE status = 'failed' AND attempts < 3;

-- name: GetEmailLogByInvitee :one
SELECT * FROM email_logs WHERE invitee_id = $1 ORDER BY created_at DESC LIMIT 1;

-- name: GetEmailLog :one
SELECT * FROM email_logs WHERE id = $1;
```

- [ ] **Step 4: Generate SQL code**
Run: `sqlc generate`

- [ ] **Step 5: Commit**
```bash
git add db/migrations/ db/query.sql db/query.sql.go
git commit -m "db: add email_logs table and queries"
```

---

### Task 2: Email Service Logging & Retries

**Files:**
- Modify: `email/email.go`
- Modify: `internal/app/orchestrator.go`
- Create: `internal/app/emails.go`

- [ ] **Step 1: Refactor Email Service**
Update `email/email.go` to handle logging.
```go
func (s *Service) logAndSend(ctx context.Context, queries *db.Queries, inviteeID uuid.NullUUID, toEmail, subject, body string) error {
    logID := uuid.New()
    queries.CreateEmailLog(ctx, db.CreateEmailLogParams{
        ID: logID, InviteeID: inviteeID, RecipientEmail: toEmail, Subject: subject, Body: body, Status: "pending",
    })
    
    err := s.sendRaw(toEmail, subject, body) // Logic moved from SendInvite
    
    status := "sent"
    errMsg := ""
    if err != nil {
        status = "failed"
        errMsg = err.Error()
    }
    return queries.UpdateEmailLogStatus(ctx, db.UpdateEmailLogStatusParams{
        ID: logID, Status: status, ErrorMessage: sql.NullString{String: errMsg, Valid: errMsg != ""},
    })
}
```

- [ ] **Step 2: Add background retry worker**
In `internal/app/orchestrator.go`, add `ProcessFailedEmails` to the loop.
```go
func (app *App) ProcessFailedEmails(ctx context.Context) error {
    failed, _ := app.Queries.GetFailedEmails(ctx)
    for _, log := range failed {
        app.EmailService.SendRaw(log.RecipientEmail, log.Subject, log.Body)
        // Update log status...
    }
    return nil
}
```

- [ ] **Step 3: Commit**
```bash
git add email/ internal/app/
git commit -m "feat: implement automated email retries and logging"
```

---

### Task 3: API & Manual Retry

**Files:**
- Modify: `api/openapi.yaml`
- Modify: `api/server.go`

- [ ] **Step 1: Update OpenAPI Specification**
Add `email_status` and `email_error` to `InviteeStatus`.
Add `POST /api/emails/{id}/retry`.

- [ ] **Step 2: Implement Retry Handler**
In `api/server.go`, implement `RetryEmail` by calling the internal service.

- [ ] **Step 3: Update Status Report**
Update `GetInviteStatus` to join with `email_logs`.

- [ ] **Step 4: Commit**
```bash
git add api/
git commit -m "api: expose email status and add manual retry endpoint"
```

---

### Task 4: Frontend UI Improvements

**Files:**
- Modify: `frontend/src/views/InvitesView.vue`

- [ ] **Step 1: Implement Human-Readable Strategy Summaries**
Replace raw JSON display in the "Phases" list with `formatStrategyConfig`.

- [ ] **Step 2: Enhance Status Modal**
Add "Email Status" column to the recipient details table.
Implement "Retry" button for failed emails.

- [ ] **Step 3: Commit**
```bash
git add frontend/
git commit -m "frontend: improve strategy config display and email status tracking"
```

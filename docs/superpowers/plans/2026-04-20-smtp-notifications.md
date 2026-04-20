# SMTP Notifications Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Implement automated email notifications with dynamic "From" addresses using Google Workspace SMTP Relay.

**Architecture:**
- **Database:** Add `from_person_id` to `invites` table.
- **Backend:** 
    - Create an `email` package for SMTP communication.
    - Update `App.InvitePerson` to trigger emails in a background goroutine.
    - Expand `config.Config` to include SMTP credentials and `BASE_URL`.
- **Frontend:** 
    - Add a "From" dropdown in the Invite creation/edit modal.
    - Default to the current mock user ("Tom Cook").

**Tech Stack:** 
- Go (net/smtp)
- Vue 3 (Composition API)
- SQL (PostgreSQL/sqlc)

---

### Task 1: Database Schema Update

**Files:**
- Create: `db/migrations/20260420000100_add_invite_sender.sql`
- Modify: `db/query.sql`
- Modify: `db/query.sql.go` (via generation)

- [ ] **Step 1: Create migration file**
```sql
-- +goose Up
ALTER TABLE invites ADD COLUMN from_person_id UUID REFERENCES persons(id);

-- +goose Down
ALTER TABLE invites DROP COLUMN from_person_id;
```

- [ ] **Step 2: Run migration**
Run: `docker-compose exec app go run . migrate`
Expected: Table updated with `from_person_id`.

- [ ] **Step 3: Update SQL queries**
Modify `db/query.sql` to include `from_person_id` in `CreateInvite` and `UpdateInvite`.

```sql
-- name: CreateInvite :one
INSERT INTO invites (id, title, description, "from", "to", duration, created_at, status, from_person_id)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
RETURNING *;

-- name: UpdateInvite :one
UPDATE invites
SET 
    title = COALESCE(sqlc.narg('title'), title),
    description = COALESCE(sqlc.narg('description'), description),
    "from" = COALESCE(sqlc.narg('from'), "from"),
    "to" = COALESCE(sqlc.narg('to'), "to"),
    duration = COALESCE(sqlc.narg('duration'), duration),
    status = COALESCE(sqlc.narg('status'), status),
    from_person_id = COALESCE(sqlc.narg('from_person_id'), from_person_id)
WHERE id = sqlc.arg('id')
RETURNING *;
```

- [ ] **Step 4: Generate SQL code**
Run: `sqlc generate` (or equivalent docker command)
Expected: `db/query.sql.go` contains updated structs and methods.

- [ ] **Step 5: Commit**
```bash
git add db/migrations/ db/query.sql db/query.sql.go
git commit -m "db: add from_person_id to invites"
```

---

### Task 2: API and Configuration

**Files:**
- Modify: `api/openapi.yaml`
- Modify: `config/config.go`
- Modify: `api/server.go`
- Modify: `api/api.gen.go` (via generation)

- [ ] **Step 1: Update OpenAPI Specification**
Add `from_person_id` to `Invite`, `NewInvite`, and `UpdateInvite`.

```yaml
    Invite:
      required: [id, title, from, created_at, status, from_person_id]
      properties:
        # ...
        from_person_id:
          type: string
          format: uuid

    NewInvite:
      required: [title, from, from_person_id]
      properties:
        # ...
        from_person_id:
          type: string
          format: uuid

    UpdateInvite:
      properties:
        # ...
        from_person_id:
          type: string
          format: uuid
```

- [ ] **Step 2: Generate API code**
Run: `go generate ./api/...`
Expected: `api/api.gen.go` reflects schema changes.

- [ ] **Step 3: Update Config struct**
Add SMTP and BaseURL fields to `config/config.go`.

```go
type Config struct {
	DatabaseURL string `env:"DB_URL,required"`
	Port        int    `env:"PORT" envDefault:"8080"`
	LogLevel    string `env:"LOG_LEVEL" envDefault:"info"`
	LogFormat   string `env:"LOG_FORMAT" envDefault:"text"`
	SMTPHost    string `env:"SMTP_HOST"`
	SMTPPort    int    `env:"SMTP_PORT" envDefault:"587"`
	SMTPUser    string `env:"SMTP_USER"`
	SMTPPass    string `env:"SMTP_PASS"`
	BaseURL     string `env:"BASE_URL" envDefault:"http://localhost:8080"`
}
```

- [ ] **Step 4: Update Server Implementation**
Update `CreateInvite` and `UpdateInvite` in `api/server.go` to handle `FromPersonID`.

```go
// In CreateInvite
		Status:      "pending",
		FromPersonID: request.Body.FromPersonId,

// In UpdateInvite
	if request.Body.FromPersonId != nil {
		params.FromPersonID = uuid.NullUUID{UUID: *request.Body.FromPersonId, Valid: true}
	}
```

- [ ] **Step 5: Commit**
```bash
git add api/openapi.yaml config/config.go api/server.go api/api.gen.go
git commit -m "api: update schemas and configuration for SMTP"
```

---

### Task 3: Email Service and Trigger

**Files:**
- Create: `email/email.go`
- Modify: `main.go`

- [ ] **Step 1: Create Email Service**
Implement `email.Service` using `net/smtp`.

```go
package email

import (
	"fmt"
	"net/smtp"
	"invite/db"
	"invite/config"
)

type Service struct {
	cfg *config.Config
}

func NewService(cfg *config.Config) *Service {
	return &Service{cfg: cfg}
}

func (s *Service) SendInvite(recipient db.Person, sender db.Person, inviteTitle string, inviteDesc string, token string) error {
	if s.cfg.SMTPHost == "" {
		return nil
	}

	auth := smtp.PlainAuth("", s.cfg.SMTPUser, s.cfg.SMTPPass, s.cfg.SMTPHost)
	to := []string{recipient.Email}
	msg := []byte(fmt.Sprintf("To: %s\r\n"+
		"From: %s <%s>\r\n"+
		"Subject: Invite: %s\r\n"+
		"\r\n"+
		"Hi %s,\r\n\r\n%s\r\n\r\nRespond here: %s/respond/%s\r\n",
		recipient.Email, sender.Name, s.cfg.SMTPUser, inviteTitle, recipient.Name, inviteDesc, s.cfg.BaseURL, token))

	return smtp.SendMail(fmt.Sprintf("%s:%d", s.cfg.SMTPHost, s.cfg.SMTPPort), auth, s.cfg.SMTPUser, to, msg)
}
```

- [ ] **Step 2: Integrate into App**
Add `EmailService *email.Service` to `App` struct in `main.go` and initialize it in `main()`.

- [ ] **Step 3: Trigger Email in InvitePerson**
Modify `app.InvitePerson` in `main.go` to send the email.

```go
func (app *App) InvitePerson(ctx context.Context, i Invite, p Person) (*Invitee, error) {
    // ... create invitee ...
    
    go func() {
        sender, err := app.Queries.GetPerson(context.Background(), i.FromPersonID)
        if err == nil {
            app.EmailService.SendInvite(p, sender, i.Title, i.Description, magicToken.String())
        }
    }()
    
    return invitee, nil
}
```

- [ ] **Step 4: Commit**
```bash
git add email/email.go main.go
git commit -m "feat: implement email service and trigger"
```

---

### Task 4: Frontend Sender Dropdown

**Files:**
- Modify: `frontend/src/views/InvitesView.vue`

- [ ] **Step 1: Update inviteForm and Modal**
Add `from_person_id` to `inviteForm`. Add a dropdown in the template to select the sender from `persons`.

- [ ] **Step 2: Implement Default Sender Logic**
In `openCreateInviteModal`, find "Tom Cook" (by email `tom@example.com`) in `persons` and set `inviteForm.from_person_id`.

- [ ] **Step 3: Update Save Logic**
Ensure `from_person_id` is sent in the body of `POST /api/invites`.

- [ ] **Step 4: Commit**
```bash
git add frontend/src/views/InvitesView.vue
git commit -m "frontend: add sender selection to invite creation"
```

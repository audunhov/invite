# Basic Auth and Sessions Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Implement email/password authentication (bcrypt), database-backed sessions (cookies), and forgotten password flow.

**Architecture:**
- **Database:** `persons` table updated for auth; new `sessions` table.
- **Security:** `bcrypt` for passwords, UUID session tokens, `HttpOnly` secure cookies.
- **Logic:** `AuthMiddleware` for route protection; in-memory rate limiting.
- **UX:** Vue 3 login views and navigation guards.

**Tech Stack:** 
- Go (bcrypt, net/http, sqlc)
- Vue 3 (Composition API)
- PostgreSQL

---

### Task 1: Database and SQL Updates

**Files:**
- Create: `db/migrations/20260420000200_add_auth_and_sessions.sql`
- Modify: `db/query.sql`
- Modify: `db/query.sql.go` (via generation)

- [ ] **Step 1: Create migration file**
```sql
-- +goose Up
ALTER TABLE persons ADD COLUMN password_hash TEXT;
ALTER TABLE persons ADD COLUMN password_reset_token TEXT UNIQUE;
ALTER TABLE persons ADD COLUMN password_reset_expires_at TIMESTAMP WITH TIME ZONE;

CREATE TABLE sessions (
    id UUID PRIMARY KEY,
    person_id UUID NOT NULL REFERENCES persons(id) ON DELETE CASCADE,
    expires_at TIMESTAMP WITH TIME ZONE NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- +goose Down
DROP TABLE sessions;
ALTER TABLE persons DROP COLUMN password_hash;
ALTER TABLE persons DROP COLUMN password_reset_token;
ALTER TABLE persons DROP COLUMN password_reset_expires_at;
```

- [ ] **Step 2: Run migration**
Run: `docker-compose exec app go run . migrate`

- [ ] **Step 3: Update SQL queries**
Add auth queries to `db/query.sql`.

```sql
-- name: GetPersonByEmail :one
SELECT * FROM persons WHERE email = $1;

-- name: GetPersonByResetToken :one
SELECT * FROM persons 
WHERE password_reset_token = $1 
AND password_reset_expires_at > NOW();

-- name: UpdatePersonAuth :exec
UPDATE persons 
SET 
    password_hash = COALESCE(sqlc.narg('password_hash'), password_hash),
    password_reset_token = sqlc.narg('password_reset_token'),
    password_reset_expires_at = sqlc.narg('password_reset_expires_at')
WHERE id = sqlc.arg('id');

-- name: CreateSession :one
INSERT INTO sessions (id, person_id, expires_at)
VALUES ($1, $2, $3)
RETURNING *;

-- name: GetSession :one
SELECT s.*, p.email, p.name 
FROM sessions s
JOIN persons p ON s.person_id = p.id
WHERE s.id = $1 AND s.expires_at > NOW();

-- name: DeleteSession :exec
DELETE FROM sessions WHERE id = $1;

-- name: CountAdmins :one
SELECT COUNT(*) FROM persons WHERE password_hash IS NOT NULL;
```

- [ ] **Step 4: Generate SQL code**
Run: `sqlc generate`

- [ ] **Step 5: Commit**
```bash
git add db/migrations/ db/query.sql db/query.sql.go db/models.go
git commit -m "db: add auth and sessions schema"
```

---

### Task 2: Backend Infrastructure

**Files:**
- Create: `internal/auth/auth.go`
- Create: `internal/limiter/limiter.go`

- [ ] **Step 1: Create auth package**
Implement Bcrypt hashing and token generation in `internal/auth/auth.go`.
```go
package auth
import (
	"crypto/rand"
	"encoding/hex"
	"golang.org/x/crypto/bcrypt"
)
func HashPassword(p string) (string, error) {
	h, err := bcrypt.GenerateFromPassword([]byte(p), bcrypt.DefaultCost)
	return string(h), err
}
func CheckPassword(p, h string) bool {
	return bcrypt.CompareHashAndPassword([]byte(h), []byte(p)) == nil
}
func GenerateSecureToken() string {
	b := make([]byte, 32)
	rand.Read(b)
	return hex.EncodeToString(b)
}
```

- [ ] **Step 2: Create limiter package**
Implement IP-based rate limiting in `internal/limiter/limiter.go`.

- [ ] **Step 3: Commit**
```bash
git add internal/
git commit -m "feat: add auth and limiter packages"
```

---

### Task 3: API Extension

**Files:**
- Modify: `api/openapi.yaml`
- Modify: `api/server.go`
- Modify: `api/api.gen.go` (via generation)

- [ ] **Step 1: Update OpenAPI Specification**
Add `/auth/login`, `/auth/logout`, `/auth/forgot-password`, `/auth/reset-password`, and `/auth/me`.
Add `has_password` and `password` fields to `Person` and `UpdatePerson`.

- [ ] **Step 2: Generate API code**
Run: `go generate ./api/...`

- [ ] **Step 3: Implement Auth Middleware**
In `api/server.go`, add `AuthMiddleware` that checks the `session_id` cookie and fetches the user from DB.

- [ ] **Step 4: Implement Auth Handlers**
Implement login (creating session), logout (deleting session), and reset password handlers in `api/server.go`.

- [ ] **Step 5: Implement Deletion Safety**
Update `DeletePerson` handler to check `CountAdmins`.

- [ ] **Step 6: Commit**
```bash
git add api/
git commit -m "api: implement auth endpoints and middleware"
```

---

### Task 4: Frontend Implementation

**Files:**
- Create: `frontend/src/views/LoginView.vue`
- Create: `frontend/src/views/ForgotPasswordView.vue`
- Create: `frontend/src/views/ResetPasswordView.vue`
- Modify: `frontend/src/router/index.ts`
- Modify: `frontend/src/App.vue`

- [ ] **Step 1: Setup Router Guards**
Add logic to check authentication before entering internal routes.

- [ ] **Step 2: Implement Login View**
Create a Tailwind-styled login form.

- [ ] **Step 3: Implement Password Reset Views**
Create views for requesting reset and submitting new password.

- [ ] **Step 4: Final Verification**
Test the full flow: login -> access dashboard -> logout -> unauthorized access check.

- [ ] **Step 5: Commit**
```bash
git add frontend/
git commit -m "frontend: implement login and password reset flows"
```

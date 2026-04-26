# Tags Feature Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Implement a tagging system for invitations with many-to-many relationships, custom colors, and deletion protection.

**Architecture:** Use a `tags` table and an `invite_tags` join table. Update the API to support tag CRUD and association. Integrate into Settings for management and Invites for assignment.

**Tech Stack:** Go (backend), PostgreSQL (DB), Vue 3 (frontend), Tailwind CSS.

---

### Task 1: Database Schema & Queries

**Files:**
- Create: `db/migrations/20260426000000_add_tags.sql`
- Modify: `db/query.sql`

- [ ] **Step 1: Create migration file**

```sql
-- +goose Up
CREATE TABLE tags (
    id UUID PRIMARY KEY,
    name TEXT NOT NULL UNIQUE,
    color TEXT NOT NULL DEFAULT '#6366f1',
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE TABLE invite_tags (
    invite_id UUID NOT NULL REFERENCES invites(id) ON DELETE CASCADE,
    tag_id UUID NOT NULL REFERENCES tags(id) ON DELETE CASCADE,
    PRIMARY KEY (invite_id, tag_id)
);

-- +goose Down
DROP TABLE invite_tags;
DROP TABLE tags;
```

- [ ] **Step 2: Run migration**

Run: `goose -dir db/migrations postgres "user=postgres password=postgres dbname=invite sslmode=disable" up`

- [ ] **Step 3: Update `db/query.sql`**

Add queries for tag management and associations.

```sql
-- name: ListTags :many
SELECT * FROM tags ORDER BY name ASC;

-- name: GetTag :one
SELECT * FROM tags WHERE id = $1;

-- name: CreateTag :one
INSERT INTO tags (id, name, color) VALUES ($1, $2, $3) RETURNING *;

-- name: UpdateTag :one
UPDATE tags SET name = $2, color = $3 WHERE id = $1 RETURNING *;

-- name: DeleteTag :exec
DELETE FROM tags WHERE id = $1;

-- name: GetTagUsageCount :one
SELECT COUNT(*) FROM invite_tags WHERE tag_id = $1;

-- name: SetInviteTags :exec
DELETE FROM invite_tags WHERE invite_id = $1;
INSERT INTO invite_tags (invite_id, tag_id)
SELECT $1, unnest($2::uuid[]);

-- name: GetTagsByInvite :many
SELECT t.* FROM tags t
JOIN invite_tags it ON t.id = it.tag_id
WHERE it.invite_id = $1;
```

- [ ] **Step 4: Generate sqlc code**

Run: `sqlc generate`

- [ ] **Step 5: Commit**

```bash
git add db/migrations/20260426000000_add_tags.sql db/query.sql db/query.sql.go db/models.go
git commit -m "db: add tags and invite_tags tables and queries"
```

---

### Task 2: OpenAPI & API Generation

**Files:**
- Modify: `api/openapi.yaml`

- [ ] **Step 1: Update `api/openapi.yaml`**

Add Tag schemas and endpoints.

```yaml
  /tags:
    get:
      summary: List all tags
      operationId: ListTags
      responses:
        '200':
          description: A list of tags
          content:
            application/json:
              schema:
                type: array
                items:
                  $ref: '#/components/schemas/Tag'
    post:
      summary: Create a new tag
      operationId: CreateTag
      requestBody:
        required: true
        content:
          application/json:
            schema:
              $ref: '#/components/schemas/NewTag'
      responses:
        '201':
          description: Created
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/Tag'
  /tags/{id}:
    patch:
      summary: Update a tag
      operationId: UpdateTag
      parameters:
        - name: id
          in: path
          required: true
          schema:
            type: string
            format: uuid
      requestBody:
        required: true
        content:
          application/json:
            schema:
              $ref: '#/components/schemas/UpdateTag'
      responses:
        '200':
          description: Updated
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/Tag'
    delete:
      summary: Delete a tag
      operationId: DeleteTag
      parameters:
        - name: id
          in: path
          required: true
          schema:
            type: string
            format: uuid
      responses:
        '204':
          description: Deleted
    get:
      summary: Get tag usage count
      operationId: GetTagUsage
      parameters:
        - name: id
          in: path
          required: true
          schema:
            type: string
            format: uuid
      responses:
        '200':
          description: Usage count
          content:
            application/json:
              schema:
                type: object
                properties:
                  count: { type: integer }

# Components updates
    Tag:
      type: object
      required: [id, name, color]
      properties:
        id: { type: string, format: uuid }
        name: { type: string }
        color: { type: string }
    NewTag:
      type: object
      required: [name, color]
      properties:
        name: { type: string }
        color: { type: string }
    UpdateTag:
      type: object
      properties:
        name: { type: string }
        color: { type: string }

# Invite updates
    Invite:
      properties:
        tags:
          type: array
          items:
            $ref: '#/components/schemas/Tag'
    NewInvite:
      properties:
        tag_ids:
          type: array
          items: { type: string, format: uuid }
    UpdateInvite:
      properties:
        tag_ids:
          type: array
          items: { type: string, format: uuid }
```

- [ ] **Step 2: Generate API code**

Run: `cd api && go generate ./...`

- [ ] **Step 3: Commit**

```bash
git add api/openapi.yaml api/api.gen.go
git commit -m "api: define tags endpoints and update invite schemas"
```

---

### Task 3: Backend Implementation

**Files:**
- Modify: `api/server.go`
- Modify: `internal/app/invites.go`

- [ ] **Step 1: Implement Tag CRUD in `api/server.go`**

Implement `ListTags`, `CreateTag`, `UpdateTag`, `DeleteTag`, and `GetTagUsage`.

- [ ] **Step 2: Update Invite creation and updates in `api/server.go`**

Pass `tag_ids` to the application layer.

- [ ] **Step 3: Update Invitation details in `api/server.go`**

Include tags in the response for `ListInvites`, `GetInvite`, and `GetInviteStatus`.

- [ ] **Step 4: Update application logic in `internal/app/invites.go`**

Handle the `invite_tags` associations during create/update.

- [ ] **Step 5: Verify with tests**

Run: `go test ./...`

- [ ] **Step 6: Commit**

```bash
git add api/server.go internal/app/invites.go
git commit -m "feat: implement tags backend logic"
```

---

### Task 4: Frontend Tag Management (Settings)

**Files:**
- Modify: `frontend/src/views/SettingsView.vue`

- [ ] **Step 1: Add Tag Management Section to `SettingsView.vue`**

Add UI to list tags, edit them, and delete them.
Implement the delete confirmation dialog using `ConfirmModal.vue`.

- [ ] **Step 2: Commit**

```bash
git add frontend/src/views/SettingsView.vue
git commit -m "ui: add tag management to settings"
```

---

### Task 5: Frontend Invite Tag Selection & Display

**Files:**
- Modify: `frontend/src/views/InvitesView.vue`
- Modify: `frontend/src/views/DashboardView.vue`

- [ ] **Step 1: Update `InvitesView.vue` Modal**

Add multi-select for tags in the Create/Edit Invite modal.

- [ ] **Step 2: Display Tag Badges in `InvitesView.vue` table**

- [ ] **Step 3: Display Tag Badges in `DashboardView.vue`**

- [ ] **Step 4: Verify with Build**

Run: `cd frontend && npm run build`

- [ ] **Step 5: Commit**

```bash
git add frontend/src/views/InvitesView.vue frontend/src/views/DashboardView.vue
git commit -m "ui: add tag selection and display to invites and dashboard"
```

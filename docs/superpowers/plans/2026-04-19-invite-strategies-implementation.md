# Invite Strategies Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Implement a resilient, state-driven invite system with Ladder and Sprint strategies using PostgreSQL and sqlc.

**Architecture:** A background orchestrator polls the database for active strategy phases whose timeout has expired. Strategies are reactive to both timeouts and external events (e.g., a person declining).

**Tech Stack:** Go, PostgreSQL, Docker Compose, sqlc, golang-migrate/migrate.

---

### Task 1: Database & Infrastructure Setup

**Files:**
- Create: `docker-compose.yaml`
- Create: `sqlc.yaml`
- Create: `db/schema.sql`
- Create: `db/query.sql`

- [ ] **Step 1: Create docker-compose.yaml**

```yaml
version: '3.8'
services:
  db:
    image: postgres:15-alpine
    environment:
      POSTGRES_USER: user
      POSTGRES_PASSWORD: password
      POSTGRES_DB: invitedoc
    ports:
      - "5432:5432"
```

- [ ] **Step 2: Create initial schema.sql**

```sql
CREATE TABLE persons (
    id UUID PRIMARY KEY,
    email TEXT UNIQUE NOT NULL,
    name TEXT NOT NULL
);

CREATE TABLE groups (
    id UUID PRIMARY KEY,
    name TEXT NOT NULL,
    description TEXT
);

CREATE TABLE group_members (
    id UUID PRIMARY KEY,
    contact_id UUID NOT NULL REFERENCES persons(id),
    group_id UUID NOT NULL REFERENCES groups(id)
);

CREATE TABLE invites (
    id UUID PRIMARY KEY,
    title TEXT NOT NULL,
    description TEXT,
    "from" TIMESTAMP WITH TIME ZONE NOT NULL,
    "to" TIMESTAMP WITH TIME ZONE,
    duration INTERVAL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL,
    status TEXT NOT NULL DEFAULT 'pending'
);

CREATE TABLE invitees (
    id UUID PRIMARY KEY,
    invite_id UUID NOT NULL REFERENCES invites(id),
    contact_id UUID NOT NULL REFERENCES persons(id),
    state TEXT NOT NULL DEFAULT 'pending',
    created_at TIMESTAMP WITH TIME ZONE NOT NULL,
    UNIQUE(invite_id, contact_id)
);

CREATE TABLE invite_phases (
    id UUID PRIMARY KEY,
    invite_id UUID NOT NULL REFERENCES invites(id),
    "order" INT NOT NULL,
    strategy_kind TEXT NOT NULL,
    strategy_config JSONB NOT NULL,
    UNIQUE(invite_id, "order")
);

CREATE TABLE invite_phase_state (
    phase_id UUID PRIMARY KEY REFERENCES invite_phases(id),
    status TEXT NOT NULL DEFAULT 'active',
    next_check_at TIMESTAMP WITH TIME ZONE,
    data JSONB NOT NULL DEFAULT '{}'
);
```

- [ ] **Step 3: Create sqlc.yaml**

```yaml
version: "2"
sql:
  - schema: "db/schema.sql"
    queries: "db/query.sql"
    engine: "postgresql"
    gen:
      go:
        package: "db"
        out: "db"
        emit_json_tags: true
```

- [ ] **Step 4: Create dummy query.sql to test sqlc**

```sql
-- name: GetPerson :one
SELECT * FROM persons WHERE id = $1;
```

- [ ] **Step 5: Run sqlc generate**

Run: `sqlc generate`
Expected: `db/db.go`, `db/models.go`, `db/query.sql.go` created.

- [ ] **Step 6: Commit**

```bash
git add docker-compose.yaml sqlc.yaml db/
git commit -m "chore: setup database infrastructure and schema"
```

### Task 2: Strategy Interface & Models

**Files:**
- Modify: `main.go`

- [ ] **Step 1: Update Invite and Phase models to match DB**

```go
type Invite struct {
	ID          uuid.UUID
	Title       string
	Description string
	From        time.Time
	To          time.Time
	Duration    time.Duration
	CreatedAt   time.Time
	Status      string // pending, active, completed, cancelled
}

type PhaseState struct {
	PhaseID     uuid.UUID
	Status      string // active, completed, failed
	NextCheckAt *time.Time
	Data        json.RawMessage
}
```

- [ ] **Step 2: Update Strategy interface**

```go
type Event struct {
	Kind      string // e.g., "invitee_declined"
	InviteeID uuid.UUID
}

type Strategy interface {
	Kind() StrategyKind
	Execute(ctx context.Context, invite Invite, phase Phase) error
	Resume(ctx context.Context, invite Invite, phase Phase, state *PhaseState) error
	HandleEvent(ctx context.Context, invite Invite, phase Phase, state *PhaseState, event Event) error
}
```

- [ ] **Step 3: Implement empty Resume and HandleEvent for Ladder and Sprint**

```go
func (ls *LadderStrategy) Resume(ctx context.Context, invite Invite, phase Phase, state *PhaseState) error {
	return nil
}

func (ls *LadderStrategy) HandleEvent(ctx context.Context, invite Invite, phase Phase, state *PhaseState, event Event) error {
	return nil
}

func (ls *SprintStrategy) Resume(ctx context.Context, invite Invite, phase Phase, state *PhaseState) error {
	return nil
}

func (ls *SprintStrategy) HandleEvent(ctx context.Context, invite Invite, phase Phase, state *PhaseState, event Event) error {
	return nil
}
```

- [ ] **Step 4: Commit**

```bash
git add main.go
git commit -m "feat: update strategy interface and models"
```

### Task 3: Background Orchestrator

**Files:**
- Create: `orchestrator.go`

- [ ] **Step 1: Implement the Orchestrator loop**

```go
func (app *App) RunOrchestrator(ctx context.Context) error {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			if err := app.ProcessActivePhases(ctx); err != nil {
				fmt.Printf("Error processing phases: %v\n", err)
			}
		}
	}
}
```

- [ ] **Step 2: Implement ProcessActivePhases**

```go
func (app *App) ProcessActivePhases(ctx context.Context) error {
	// 1. Query DB for active phases where next_check_at <= NOW()
	// 2. For each phase:
	//    a. Load Invite and Phase data
	//    b. Load Strategy via LoadStrategy
	//    c. Call strategy.Resume(...)
	//    d. Update state in DB
	return nil
}
```

- [ ] **Step 3: Commit**

```bash
git add orchestrator.go
git commit -m "feat: implement background orchestrator skeleton"
```

### Task 4: Ladder Strategy Implementation

**Files:**
- Modify: `main.go`
- Create: `db/query.sql` (add queries for updating state and invitees)

- [ ] **Step 1: Implement LadderStrategy.Execute**

```go
func (ls *LadderStrategy) Execute(ctx context.Context, invite Invite, phase Phase) error {
	if len(ls.List) == 0 {
		return nil
	}
	// 1. Invite the first person: ls.app.InvitePerson(invite, ls.List[0])
	// 2. Initialize PhaseState in DB: next_check_at = now + ls.Timeout, data = {"index": 0}
	return nil
}
```

- [ ] **Step 2: Implement LadderStrategy.Resume**

```go
func (ls *LadderStrategy) Resume(ctx context.Context, invite Invite, phase Phase, state *PhaseState) error {
	var data struct{ Index int }
	json.Unmarshal(state.Data, &data)

	nextIndex := data.Index + 1
	if nextIndex >= len(ls.List) {
		// Mark phase completed
		return nil
	}

	// 1. Invite next person: ls.app.InvitePerson(invite, ls.List[nextIndex])
	// 2. Update state: index = nextIndex, next_check_at = now + ls.Timeout
	return nil
}
```

- [ ] **Step 3: Implement LadderStrategy.HandleEvent**

```go
func (ls *LadderStrategy) HandleEvent(ctx context.Context, invite Invite, phase Phase, state *PhaseState, event Event) error {
	if event.Kind != "invitee_declined" {
		return nil
	}
	// 1. Check if declined person is the current person in the ladder
	// 2. If yes, trigger immediate Resume logic (move to next person)
	return nil
}
```

- [ ] **Step 4: Commit**

```bash
git add main.go db/query.sql
git commit -m "feat: implement ladder strategy logic"
```

### Task 5: Sprint Strategy Implementation

**Files:**
- Modify: `main.go`

- [ ] **Step 1: Implement SprintStrategy.Execute**

```go
func (ls *SprintStrategy) Execute(ctx context.Context, invite Invite, phase Phase) error {
	// 1. Loop through Recipients and call InvitePerson (using worker pool if needed)
	// 2. Set next_check_at = ls.Deadline (or ls.app.Timeout if relative)
	return nil
}
```

- [ ] **Step 2: Implement SprintStrategy.Resume**

```go
func (ls *SprintStrategy) Resume(ctx context.Context, invite Invite, phase Phase, state *PhaseState) error {
	// If now >= Deadline, mark phase as completed
	return nil
}
```

- [ ] **Step 3: Commit**

```bash
git add main.go
git commit -m "feat: implement sprint strategy logic"
```

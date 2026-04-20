# Main.go Refactoring Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Refactor the bloated `main.go` and `orchestrator.go` into a clean, modular package structure.

**Architecture:**
- **Shared Models:** Leaf package `models/` for all domain structs.
- **Strategies:** Isolated packages in `internal/strategy/` with an interface-driven dependency on the app.
- **App Service:** Core logic moved to `internal/app/`.
- **Wiring:** `main.go` reduced to initialization and server lifecycle.

**Tech Stack:** Go 1.25, standard library packages.

---

### Task 1: Domain Models Migration

**Files:**
- Create: `models/invite.go`, `models/phase.go`, `models/person.go`, `models/group.go`
- Modify: `main.go`, `orchestrator.go`, `api/server.go`, `api/api.gen.go` (via generation), `main_test.go`, `api/server_test.go`

- [ ] **Step 1: Create models package**
Move all domain structs from `main.go` and `orchestrator.go` to the `models/` package.
- `models/person.go`: `Person` (actual struct, not alias)
- `models/group.go`: `Group`, `GroupMember`
- `models/invite.go`: `Invite`, `Invitee`, `InviteeState`
- `models/phase.go`: `Phase`, `PhaseState`, `StrategyKind`

- [ ] **Step 2: Update all imports**
Find and replace all references to these models in `api/`, `email/`, `internal/`, and root.
Expected: Project builds with new model paths.

- [ ] **Step 3: Commit**
```bash
git add models/
git commit -a -m "refactor: move domain models to models package"
```

---

### Task 2: Strategy Decoupling

**Files:**
- Create: `internal/strategy/strategy.go`, `internal/strategy/factory.go`
- Create: `internal/strategy/ladder/ladder.go`, `internal/strategy/sprint/sprint.go`
- Modify: `main.go`

- [ ] **Step 1: Define Strategy interface and Inviter interface**
In `internal/strategy/strategy.go`:
```go
package strategy
import (
    "context"
    "invite/models"
)
type Inviter interface {
    InvitePerson(ctx context.Context, i models.Invite, p models.Person) (*models.Invitee, error)
}
type Strategy interface {
    Kind() models.StrategyKind
    Execute(ctx context.Context, invite models.Invite, phase models.Phase) error
    Resume(ctx context.Context, invite models.Invite, phase models.Phase, state *models.PhaseState) error
    HandleEvent(ctx context.Context, invite models.Invite, phase models.Phase, state *models.PhaseState, event models.Event) error
    Progress(state *models.PhaseState) string
}
```

- [ ] **Step 2: Move Strategy implementations**
Move `LadderStrategy` and `SprintStrategy` to their sub-packages. Update them to use the `Inviter` interface instead of the `App` pointer.

- [ ] **Step 3: Move Factory logic**
Move `LoadStrategy` to `internal/strategy/factory.go`.

- [ ] **Step 4: Commit**
```bash
git add internal/strategy/
git commit -a -m "refactor: isolate strategy implementations"
```

---

### Task 3: App Service and Orchestrator

**Files:**
- Create: `internal/app/app.go`, `internal/app/invites.go`, `internal/app/orchestrator.go`
- Modify: `main.go`, `orchestrator.go` (to be deleted)

- [ ] **Step 1: Move App struct and logic**
Move the `App` struct and its business methods (`InvitePerson`, `StartInviteProcess`, `GetPhaseProgress`) to `internal/app/`.

- [ ] **Step 2: Move Orchestrator logic**
Move `RunOrchestrator` and `ProcessActivePhases` to `internal/app/orchestrator.go`.

- [ ] **Step 3: Delete old orchestrator.go**
`rm orchestrator.go`

- [ ] **Step 4: Commit**
```bash
git add internal/app/
git rm orchestrator.go
git commit -a -m "refactor: move app and orchestrator logic to internal/app"
```

---

### Task 4: Main cleanup and Wiring

**Files:**
- Modify: `main.go`
- Create: `internal/logging/logging.go`

- [ ] **Step 1: Move logging setup**
Move `setupLogger` to `internal/logging/logging.go`.

- [ ] **Step 2: Final main.go cleanup**
Remove all logic from `main.go`. It should only contain `main()` and any package-level variables for wiring.

- [ ] **Step 3: Verification**
Run `go build ./...` and `go test ./...` to ensure everything is still working.

- [ ] **Step 4: Commit**
```bash
git commit -a -m "refactor: final cleanup of main.go"
```

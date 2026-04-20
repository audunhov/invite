# Design Spec: Main.go Refactoring and Modularization

## Overview
Refactor the current `main.go` and `orchestrator.go` into a modular package structure. This will improve maintainability, testing, and prevent the entry point from becoming a "kitchen sink" file.

## Goals
- Split shared domain models into a dedicated package.
- Isolate strategy implementations to allow for easy extension.
- Move business logic out of the global `main` package.
- Simplify `main.go` to focus on wiring and lifecycle management.

## Proposed Package Structure

### 1. `models/`
Shared domain structs used across the entire application.
- `invite.go`: `Invite`, `Invitee`, `InviteeState`.
- `phase.go`: `Phase`, `PhaseState`, `StrategyKind`.
- `person.go`: `Person` (alias or new struct).

### 2. `internal/strategy/`
The strategy execution framework.
- `strategy.go`: `Strategy` interface, `Event` struct.
- `factory.go`: `LoadStrategy` function.
- `ladder/`: `LadderStrategy` implementation.
- `sprint/`: `SprintStrategy` implementation.

### 3. `internal/app/`
The core application service.
- `app.go`: `App` struct and constructor.
- `invites.go`: `InvitePerson`, `StartInviteProcess`, `GetPhaseProgress`.
- `orchestrator.go`: `RunOrchestrator`, `ProcessActivePhases`.

### 4. `main.go`
The entry point.
- Configuration loading.
- Database connection and migration.
- Dependency injection (Wiring `App`, `Server`, `EmailService`).
- HTTP server initialization and graceful shutdown.

## Circular Dependency Resolution
Currently, strategies depend on `App` for `InvitePerson`. 
**Solution:**
Strategies will now depend on a `Inviter` interface:
```go
type Inviter interface {
    InvitePerson(ctx context.Context, i models.Invite, p models.Person) (*models.Invitee, error)
}
```
The `App` struct will implement this interface. The strategy factory will inject the `App` (as an `Inviter`) and the database `Queries` into the strategies.

## Migration Steps
1. Create new directory structure.
2. Move models and update imports (this is the most widespread change).
3. Move strategy implementations and introduce the `Inviter` interface.
4. Move `App` methods and Orchestrator logic.
5. Clean up `main.go`.

## Success Criteria
- [ ] Application builds and runs exactly as before.
- [ ] No logic is left in `main.go` except for wiring.
- [ ] Strategies are isolated in their own packages.
- [ ] Unit tests (if any) are updated to reflect the new paths.

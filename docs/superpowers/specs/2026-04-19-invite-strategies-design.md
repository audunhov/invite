# 2026-04-19 Invite Strategies State Management Design

## Objective
Implement a resilient, state-driven execution system for `LadderStrategy` and `SprintStrategy`. The system must handle long-running timeouts (hours/days) and immediate transitions based on user actions (declines) while surviving service restarts.

## Proposed Architecture

### 1. Data Schema
To track execution state, we'll introduce a state tracking mechanism in the database.

**Table: `invite_phase_state`**
- `phase_id`: UUID (Primary Key, FK to `Phase`)
- `status`: String (`active`, `completed`, `failed`)
- `next_check_at`: Timestamp (When the orchestrator should next process this phase)
- `data`: JSON (Strategy-specific transient state, e.g., current index for Ladder)

### 2. Strategy Interface
The `Strategy` interface will be expanded to support asynchronous, stateful execution.

```go
type Strategy interface {
    Kind() StrategyKind
    // Execute starts the phase and initializes state
    Execute(ctx context.Context, invite Invite, phase Phase) error
    // Resume is called by the orchestrator when next_check_at is reached
    Resume(ctx context.Context, invite Invite, phase Phase, state *PhaseState) error
    // HandleEvent is called when an external event occurs (e.g., a person declines)
    HandleEvent(ctx context.Context, invite Invite, phase Phase, state *PhaseState, event Event) error
}
```

### 3. Orchestration Logic
- **Background Worker**: A ticker-based goroutine that periodically queries `invite_phase_state` where `next_check_at <= NOW()` and `status = 'active'`.
- **Event Dispatcher**: When an `Invitee` state changes (e.g., to `declined`), the system looks up the active `PhaseState` for that invite and calls `HandleEvent` on the corresponding strategy.

## Strategy Implementation Details

### Ladder Strategy
- **`Execute`**: Sends to the first person in `List`. Sets `next_check_at = NOW() + Timeout`. Sets `data = {"current_index": 0}`.
- **`Resume`**: If `current_index < len(List) - 1`, increment index, send next invite, update `next_check_at`. Else, mark phase `completed`.
- **`HandleEvent`**: If event is `declined` and the person matches `List[current_index]`, immediately trigger the same logic as `Resume` but with `next_check_at = NOW()`.

### Sprint Strategy
- **`Execute`**: Iterates through `Recipients`, sending all invites concurrently (using a worker pool).
- **`Resume`**: Checks if the `Deadline` has passed. Marks phase `completed`.
- **`HandleEvent`**: (Optional) Could be used to track progress, but `Sprint` primarily waits for the `Deadline` or all responses.

## Success Criteria
- [ ] `LadderStrategy` moves to the next person after `Timeout`.
- [ ] `LadderStrategy` moves to the next person immediately upon `Decline`.
- [ ] Strategies resume correctly after a process restart.
- [ ] Failed notification attempts are logged/tracked in `SprintStrategy`.

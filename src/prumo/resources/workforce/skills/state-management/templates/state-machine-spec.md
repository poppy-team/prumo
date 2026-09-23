# State Machine Specification: {{Machine.Name}}

## 1. Overview & Domain Boundaries
- **Machine ID**: `{{machine-id}}`
- **Initial State**: `{{initial_state}}`
- **Context Schema**:
  ```typescript
  export interface {{Machine.Name}}Context {
    // Extended state attributes
    id: string;
    retryCount: number;
    error: Error | null;
  }
  ```

---

## 2. State & Event Definitions

### States
- `idle`: Initial resting state awaiting trigger.
- `loading`: Asynchronous operation in flight.
- `success`: Terminal/settled successful state.
- `failure`: Error state with retry options.

### Events
- `START`: Triggers workflow start.
- `RESOLVE`: Indicates operation success with payload.
- `REJECT`: Indicates operation failure with error details.
- `RETRY`: Re-attempts operation from failure state.
- `RESET`: Clears context and returns machine to `idle`.

---

## 3. Transition Matrix

| Current State | Event | Target State | Guard | Actions (Entry / Exit) |
| :--- | :--- | :--- | :--- | :--- |
| `idle` | `START` | `loading` | None | Reset retry counter |
| `loading` | `RESOLVE` | `success` | Valid payload | Store response data |
| `loading` | `REJECT` | `failure` | None | Record error, increment retries |
| `failure` | `RETRY` | `loading` | `retryCount < maxRetries` | Trigger retry backoff |
| `failure` | `RESET` | `idle` | None | Clear error and retries |
| `success` | `RESET` | `idle` | None | Clear payload |

---

## 4. Concurrency & Side Effects Contract
- **Cancellation**: Dispatching `RESET` while in `loading` must abort the underlying inflight task via `AbortController`.
- **Optimistic Updates**: If applicable, describe pre-mutation snapshot capture and rollback recovery.
- **Persistence**: Document if state or context is serialized across sessions.

---

## 5. Test & Verification Plan
- [ ] Test all legal transitions in the Transition Matrix.
- [ ] Test that illegal events in each state are safely discarded without state mutation.
- [ ] Test guard condition limits (e.g. max retries exceeded).
- [ ] Test cancellation and cleanup on machine reset.

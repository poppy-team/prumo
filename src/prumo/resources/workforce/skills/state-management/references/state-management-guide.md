# State Management Architecture & Patterns: Technical Reference Guide

## 1. Formal State Modeling: Finite State Machines & Statecharts

Complex state logic suffers when represented as collections of independent flags (`isSubmitting`, `isFailed`, `canRetry`). The number of potential states grows exponentially ($2^N$), creating invalid edge cases.

### 1.1 Finite State Machine (FSM) Formal Definition

An FSM is a 5-tuple $(S, \Sigma, \delta, s_0, F)$:
- $S$: A finite set of states (e.g. `Idle`, `Loading`, `Success`, `Error`).
- $\Sigma$: A finite set of input symbols/events (e.g. `SUBMIT`, `RESOLVE`, `REJECT`).
- $\delta$: The transition function $\delta: S \times \Sigma \to S$, mapping a current state and event to the next state.
- $s_0$: The initial state ($s_0 \in S$).
- $F$: The set of final/terminal states ($F \subseteq S$).

### 1.2 Harel Statecharts Extensions
David Harel's Statecharts extend standard FSMs with:
1. **Hierarchy (Nested States)**: Common behavior shared across child states (e.g., handling `CANCEL` across any sub-step of a checkout flow).
2. **Orthogonality (Parallel States)**: Independent state machines running concurrently (e.g., text editing state and network synchronization state).
3. **Guards & Extended State (Context)**: Conditional transitions based on quantitative data (e.g., `event.amount > 0`).

---

## 2. Unidirectional Data Flow & Immutability

```
   +------------------------------------------------------+
   |                                                      |
   v                                                      |
[Event / Action] ---> [Pure Reducer / FSM] ---> [New State] ---> [View / Subscriptions]
```

### 2.1 Pure Transition Functions
A state transition function must be strictly pure:
- Deterministic: Same state + same event = identical next state.
- Zero side-effects: Network requests, storage writes, and timer scheduling must be delegated to action effect runners outside the reducer.
- Immutability: Modify state via structural sharing. Never mutate existing objects in place.

---

## 3. Server State vs. Client State

| Property | Client Domain State | Server Cache State |
| :--- | :--- | :--- |
| **Ownership** | Exclusively owned by client runtime | Owned by remote backend/database |
| **Persistence** | Ephemeral or local storage | Authoritative remote database |
| **Freshness** | Synchronously up to date | Potentially stale immediately |
| **Lifecycle** | Follows UI session / component | Requires TTL, invalidation, refetching |
| **Examples** | Theme, current step, open modal | User profile, product catalog, cart |

### 3.1 Optimistic Updates with Rollback
When user experience demands instantaneous feedback:
1. **Snapshot**: Capture current state $S_0$.
2. **Optimistic Transition**: Apply predicted state $S_{\text{predicted}}$ immediately to the UI store.
3. **Dispatch Remote Mutation**: Trigger asynchronous API call.
4. **On Success**: Reconcile with server payload.
5. **On Error**: Roll back store to snapshot $S_0$, and dispatch failure notice to user.

---

## 4. Fine-Grained Reactivity: Signals & Dependency Graphs

Modern reactivity models (Signals) represent reactive values as nodes in a directed acyclic graph (DAG):

- **Signal (Source)**: Holds a value and tracks subscribers.
- **Computed (Derived)**: Memoized pure derivation of one or more signals.
- **Effect (Sink)**: Side-effect triggered when dependencies change.

### 4.1 Glitch Freedom via Topological Sorting
When signal $A$ updates, and derived values $B = f(A)$ and $C = g(A, B)$ exist, evaluating $C$ before $B$ produces a "glitch" (an intermediate inconsistent state). Modern signal implementations perform a topological sort or push-pull phase (e.g. marking dirty, then pulling on demand) to guarantee that consumers observe only completely settled states.

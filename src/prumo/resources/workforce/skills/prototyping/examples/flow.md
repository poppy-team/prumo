# User Flow & Cognitive Walkthrough Example

## Scenario: First-Time Project Bootstrap & Authentication

### 1. Journey Step Matrix

```
[1. Landing View] ──(Click 'Start')──▶ [2. Organization Setup]
                                              │
                         ┌────────────────────┴────────────────────┐
                         ▼                                         ▼
            [3a. Success / Key Gen]                     [3b. Error / Duplicate Org]
                         │                                         │
                         ▼                                         ▼
                 [4. Active Dashboard]                     [Retry Input]
```

---

## 2. Cognitive Walkthrough Analysis

### Step 1: User lands on empty dashboard
1. **Will user try to achieve the right effect?**
   - *Yes*: User wants to initialize their first workspace.
2. **Will user notice that the action is available?**
   - *Yes*: Primary CTA button *"Create Workspace"* is visually dominant with high contrast ratio (7.2:1) and positioned at the visual center.
3. **Will user associate action with effect?**
   - *Yes*: Label clearly says *"Create Workspace"* with a plus icon.
4. **Will user see progress is being made?**
   - *Yes*: On click, button displays inline micro-spinner and smoothly animates to modal step.

### Step 2: Workspace Name Input
1. **Will user try to achieve the right effect?**
   - *Yes*: Typing workspace identifier.
2. **Will user notice input field?**
   - *Yes*: Auto-focused text input with subtle border and clear label *"Workspace Identifier"*.
3. **Will user associate action with effect?**
   - *Yes*: Helper text provides real-time slug preview: `prumo.io/workspaces/{slug}`.
4. **Will user see progress is being made?**
   - *Yes*: Green checkmark badge appears when slug availability check succeeds (debounced 200ms).

---

## 3. Interaction State Spec

```javascript
// Interactive state declaration for prototype harness
export const workspaceFormMachine = {
  initial: 'idle',
  states: {
    idle: {
      on: { INPUT_CHANGE: 'validating' }
    },
    validating: {
      on: {
        SLUG_VALID: 'ready',
        SLUG_TAKEN: 'error'
      }
    },
    ready: {
      on: { SUBMIT: 'submitting' }
    },
    submitting: {
      on: {
        SUCCESS: 'completed',
        FAIL: 'error'
      }
    },
    error: {
      on: { RETRY: 'idle' }
    },
    completed: {
      type: 'final'
    }
  }
};
```

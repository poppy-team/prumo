# User Flow Specification: {{Flow.Title}}

## 1. Flow Metadata & Overview
- **Flow ID**: `{{flow-id}}`
- **Primary Persona**: `{{target_persona}}`
- **Job-To-Be-Done (JTBD)**: {{When I [situation], I want to [motivation], so that I can [expected outcome]}}
- **Entry Points**: [Deep link | In-app CTA | Navigation menu | Notification]
- **Target Success State**: `{{success_terminal_state}}`

---

## 2. Mermaid Flowchart Diagram

```mermaid
flowchart TD
    %% Define entry points
    Start([{{Entry Point}}]) --> ScreenA[{{First Screen / Action}}]
    
    %% Main interaction steps
    ScreenA --> Decision1{{"{{Condition / Fork}}?"}}
    Decision1 -- Yes --> ScreenB[{{Nominal Next Step}}]
    Decision1 -- No --> ErrorA[{{Error State / Form Feedback}}]
    
    %% Error recovery loop
    ErrorA --> ScreenA
    
    %% Success resolution
    ScreenB --> Complete([{{Terminal Success State}}])
```

---

## 3. Step-by-Step State Transition Matrix

| Step ID | Screen / Component | User Action | System Reaction | Next State | Error / Recovery |
| :--- | :--- | :--- | :--- | :--- | :--- |
| **01** | `{{screen_1}}` | Clicks Primary CTA | Validates input format | Step 02 | Field error message, focus shifted |
| **02** | `{{screen_2}}` | Selects Option | Computes summary | Step 03 | Retry button if network fails |
| **03** | `{{screen_3}}` | Confirms Transaction | Executes API mutation | Step 04 | Rollback to Step 02 on card decline |
| **04** | `{{screen_4}}` | Views Confirmation | Sends confirmation email | Complete | N/A (Terminal) |

---

## 4. Edge Cases & Resilience Strategy
- **Session Timeout**: If session expires mid-flow, prompt for re-authentication via modal and restore current form state.
- **Offline / Spotty Connection**: Display offline banner, queue mutation locally, and auto-submit on reconnect.
- **Input Error Handling**: Highlight exact field in error with inline message; do not clear unaffected inputs.
- **Cancel / Abandonment Route**: Provide "Cancel" or "Save & Close" link returning user safely to previous context.

---

## 5. Verification & Acceptance Criteria
- [ ] Happy path completes in $\le {{target_max_clicks}}$ interactions.
- [ ] Zero dead-end screens detected across all branches.
- [ ] All error pathways provide clear remediation and recovery loops.
- [ ] Mermaid diagram passes syntax linting with 0 errors.

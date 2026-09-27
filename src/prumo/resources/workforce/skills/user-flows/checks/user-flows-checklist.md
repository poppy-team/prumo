# User Flows Quality & Completeness Checklist

## 1. Goal Alignment & Task Structure
- [ ] **Single Primary Goal**: The flow serves a clear Job-To-Be-Done with defined success metrics.
- [ ] **Minimal Steps**: Happy path eliminates unnecessary steps, redundant questions, and premature profile requirements.
- [ ] **Clear Entry Points**: All triggers (deeplinks, push notifications, global navigation) have defined landing states.

## 2. Decision Branches & State Handling
- [ ] **Exhaustive Decision Forks**: Every decision diamond handles all mutually exclusive paths (Yes/No, Authenticated/Guest, Granted/Denied).
- [ ] **Zero Dead Ends**: Every screen provides a clear forward action, back navigation, or explicit dismiss/cancel route.
- [ ] **State Representation**: Empty states, loading states, and partial data states are accounted for.

## 3. Error Handling & Recovery Loops
- [ ] **Input Preservation**: Error states retain user-entered input rather than wiping forms.
- [ ] **Actionable Error Messages**: System errors explain what went wrong and how the user can recover.
- [ ] **Retry Mechanisms**: Transient errors (network timeout, payment gateway blip) provide direct retry buttons without restarting the entire flow.

## 4. Asynchronous & Multi-Channel Handoffs
- [ ] **Context Switching**: Flows involving external apps (SMS OTP, banking authenticators) maintain session continuity upon return.
- [ ] **Progress Indicators**: Multi-step workflows clearly communicate current step and total steps (e.g., Step 2 of 4).
- [ ] **Exit Options**: Users can safely exit, save as draft, or postpone completion where appropriate.

## 5. Diagrammatic Notation & Documentation
- [ ] **Mermaid Syntax**: Flowchart is syntactically valid and uses standard shapes (rectangles for steps, diamonds for decisions, cylinders for data).
- [ ] **Label Precision**: Edges clearly state user actions ("Clicks Submit", "Declines Permission") or system outcomes ("HTTP 200", "Timeout").

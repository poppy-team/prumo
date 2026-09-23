# Competitor Teardown Specification — Team Onboarding

## Scope
- **Products**: Acme Cloud 4.2 web, Nimbus 8.1 web, Orbit 2.6 web
- **Workflow**: create account → create team → invite teammate → reach shared board
- **Capture window**: 2026-09-16 to 2026-09-23
- **Rubric**: fixed 1–5 heuristic and accessibility scale

## Evidence Register
| ID | Product | Step | Observation | Capture |
|---|---|---|---|---|
| OBS-01 | Acme Cloud 4.2 | 2 | Team name and workspace type appear on one screen | `AC-04-2.png` |
| OBS-02 | Nimbus 8.1 | 3 | Invitation is deferred until board creation | `NI-08-3.mp4`, 00:41 |
| OBS-03 | Orbit 2.6 | 4 | Invitee role options use internal permission jargon | `OR-02-4.png` |

## Scorecard
| Product | Visibility | Control | Error recovery | Keyboard path | Notes |
|---|---:|---:|---:|---:|---|
| Acme Cloud | 4 | 4 | 3 | 4 | No undo after workspace creation |
| Nimbus | 3 | 2 | 4 | 3 | Strong draft recovery, delayed invite |
| Orbit | 2 | 3 | 2 | 2 | Role vocabulary needs testing |

## Opportunities
1. **Rank 1 — preserve draft state**: high frequency, medium effort. Validate with interrupted-onboarding tests.
2. **Rank 2 — plain-language roles**: medium impact, low effort. Validate with a card sort of five role labels.
3. **Rank 3 — contextual invite**: high impact, high effort. Defer until demand is measured.

## Decision
Use plain-language role labels and retain local drafts. Treat Orbit's permission model as inspiration, not a specification.

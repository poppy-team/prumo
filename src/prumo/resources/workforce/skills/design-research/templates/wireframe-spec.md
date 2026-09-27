# Design Research Plan — Merchant Payout Clarity

## Decision Question
Do merchants understand payout status and the next required action before funds settle?

## Research Inputs
- Product goal: reduce payout-status support contacts.
- Participants: 8 US merchants who receive payouts at least monthly.
- Context: desktop and mobile, first 30 days after payout onboarding.
- Sources: product analytics, 14 support conversations, current payout flow, competitor references.

## Method
1. Conduct eight moderated task sessions.
2. Ask participants to locate the last payout, inspect its status, and explain what happens next.
3. Capture completion, hesitation, backtracking, and expected settlement date.
4. Review public references separately and label observations apart from interpretation.

## Evidence Record
| ID | Type | Finding | Confidence |
|---|---|---|---|
| R-01 | Observation | 6 of 8 opened Support to interpret `Processing` | High |
| R-02 | Inference | Status copy may not define the completion condition | Medium |
| R-03 | Observation | 4 of 8 expected a bank date before entering the detail view | High |

## Synthesis
The flow should state whether funds are sent, expected delivery, and whether action is required. Confirm wording with a follow-up comprehension test before implementation.

## Stop Rule
Stop discovery when two consecutive participants reach the same interpretation with no new failure mode. Keep contradictions visible in the evidence register.

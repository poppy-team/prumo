# Information Architecture Reference Guide

## Inventory Before Hierarchy

List every screen, entity, action, setting, help topic, and cross-link with its current owner and entry points. Mark orphans, dead ends, duplicate destinations, and inaccessible legal or safety content before drawing a new tree.

## Taxonomy and Labels

Group by the user's mental model, not the organization chart. Validate labels with card sorting or tree testing. Keep user vocabulary, reject internal jargon, and record why close alternatives lost.

## Navigation Budgets

Set explicit depth limits for global navigation, settings, and transactional flows. Critical tasks should have a primary path and at least one efficient recovery path such as search, deep link, or recent items. Depth is a budget, not a preference.

## Progressive Disclosure

Show common decisions on the primary surface and move advanced controls behind explicit, understandable affordances. Never hide destructive consequences, required consent, error context, or a task's only completion path.

## Wayfinding Tests

Tree tests measure labels and hierarchy without clickable screens. First-click tests measure the first action on a real surface. Record task success, path, hesitation, backtracking, and completion time against a predeclared threshold.

## Patterns and Anti-Patterns

| Do | Avoid |
|---|---|
| Test labels before visual polish | Assume a familiar label is findable |
| Preserve direct and recovery paths | Hide critical tasks behind search only |
| Remove or merge content when depth is exceeded | Add a deeper level for convenience |
| Make advanced scope visible | Hide complexity without explanation |
| Track failed wayfinding paths | Count page views as task success |

## Short Example

A billing-history task sits at depth three and only in `Settings`. The revised tree exposes `Billing` in primary navigation, keeps `History` one level below it, and provides a receipt deep link as the recovery path.

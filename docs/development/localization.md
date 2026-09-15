# Localization & Translation

> Authority: repository-canonical (W6). Schemas:
> `schemas/translation-record.schema.json`, `schemas/prumo.schema.json`.

## Canonical source locale

English (`en`) is the canonical source locale. A translation is **never** a
co-equal source for implementation intent: agent context is sourced from
canonical English unless the task explicitly targets localized user-facing
content (W6.7).

Project configuration declares:

```json
{ "documentation": { "source_locale": "en", "locales": ["en", "pt-BR"] } }
```

## Lifecycle

`missing → machine-draft → needs-review → current → needs-update → stale → superseded`

Staleness is digest-driven: when a source segment changes, the affected
translation segment becomes `needs-update`. Translation is not all-or-nothing —
unaffected segments may be re-affirmed as `current` without retranslating the
whole page.

## Locale-key identity

Translation segments are addressed by a **stable locale key**, not by file path
or line number, so a source rename never orphans a translation
(`schemas/translation-record.schema.json#segments.id`).

## Quality rules

| Rule | Enforcement |
|------|-------------|
| Placeholder integrity | `placeholders_ok` per segment; CI fails on broken placeholders |
| Code/CLI literal protection | Inline code and command literals are never translated |
| Terminology | Glossary IDs pinned via `terminology_revision` |
| Fallback | Never render an empty locale surface; explicit fallback chain |
| Formatting | Date/number/plural policy per locale |
| RTL readiness | Data model is RTL-ready even before an RTL locale ships |
| Pseudo-localization | Verifier catches truncation/expansion before real work |

## Evidence and review

Each translation record carries `reviewer`, `quality_findings` and
`generated_by` (`human` | `machine` | `prumo-assisted`), so a machine draft is
never mistaken for a reviewed translation.

## Coverage

A translation coverage report per release lists each locale's current /
needs-update / stale / missing share. Agent-retrieved documentation remains
English by default; localized content is user-facing only.

## Verification

- `prumo docs translate status` (W20) — per-locale coverage.
- Test fixtures: source-change → `needs-update` (audit case D006).

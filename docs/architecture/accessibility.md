# Accessibility Contracts

> Authority: repository-canonical (W11). Contracts: `docs/contracts/builtin.json`.
> Reference: WCAG 2.2 and WCAG2ICT for non-web surfaces.

Accessibility is decomposed into independently verifiable sub-contracts so a
project can satisfy them proportionally to its surface, and so evidence can be
required per contract rather than as a vague whole.

## Sub-contracts

| Contract | Obligation | Typical evidence |
|----------|------------|------------------|
| `accessibility.keyboard` | Every action reachable and operable by keyboard | automated + manual |
| `accessibility.focus` | Focus visible, ordered, never trapped or lost | visual + manual |
| `accessibility.contrast` | Text and non-text contrast measured | automated |
| `accessibility.screen-reader` | Name/role/value and structure exposed | manual |
| `accessibility.motion` | Motion optional; reduced-motion honoured | automated + manual |
| `accessibility.zoom-reflow` | Reflow under zoom without loss | manual |
| `accessibility.target-size` | Minimum target size with documented exceptions | automated |
| `accessibility.drag-alternatives` | Non-drag alternative for drag interactions | manual |
| `accessibility.accessible-auth` | Auth without a cognitive function test | manual |
| `accessibility.cognitive-clarity` | Predictable language, errors and navigation | manual or exception |

Surface-specific contracts extend these: `tui.accessibility` (terminal color
capability, Unicode fallback, no-color mode, screen-reader strategy) and
`ui.platform-conventions` (capability matrix and graceful fallback).

## Evidence classes

`automated`, `manual`, `visual`, `interaction`, `platform-specific`,
`exception-with-rationale`.

**Invariant (W11.6):** automated checks alone never constitute full
accessibility verification. Where a contract requires manual evidence, an
automated pass does not complete it. A waived obligation must be recorded as
`exception-with-rationale`, never silently dropped.

## Surface mapping

| Surface | Mandatory evidence |
|---------|--------------------|
| Web / desktop GUI | automated contrast + keyboard, manual screen reader, visual focus |
| TUI | keyboard-only completeness, no-color rendering, Unicode fallback, golden frames |
| CLI | keyboard-redundant output, non-color-dependent signalling, plain language |

## Verification

- Contract presence and shape: `conformance/schema-runtime` (W1).
- Per-project obligations: `prumo docs audit` / `prumo docs readiness` (W15 semantics).
- Evidence freshness: media and evidence records (W20).

# H10 — TUI Spike Scope (Prumo Code first client)

Status: PROPOSED (becomes the Goal contract when the H10 increment starts).
Sources: Living Book pages 13 (TUI spec), 14 (tech matrix), 18 (roadmap), 21 (Stack H), 23 (Shared Product Contract) — snapshot `2026-09-11-d41f18bb65f5`.

## Goal

Prove that the Prumo Harness can drive a real Zed-like terminal client:
one Go/Bubble Tea v2 binary that connects to `prumo agentd` over the Agent
Protocol and delivers the agent-first core loop (palette → run → stream →
approve → evidence), with zero `internal/` imports.

## Preconditions (tracked, not TUI code)

1. Current branch landed: PR to `main` (governance).
2. Event reconnect/replay verified as a client-visible contract (JSONL replay
   and daemon reconnect exist; add a client-facing conformance test if needed).
3. Shared Product Contract (entities, state ownership, capabilities) promoted
   from Living Book page 23 into repo contracts before first layout commit.

## Stack (ACCEPTED — Stack H)

- Bubble Tea v2, Bubbles v2, Lip Gloss v2, Glamour v2, Huh v2 (forms only).
- `charmbracelet/x/vt` + `x/xpty` allowed behind a Prumo interface only; no
  experimental API in public contracts.
- Wish v2 (SSH serving) is P1 — out of the spike.
- Go only; no canonical logic duplication: TUI is a pure protocol client.
  In the simple mode the daemon may be embedded/in-process, same protocol.

## In scope (spike)

- App shell: panes, tabs, thin status bar, focus model (Zed-like, contrast
  over boxes; semantic tokens: surface, panel, border-subtle, text-muted,
  accent, success, warning, error, focus).
- Command palette (fuzzy, palette-first navigation).
- Agent run panel: goal input, live event stream render (AgentEvent), tool
  activity via spinner states.
- Permissions/approvals inline surface.
- File tree (read-only) + evidence/gates mini-panel.
- Provider modes: embedded daemon (simple mode) + remote daemon (TLS+token).

## Out of scope (explicit)

- Rope editor, multi-cursor, LSP autocomplete UI (editor v0 = Bubbles
  textarea + open-in-external-editor; editor v1/v2 later).
- PTY terminal multiplexer (x/vt wiring interface only).
- Wish/SSH serving, themes/i18n packs, mouse-heavy interactions.
- Desktop/Floem parity work.

## Acceptance criteria

1. `TestBoundaryNoInternalImports` equivalent: the TUI package imports only
   public SDK/protocol packages (no `internal/`).
2. Live: start embedded daemon with FakeProvider, run a Goal from the palette,
   observe streamed events and tool activity, approve one permission, see
   evidence/gate result — no restart of the TUI during the flow.
3. Kill the daemon mid-run, restart, reconnect and replay the run's events
   without UI state corruption.
4. Remote mode: same flow against `agentd` over TCP+TLS+token.
5. Redraw on a ~200-line event stream stays under the spike budget measured
   in ADR 009 methodology (baseline recorded, no threshold invented).

## Stopping condition

Spike ends when criteria 1–4 pass once end-to-end (recorded as evidence in
the gap-register/roadmap) or when a blocker proves Bubble Tea v2 cannot carry
the shell — in that case the exit artifact is a written finding for the
strategic-challenger re-evaluation, not a silent stack drift.

# Authority & Projection Policy

> Authority: repository-canonical. Single source of truth for documentation
> authority order, roles and drift rules (W0.6, W0.8, W0.11).

## Authority order

1. Canonical repository specs, schemas, contracts and accepted ADRs
2. Accepted engineering documentation under `docs/`
3. Prumo Living Book (Notion) — design input until promoted
4. Agent inference
5. External sources

Once a design decision is promoted into a canonical repository specification,
the repository version has authority over the Notion version for implementation.

## Roles

| Role | Meaning | Drift-checked |
|------|---------|---------------|
| `canonical` | Owns project facts | yes |
| `projection` | Derived surface; must resolve to a canonical source | yes |
| `historical` | Record (ADR, migration note, progress log); never rewritten | no |

The machine-readable classification is `docs/AUTHORITY_MAP.json`, verified by
`prumo docs authority` and the `docs-authority` CI gate.

## Projection policy (W0.11)

- Agent and provider adapters — `AGENTS.md`, Copilot/Cursor/Claude rule files,
  connector manifests, generated SDKs — are **projections**: they may not
  introduce independent project facts.
- A projection declares its `canonical_source`. A projection whose source is
  itself a projection is a **cycle** and fails the gate.
- Generated adapters carry provenance/fingerprint metadata once the Agent
  Surface Compiler (W16) lands.
- Derived indexes, summaries and Working Context Capsules live in runtime/cache
  and never become canonical.

## Drift rules (W0.8)

- An **active** document (canonical or projection) may not reference a project
  release line older than the current CLI version unless the line carries a
  historical annotation (`historical`, `legacy`, `migration`, `retired`,
  `superseded`, `deprecated`, `no longer`, `compatibility oracle`).
- Routing surfaces (`AGENTS.md`, `ENTRYPOINT.md`, `README.md`, `FRAMEWORK.md`,
  `docs/PRUMO.md`) must have resolvable relative links.
- Per-document exemptions require an explicit `drift_exempt_reason` in the
  authority map — never a silent weakening of the gate.

## Verification

```bash
prumo docs authority            # human output
prumo docs authority --json     # machine envelope
go test ./internal/documentation/ -run TestAuthority
```

CI gate group: `docs-authority` (see `docs/development/waves.md`).

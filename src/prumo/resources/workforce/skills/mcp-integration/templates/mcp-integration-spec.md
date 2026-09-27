# MCP Integration Specification — Harbor Issue Tracker Server (Release 1.6.0)

## 1. Surface and Topology
- **Protocol**: MCP `2025-06-18`; SDK `@modelcontextprotocol/sdk` 1.12.0; transport remote Streamable HTTP.
- **Clients**: Harbor Desktop 3.1, CI agent runner 0.8; OAuth via Harbor Identity with `tracker:read` / `tracker:write` scopes.
- **Candidate triage**: 23 operations reviewed, 9 exposed (6 reads, 3 writes); 14 excluded (raw SQL export, bulk delete, admin role grant — see section 5).
- **Spec date and owner**: 2026-09-16, integrations pod (Rafael Okafor).

## 2. Tool Catalog

| Tool | Scope | Destructive | Input properties | Rate limit |
|---|---|---|---|---|
| `issue_search` | tracker:read | no | query (max 200), labels enum, cursor, limit max 50 | 200/min |
| `issue_get` | tracker:read | no | id integer | 200/min |
| `issue_list_mine` | tracker:read | no | state enum, cursor, limit max 50 | 120/min |
| `comment_list` | tracker:read | no | issueId, cursor, limit max 50 | 200/min |
| `label_list` | tracker:read | no | none | 60/min |
| `attachment_get` | tracker:read | no | attachmentId, max 25 MB | 30/min |
| `issue_create` | tracker:write | no | title, body, labels, confirm | 60/min |
| `comment_add` | tracker:write | no | issueId, body, confirm | 120/min |
| `issue_close` | tracker:write | yes | issueId, reason enum, confirm: true | 30/min + approval |

## 3. Resources and Errors
- **Resources**: `tracker://issues/{id}`, `tracker://issues/{id}/comments` with cursor pagination (50/page) and ETags; internal-only fields (`reporter_email`, `cost_center`) stripped at serialization, verified by field-diff test.
- **Error codes**: `-32001` not found, `-32002` forbidden, `-32003` throttled, `-32004` confirm-required; probe battery of 40 cases green with zero internal leakage.
- **Descriptions**: declarative only; injection review passed 2026-09-16 (reviewer T. Lindqvist).

## 4. Verification Results

| Gate | Result |
|---|---|
| Schema strictness (SDK validator) | 9 of 9 tools strict, CI-pinned |
| Load test 200 calls/min x 15 min | p95 610 ms, zero scope violations |
| Audit log sample (2026-09-16) | tool, principal, args hash present on 100% of 3,000 sampled calls |
| Confirm gates | `issue_close` without confirm rejected `-32004`; approval logged |

## 5. Excluded Operations (Deny by Default)
- `sql_export`, `bulk_delete`, `role_grant`, plus 11 admin/reporting ops: excluded as god tools or unscoped writes; re-review requires security sign-off (ticket INT-551).

## 6. Regression Evidence
- [x] Tool catalog v1.6.0 published with examples and changelog.
- [x] Probe, load, and audit evidence archived under `mcp/evidence/1.6.0/`.
- [x] `scripts/verify.sh` exits 0 on the skill package; integration gate green on server commit 60ad31.

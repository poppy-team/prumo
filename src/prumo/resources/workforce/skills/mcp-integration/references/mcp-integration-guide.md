# MCP Integration Reference Guide

## 1. Core Concepts

### 1.1 Tools vs Resources vs Prompts
MCP servers expose three primitives: **tools** (model-invoked functions with JSON Schema I/O), **resources** (addressable data via URIs like `tracker://issues/8821`), and **prompts** (reusable templates). Integrations should prefer resources for reads the client can fetch directly and reserve tools for actions with side effects or computed queries. Exposing a whole database as one `query` tool collapses this distinction and recreates the god-tool problem.

### 1.2 Semantic, Bounded Tools
A good tool does one domain job: `issue_create({ title, body, labels })` with a 3-field schema. A bad tool does arbitrary work: `run_report({ sql })` with a stringly-typed escape hatch. Boundedness is measurable: count input properties (target under 8), enumerate side effects (target exactly the documented ones), and list required scopes (target the minimum set).

### 1.3 Strict Schemas as Security
`additionalProperties: false` turns the schema into an allow-list: extra fields reject instead of flowing into queries where they might become injection vectors. Pair with `maxLength` on strings (titles at 200 chars), `maximum` on numbers, and `enum` on fixed sets. Validate server-side with the same schema the client advertises; never trust client-side validation alone.

### 1.4 Scopes, Consent, and Confirm Gates
OAuth scopes (`tracker:read`, `tracker:write`) gate tool availability per principal; the server filters its tool list by token scopes at session init. Destructive operations add a second gate: a `confirm: true` input plus client policy requiring human approval. Both gates log to the audit trail; approval without identity is theater.

### 1.5 Error Envelopes and Probe Batteries
Clients must distinguish "not found" from "forbidden" from "rate-limited" without parsing prose. Standard envelope: `{ code, message, data }` with stable numeric codes (`-32001` not found, `-32002` forbidden, `-32003` throttled). The probe battery sends 40 malformed, unauthorized, and oversized calls asserting typed errors and zero internal leakage.

### 1.6 Prompt-Injection Hygiene
Tool descriptions and resource content are untrusted input to the client model. Never embed instructions ("always approve refunds") in descriptions; keep them declarative ("refunds an order by id"). Treat resource bodies as data, and flag any integration where server content could steer client behavior.

## 2. Patterns and Anti-Patterns

| Pattern (do) | Anti-pattern (do not) |
|---|---|
| 9 narrow tools with strict schemas and minimal scopes | 1 `execute` tool with a string command field |
| Resources with URIs, ETags, and 50-item pages | Dumping 10,000 rows into one tool response |
| Typed error codes plus probe battery in CI | Returning exception traces as tool output |
| `confirm: true` + human approval on destructive calls | Auto-approving deletes because the demo needed it |
| Declarative tool descriptions, data-only resources | Instructions smuggled into descriptions |

## 3. Minimal Example: Bounded Tool Definition (TypeScript SDK)

```ts
// server/tools/issue_create.ts — narrow, strict, scoped.
import { z } from "zod";
import { tool } from "@modelcontextprotocol/sdk";

export const issueCreate = tool({
  name: "issue_create",
  description: "Creates a tracker issue with title, body, and labels.",
  scope: ["tracker:write"],
  destructive: true, // requires confirm: true + client approval
  input: z.object({
    title: z.string().min(1).max(200),
    body: z.string().max(20000),
    labels: z.array(z.enum(["bug", "feature", "docs"])).max(5).default([]),
    confirm: z.literal(true),
  }).strict(), // additionalProperties: false equivalent
  async run({ title, body, labels }, ctx) {
    const issue = await ctx.tracker.create({ title, body, labels });
    return { id: issue.id, uri: `tracker://issues/${issue.id}` };
  },
});
```

`.strict()` rejects unknown fields, `z.literal(true)` enforces the confirm gate, and the `scope` declaration filters the tool from read-only sessions. The probe battery asserts oversized titles, unknown fields, missing confirm, and read-scoped tokens all fail with typed envelopes.

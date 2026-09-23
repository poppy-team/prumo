# MCP Tooling Reference Guide

## 1. Core Concepts

### 1.1 Protocol Versioning and Capability Negotiation
MCP evolves via dated protocol versions (`2025-06-18`). The `initialize` handshake exchanges versions and capability flags (tools, resources, prompts, sampling, elicitation); each side enables only the intersection. Servers must reject unknown majors loudly (`-32099`) rather than limping on a mismatched contract — silent downgrades cause the worst interop bugs.

### 1.2 Transports: stdio vs Streamable HTTP
**stdio** spawns the server as a subprocess with newline-delimited JSON-RPC over pipes: zero network surface, ideal for local tools, capped by the 10 MB message limit and one-client-per-process model. **Streamable HTTP** serves remote clients with session IDs, resumable streams, and heartbeats: multi-client and network-exposed, requiring TLS, auth, and session stores. Supporting both doubles the conformance surface; advertise only what the battery covers.

### 1.3 Sessions, Resume, and Exactly-Once Semantics
HTTP sessions bind a client to server-side state (subscriptions, progress tokens, partial results). On disconnect, the client reconnects with its session ID inside the heartbeat window (60 s) and resumes pending calls. Design tools idempotently (client-generated `callId` deduplicated server-side) so a retried call after a dropped connection executes its side effect exactly once.

### 1.4 Timeouts, Backoff, and Progress
Networks fail; tooling must fail well. Request timeout 30 s with typed errors, reconnect backoff 200 ms doubling to a 5 s cap over 8 attempts with jitter, progress notifications every 5 s on long calls so clients can display status instead of timing out. Cancellation (`notifications/cancelled`) must interrupt server work promptly — a cancel that keeps burning GPU for 10 minutes is a resource bug.

### 1.5 Sampling and Elicitation
**Sampling** lets servers ask the client model to complete text mid-tool (e.g., summarize before storing); **elicitation** asks the human for input (approve, pick a value). Both cross trust boundaries: constrain sampling to declared prompt templates and log elicitation requests, since a malicious server could otherwise phish through the client UI.

### 1.6 Conformance Battery Discipline
The 214-case battery covers framing, batching, ordering, cancellation, error shapes, and version edges. Run it per transport on every SDK bump; a green stdio run says nothing about HTTP session handling. Keep a known-failures file with upstream issue links — "fails, ignored" without a link is how regressions hide.

## 2. Patterns and Anti-Patterns

| Pattern (do) | Anti-pattern (do not) |
|---|---|
| Official SDK, pinned versions, lockfile committed | Hand-rolled JSON-RPC over raw sockets |
| Per-transport conformance runs on every SDK bump | Testing stdio, advertising HTTP too |
| Idempotent tools with client `callId` dedup | Retried calls double-charging side effects |
| Typed timeouts, backoff with jitter, 5 s progress | Infinite hangs and thundering reconnect storms |
| Known-failures file with upstream issue links | Red battery normalized as "flaky infra" |

## 3. Minimal Example: Dual-Transport Server Bootstrap (TypeScript)

```ts
// server/index.ts — harbor-notes-mcp 2.1.0, protocol 2025-06-18.
import { McpServer } from "@modelcontextprotocol/sdk/server/mcp.js";
import { StdioServerTransport } from "@modelcontextprotocol/sdk/server/stdio.js";
import { StreamableHTTPServerTransport } from "@modelcontextprotocol/sdk/server/streamableHttp.js";

const server = new McpServer(
  { name: "harbor-notes-mcp", version: "2.1.0" },
  { capabilities: { tools: {}, resources: { subscribe: true } } },
);
server.registerTool("note_search", { /* strict schema, see mcp-integration */ }, async () => ({}));

// Local: stdio, 10 MB cap, SIGTERM shutdown in 2 s (SDK defaults).
if (process.env.HARBOR_MCP_TRANSPORT !== "http") {
  await server.connect(new StdioServerTransport());
} else {
  // Remote: resumable sessions, 60 s heartbeat, Redis session store.
  await server.connect(new StreamableHTTPServerTransport({
    sessionIdGenerator: () => crypto.randomUUID(),
    heartbeatIntervalMs: 60_000,
    eventStore: new RedisEventStore(process.env.HARBOR_REDIS_URL!),
  }));
}
```

One codebase, two transports, each covered by the battery. The `initialize` handshake advertises exactly the capabilities registered; adding elicitation later means bumping the flag, the tests, and the changelog together.

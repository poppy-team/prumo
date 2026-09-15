# ADR 008: Plugin Isolation Model

# Status

Proposed (decision deferred until plugins exist)

# Context

- GAP-035 asked for a plugin isolation decision from page 19 of the accepted
  design. Prumo currently has no plugin runtime: extension points are
  compile-time (connectors, adapters, MCP servers, AgentProviders).
- The framework forbids unbounded recursion and mandates honest capability
  reporting; any in-process plugin host would fight those invariants.
- External integration surfaces already exist and are process-isolated by
  construction: MCP servers (stdio/HTTP, untrusted-by-default), ACP agents
  (stdio JSON-RPC child process), CLI AgentProviders (`CursorCLI`,
  `CodexCLI`), and external agent servers (`opencode serve`).

# Decision

1. No in-process plugin loader is adopted now. Extensions are out-of-process
   via the existing contracts (MCP, ACP, CLI providers, connectors).
2. When a plugin need is proven by real usage, the default isolation model
   is process-per-plugin over stdio with a typed handshake — the same shape
   as ACPClient — with capability negotiation, bounded output, and
   fail-fast on unavailability.
3. In-process dynamic loading (Go plugin, WASM) is out of scope until an
   ADR amendment justifies it against the isolation and distribution
   invariants.

# Consequences

- GAP-035 is answered with a direction (process isolation) instead of an
  open question, without building speculative infrastructure (YAGNI).
- The existing external-adapter matrix doubles as the plugin substrate;
  hardening work (handshake versioning) benefits both.
- A future plugin marketplace/registry would distribute manifests, not
  binaries loaded into the harness process.

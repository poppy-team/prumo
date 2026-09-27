# ADR 011: No Documentation Telemetry — Query and Intent Tracking Refused

# Status

Accepted (2026-09-15)

# Context

The Documentation Control Plane audit (W21.2/W21.3) proposed two metrics:

- retrieval/query success rate, "without storing sensitive user content";
- the top missing or failed documentation intents.

Both require observing what a user or agent **asked for**. Prumo does not have
that observation:

- There is no telemetry channel. The harness records run-scoped events
  (`obs-<run>.jsonl`, `evidence-<run>.jsonl`) for one run the operator started,
  on that operator's machine, and prunes them by retention policy. There is no
  aggregation endpoint, no vendor SDK and no phone-home path.
- Documentation retrieval is a **read surface**: `prumo-agent docs query` reads a
  generated index, `prumo-agent docs resources read` serves a resource over MCP, and
  `prumo-agent docs metrics` computes figures from repository state. None of these
  writes a query log, and adding one would make the natural-language text of
  every question a persisted artifact.
- The framework's own trust model (`docs/security/trust-model.md`) classifies
  project content as potentially confidential and requires that leaving the
  machine be an explicit, allowlisted act. A retrieval transcript is exactly the
  kind of content that rule exists to protect: it can contain unreleased product
  names, customer identifiers and private repository paths.
- "Success without storing content" is not achievable for these two metrics in a
  useful form. A success rate needs to know which query failed; knowing the
  query means storing it. Storing only a hash makes the second metric (which
  intent was missing) impossible and the first metric nearly useless, because
  identical intents phrased differently hash differently.

# Decision

1. **W21.2 and W21.3 are not implemented.** Prumo collects no retrieval or query
   telemetry, locally or remotely.
2. **Documentation intelligence is derived from repository state only.** Every
   figure in `prumo-agent docs metrics` is computed by reading the repository:
   coverage, stale translations, stale media, broken links, claim drift, token
   counts, documentation debt. A metric that would need to observe a reader is
   out of scope by construction.
3. **If such a metric is ever wanted, it needs a new ADR and an opt-in
   mechanism first** — a local-only, human-invoked report whose scope and
   retention the operator declares, never a default-on channel. The burden of
   proof is on the proposal.
4. **The gap is recorded, not hidden.** `docs/harness/gap-register.md` and
   `docs/development/waves.md` mark W21.2/W21.3 as deliberately not implemented
   with the reason, so future work inherits the decision instead of
   re-discovering it.

# Consequences

- Documentation improvement is driven by observed *repository* defects (stale
  claims, drift, missing evidence, unverified contracts) rather than by
  observed *reader* behaviour. This is a real limitation: Prumo cannot tell
  which existing page people fail to find.
- The limitation is compatible with the framework's provider-neutral,
  local-first posture, and it is auditable: an operator can verify by inspection
  that no query log exists.
- The decision is reversible but not free. Reversing it means introducing a
  telemetry surface into a framework whose trust model currently has none, which
  is a larger change than the two metrics are worth.

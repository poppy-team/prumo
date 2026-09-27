# ADR 021 — Prumo Viewer Extension Contract v1

Status: Accepted (2026-09-24)
Relates to: ADR 008, PART2-CH-17, PHASE-69, CONST-82

## Context

The native viewer needs editor navigation, language intelligence, workspace actions, panels, themes, tasks and diagnostics. These capabilities must be reusable by other projects without linking third-party code to Freya, Rust internals or the Prumo Core socket.

A native dynamic library ABI would couple every extension to the viewer build. In-process loading would also make crashes and state corruption part of the viewer process.

## Decision

1. The viewer owns the UI, document state, permissions, lifecycle and rendering. Extensions never receive Freya components, Rust pointers, mutable internal state or the Core socket.
2. Version `prumo.viewer.extensions/v1` is a wire contract independent from the viewer version and the Agent Protocol.
3. Version one supports two runtime classes:
   - declarative packages for commands, themes, keybindings and host-owned UI contributions;
   - process packages using newline-delimited JSON-RPC 2.0 over stdio.
4. Process packages are started outside the viewer process. They receive a bounded, explicit capability set and a versioned handshake. Failure disables only that extension and never mutates a document implicitly.
5. The viewer is the only component allowed to resolve workspace paths, read or write documents, invoke Git, control terminals or access the Agent Protocol. Extensions request these operations through brokered methods.
6. A package declares its API range, host range, platform targets, capabilities, permissions, dependencies, entrypoints, cleanup policy and contribution IDs in a versioned `manifest.json`.
7. Permission grants are separate from installation. A grant is bound to the extension ID, version and package digest (manifest plus payload). Permission expansion requires a new grant.
8. The v1 package source may be a directory containing `manifest.json` and its payload, or a `.prumoext` TAR+GZ archive with the same manifest contract and detached Ed25519 signature.
9. Contributions use namespaced IDs. Unsupported contributions are rejected during discovery instead of being displayed as fake functionality.
10. The SDK contains only serializable contracts, validation and JSON-RPC helpers. It does not depend on Freya, the viewer or Core.

## Capability families

The contract is designed to carry the following independent extension families:

- editor navigation, selections, decorations and commands;
- LSP and language intelligence;
- workspace search, indexing and file operations;
- Git, tasks, terminal and debugger adapters;
- panels, views, themes, keybindings and renderers;
- diagnostics, timeline, evidence and quality visualizers.

Each family must declare its own capabilities. Installing one family must not silently grant another family's permissions.

## Compatibility

- Manifest schema, package format, extension API, viewer version and Agent Protocol are separate version axes.
- Additive methods may be introduced in a minor API version.
- Breaking changes require a major API version.
- The viewer supports the current minor and the previous two minor API versions.
- Missing required capabilities prevent activation. Optional capabilities may degrade only when declared optional by the extension.

## Trust and distribution

- Declarative local packages are the first supported distribution path.
- Third-party process packages are untrusted by default and remain developer-mode until a platform-appropriate confinement profile is available.
- A process boundary is crash containment, not a complete security sandbox.
- Signing, provenance, revocation, archive extraction and offline bundles are implemented in the SDK boundary; a public marketplace still requires a platform-appropriate confinement profile.
- The registry is discovery only. The package digest and lockfile remain authoritative.

## Consequences

- Other projects can ship independent packages without recompiling the viewer.
- The viewer remains a surface/client and does not become a second Control Plane.
- Some capabilities remain deferred until their host API is real. A package cannot advertise a fake panel, renderer or debugger; implemented contributions are backed by host behavior and tests.
- The SDK and reference packages become separately testable and distributable artifacts.

## Implemented slices

The current implementation provides:

- `prumo-extension-sdk` with typed manifests, validation, grants, package digest binding, bounded JSON-RPC helpers, LSP/DAP clients and process runtime;
- local extension discovery from workspace, development and environment roots, plus lockfile digest enforcement;
- declarative and grant-gated process commands, editor decorations, workspace index/search/read/write, Git and notification broker methods;
- LSP symbols/hover/definition, static panels/views, token themes, global keybindings, tasks and DAP debugger requests;
- `.prumoext` TAR+GZ packaging, Ed25519 detached signatures, trust-store revocation and atomic installation primitives;
- a reference `com.prumo.editor-navigation` package and conformance tests.

The process boundary remains crash containment rather than a complete sandbox. Marketplace release still requires confinement, provenance policy and adversarial testing beyond the local viewer contract.

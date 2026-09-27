# 84.H — Native Graphics/DCC Assurance Profile: GPU, Geometry, Assets e Long Sessions

> Authority: canonical specification.
> Logical ID: 84 H
> Source: Notion Living Book (3e29bb7d023f812d8825ffe1738c4da2)
> Status: Constituição 84: Perfis do Gauntlet / Total Assurance (84 H — Native Graphics DCC Assurance Profile GPU,).


<aside>
🧊

**Profile extension:** assurance especializada para DCC, CAD, modeladores 3D, renderers, game/editor tooling e aplicações desktop nativas com GPU, assets grandes, estado editável complexo e sessões longas.

</aside>

# Por que este profile existe

Aplicações gráficas pesadas acumulam riscos que não aparecem em CRUD/web comum: GPU/driver diversity, VRAM, frame pacing, geometria degenerada, importadores hostis, projetos grandes, caches, Undo profundo, save crash-consistency, long sessions, device loss, shader failure, image decode, plugin/native boundaries e interação direta de alta frequência.

# Graphics/Device matrix

Quando aplicável, declarar:

- graphics backend;
- API/version;
- adapter class;
- integrated/discrete;
- VRAM classes;
- driver/OS matrix;
- software fallback;
- headless capability;
- device-loss behavior.

Unsupported configuration deve falhar claramente, não produzir corruption silenciosa.

# Renderer assurance

Validar separadamente:

- correctness;
- resource lifetime;
- synchronization;
- shader/pipeline compilation;
- device loss/recreation;
- resize/swapchain;
- color space/gamma;
- depth/stencil;
- selection/picking;
- overlay correctness;
- offscreen/export rendering quando aplicável.

Visual output pode exigir golden/reference + tolerance + invariant checks; screenshot bonito não prova buffer/state correctness.

# Frame budget

Editor interativo deve possuir budgets para:

- input latency;
- CPU frame;
- GPU frame;
- frame pacing;
- resize;
- selection/picking;
- gizmo drag;
- paint stroke;
- UV manipulation;
- panel interaction.

Medir percentis/worst spikes quando possível, não somente média.

# VRAM/GPU resource budget

Rastrear textures, meshes, buffers, render targets, pipelines e staging resources. Testar create/delete/reload, workspace switching, document close e repeated import. Growth monotônico sem justificativa é finding.

# Geometry invariants

Após operações:

- indices válidos;
- finite coordinates;
- no NaN/Infinity;
- valid face loops;
- valid references;
- normals/tangents coherent quando required;
- bounds finite;
- selection IDs live;
- UV references coherent;
- topology-specific invariants por operation.

Property/fuzz/metamorphic tests são especialmente úteis em geometry kernels.

# Numerical robustness

Testar:

tiny/large coordinates, near-zero scale, degenerate triangles, duplicate vertices, collinear points, extreme camera near/far, precision boundaries e invalid numeric input. Operação pode recusar caso patológico, mas não crashar/corromper.

# Asset ingestion threat/performance model

Importadores de mesh/image/project são trust boundaries. Testar:

- malformed/truncated;
- huge dimensions/counts;
- integer/offset overflow;
- decompression bombs quando formatos comprimidos;
- pathological topology;
- invalid UTF-8/names;
- path references;
- external resource traversal;
- unsupported extension/content mismatch;
- decoder failure;
- partial import rollback.

# Project durability

Projetos editáveis precisam de:

- versioned format;
- atomic save strategy;
- crash-consistency;
- failed-save semantics;
- autosave/recovery quando implementado;
- backup/temporary cleanup;
- migration tests;
- round-trip tests;
- corruption detection;
- forward/backward compatibility policy.

# Undo/Redo as durability subsystem

History precisa testar:

- deep sequences;
- branching;
- large payloads;
- cancellation;
- cross-workspace operations;
- delete/restore identity;
- memory budget;
- persistence only if intentionally supported.

# Paint/texture assurance

Validar:

- brush continuity;
- sampling;
- seam behavior;
- channel/layer semantics;
- dirty-region correctness;
- synchronization 3D↔2D;
- texture bounds;
- layer compositing;
- memory/VRAM;
- save/reload;
- color space.

# UV assurance

Validar:

- finite UV coordinates;
- topology mapping;
- seam data;
- unwrap/pack determinism/tolerance;
- island transform;
- selection sync;
- invalid/degenerate geometry;
- save/reload;
- performance em meshes maiores.

# Long-session editor profile

Soak deve combinar ações reais:

open project → model → paint → UV → import → delete → Undo/Redo → save → switch workspace → repeat.

Observar RAM, VRAM, handles, tasks, queues, caches, save latency e frame pacing.

# Concurrency

Background import, thumbnails, autosave, shader compilation e asset processing precisam de cancellation, ownership, backpressure, stale-result rejection e safe handoff para UI thread.

# Plugin/script boundary

Quando existir plugin/script:

permissions, filesystem/network/process scopes, time/resource limits, API version, crash containment, malformed output, update/revocation e dependency isolation.

# Release profile

Artefato final deve ser testado com:

clean settings, missing cache, representative GPU/device, project fixture, import/export, save/reopen, failure/recovery e uninstall/update quando aplicável.

# Exit Gate

Aplicação DCC/3D não recebe release-ready enquanto houver:

- data-loss path conhecido;
- invalid geometry corruption;
- uncontrolled RAM/VRAM growth;
- device-loss crash sem recovery policy;
- importer crash em malformed fixture crítica;
- frame/interaction budget crítico violado;
- save/load round-trip inconsistente;
- dead interactive surface;
- performance claim sem workload/baseline;
- unsupported hardware implicitly treated as supported.
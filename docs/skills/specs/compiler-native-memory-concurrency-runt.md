# 79.J — Compiler, Native, Memory, Concurrency, Runtime e Low-Level Skills

> Authority: canonical specification.
> Logical ID: 79 J
> Source: Notion Living Book (3d89bb7d023f8185876bc2a9c64e6af7)
> Status: Skill Package de referência para 79 J — Compiler, Native, Memory, Concurrency, Runt.


<aside>
⚙️

Esta família cobre as capacidades em que bugs podem ser silenciosos, dependentes de plataforma ou destrutivos. O hardening deve privilegiar invariantes, conformance, sanitizers, property/fuzz tests e profiling reproduzível.

</aside>

# Componentes centrais

| Skill | Pri. | Aprimoramento | Evidence/aceite |
| --- | --- | --- | --- |
| **compiler-development** | P0 | Transformar em skill central: lexer/parser/AST/semantic/IR/codegen/runtime contracts, diagnostics, conformance, fuzz/differential, incremental, ABI e performance. | Compiler changes referenciam contracts compartilhados. |
| **ast-transformation** | P1 | Preservar source ranges/trivia/comments quando aplicável; idempotence, semantic equivalence, round-trip e golden fixtures. | Transform twice não cria drift inesperado. |
| **memory-management** | P0 | Modelos manual/RAII/ARC/GC/arena/ownership; lifetime, fragmentation, allocation budgets, UAF/leak detection e FFI boundaries. | Strategy selecionada conforme linguagem/runtime. |
| **concurrency-quality** | P1 | Race/deadlock/livelock/starvation, atomics/memory order, cancellation, structured concurrency, backpressure, stress/sanitizers. | Concurrency risks possuem provider/test apropriado. |
| **performance-native** | P1 | CPU/GPU/memory/I/O/allocation budgets, profiler methodology, cache behavior, platform variance e regression gates. | Optimization exige baseline e profile evidence. |
| **serialization** | P1 | Schema versions, unknown fields, backward/forward compatibility, canonical form, limits, polymorphism risks e fuzzing. | Old/new round-trip matrix. |
| **rpc-protocols** | P1 | Schema evolution, deadlines/cancellation, retries/idempotency, streaming/backpressure, auth, version negotiation e observability. | N/N-1 compatibility fixtures. |

# Compiler correctness dimensions

“Passa os testes” deve ser decomposto em:

- lexical/syntactic correctness;
- semantic/type correctness;
- diagnostic correctness/stability;
- IR invariants;
- runtime behavior;
- ABI/FFI compatibility;
- malformed/adversarial input robustness;
- performance/compile-time budgets;
- determinism/reproducibility.

# Differential testing

Quando existir implementação de referência/oracle ou dois paths equivalentes, comparar outputs/semantics automaticamente. Divergência deve produzir minimal counterexample quando possível.

# Fuzz targets prioritários

Lexer/parser, serializers/deserializers, bytecode decoders, protocol parsers, asset readers, file formats, RPC inputs, AST transformations e boundary adapters.

# Memory profile schema

Cada project/profile deve poder declarar: allocation model, allocator(s), long-lived regions, realtime/no-allocation paths, ownership boundaries, FFI escape hatches, leak/UAF tools e memory budgets.

# Low-level escape hatch contract

Para C/C++/Rust/Odin/ASM/FFI:

- motivo;
- scope mínimo;
- safety invariant;
- owner;
- associated tests/sanitizers;
- platform assumptions;
- expiry/review trigger quando temporário.

# Exit Gate

Compiler/native skills deixam de depender de advice genérico; safety/performance rules conhecem linguagem/target; risky parsers e boundaries possuem fuzz/property strategy; escape hatches são auditáveis.
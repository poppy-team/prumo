# 79.I — Language Skills Contract e Hardening de Todas as lang-*

> Authority: canonical specification.
> Logical ID: 79 I
> Source: Notion Living Book (3d89bb7d023f81faa922ff4c08b7a7d0)
> Status: Skill Package de referência para 79 I — Language Skills Contract e Hardening de Tod.


<aside>
🔤

A auditoria encontrou **profundidade muito desigual** entre language skills. O objetivo desta página é estabelecer um contrato mínimo comum sem reduzir as diferenças reais entre linguagens, runtimes, versões e toolchains.

</aside>

# Language Skill Contract mínimo

Toda `lang-*` estável deve cobrir, quando aplicável:

1. versões/editions/LTS suportadas e política de freshness;
2. build/package/toolchain principal e alternativas relevantes;
3. idioms e estilo canônico;
4. error model;
5. memory/resource model;
6. concurrency/async model;
7. safety/UB/escape hatches;
8. FFI/interoperability;
9. testing/static analysis/lint/sanitizers;
10. performance/profiling;
11. platform/target/runtime matrix;
12. generated-code policy;
13. good/bad examples;
14. compatibility notes e version-specific deltas.

# Regra de profiles

Uma única linguagem pode ter profiles diferentes. Exemplos: Rust app/library/no_std/embedded/FFI; C++ application-safe/systems/engine; Kotlin JVM/Android/Native/Multiplatform; JavaScript browser/Node; C# server/desktop/NativeAOT.

| Skill | Pri. | Hardening necessário | Evidence/aceite |
| --- | --- | --- | --- |
| **lang-asm** | P1 | Separar ISA, dialect e ABI; x86/x64/ARM quando suportados; calling conventions, stack alignment, registers, SIMD e privileged boundaries. | Context exige ISA+ABI+dialect. |
| **lang-bash** | P1 | Quoting, arrays, pipelines, subshells, `set -e` caveats, traps, POSIX-vs-Bash, injection, ShellCheck/Bats. | Fixtures com whitespace/metacharacters/failures. |
| **lang-c** | P1 | C11/17/23 profiles, ownership/bounds, UB/integer overflow, sanitizers/static analysis, ABI/FFI e toolchains. | Safety profile + sanitizer/static evidence por risco. |
| **lang-c3** | P1 | Compiler/version pin, build/package, C interop, optional/error semantics, memory model e platform maturity. | Knowledge pack declara versão validada. |
| **lang-cpp** | P0 | Manter RAII/sanitizers, mas substituir proibições absolutas por profiles application-safe/systems/engine/FFI; exception/no-exception policies e controlled unsafe/escape hatches. | Regras rigorosas sem bloquear low-level válido. |
| **lang-csharp** | P1 | .NET LTS matrix, nullable, async/cancellation, IDisposable, analyzers, trimming/NativeAOT, interop e profiles desktop/server. | Target/runtime explícitos. |
| **lang-css** | P1 | Cascade layers, specificity, custom/logical props, container queries, responsive, reduced motion, compatibility e visual tests. | Browser support associado a features não-baseline. |
| **lang-d** | P1 | DMD/LDC/GDC, GC/@nogc/BetterC, RAII/scoped resources, templates, C/C++ interop, Dub e cross compilation. | Compiler+memory profile explícito. |
| **lang-dart** | P1 | Dart VM/AOT/web, null safety, isolates, async e package tooling; Flutter como profile separado, não default. | Não mistura Flutter em Dart CLI/library. |
| **lang-dockerfile** | P1 | Limitar a syntax/build semantics, stages, cache mounts, build secrets e pinning; delegar runtime hardening a `containers`. | Ownership claro com containers. |
| **lang-elixir** | P1 | OTP/supervision, GenServer/process lifecycle, failure philosophy, message ordering, backpressure, telemetry e property testing. | Concurrent service recebe OTP design evidence. |
| **lang-glsl** | P1 | OpenGL/Vulkan variants, versions/extensions, layouts, precision, SPIR-V/reflection, shader compiler/GPU matrix. | Shader target/version obrigatório. |
| **lang-go** | P1 | Supported versions, modules/workspaces, context/cancellation, goroutine ownership, race/fuzz/vet, errors, profiling e cgo. | Race/fuzz/tools ativados por risk profile. |
| **lang-graphql** | P1 | Schema evolution, nullability, resolver auth, N+1/DataLoader, depth/complexity, persisted queries e subscriptions. | Compatibility + abuse contract tests. |
| **lang-hlsl** | P1 | Shader models, DXC, DXIL/SPIR-V where relevant, semantics, registers/bindings, precision e GPU matrix. | Target/compiler part of evidence. |
| **lang-html** | P1 | Semantic HTML, forms, native a11y, metadata/SEO, security boundaries, progressive enhancement e template constraints. | Semantic/a11y lint evidence. |
| **lang-java** | P1 | LTS versions, Maven/Gradle, JVM/GC, virtual threads/concurrency, nullability, serialization security e framework-neutral core. | Java/runtime version explícita. |
| **lang-javascript** | P1 | ECMAScript target, browser/Node profiles, ESM/CJS, async/event loop, runtime validation, prototype/security e package hygiene. | Runtime incompatibilities detectadas. |
| **lang-json** | P1 | Encoding, number precision, duplicate keys, canonicalization, schema validation e JSON-vs-JSON5 boundary. | Boundary/round-trip fixtures. |
| **lang-kotlin** | P1 | JVM/Android/Native/Multiplatform, coroutines/cancellation, nullability, Gradle, Java interop, Compose only when relevant. | Target-specific recommendations. |
| **lang-lua** | P1 | Lua 5.1–5.5/LuaJIT, embedding/C API, GC, metatables, modules, sandboxing e realtime/engine profile. | Host+Lua version registrados. |
| **lang-odin** | P0 | Expandir de allocator/context/SoA para packages, procedures, error idioms, defer, allocators/arenas, C interop, threads, build/test, SIMD, platform/toolchain. | Suporta projeto Odin completo. |
| **lang-php** | P1 | PHP 8.x, strict types, Composer/autoloading, errors, framework boundaries, security e static analysis. | Runtime/framework profile. |
| **lang-python** | P1 | Supported versions, env/packaging, typing vs runtime guarantees, async/multiprocessing, serialization safety, perf e test tooling. | Type hints nunca tratados como runtime validation. |
| **lang-ruby** | P1 | Versions, Bundler, blocks/metaprogramming, Ractors/threads, GC, RBS/Sorbet options, security e perf; Rails separado. | Core Ruby e Rails profiles distintos. |
| **lang-rust** | P0 | Editions/MSRV/features, ownership/API design, async/Send/Sync, unsafe contracts, Miri/sanitizers, FFI, cargo audit/deny e no_std profiles. | Policies variam por app/library/kernel/embedded/FFI. |
| **lang-sql** | P1 | Dialect profiles PostgreSQL/MySQL/SQLite/etc., NULL, transactions/isolation, locks, indexes/EXPLAIN, migrations e injection. | Dialect/schema obrigatórios. |
| **lang-swift** | P1 | Swift version, actors/Sendable, structured concurrency, ARC cycles, errors, SwiftPM, Apple/Linux boundaries. | Concurrency safety evidence. |
| **lang-terraform** | P1 | Terraform/OpenTofu profile, provider pinning, state/locking, plan/apply split, imports/moved, drift, secrets e module contracts. | Mutation real requer plan/evidence. |
| **lang-typescript** | P1 | Strictness profiles, module resolution, project refs, ESM/CJS, decorators/version differences, runtime validation e generated types. | Boundary data validada em runtime quando necessário. |
| **lang-wgsl** | P1 | Spec/implementation freshness, address spaces, binding layouts, limits/features, browser/native WebGPU matrix. | Adapter/features/limits in evidence. |
| **lang-yaml** | P1 | YAML 1.1/1.2, implicit scalars, anchors/aliases, tags, schema constraints e parser security. | Coercion/alias hazards em fixtures. |
| **lang-zig** | P1 | Pinned compiler, build.zig, allocators, error unions, comptime, C interop, cross compilation e version migration. | Guidance ligada a versão validada. |

# Migration method por language skill

1. Capturar conteúdo atual e supported assumptions.
2. Classificar knowledge permanente vs version-sensitive.
3. Aplicar Language Skill Contract.
4. Separar profiles por runtime/target quando necessário.
5. Extrair regras deterministicamente verificáveis para checks/scripts.
6. Criar exemplos good/bad e fixtures de version boundaries.
7. Rodar eval sem skill vs skill antiga vs skill nova.
8. Promover somente se qualidade aumentar sem over-activation/context explosion.

# Freshness

Languages/toolchains jovens ou de rápida evolução (`zig`, `c3`, `odin`, WGSL/WebGPU etc.) exigem review cadence menor que linguagens/ecossistemas estáveis. Freshness deve ser metadata, não nota perdida em prosa.

# Exit Gate

Todas as `lang-*` ativas cumprem o contrato mínimo, expõem target/version/context necessário e nenhuma orientação absoluta conflita com profiles válidos de systems/embedded/engine/FFI.

# Atualização de auditoria — 2026-09-16

## Correções imediatas de freshness

- **Lua:** Lua 5.5 é a linha atual; a skill não deve mais se descrever como “Lua 5.4/LuaJIT”. Separar profiles `lua-5.5`, `lua-5.4-maintenance/legacy`, `lua-5.1-compatible` e `luajit`, porque API/ABI/bytecode/runtime assumptions divergem.
- **Odin:** `odin test` já possui memory tracking por teste habilitado por padrão. A skill atual não deve exigir a troca manual por tracking allocator como regra universal; documentar quando usar o mecanismo integrado e quando instrumentação customizada é necessária.
- **C3:** alinhar contratos com `@require`/`@ensure`, safe-vs-fast, `@test`, `@benchmark`, `c3c test/build/benchmark` e `project.json`; não inventar significado genérico para `@param`.
- **D:** modelar `@safe/@trusted/@system`, `scope`/escape analysis e `-preview=dip1000` como profile/version-sensitive; incluir DMD/LDC/GDC, Dub, GC/@nogc/BetterC e C/C++ interop.
- **C:** remover regras que dão falsa sensação de segurança. `ptr = NULL` não elimina aliases UAF; `strlcpy` não é ISO C/portável em todos os targets; `__attribute__((cleanup))` é extension-specific. Build/checks precisam variar por GCC/Clang/MSVC e plataforma. Sanitizers devem ser parte de uma matriz de evidence, não o único gate.

## Nova skill P0/P1: `lang-nim`

A varredura de adoção já reconhece `.nim`, mas não há skill dedicada no catálogo atual. Criar `lang-nim` com profiles por versão/runtime/backend e pelo menos: Nim 2.2.x/current supported line; ARC/ORC/destructors/move semantics; refs/ptrs e FFI; macros/templates/generics; NimScript; targets C/C++/JS quando aplicáveis; Nimble/Atlas; nimsuggest; nimpretty; `nim doc`; unittest/Testament; profiling/debugging; threads/async; C interoperability/c2nim; packaging/cross-platform.

## Regra de ecosystem pack por linguagem

`lang-X` não deve absorver todo o ecossistema. Quando o volume justificar, knowledge/workflows separados devem cobrir `toolchain`, `package/build`, `lint/format`, `test`, `debug/profile`, `ffi`, `cross-compile`, `security` e `release`. O resolver carrega somente o subset necessário.

## Referências canônicas adicionadas

- [Lua manuals](https://www.lua.org/manual/)
- [Odin testing](https://odin-lang.org/docs/testing/)
- [C3 specification](https://c3-lang.org/implementation-details/specification/)
- [D function specification](https://dlang.org/spec/function.html)
- [Nim documentation](https://nim-lang.org/docs/index.html)
- [Nim tools](https://nim-lang.org/docs/tools.html)

# Revisão 2026-09-16 — C++, ASM, Wasm/WAT e WGSL

- `lang-cpp` permanece skill P0, mas regras absolutas atuais devem migrar para profiles `application-safe`, `library-api`, `systems`, `engine-realtime`, `embedded-freestanding`, `ffi-abi` e `hpc-simd`. Raw pointers, casts de baixo nível, exceptions e custom allocators são tratados por semântica/risco/profile, não por ban universal.
- `lang-asm` torna obrigatórios ISA + dialect + ABI + OS/target antes de emitir guidance. x86-64, AArch64 e RISC-V possuem knowledge packs independentes e versionados.
- WAT é representação textual formal de WebAssembly e pode ter uma skill de authoring/debugging (`lang-wat` ou `format-wat`) compondo uma skill de plataforma WebAssembly; não deve carregar WASI/Component Model automaticamente.
- `lang-wgsl` deve cuidar da linguagem WGSL; a API WebGPU passa a `graphics-webgpu`. Remover referências a `std140/std430` como se fossem regras WGSL: WebGPU/WGSL possui layout/alignment próprio definido pela especificação.
- Language skill nunca substitui runtime/platform/framework pack. O resolver pode compor `lang-wgsl + graphics-webgpu`, `lang-wat + platform-webassembly`, `lang-cpp + build-cmake + package-conan` etc.
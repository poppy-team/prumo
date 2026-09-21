# 79.V — Language Ecosystem Packs: C, C3, Odin, D, Lua e Nim

> Authority: canonical specification.
> Logical ID: 79 V
> Source: Notion Living Book (3dd9bb7d023f81cbbf63d68a19e9bd87)
> Status: Skill Package de referência para 79 V — Language Ecosystem Packs C, C3, Odin, D, Lu.


<aside>
🧬

**Objetivo:** transformar `lang-*` de guias curtos de sintaxe/estilo em portas de entrada para language + toolchain + runtime + testing + performance + FFI + release knowledge, mantendo progressive disclosure. O `SKILL.md` guarda invariants e routing; detalhes version-sensitive ficam em knowledge packs versionados.

</aside>

# Language Ecosystem Pack Contract

Cada linguagem deve possuir, quando aplicável: `version_profile`, `compiler/toolchain`, `build/project`, `package/dependency`, `formatter/lint/static`, `test/fuzz/property`, `debug/sanitizer`, `benchmark/profile`, `memory/resource`, `concurrency`, `ffi/abi`, `cross_compile/targets`, `docs`, `release`, `security`, `migration`, `good_bad_examples`, `known_footguns`, `source_provenance` e `eval_matrix`.

A skill não deve impor uma única ferramenta universal quando a linguagem possui múltiplos toolchains. O resolver seleciona profile por manifesto, target, OS, compiler e requisitos do projeto.

# `lang-c` — hardening P0/P1

## Profiles

- `c23-modern`, `c17-compat`, `embedded/freestanding`, `systems/native`, `ffi/library`, `msvc`, `gcc`, `clang`.
- Toolchain e standard precisam ser explícitos; extensions GNU/Clang/MSVC nunca devem ser ensinadas como ISO C.

## Knowledge packs

- **build-cmake:** target-based CMake, presets, toolchain files, CTest/CPack, install/export/package boundaries.
- **compiler-gcc / compiler-clang / compiler-msvc:** diagnostics, warnings, extensions, optimization, debug info e target-specific behavior.
- **c-sanitizers-analysis:** ASan/UBSan/TSan/MSan/LSan onde suportados, Clang Static Analyzer e outras ferramentas conforme plataforma; nenhuma delas prova ausência de UB por si só.
- **c-abi-ffi:** calling convention, layout/alignment, symbol visibility, ownership transfer, allocator boundary e versioned ABI.

## Regras a corrigir na skill atual

`ptr = NULL` após `free` pode reduzir reuse acidental daquele alias, mas **não previne UAF via aliases**. `strlcpy` não é ISO C e não deve ser recomendação portátil universal. `__attribute__((cleanup))` é extension-specific. Valgrind/GCC não podem ser gates universais em Windows/macOS/embedded. Checks devem ser capability/platform-aware.

## Evals

Boundary/length arithmetic, integer overflow, alias-UAF, ownership across FFI, partial initialization, error cleanup, cross-compiler warnings, sanitizer-positive fixture, freestanding build e CMake cross-toolchain fixture.

# `lang-c3` — hardening P1 com freshness curta

## Knowledge

- compiler version pin + `project.json`/`project.json5`;
- safe vs fast compilation profiles;
- contracts reais (`@require`, `@ensure`, `@param`, `@pure`, `@return?`) e nível de suporte do compiler;
- built-in `@test` / `@benchmark`, `c3c test`, `c3c benchmark` e test/benchmark target types;
- modules/generics/macros/comptime, errors/faults/optionals, memory/allocators, C interop, linking e targets.

## Regra de segurança

Contracts não são prova formal nem obrigatoriamente analisados de forma idêntica por todo compilador conforme a spec; safe mode adiciona checks relevantes, enquanto fast mode pode assumir invariants. Skills devem registrar compile profile na evidence.

## Evals

Safe-vs-fast behavior, contract violation, test/benchmark discovery, C ABI fixture, project config matrix e compiler-version migration.

# `lang-odin` — hardening P0

## Knowledge

- packages/import collections, procedures, multiple returns e error idioms;
- `defer`, `context`, general/temp allocators, arenas/pools/scratch/tracking;
- SoA/SIMD/data-oriented patterns quando apropriados, sem transformar preferência em regra universal;
- C interop/foreign blocks/layout/ABI;
- build/run/check/test/doc workflows e platform/LLVM matrix;
- threads/synchronization e ownership de resources.

## Testing

`odin test` deve ser baseline: runner multi-threaded, seed reproduzível e **memory tracking habilitado por padrão**, incluindo leak/bad-free diagnostics. Tracking allocator manual fica para programas/fixtures específicos, não como requisito universal de todo teste.

## Evals

Allocator lifetime, temp allocator reset, leak/bad-free fixture, FFI struct layout, multi-thread test reproducibility, target builds e debug/release behavior.

# `lang-d` — hardening P1

## Profiles

`dmd`, `ldc`, `gdc`; `gc-default`, `@nogc`, `betterC`, `native-ffi`.

## Knowledge

- `@safe`, `@trusted`, `@system` boundaries;
- `scope`/escape semantics e DIP1000/version flags;
- GC, RAII/scoped resources, destructors, `@nogc`, BetterC;
- templates/CTFE/mixins sem abusar de metaprogramação;
- Dub package/build workflows;
- C/C++ interop, shared/static libraries, cross-compilation e compiler differences.

## Evidence

Sempre registrar compiler + version + active preview/transition flags. Não assumir que DMD behavior representa LDC/GDC em codegen/performance/platform support.

# `lang-lua` — hardening P1

## Profiles

`lua-5.5`, `lua-5.4-maintenance/legacy`, `lua-5.1-compatible`, `luajit`, `embedded-hosted`.

## Knowledge

- current official baseline é Lua 5.5; registrar host/runtime exato;
- C API/stack discipline, registry/references, userdata/lifetime, error boundaries;
- incremental/generational GC conforme runtime/version;
- metatables/metamethods/coroutines/modules;
- sandboxing deve ser host/threat-model-specific: não existe um toggle universal de “sandbox seguro”;
- LuaRocks quando adotado pelo projeto; LuaJIT tratado como runtime distinto, não como sinônimo de Lua atual.

## Evals

C API stack balance, host-owned lifetime, untrusted script capability restrictions, module resolution, GC pressure e 5.1/LuaJIT-vs-5.5 compatibility fixture.

# `lang-nim` — nova skill P1

A documentação oficial atual está em **Nim 2.2.12**. A ausência de skill dedicada é uma lacuna do catálogo.

## Knowledge

- compiler/backend profiles C/C++/ObjC/JS conforme target;
- ARC/ORC, destructors, sink/move semantics, refs/pointers e realtime memory considerations;
- generics/templates/macros e NimVM/comptime boundaries;
- exceptions/effects/threading/async conforme versão/profile;
- C/C++ interoperability e `c2nim` quando aplicável;
- **Nimble** para packages + **Atlas** como dependency workspace/cloner compatível com Nimble format;
- `nimsuggest`, `nimpretty`, `nim doc`, `testament`, compiler user guide e NimScript;
- cross-compilation, generated C/C++ inspection e release packaging.

## Freshness

Nim 2.2 continua recebendo patch releases relevantes (2.2.12 em 2026-09-08), inclusive fixes ARC/ORC. `last_verified` curto e regression fixtures de memory model são obrigatórios.

## Evals

ARC/ORC ownership/move, FFI C header wrapper, multi-backend build, Nimble/Atlas dependency resolution, macro hygiene, formatter/tooling, Testament multi-target e generated-code inspection.

# Common language eval matrix

| Dimensão | Obrigatório |
| --- | --- |
| Activation | Detectar linguagem/toolchain sem ativar por arquivo vendorizado/transitivo irrelevante. |
| Freshness | Versão suportada, release/current docs, stale-source fixture. |
| Correctness | Compile/build/test fixture real da versão declarada. |
| Safety | Positive + negative fixture para memory/error/FFI boundaries quando aplicável. |
| Performance | Benchmark/profile guidance sem otimização prematura e com baseline reproduzível. |
| Portability | OS/compiler/target matrix e unsupported combinations explícitas. |
| Interop | ABI/ownership/lifetime/error translation. |
| Context cost | Carregar apenas knowledge requerido pelo profile atual. |

# Referências upstream

- [Clang documentation](https://clang.llvm.org/docs/)
- [CMake documentation](https://cmake.org/cmake/help/latest/)
- [C3 specification](https://c3-lang.org/implementation-details/specification/)
- [Odin documentation](https://odin-lang.org/docs/)
- [D specification](https://dlang.org/spec/spec.html)
- [Lua manuals](https://www.lua.org/manual/)
- [Nim documentation](https://nim-lang.org/docs/index.html)

# Exit Gate

Cada language skill ativa deve saber responder **qual versão/toolchain/profile**, carregar somente o knowledge necessário, possuir uma test/evidence matrix real e nunca apresentar extension/runtime/tool específico como regra universal da linguagem.

# Expansão decidida — C++ / Wasm / Low-level / CS

A auditoria de 2026-09-16 amplia este contrato com páginas especializadas para C++ moderno e seguro, WebAssembly/WAT/WASI, WebGPU/Three.js, assembly/ISA/ABI/IR, language engineering, Computer Science/Software Engineering, Database Engineering/Administration e Linux avançado. Essas páginas seguem o mesmo princípio: a `lang-*` contém invariants e routing; ecossistema, runtime, platform e tools ficam em packs progressivamente carregados. O detalhamento desta expansão está nas novas páginas 79.W–[79.AD](http://79.AD) do Programa 79.
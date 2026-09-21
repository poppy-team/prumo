# 79.W — Modern C++, WASM/WAT, WebGPU, Three.js e Low-Level Systems Packs

> Authority: canonical specification.
> Logical ID: 79 W
> Source: Notion Living Book (3dd9bb7d023f81e0905bc500fc0354cc)
> Status: Skill Package de referência para 79 W — Modern C++, WASM WAT, WebGPU, Three js e Lo.


<aside>
⚙️

**Decisão:** criar uma família de packs para sistemas, runtime portátil e GPU. O princípio é profile-driven: a mesma linguagem/tecnologia muda de regras conforme app, engine, embedded, FFI, browser, compute ou tooling.

</aside>

# Modern C++ — reconstrução de `lang-cpp`

A skill atual é excessivamente absoluta para C++ real de systems/engines/FFI. O novo desenho deve seguir C++ Core Guidelines, toolchain/version profile e escape hatches verificáveis.

## Profiles

- `application-safe`: value semantics, RAII, ranges/span/string_view, ownership explícito, exceptions policy, sanitizers/static analysis.
- `engine-realtime`: deterministic allocation, custom allocators, cache/data layout, SIMD, no-exception option, frame budgets.
- `systems-lowlevel`: raw pointers e pointer arithmetic permitidos em boundaries documentadas, ABI/OS APIs, atomics, lock-free e intrinsics.
- `ffi-library`: ABI, calling convention, symbol visibility, allocator/exception boundary, stable C facade quando apropriado.
- `embedded/freestanding`: limited runtime, no heap/RTTI/exceptions conforme target, linker/startup constraints.

## Knowledge packs obrigatórios

C++20/23/26 feature matrix; Core Guidelines; GSL; ownership/lifetimes; templates/concepts/ranges; coroutines; atomics/memory model; modules; allocators/PMR; SIMD/intrinsics; exceptions/errors/`std::expected`; build CMake/Meson; package Conan/vcpkg; Clang/GCC/MSVC; clang-tidy/cppcheck; ASan/UBSan/TSan/MSan quando suportados; fuzz/property; perf/profiling; ABI/FFI; C++ modules/toolchain compatibility.

## Regra central

Raw pointer não significa automaticamente ownership ou bug. `reinterpret_cast`, placement new, custom allocators, intrinsics e pointer arithmetic podem ser necessários em low-level code; o Prumo deve exigir **invariant + boundary + test/evidence**, não proibição cega.

## Evals

Lifetime bug, aliasing/strict-aliasing, dangling view, iterator invalidation, exception safety, data race, false sharing, allocator mismatch, ABI layout, cross-toolchain build, real-time allocation regression e compiler sanitizer matrix.

# WebAssembly / WAT

O baseline atual é **WebAssembly 3.0**. `lang-wat` deve representar o text format; `runtime-wasm` cobre execução/embedding; `wasi` e `component-model` entram como packs próprios/componíveis.

## `lang-wat`

S-expressions, modules/types/imports/exports/functions/tables/memories/globals, control flow, numeric/vector/ref instructions, validation, text↔binary roundtrip, source maps/debug names e canonical formatting.

## `runtime-wasm`

Validation/instantiation, host imports, linear memory, tables/references, memory64, exceptions/tail calls/GC/threads/SIMD conforme engine support, sandbox boundaries, fuel/epoch/resource limits e deterministic execution quando aplicável.

## `wasi` / Component Model

WASI 0.2.x é baseado no Component Model; Component Model segue evoluindo em Developer Preview. Cobrir WIT, canonical ABI, resources/handles, component composition, capabilities, filesystem/network/http/clock/random interfaces e version negotiation. Não confundir Core Wasm, WASI e Component Model como uma só versão.

## Tooling

Wasmtime/Wasmer/WAMR/Node/browser profiles; `wat2wasm`/`wasm2wat`, `wasm-tools`, `wasm-opt`, WIT tooling, component composition, fuzzing/differential engine tests.

# WebGPU + WGSL

A atual `lang-wgsl` é insuficiente para cobrir WebGPU host-side. Criar `graphics-webgpu` que compõe `lang-wgsl`.

## `graphics-webgpu`

Adapter/device acquisition, feature/limit negotiation, queue/buffer/texture/sampler lifecycle, bind groups/layouts, pipeline layouts, render/compute pipelines, command encoders/passes, synchronization/usage transitions, mapping/upload/readback, device loss, validation/error scopes, timestamp/query support, browser/native capability matrix, memory/bandwidth/perf budgeting e CTS-oriented compatibility.

## WGSL hardening

Address spaces/layout/alignment devem seguir **WGSL/WebGPU**, não nomenclatura Vulkan `std140/std430` como regra. Cobrir workgroups/barriers/atomics, override constants, resource bindings, shader stages, validation, numeric behavior e spec freshness.

# Three.js moderno

Criar `framework-threejs` com profiles WebGL/WebGPU e forte freshness por release.

## Knowledge atual

Scene graph, transforms, cameras, BufferGeometry, loaders/assets, texture/color management, PBR/materials, animation, raycasting, instancing, culling/LOD, post-processing, disposal/resource lifecycle, render-loop/performance, workers/off-main-thread quando possível.

## WebGPU/TSL

O stack moderno deve conhecer `WebGPURenderer`, NodeMaterial e **TSL**. A documentação atual recomenda Node Material/TSL para customização em WebGPU; o novo post-processing possui MRT nativo, node composition e combinação de effects. Registrar fallback/capability behavior e diferenças reais entre WebGPU e WebGL.

# Low-level / Assembly family

`lang-asm` vira orchestrator por ISA/ABI/dialect, nunca uma skill genérica única.

## Packs

- `asm-x86-64`: Intel/AT&T syntax, System V + Windows x64 ABI, registers/flags, stack/alignment, scalar/SSE/AVX/AVX-512/AVX10 conforme CPU, atomics/memory ordering, syscalls.
- `asm-aarch64`: AAPCS64, registers, SIMD/NEON/SVE quando disponível, atomics, memory ordering, system instructions por privilege profile.
- `asm-riscv`: RV32/RV64, extensions, psABI, calling convention, compressed/vector profiles e extension negotiation.
- `binary-elf`, `binary-pe`, `binary-mach-o`: sections, relocations, symbols, dynamic linking, TLS e debug info.
- `linker-loader`: ld/lld/link, GOT/PLT, PIC/PIE, startup/runtime, loader boundaries.
- `debug-native`: GDB/LLDB, disassembly, registers, watchpoints, core dumps, DWARF/PDB.
- `perf-microarchitecture`: perf counters, cache/TLB/branch behavior, `llvm-mca`, optimization remarks e hardware measurement.

## Ensino obrigatório em low-level

Assembly deve ser ensinado como **modelo de máquina + ABI + compiler output**, não lista de opcodes. Sempre ligar fonte de alto nível → IR → assembly → binary → execução/microarchitecture.

# Fontes upstream

- C++ Core Guidelines: [https://github.com/isocpp/CppCoreGuidelines](https://github.com/isocpp/CppCoreGuidelines)
- WebAssembly specs: [https://webassembly.org/specs/](https://webassembly.org/specs/)
- WASI: [https://github.com/WebAssembly/WASI](https://github.com/WebAssembly/WASI)
- Component Model: [https://github.com/WebAssembly/component-model](https://github.com/WebAssembly/component-model)
- WebGPU/WGSL: [https://gpuweb.github.io/gpuweb/](https://gpuweb.github.io/gpuweb/)
- Three.js: [https://threejs.org/docs/](https://threejs.org/docs/)
- Intel SDM: [https://www.intel.com/content/www/us/en/developer/articles/technical/intel-sdm.html](https://www.intel.com/content/www/us/en/developer/articles/technical/intel-sdm.html)
- RISC-V specs: [https://riscv.org/specifications/ratified/](https://riscv.org/specifications/ratified/)
- LLVM MCA: [https://llvm.org/docs/CommandGuide/llvm-mca.html](https://llvm.org/docs/CommandGuide/llvm-mca.html)

# Exit Gate

Nenhum pack confunde linguagem, ABI, ISA, runtime ou API; nenhuma regra de safety bloqueia low-level válido sem oferecer boundary contract; e toda recomendação version-sensitive possui provenance/freshness.
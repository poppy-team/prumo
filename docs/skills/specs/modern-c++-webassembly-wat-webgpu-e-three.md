# 79.W — Modern C++, WebAssembly/WAT, WebGPU e Three.js Skill Packs

> Authority: canonical specification.
> Logical ID: 79 W
> Source: Notion Living Book (3de9bb7d023f812eb484d49cf076f909)
> Status: Skill Package de referência para 79 W — Modern C++, WebAssembly WAT, WebGPU e Three.


<aside>
⚙️

**Objetivo:** substituir regras absolutas por profiles modernos, seguros e verificáveis para C++; criar packs completos para WebAssembly/WAT, WebGPU/WGSL e Three.js/WebGPURenderer, com versionamento e evidence próprios.

</aside>

# Modern C++: problema atual

A skill `lang-cpp` atual possui boas intenções — RAII, sanitizers, static analysis e escape-hatch quarantine — mas trata recomendações de application-safe code como leis universais. Isso quebra casos legítimos de engines, allocators, FFI, SIMD, kernel-adjacent code, embedded e performance-critical systems.

A evolução deve seguir a filosofia do C++ Core Guidelines: ownership explícito, RAII, type safety, bounds/lifetime awareness e tooling forte, mas com **profiles por domínio** em vez de proibições universais.

# Profiles propostos

- `cpp-application-safe`: value semantics, RAII, spans/views, ownership explícito, exceptions/result policy definida.
- `cpp-library-api`: ABI/API stability, noexcept contract, ownership/lifetime annotations, modules/headers, compatibility.
- `cpp-engine-realtime`: deterministic allocation policy, arenas/pools, cache locality, SIMD, frame budgets, no hidden blocking/destruction spikes.
- `cpp-systems-low-level`: raw pointers/bit manipulation/atomics permitidos com invariants, bounds, provenance e review especializado.
- `cpp-embedded-freestanding`: exceptions/RTTI/STL profile conforme target; static allocation/resource bounds; cross toolchain.
- `cpp-ffi`: layout/calling convention/allocator ownership/error translation/string representation.

# C++ ecosystem knowledge

- standards: C++20/C++23 baseline + C++26 feature awareness; não apresentar C++29 safety work como feature disponível;
- compilers: Clang/GCC/MSVC e diferenças reais de diagnostics/sanitizers/ABI;
- build: CMake target-based design, presets/toolchain files, Ninja, Conan/vcpkg apenas quando adotados;
- analysis: clang-tidy, clang static analyzer, cppcheck quando apropriado, compiler warnings por toolchain;
- runtime verification: ASan/UBSan/TSan/MSan/LSan conforme plataforma;
- fuzz/property: libFuzzer/AFL++/RapidCheck/etc. por risk profile;
- perf: perf, VTune, Tracy, Instruments, compiler optimization reports e microbenchmarks;
- concurrency: atomics, memory model, lock-free caveats, false sharing, thread lifetime, executors/coroutines conforme versão;
- unsafe boundary: escape hatch exige owner, invariant, threat/risk, tests e evidence — não mero comentário.

# Correções normativas

- raw pointer **não significa ownership** e não deve ser banido universalmente; owning raw pointer deve ser excepcional.
- `reinterpret_cast` e pointer arithmetic podem ser necessários em low-level/FFI; devem ser isolados e justificados.
- `std::expected` não substitui toda exceção; error model é profile/API decision.
- `-Wall -Wextra ... -Werror` não é matriz universal entre MSVC/GCC/Clang.
- ASan/TSan/MSan não são simultaneamente suportados/úteis em todo target.
- TDD pode ser workflow recomendado, não regra sem exceção para toda modificação.

# WebAssembly / WAT

A especificação Core atual é **WebAssembly 3.0**. A skill deve separar claramente:

- `wasm-core`: validation, modules, types, functions, tables, memories, globals, references, SIMD, multi-memory/memory64 quando aplicável;
- `wat`: representação textual, round-trip, validation, symbolic names, folded/unfolded expressions;
- `wasm-js-web`: JS/Web APIs, streaming compile/instantiate, browser security/origin boundaries;
- `wasi`: system interface versionada;
- `wasm-component-model`: WIT, components, canonical ABI, composition/linking;
- `wasm-runtime`: Wasmtime/Wasmer/WAMR/etc. somente quando stack detectada;
- `wasm-tooling`: wabt, Binaryen, wasm-tools, objdump/disassembly/debug info;
- `wasm-security`: sandbox assumptions, host capabilities/imports, resource exhaustion, untrusted modules.

**Importante:** WASI 0.2.x já é baseado no Component Model. O resolver deve registrar versões separadas de Core Wasm, WASI e Component Model, pois maturidade e compatibilidade não caminham juntas.

# Evals WASM/WAT

- wat↔wasm round trip;
- invalid module rejection;
- import/export capability restriction;
- memory/table boundary cases;
- browser vs WASI behavior;
- component/WIT version mismatch;
- fuel/epoch/resource-limit fixture em runtimes que suportam;
- differential execution entre runtimes quando necessário.

# WebGPU / WGSL

`lang-wgsl` já existe, mas hoje confunde WGSL com a totalidade do WebGPU. Criar `graphics-webgpu` como skill própria, mantendo `lang-wgsl` como shader language specialist.

## `graphics-webgpu` deve cobrir

- adapter/device/queue/surface/canvas lifecycle;
- features/limits e capability negotiation;
- buffers/textures/samplers/bind groups/pipeline layouts;
- render/compute pipelines e command encoding;
- upload/readback/alignment/lifetime;
- synchronization/implicit ordering model;
- device lost, validation errors e error scopes;
- render bundles/pipeline caching quando justificável;
- timestamp/occlusion query e profiling;
- browser compatibility + CTS evidence;
- security/resource exhaustion e untrusted shader/data boundaries.

## `lang-wgsl` deve corrigir

Não usar termos de GLSL como `std140/std430` como se fossem regras WGSL universais. Deve seguir diretamente layout/alignment definidos pela especificação WGSL/WebGPU e validar com CTS/Naga/Tint/tooling adequado.

# Three.js modern pack

Criar `framework-threejs` como P1 para projetos web 3D.

## Knowledge obrigatório

- scene graph, transforms, cameras, geometries, materials, textures;
- loaders, disposal/resource lifetime e GPU memory leak avoidance;
- animation mixer, instancing, LOD, frustum/occlusion considerations;
- `WebGPURenderer` como renderer moderno;
- fallback atual do `WebGPURenderer` para WebGL2 quando WebGPU não está disponível;
- Three.js Shading Language / TSL e node materials;
- WebGPU post-processing, MRT e effect composition;
- WebXR quando aplicável;
- render-loop budgets e asset streaming.

# Skill graph

```
framework-threejs
├── requires: lang-javascript | lang-typescript
├── composes: graphics-webgpu
├── optional: lang-wgsl
├── optional: graphics-webgl
├── knowledge: tsl
├── knowledge: webgpu-renderer
└── workflows: asset-loading, frame-budget, gpu-resource-lifecycle
```

# Referências upstream

- C++ Core Guidelines: [https://isocpp.github.io/CppCoreGuidelines/](https://isocpp.github.io/CppCoreGuidelines/)
- WebAssembly specs: [https://webassembly.org/specs/](https://webassembly.org/specs/)
- W3C Wasm Core: [https://www.w3.org/TR/wasm-core/](https://www.w3.org/TR/wasm-core/)
- WASI: [https://github.com/WebAssembly/WASI](https://github.com/WebAssembly/WASI)
- Component Model: [https://github.com/WebAssembly/component-model](https://github.com/WebAssembly/component-model)
- WebGPU: [https://gpuweb.github.io/gpuweb/](https://gpuweb.github.io/gpuweb/)
- WGSL: [https://gpuweb.github.io/gpuweb/wgsl/](https://gpuweb.github.io/gpuweb/wgsl/)
- Three.js WebGPURenderer: [https://threejs.org/docs/pages/WebGPURenderer.html](https://threejs.org/docs/pages/WebGPURenderer.html)

# Exit Gate

Nenhuma guidance de C++ confunde safety profile com linguagem universal; Wasm/WASI/Component Model são versionados separadamente; WebGPU e WGSL possuem owners distintos; Three.js conhece WebGPURenderer/TSL e lifecycle real de recursos.
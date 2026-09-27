# 79.X — Assembly, ISA, ABI, Reverse Engineering e Low-Level Systems Skills

> Authority: canonical specification.
> Logical ID: 79 X
> Source: Notion Living Book (3de9bb7d023f8116ae44dde0a1901cc6)
> Status: Skill Package de referência para 79 X — Assembly, ISA, ABI, Reverse Engineering e L.


<aside>
🧱

**Objetivo:** elevar `lang-asm` de sintaxe de assembly para uma família completa de arquitetura, ABI, machine code, calling conventions, performance, debugging e systems engineering, sem confundir ISA com assembler ou sistema operacional.

</aside>

# Separação obrigatória

Assembly precisa ser modelado por quatro eixos independentes:

1. **ISA/architecture** — x86-64, AArch64, RISC-V etc.;
2. **assembler syntax/tool** — GAS AT&T/Intel, NASM/YASM, MASM, LLVM integrated assembler;
3. **ABI/platform** — System V AMD64, Windows x64, AAPCS64, ELF/Mach-O/PE;
4. **execution context** — userspace, kernel, bootloader, embedded, JIT/codegen.

Nenhuma skill deve inferir calling convention apenas pela ISA.

# Packs P1

- `isa-x86-64`: registers, flags, addressing, SIMD/AVX families, memory ordering, CPUID/features, privilege levels, exceptions.
- `isa-aarch64`: A64 ISA, register model, condition flags, SIMD/FP, exception levels, memory ordering, atomics.
- `isa-riscv`: RV32/RV64, base ISA + extensions, privilege spec, calling convention awareness, vector extension quando ativada.
- `abi-sysv-amd64`, `abi-windows-x64`, `abi-aapcs64`;
- `binary-elf`, `binary-pe-coff`, `binary-mach-o`;
- `tooling-assembler-linker`: GAS/NASM/LLVM/LLD/binutils;
- `debug-native`: GDB/LLDB, core dumps, DWARF/PDB awareness;
- `disassembly-re`: objdump/llvm-objdump, Ghidra/radare2 knowledge quando permitido, symbols/relocations/control flow;
- `microarchitecture-performance`: caches, TLB, branch prediction, OoO, pipelines, SIMD/vectorization, perf counters.

# Source hierarchy

Para ISA, a fonte primária deve ser o manual do fabricante/standard body. Em setembro de 2026, Intel mantém os SDMs atualizados (rev. 092 listada no portal); RISC-V possui specs ratificadas de janeiro de 2026. Community cheat sheets são auxiliares, nunca normativas.

# Low-level correctness contract

Toda mudança de baixo nível deve declarar:

- ISA + extension set;
- assembler syntax;
- ABI/calling convention;
- object format;
- endianness e data model;
- stack alignment/red-zone/shadow-space rules;
- caller/callee-saved registers;
- unwind/debug expectations;
- privileged vs unprivileged context;
- memory ordering/atomic assumptions;
- feature detection/fallback.

# Low-level skills adicionais

- `memory-models`: C/C++/Rust/hardware memory ordering, atomics, fences, happens-before;
- `cache-aware-design`: locality, SoA/AoS tradeoffs, false sharing;
- `simd-vectorization`: intrinsic vs autovectorization vs handwritten assembly;
- `linkers-loaders`: relocations, symbols, PLT/GOT, dynamic loading;
- `object-code-generation`: instruction selection, register allocation concepts, relocation emission;
- `boot-bare-metal`: startup, linker scripts, vectors, MMIO, freestanding runtime;
- `firmware-uefi` quando stack exigir;
- `reverse-engineering` com escopo legítimo de debugging/interoperability/security review.

# Verification

- assemble + disassemble roundtrip;
- ABI conformance fixture chamado a partir de C/C++/Rust;
- stack/register preservation tests;
- sanitizer/Valgrind não substituem hardware/ABI verification;
- QEMU/emulator para cross-arch quando aplicável;
- differential output entre assembler/disassembler;
- perf counters para claims microarchitecturais;
- feature-disabled hardware fixture/fallback.

# Filosofia

Assembly não deve ser recomendado como otimização por intuição. Primeiro medir, inspecionar compiler output, vectorization reports e profiling; só então decidir entre source-level fix, intrinsics ou handwritten assembly.

# Referências

- Intel SDM: [https://www.intel.com/content/www/us/en/developer/articles/technical/intel-sdm.html](https://www.intel.com/content/www/us/en/developer/articles/technical/intel-sdm.html)
- Arm architecture documentation: [https://developer.arm.com/documentation](https://developer.arm.com/documentation)
- RISC-V Ratified Specifications: [https://docs.riscv.org/reference/home/index.html](https://docs.riscv.org/reference/home/index.html)
- LLVM IR: [https://llvm.org/docs/LangRef.html](https://llvm.org/docs/LangRef.html)
- Linux kernel docs: [https://docs.kernel.org/](https://docs.kernel.org/)

# Exit Gate

Toda orientação assembly conhece ISA+syntax+ABI+object format; performance claims exigem measurement; cross-platform ABI não é inferida; privileged code ativa security/testing especializados.
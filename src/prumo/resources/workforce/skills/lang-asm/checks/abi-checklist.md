# Assembly x86_64 & ARM64 ABI Safety Checklist

## 1. Calling Convention & Register Preservation
- [ ] **System V AMD64**: Integer arguments mapped correctly: `rdi`, `rsi`, `rdx`, `rcx`, `r8`, `r9`.
- [ ] **Callee-Saved Invariant**: Registers `rbx`, `rbp`, `r12`, `r13`, `r14`, `r15` are saved on entry and restored on exit.
- [ ] **Caller-Saved Scratch**: Scratch registers (`rax`, `rcx`, `rdx`, `rsi`, `rdi`, `r8`-`r11`, `xmm0`-`xmm15`) are assumed clobbered across `call` instructions.
- [ ] **Return Values**: Integer returns in `rax`/`rdx`; floating-point returns in `xmm0`/`xmm1`.

## 2. Stack Frame & Alignment Mathematics
- [ ] **16-Byte Alignment**: Immediately before any `call` instruction, `RSP` satisfies `(RSP % 16) == 0`.
- [ ] **Prologue / Epilogue Balance**: Every `push` is balanced by an identical `pop` or `rsp` adjustment before `ret`.
- [ ] **Red Zone Hygiene**: 128-byte red zone usage is restricted to leaf functions on System V AMD64 user space; never used in functions that execute nested `call`s.

## 3. DWARF Unwinding & Debuggability
- [ ] **CFI Directives**: `.cfi_startproc` and `.cfi_endproc` surround each function.
- [ ] **Stack Offset Tracking**: Every `push`, `pop`, or `sub rsp, N` updates CFI via `.cfi_def_cfa_offset` or `.cfi_adjust_cfa_offset`.
- [ ] **Frame Pointer Registration**: When `rbp` is used as frame pointer, `.cfi_def_cfa_register rbp` is declared.

## 4. Security Mitigations
- [ ] **Non-Executable Stack**: Every assembly source concludes with `.section .note.GNU-stack,"",@progbits`.
- [ ] **Control-Flow Enforcement (CET)**: Entry points of global procedures include `endbr64`.
- [ ] **Direction Flag**: The direction flag `DF` is cleared (`cld`) before executing string instructions (`rep movsb`, `stosq`).
- [ ] **RIP-Relative Addressing**: All static data and jump tables use position-independent RIP-relative addressing (`[rip + symbol]`).

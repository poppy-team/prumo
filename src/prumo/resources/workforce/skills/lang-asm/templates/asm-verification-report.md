# Assembly x86_64 / ARM64 ABI Audit Report: {{Target.Routine}}

## 1. Metadata
- **Routine Identifier**: `{{routine_symbol}}`
- **Source File**: `{{source_file}}`
- **Target Architecture**: `x86_64` / `aarch64`
- **Target ABI**: [System V AMD64 | Microsoft x64 | ARM AAPCS64]
- **Date**: {{YYYY-MM-DD}}
- **Author / Agent**: {{agent_id}}

---

## 2. ABI & Calling Convention Compliance
- **Argument Registers Mapped**: `RDI`, `RSI`, `RDX`, `RCX`, `R8`, `R9` (System V)
- **Callee-Saved Registers Modified**: `{{callee_saved_regs_modified}}`
  - Preservation in Prologue (`push`): [VERIFIED / ALL SAVED]
  - Restoration in Epilogue (`pop`): [VERIFIED / REVERSE ORDER]
- **Return Value Placement**: `RAX` (64-bit integer) / `XMM0` (floating point)

---

## 3. Stack Frame & Alignment Mathematics
- **Entry `RSP` State**: `8 mod 16` (after return address pushed)
- **Downstream `call`s Made**: [YES / NO]
  - If YES: `RSP` adjusted by: `{{rsp_adjustment_bytes}}` bytes
  - Stack alignment before every `call`: `RSP mod 16 == 0` [VERIFIED]
- **Red Zone Utilization**: [LEAF ONLY / NOT USED]

---

## 4. DWARF CFI & Security Audit
- **CFI Directives Present**:
  - `.cfi_startproc` & `.cfi_endproc`: [PASS]
  - Stack offset tracking (`.cfi_def_cfa_offset`): [PASS]
- **Security Mitigations**:
  - Non-executable stack: `.section .note.GNU-stack,"",@progbits` [PRESENT]
  - Intel CET Landing Pad: `endbr64` present at function entry [YES / N/A]
  - Direction Flag: Cleared (`cld`) prior to string operations [PASS]

---

## 5. Verification & Assembly Output
- **Compilation Command**: `as -o {{object_file}} {{source_file}}`
- **Assembler Result**: [EXIT 0 / CLEAN]
- **ELF Stack Flags**: `readelf -l {{object_file}}` -> `RW` (No `E` execute permission)

```text
{{assembler_output_log}}
```

---

## 6. Architecture Compliance Sign-Off
- [ ] No unpreserved callee-saved registers.
- [ ] Mathematical 16-byte stack alignment guaranteed before all calls.
- [ ] Non-executable stack note confirmed in binary object.

# x86_64 System V AMD64 ABI Compliant Assembly Example
# Features:
# - Intel CET IBT landing pad (endbr64)
# - CFI unwind directives (.cfi_startproc, .cfi_endproc, .cfi_def_cfa_offset)
# - Proper 16-byte stack alignment before nested calls
# - Callee-saved register preservation (rbx)
# - Non-executable stack note (.note.GNU-stack)

.intel_syntax noprefix
.text

# ------------------------------------------------------------------------------
# 1. Leaf Function: safe_add_integers(int64_t a, int64_t b) -> int64_t
# System V AMD64 ABI:
#   RDI = a
#   RSI = b
#   RAX = return value
# ------------------------------------------------------------------------------
.globl safe_add_integers
.type safe_add_integers, @function
safe_add_integers:
    .cfi_startproc
    endbr64                    # Intel CET Landing Pad

    mov rax, rdi               # RAX = a
    add rax, rsi               # RAX = a + b
    ret
    .cfi_endproc
.size safe_add_integers, .-safe_add_integers

# ------------------------------------------------------------------------------
# 2. Non-Leaf Function: safe_vector_dot(int64_t* a, int64_t* b, size_t len)
# System V AMD64 ABI:
#   RDI = pointer to a (int64_t*)
#   RSI = pointer to b (int64_t*)
#   RDX = len (size_t)
#   RAX = dot product sum
# ------------------------------------------------------------------------------
.globl safe_vector_dot
.type safe_vector_dot, @function
safe_vector_dot:
    .cfi_startproc
    endbr64

    # Save callee-saved register RBX
    push rbx
    .cfi_def_cfa_offset 16
    .cfi_offset rbx, -16

    # Align stack to 16 bytes before any call or local work:
    # After 'call': RSP = 8 mod 16
    # After 'push rbx': RSP = 0 mod 16
    # Need to keep it 0 mod 16 or adjust by multiple of 16 if making calls

    xor rax, rax               # Accumulator = 0
    xor rcx, rcx               # Loop index = 0

.L_dot_loop:
    cmp rcx, rdx
    jge .L_dot_done

    mov r8, [rdi + rcx * 8]    # Load a[i]
    imul r8, [rsi + rcx * 8]   # r8 = a[i] * b[i]
    add rax, r8                # Accumulate

    inc rcx
    jmp .L_dot_loop

.L_dot_done:
    # Restore callee-saved register RBX
    pop rbx
    .cfi_def_cfa_offset 8
    .cfi_restore rbx

    ret
    .cfi_endproc
.size safe_vector_dot, .-safe_vector_dot

# ------------------------------------------------------------------------------
# 3. Security Directive: Non-Executable Stack Note
# Prevents the linker from marking the application stack as executable (RWE)
# ------------------------------------------------------------------------------
.section .note.GNU-stack,"",@progbits

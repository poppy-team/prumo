# C3 Systems Programming & Safety Model: Technical Reference Guide

## 1. Philosophical Overview

C3 is designed as a direct evolutionary upgrade to the C programming language. It retains the low-level machine model, execution performance, and predictable ABI of C while systematically eliminating the linguistic pitfalls that cause memory corruption, resource leaks, and ambiguous error states.

---

## 2. Defer Semantics: Standard, Catch, and No-Catch

C3 extends classical `defer` mechanisms with conditional execution modes:

```c3
fn void*! create_session(int id) {
    void* buffer = mem::alloc(512);
    // Standard defer: always executes at scope exit
    defer io::printfln("create_session completed");

    // Catch defer: executes ONLY if an error/fault occurs
    defer (catch err) mem::free(buffer);

    // No-catch defer: executes ONLY on successful return
    defer (no_catch) io::printfln("Session initialized successfully");

    if (id < 0) return Error.INVALID_ID?;

    return buffer;
}
```

---

## 3. Slices vs Pointers

A C3 slice `T[]` is a first-class fat pointer consisting of two words:
- Pointer to memory: `T*`
- Length of slice: `usz len`

```c3
// Array declaration
int[4] arr = { 1, 2, 3, 4 };

// Creating a slice from an array or pointer
int[] s = &arr;

// Slicing operations
int[] first_two = s[0..2];

// Bounds safety
// In Safe builds, accessing s[10] immediately traps with an out-of-bounds panic
```

---

## 4. Fault Handling Model

Unlike C's reliance on `errno` or C++ exceptions, C3 uses native fault returns (`!Type`):

```c3
fault MathError {
    DIVIDE_BY_ZERO,
    OVERFLOW
}

fn int! divide(int a, int b) {
    if (b == 0) return MathError.DIVIDE_BY_ZERO?;
    return a / b;
}

// Invocation and handling
fn void run() {
    // 1. Explicit catch branch
    if (catch err = divide(10, 0)) {
        io::printfln("Division failed: %s", err);
        return;
    }

    // 2. Propagate fault upwards with '!'
    int result = divide(10, 2)!;
}
```

---

## 5. Contract & Function Annotations

C3 provides attributes for compile-time contract enforcement:
- `@pure`: Guarantees the function accesses only its parameters and causes zero observable state modifications.
- `@nodiscard`: Generates a compilation error if the return value or fault is ignored.
- `@inline`: Advises the compiler to inline the procedure body.
- `@deprecated("reason")`: Emits compiler deprecation warnings.

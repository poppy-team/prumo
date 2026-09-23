# Compiler Engineering & Conformance Invariants Checklist

## 1. Lexical Analysis & Token Invariants
- [ ] **Accurate Source Spans**: Every token contains valid byte offset, line, and column indices.
- [ ] **Unicode Compliance**: Identifiers and string literals handle multi-byte UTF-8 sequences properly without truncation.
- [ ] **EOF Handling**: Token stream terminates cleanly with explicit `Token::Eof` containing final source position.

## 2. Parsing & Error Recovery
- [ ] **Resilient Recovery**: Parser resynchronizes at statement delimiters (`;`, `}`, `let`, `fn`) and reports multiple distinct errors on corrupted source files.
- [ ] **Precedence & Associativity**: Pratt parser correctly enforces relative binding powers for infix, prefix, and postfix operators.
- [ ] **Memory Management**: AST nodes are allocated within an `Arena` or pool; no individual uncoordinated `malloc`/`new` operations.
- [ ] **Trivia & CST Preservation**: If targeting language server tooling, comments and whitespace are preserved in node trivia.

## 3. Semantic Analysis & Type System
- [ ] **Scope Resolution**: Symbol table correctly resolves shadowing, nested block scopes, and lexical closures.
- [ ] **Type Soundness**: Bidirectional type checking correctly infers types bottom-up and checks expressions top-down.
- [ ] **Definite Assignment**: Accessing uninitialized variables produces a compile-time error.
- [ ] **Exhaustiveness**: Pattern matching checks all enum variants or type union branches; warns/errors on non-exhaustive match without wildcard.

## 4. Diagnostics & Reporting
- [ ] **Diagnostic Visuals**: Errors display the offending source line with caret underline (`^^^^`) at the exact column span.
- [ ] **Deterministic Ordering**: Diagnostic reports are sorted deterministically by `(file, line, col)` across parallel compilation units.
- [ ] **Actionable Suggestions**: Common syntax mistakes (e.g. missing semicolons, misspelled identifiers) provide `"did you mean X?"` suggestions.

## 5. IR Optimization & Backend
- [ ] **SSA Dominance**: Variable definitions strictly dominate all their uses in the Control Flow Graph.
- [ ] **Semantics Preserving**: Optimizations (constant folding, DCE, GVN) preserve observable side effects and termination properties.
- [ ] **ABI Conformance**: Generated machine code honors 16-byte stack alignment, callee-saved registers, and non-executable stack flags.

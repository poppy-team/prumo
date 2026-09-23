# Compiler Architecture Specification: {{Language.Name}}

## 1. Executive Summary & Pipeline Strategy
- **Compiler Name**: `{{compiler_name}}`
- **Source Language**: `{{source_language_name}}`
- **Target Backend(s)**: [LLVM IR | WebAssembly | Native x86_64/AArch64 | C Transpilation]
- **Execution Strategy**: [AOT Compiler | JIT Compiler | Bytecode VM | Tree-Walking Interpreter]

---

## 2. Grammar & Operator Precedence Matrix

| Operator | Associativity | Precedence (Binding Power) | Semantics |
| :--- | :--- | :--- | :--- |
| `()` / `[]` / `.` | Left | 100 | Call, Index, Member Access |
| `-` / `!` | Prefix | 80 | Unary Negation / Logical NOT |
| `*` / `/` / `%` | Left | 60 | Multiplicative Arithmetic |
| `+` / `-` | Left | 50 | Additive Arithmetic |
| `<` / `>` / `==` | Non-assoc | 40 | Relational Comparisons |
| `&&` | Left | 30 | Logical AND (short-circuit) |
| `\|\|` | Left | 20 | Logical OR (short-circuit) |
| `=` / `+=` / `-=` | Right | 10 | Assignment Operations |

---

## 3. Abstract Syntax Tree (AST) & Memory Model
- **Allocation Strategy**: Monolithic `Arena` allocator for all AST nodes per compilation unit.
- **Node Taxonomy**:
  - `Stmt`: `VarDecl`, `Assign`, `If`, `While`, `Return`, `ExprStmt`
  - `Expr`: `Literal`, `Ident`, `Binary`, `Unary`, `Call`, `Block`
  - `Type`: `Primitive`, `Pointer`, `Array`, `Function`, `Struct`
- **Span Tracking**: Every node contains `span: { file_id, start_offset, end_offset }`.

---

## 4. Semantic Analysis & Type System Invariants
- **Inference Model**: [Bidirectional | Hindley-Milner | Explicit Nominal]
- **Symbol Table Structure**: Scoped chained hash maps with lexical parent pointers.
- **Definite Assignment Rules**: All local variables must be assigned prior to read.
- **Pattern Match Exhaustiveness**: Verified via decision tree compilation or matrix algorithm.

---

## 5. Diagnostic Reporting Contract
- **Diagnostic Format**:
  ```text
  error[E0123]: mismatched types
   --> src/main.lang:14:9
    |
  14|     let x: int = "string";
    |         ^ expected 'int', found 'string'
    |
    = note: expected integer primitive type
    = help: convert string to integer using `int::parse(...)`
  ```

---

## 6. IR & Optimization Passes
- **IR Representation**: Basic Blocks of SSA instructions.
- **Pass Pipeline**:
  1. High-level AST Lowering to SSA.
  2. Sparse Conditional Constant Propagation (SCCP).
  3. Dead Code Elimination (DCE).
  4. Global Value Numbering (GVN).
  5. Machine Instruction Selection & Register Allocation.

---

## 7. Verification & Conformance Test Plan
- [ ] Lexer test suite covering all tokens and malformed UTF-8.
- [ ] Parser conformance test suite for valid grammar and precedence.
- [ ] Error recovery test suite asserting multiple diagnostics emitted.
- [ ] Type checker positive and negative test suites.
- [ ] Codegen end-to-end execution tests.

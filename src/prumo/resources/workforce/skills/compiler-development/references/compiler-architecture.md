# Modern Compiler Architecture: Front-End to Back-End Pipeline

## 1. Classical and Modern Pipeline Stages

```
Source Code
    │
    ▼ (Lexer / Scanner)
Token Stream [with Source Spans & Trivia]
    │
    ▼ (Pratt / Recursive-Descent Parser)
Abstract Syntax Tree (AST) [Arena Allocated]
    │
    ▼ (Name Resolution & Type Checker)
Decorated / Typed AST + Symbol Tables
    │
    ▼ (IR Lowering)
High-Level IR / SSA Control Flow Graph (CFG)
    │
    ▼ (Optimization Passes: SCCP, DCE, GVN, Inlining)
Optimized SSA IR
    │
    ▼ (Instruction Selection & Register Allocation)
Target Machine Code (x86_64, AArch64, WASM)
```

---

## 2. Pratt Parsing (Top-Down Operator Precedence)

Vaughan Pratt's 1973 algorithm elegantly solves parsing mathematical and logical expressions with diverse operator precedences and associativities without requiring grammar factorization or deep grammar recursion.

### 2.1 Binding Power Mechanics
Each token has:
- **Prefix Parselet (Null Denotation - Nud)**: Handles literals, identifiers, unary operators (`-x`, `!x`), and parenthesized groups `(expr)`.
- **Infix Parselet (Left Denotation - Led)**: Handles binary operators (`+`, `-`, `*`, `/`), assignments (`=`), calls (`foo(x)`), and array indexing (`arr[i]`).
- **Left Binding Power ($LBP$)** and **Right Binding Power ($RBP$)**: Determine operator precedence and associativity:
  - *Left-associative* (`+`, `-`): $RBP = LBP + 1$.
  - *Right-associative* (`=`, `^`): $RBP = LBP$.

```cpp
// Core Pratt expression parsing loop
Expr* parse_expr(int min_bp) {
    Token token = advance();
    Expr* left = parse_nud(token);

    while (peek().lbp() > min_bp) {
        Token op = advance();
        left = parse_led(op, left);
    }
    return left;
}
```

---

## 3. Resilient Error Recovery & Panic Mode Synchronization

Compilers must never stop on the first error. Resilient parsing guarantees multiple errors are surfaced:

```
Error encountered: 'unexpected token in expression'
   │
   ├─► Emit diagnostic: "Expected expression, found '}' at line 42, col 12"
   │
   └─► Panic-mode resynchronization:
       Skip tokens until encountering:
       - Semicolon (';')
       - Closing brace ('}')
       - Keyword starting a statement ('let', 'fn', 'if', 'return')
       Then resume standard statement parsing!
```

---

## 4. Static Single Assignment (SSA) Form & Dominance

An intermediate representation is in SSA form if and only if:
1. Every variable is defined (assigned) exactly once.
2. Every use of a variable is dominated by its definition.

### 4.1 Dominance Definition
A basic block $D$ **dominates** a block $N$ ($D \text{ dom } N$) if every path from the entry node to $N$ must go through $D$.

### 4.2 $\phi$ (Phi) Node Placement
When control flow branches merge (e.g. after an `if-else`), a variable assigned in both branches has multiple potential definitions. A $\phi$-node selects the value based on the predecessor basic block that executed immediately prior to entering the merge block:
$$x_3 = \phi(x_1, x_2)$$

---

## 5. Register Allocation Taxonomy

- **Chaitin-Briggs Graph Coloring**:
  - Models interference between live ranges as an undirected graph.
  - NP-complete in general; uses Kempe's heuristic to find a $K$-coloring (where $K$ is the number of available physical CPU registers).
  - High compilation time overhead, produces near-optimal machine code. Preferred for release builds (`-O2`, `-O3`).
- **Poletto-Sarkar Linear Scan**:
  - Sorts live ranges by start point and allocates registers linearly.
  - Scales in $O(N)$ time with live interval count.
  - Essential for Just-In-Time (JIT) engines (V8, WebKit JavaScriptCore, JVM).

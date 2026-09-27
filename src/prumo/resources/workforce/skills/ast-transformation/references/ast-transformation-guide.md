# AST Transformation, Codemods, and Lossless Syntax Trees: Technical Reference Guide

## 1. Abstract Syntax Trees (AST) vs Concrete Syntax Trees (CST)

- **Standard AST**: Strips all non-semantic tokens (spaces, line breaks, comments, optional parentheses, trailing commas). Excellent for compilers; catastrophic for source code rewriting because re-serializing an AST wipes out developer formatting and comments.
- **Concrete Syntax Tree (CST / Lossless Tree)**: Retains 100% of characters in the original file as tokens or trivia.
- **Red-Green Tree Architecture** (used by Roslyn, rust-analyzer, Swift, Biome):
  - **Green Nodes**: Immutable, functional, relative widths, hold trivia. Can be cached and shared across trees.
  - **Red Nodes**: Lightweight wrappers lazily instantiated around green nodes to provide absolute file offsets and parent navigation.

---

## 2. Text-Range Diffing vs Whole-File Printing

When executing codemods, avoid re-serializing the entire AST back into text. Instead, emit **Text Edits**:

$$\text{Edit} = (\text{start\_offset}, \text{end\_offset}, \text{replacement\_string})$$

```
Original:
  function calculate(x) {
      /* compute total */
      return legacy_math.add(x, 10);
  }

Target Node: legacy_math.add(x, 10)  [Span: byte 58 to byte 81]
Replacement: modern_add(x, 10)

Result:
  Only bytes 58..81 are replaced.
  Comments, indentation, and outer function remain 100% byte-for-byte identical!
```

---

## 3. Scope Hygiene and Variable Capture

When an AST transformation injects a new variable or helper function into user code, it must not accidentally capture or shadow existing variables:

```javascript
// DANGEROUS REWRITE:
// User code:
const temp = "important session data";
// Codemod injects swap helper using variable 'temp':
const temp = a; a = b; b = temp; // CRASH / SHADOWING ERROR!

// HYGIENIC REWRITE:
// Generate a unique identifier guaranteed not to exist in enclosing scope:
const __prumo_swap_temp_0 = a; a = b; b = __prumo_swap_temp_0;
```

---

## 4. Idempotence Proof

A transformation pass $T$ is idempotent if and only if for any input program $P$:
$$T(T(P)) = T(P)$$

### 4.1 Common Non-Idempotent Bugs
1. **Unconditional Import Insertion**: Injecting `import { x } from 'mod'` every run without checking if `x` is already imported, leading to duplicated import lines.
2. **Double Wrapping**: Wrapping an expression in a helper `wrap(expr)` without checking if `expr` is already an instance of `wrap(...)`.
3. **Repeated Renaming**: Renaming `oldName` to `newName`, but then having another rule that transforms `newName` to `newNewName`.

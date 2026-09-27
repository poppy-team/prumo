# AST Transformation & Codemods Verification Checklist

## 1. Syntax Parsing & Trivia Fidelity
- [ ] **Lossless Parsing**: Parser retains comments, whitespace, and formatting for untouched source regions.
- [ ] **Accurate Node Spans**: Transformed nodes preserve or correctly compute new source line/column spans.
- [ ] **Error Resilience**: Files with minor syntax errors fail gracefully without writing partial corruptions to disk.

## 2. Transformation Safety & Scope Hygiene
- [ ] **Hygienic Identifiers**: Newly introduced bindings use collision-free names (`gensym` or scoped prefix) to prevent accidental shadowing.
- [ ] **Scope Disambiguation**: Pattern matcher verifies that target identifiers are imported from target modules rather than shadowed locally.
- [ ] **Import Management**: Deprecated imports removed; newly required symbols added to import declarations cleanly.

## 3. Idempotence & Correctness
- [ ] **Idempotence Verified**: Running the codemod on previously transformed code produces zero file modifications ($T(T(x)) == T(x)$).
- [ ] **Targeted Diffs**: Diffs are restricted strictly to targeted code structures; no unnecessary global file reformats.
- [ ] **Golden Snapshots**: Test suite includes comprehensive `.before` and `.after` snapshot pairs.

## 4. Verification & Cleanliness
- [ ] **Post-Transform Syntax Validity**: Transformed code parses cleanly with native language compiler/interpreter.
- [ ] **Test Suite Green**: Downstream unit tests pass after applying the codemod.

# Clang C/C++ Context Indexing — Verification Checklist

## 1. Compilation Database Integrity
- [ ] `compile_commands.json` exists at the repository root or a documented path and parses as valid JSON.
- [ ] At least 98% of translation units carry explicit `-std=`, `-I`, and `-D` flags; entries with empty commands are rejected.
- [ ] Database was produced by a reproducible command such as `bear -- make -j8` or a CMake exported-commands build, recorded in the spec.

## 2. Symbol Extraction Completeness
- [ ] Visitor covers function, method, variable, typedef, enum, field, call, and declaration-reference cursors.
- [ ] Every symbol row stores USR, qualified spelling, cursor kind, file, line, column, and extent hash.
- [ ] Macro references record both expansion and spelling locations.

## 3. Incremental Reindex Correctness
- [ ] File tracking uses SHA-256 of each translation unit plus its `-MD` depfile header closure.
- [ ] Editing one header reparses only dirty units and dependents; unrelated rows are untouched.
- [ ] Rows whose file hash mismatches are deleted; no orphaned references survive reindexing.
- [ ] Failed translation units appear in a quarantine list with diagnostics instead of vanishing.

## 4. Query API & Ranking Guarantees
- [ ] Definition, reference, and depth-bounded caller queries resolve by USR, never by file plus line.
- [ ] Ranking applies exact-name, file-proximity, include-distance, and recency weights in documented order.
- [ ] All query paths enforce the 40-symbol and 12000-token output caps.

## 5. Performance & Verification Evidence
- [ ] Full index of the target corpus completes inside the documented wall-clock budget.
- [ ] Query p95 latency on a warm index measures under 250 ms.
- [ ] Golden symbol recall is 100% with the recall log attached.
- [ ] `scripts/verify.sh` executes with exit code 0.

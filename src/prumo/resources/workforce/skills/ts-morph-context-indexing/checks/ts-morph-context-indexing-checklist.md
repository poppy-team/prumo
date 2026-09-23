# TypeScript ts-morph Context Indexing — Verification Checklist

## 1. Project Loading & Compiler Options
- [ ] Project is constructed from the root tsconfig; compiler options parse without fallback defaults.
- [ ] Source-file count matches the expected inventory; project references resolve.
- [ ] ts-morph and TypeScript versions are recorded and match the build toolchain.

## 2. Typed Symbol Extraction
- [ ] Class, interface, function, type-alias, enum, and variable-statement nodes are all visited.
- [ ] Heritage clauses and return types are recorded via the compiler type checker.
- [ ] Path aliases resolve through compiler options, never by string guessing.

## 3. References, Implementations & Import Graph
- [ ] `findReferences` covers cross-file call sites for golden query symbols.
- [ ] `getImplementations` resolves interface and abstract-class members.
- [ ] Import declarations store resolved module specifiers; barrels carry re-export edges to defining modules.

## 4. Incremental Freshness & Declaration Policy
- [ ] File changes refresh via filesystem sync calls inside forget-block scopes; stale nodes never leak.
- [ ] Removed files delete their rows in the same pass.
- [ ] Declaration-file symbols carry declaration markers preferring implementation sources.

## 5. Ranking & Verification Evidence
- [ ] Ranking applies exact qualified-name, same-package, import-distance, and recency order.
- [ ] All query paths enforce the 40-symbol and 12000-token caps.
- [ ] Golden recall is 100% including overloads, merging, generics, and barrels.
- [ ] Query p95 under 300 ms warm and `scripts/verify.sh` exits 0.

# ts-morph Context Index — Deliverable Specification

## 1. Project & Toolchain
- **Monorepo**: Harbor storefront at /srv/harbor, 860 TypeScript files, 246,000 lines
- **tsconfig**: root tsconfig.json with bundler module resolution and 6 project references
- **Versions**: ts-morph 24.0.0, TypeScript 5.6.3, Node 20.18.1
- **Load result**: 860 of 860 expected source files loaded 2026-09-13
- **Path aliases**: @shop/*, @checkout/*, @shared/*, each verified against one aliased import per package

## 2. Extraction Coverage
- **Symbols stored**: 29,640 rows with qualified names, heritage clauses, and return types
- **Reference edges stored**: 88,412 rows from findReferences across packages
- **Implementations mapped**: 148 interface and abstract members to 402 concrete classes
- **Import edges**: 9,204 resolved specifiers including 214 barrel re-export edges

## 3. Freshness Policy
- **Refresh calls**: refreshFromFileSystem for edits, createSourceFile for adds, delete for removals
- **Leak guard**: all query paths run inside forgetNodesCreatedInBlock scopes
- **Measured cost**: single-file edit in checkout package fresh in 1.8 seconds
- **Declaration markers**: 3,120 symbols flagged from .d.ts files, ranked below implementation sources

## 4. Query API & Ranking
- **Commands**: `symbols --name 'CheckoutService.charge'`, `refs --symbol 'CheckoutService.charge'`, `imports --file 'packages/checkout/service.ts'`
- **Ranking weights**: exact qualified-name 1.0, same-package proximity 0.6, import-distance hops 0.3, recency 0.1
- **Caps enforced**: 40 symbols and 12000 tokens per answer on every code path

## 5. Measured Results
- **Project load wall clock**: 38 seconds cold on the reference machine
- **Query p95 warm**: 224 ms over 400 sampled queries
- **Golden recall**: 160 of 160 golden symbols resolved, 100%, including 21 overloads, 12 merged declarations, 17 generics, and 15 barrel re-exports

## 6. Verification Evidence
- Source inventory at .prumo/index/tsmorph-files-2026-09-13.log, 860 files
- Recall log at .prumo/index/tsmorph-recall-2026-09-13.log
- `scripts/verify.sh` exit code 0 on 2026-09-13

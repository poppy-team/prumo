# Clang Context Index — Deliverable Specification

## 1. Corpus & Toolchain
- **Repository**: HeliosEngine, C++20 game engine, rooted at /srv/helios
- **Scope**: src/ plus include/, 842 translation units, 1.9 MLOC
- **Clang version**: 17.0.6, libclang Python bindings 17.0.6, clangd 17.0.3
- **Compilation database**: generated 2026-09-18 via `bear -- make -j8`, 842 of 842 units with flags, 0 empty commands
- **Storage**: SQLite at .prumo/index/clang-symbols.sqlite3, FTS5 on qualified names

## 2. Extraction Coverage
- **Cursor kinds visited**: FunctionDecl, CXXMethod, VarDecl, TypedefDecl, EnumDecl, FieldDecl, CallExpr, DeclRefExpr
- **Symbols stored**: 61,204 rows with USR, qualified spelling, kind, file, line, column, extent hash
- **Reference edges stored**: 188,930 caller-to-callee rows
- **Quarantined units**: 3, namely src/platform/legacy_oss.cpp with missing ALSA headers, diagnostics attached in section 6

## 3. Incremental Policy
- **Dirty detection**: SHA-256 of translation unit plus `-MD` depfile header closure
- **Measured cost**: single dirty unit with 14 dependent units reindexed in 3.8 seconds
- **Orphan rule**: rows whose file hash mismatches are deleted in the same transaction as the reparse

## 4. Query API & Ranking
- **Commands**: `symbols --name 'Renderer::draw' --limit 20`, `refs --usr c:@N@Renderer@F@draw#`, `callers --usr c:@N@Renderer@F@draw# --depth 2`
- **Ranking weights**: exact qualified-name match 1.0, same-directory proximity 0.6, include-distance hops 0.3, recency 0.1
- **Caps enforced**: 40 symbols and 12000 tokens per answer on every code path

## 5. Measured Results
- **Full index wall clock**: 3 min 12 s on 8 cores
- **Query p95 warm**: 184 ms over 500 sampled queries
- **Golden recall**: 240 of 240 golden symbols resolved, 100%

## 6. Verification Evidence
- Quarantine diagnostics for the 3 failed units archived at .prumo/index/quarantine-2026-09-18.log
- Recall log at .prumo/index/golden-recall-2026-09-18.log
- `scripts/verify.sh` exit code 0 on 2026-09-18

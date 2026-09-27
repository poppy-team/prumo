# Clang C/C++ Context Indexing — Technical Reference Guide

## 1. Core Concepts

### 1.1 Translation Units and the Compilation Database
Clang parses one translation unit at a time: a `.cpp` file plus everything it includes. Correct flags per unit are non-negotiable, which is why `compile_commands.json` is the foundation. Each entry binds a file to its exact compiler invocation:

```json
{
  "directory": "/srv/helios/build",
  "command": "clang++ -std=c++20 -I/srv/helios/include -DRENDER_VULKAN=1 -c ../src/renderer.cpp",
  "file": "/srv/helios/src/renderer.cpp"
}
```

Generate it with `bear -- make -j8` for Make builds or `cmake -DCMAKE_EXPORT_COMPILE_COMMANDS=ON -B build` for CMake. `clangd --background-index` consumes the same file automatically.

### 1.2 Cursors and USRs
libclang exposes the AST as cursors (`CXCursor`). Declarations of interest include `CXCursor_FunctionDecl`, `CXXMethod`, `VarDecl`, `TypedefDecl`, `EnumDecl`, and `FieldDecl`; reference edges come from `CallExpr` and `DeclRefExpr`. The Unified Symbol Resolution (USR) string is the stable identity of a symbol across edits — file plus line is not an identity and must never be used as one.

### 1.3 Header Closures via Depfiles
Compiling with `-MD -MF renderer.d` emits the full header closure of a unit. Hashing the unit plus every listed header gives the dirty check for incremental reindexing: only units whose closure hash changed are reparsed.

### 1.4 Ranking Model
Candidate symbols are scored as `1.0 * exact_match + 0.6 * same_dir + 0.3 * (1 / (1 + include_hops)) + 0.1 * recency`, then truncated to the symbol and token caps. Exact qualified-name matches therefore always outrank proximity heuristics.

## 2. Minimal Extraction Example

```python
import clang.cindex as CX

index = CX.Index.create()
tu = index.parse("src/renderer.cpp", args=[
    "-std=c++20", "-Iinclude", "-DRENDER_VULKAN=1",
])
def walk(cursor, out):
    if cursor.kind in (CX.CursorKind.FUNCTION_DECL,
                       CX.CursorKind.CXX_METHOD,
                       CX.CursorKind.CALL_EXPR):
        out.append((cursor.get_usr(), cursor.kind.name,
                    cursor.spelling, cursor.location.line))
    for child in cursor.get_children():
        walk(child, out)
rows = []
walk(tu.cursor, rows)
print(f"extracted {len(rows)} symbols and edges")
```

Guard this loop with a quarantine: wrap `index.parse` per unit in try/except, log diagnostics from `tu.diagnostics`, and record failures instead of aborting the whole pass.

## 3. Common Pitfalls
- Indexing headers standalone: headers have no flags of their own; always index them through their including translation units.
- Ignoring implicit template instantiations: request them explicitly or call-graph edges through templates will be missing.
- Storing absolute build-machine paths: rewrite to repository-relative paths so the index survives checkout moves.

# ts-morph Context Indexing — Technical Reference Guide

## 1. Core Concepts

### 1.1 Projects Bound to tsconfig
ts-morph wraps the TypeScript compiler, so the project must load the same options the build uses. Loading from the workspace tsconfig keeps module resolution identical:

```ts
import { Project } from "ts-morph";

const project = new Project({ tsConfigFilePath: "tsconfig.json" });
const files = project.getSourceFiles();
console.log(`loaded ${files.length} source files`);
```

If the file count diverges from the expected inventory, project references or include globs are misconfigured and the index must not proceed.

### 1.2 Typed Node Visits
Every declaration node exposes both syntax and compiler types. Recording heritage and return types at extraction time avoids re-querying the checker later:

```ts
for (const cls of sourceFile.getClasses()) {
  const entry = {
    name: cls.getName(),
    extends: cls.getExtends()?.getText(),
    methods: cls.getMethods().map((m) => ({
      name: m.getName(),
      returnType: m.getReturnType().getText(),
    })),
  };
  store(entry);
}
```

`getReturnType().getText()` resolves through aliases and generics exactly as the compiler reports them.

### 1.3 References and Implementations
`node.findReferences()` returns every referencing node solution-wide, while `getImplementations()` on interfaces and abstract members maps to concrete classes. Import edges come from `getImportDeclarations()` with `getModuleSpecifierSourceFile()` resolving barrels, aliases, and relative paths to real files.

### 1.4 Incremental Freshness Without Leaks
ts-morph caches AST nodes aggressively, so edits must synchronize the filesystem view and discard cached nodes:

```ts
project.forgetNodesCreatedInBlock(() => {
  sourceFile.refreshFromFileSystem();
  answer = runQuery(sourceFile);
});
```

Nodes created inside the block are forgotten on exit, which guarantees no stale node survives into the next answer. Additions use `createSourceFile`, removals use `delete()`.

## 2. Module Resolution Notes
- bundler mode matches Vite and esbuild workspaces; node16 matches strict Node ESM packages — mixing them misresolves extensionless imports.
- Path aliases like `@shop/*` resolve only when `paths` in the loaded tsconfig matches the build; verify with one aliased import per package.
- Declaration merging means one qualified name can own several declarations; store all of them, not just the first.

## 3. Common Pitfalls
- Instantiating ts-morph with inline compiler options that contradict the tsconfig, producing an index the build would never see.
- Forgetting barrel re-export edges, which orphans symbols imported through index files.
- Holding node references across refreshes outside forget-blocks, serving pre-edit text as current truth.

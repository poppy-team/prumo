# Roslyn C# Context Indexing — Technical Reference Guide

## 1. Core Concepts

### 1.1 Workspaces, Solutions, and Semantic Models
Roslyn models code as immutable snapshots: a `Solution` contains projects, projects contain documents, each document has a `SyntaxTree`, and `SemanticModel` binds syntax to symbols. Immutability makes incremental indexing safe — a document edit produces a new solution snapshot while old snapshots stay readable for in-flight queries.

### 1.2 Loading a Solution Reliably
MSBuild must be registered before any workspace opens, otherwise project loading fails with opaque composition errors:

```csharp
using Microsoft.Build.Locator;
using Microsoft.CodeAnalysis.MSBuild;

MSBuildLocator.RegisterDefaults();
using var workspace = MSBuildWorkspace.Create();
var solution = await workspace.OpenSolutionAsync("/srv/billing/Billing.sln");
foreach (var diag in workspace.Diagnostics)
    Console.WriteLine($"LOAD {diag.Kind}: {diag.Message}");
```

Treat any load diagnostic as fatal for indexing until resolved or waived; a half-loaded solution yields half-true references.

### 1.3 From Syntax to Symbols
Walk syntax nodes, then bind each declaration through the semantic model to get the canonical `ISymbol`:

```csharp
var model = await doc.GetSemanticModelAsync();
foreach (var node in root.DescendantNodes().OfType<MethodDeclarationSyntax>())
{
    if (model.GetDeclaredSymbol(node) is IMethodSymbol m)
        store(m.ToDisplayString(), m.ContainingAssembly.Name,
              doc.FilePath, node.GetLocation().GetLineSpan().StartLinePosition.Line);
}
```

`IMethodSymbol`, `INamedTypeSymbol`, and `IPropertySymbol` carry overloads, type arguments, and explicit interface implementations that raw text can never distinguish.

### 1.4 References, Implementations, Hierarchies
`SymbolFinder.FindReferencesAsync(symbol, solution)` returns every referencing location solution-wide, `FindImplementationsAsync` maps interface members to concrete overrides, and `INamedTypeSymbol.BaseType` plus `AllInterfaces` walk the hierarchy. Bound hierarchy walks to depth 3 — real .NET chains like `InvoiceCalculator -> CalculatorBase -> Object` plus three implemented interfaces fit comfortably inside that budget.

## 2. Incremental Strategy
Each document carries a `VersionStamp` that advances on every edit. The indexer diffs stamps between snapshots, re-resolves advanced documents, re-resolves their direct dependents for possibly-changed overload resolution, and deletes rows for removed documents. A single-file edit in a 14-project solution typically re-resolves under 30 documents.

## 3. Common Pitfalls
- Indexing without restoring NuGet packages: missing references silently turn semantic binds into error symbols.
- Confusing syntax text with symbol identity: two partial-class files share one `INamedTypeSymbol`; store it once.
- Ignoring multi-targeting: a project targeting net8.0 and net48 can resolve the same name to different assemblies per framework.

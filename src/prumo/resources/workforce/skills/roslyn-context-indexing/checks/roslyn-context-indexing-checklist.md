# Roslyn C# Context Indexing — Verification Checklist

## 1. Workspace Loading & SDK Integrity
- [ ] `MSBuildLocator.RegisterDefaults()` targets the .NET 8 SDK and the Roslyn version is recorded.
- [ ] `OpenSolutionAsync` loads every project; load diagnostics are empty or explicitly waived with justification.
- [ ] Target frameworks per project are listed; multi-targeted projects resolve each framework separately.

## 2. Symbol Extraction & Semantic Resolution
- [ ] Class, method, property, and interface declarations are all visited in every document.
- [ ] Every stored symbol resolved through `SemanticModel.GetDeclaredSymbol`; syntax-only rows are absent.
- [ ] Symbol identity uses fully qualified metadata name plus containing assembly.

## 3. References, Implementations & Hierarchies
- [ ] `FindReferencesAsync` covers cross-project call sites for golden query symbols.
- [ ] `FindImplementationsAsync` resolves interface members to concrete implementations.
- [ ] Base-type and interface chains expand to at most depth 3 and 60 nodes per query.

## 4. Incremental Reindex & Generated-Code Policy
- [ ] Freshness keys on document `VersionStamp`; only advanced documents and dependents re-resolve.
- [ ] Removed documents are purged in the same pass; no orphaned symbols survive.
- [ ] obj/, bin/, and designer outputs are excluded from ranking and listed separately.

## 5. Query Ranking & Verification Evidence
- [ ] Ranking applies exact metadata-name, same-project, dependency-distance, and recency order.
- [ ] All query paths enforce the 40-symbol and 12000-token caps.
- [ ] Golden recall is 100% including overloads, explicit implementations, and generics.
- [ ] Query p95 under 300 ms warm and `scripts/verify.sh` exits 0.

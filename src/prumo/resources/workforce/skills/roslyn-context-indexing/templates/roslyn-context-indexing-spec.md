# Roslyn Context Index — Deliverable Specification

## 1. Solution & Toolchain
- **Solution**: Billing.sln at /srv/billing, 14 projects, 1,180 C# files, 402,000 lines
- **Frameworks**: net8.0 for 12 projects, net8.0 plus net48 dual-target for Billing.LegacyShim
- **Roslyn version**: 4.11.0, .NET 8.0.403 SDK registered via MSBuildLocator
- **Load result**: 14 of 14 projects loaded 2026-09-16, zero load diagnostics
- **Storage**: SQLite at .prumo/index/roslyn-symbols.sqlite3

## 2. Extraction Coverage
- **Symbols stored**: 48,512 rows keyed by metadata name plus assembly
- **Reference edges stored**: 121,770 rows from FindReferencesAsync across all projects
- **Implementations mapped**: 312 interface members to 694 concrete overrides
- **Excluded generated code**: 212 files under obj/, bin/, and 38 Designer.cs files, listed at .prumo/index/roslyn-excluded.log

## 3. Hierarchy Policy
- **Expansion cap**: depth 3, 60 nodes per query
- **Example chain**: InvoiceCalculator derives CalculatorBase derives Object, implements ICalculator, IAuditable, IRefundable
- **Overflow rule**: chains exceeding the cap truncate deepest-first with a truncation flag on the answer

## 4. Incremental Policy
- **Freshness key**: Roslyn document VersionStamp per snapshot
- **Measured cost**: single edit in Billing.Core re-resolved 22 documents in 2.9 seconds
- **Purge rule**: removed documents delete their rows in the same pass

## 5. Measured Results
- **Solution load wall clock**: 47 seconds cold including NuGet restore verification
- **Query p95 warm**: 236 ms over 400 sampled symbol, reference, and hierarchy queries
- **Golden recall**: 180 of 180 golden symbols resolved, 100%, including 24 overload groups, 11 explicit interface implementations, and 19 generic instantiations

## 6. Verification Evidence
- Load diagnostics report at .prumo/index/roslyn-load-2026-09-16.log, empty as required
- Recall log at .prumo/index/roslyn-recall-2026-09-16.log
- `scripts/verify.sh` exit code 0 on 2026-09-16

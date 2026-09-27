# Compiler & Virtual Machine Conformance

## Purpose
Execute end-to-end verification of compiler passes, type systems, SSA lowering, bytecode VM, and grammar fuzzing.

## Preconditions
- Language grammar and type system formal specifications available
- Target VM and execution environment initialized

## Participating Agents
- `architect`
- `compiler-engineer`
- `systems-architect`
- `tester`
- `reviewer`

## Required Skills
- `compiler-development`
- `compiler-frontend-engineering`
- `type-system-theory`
- `compiler-ir-optimization`
- `bytecode-vm-architecture`
- `fuzz-grammar-testing`
- `testing-quality`
- `code-review`

## Quality Gates
- `type-soundness`
- `fuzz-clean`
- `conformance-100`
- `review`

## Stop Conditions
- 100% of language conformance test cases pass without regressions

# TypeScript Strict Soundness & Typing Invariants Checklist

## 1. Compiler Configuration & Soundness
- [ ] `tsconfig.json` contains `"strict": true`.
- [ ] `tsconfig.json` contains `"noUncheckedIndexedAccess": true`.
- [ ] `tsconfig.json` contains `"noImplicitReturns": true` and `"noFallthroughCasesInSwitch": true`.
- [ ] `tsc --noEmit` produces zero diagnostic warnings or errors.

## 2. Type System Discipline
- [ ] Zero usage of the `any` type across all source files.
- [ ] External untyped data is typed as `unknown` before parsing.
- [ ] Zero usage of the non-null assertion operator (`!`) in production code.
- [ ] Zero usage of `@ts-ignore` or `@ts-nocheck` (use `@ts-expect-error` with comment only in test suites).
- [ ] The `satisfies` operator is used to validate object shapes without widening literal types.

## 3. Domain Modeling & Nominal Types
- [ ] Domain entity identifiers use Branded Types (`Brand<string, "UserId">`) to prevent primitive obsession.
- [ ] Multi-state workflows are modeled as Discriminated Unions with a common discriminant field (`status`, `kind`).
- [ ] All `switch` statements over discriminated unions include an exhaustive `default: assertNever(state)` check.

## 4. Boundary Validation Invariants
- [ ] All incoming data (HTTP requests, URL params, environment variables, WebSocket messages) is parsed through runtime schemas (Zod, Valibot, TypeBox).
- [ ] TypeScript types are derived from schemas (`z.infer<typeof Schema>`) to maintain a single source of truth.
- [ ] Zero type assertions (`as TargetType`) applied to unvalidated external payloads.

## 5. Modern Modules & Clean Code
- [ ] ECMAScript Modules (ESM) used with explicit `.js` extensions where required by `NodeNext`.
- [ ] Asynchronous operations properly handle rejection; Promises are awaited or returned with typed signatures.
- [ ] ESLint passes with `@typescript-eslint/strict-type-checked` without warnings.

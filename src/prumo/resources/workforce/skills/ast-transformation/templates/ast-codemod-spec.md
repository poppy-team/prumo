# AST Codemod Specification: {{Codemod.Name}}

## 1. Overview & Migration Scope
- **Codemod ID**: `{{codemod_id}}`
- **Target Language**: [TypeScript | Python | Go | Rust | C++]
- **Target Parser Toolchain**: [Babel / jscodeshift | Python ast | syn/quote | LibTooling]
- **Intent**: {{Brief explanation of deprecated pattern being modernized and target semantic behavior}}

---

## 2. Match Pattern & Trigger Invariants

```typescript
// Target AST Pattern to Match
{
  type: "CallExpression",
  callee: {
    type: "MemberExpression",
    object: { name: "{{legacy_object}}" },
    property: { name: "{{legacy_method}}" }
  },
  arguments: [...]
}
```

- **Scope Pre-conditions**: Must verify `{{legacy_object}}` is imported from `{{source_package}}` and not locally declared.

---

## 3. Transformation & Replacement Rules

### Before Transformation (Input)
```typescript
{{example_input_code}}
```

### After Transformation (Output)
```typescript
{{example_output_code}}
```

### Text Edit / Node Mutation Details
- **Node Replacement**: Convert `{{legacy_object}}.{{legacy_method}}` to `{{modern_api}}`.
- **Arguments Mapping**: {{Explain positional argument shifts or keyword mappings}}.
- **Import Adjustments**: Remove `{{legacy_symbol}}` from `{{legacy_package}}`, add `{{new_symbol}}` to `{{new_package}}`.

---

## 4. Hygiene & Idempotence Proof
- **Hygienic Identifier Generation**: Newly injected bindings prefixed with `__prumo_{{name}}_`.
- **Idempotence Invariant**: If input code already contains `{{modern_api}}`, transformation is a no-op ($T(T(x)) \equiv T(x)$).

---

## 5. Verification & Test Plan
- [ ] Golden test snapshot: `tests/fixtures/nominal.input` -> `tests/fixtures/nominal.output`.
- [ ] Idempotency test asserting second pass produces 0 diffs.
- [ ] Negative test: ensuring non-matching local variables with the same name are ignored.
- [ ] Syntax parse test on transformed output.

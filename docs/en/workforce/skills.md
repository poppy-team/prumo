# Precision Skills Catalog (189 Skills)

**Skills** in Prumo are canonical procedure modules, structured with preconditions, allowed tools, operational limits, and verifiable completion criteria.

## How Skills Work

Each skill in the catalog defines:
1. **Precise Intent**: An unambiguous purpose (e.g. AST refactoring, accessible color palette creation, lockfile audit).
2. **Required Context**: The minimum subset of files the agent must examine (Lean Progressive Context principle).
3. **Execution Constraints**: Which ACI tools may be invoked.
4. **Exit Criterion (Done Definition)**: The test or evidence that proves the operation succeeded.

## Functional Skill Groups

### 🎨 Design, UI/UX, Vectors & Visual Communication

- **Design Systems & Tokens**: `design-system-architect`, `design-tokens-specifier`, `color-palette-generator`, `typography-scale-builder`, `spacing-system-designer`, `theme-switcher-implementer`.
- **Componentization & UI**: `accessible-component-builder`, `form-control-designer`, `data-table-designer`, `modal-dialog-builder`, `toast-notification-system`, `responsive-layout-grid`.
- **Vector Art, Icons & SVG**: `svg-icon-author`, `vector-path-optimizer`, `scalable-logo-generator`, `svg-illustration-composer`, `canvas-renderer-adapter`.
- **Motion & Microinteractions**: `css-animation-choreographer`, `motion-physics-easing`, `page-transition-designer`, `skeleton-loader-animator`.
- **Accessibility & Ergonomics**: `wcag-aaa-auditor`, `screen-reader-announcer`, `focus-management-specifier`, `high-contrast-mode-checker`.
- **Marketing, Branding & Advertising**: `brand-identity-guideline-author`, `hero-section-copywriter`, `social-share-card-generator`, `conversion-rate-optimizer-designer`.

### ⚙️ Compiler, Runtime & Systems Engineering

- **Lexical Analysis & Parsing**: `lexer-tokenizer-builder`, `recursive-descent-parser`, `ast-node-definer`, `syntax-error-reporter`.
- **Semantics & Typing**: `type-inference-solver`, `scope-resolution-analyzer`, `contract-checker-static`, `borrow-checker-emulator`.
- **Code & Bytecode Generation**: `bytecode-assembler-emitter`, `virtual-machine-dispatcher`, `js-es2022-transpiler`, `wasm-module-builder`.
- **Concurrency & Time**: `async-await-state-machine`, `cooperative-scheduler-designer`, `virtual-clock-tester`, `channel-mailbox-handler`.
- **Memory & Performance**: `arena-allocator-engineer`, `memory-leak-hunter`, `cache-locality-optimizer`, `flamegraph-profiler`.

### 🏗️ Architecture, Governance & Quality

- **Governance & Prumo Protocol**: `goal-lifecycle-manager`, `sha256-integrity-locker`, `plan-dag-validator`, `evidence-collector`, `gate-evaluator`.
- **Lean Progressive Context (LPC)**: `context-capsule-compiler`, `megaprompt-reducer`, `token-budget-enforcer`, `document-relevance-scorer`.
- **Testing & Conformance**: `differential-testing-runner`, `fuzz-harness-generator`, `golden-master-comparer`, `flaky-test-eliminator`.
- **Security & Hermeticity**: `sandbox-boundary-enforcer`, `secret-leak-detector`, `dependency-cve-auditor`, `env-var-isolation-checker`.
- **Living Documentation**: `living-book-sync`, `adr-author`, `contradiction-detector`, `authority-drift-checker`.

## Invoking Skills via the CLI

Skills can be invoked directly in the harness or inspected:

```bash
# List all cataloged skills
prumo workforce skills list

# Inspect a skill's details and tools
prumo workforce skill inspect design-tokens-specifier

# Run a skill-guided task
prumo run --skill accessible-component-builder --path ./src/components
```

# Deterministic Recipes (20 Recipes)

**Recipes** in Prumo are standardized operational sequences that orchestrate multiple steps and agents in a deterministic execution pipeline with gate validation.

## Recipe Catalog

| Recipe | Description | Agents Involved |
|---|---|---|
| **`architecture-change`** | Structural or dependency change with prior ADR drafting and contract audit. | `architect`, `quality-reviewer` |
| **`brand-creation`** | Complete brand identity creation: aesthetics, SVG logo, OKLCH palette, fluid typography, and guidelines. | `creative-director`, `brand-designer`, `svg-artist` |
| **`bug-fix`** | Evidence-driven defect fix (deterministic reproduction with a failing test before the fix). | `debugger`, `implementer`, `tester` |
| **`compiler-change`** | Evolution of syntax, AST, or compilation rules with differential parity verification. | `compiler-engineer`, `tester` |
| **`compiler-conformance`** | Execution of exhaustive conformance suites and tests against canonical baselines. | `compiler-engineer`, `release-verifier` |
| **`design-system-foundation`** | Design system architecture: W3C DTCG tokens, OKLCH palettes, atomic components, and a living styleguide. | `design-system-engineer`, `ui-component-engineer`, `accessibility-reviewer` |
| **`documentation-refactor`** | Update of canonical documentation, authority audit, and drift mitigation. | `documentation-maintainer`, `reviewer` |
| **`engine-renderer`** | Changes to the graphics rendering subsystem, shaders, or main loop. | `engine-engineer`, `renderer-engineer` |
| **`feature-standard`** | Implementation of a standard feature with Goals, Plans, tests, and documentation. | `architect`, `implementer`, `tester` |
| **`github-issue`** | Full GitHub issue resolution cycle: triage, plan, implementation, and PR. | `issue-triager`, `implementer`, `reviewer` |
| **`icon-library-creation`** | Production and engineering of an icon library: canonical grid, SVG path optimization, and packaging. | `svg-artist`, `design-system-engineer`, `accessibility-reviewer` |
| **`marketing-campaign`** | Advertising campaign pipeline using the AIDA methodology: multi-format creatives and a responsive landing view. | `creative-director`, `advertising-designer`, `frontend-engineer` |
| **`multiplayer-feature`** | Development of networked mechanics with synchronization and latency handling. | `networking-engineer`, `tester` |
| **`project-bootstrap`** | Canonical repository initialization, profile configuration, and first Goals. | `architect`, `devops-engineer` |
| **`release`** | Formal verification of release prerequisites, changelog, tags, and binary generation. | `release-verifier`, `security-reviewer` |
| **`security-audit-release`** | Formal pre-release security audit with privilege checks. | `security-architect`, `security-reviewer` |
| **`security-review`** | Periodic review of attack surfaces, dependencies, and secret isolation. | `security-reviewer`, `isolation-auditor` |
| **`ui-feature`** | Building interface components with design tokens and usability review. | `ui-component-engineer`, `ux-architect` |
| **`ui-review`** | Visual audit, responsiveness testing, and accessibility conformance (WCAG). | `accessibility-reviewer`, `visual-identity-auditor` |
| **`web-feature`** | Implementation of a web service, REST/HTTP endpoints, and payload validation. | `backend-engineer`, `frontend-engineer` |

## How to Run a Recipe

```bash
# Run the bug-fix recipe for a specific case
prumo run --recipe bug-fix --param issue=142

# Run the release verification pipeline
prumo run --recipe release --param version=0.6.0
```

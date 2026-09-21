# 79.H — Testing, Code Quality, Review, Benchmarking e Evidence Strategy

> Authority: canonical specification.
> Logical ID: 79 H
> Source: Notion Living Book (3d89bb7d023f816cb230dc8ef54cbe94)
> Status: Skill Package de referência para 79 H — Testing, Code Quality, Review, Benchmarking.


<aside>
🧪

Testing não deve significar “rodar tudo”. O Prumo precisa selecionar a **menor estratégia de evidência suficiente para o risco**, preservando determinismo onde possível e usando evals apenas para comportamento probabilístico.

</aside>

# Modelo de qualidade

```
Implementation rules
→ deterministic checks/lint
→ unit/property tests
→ integration/contract
→ conformance
→ E2E/system
→ performance/security/a11y specialists
→ independent review
→ probabilistic evals where needed
```

| Skill | Pri. | Responsabilidade alvo | Aprimoramento | Aceite |
| --- | --- | --- | --- | --- |
| **testing-quality** | P0 | Orquestrador de test strategy. | Risk-based pyramid, deterministic fixtures, contract/integration/E2E/property/fuzz/perf/security selection, flaky governance e coverage semantics. | Test plan explica por que cada nível foi ou não usado. |
| **clean-code** | P0 | Princípios de implementação legível/manutenível. | Definir fronteira com code-quality/refactoring; exceptions por domínio; exemplos good/bad; checks somente para regras objetivas. | Não duplica findings de métricas/refactoring. |
| **code-quality** | P1 | Métricas e structural smells. | Complexity, coupling, duplication, dead code, maintainability trends, thresholds e false-positive policy. | Finding mensurável possui baseline/métrica. |
| **code-review** | P1 | Review de mudança. | Diff scoping, behavior/invariant delta, severity, evidence validation, specialist routing, exact commit/revision. | Approval está vinculado ao estado exato revisado. |
| **refactoring** | P1 | Mudança estrutural sem alteração comportamental. | Characterization tests, incremental transformations, scope boundary, before/after metrics, easy rollback. | Behavior preservation demonstrado. |
| **error-handling** | P1 | Taxonomia de erros de produto/código. | Domain/validation/transient/permanent, retryability, typed errors, causal chain, user-safe messages e telemetry. | Nenhum permanent error recebe retry automático. |
| **benchmarking** | P1 | Medição estatisticamente válida. | Warmup, repetitions, variance/confidence, machine normalization, baseline storage, noise threshold. | Ganho sem significância é `inconclusive`. |
| **concurrency-quality** | P1 | Race/deadlock/cancellation/backpressure. | Race, deadlock/livelock/starvation, atomics/memory ordering conforme linguagem, structured concurrency, sanitizer/stress. | Concurrency evidence apropriada ao runtime. |
| **playwright-ui** | P1 | Web E2E robusto. | Semantic selectors, deterministic fixtures, trace/video/screenshot, network mocking policy, a11y snapshots, retry/flaky quarantine. | Retry não transforma flaky em pass confiável. |

# Ownership da família Clean Code

- `clean-code`: princípio/idioma de implementação.
- `code-quality`: diagnóstico mensurável do estado do código.
- `code-review`: avaliação do changeset e evidence.
- `refactoring`: transformação controlada para melhorar estrutura.

O resolver pode compor mais de uma, mas cada uma deve produzir findings distintos.

# Flaky Test Policy

Estados: `stable`, `suspected_flaky`, `confirmed_flaky`, `quarantined`, `fixed`. Quarantine requer owner, issue, rationale e expiry. Retry count é evidence auxiliar, nunca cura.

# Property/Fuzz integration

Embora uma skill transversal própria seja candidata, testing-quality deve conhecer quando solicitar property/fuzz: parser/serialization/protocol/security/geometry/state machines e inputs combinatórios. Failure seed/corpus deve ser persistido como regression fixture.

# Coverage semantics

Line coverage isolada não é quality gate universal. Registrar quando útil: branch, mutation, requirements/acceptance coverage, API/contract matrix e risk coverage.

# Evidence completeness score

Uma test evidence só é completa quando identifica target, environment/tool version, command/action, result, artifacts/logs relevantes, limitations/skips e timestamp. Review deve validar evidence, não apenas repetir seu summary.

# Exit Gate

A estratégia de teste é derivada do risco, flaky possui lifecycle, benchmark possui rigor estatístico, famílias de code quality não duplicam responsabilidades e evidence pode ser consumida deterministicamente pelos gates.

# Amendment 2026-09-21 — Testing Beyond Pass/Fail

[84 — Total Assurance Constitution: Gauntlet Loop, Evidence e Anti-False-Green](../../quality/gauntlet-84.md) amplia esta família com test-quality review, oracle classification, mutation sensitivity, negative-space validation, fault injection, state-sequence testing, fresh-state proof e endurance/resource coverage. Todo test plan material deve explicar também **como poderia dar um falso positivo** e quais mecanismos reduzem esse risco.
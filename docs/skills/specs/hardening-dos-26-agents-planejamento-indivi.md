# 79.C — Hardening dos 26 Agents: Planejamento Individual

> Authority: canonical specification.
> Logical ID: 79 C
> Source: Notion Living Book (3d89bb7d023f810482f1db59bd5ab75a)
> Status: Skill Package de referência para 79 C — Hardening dos 26 Agents Planejamento Indivi.


<aside>
🤖

Esta página descreve **um a um** os 26 agents identificados na auditoria. O objetivo não é criar novos papéis por padrão, mas tornar os existentes mais especializados, mensuráveis e compatíveis com os contratos transversais.

</aside>

# Regras comuns para todos os agents

Todo agent deve ganhar: typed inputs/outputs, capability audit, evidence responsibility, measurable stop conditions, specialist handoff, failure taxonomy, idempotency quando muta estado e evals de papel. `Required Skills` deve ser validado contra o próprio procedimento.

| Agent | Pri. | Lacuna | Aprimoramento planejado | Aceite |
| --- | --- | --- | --- | --- |
| **accessibility-reviewer** | P0 | Responsabilidade maior que skills declaradas. | Vincular screen-reader, motion, zoom/reflow, focus, keyboard, contrast e cognitive clarity; matriz browser+AT; retest. | Finding aponta WCAG/critério, cenário, evidência e prova pós-correção. |
| **architect** | P1 | ADR/DAG fortes; NFR, evolução e rollback pouco formais. | NFR budgets, alternatives, compatibility matrix, migration/deprecation, rollback e fitness functions. | ADR médio/alto possui trade-offs e reversibilidade. |
| **backend-engineer** | P1 | Implementação sem contrato operacional completo. | Authn/authz, idempotency, limits, versioning, observability, resilience, cancellation/timeouts, migration impact. | Contract/integration tests + operational evidence. |
| **compiler-engineer** | P1 | Conformance sem arsenal completo de compiler testing. | AST/IR invariants, golden/differential/property/fuzz, diagnostics stability, ABI e perf. | Correctness mede semântica, diagnóstico, compatibility e performance. |
| **database-engineer** | P1 | Schema/query sem lifecycle de produção. | Forward/backward migration, expand-contract, locks/isolation, backup/restore, retention, PITR, EXPLAIN. | Migration crítica tem roll-forward/rollback verificado. |
| **debugger** | P1 | Root cause precisa ser reproduzível. | Hypothesis ledger, bisect, minimal repro, instrumentation, differential debugging, regression test obrigatório. | Symptom → cause → fix → regression evidence. |
| **design-researcher** | P1 | Pesquisa sem rigor/provenance suficientes. | Method, source/sample, bias, confidence, ethics/privacy, competing hypotheses, insight→decision trace. | Conclusão distingue observação, inferência e confidence. |
| **design-system-engineer** | P1 | Design system não tratado plenamente como API versionada. | Component/token semver, modes/themes, deprecation, migrations, a11y, cross-platform, visual regression. | Breaking component/token change exige migration note. |
| **devops-engineer** | P1 | Infra/CI sem supply-chain e operação completos. | IaC drift, provenance, SBOM/signing, canary, rollback, DR, SLO/alerts, runtime hardening. | Deploy produção possui provenance e recovery evidence. |
| **documentation-maintainer** | P1 | Doc correta pode ficar divergente do código. | Doc-code drift, executable snippets, links, versioning, redirects, freshness. | Changeset identifica docs afetadas e tests de docs. |
| **editor-engineer** | P1 | Estado editável precisa de garantias transacionais. | Undo/redo transactional, autosave/crash recovery, corruption-safe writes, plugin isolation, huge project behavior. | Crash/restart e undo/redo preservam consistência. |
| **engine-engineer** | P1 | Hot-loop rules sem invariantes sistêmicos. | Frame budgets, timing, allocator strategy, fragmentation, thread ownership, platform abstraction, profiling gates. | Critical change prova CPU/memory/frame budgets. |
| **explorer** | P1 | Discovery não expressa confiança/provenance claramente. | Confidence, stale-index detection, generated/vendor exclusion, trust boundaries, context escalation. | Impact map separa fato/inferência/incerteza. |
| **frontend-engineer** | P1 | UI implementation sem full web quality contract. | Browser matrix, responsive, i18n, SSR/hydration, async/errors, a11y, Web Vitals, security, visual tests. | Gates aplicáveis derivados do impact/risk. |
| **implementer** | P1 | Patch mínimo pode chegar cedo demais ao tester. | Contract preflight, dependency risk, checkpoints, local verification, evidence bundle. | Handoff só após self-test mínimo e base-state validation. |
| **issue-author** | P1 | Issue legível não é necessariamente implementation-ready. | Testable acceptance, non-goals, dependencies, test strategy, security/migration impact. | Architect planeja sem reinterpretar intenção. |
| **issue-triager** | P1 | Falta severity/SLA/confidence routing. | Priority/severity, repro confidence, duplicate confidence, blockers, security/private path e SLA. | Triagem justificável e reproduzível. |
| **networking-engineer** | P1 | Happy-path networking não é suficiente. | Loss/jitter/reorder/partition, reconnect, backpressure, NAT/firewall, protocol evolution, soak/chaos. | Critical changes passam por network simulation. |
| **quality-reviewer** | P1 | Qualidade pode ficar subjetiva. | Severity taxonomy, complexity/debt thresholds, architecture fitness, measurable maintainability. | Finding contém impacto, evidência e ação. |
| **release-verifier** | P1 | Boa matriz de release, falta fechar supply chain. | Reproducible build, signing/provenance, SBOM, upgrade/downgrade migration, staged rollout e post-release validation. | Artefato rastreável até commit/build/tests. |
| **renderer-engineer** | P1 | GPU/driver/resource lifecycle incompletos. | Capability/fallback matrix, shader validation/hot reload, GPU memory/leaks, frame budgets, golden images. | Backends/GPU classes suportados verificados. |
| **reviewer** | P1 | Review geral precisa de specialist routing. | Severity, risk-based checklist, specialist routing, states approve/block/inconclusive, exact-commit review. | Não aprova domínio crítico sem verificação requerida. |
| **security-architect** | P0 | Threat model precisa cobrir abuso/dados/residual risk. | Data classification, attacker capabilities, misuse, privacy, secure defaults, residual risk, refresh triggers. | Trust-boundary change invalida threat model relacionado. |
| **security-reviewer** | P0 | Scan automatizado e análise de exploitability não estão suficientemente separados. | SAST/DAST/SCA/secrets + abuse analysis, severity/exploitability, false-positive disposition, waiver expiry, remediation proof. | High/critical nunca waived implicitamente. |
| **tester** | P0 | Evidência existe; seleção de estratégia precisa ser risk-based. | Unit/integration/contract/E2E/property/fuzz/perf/security, deterministic seeds, flaky governance, coverage statement. | Relatório declara testado e não testado. |
| **ux-architect** | P1 | Flows e states precisam de NFR/aceite mensurável. | Responsive/i18n/a11y loops, usability hypotheses, recovery/cancel paths, UX metrics. | Fluxos principal/alternativos viram acceptance scenarios. |

# Regras de independência

- `implementer` não pode produzir o único review de sua própria mudança em risco crítico.
- `security-reviewer`, `accessibility-reviewer` e outros specialist reviewers devem ser independentes quando gate exigir.
- `reviewer` geral agrega e roteia; não substitui domínio especializado.

# Agent evals mínimos

Para cada papel: 3 casos positivos, 2 negativos (“não ativar”), 2 boundary/ambiguous, 1 failure/recovery e, para papéis críticos, adversarial cases. Métricas: role selection precision, output completeness, evidence completeness, unnecessary capabilities, handoff continuity e token/cost budget.
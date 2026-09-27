# 79.D — Hardening das 14 Recipes: DAGs, Gates Condicionais, Recovery e Aceite

> Authority: canonical specification.
> Logical ID: 79 D
> Source: Notion Living Book (3d89bb7d023f81a3be24cf3141c30cf2)
> Status: Skill Package de referência para 79 D — Hardening das 14 Recipes DAGs, Gates Condic.


<aside>
🧭

Recipes devem deixar de ser apenas sequências bem escritas e passar a funcionar como **DAGs executáveis governados por risco, evidence, resume e conditional gates**.

</aside>

# Regras comuns

Cada recipe deve validar preconditions, typed inputs, role/skill compatibility, step dependencies, mutation class, evidence producers, failure class, retryability, idempotency, resume e completion criteria.

| Recipe | Pri. | Lacuna | Aprimoramento | Exit Gate |
| --- | --- | --- | --- | --- |
| **architecture-change** | P1 | ADR/staging/compatibility bons; rollback e NFR precisam aprofundar. | Threat/NFR review, consumer matrix, expand-contract, deprecation window, rollback rehearsal, observability. | Breaking change possui migration+rollback+compatibility evidence. |
| **bug-fix** | P1 | Regression-first forte; flaky/concurrency/environment precisam branches. | Defect classification, bisect/minimal repro, flaky branch, concurrency reproduction, rollback check. | Symptom→root cause→fixture→fix rastreável. |
| **compiler-change** | P1 | 100% conformance não cobre tudo. | Conditional parser fuzz, differential semantics, diagnostics golden, ABI, benchmark/perf gate. | Conformance + risk-specific compiler evidence. |
| **documentation-refactor** | P1 | Docs tratadas mais como texto que artefato executável. | Link/snippet tests, terminology consistency, redirects, preview build, code-doc drift, retrieval eval. | Sem broken links/snippets/retrieval regressions. |
| **engine-renderer** | P1 | GPU/platform/perf matrix insuficiente. | Golden render, GPU validation, leak/resource tests, CPU/GPU budgets, capability fallback matrix. | Correctness visual + perf baseline. |
| **feature-standard** | P0 | Default workflow usa tests/review, mas specialist gates não são derivados dinamicamente. | Risk-driven gate planner adiciona security/a11y/API/DB/perf/migration/docs etc. | High-risk feature nunca fecha só com testes genéricos. |
| **github-issue** | P1 | Issue workflow precisa readiness completo. | Duplicate/repro search, acceptance, non-goals, dependencies, security sensitivity, validation proposal. | Readiness score mínimo antes de implementation planning. |
| **multiplayer-feature** | P1 | Feature comum não cobre adversarial network/game authority. | Protocol/version, determinism, authority, reconciliation, cheat/abuse, latency/loss/jitter, soak. | Passa adverse network matrix. |
| **project-bootstrap** | P1 | Hardening pode ficar para depois do bootstrap. | Repo policies, threat baseline, CI, lint/test, supply chain, docs, observability, reproducible dev env. | Novo projeto passa doctor/hardening baseline. |
| **release** | P1 | Build/test/CVE/checksum/rollback existentes; supply-chain lifecycle ainda incompleto. | Signing/attestation/SBOM, reproducible builds, migrations, staged rollout, post-release smoke. | Release reconstruível e autenticável. |
| **security-review** | P1 | Threat→audit→remediate→verify forte; lifecycle de finding precisa completar. | Attack-surface delta, exploitability, false-positive disposition, waiver owner/expiry, regression corpus. | Todo finding tem closure ou accepted-risk explícito. |
| **ui-feature** | P1 | Estados/modalidades podem ficar incompletos. | Responsive, keyboard, SR, motion, zoom, visual regression, i18n, loading/error/empty/offline e perf. | State/input matrix completa. |
| **ui-review** | P1 | Precisa agregar specialists sem duplicar knowledge. | Compose UI/UX review + a11y subskills + visual QA/regression + browser/responsive matrix. | Scorecard único com owner/evidence por finding. |
| **web-feature** | P1 | SPA simples e full-stack recebem riscos diferentes. | Impact-driven DAG para API, web security, a11y, performance/CWV, telemetry, SEO e rollback. | Workflow condicionado à arquitetura real. |

# Conditional Gate Planner

O planner deve receber risk/impact facts, não apenas palavras do prompt. Exemplos:

- altera auth/permission → security architect + review gate;
- altera UI interativa → a11y + visual/interaction gates;
- altera public API → contract + compatibility gate;
- altera schema persistente → migration/recovery gate;
- altera hot loop/render/network → performance/simulation gate;
- cria release artifact → supply-chain/release gate.

# Recipe simulation

Antes de executar uma recipe alterada, permitir um dry-run que emita:

- DAG resolvido;
- agents/skills escolhidos;
- permissions;
- conditional gates;
- expected evidence;
- side effects;
- retry/resume behavior;
- estimated context/workforce size.

# Recipe evals

Avaliar: recipe selection accuracy, unnecessary steps, missing specialist gates, DAG validity, recovery correctness, evidence completeness, final acceptance coverage e custo/contexto. Casos negativos devem demonstrar que a recipe **não** é selecionada quando outra é mais específica.
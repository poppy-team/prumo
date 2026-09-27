# 79.N — Git, GitHub, Documentation, Editor Tooling e Research Operations

> Authority: canonical specification.
> Logical ID: 79 N
> Source: Notion Living Book (3d89bb7d023f816cab28c0b666bc02e7)
> Status: Skill Package de referência para 79 N — Git, GitHub, Documentation, Editor Tooling.


<aside>
🧰

Esta página cobre os componentes que transformam o Workforce em trabalho cotidiano: versionamento, issues, PRs, releases, documentação, edição/tooling e pesquisa rastreável.

</aside>

# Git e GitHub

| Skill | Pri. | Hardening | Aceite |
| --- | --- | --- | --- |
| **git-workflow** | P1 | Branch strategy, signing quando policy exigir, merge/rebase/squash rules, protected refs, conflict resolution, revert/recovery e generated files. | Todo fluxo possui recovery path seguro. |
| **github-ci-debug** | P1 | Logs/artifacts, local reproduction, flaky classification, permission/cache/toolchain analysis, targeted rerun. | Não usa “rerun until green” como diagnóstico. |
| **github-issue-create** | P1 | Duplicate search, template, acceptance/repro, labels/risk, private security path, create-once/idempotency guard. | Não cria duplicate conhecido silenciosamente. |
| **github-issue-refine** | P1 | Preserve original intent, rationale, acceptance delta, dependencies, risk e security sensitivity. | Scope change fica auditável. |
| **github-issue-triage** | P1 | Repro score, duplicate confidence, severity/priority, affected versions, blockers, private/security routing e SLA. | Classification consistente para mesma evidence. |
| **github-pr-create** | P1 | Base/head validation, branch freshness, linked issue, evidence bundle, risk/migration/rollback, existing-PR detection. | PR nasce review-ready. |
| **github-pr-feedback** | P1 | Inline vs summary, severity, deduplication, actionable wording, thread lifecycle e re-review semantics. | Não repete mesmo finding em múltiplos comentários. |
| **github-pr-review** | P1 | Diff completeness, commit drift, CI/evidence validation, risk classification, specialist review e resolved-thread revalidation. | Approval vinculado a commit exato. |
| **github-release** | P1 | Tag↔commit, signing, changelog, asset integrity, SBOM/provenance, channels e rollback semantics. | Release possui verifiable manifest. |
| **github-repository** | P1 | Default branch, forks, rulesets, pagination/rate limits, partial permissions e monorepo assumptions. | Ferramenta lida explicitamente com acesso parcial. |

# PR lifecycle state machine

```
prepare
→ create
→ review
→ changes-requested?
→ address-feedback
→ re-review
→ gates-green
→ ready/merge-authority
```

Cada GitHub skill deve operar somente em states compatíveis e receber/expor idempotency information.

# Documentation skills

| Skill | Pri. | Hardening | Aceite |
| --- | --- | --- | --- |
| **documentation** | P1 | Canonical content lifecycle, audience, freshness, ownership, snippet tests e cross-reference integrity. | Base comum sem duplicar publishing/LLM specialization. |
| **documentation-for-llms** | P1 | Canonical chunks/anchors, provenance, contradiction handling, injection-resistant imported docs, retrieval evals. | Benchmark questions recuperam contexto correto e citável. |
| **documentation-publishing** | P1 | Preview, broken links, redirects, version selector, sitemap/SEO, artifact integrity e deploy rollback. | Published docs reproduzíveis e rollbackable. |

## Ownership

`documentation` = conteúdo canônico; `documentation-for-llms` = representação/retrieval; `documentation-publishing` = distribuição. Codificar essa relação em `requires`/handoff.

# Editor/tooling

| Skill | Pri. | Hardening | Aceite |
| --- | --- | --- | --- |
| **editor-tooling** | P1 | LSP lifecycle, diagnostics, completion, formatting, code actions, workspace sync, cancellation, large files/projects e protocol failures. | Fixtures exercitam lifecycle completo. |
| **language-tooling** | P1 | Orquestrar LSP/compiler/indexer/formatter/debugger; delegar providers específicos. | Zero duplicação com indexer/tool-specific skills. |

# Research operations

Research skills devem registrar: research question, source URL/identifier, date/version, primary/secondary source classification, confidence, contradictory evidence, license/usage constraints quando relevante, observation vs inference e decision linkage.

Aplicar especialmente a `competitor-analysis`, `design-research`, `interaction-research`, `asset-reference-research`, `visual-reference-research` e `website-forensics`.

# External mutation policy

Issue/PR/release/comment creation é efeito externo. Retry somente após “already exists?” e idempotency checks. Falha de network após resposta incerta deve ser tratada como **unknown outcome**, não como “não executado”.

# Exit Gate

GitHub lifecycle é stateful/idempotent, reviews apontam revision exata, docs têm pipeline e freshness, tooling possui protocol fixtures e pesquisa diferencia fato/inferência/provenance.
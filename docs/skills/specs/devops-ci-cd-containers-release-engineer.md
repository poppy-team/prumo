# 79.M — DevOps, CI/CD, Containers, Release Engineering e Observability

> Authority: canonical specification.
> Logical ID: 79 M
> Source: Notion Living Book (3d89bb7d023f8123ab90f6202042d65b)
> Status: Skill Package de referência para 79 M — DevOps, CI CD, Containers, Release Engineer.


<aside>
🚀

Esta família fecha o caminho **source → build → test → artifact → deploy/release → observe → recover**. O hardening deve garantir reproducibilidade, least privilege, provenance e rollback real, não apenas automação conveniente.

</aside>

| Skill | Pri. | Aprimoramento | Aceite |
| --- | --- | --- | --- |
| **ci-cd** | P0 | Hermetic/reproducible builds, pinned actions/tools, cache integrity, target matrix, least privilege, OIDC, required gates, artifact retention e provenance. | CI não depende de mutable/unpinned inputs críticos e produz evidence normalizada. |
| **containers** | P0 | Pinned base digests, multi-stage, rootless, read-only filesystem, dropped capabilities, seccomp/sandbox, secret mounts, minimal image, SBOM/scanning. | Policy bloqueia privileged/insecure defaults. |
| **release-engineering** | P0 | SemVer/changelog, reproducible builds, signing/attestation, SBOM, migrations, staged rollout, rollback/hotfix, post-release validation. | Release manifest source→build→tests→artifact. |
| **observability** | P0 | Structured logs, metrics, traces, correlation, sampling, cardinality budgets, PII redaction, SLOs/alerts quando justificáveis e linkage com incidents. | Operational change possui observability acceptance. |
| **performance-native** | P1 | Integrar performance evidence aos gates de build/release quando houver budget crítico. | Regression threshold bloqueia release quando policy exigir. |
| **performance-web** | P1 | Integrar CWV/bundle/network budgets ao CI/release web. | Baseline comparável por target/device profile. |

# CI pipeline contract

Um pipeline deve declarar:

- source revision;
- toolchain versions/digests;
- permissions/tokens;
- network policy quando aplicável;
- caches e key strategy;
- build/test matrix;
- evidence outputs;
- artifact retention;
- provenance/signing stage;
- mutation/deploy stage separado de validation sempre que possível.

# Supply-chain integration

`ci-cd`, `containers` e `release-engineering` consomem `supply-chain-security`; não devem duplicar seu threat knowledge. Release critical gate deve verificar lock/pin, dependency scan policy, SBOM/provenance e signing/attestation conforme plataforma suportada.

# Deployment strategies

Suportar por profile, não universalmente: rolling, blue/green, canary, staged desktop updater, package channel/prerelease. Cada strategy deve declarar rollback feasibility e health verification.

# Rollback não é texto

Rollback evidence pode exigir: previous artifact retained, migration compatibility, restore test, feature flag/traffic switch, version downgrade support, data recovery plan. Se rollback não for possível, o risk class deve aumentar.

# Observability quality rules

- logs estruturados e semanticamente estáveis;
- correlation IDs propagados em boundaries relevantes;
- evitar high-cardinality labels perigosas;
- secrets/PII redacted antes da emissão;
- metrics escolhidas por decision usefulness;
- traces sampled com policy explícita;
- alerts ligados a ação, não ruído.

# Release artifact manifest

Campos: revision, build environment, toolchain, target, dependencies/SBOM, checksums, signatures/attestations, test/evidence IDs, migration notes, known limitations, rollback target e publication channel.

# Exit Gate

Critical CI/release paths são reproducíveis e least-privilege; artifacts possuem provenance; rollback/recovery é verificável; observability produz sinais úteis sem vazar dados sensíveis.

# Amendment 2026-09-21 — Performance/Release Assurance

[84 — Total Assurance Constitution: Gauntlet Loop, Evidence e Anti-False-Green](../../quality/gauntlet-84.md) e 84.C/84.G ampliam release engineering: validar artefato distribuído em estado limpo; comparar source-tested revision com package; incluir resource/endurance budgets, leak/regression checks e post-release observation. Rollback precisa ser exercitado quando risco justificar, não apenas documentado.
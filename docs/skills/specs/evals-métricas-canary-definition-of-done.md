# 79.R — Evals, Métricas, Canary, Definition of Done e Exit Gates do Workforce

> Authority: canonical specification.
> Logical ID: 79 R
> Source: Notion Living Book (3d89bb7d023f811abb89c3a8262d2d86)
> Status: Skill Package de referência para 79 R — Evals, Métricas, Canary, Definition of Done.


<aside>
✅

A fase final não pergunta “reescrevemos tudo?”. Ela pergunta: **o novo workforce seleciona melhor, produz evidência melhor, custa menos contexto desnecessário e falha de modo mais seguro que o baseline?**

</aside>

# 1. Princípio de avaliação

Usar testes determinísticos para contratos/invariantes e LLM evals apenas onde comportamento é probabilístico: activation, planning, review quality, context selection, classification e output quality subjetiva/semiestruturada.

# 2. Corpus padrão

Cada skill/agent/recipe crítica deve possuir:

- **positive cases:** deve ativar/executar;
- **negative cases:** não deve ativar;
- **near-neighbor cases:** outra skill/recipe é owner melhor;
- **ambiguous cases:** requer confidence/routing apropriado;
- **edge/boundary cases:** limitações do domínio;
- **failure/recovery cases:** tool/provider/validation failure;
- **adversarial cases:** security, untrusted input, misleading project content, malformed data conforme domínio.

# 3. Metrics — Activation

| Métrica | O que mede | Falha típica detectada |
| --- | --- | --- |
| Precision | Selected components realmente úteis. | Over-activation/context bloat. |
| Recall | Required specialists encontrados. | Missing security/a11y/perf gate. |
| Near-neighbor confusion | Owner correto entre skills parecidas. | clean-code vs code-quality; visual QA vs regression. |
| Workforce size | Número de roles/skills por task. | Violação de Least Workforce. |
| Context overhead | Tokens adicionados por component. | Knowledge carregado cedo demais. |

# 4. Metrics — Output/Evidence

- schema validity rate;
- acceptance-criteria coverage;
- evidence completeness;
- unsupported-claim rate;
- finding actionability;
- severity calibration;
- handoff completeness;
- specialist coverage;
- false approval / false block rate em corpora rotulados.

# 5. Metrics — Reliability

- step success/failure by class;
- deterministic failure retried incorrectly;
- retry success rate para transient failures;
- duplicate external mutation rate;
- resume success without transcript;
- merge/conflict incidents;
- stale-context incidents;
- permission-denied frequency e unnecessary elevation rate.

# 6. Metrics — Quality by domain

## Security

Threat coverage, exploitable finding recall, false positives, unresolved high/critical, secret leakage, unsafe permission escalation.

## Accessibility

Criterion coverage, keyboard path completion, focus errors, AT announcement correctness, contrast/zoom/motion test pass rates.

## Compiler/native

Conformance, fuzz discoveries, differential mismatches, sanitizer findings, ABI regressions, compile/runtime performance.

## UI/visual

State coverage, visual diff false positives, accidental baseline acceptance, browser/device coverage, interaction recovery coverage.

## Game/network

Deterministic replay success, physics tolerance failures, network adversity success, frame/memory/audio budgets, asset reproducibility.

# 7. Threshold policy

Não congelar números universais nesta documentação sem baseline real. Thresholds devem ser aprovados após WP0/WP10. Porém algumas invariantes são binárias desde o início:

- reference/schema validation: 100% para active stable packages;
- unresolved P0 relationship errors: 0;
- unregistered critical permission elevation: 0;
- silent external duplicate mutation: 0;
- implicit high/critical security waiver: 0;
- missing required evidence accepted as pass: 0.

# 8. Eval comparison protocol

Para rewrite importante:

```
same corpus
→ baseline without skill / current skill
→ candidate skill
→ same model roster/policy where possible
→ repeated runs for probabilistic metrics
→ normalized metrics
→ human/sample review for disputed labels
→ promotion decision
```

Registrar model/provider/version, prompt/package version, context budget e seed/settings quando disponíveis.

# 9. Canary

Antes de promover mudança ampla de resolver/activation/context:

- executar em subset de Goals/repositories;
- comparar selection, cost, failures e evidence;
- manter rollback para resolver/package version anterior;
- não promover se regressão crítica aparecer mesmo com ganho médio positivo.

# 10. Dogfooding

Prumo deve usar o próprio workforce hardenizado para revisar futuras mudanças no workforce. Dogfood não substitui corpus; serve para encontrar scenarios reais que podem virar fixtures após sanitização/generalização.

# 11. Definition of Done — Skill stable

Uma skill `stable` exige:

- manifest/contract v3 válido;
- purpose e ownership claros;
- positive e negative activation rules;
- inputs/outputs tipados quando estruturados;
- requires/conflicts/permissions/context budget;
- procedure domain-specific;
- checks/scripts para invariantes determinísticas;
- good/bad/edge examples quando útil;
- tests/evals proporcionais ao risco;
- provenance/version/freshness;
- evidence contract;
- migration/deprecation metadata quando aplicável;
- thresholds mínimos atingidos.

# 12. Definition of Done — Agent stable

- role contract válido;
- required/optional skills coerentes com procedure;
- least-privilege capability profile;
- typed input/output;
- measurable stop conditions;
- failure/retry/resume semantics;
- independent-review constraints quando aplicáveis;
- handoff artifact completo;
- positive/negative/ambiguous evals;
- sem self-approval proibida.

# 13. Definition of Done — Recipe stable

- preconditions e typed inputs;
- DAG acíclico;
- role/skill compatibility;
- conditional gates;
- side-effect/irreversibility classification;
- evidence producers;
- retry/idempotency/resume/compensation;
- dry-run/simulation;
- recipe selection negative cases;
- completion criteria mapeados a Goal acceptance.

# 14. Definition of Done — Família/domain

Uma família só fecha quando ownership entre suas skills é explícito, near-neighbor confusion está abaixo do threshold, nenhum componente stable permanece boilerplate e o resolver compõe a família sem carregar o conjunto inteiro desnecessariamente.

# 15. Program Exit Gate

O Programa 79 só fecha quando:

1. WP0–WP13 aplicáveis concluídos;
2. nenhum finding P0 permanece aberto;
3. second audit mostra redução mensurável de drift/boilerplate/orphans/excess permissions;
4. activation/evidence/handoff metrics atingem baselines aprovados;
5. critical skills têm adversarial coverage quando relevante;
6. language skills cumprem contrato mínimo comum;
7. candidate capabilities foram decididas explicitamente, mesmo que DEFER/REJECT;
8. maturity/deprecation state está consistente;
9. documentação e repositório concordam;
10. `workforce` pode ser explicado por relações e métricas, não por contagem de arquivos.

# 16. Relatório final

Produzir um documento `Workforce Hardening — Before/After` contendo baseline commit, final commit, principais mudanças, metrics delta, accepted residual debt, deferred candidates, regressions conhecidas e próxima review/freshness date.

# Amendment 2026-09-21 — 10/10 sem compensação

[84 — Total Assurance Constitution: Gauntlet Loop, Evidence e Anti-False-Green](../../quality/gauntlet-84.md) formaliza que 10/10 é um estado de contrato dentro do scope: failed required = 0, untested required = 0, unknown required = 0, stale required evidence = 0 e known unwaived defect = 0. Scores médios continuam úteis para comparação, mas não promovem maturidade contra hard gate falho. Acrescentar false-green rate, stale-evidence rate, proof-of-use coverage e required-surface verified coverage às métricas.
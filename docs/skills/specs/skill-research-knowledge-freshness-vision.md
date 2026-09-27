# 79.U — Skill Research, Knowledge Freshness, Vision e Evidence Operations

> Authority: canonical specification.
> Logical ID: 79 U
> Source: Notion Living Book (3dd9bb7d023f815984c2debfd46acea1)
> Status: Skill Package de referência para 79 U — Skill Research, Knowledge Freshness, Vision.


<aside>
🔬

**Objetivo:** transformar pesquisa na internet e documentação upstream em parte formal do lifecycle das skills. Conhecimento version-sensitive não pode permanecer indefinidamente dentro de um `SKILL.md` sem provenance, freshness e mecanismo de revisão.

</aside>

# Problema observado

O catálogo atual mistura regras estáveis, opiniões de estilo e fatos que mudam com versões. Isso já produz drift: Lua mudou para 5.5; Odin possui test-runner com memory tracking integrado; C3/D têm detalhes de contratos/escape analysis que exigem precisão de versão. UI frameworks evoluem ainda mais rápido. O Prumo precisa distinguir **knowledge stable** de **knowledge volatile**.

# Knowledge Unit Contract

Cada unidade version-sensitive deve poder declarar:

```yaml
id: egui-accessibility
source_kind: official-docs
source_urls: [...]
upstream_project: emilk/egui
applies_to: ">=0.36,<0.37"
last_verified: 2026-09-16
review_after_days: 45
confidence: high
normative_status: upstream-guidance
supersedes: null
```

Campos adicionais: platforms, feature flags, known exceptions, source excerpts/anchors permitidos, checksum/version/release tag quando útil, owner e linked evals.

# Source hierarchy

1. specification/standard oficial;
2. documentação oficial versionada;
3. source/release notes/changelog upstream;
4. maintainers/design docs/issues oficiais;
5. high-quality secondary engineering references;
6. community evidence para edge cases, sempre rotulada como tal.

Blogs aleatórios não podem sobrescrever spec/upstream sem evidence melhor.

# Research workflow para criar/aprimorar uma skill

1. definir capability, owner e negative triggers;
2. auditar skill/agent/recipe existentes para evitar duplicação;
3. listar perguntas técnicas que a skill precisa responder;
4. buscar upstream/specs atuais e versões suportadas;
5. separar invariant knowledge de version-sensitive knowledge;
6. coletar pitfalls, failure modes e migration deltas;
7. extrair procedimentos verificáveis;
8. criar scripts/checks somente quando determinísticos;
9. criar positive/negative/adversarial fixtures;
10. comparar baseline sem skill vs skill antiga vs skill nova;
11. registrar provenance/freshness;
12. promover maturity apenas após eval.

# Web Research Contract

Skills autorizadas a pesquisar devem declarar `network` permission e uma **search policy**. A pesquisa deve priorizar docs oficiais, conferir data/versão, distinguir release estável de draft, guardar links e registrar conflito entre fontes. Quando dados atuais importarem, a skill nunca deve confiar apenas em memória do modelo.

# Vision-Aware Research Contract

Para design/UI/visual QA, `vision` é uma capability opcional e explícita.

## Quando vision existe

- pesquisar/reunir múltiplas referências relevantes;
- capturar screenshot/frame com source/date/product/platform;
- decompor hierarchy, grid, spacing, density, typography, colors, component anatomy, states e affordances;
- distinguir observação pixel/visual de inferência comportamental;
- comparar referências entre si antes de propor padrão;
- registrar padrões a abstrair, anti-patterns e elementos que não devem ser copiados;
- produzir annotated reference board e confidence.

## Quando vision não existe

- pesquisar documentação/texto e referências de implementação;
- registrar screenshots como fontes não inspecionadas se não houver capacidade de visão;
- jamais declarar pixel/fidelity findings inexistentes.

# UI Reference Pack Schema

```
reference-pack/
├── index.md
├── sources.json
├── screenshots/
├── annotations/
├── patterns/
│   ├── layout.md
│   ├── components.md
│   ├── interaction.md
│   └── accessibility.md
├── decisions.md
└── provenance.json
```

O pack deve indicar licença/origem e usar screenshots para análise/referência interna conforme política aplicável, sem redistribuir assets proprietários como parte do produto.

# Scripts: deterministic, narrow, fail-correctly

Scripts de skill devem provar algo relacionado à capability. Um `wireframe-styleguide` não deve “verificar” apenas `go vet`/`cargo check`. Regras:

- exit code semântico;
- nada de `|| true` em invariant gates;
- machine-readable output quando possível;
- tool availability distinguida de pass/fail;
- fixture própria para o script;
- versão/toolchain registrada;
- timeout e resource bounds;
- zero network mutation por default.

# Skill Evaluator

Criar/fortalecer um `skill-evaluator` capaz de medir:

- activation precision/recall;
- task correctness;
- domain coverage;
- false-confidence rate;
- evidence completeness;
- stale-source detection;
- context/token cost;
- latency/tool-call cost;
- conflict/overlap com outras skills;
- robustness em adversarial/ambiguous cases.

# Skill Curator / Knowledge Maintainer

O curator deve detectar overlap, dead references, versão obsoleta, duplicated instructions, generic verify scripts, knowledge sem provenance, low-use skills e capability gaps. Um `knowledge-maintainer` pode pesquisar releases/docs e **propor** patches; não deve alterar regras críticas silenciosamente.

# Agent contract refinado

Agents não acumulam encyclopedic framework knowledge. Eles declaram role, authority, permissions, inputs/outputs, mandatory/optional capability bundles, escalation, handoff e independent-verification rules. Technology/language/security knowledge entra via resolver.

## Agents a aprofundar

- `design-researcher`: web + vision research, provenance, reference packs;
- `design-system-engineer`: tokens/components/governance/migrations;
- `accessibility-reviewer`: platform profiles + evidence matrix;
- `security-reviewer`: independent finding/retest/waiver workflow;
- `compiler-engineer`: language/toolchain packs e differential/fuzz/evidence;
- `skill-curator`: catálogo/freshness/conflicts/evals;
- `documentation-maintainer`: links/version drift e doc-contract deltas.

Criar novo agent somente quando authority/handoff/workflow forem realmente distintos.

# Freshness automation

Propor `prumo skill freshness`:

```
prumo skill freshness [id]
prumo skill sources verify [id]
prumo skill eval [id] --freshness
prumo skills stale --severity high
```

O processo compara release/version metadata, broken links, last_verified, known support windows e linked evals. A saída gera proposal/issue, nunca atualização cega de conteúdo normativo.

# Portable Agent Skills compatibility

O formato Prumo pode exportar uma visão compatível com o padrão comum de `SKILL.md` para harnesses que suportam Agent Skills, mantendo internamente manifest/evidence/evals mais ricos. Exportação deve preservar progressive disclosure e nunca conceder `shell/network` implicitamente.

# Referências iniciais

- [Anthropic — Agent Skills / progressive disclosure](https://www.anthropic.com/engineering/equipping-agents-for-the-real-world-with-agent-skills)
- [GitHub Copilot — Agent Skills](https://docs.github.com/en/copilot/how-tos/copilot-on-github/customize-copilot/customize-cloud-agent/add-skills)
- [OpenAI — Using skills](https://openai.com/academy/skills/)
- [W3C WCAG 2.2](https://www.w3.org/TR/WCAG22/)
- [DTCG 2025.10](https://www.designtokens.org/tr/2025.10/)
- [NIST SSDF](https://csrc.nist.gov/pubs/sp/800/218/final)
- [SLSA 1.2](https://slsa.dev/spec/v1.2/)

# Exit Gate

Toda skill `recommended/verified` possui provenance suficiente, source freshness adequada ao ritmo do upstream, evidence real ligada à capability, eval que demonstra ganho e um context budget que preserve progressive disclosure.
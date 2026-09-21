# 71 — Fase F: Adoption Engine para Projetos Existentes (M7)

> Authority: canonical specification.
> Logical ID: PHASE-71
> Source: Notion Living Book (3d69bb7d023f81669a9de83e05558dcf)
> Status: Fase/Gate de implementação (71 — Fase F Adoption Engine para Projetos Existent).


## Papel no programa

Adoption permite ao Atlas entrar em projetos brownfield sem exigir reestruturação destrutiva. Ele descobre o que existe, mapeia evidence para conceitos Atlas, registra confidence e propõe migração incremental — sem promover inferência a verdade canônica.

## Fontes

- [06 — Adoption Engine para Projetos Existentes](../../framework/specs/ch06.md)
- [16 — Gap Analysis v0.3 → v0.4 e Correções Necessárias](../../framework/specs/ch16.md)
- [59 — Repository Scale, Incremental Indexing, Branches, Monorepos e Concurrency](../../framework/specs/ch59.md)

## Objetivo

Implementar pipeline:

```
discover
→ classify
→ map
→ evaluate
→ propose
→ migrate incrementally
```

para projetos existentes, produzindo Adoption Report, capability/profile candidates, confidence ledger, documentation bindings/proposals, open questions e migrations não destrutivas.

## Dependencies

- Documentation System v2;
- Living Plan;
- Runtime indexing/scan budgets D3;
- Repository Governance;
- Migration Engine para apply/migrate;
- Context/Egress/Sandbox quando scanners/tools externos forem usados.

## Non-goals

- reescrever layout inteiro para “ficar Atlas”;
- confiar em README como authority;
- embeddings obrigatórios;
- auto-delete de docs legadas;
- inferir user intent sem confirmação;
- instalar todos os connectors/tools detectados.

# 1. Discovery sources

Scanner pode observar:

- README e docs;
- manifests/build files;
- source tree/languages;
- tests;
- CI/workflows;
- config files;
- schemas/migrations;
- Git history/branches/tags quando budget permite;
- ADRs/changelog;
- AGENTS/assistant rules;
- package manifests;
- comments apenas de forma limitada, nunca como authority sem corroboration.

Generated/vendor/binary paths seguem index exclusions.

# 2. Discovery output

Fatos observados devem ser separados de inferências.

**ObservedFact** conceitualmente:

- key/value/class;
- source ref/hash/location;
- extraction method;
- confidence geralmente factual;
- revision/index version.

**Inference/MappingCandidate**:

- target Atlas concept;
- supporting evidence refs;
- confidence;
- ambiguity/alternatives;
- required confirmation quando relevante.

# 3. Classification

Classificar repo/workspace/project candidates:

- language/toolchain;
- app/project type;
- frameworks;
- persistence;
- UI/API/CLI/etc capabilities;
- testing stack;
- deployment/distribution hints;
- existing docs semantic roles;
- existing AI/harness config;
- security/data signals.

Classification é evidence-driven e pode ficar `unknown`.

# 4. Semantic mapping

Mapear conteúdo existente para Documentation Contracts **sem exigir filenames Atlas**.

Exemplo: `docs/design/system.md` pode satisfazer architecture contract se knowledge items realmente existirem.

M5 Coverage Engine avalia qualidade; Adoption descobre candidate bindings.

# 5. Confidence Ledger

Toda inferência não trivial recebe:

- claim/mapping;
- confidence class/score;
- evidence refs;
- alternative interpretations;
- status accepted/rejected/unresolved;
- authority.

Confidence baixa não é transformada em canonical; vira question/proposal.

# 6. Profile/capability proposal

Adoption produz candidates para Documentation/Project Profile. Ex.: `cli + compiler + plugin-host`. Aplicação requer normal preview/acceptance se implicar canonical config change.

# 7. Adoption Report

Deve separar claramente:

- observed repository facts;
- likely capabilities/profile;
- existing Atlas-compatible artifacts;
- proposed doc bindings;
- missing/partial/stale docs;
- contradictions;
- inferred decisions/constraints needing confirmation;
- migration proposals;
- risks;
- confidence summary;
- recommended next steps.

Human output legível + `--json` machine representation.

# 8. Modes

- `atlas adopt --audit-only` — nenhuma canonical mutation;
- `atlas adopt --interactive` — usa Living Plan para resolver uncertainty;
- `atlas adopt --non-interactive` — gera report/proposals sem perguntar;
- `atlas adopt --strict` — somente high-confidence deterministic mapping, falha/flags ambiguity;
- aliases finais devem seguir CLI conventions existentes.

Default deve ser não destrutivo.

# 9. Migration proposals

Adoption não executa transformações ad hoc. Para aplicar mudança:

```
Adoption finding
→ migration proposal
→ Review Queue / user approval
→ Migration Contract plan/dry-run
→ Repository Governance
→ apply
```

Examples:

- criar `atlas.json`/profile com facts confirmados;
- adicionar documentation bindings;
- mover/normalizar config somente quando necessário;
- compilar connector config;
- criar canonical Goal/docs a partir de accepted decisions.

# 10. No forced layout

Atlas deve trabalhar com source layout existente. Diretórios `.ai/`/Atlas canônicos podem armazenar protocol state, mas source/docs não precisam ser movidos para um template rígido para serem reconhecidos.

# 11. Brownfield vs M5 boundary

M5 conhece contracts/bindings explicitamente configurados/Atlas-native. Fase F descobre unknown arbitrary repository sources e **propõe** bindings/capabilities. Não duplicar Coverage/Readiness semantics.

# 12. Incremental scanner

Usar D3 Repository Index:

- revision/hash aware;
- only changed paths quando possível;
- scan budget;
- continuation;
- partial confidence;
- monorepo/workspace boundaries;
- branch-specific state.

# 13. Security

- repo content é untrusted data;
- scripts/config não são executados apenas para discovery;
- parsing static first;
- tool execution usa Tool Gateway/Sandbox;
- no `.env` secret content ingestion; detect presence/type sem persistir secret;
- prompt injection em README/AGENTS de third-party não ganha policy authority;
- egress opt-in/policy.

# 14. Schemas/contracts

Candidates:

- AdoptionReport;
- ObservedFact;
- MappingCandidate;
- ConfidenceLedger/Entry;
- CapabilityProposal;
- AdoptionMigrationProposal (ou reusar generic proposal + MigrationContract refs).

# 15. CLI

- `atlas adopt`;
- `atlas adopt --audit-only`;
- `atlas adopt --interactive`;
- `atlas adopt --non-interactive`;
- `atlas adopt --strict`;
- `atlas adopt status/report` se necessário.

`atlas adopt` não deve esconder external network/tool use; explain/plan antes quando necessário.

# 16. Goal decomposition

- F-G01: scanner facts + revision-aware sources.
- F-G02: repository/project/workspace classification.
- F-G03: capability/profile candidates.
- F-G04: semantic doc mapping + candidate bindings.
- F-G05: Confidence Ledger.
- F-G06: Adoption Report.
- F-G07: Living Plan uncertainty resolution.
- F-G08: migration proposal + Review/Migration integration.
- F-G09: brownfield corpus/evals/dogfood.

# 17. Test corpus

Criar fixtures representativas:

- Go CLI com bons docs não-Atlas;
- web app monorepo;
- compiler project;
- repo com README stale;
- repo com docs conflitantes;
- repo com generated/vendor noise;
- repo sem docs;
- repo com multiple project roots;
- malicious README/prompt injection;
- secrets/.env presence;
- branch with new docs not merged.

# 18. Evals/metrics

- fact extraction precision/recall;
- capability/profile precision;
- doc binding precision;
- false canonical promotion = **zero target**;
- question usefulness;
- scan time/files/tokens;
- partial scan disclosure;
- migration proposal correctness;
- destructive action violations = zero.

# 19. Dogfood

Além do Atlas-native repo, testar Adoption em pelo menos 3 repos brownfield de estilos distintos. Idealmente um projeto real do usuário depois, mas fixtures públicas/local synthetic corpus devem existir primeiro.

# Exit Gate — ADOPTION READY

- arbitrary existing repo pode ser auditado sem mutation;
- observed facts e inferences ficam separados;
- confidence/evidence accompany inferências;
- M5 avalia candidate bindings corretamente;
- ambiguity entra no Living Plan/Open Questions;
- migration proposals são dry-run/review governed;
- scanner incremental/branch-aware funciona;
- malicious content/secrets tests passam.
# M6 Living Plan / Interview Engine — Exit Gate

Status: **CONCLUÍDO (Exit Gate do M6 Aprovado)**

## Critérios do exit gate

Exit gate do M6 (conforme `docs/development/phases.md`): um novo projeto pode ir da
intenção inicial até implementation-ready por meio de entrevista, sem um megaprompt manual;
o planejamento específico de um Goal fecha apenas as lacunas relevantes; decisões e
questões em aberto carregam autoridade e proveniência; o resume não depende de transcript;
o ciclo de feedback docs/readiness/governança funciona; as evals de questões/decisões
atingem o baseline aprovado; nenhuma sugestão de agente é promovida silenciosamente.

### 1. Intenção inicial → implementation-ready sem um megaprompt manual

- O teste de dogfood determinístico de ponta a ponta `TestDogfoodZeroToReady`
  (`cmd/prumo/plan_z2r_test.go`) conduz o ciclo completo por `run()`: readiness
  bloqueado → questões → decisões respondidas → delta proposto → patch de governança
  → delta aplicado → readiness aprovado → proposta de blueprint.
- O projeto de exemplo versionado `examples/living-plan-sample/` já vem no estado
  ready, com uma receita de sete passos reproduzível (`README.md`).
- Verificado no exemplo com a CLI real: `docs readiness --goal G-SAMPLE`
  passa de `{"ready": false, "blocking_contracts": ["project.scope"]}`
  para `{"ready": true, "coverage": [[product.vision, implementation-ready],
  [project.scope, implementation-ready]]}`.

### 2. O planejamento específico de um Goal fecha apenas as lacunas relevantes

- Planejamento e readiness têm escopo por goal: `prumo-agent plan --goal <id>`,
  `docs readiness --goal <id>`; o engine de documentação ignora contratos não
  relacionados ao calcular a cobertura de um Goal (suíte de regressão do M5).
- O ciclo do exemplo só toca `product.vision` e `project.scope`; contratos não
  relacionados permanecem intocados.

### 3. Decisões / questões em aberto carregam autoridade e proveniência

- Propostas de decisão expõem classe, ator/fonte, autoridade, confiança
  (apenas para inferência), status e contratos/docs afetados
  (`internal/planning/decision.go`, resolver `internal/planning/resolver.go`,
  E-G02/E-G03).
- O registro de questões em aberto rastreia status de bloqueio, contrato vinculado, responsável,
  estado de criada/resolvida/substituída e evidência de resolução
  (`internal/planning/question.go`, E-G01).
- Uma questão resolvida não é apagada; permanece rastreável até a sua decisão.

### 4. O resume não depende de transcript

- Checkpoint de sessão persistido em `.ai/plan/sessions/<id>.json` como
  `session.Checkpoint()` (`internal/app/sessionstore.go`, E-G07).
- `prumo-agent plan resume` recompila o contexto a partir das decisões canônicas +
  questões em aberto + checkpoint (`internal/app/resume_test.go`,
  `internal/planning/session_test.go`); nenhum transcript é reintroduzido.

### 5. O ciclo de feedback docs/readiness/governança funciona

- Ciclo comprovado por `TestDogfoodZeroToReady` e pelos passos do README do exemplo:
  readiness → questões → decisões respondidas → delta propose → autoria de
  governança do repositório → delta apply (registra a evidência da decisão) → recálculo do
  readiness → proposta de goal/plano.
- `plan delta [--apply]` nunca escreve docs canônicos; a autoria de governança + o
  README documentam esse fluxo (E-G05/E-G09).

### 6. As evals de questões/decisões atingem o baseline aprovado

- O design aprovado (Notion §70.18) define o baseline como evals de conversa
  determinísticas; as evals com modelo real pertencem à futura Harness Eval Suite
  e explicitamente **não** são um requisito de teste unitário.
- Todo cenário determinístico listado tem um teste de regressão: bloqueante ordenado
  antes de nice-to-have; decisão explícita acima de inferência; sugestão de agente não
  aceita automaticamente; conflito com decisão travada produz um achado (E-G04);
  questão resolvida atualiza o registro; lacunas de docs não relacionadas ignoradas no
  readiness do Goal; o resume preserva as decisões aceitas sem transcript.
- Executadas no CI via `go test -race ./...`.

### 7. Nenhuma sugestão de agente é promovida silenciosamente

- Classificações `agent-suggestion` nunca são promovidas a decisões e mantêm a
  questão aberta (enforcement da autoridade de resposta em `internal/planning` +
  `cmd/prumo/plan_z2r_test.go`: `TestPlanAnswerNeverPromotesAgentSuggestion`,
  `TestPlanAnswerUnresolvedKeepsQuestionOpen`).

## Evidência (dogfood do framework neste repositório)

- `examples/living-plan-sample/` — exemplo versionado, estado ready, README
  de replay e o registro do delta aplicado
  `docs/governance-delta-applied.json`.
- Superfície completa da CLI exercitada no exemplo: `plan questions`, `plan
  answer`, `plan decisions`, `plan delta [--apply]`, `plan blueprint [--plan]`,
  `docs readiness --goal` (envelope JSON E-G08 + modos de texto).

## Evidência de testes

- `go test -race ./...` — passa em todos os pacotes.
- `go vet ./...`, `gofmt -l .` — limpos.
- `pytest -q` — 127 passam (oracle Python v0.3 preservado).
- Suítes de regressão M5/M6: isolamento de cobertura/readiness, ciclo de vida do delta,
  achados de contradição na prévia de decisões, resolução de autoridade, round-trip de
  checkpoint, compilação de contexto no resume, envelopes JSON da linha de comando.

## Governança

- O M6 foi mergeado em `main` via PRs squash: E-G01 (#24), E-G02 (#25), E-G03 (#26),
  E-G04 (#27), E-G05 (#28), E-G06 (#29), E-G07 (#30), E-G08 (#31),
  E-G09 (#32).
- Spec canônica: `docs/runtime/living-plan.md`; exemplos/dogfood em
  `examples/living-plan-sample/`.

## Fora do exit gate do M6

- As evals de conversa/decisão com modelo real continuam sendo um item da Harness Eval Suite
  (páginas 61 e 70 §18 do Notion); as evals de regressão determinísticas já satisfazem
  o gate.
- O Adoption Engine (M7), a rastreabilidade (M8) e a Experience (M9) são
  explicitamente não-objetivos do M6.

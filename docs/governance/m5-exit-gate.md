# M5 Documentation System v2 — Exit Gate

Status: **CONCLUÍDO (Exit Gate do M5 Aprovado)**

## Critérios do exit gate

Exit gate do M5 (conforme `docs/development/phases.md`): contratos de documentação aplicáveis; profiles selecionáveis; delta rastreado; contradições detectadas.

### 1. Contratos de documentação aplicáveis

- Registro de contratos: `docs/contracts/builtin.json` (8 contratos), schema `schemas/documentation-contract.schema.json`.
- Bindings: `docs/contracts/bindings.json` com fontes `owned`, `answered_questions` e `evidence`.
- Validação de schema por máquina no carregamento; todo delta armazenado é validado contra `schemas/documentation-delta.schema.json`.
- Verificação: `prumo-agent docs audit` reporta todos os contratos aplicáveis como `implementation-ready`.

### 2. Profiles selecionáveis

- Registro de profiles: `docs/profiles/builtin.json` (7 profiles), schema `schemas/documentation-profile.schema.json`.
- Resolução de aplicabilidade via capacidades do projeto (tipo/features do projeto em `prumo.json`, mais `cmd`, `schemas` e `go.mod` detectados).
- Verificação: `prumo-agent docs profiles` lista `api-service`, `cli`, `compiler`, `core-software`, `desktop-gui`, `library`, `web-application`.

### 3. Delta rastreado

- Ciclo de vida persistente do Delta: `internal/documentation/delta.go` (C-G06).
- IDs determinísticos (SHA-256 de goal, contratos ordenados, documentos ordenados); estáveis entre reanálises.
- Ciclo de vida: `proposed → reviewed → accepted → applied` (applied exige evidência) ou `proposed → rejected`; as versões avançam e todas as transições são validadas.
- Persistência local em `.ai/docs/deltas/DD-<12hex>.json`, validada por schema na leitura, com rejeição de arquivos corrompidos.
- CLI: `prumo-agent docs delta propose|list|show|transition`.
- Fixture de conformance: `conformance/documentation/valid_delta.json`.

### 4. Contradições detectadas

- Achados determinísticos de contradição e de obsolescência causal (`DetectContradictions`, `DetectStaleness`).
- Contradições são apenas achados; nunca são resolvidas silenciosamente por recência ou por saída de modelo.
- CLI: `prumo-agent docs contradictions`.

## Evidência (dogfood do framework neste repositório)

- `prumo-agent docs audit` — todos os 7 contratos aplicáveis `implementation-ready`.
- `prumo-agent docs readiness --goal M5` — `ready: true`, sem contratos bloqueantes, sem questões bloqueantes.
- `prumo-agent docs contradictions` — achados determinísticos (0 na árvore atual).
- `prumo-agent docs profiles` — todos os 7 profiles embutidos resolvíveis.

## Evidência de testes

- `go test ./... -race` — passa em todos os pacotes.
- `go vet ./...` — limpo.
- `pytest -q` — 127 passam (oracle Python v0.3 preservado).
- Ciclo de vida do delta, round-trip de storage, rejeição de corrupção, validação de schema e testes de integração da CLI passam.
- Testes de regressão de determinismo do delta e da fixture de conformance passam.

## Governança

- A aplicação do Documentation Delta continua governada pela Repository Change Governance (`.prumo/repository/policy.json`): branch, commit, Pull Request, validação, revisão e merge.
- `feat/c-g06-documentation-delta-tracking` foi mergeada em `main` via PR #21 (squash).
- Os schemas de máquina em `schemas/documentation-*.schema.json` são a superfície do contrato; mudanças exigem atualização de schema e fixtures de conformance.

## Fora do exit gate do M5

- A assistência semântica a contradições em prosa continua sendo um achado futuro dependente de provider; os achados determinísticos guiados por contrato já satisfazem o exit gate.
- A entrevista do Living Plan (M6), a Adoção (M7), a rastreabilidade completa (M8) e a Experience (M9) são explicitamente não-objetivos do M5.

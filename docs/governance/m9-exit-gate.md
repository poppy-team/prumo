# M9 Experience Layer — Exit Gate

Status: **CONCLUÍDO (Exit Gate do M9 Aprovado)**

## Critérios do exit gate

Exit gate do marco M9 (conforme `docs/development/phases.md`):
um agente pode fazer handoff para outro agente ou sessão com contexto estruturado completo;
os eventos de sessão são unidades semânticas estruturadas, sem transcripts conversacionais;
resumos são gerados por sessão e por Goal; o Experience Provider Contract
governa o armazenamento episódico; as propostas de experiência são validadas antes da promoção;
e as políticas de retenção gerenciam a expiração do armazenamento.

---

### 1. Eventos de Sessão Estruturados (Não Transcript)

- Implementado em `internal/experience/events.go`:
  - Tipos de evento definidos: `session_started`, `goal_selected`, `decision_made`, `checkpoint_saved`, `tool_executed`, `blocker_occurred`, `session_completed`, `handoff_created`.
  - Invariante: apenas eventos semânticos são registrados; transcripts conversacionais e dumps brutos de tokens da LLM são proibidos no armazenamento episódico.
- Contrato de schema: `schemas/experience-event.schema.json`.

---

### 2. Resumos de Sessão e de Goal

- Implementados em `internal/experience/summary.go`:
  - `SynthesizeSessionSummary`: agrega tarefas concluídas, foco atual, bloqueios ativos e decisões.
  - `SynthesizeGoalSummary`: agrega todas as sessões que contribuem para um Goal, derivando o status de progresso (`planned`, `in_progress`, `blocked`).

---

### 3. Protocolo de Handoff (Transferência de Estado entre Agentes / Sessões)

- Implementado em `internal/experience/handoff.go`:
  - Permite a transferência fluida de contexto entre agentes autônomos sem repassar transcripts completos de conversa.
  - O pacote de handoff encapsula:
    - Resumo estruturado da sessão.
    - Decisões ativas e questões em aberto.
    - Hashes de evidência.
    - Estados completos do ciclo de vida (`pending`, `transferred`, `acknowledged`, `rejected`).
- Contrato de schema: `schemas/experience-handoff.schema.json`.

---

### 4. Experience Provider Contract e File Provider

- Definido em `internal/experience/provider.go`:
  - Interface `ExperienceProvider` com métodos para registrar eventos, salvar/carregar resumos e criar/confirmar handoffs.
  - O `FileProvider` implementa armazenamento em arquivos thread-safe em `.prumo/experience/`.

---

### 5. Propostas de Experiência (Governadas por Revisão)

- Implementadas em `internal/experience/proposal.go`:
  - Quando os agentes observam heurísticas reutilizáveis ou padrões recorrentes, eles geram uma `ExperienceProposal`.
  - Invariante: as propostas de experiência são governadas por revisão (`review_required: true`) e não podem ser promovidas a regras canônicas ou skills da workforce sem aprovação da governança.

---

### 6. Políticas de Retenção

- Implementadas em `internal/experience/retention.go`:
  - Idade máxima em horas e máximo de eventos por sessão configuráveis.
  - A poda automatizada (`FileProvider.Prune`) limpa eventos brutos expirados e preserva indefinidamente os resumos sintetizados.

---

### 7. Superfície da CLI

- Implementada em `cmd/prumo/experience_commands.go` e `cmd/prumo/main.go`:
  - `prumo-agent experience status`
  - `prumo-agent experience handoff create --id <id> --from <from> --to <to> --goal <goal>`
  - `prumo-agent experience handoff show <id>`
  - `prumo-agent experience handoff ack <id> --actor <actor>`
  - `prumo-agent experience events --session <id>`
- Verificada por testes automatizados em `cmd/prumo/experience_commands_test.go`.

---

## Evidência e Verificação de Testes

- `go test -race ./...` — 100% passam em todos os pacotes.
- `go vet ./...` — limpo.
- `gofmt -l .` — limpo.
- `python -m pytest -q` — 136 passam (oracle Python preservado).
- Suíte de experience (`internal/experience`): todos os testes passando (100%).
- Suíte da CLI (`cmd/prumo`): todos os testes passando (100%).

---

## Conclusão

O marco M9 (Experience Layer) está **CONCLUÍDO**. Todos os critérios e invariantes do exit gate foram satisfeitos.

# M8 History + Traceability — Exit Gate

Status: **CONCLUÍDO (Exit Gate do M8 Aprovado)**

## Critérios do exit gate

Exit gate do marco M8 (conforme `docs/development/phases.md`):
qualquer mudança de código é rastreável até uma decisão e um goal; o journal de implementação é consultável;
experimentos, rejeições e dívida técnica são rastreados; o grafo de rastreabilidade tipado liga
requisitos, decisões, goals, código, testes, documentação e evidência.

---

### 1. Grafo de Rastreabilidade Tipado (req ↔ dec ↔ goal ↔ code ↔ test ↔ doc ↔ evidence)

- Modelo canônico do grafo implementado em `internal/traceability/graph.go`:
  - Tipos de nó: `req`, `dec`, `goal`, `code`, `test`, `doc`, `evidence`, `experiment`, `rejection`, `debt`.
  - Tipos de aresta: `satisfies`, `derives_from`, `implements`, `verifies`, `documents`, `evidenced_by`, `rejects`, `incurs_debt`.
  - A travessia bidirecional (`TracePath`) resolve linhagens completas a montante (decisões, requisitos) e a jusante (testes, evidência, documentação).
- Contrato de schema: `schemas/trace-graph.schema.json`.

---

### 2. Journal de Implementação (Síntese, Não Chain-of-Thought)

- Implementado em `internal/traceability/journal.go`:
  - Faz valer o Lean Progressive Context: armazena resumos estruturados de implementação, decisões afetadas, código alterado, testes de verificação e hashes de evidência.
  - Invariante: chain-of-thought bruto, transcripts de conversa verbosos e telemetria não determinística são proibidos no armazenamento canônico do journal.
  - Consultável por Goal, Decisão ou caminho de arquivo.
- Contrato de schema: `schemas/journal-entry.schema.json`.

---

### 3. Registros de Experimentos, Rejeições e Dívida

- Implementados em `internal/traceability/registers.go`:
  - **Experiment Register**: captura hipóteses, métodos, resultados e conclusões arquiteturais.
  - **Rejection Register**: rastreia propostas descartadas, alternativas avaliadas e a justificativa autoritativa da rejeição.
  - **Debt Register**: rastreia dívida técnica/arquitetural com severidade (`low`, `medium`, `high`, `critical`), contratos impactados e planos de remediação concretos.

---

### 4. Superfície da CLI (`prumo-agent trace` e `prumo-agent journal`)

- Implementada em `cmd/prumo/trace_commands.go`:
  - `prumo-agent trace <ref>`: rastreia qualquer goal, decisão, arquivo de código ou requisito, retornando linhagens estruturadas a montante/jusante em formato humano de terminal ou em envelope JSON.
  - `prumo-agent journal [--goal <goal>] [--decision <decision>] [--file <file>]`: consulta o histórico de implementação.
- Verificada por testes automatizados em `cmd/prumo/trace_commands_test.go`.

---

## Evidência e Verificação de Testes

- `go test -race ./...` — 100% passam em todos os pacotes.
- `go vet ./...` — limpo.
- `gofmt -l .` — limpo.
- `python -m pytest -q` — 136 passam (oracle Python preservado).
- Suíte de rastreabilidade (`internal/traceability`): todos os testes passando (100%).
- Suíte da CLI (`cmd/prumo`): todos os testes passando (100%).

---

## Conclusão

O marco M8 (History + Traceability) está **CONCLUÍDO**. Todos os critérios e invariantes do exit gate foram satisfeitos.

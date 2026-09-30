# M7 Adoption Engine — Exit Gate

Status: **CONCLUÍDO (Exit Gate do M7 Aprovado — ADOPTION READY)**

## Critérios do exit gate

Exit gate do marco M7 (conforme `docs/development/phases.md` e `docs/runtime/adoption-engine.md`):
um repositório existente qualquer pode ser auditado sem mutação; fatos observados e inferências
são mantidos separados; confiança e evidência acompanham todas as inferências no Confidence Ledger;
os contratos de documentação do M5 avaliam os bindings candidatos; a ambiguidade entra no Living Plan por meio de
questões de incerteza; as propostas de migração são formais, não destrutivas e governadas por revisão;
o scanner é incremental e ciente de revisão/branch; conteúdo malicioso e vazamento de segredos são prevenidos.

---

### 1. Um repositório existente qualquer pode ser auditado sem mutação (Zero Forced Layout)

- Comprovado por testes de regressão automatizados em 6 repositórios de fixture brownfield (`internal/adoption/evals_test.go`):
  - `TestBrownfieldCorpusGoCLI`: CLI em Go com layout não-Prumo (`cmd/`, `internal/`, `README.md` padrão, `docs/architecture.md`).
  - `TestBrownfieldCorpusWebMonorepo`: Aplicação web com múltiplos pacotes (React, Vite, Express, PostgreSQL).
  - `TestBrownfieldCorpusMaliciousInjection`: Repositório não confiável contendo diretivas de prompt injection.
  - `TestBrownfieldCorpusSecretsEnv`: Repositório contendo `.env` e chaves de API sensíveis.
  - `TestBrownfieldCorpusStaleConflictingDocs`: Repositório com documentação histórica conflitante.
  - `TestBrownfieldCorpusNoDocs`: Projeto sem nenhum arquivo de documentação.
- Invariante: `hashDirectory` verifica que, antes e depois de `RunAdoptionAudit`, a árvore do repositório e o conteúdo de todos os arquivos permanecem 100% idênticos byte a byte. Nenhum arquivo é movido, renomeado ou apagado.

---

### 2. Fatos observados e inferências são mantidos estritamente separados

- A verdade factual é capturada exclusivamente em `ObservedFact` (`internal/adoption/fact.go`) com `ConfidenceFactual` e fontes de extração concretas (nomes de arquivos, estruturas de diretórios, campos de manifestos parseados).
- Suposições derivadas são isoladas como `ClassificationSignal`, `ProfileCandidate`, `CapabilityProposal` e `CandidateBinding`.
- Nem o scanner nem o classifier elevam inferências a fatos.

---

### 3. Confiança e evidência acompanham todas as inferências (Confidence Ledger)

- O `ConfidenceLedger` canônico (`internal/adoption/ledger.go`, schema `schemas/confidence-ledger.schema.json`) registra todas as afirmações factuais e inferidas.
- Invariante garantido por `LedgerEntry.Validate()`: toda entrada de confiança baixa e média exige confirmação humana (`RequiresConfirmation: true`) e aponta de volta, em `Evidence`, para IDs de fatos observados concretos.
- `prumo-agent adopt --strict` rejeita a execução se restarem inferências não confirmadas ou contradições não resolvidas.

---

### 4. O M5 avalia corretamente os bindings candidatos

- Layouts de documentação não-Prumo são mapeados para os contratos de documentação canônicos do Prumo (`internal/adoption/mapping.go`, schema `schemas/mapping-candidate.schema.json`).
- Bindings de alta confiança são propostos para os contratos exigidos pelos profiles detectados (p. ex. `product.vision`, `system.architecture`, `testing.strategy`).
- Contratos obrigatórios ausentes são reportados em `DocCoverageSummary` sem fazer a auditoria falhar.

---

### 5. A ambiguidade entra no Living Plan / Open Questions

- `GenerateQuestions` (`internal/adoption/uncertainty.go`, schema `schemas/adoption-resolution.schema.json`) transforma inferências não confirmadas em questões interativas de entrevista.
- Questões não resolvidas bloqueiam a adoção strict até que uma decisão explícita seja registrada por um operador humano ou ator autorizado.
- A passada não interativa (`ResolveNonInteractive`) aplica de forma determinística as escolhas padrão e registra a proveniência.

---

### 6. As propostas de migração são dry-run e governadas por revisão

- O Adoption Engine não aplica mudanças ad hoc. Achados confirmados produzem itens formais `AdoptionMigrationProposal` (`internal/adoption/migration.go`, schema `schemas/adoption-migration-proposal.schema.json`).
- Toda proposta exige aprovação da Review Queue antes de ser aplicada (`review_required: true`).
- `DryRun` avalia pré-condições (p. ex. `manifest_absent:prumo.json`) e produz prévias de diff unificado sem mutações em disco.
- `Apply` faz valer o invariante de aprovação (`ProposalStatusApproved`), executa mudanças atômicas e reversíveis e registra a evidência de auditoria em `migrations.JournalEntry` com hash de integridade SHA-256.
- Flags da CLI implementadas em `cmd/prumo/adopt_commands.go`:
  - `prumo-agent adopt --propose-migration`
  - `prumo-agent adopt --dry-run`
  - `prumo-agent adopt --apply`

---

### 7. O scanner é incremental e ciente de revisão/branch

- Usa o índice de repositório D3 e `runtime.InspectRepository` (`internal/adoption/scanner.go`).
- Captura `branch`, `revision` e o estado `dirty` da árvore de trabalho.
- Suporta `ScanBudget` (`MaxFiles`, `MaxBytes`) com marcadores de continuação graciosa.
- Avalia hashes SHA-256 de arquivos para rastrear mudanças entre revisões.

---

### 8. Os invariantes de segurança contra conteúdo malicioso e segredos se mantêm

- **Nenhuma ingestão de segredos**: arquivos `.env` e de credenciais são detectados (`env-presence:.env`) para divulgação de risco, mas os valores secretos nunca são lidos, indexados nem persistidos em fatos ou no ledger (`content: [REDACTED_BY_SECURITY_POLICY]`). Testado em `TestBrownfieldCorpusSecretsEnv`.
- **Zero promoção canônica falsa**: prompt injection em arquivos README/AGENTS de terceiros (`TestBrownfieldCorpusMaliciousInjection`) não ganha nenhuma autoridade de política. Todos os bindings candidatos permanecem `inferred-state`, sem nenhuma promoção a especificações canônicas.

---

## Evidência (Auditoria de Dogfood no Prumo)

- Auditoria completa executada contra o próprio repositório do Prumo (`TestDogfoodPrumoSelfAudit`):
  - Linguagens detectadas: Go, Python, Shell.
  - Tipo de aplicação detectado: CLI.
  - Frameworks e toolchains detectados: toolchain padrão do Go, pytest, JSON Schema.
  - Artefatos Prumo existentes descobertos: `prumo.json`, docs canônicos, contratos, schemas.
  - Saída renderizada via `prumo-agent adopt` e `RenderHumanReport` sem nenhuma modificação de arquivo.

---

## Verificação de Testes

- `go test -race ./...` — 100% passam em todos os pacotes.
- `go vet ./...` — limpo.
- `gofmt -l .` — limpo.
- `python -m pytest -q` — 136 passam (oracle Python preservado).
- Suíte do Adoption Engine (`internal/adoption`): 48 testes passando (100%).
- Suíte da CLI (`cmd/prumo`): todos os testes passando (100%).

---

## Conclusão

O marco M7 (Adoption Engine) está **CONCLUÍDO**. Todos os critérios de aceitação e invariantes do exit gate foram satisfeitos. O framework é declarado **ADOPTION READY**.

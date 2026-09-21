# 64 — Fase A: Go Core Parity & Distribution Closure

> Authority: canonical specification.
> Logical ID: PHASE-64
> Source: Notion Living Book (3d69bb7d023f81689e22f267fe00b0d3)
> Status: Fase/Gate de implementação (64 — Fase A Go Core Parity & Distribution Closure).


## Papel no programa

Encerrar de forma objetiva o período de migração Python → Go e transformar a fundação existente em um produto distribuível, sem adicionar novas famílias v0.4 antes de a base possuir parity, conformance e release verificáveis.

## Fontes

- [01 — Stack Go, Clean Code e Migração do Core Python](../../framework/specs/ch01.md)
- [11 — Instalação, Atualização, Desinstalação e Distribuição](../../framework/specs/ch11.md)
- [14 — Roadmap de Implementação v0.4](../../framework/specs/ch14.md)
- [16 — Gap Analysis v0.3 → v0.4 e Correções Necessárias](../../framework/specs/ch16.md)
- [17 — Estrutura Canônica de Repositório Alvo](../../framework/specs/ch17.md)

## Objetivo

Declarar **GO CORE PARITY COMPLETE** somente quando o runtime Go reproduzir todos os contracts críticos do v0.3 que decidimos preservar, a distribuição funcionar sem Python/Node como dependência do Core e houver uma estratégia explícita para remoção/depreciação dos artefatos Python restantes.

## Dependencies

- baseline v0.3 congelado;
- conformance fixtures/golden conhecidos;
- Go module e CLI existentes;
- schemas/resources canônicos preservados.

## Non-goals

- Documentation System v2;
- Living Plan;
- Experience;
- Control Plane avançado;
- novos connectors;
- migração estética de todo arquivo do repositório.

## Workstreams

### A1 — Protocol parity final

Verificar e fechar:

- Goal v2, locking, amendments e hashes;
- Plans/Tasks e DAG;
- Evidence/Gates/Events;
- project/config/profile;
- resolver/workforce/model policy legado compatível;
- validation/schema registry;
- doctor/explain/framework-check;
- machine envelope e exit codes.

**Invariante:** Go se conforma ao protocolo; não alterar protocolo somente para facilitar Go.

### A2 — Compiler parity

Inventariar todos os targets ainda suportados. Cada target é classificado como `retain`, `replace`, `deprecate` ou `remove-with-migration-note`.

Para targets retidos:

- output determinístico;
- ownership markers;
- cleanup behavior;
- golden tests;
- conformance semântica/byte-level quando aplicável.

### A3 — Python retirement

Classificar cada `.py` remanescente:

1. oracle/conformance temporário;
2. migration helper temporário;
3. dev-only script substituível;
4. obsolete.

Remover runtime Python somente depois de parity crítico 100%. Preservar uma tag histórica do baseline e fixtures necessárias para regressão.

Nenhum projeto Atlas deve exigir `pip install` para executar o Core v0.4.

### A4 — Embedded assets

Schemas, templates e resources distribuídos com o binário devem usar `go:embed` ou uma abstração equivalente sem duplicar a fonte canônica.

Regras:

- arquivos fonte continuam revisáveis no repositório;
- embedded copy é build artifact lógico;
- hashes/versions verificáveis;
- não esconder templates em constantes Go gigantes.

### A5 — Distribution closure

Release matrix mínima:

- Linux amd64/arm64;
- macOS amd64/arm64;
- Windows amd64.

Artifacts:

- binary;
- checksum manifest;
- release metadata;
- signature/provenance quando estratégia estiver fechada;
- install/uninstall scripts.

Canais:

- GitHub Releases obrigatório;
- script oficial shell;
- Homebrew;
- Windows baseline via WinGet/Scoop conforme viabilidade real;
- `go install` para desenvolvimento.

### A6 — Installation ownership

Separar:

- binary;
- `$ATLAS_HOME` global config/cache/runtime;
- project canonical state;
- connector-managed fragments.

`atlas uninstall` nunca remove documentação/Goals/canonical project state por padrão.

Cleanup Manifest registra paths criados/modificados por connector/provider.

### A7 — Setup/Doctor

`atlas setup` deve:

- validar ambiente;
- detectar harnesses disponíveis;
- explicar o que será configurado;
- ser idempotente;
- não instalar integrações globais silenciosamente.

`atlas doctor` deve diagnosticar instalação, PATH, versões/protocol mismatches, embedded resources e permissions relevantes.

## Package boundaries esperados

A estrutura exata segue o repositório real. Boundaries mínimas:

- `cmd/atlas` fino;
- `internal/protocol` sem dependência de CLI/host;
- `internal/project` para project state;
- `internal/validation`/validator;
- `internal/resolver`;
- `internal/compiler` se materializado;
- `internal/install` para lifecycle de instalação;
- resources embutidos isolados de business logic.

Não criar SDK Go público nesta fase.

## Schemas/contracts afetados

- todos os v0.3 critical contracts preservados;
- machine envelope;
- installation/cleanup manifest se persistidos;
- project config/profile;
- compiler artifact ownership metadata se canônico.

Toda mudança de schema requer migration explícita e fixture de versão anterior.

## CLI de fechamento

Verificar integralmente a superfície pública preservada e:

- `atlas version`;
- `atlas doctor`;
- `atlas setup`;
- install/uninstall/update-related commands definidos;
- compiler commands;
- `--json` estável onde aplicável.

## Goal decomposition recomendado

- A-G01: freeze/current protocol inventory reconciliation.
- A-G02: critical protocol conformance zero-diff.
- A-G03: compiler target inventory + parity.
- A-G04: embedded assets + single-binary validation.
- A-G05: installation manifest + cleanup semantics.
- A-G06: cross-platform release pipeline.
- A-G07: Python runtime retirement proposal/removal.
- A-G08: release dogfood + rollback.

Não agrupar tudo em um único PR.

## Test strategy

### Unit

Todas as regras determinísticas de protocol, install e compiler.

### Conformance

Mesma fixture → Python oracle e Go → comparar:

- exit code;
- stdout/stderr semântico;
- JSON;
- files created/modified/deleted;
- hashes quando contract exigir.

### Integration

Temp repositories com múltiplos estados e versions.

### Release

Clean environment/VM/container por plataforma, cobrindo install → setup → init/use → upgrade/rollback quando disponível → uninstall.

### Quality gates

- gofmt;
- go vet;
- go test ./...;
- race onde apropriado;
- schema validation;
- conformance 100% em contracts críticos.

## Dogfood

O próprio Atlas deve ser instalável a partir de um release candidate limpo e executar seus próprios validation/conformance commands sem Python instalado.

## Evidence

- conformance report;
- target compiler matrix;
- release artifact manifest;
- install/uninstall logs sanitizados;
- platform validation matrix;
- lista final de Python files e sua classificação.

## Rollback

- release anterior permanece recuperável;
- migration nunca destrutiva sem backup/rollback;
- tag baseline Python preservada;
- falha de release não altera project canonical state.

## Exit Gate — GO CORE PARITY COMPLETE

Passa somente se:

- conformance crítico = 100%;
- Go executa Core/CLI sem Python;
- compiler retained targets passam seus gates;
- release artifacts reproduzíveis/validáveis;
- install/uninstall seguro em target platforms definidos;
- nenhuma dependência de Node no Atlas Core;
- Python residual é removido ou explicitamente dev/oracle-only com plano de retirada;
- docs de instalação, quickstart, update, rollback, uninstall e troubleshooting refletem a realidade.
# 91 — Glossário Canônico, Ontologia Operacional e Termos Proibidos de Confundir

> Authority: canonical specification.
> Logical ID: CONST-91
> Source: Notion Living Book (3e29bb7d023f81feb977ec88bac4193a)
> Status: Constituição 91: Glossário Canônico e Ontologia Operacional.


<aside>
📖

**Status: vocabulário canônico.** Agents, documentação, schemas, CLI/TUI/GUI e prompts devem usar estes termos de forma consistente. Sinônimos históricos podem existir para migration/search, mas não devem criar entidades semânticas duplicadas.

</aside>

# Regra geral

Quando dois termos abaixo forem semanticamente diferentes, **não tratá-los como sinônimos para simplificar uma implementação**.

Quando um termo histórico aparecer, o resolver deve mapear alias → termo canônico e preservar provenance.

# Identidade do produto

## Prumo

Nome canônico da plataforma inteira.

Inclui Framework/Project Engineering, Harness, runtimes, knowledge/documentation, workforce, quality/evidence, protocol/SDK e surfaces.

**Alias histórico:** Project Atlas / Atlas.

Uso atual de Atlas deve ser limitado a migration/provenance ou arquivos ainda não migrados.

## Prumo Platform

Forma explícita de se referir ao conjunto de capacidades genéricas do Prumo quando é necessário contrastá-lo com um produto especializado.

## Prumo Framework

Termo histórico e ainda útil para a camada de contracts, protocol de engenharia, documentação, project/governance e policies.

Não significa “somente uma library” e não é uma segunda plataforma separada.

## Prumo Harness

Execution/control plane que transforma Task/Run em execução agentic governada.

Não é UI.

## Prumo Native Agent

Agent Runtime controlado pelo próprio Prumo, usando ModelProvider + ToolGateway + policies Prumo.

## External Agent

Runtime externo que controla seu próprio loop e é integrado como AgentProvider.

## Prumo Code Agent

Domínio/specialization de coding sobre o Prumo Harness.

No Livro Vivo, é a Parte II com runtime/client/product detail.

Não é um segundo Core.

## Prumo Code

Nome reservado ao futuro produto/repositório coding-oriented que consome o Prumo Protocol.

Não importar internal/ do Prumo.

# Trabalho e execução

## Goal

Resultado de engenharia de nível superior com acceptance e lifecycle.

## Plan

Decomposição ordenada/estruturada de um Goal.

## Task

Unidade executável com scope, acceptance e dependencies.

## Run

Instância durável de execução Prumo.

Sobrevive a troca de model, agent, client e processo conforme policy.

**Run != GUI.**

**Run != model call.**

**Run != transcript.**

## Session

Contexto de interação/continuidade associado a uma surface, AgentProvider ou runtime conforme contract.

Pode estar relacionada a uma Run, mas não substitui sua identidade.

## Thread

Projeção/conversa lógica para UX e organização.

Não deve ser usada como state owner de engenharia.

## Turn

Unidade de interação model/agent dentro de uma session/run.

## Step

Boundary pequena da state machine reentrante do Agent Runtime.

## Checkpoint

Snapshot/record suficiente para retomar estado sem repetir side effects indevidamente.

## Continuation

Artefato explícito de continuidade após stop, crash, handoff ou mudança de executor.

# Providers e modelos

## Model

Modelo de inferência identificado por metadata/capabilities/version.

## ModelProvider

Adapter de inferência/model calls.

Prumo controla o agent loop.

## AgentProvider

Adapter para um runtime de agent externo.

O provider externo controla seu próprio loop/session/tool semantics dentro de seus boundaries.

**ModelProvider != AgentProvider.**

## ProviderProfile

Configuração concreta de conexão/auth/provider.

Pode conter credencial, endpoint, account, policy e overrides.

## ModelGateway

Runtime que resolve ModelRoute/RouteTarget, health, quotas, fallback e pools.

## ModelRoute

Alias/policy lógico para escolher target(s) de modelo.

## RouteTarget

Combinação elegível de provider profile + model + constraints.

## AccountPool

Conjunto autorizado de ProviderProfiles para routing conforme policy e termos.

## QuotaState

Conhecimento sobre limite externo:

known, estimated, unknown, cooldown ou exhausted.

Unknown nunca deve ser materializado como saldo inventado.

# Workforce

## Workforce Runtime

Seleciona/compoõe agents/roles necessários a uma Run.

## AgentRole

Responsabilidade: Planner, Implementer, Reviewer, Researcher etc.

## AgentBinding

Binding entre role e AgentProvider/Native Agent/model route/budget/context/permissions.

## Least Workforce

Princípio: usar menor composição que cobre risco/capability.

Não significa sempre um agent.

## Model diversity

Uso deliberado de modelo/provider diferente para reduzir correlação, especialmente review.

**Workforce routing != Model routing.**

# Falha, fallback e handoff

## Retry

Nova tentativa semanticamente do mesmo passo/operação segundo idempotency policy.

## Fallback

Troca transparente de model/route somente enquanto ainda é seguro fazê-la sem esconder efeitos observáveis.

## Handoff

Transferência explícita de execução depois que existe state/side effect/agent ownership relevante.

**Fallback depois de side effect deve virar Handoff.**

## Recovery

Restauração de estado consistente após failure/cancel/crash.

## Rollback

Retorno explícito a estado anterior quando semanticamente suportado.

## Roll-forward

Correção/progressão para estado consistente posterior sem voltar à versão anterior.

# Contexto e conhecimento

## Knowledge

Informação durável estruturada: decisions, requirements, findings, patterns, evidence etc.

## Context

Subset compilado para uma task/role/turn.

Context não é repositório inteiro.

## Context Compiler

Runtime que seleciona/empacota contexto sob authority, freshness, relevance e budget.

## Context Manifest

Registro reproduzível do que entrou/saiu e por quê.

## Implementation Dossier

Contrato task-specific compilado com scope, reality, mutation, acceptance, verification, uncertainty e provenance.

## Microcontext

Contexto mínimo especializado para uma role/step.

Não é um resumo arbitrário.

## Compaction

Transformação estruturada para reduzir contexto preservando informação necessária.

Não significa apagar constraints silenciosamente.

# Documentação

## Canonical Record

Entidade normativa/operacional estruturada.

## Knowledge IR

Representação normalizada do conhecimento.

## Document IR

Representação semântica de um documento/projeção.

## Generated Artifact

Saída derivada reconstruível.

## Curated Document

Documento com ownership humano/assistido explícito.

## Imported Snapshot

Cópia imutável de fonte externa/notebook para migration/reconciliation.

Não vira canonical automaticamente.

## Promotion

Processo que transforma candidate/imported knowledge em estado canônico após reconciliation/validation.

## Documentation Delta

Impacto documental derivado de mudança de behavior/contract/code.

## Documentation Contract

Requisitos de coverage/freshness/authority/evidence de um semantic document role.

## Documentation Profile

Composição de contracts ativada pelas capabilities do projeto.

# Verdade e incerteza

## Claim

Afirmação material com source/status/authority.

## Decision

Escolha governada com status/lifecycle.

Proposal não é Accepted.

## Requirement

Comportamento/constraint exigido por autoridade válida.

## Constraint

Limite que restringe soluções.

## Assumption

Valor provisório explicitamente registrado para preencher uma incerteza permitida.

## Unknown

Informação necessária ausente/insuficiente.

## Contradiction

Claims incompatíveis que não podem ser fundidos por resumo.

## Superseded

Status de decisão/claim substituído por outro mais atual/aplicável.

## Stale

Informação cuja freshness não é suficiente para uso current sem revalidação.

## Drift

Divergência entre sources/target/implementation/documentation.

# Skills, agents e recipes

## Capability

O que o sistema consegue fazer ou precisa cobrir.

Não implica package físico.

## Skill

Pacote versionado de conhecimento, procedimento, checks, workflows, schemas, tests/evals e outputs.

**Skill != prompt.**

## Agent

Executor/role com autonomia delimitada, lifecycle, permissions e handoff.

## Recipe

DAG/workflow recorrente que compõe capabilities/skills/agents/gates.

## Bundle/Profile

Conjunto candidato/configuração de componentes.

Resolver ainda deve aplicar Least Workforce/Least Context.

## Check

Validação determinística.

Não criar skill somente para executar uma condição que poderia ser check.

# Tools e protocols

## Tool

Capability executável com input/output schema, permissions e side-effect semantics.

## ToolGateway

Boundary Prumo para discovery, permission, execution e normalization de tools.

## ACI

Agent-Computer Interface: conjunto estruturado de operações para coding/computer work.

Não é apenas shell.

## MCP

Agent ↔ Tool/Context protocol.

## ACP

Client/Editor ↔ Agent protocol.

## A2A

Agent ↔ Agent interoperabilidade remota/independente conforme standard/profile suportado.

**MCP != ACP != A2A.**

## Prumo Protocol

Boundary pública versionada de Commands/Events/Projections/Capabilities entre Core e clients.

## SDK

Bindings/helpers que consomem o Protocol.

SDK não é source da semântica; schema/protocol é.

# Surfaces

## Headless CLI

Surface machine/human-friendly sem UI persistente.

## prumo ask

One-shot read-only/no-tools por default.

## prumo agent run

Execução agentic durável.

## TUI

Client terminal interativo.

Direção vigente: Go + Bubble Tea v2.

## Native GUI / Workspace Viewer

Client desktop agent-aware.

Direção vigente: base Oxi, com editor/viewer leve.

Não é um segundo Harness.

## IDE/ACP Client

Client de editor que consome capabilities de agent/protocol.

# GUI stack status

## Oxi

Base vigente do Native Workspace Viewer.

## Floem

Direção histórica/SUPERSEDED para o Desktop principal.

Pode permanecer como research/reference; não selecionar automaticamente.

## Bubble Tea v2

Direção vigente do TUI.

# Quality

## Evidence

Artefato/fato verificável sobre execução/comportamento.

## Gate

Regra que avalia Evidence/estado e permite/bloqueia progressão.

## Readiness

Estado derivado de contracts, coverage, evidence, blockers e policies.

## Gauntlet Loop

Ciclo explícito de implementação→teste→blind-spot/review→fix→revalidation até stopping rule.

## False-green

Estado aparentemente bem-sucedido sem proof suficiente.

## Proof-of-use

Evidence de que uma surface/capability realmente foi alcançada e exercitada.

## Waiver

Exceção formal com owner, rationale, risk, expiry e compensating controls.

Waiver não equivale a ignore.

# Status de implementação

Usar semanticamente:

- not_started;
- planned;
- in_progress;
- partial;
- implemented;
- reachable;
- exercised;
- evidenced;
- verified;
- accepted;
- released;
- blocked;
- deprecated;
- superseded.

Não comprimir esses estados em um boolean done.

# Budget

## Budget Envelope

Limites hierárquicos de recursos.

## Reservation

Fatia reservada para child/step/reviewer.

## Review Reserve

Budget preservado para verification/fix/docs delta.

## Hard Limit

Não pode ser excedido; produz stop seguro/checkpoint.

## Soft Limit

Aciona otimização/redução opcional sem violar quality floor.

## Cost

Consumo monetário observado/estimado com pricing revision.

**Budget != quota.**

Budget é policy interna; quota é restrição externa/provider.

# Decision Runtime

Runtime de decisões pequenas/fechadas com abstention e reason codes.

Não possui agent loop aberto nem side effects arbitrários.

**Decision Runtime != Agent Runtime.**

# Experience e learning

## Experience

Registro de resultado/padrão observado em Runs.

## Global Learning Layer

Mecanismo para propor padrões cross-project com governance.

## Pattern Proposal

Candidato derivado de experience, ainda não canonical.

Experience nunca altera policy/skill automaticamente sem eval/review.

# Aliases e migration

Aliases permitidos:

- Atlas → Prumo;
- Atlas Core → Prumo Core;
- Atlas Skill Package → Prumo Skill Package;
- Atlas CLI commands → Prumo namespace quando migrados.

Aliases devem servir search/migration, não manter documentação corrente eternamente duplicada.

# Regra para agents

Se um termo encontrado em documentação antiga conflitar com este glossário:

1. preservar o source;
2. mapear alias/status;
3. usar termo canônico no novo output;
4. não reescrever histórico sem migration task;
5. registrar ambiguity se mapping não for seguro.

# Definition of Done

- principais entidades possuem definição única;
- relações negativas importantes estão explícitas;
- aliases Atlas→Prumo estão documentados;
- Oxi/Floem status não é ambíguo;
- ModelProvider/AgentProvider e Workforce/Gateway não são confundidos;
- Run/Session/Thread não são fundidos;
- Context/Knowledge não são confundidos;
- Evidence/Gate/Readiness permanecem distintos;
- agents usam termos canônicos em novos artifacts.
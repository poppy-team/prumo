# 86 — Arquitetura Unificada: Framework, Harness, Code Agent e Surface Contracts

> Authority: canonical specification.
> Logical ID: CONST-86
> Source: Notion Living Book (3e29bb7d023f81d0b8adef7ab3ea69d3)
> Status: Constituição 86: Arquitetura Unificada de Framework, Harness, Code Agent e Surfaces.


<aside>
🏗️

**Status: arquitetura consolidada.** O Prumo passa a ser documentado como uma única plataforma. “Framework” descreve contratos e engenharia do projeto; “Harness/Agent Runtime” executa; “Prumo Code Agent” é o domínio/produto especializado em software; CLI, TUI, GUI, ACP e automações são surfaces do mesmo Core.

</aside>

# Regra de ownership

PRUMO PLATFORM

- Canonical Project/Knowledge Model
- Control Plane
- Decision Runtime
- Workforce Runtime
- Native Agent Runtime
- External Agent Providers
- Context Compiler
- Model Gateway
- Tool Gateway / ACI / MCP
- Permissions / Egress / Sandbox
- Budget Runtime
- Quality / Evidence / Gauntlet
- Documentation Runtime
- Experience / Global Learning
- Protocol / SDK
- Surfaces: Headless CLI, TUI, Native GUI / Workspace Viewer, ACP/IDE clients e automation/CI.

# Boundary principal

**Uma regra de domínio deve existir uma vez.**

Se uma surface precisa conhecer regra específica para funcionar, a regra deve estar em application/domain service ou protocol capability, não duplicada na UI.

# Framework domain

Responsável por:

- Project/Goal/Plan/Task;
- documentation contracts;
- knowledge/decision/requirement model;
- policy/governance;
- traceability;
- evidence/gates/readiness;
- experience/global learning;
- adoption/planning;
- repository change governance.

# Harness domain

Responsável por:

- Run lifecycle;
- state machine agentic;
- checkpoint/resume;
- model/tool execution;
- permission lifecycle;
- context assembly;
- delegation/handoff;
- budgets;
- observability;
- sandbox/worktrees;
- cancellation/retry.

# Code Agent domain

É specialization profile sobre o Harness:

- coding ACI;
- repository intelligence;
- patch/diff/review;
- tests/build/debug workflows;
- code-oriented recipes;
- code-agent UX;
- coding-specific evidence.

Não é um segundo Core.

# Surface contract

Toda surface:

1. descobre capabilities;
2. envia Commands;
3. recebe Events/projections;
4. nunca acessa storage interno diretamente;
5. nunca possui vendor-specific model logic;
6. nunca interpreta state privado como canonical;
7. pode reconectar/replay;
8. deve funcionar sob protocol version negotiation.

# Headless CLI

## prumo ask

- one-shot;
- read-only/no-tools por default;
- stdout limpo; diagnósticos em stderr;
- usa Context Compiler e Model Gateway;
- não cria agent run completa sem pedido explícito.

## prumo agent run

- Run durável;
- tools/mutations conforme policy;
- checkpoints;
- evidence;
- budget;
- resume/cancel.

## prumo explain

Direção obrigatória para explainability:

- prumo explain run ID;
- prumo explain context ID;
- prumo explain route ID;
- prumo explain workforce ID;
- prumo explain budget ID;
- prumo explain decision ID.

O output fornece reason codes e refs, não raciocínio privado do modelo.

# TUI

Direção vigente:

- Go + Bubble Tea v2;
- referências de UX do OpenCode antigo podem ser reaproveitadas/migradas;
- nenhum runtime agentic duplicado;
- multi-agent simultâneo é projeção/controle do Workforce Runtime;
- model/quota selection vem do Gateway;
- imagem/clipboard e outras features são adapters da surface, não semântica do Core;
- editor/viewer somente onde útil ao workflow.

# Native GUI

A decisão mais recente é **reconstruir o Workspace Viewer agent-aware do zero com Freya** (GUI declarativa em Rust, renderização Skia), supersedendo tanto a antiga direção Floem quanto a direção Oxi (agora histórica).

Objetivo:

1. ver o que o agent está fazendo;
2. entender arquivos e alterações;
3. acompanhar execução e evidence;
4. revisar;
5. fazer pequenas correções.

Não transformar em IDE generalista sem evidence de necessidade.

## GUI invariants

- Run existe sem GUI;
- fechar GUI não encerra Run por semântica implícita;
- file badges vêm de events + repository state;
- Follow Agent usa path/range estruturado;
- evidence/gate state não é inferido por cor local;
- conflitos human edit × agent edit são explicitamente modelados;
- GUI não grava canonical Run state por bypass.

# Prumo Protocol

Protocol é a fronteira real entre Core e clients.

Elementos mínimos:

- Command;
- Event;
- Projection;
- Capability;
- Error;
- Version;
- Cursor.

DTO vendor-specific não atravessa a boundary.

# Daemon

prumo serve hospeda os mesmos application services.

Local transport:

- Unix socket / named pipe preferencial;
- fallback transport versionado quando necessário.

Futuro remoto exige auth, egress policy, replay e capability negotiation; não alterar domínio para acomodar transporte.

# Decision Runtime

Executa pequenas decisões fechadas:

- task classification;
- docs impact;
- workforce recommendation;
- routing signals;
- quality relevance.

Não pode virar um segundo Agent Runtime e não possui side effects arbitrários.

# Workforce Runtime

Seleciona a menor composição suficiente.

Default:

- solo/manual para tarefa simples;
- suggested quando há ganho claro;
- automatic somente em profile explícito.

Implementer e Reviewer podem exigir diversidade de modelo/provider em risco alto.

# Model Gateway

Resolve ModelRoute, health, quota, fallback e account pools.

Fallback silencioso só antes de side effects observáveis.

Depois de side effects: Handoff explícito.

# Context Compiler

Produz contexto por task/role e não por “caderno inteiro”.

Ordenação:

1. invariants/contracts;
2. task acceptance;
3. relevant current implementation;
4. exact supporting knowledge;
5. optional research.

Transcript completo não é contexto padrão.

# Tool Gateway

Tools são capability contracts e devem declarar:

- identity;
- permissions;
- side-effect class;
- input schema;
- output schema;
- timeout/cancel;
- evidence hooks;
- result size budget.

Shell é uma tool importante, não a representação inteira do computador.

# Quality Plane

Gauntlet/Evidence é transversal.

Cada surface somente exibe/aciona; o Quality Orchestrator decide estado.

# Documentation Plane

Documentação agent-facing e human-facing usam Knowledge IR/Document IR compartilhados.

A UI nunca “edita verdade” sem passar por command/reconciliation.

# Experience/Global Learning

Experiência pode propor pattern/skill/policy.

Nunca aplica mudança canônica cross-project automaticamente.

Fluxo:

experience → candidate → evidence/eval → review → accepted pattern/skill/policy.

# Product/repository split

O Harness permanece em raillen/prumo.

Um produto prumo-code separado só deve nascer quando o protocol público permitir construir o client sem importar internal/.

O caderno unificado não desfaz essa boundary; apenas elimina a divisão documental artificial.

# Supersession importante

A antiga decisão “Desktop Rust + Floem” permanece como registro histórico do período exploratório.

A direção corrente para o app nativo é Freya/Workspace Viewer, sujeita aos gates de performance e arquitetura do Livro Vivo.

# Surface Conformance Suite

Para cada command/capability crítica, validar em CLI, TUI e GUI:

- mesma semântica de Run;
- mesmo PermissionResult;
- mesmo BudgetState;
- mesmo Evidence/Gate state;
- mesmo error taxonomy;
- reconnect/replay;
- version negotiation;
- cancel/resume;
- state projection consistency.

O Protocol é a fonte para a semântica; clients não possuem forks comportamentais.

# Definition of Done

- Code Agent é subdomínio documentado do Livro Vivo único;
- nenhuma regra canônica vive apenas em client;
- CLI/TUI/GUI usam services/protocol comuns;
- Freya é tratado como toolkit de UI do client, não Harness;
- Oxi está marcado como direção histórica/superseded;
- Surface Conformance Suite detecta drift;
- protocol é suficiente para construir clients sem internal/.
# 85 — Canonical Truth, Authority Graph e Constituição Anti-Invenção

> Authority: canonical specification.
> Logical ID: 85
> Source: Notion Living Book (3e29bb7d023f8190bd95ce21475ae2e3)
> Status: Constituição 85.A: Execution Contract Grounding, Tools, Evidence e Handoff.


<aside>
🧭

**Status: CONSTITUIÇÃO TRANSVERSAL.** Esta página define como o Prumo distingue verdade, decisão, implementação, evidência, hipótese e desconhecido. Nenhum LLM, agent, CLI, TUI, GUI, plugin ou adapter pode transformar ausência de informação em fato. **Silêncio documental não autoriza invenção.**

</aside>

# Objetivo

O Prumo deve reduzir duas classes de falha típicas de sistemas agentic:

1. **fabrication drift** — o agent inventa APIs, requisitos, paths, estados, comportamentos ou decisões para completar uma tarefa;
2. **authority drift** — uma fonte antiga, secundária ou inferida passa a competir com uma decisão canônica mais recente.

A filosofia deve ser simultaneamente **explícita** em contracts, prompts, manifests e UI, e **implícita** na arquitetura: o caminho feliz do sistema deve tornar difícil inventar e fácil declarar lacuna.

# Regra fundamental

> **O agent pode descobrir, inferir com rótulo, propor e perguntar. Não pode promover inferência a verdade canônica sem autoridade e provenance.**
> 

# Classes de verdade

Todo claim material deve poder ser classificado como:

- USER_DIRECTIVE — instrução explícita vigente do usuário;
- ACCEPTED_DECISION — decisão arquitetural/produto aceita;
- CANONICAL_CONTRACT — schema, protocolo, policy ou documento canônico operacional;
- VERIFIED_IMPLEMENTATION — comportamento observado e comprovado por código + evidence;
- OBSERVED_IMPLEMENTATION — comportamento observado, ainda sem verificação suficiente;
- PLANNED_TARGET — arquitetura/feature aprovada, mas ainda não implementada;
- RESEARCH_FINDING — achado de pesquisa com fonte/provenance;
- HISTORICAL — material válido apenas para história/migração;
- INFERENCE — conclusão plausível derivada, nunca canônica por si só;
- UNKNOWN — informação necessária ausente;
- CONTRADICTED — claims incompatíveis aguardando resolução;
- STALE — claim cuja freshness expirou ou cuja dependência mudou.

# Authority Graph

Autoridade não é uma lista global simples; depende da pergunta.

## Para direção de produto/arquitetura alvo

1. instrução explícita atual do usuário;
2. decisão ACCEPTED não superseded mais recente;
3. contrato arquitetural canônico;
4. pesquisa aprovada como base de decisão;
5. hipótese/inferência.

## Para realidade implementada

1. evidence verificável e testes;
2. código/config/schema realmente presente;
3. documentação operacional canônica reconciliada;
4. planejamento/Notion;
5. inferência.

## Para operação de runtime

Policy/schema/contract executável vence prosa. UI nunca redefine semântica.

# Claim Record mínimo

Campos mínimos: claim_id, statement, truth_class, status, scope, source_refs, source_digests, authority_rank, validity, supersedes, contradicted_by, freshness_policy, implementation_status e confidence somente quando aplicável.

**confidence nunca substitui autoridade.**

# Regra de supersession

Uma decisão nova não apaga silenciosamente a antiga.

Fluxo obrigatório:

old ACCEPTED → new decision → contradiction/supersession analysis → affected surfaces → migration delta → old SUPERSEDED → new ACCEPTED.

Retrieval deve preferir ACCEPTED current e excluir SUPERSEDED do contexto padrão, mantendo-o acessível como histórico.

# Unknown-first protocol

Quando informação necessária estiver ausente:

1. registrar UNKNOWN;
2. identificar por que ela importa;
3. procurar fontes canônicas relevantes;
4. somente se a policy permitir, produzir uma assumption proposal explicitamente rotulada;
5. bloquear mutações irreversíveis ou decisões estruturais quando a lacuna for material;
6. permitir progresso somente em partes independentes da lacuna.

Nunca usar uma suposição como verdade silenciosa.

# AssumptionRecord

Campos: id, question/gap, proposed_value, rationale, risk, affected_tasks, reversible, expires_when, approval_required e status.

Default silencioso só é permitido quando já existe **DefaultPolicy versionada** para aquela classe.

# Proibição de invenção operacional

Agents não podem, sem inspeção/evidence:

- afirmar que arquivo/path existe;
- citar função, símbolo ou API como existente;
- afirmar que teste passou;
- dizer que feature está concluída;
- afirmar suporte de plataforma/formato;
- inventar saldo/quota de provider;
- inventar opção de CLI;
- inventar dependência instalada;
- inventar requirement ausente;
- transformar proposta em decisão;
- transformar screenshot/UI aparente em implementação funcional.

Quando necessário, o agent deve usar search/read/schema/tooling antes de responder ou modificar.

# Implementation Reality Check

Antes de implementar tarefa material:

Task → resolve canonical contract → inspect repository reality → classify planned-vs-implemented drift → build minimal context dossier → implement only authorized delta.

Se documentação afirma A e código mostra B:

- não corrigir automaticamente um lado;
- criar DriftRecord;
- classificar docs stale, implementação incompleta ou contradição real.

# Completion language

Palavras como done, complete, implemented, fixed, verified e ready são estados derivados.

Aplicar Total Assurance:

implemented ≠ reachable ≠ exercised ≠ evidenced ≠ verified ≠ accepted ≠ released.

# Regras por papel

## Planner/Architect

Pode propor decisões. Deve separar proposal de accepted. Não pode usar preferência inferida como requirement.

## Implementer

Só altera dentro de scope/contract. Ao encontrar lacuna material, registra e segue somente o que é independente.

## Reviewer

Não aceita autoavaliação do implementer como proof. Procura drift, false-green, behavior negativo e superfícies esquecidas.

## Researcher

Distingue fonte primária, secundária, hipótese e extrapolação. Research Finding não vira policy automaticamente.

## Documentation Agent

Nunca preenche heading vazio com prosa plausível. Sem authoritative inputs, produz GAP.

## UI/UX Agent

Pode propor interação, nunca afirmar que comportamento backend existe porque há affordance visual.

# Core enforcement

Estas regras devem migrar de prompt para mecanismos:

- schemas;
- status enums;
- authority resolver;
- provenance;
- contradiction engine;
- freshness;
- Context Manifest;
- Assumption/Unknown registries;
- hard gates;
- evidence graph;
- UI badges;
- CLI explainability.

# Context Manifest

Toda execução não trivial deve conseguir expor:

- context_manifest_id;
- goal/task;
- claims selecionados;
- source refs + digests;
- stale/superseded excluídos;
- unknowns;
- assumptions;
- budget;
- skills/agents ativados;
- policies aplicadas.

Isso permite explicar por que o agent “sabia” algo.

# CLI/TUI/GUI requirements

Todas as surfaces devem permitir inspecionar:

- source de decisão/claim;
- estado planned/implemented/verified;
- assumptions/unknowns;
- contradictions;
- evidence;
- budget;
- skills/agents escolhidos;
- reason code de routing.

A GUI/TUI nunca mascara UNKNOWN como texto neutro. Use estado visual explícito.

# Fail-closed vs fail-safe

- security, migration, destructive mutation, release, secrets e permission boundaries: **fail closed**;
- baixa criticidade e leitura: pode cair para generic/conservative path;
- indisponibilidade de Decision Runtime nunca autoriza relaxar hard policy.

# Constituição para prompts de agents

Todo prompt/runtime instruction gerado pelo Prumo deve incorporar semanticamente:

1. inspect before claim;
2. source before authority;
3. distinguish target from current implementation;
4. do not invent missing requirements;
5. label assumptions;
6. abstain/escalate quando uncertainty cruza policy;
7. preserve scope/non-goals;
8. prove completion with evidence.

Não depender de frase fixa. O Core fornece dados e gates que tornam a regra executável.

# Definition of Done

- truth classes e lifecycle existem em schema;
- supersession é resolvida deterministicamente;
- Context Compiler exclui stale/superseded por default;
- gaps viram UNKNOWN, não texto inventado;
- assumptions são explícitas e auditáveis;
- CLI/TUI/GUI exibem provenance e estado;
- reviewer pode reconstruir o contexto usado;
- completion exige evidence;
- nenhuma surface implementa regra paralela de autoridade.

[85.A — LLM & Agent Execution Contract: Grounding, Scope, Tools, Evidence e Handoff](llm-agent-execution-contract.md)
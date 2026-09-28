---
title: Roadmap Oficial (v0.1.0 → v1.0.0)
description: Histórico evolutivo e projeção de desenvolvimento do Prumo Framework até a versão 1.0.
---

# Roadmap Oficial do Prumo (v0.1.0 → v1.0.0)

> **Documento Canônico**: Consulte também o [`ROADMAP.md`](https://github.com/poppy-team/prumo/blob/main/ROADMAP.md) na raiz do repositório para o registro completo de governança e rastreabilidade técnica.

---

## Visão Geral e Princípios Fundamentais

O **Prumo** é um protocolo nativo de engenharia e harness autônomo concebido para pareamento determinístico entre humanos e agentes de IA de código. O projeto é regido por quatro invariantes arquiteturais centrais:

1. **Protocol-Oriented Programming (POP)**: Interações baseadas em contratos tipados com envelopes de máquina verificáveis (`protocol_version`, `ok`, `data`, `diagnostics`, `warnings`).
2. **Lean Progressive Context (LPC)**: Gestão estrita de orçamentos de tokens em camadas progressivas (L0 metadados → L1 sumários → L2 textos completos → L3 símbolos AST → L4 contexto profundo).
3. **Neutralidade de Provedores**: Nenhum modelo ou fornecedor de IA específico é obrigatório ou embutido de forma rígida. As escolhas são dinâmicas, providas pelo workspace ou pelo ambiente do usuário.
4. **Fronteiras Claras entre Superfícies**:
   - **Harness Core & CLI (`prumo`)**: Binário único em Go que mantém a autoridade de protocolo, execução isolada (ACI) e automação headless.
   - **Terminal User Interface (`prumo-tui`)**: Cliente de terminal completo em Go e Bubble Tea v2, com entrega definitiva na versão **v0.7.0**.
   - **Workspace Viewer & IDE Desktop (`prumo-viewer` / `prumo-native`)**: Interface gráfica nativa em Rust construída do zero com **Freya 0.4+** (Skia, Torin, Tree-sitter e Extension SDK para LSP/DAP), com lançamento de produção na versão **v1.0.0**.

---

## Histórico de Versões (v0.1.0 — v0.6.0)

```
2026-08-10      2026-08-15      2026-08-20      2026-09-09      2026-09-10      2026-09-27
   v0.1.0          v0.2.0          v0.3.0          v0.4.0          v0.5.0          v0.6.0 (Atual)
     │               │               │               │               │               │
     ▼               ▼               ▼               ▼               ▼               ▼
  Gênese em       Protocolo       Freeze de       Migração Go     Marca Prumo     CLI Unificada,
  Protótipo     Conformidade       Python        Core Completo    & Doc Control   Adoção v3 &
   Python        & Aprovações     (ADR 001)        (M1 - M4)      Plane (W0-W21)  Roster Dinâmico
```

- **v0.1.0 (Agosto 2026)**: Formalização inicial do LPC e POP; primeiro catálogo de Força de Trabalho (`catalog.json`) com 39 agentes, 189 skills e 20 receitas; ciclo de vida inicial de Metas com lock SHA-256.
- **v0.2.0 (Agosto 2026)**: Introdução dos envelopes de máquina universais; validação JSON Schema (Draft 2020-12); protocolo base de aprovações com human-in-the-loop; fixtures douradas para conformidade.
- **v0.3.0 (Agosto 2026)**: Inventário completo de contratos (`docs/migration/protocol-inventory.md`); freeze da base Python com 127 testes verdes; descoberta de modelos via operação `models`; aceitação dos **ADR 001** e **ADR 002** para migração do Core para Go.
- **v0.4.0 (Setembro 2026)**: Execução das fases M1 a M4 com 100% de paridade; 33 perfis de engenharia de linguagens embutidos; streaming push via operação `subscribe` (ADR 014); inspeção de diffs sob demanda via operação `diff`; spike exploratório TUI H10.
- **v0.5.0 (Setembro 2026)**: Rebranding para Prumo (ADR 004); ondas pré-TUI W0 a W21; Plano de Controle Semântico de Documentação (W15: requisito → afirmação → evidência); runtime do Docs Gauntlet para prevenção contínua de drift; pontes ACP v1 e MCP.
- **v0.6.0 / v0.6.1 (Atual)**: Unificação CLI & Harness no binário único `prumo` (ADR 016); Motor de Adoção Brownfield v3 (`prumo adopt`); gerenciamento de conectores (`prumo connector status/install`); eliminação de listas fixas de modelos em prol de resolução dinâmica; suporte a imagens em anexos `@caminho` (GAP-090); provedor delegado opencode (GAP-093).

---

## Projeção de Desenvolvimento (v0.7.0 — v1.0.0)

```
2026-10 (Meta)            2026-11 (Meta)            2026-12 (Meta)            2027-Q1 (Meta)
    v0.7.0                    v0.8.0                    v0.9.0                    v1.0.0
      │                         │                         │                         │
      ▼                         ▼                         ▼                         ▼
TUI Completo             Gateway Resiliente        DAGs Multi-Agente         Release Final
 (prumo-tui)             & Roteador c/ Cota          & Delegação            & IDE Nativa Freya
```

### v0.7.0 — Cliente de Terminal Completo (`prumo-tui`)
*Meta: Outubro de 2026*

- **Supervisão Transparente**: O comando oficial `prumo agent` (ou `prumo tui`) detecta sockets locais existentes e inicializa o daemon em segundo plano apenas quando necessário, evitando conflitos de processos (GAP-086).
- **Gestão de Múltiplas Sessões**: Navegação entre sessões históricas, retomada contínua e dobra inteligente de timeline com consumo constante de memória (GAP-052, GAP-080).
- **Inspetor de Arquivos e Diffs**: Diálogo interativo de arquivos tocados (`ctrl+g`) com visualização de diffs colorizados sob demanda usando a operação `diff` do protocolo (GAP-084).
- **Paleta de Comandos Customizáveis**: Biblioteca de comandos do usuário em Markdown (`$XDG_CONFIG_HOME/prumo-tui/commands/*.md`) com suporte a preenchimento de variáveis `{{arg}}` (GAP-082).
- **Inspeção de Recursos de Modelos**: Emblemas visuais dinâmicos (`[text reasoning vision tools audio]`) informando exatamente as capacidades de cada modelo registrado (GAP-089).
- **Acessibilidade Plena (WCAG)**: 9 temas com contraste medido acima de 4.5:1 (GAP-070), modo de movimento reduzido (GAP-065), indicadores de foco por glifo (GAP-062) e saída linear acessível para leitores de tela (`--prompt`, `--plain`, `--json`) (GAP-074).
- **Exportação de Timeline**: Exportação determinística da sessão para `.prumo/runtime/exports/<run>.txt` com métricas detalhadas de tokens, cache e custos (GAP-067, GAP-087).

---

### v0.8.0 — Gateway Resiliente & Roteamento Inteligente com Cotas
*Meta: Novembro de 2026*

- **Execução Fidedigna e Códigos de Saída Reais (Onda 0)**:
  - **GAP-097**: Propagação honesta dos códigos de saída de processos executados; testes quebrados falham com erro real em vez de reportar código 0.
  - **GAP-123**: O comando `resume` retoma a execução genuína em vez de marcar prematuramente o run como concluído.
- **Fronteiras de Segurança e Sandboxing (Onda 1)**:
  - **GAP-110**: Contenção rigorosa de links simbólicos, impedindo fuga do diretório do projeto.
  - **GAP-111**: Validação de URLs de provedores contra allowlist para proteger credenciais de vazamentos externos.
  - **GAP-112**: Proteção contra travessia de caminho em identificadores de execução (`run_id`) via `safepath`.
- **Gateway em Produção (Onda 4)**:
  - **GAP-102**: Integração do `internal/harness/gateway` no loop de execução do runtime de produção (ADR 020).
  - **GAP-103**: Recuperação automática do disjuntor de circuito (*circuit breaker*) após expiração do tempo de esfriamento (*cooldown*).
  - **GAP-108**: Backoff exponencial com jitter e respeito ao cabeçalho `Retry-After` para requisições em 429 (`RESOURCE_EXHAUSTED`).
- **Orçamentos e Ciclo do Daemon**:
  - **GAP-098 & GAP-100**: Tetos rígidos de custo no daemon e verificação de pré-voo (*preflight*) antes de disparar chamadas LLM.
  - **GAP-104 & GAP-105**: Trava atômica de PID via `O_EXCL`/`flock` e encerramento gracioso (*drain*) ao receber sinal de término.
  - **GAP-106 & GAP-107**: Recarga persistente de decisões de aprovação por fingerprint de ação/recurso entre reinícios do daemon.

---

### v0.9.0 — Orquestração Multi-Agente em DAG & Delegação Bounded
*Meta: Dezembro de 2026*

- **Equipes de Agentes em Produção**:
  - **GAP-003 & GAP-101**: Ativação do executor de times (`team.Runner` / `RunWork`) na CLI e daemon em produção, isolando orçamentos e checkpoints por papel.
- **Delegação de Subagentes em 1 Nível**:
  - **ADR 017, GAP-022, GAP-109**: Disponibilização das ferramentas `agent.delegate` e `agent.ask` com profundidade estrita de 1 nível para evitar loops de custo recursivos descontrolados.
- **Isolamento via Git Worktrees**:
  - **GAP-008 & GAP-168**: Criação de worktrees Git efêmeras e isoladas para subagentes com verificação de merge de três vias antes de consolidar no branch principal.
- **Protocolo de Handoff entre Provedores**:
  - **Fase M9 / GAP-135**: Transferência tipada de estado e sumários de sessão entre diferentes modelos e harnesses (Claude Code ↔ Codex ↔ Modelos Locais).
- **Resolução de Workforce por Impacto**:
  - **GAP-146 & DOC-GAP-024**: Resolução contextual de habilidades baseada na análise de impacto semântico da Meta em execução.

---

### v1.0.0 — Lançamento Oficial: Prumo Native Workspace Viewer (IDE em Rust) & Ecossistema Maduro
*Meta: 1º Trimestre de 2027*

- **Prumo IDE / Native Workspace Viewer (`prumo-native` / `prumo-viewer`)**:
  - **Construção Nativa em Rust com Freya 0.4+**: Aplicativo desktop leve com renderização acelerada por GPU/CPU via Skia e motor de layout Torin.
  - **Foco Agent-Aware (CONST-82)**: Visualizador e editor leve voltado especificamente para observabilidade, inspeção de alterações e pequenas intervenções humanas enquanto os agentes trabalham, sem a sobrecarga de uma IDE pesada em Electron.
  - **Editor de Código com Tree-sitter**: Realce sintático e inspeção de AST para Rust, Go, JSON, YAML, Markdown, JS, TS e TOML.
  - **Visualizador de Diffs e Terminal PTY**: Comparador de alterações inline e lado a lado via `similar`; terminal virtual embutido via `portable-pty` e `vt100`.
  - **SDK Nativo de Extensões (`prumo-extension-sdk`)**: Barramento seguro para plugins de linguagem via LSP (Language Server Protocol) e depuradores via DAP (Debug Adapter Protocol).
- **Living Plan (Fase M6)**:
  - Motor de entrevista conversacional iterativa para levar projetos do zero ao estado pronto para implementação sem necessidade de prompts manuais gigantescos.
  - Atualização documental incremental automática a partir de decisões de arquitetura aprovadas.
- **Daemon Global e Ecossistema de Conectores (Fases M10/M11, ADR 018)**:
  - Serviço de sistema no nível do usuário gerenciando roteamento de modelos, cofre seguro de chaves e executores isolados por repositório.
- **Grafo Tipado de Rastreabilidade Ponta a Ponta (Fase M8)**:
  - Navegação bidirecional completa conectando Requisitos ↔ Decisões ↔ Código ↔ Testes ↔ Documentação ↔ Evidências (`prumo trace <referencia>`).
- **Distribuição Enterprise**:
  - Binários estáticos universais assinados digitalmente para Linux, macOS e Windows.
  - Publicação nos gerenciadores de pacotes Homebrew, WinGet, Scoop e Arch AUR.

---

> **Plano Granular Completo**: Para o detalhamento minucioso por arquivos, pacotes de entrega e análise comparativa contra Claude Code, OpenCode e Aider, consulte o [**Plano Granular de Patches**](/granular-plan).

## Tabela de Implementação & Comparativo de Versões

| Versão | Data-Alvo | Foco Principal | Stack Tecnológica | Entregáveis Principais | Critério de Aceite (DoD) |
|---|---|---|---|---|---|
| **v0.1.0** | 2026-08-10 | Protótipo Conceitual | Python 3.11+ | POP, LPC, Catálogo de Força de Trabalho (189 skills, 39 agentes, 20 receitas), Metas v1 | Metas e catálogo funcionando na base inicial Python |
| **v0.2.0** | 2026-08-15 | Baseline de Conformidade | Python 3.11+ | Envelopes de máquina padronizados, JSON Schema Draft 2020-12, protocolo inicial de aprovações | Fixtures douradas de teste de regressão aprovadas |
| **v0.3.0** | 2026-08-20 | Freeze de Migração (M0) | Python 3.11+ | Inventário de protocolo, descoberta de modelos (`models`), decisão de migração Go (ADR 001) | 127 testes Python passando; blueprint de migração Go aprovado |
| **v0.4.0** | 2026-09-09 | Migração do Core Go | Go 1.22+ | M1 a M4 concluídas, CLI em binário único, streaming push (`subscribe`), diffs sob demanda (`diff`) | 100% de paridade em fixtures comparativas Go vs Python |
| **v0.5.0** | 2026-09-10 | Plano de Controle Documental | Go 1.22+ | Rebranding para Prumo, Ondas W0 a W21, Plano Semântico (W15), Docs Gauntlet | Validador semântico com relatório `ready: true` verificado |
| **v0.6.0** | 2026-09-27 | Unificação CLI (Atual) | Go 1.26 | Binário unificado `prumo` (ADR 016), Adoção v3 (`prumo adopt`), Conectores, Resolução Dinâmica de Modelos | Execução única zero-cerimônia; CI 100% verde |
| **v0.6.1** | 2026-09-28 | Ergonomia da CLI | Go 1.26 | Autodetecção de stack, `prumo init` com dashboard cognitivo, fix de autoupdate | Zero-cerimônia comprovada em repositórios limpos |
| **v0.6.2** | 2026-10-05 | Onda 0: Códigos de Saída | Go 1.26 | Exit codes reais de subprocessos (GAP-097), retomada honesta de sessão (GAP-123) | Testes quebrados falham fechado com código != 0 |
| **v0.6.3** | 2026-10-10 | Onda 1: Contenção de Segurança | Go 1.26 | Resolução de links simbólicos (GAP-110), allowlist de provedores (GAP-111), safepath (GAP-112) | Fugas de sandbox rejeitadas em testes |
| **v0.7.0** | 2026-10-15 | Cliente TUI Completo | Go, Bubble Tea v2 | `prumo-tui` auto-suficiente, multi-sessões, paleta de comandos (`commands/*.md`), diffs (`ctrl+g`), A11y total | Navegação completa comprovada em todos os 22 estados de UI |
| **v0.7.1** | 2026-10-25 | Ergonomia TUI & Telemetria | Go, Bubble Tea v2 | Accordions colapsáveis de `<think>` (`Ctrl+O`), cards paginados, Cache Hit Ratio % | Snapshot tests de frame verificados |
| **v0.7.2** | 2026-11-05 | Git-Nativo & Web no TUI | Go, Git CLI | `/undo` atômico, proteção de dirty tree, auto-commit convencional, `/web <url>` | `/undo` reverte commit anterior de forma limpa |
| **v0.7.3** | 2026-11-15 | Repo Map Tree-sitter | Go, Tree-sitter | AST do Tree-sitter + PageRank personalizado em orçamento estrito LPC | Salto de 40% na precisão de navegação multi-arquivo |
| **v0.8.0** | 2026-11-25 | Gateway Resiliente & Cotas | Go 1.26 | Gateway ativo em produção (GAP-102/103/108), tetos orçamentários (GAP-098/100), trava atômica PID (GAP-104) | Disjuntor auto-recupera; bloqueio em teto de custo |
| **v0.8.1** | 2026-12-05 | Failover Multi-Provedor | Go 1.26 | Roteamento por prioridade (Local/Free → Cloud) com failover automático por cota | Transição transparente sem queda de sessão |
| **v0.8.2** | 2026-12-15 | Local Intel & Offline | Go, llama/vLLM | Conectores para Ollama, vLLM e llama-server com diffs de busca/substituição | Execução autônoma end-to-end desconectado |
| **v0.9.0** | 2026-12-22 | DAGs Multi-Agente & Times | Go 1.26, Worktrees | Execução de times (GAP-003/101), subagentes de 1 nível (ADR 017), worktrees efêmeras (GAP-008/168) | Metas distribuídas em worktrees com merge de 3 vias |
| **v0.9.1** | 2027-01-10 | Dual Architect / Editor | Go 1.26 | Modo Arquiteto (`/architect`, read-only) + Editor (`/code`, write tools) | >50% de redução de custo de API em refatoração |
| **v0.9.2** | 2027-01-20 | Verification Repair Loop | Go 1.26 | Feedback automático de erros de testes/lints com $K$ tentativas de auto-reparo | Autocorreção autônoma de falhas de sintaxe |
| **v1.0.0-alpha**| 2027-01-30 | IDE Desktop em Rust | Rust 2024 (Freya 0.4+) | Prumo Native Workspace Viewer (`prumo-viewer`), Skia, Torin, Tree-sitter, PTY | Inicialização sub-100ms e observabilidade em tempo real |
| **v1.0.0-beta** | 2027-02-15 | Extension SDK & Living Plan | Rust, Go | `prumo-extension-sdk` (LSP/DAP), motor do Living Plan (Fase M6) | Conformance suite de extensões 100% verde |
| **v1.0.0-rc1/2**| 2027-03-01 | Daemon Global & Rastreabilidade| Go, Rust | Daemon global de sistema (ADR 018), Grafo de Rastreabilidade ponta a ponta (M8) | Gauntlet estrito com zero advertências |
| **v1.0.0** | 2027-03-15 | Lançamento Oficial (GA) | Multi-plataforma | Binários estáticos universais assinados, pacotes Homebrew, WinGet, Scoop, AUR | Disponibilidade geral pública oficial |

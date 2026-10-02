---
title: Plano Granular de Implementação por Patches
description: Cronograma granular de versões intermediárias e análise comparativa de CLI agents de ponta.
---

# Plano Granular de Implementação e Evolução por Patches (v0.6.0 → v1.0.0)

> **Documento Canônico**: Consulte também [`ROADMAP.md`](https://github.com/poppy-team/prumo/blob/main/ROADMAP.md) e [`docs/development/granular-implementation-plan.md`](https://github.com/poppy-team/prumo/blob/main/docs/development/granular-implementation-plan.md).

---

## 1. Pesquisa & Benchmark Competitivo de CLI Code Agents

Para garantir que o **Prumo TUI (`prumo-tui`)** e o **Harness Core (`prumo`)** ofereçam a melhor experiência de engenharia para desenvolvedores do mundo, analisamos minuciosamente as principais plataformas de ponta:

1. **Claude Code (Anthropic)**: O estado da arte em ergonomia de REPL de terminal, micro-compactação aos 70–75% da janela de contexto, telemetria financeira em tempo real ($/req, tokens de cache), blocos de raciocínio colapsáveis e suspensão segura de sessão.
2. **OpenCode & Crush (Anomaly / Charm Go lineage)**: A referência em arquitetura desacoplada cliente-servidor (IPC por socket/HTTP), modularidade de ferramentas com tool guards determinísticos, streaming via Charm v2 (Bubble Tea v2 / Lip Gloss v2) e gestão limpa de sessões.
3. **Aider (`aider-chat`)**: O pioneiro em engenharia Git-nativa: mapeamento de repositório via AST do Tree-sitter + PageRank personalizado, commits atômicos automáticos convencionais, rede de segurança com comando `/undo`, padrão dual-agent (Architect vs. Editor) e loops fechados de lint/teste.
4. **Cline & Roo Code (com Freebuff)**: O ecossistema líder em modos especializados de personas (`.roomodes`: Architect, Code, Ask, Debug) com whitelists estritas de ferramentas por modo, suporte extensivo a Model Context Protocol (MCP) e proteções pré-execução.
5. **OpenAI Codex / Operator CLI & Command-Code**: Padrões de terminal com sub-tarefas assíncronas, revisores de diffs nativos (`/review`), isolamento em sandbox e buffers de histórico de mensagens.
6. **Antigravity CLI / Gemini CLI**: O modelo unificado de harness de terminal e IDE, com orquestração de subagentes especializados, habilidades modulares (Agent Skills) e governança corporativa de ferramentas.

---

### Matriz Comparativa de Recursos de Mercado

| Dimensão | Claude Code | OpenCode / Crush | Aider | Cline / Roo Code | **Prumo TUI (`prumo-tui`)** |
| :--- | :--- | :--- | :--- | :--- | :--- |
| **Linguagem / Runtime** | Node.js (Ink TUI) | TS / Go (Charm v2) | Python (`prompt_toolkit`) | TypeScript (VS Code / CLI) | **Go nativo (Charm v2, binário estático zero-dependências)** |
| **Arquitetura de Processo** | Monolito local interativo | Cliente / Daemon IPC | Monolito interativo local | Extensão / CLI em Node | **Cliente TUI desacoplado sobre Daemon `agentd` (ADR 013/018)** |
| **Independência de Provedor** | Acoplado (Anthropic) | Agnóstico multi-provedor | 100% Agnóstico (LiteLLM) | Agnóstico multi-provedor | **100% Agnóstico (Dynamic Roster, sem listas fixas)** |
| **Mapeamento de Repositório** | Leitura de `CLAUDE.md` + busca | Hooks de contexto + busca | **Tree-sitter AST + PageRank** | Embeddings / Busca semântica | **LPC + Repo Map Tree-sitter + PageRank (v0.7.3)** |
| **Segurança Git** | Prompts de terminal | Histórico interno | **Auto-commit + `/undo` + dirty check** | Histórico de arquivos em disco | **Auto-commit convencional + `/undo` atômico (v0.7.2)** |
| **Padrões de Agente** | Agente único com sub-threads | Subagentes separados | **Dual: Architect + Editor** | **Multi-mode (`.roomodes`)** | **Subagentes 1-nível em Worktree Git (ADR 017, v0.9.0)** |
| **Verificação de Código** | Bash prompts manuais | Hooks de sessão | **Auto-lint + Auto-test repair loop** | Execução de comandos no terminal | **Verification-Guided Repair Loop com cards visuais (v0.9.2)** |
| **Raciocínio (`<think>`)** | Accordion colapsável (`Ctrl+O`) | Bloco formatado com tempo | Transcrito no chat | Bloco colapsável na UI | **Accordion colapsável com duração/tokens (v0.7.1)** |
| **Telemetria de Custo** | Tokens in/out/cache + USD | Tokens totais | Tokens totais + custo | Custo USD e contagem de tokens | **Real-time in/out/cache tokens, USD, ~$USD/req e Hit Ratio %** |
| **Inspeção de Diffs** | Inline unified diff (`/diff`) | Diálogo de arquivos | Diffs de busca/substituição | Diffs interativos inline | **Unified & Side-by-Side Chroma highlight sob demanda (GAP-084)** |

---

## 2. Especificação de Recursos Adotados para o Prumo TUI

### A. UI/UX & Design de Terminal
1. **Accordions de Raciocínio Colapsáveis (`<think>`)**:
   - Modelos como DeepSeek R1, Claude 3.7 Sonnet e OpenAI o1/o3-mini emitem longos fluxos de raciocínio antes do código.
   - O `prumo-tui` encapsula esses deltas em um widget colapsável animado:
     - *Estado colapsado*: `▶ Pensando (12.4s · 850 tokens) — Ctrl+O para expandir`
     - *Estado expandido*: Texto em itálico com bordas atenuadas exibindo todo o raciocínio.
     - *Atalho global*: `Ctrl+O` alterna a visibilidade de todos os blocos de pensamento da sessão.
2. **Cards de Execução de Ferramentas Paginados**:
   - Em vez de poluir a tela com saídas brutas de 500 linhas de comandos bash ou testes, as saídas são truncadas em até 10 linhas no corpo da mensagem.
   - Um rodapé interativo permite expandir: `[Enter para expandir 48 linhas · Esc para fechar]`, abrindo a saída completa em um modal scrollável.
3. **Telemetria Completa na Statusline**:
   - Exibição em tempo real de tokens de entrada (`in`), tokens de leitura de cache (`cache`), tokens de saída (`out`), custo financeiro estimado acumulado (`$`) e taxa de acerto de cache (*Cache Hit Ratio %*):
     $$\text{Hit Ratio} = \frac{\text{CacheReadTokens}}{\text{PromptTokens} + \text{CacheReadTokens}} \times 100$$
   - Exibição de gauge de pressão de contexto: `ctx: 45K/128K [35%]`, mudando de cor (verde <70%, âmbar 70-84%, vermelho >85%).
4. **Ergonomia Modal & Chords**:
   - Suporte a navegação Vim opcional no transcript (`j`/`k` para rolar, `/` para pesquisar no histórico).
   - Suporte a `$EDITOR` externo via `Ctrl+E`: suspensão limpa do TUI via `tea.Suspend()`, abrindo Neovim/Helix/VS Code para rascunhar prompts complexos e retornando ao TUI ao salvar.
5. **Multimodalidade & Colar Imagens (`Ctrl+V` / `/paste`)**:
   - Captura direta da área de transferência do sistema via utilitários nativos (`wl-paste` no Wayland, `xclip` no X11, `pbpaste` no macOS).
   - Armazenamento em `.prumo/cache/media/<hash>.png` e renderização de prévia visual no terminal em meio-bloco ANSI de 24-bit (`▀`).
6. **Sistema de Tecla Líder (*Leader Key* — `Ctrl+X`)**:
   - Para erradicar conflitos com atalhos padrão de terminal GNU Readline (`Ctrl+A`, `Ctrl+E`, `Ctrl+K`), o `prumo-tui` adota a convenção de Tecla Líder do OpenCode/tmux:
     - `Ctrl+X` seguido de `N`: Iniciar Nova Sessão.
     - `Ctrl+X` seguido de `L`: Listar e alternar entre sessões ativas.
     - `Ctrl+X` seguido de `U`: Desfazer (*undo*) a última mutação do agente via checkpoint Git.
     - `Ctrl+X` seguido de `R`: Refazer (*redo*).
     - `Ctrl+X` seguido de `C`: Compactar contexto manualmente (*rolling capsule*).
     - `Ctrl+X` seguido de `M`: Troca dinâmica de modelo/provedor.
     - `Ctrl+P` / `Ctrl+K`: Paleta global de comandos com busca fuzzy.
7. **Prefixos Rápidos de Entrada no Composer**:
   - `@<caminho>`: Gatilho de fuzzy completion para anexar caminhos de arquivos e pastas diretamente no prompt.
   - `!<comando>`: Pass-through shell imediato executado no workspace sem disparar inferência LLM (ex: `!git status`, `!go test ./...`), exibindo o resultado em card formatado.
   - `/<comando>`: Execução direta de slash commands integrados (`/undo`, `/model`, `/provider`, `/sidebar`, `/help`).

### B. Ferramentas, Inteligência & Engenharia de Runtime
1. **Pipeline de Ferramentas em 6 Estágios**:
   - `Registry Lookup`: Validação de assinatura e schema de parâmetros.
   - `Permission Gate (Human-in-the-Loop)`: Verificação estrita de políticas declaradas (`internal/harness/perm`), exigindo aprovação prévia para comandos mutantes ou destrutivos com preview de diff.
   - `Pre-Tool Hooks`: Registro de auditoria e interceptores de plugins.
   - `Execution`: Execução contida com captura assíncrona de streams stdout/stderr.
   - `Post-Tool Hooks`: Verificação de exit codes e emissão de eventos para a timeline.
   - `Output Compaction`: Truncamento inteligente e paginação de saídas volumosas para evitar context blowouts.
2. **Inteligência Semântica via LSP Nativo (`internal/harness/lsp`)**:
   - Acoplamento do agente aos Language Servers instalados no ambiente (`gopls`, `rust-analyzer`, `tsserver`, `pyright`).
   - Fornece operações semânticas: `goToDefinition`, `findReferences` e `getDiagnostics` (capturando alertas e erros do compilador antes mesmo da execução de suítes de testes lentas).
3. **Persistência Multi-Sessão & Branching em SQLite**:
   - Armazenamento em `.prumo/state/sessions.db` com recuperação instantânea de checkpoints, histórico completo de turnos e bifurcação (*branching*) de conversas a partir de qualquer ponto do histórico.
4. **Trindade Unificada de Modos Operacionais**:
   - **TUI Interativo**: `prumo tui` (interface full-screen rica em Bubble Tea v2).
   - **One-Shot CLI**: `prumo agent "<prompt>" -q` ou `prumo run "<prompt>"` (para scripts, pipes UNIX e automações CI/CD).
   - **Daemon / RPC Server**: `prumo daemon` (supervisão de background servindo TUI, IDEs e UIs remotas).
5. **Comando `/undo` Atômico**:
   - Desfaz de forma limpa o último turno executado pelo agente, revertendo as alterações no sistema de arquivos e restaurando o commit anterior via Git (`git reset --hard HEAD~1`).
6. **Proteção de Repositório Sujo (*Dirty Working Tree*)**:
   - Se o usuário tiver modificações manuais não salvas antes de o agente iniciar o trabalho, o TUI apresenta um aviso de segurança imediato com opções: `[s] Criar Stash temporário | [c] Realizar Commit prévio | [a] Abortar`.
7. **Auto-Commit Convencional**:
   - Conclusão de metas com commit automático estruturado conforme `internal/repositorypolicy` (`feat(escopo): resumo da meta`).
8. **Repo Map com Tree-sitter & PageRank**:
   - Extração estrutural de definições e assinaturas via AST do Tree-sitter sem corpos de funções.
   - Aplicação de PageRank personalizado para ordenar os arquivos centrais do projeto e encaixá-los de forma ótima dentro de um orçamento estrito de tokens (1024 a 2048 tokens).
9. **Modo Dual Architect / Editor (`Ctrl+M`)**:
   - **Modo Arquiteto**: Modelo de alto raciocínio (Claude 3.7 / o1 / DeepSeek R1) opera estritamente em modo somente-leitura, analisando o repo map e emitindo um plano estruturado de tarefas.
   - **Modo Editor**: Modelo econômico e rápido (Claude Haiku / Qwen 2.5 Coder) assume o plano e executa as edições cirúrgicas no código.
10. **Loop de Reparo Guiado por Verificação (Lints & Testes)**:
    - Configuração de comandos de validação (`test_cmd`, `lint_cmd`). O TUI exibe o progresso:
      ```
      ● Verificando alterações...
        ✔ Linter (golangci-lint): Passou (0.4s)
        ✖ Testes (go test ./...): Falhou (2 falhas)
        ◌ Auto-reparando com o agente: Tentativa 1 de 3...
      ```
    - O erro do teste é reinjetado como observação sintética em até $K$ tentativas limitadas.
11. **Destilação Web (`/web <url>`)**:
    - Comando `/web <url>` busca a página externa, extrai o conteúdo limpo via Readability (eliminando menus, anúncios e scripts) e injeta uma cápsula Markdown imutável no contexto.

---

## 3. Cronograma Granular de Implementação por Versões de Patch

```
v0.6.x (Hardening)      v0.7.x (TUI de Produção)      v0.8.x (Gateway & Resiliência)     v0.9.x (Multi-Agente)          v1.0.x (Release Oficial & IDE)
├── v0.6.0 (Concluído)  ├── v0.7.0 (Out 2026)          ├── v0.8.0 (Nov 2026)              ├── v0.9.0 (Dez 2026)          ├── v1.0.0-alpha (Jan 2027)
├── v0.6.1 (Concluído)  ├── v0.7.1 (Ergonomia TUI)     ├── v0.8.1 (Failover Provedor)     ├── v0.9.1 (Dual Architect)    ├── v1.0.0-beta (Fev 2027)
├── v0.6.2 (Onda 0 Fix) ├── v0.7.2 (Git & Web no TUI)  └── v0.8.2 (Modelos Locais)        └── v0.9.2 (Verification Loop)├── v1.0.0-rc1/rc2 (Mar 2027)
└── v0.6.3 (Onda 1 Fix) └── v0.7.3 (Tree-sitter Repo)                                                                    └── v1.0.0 GA (Março 2027)
```

---

### Linha v0.6.x — Hardening do Core & Fechamento de Fugas

#### v0.6.0 (27/09/2026) — Concluído ✅
- **Foco**: Unificação CLI & Harness no binário `prumo` (ADR 016), Adoção Brownfield v3 (`prumo adopt`), Conectores e Roster Dinâmico sem modelos hardcoded.
- **Entregáveis**: `cmd/prumo/main.go`, `internal/adoption/`, `internal/connectors/`, resolução dinâmica de modelos (`PRUMO_MODEL`/`PRUMO_PROVIDER`).

#### v0.6.1 (28/09/2026) — Concluído ✅
- **Foco**: Ergonomia de comandos da CLI, autodetecção profunda de stack, zero-cerimônia `prumo init` com dashboard cognitivo interativo, fix no repositório do autoupdate (`poppy-team/prumo`) e deploy do portal de documentação.
- **Entregáveis**: `internal/cliops/`, `ROADMAP.md`, `docs/`, suite de testes de integridade semântica.

#### v0.6.2 (Previsão: Início de Outubro 2026) — Onda 0: Fidelidade de Execução
- **Foco**: Correção imediata dos gaps de honestidade de execução e códigos de saída.
- **Entregáveis**:
  - **GAP-097**: `internal/harness/runtime/runtime.go`: Propagação real do exit code de processos; remoção de pipes intermediários que mascaravam erros em exit code 0.
  - **GAP-123**: `internal/harness/runtime/resume.go`: Correção na máquina de estados para que `resume` retome a execução genuína de tarefas pendentes em vez de marcar o run como concluído.
  - Testes de regressão: `TestFailingSubprocessExitsNonZero` e `TestResumeDoesNotPrematurelyComplete`.
- **DoD**: Comandos e testes falhos geram saída falha fechada comprovada em CI.

#### v0.6.3 (Previsão: Outubro 2026) — Onda 1: Contenção de Segurança e Sandboxing
- **Foco**: Blindagem do sistema de arquivos e isolamento de credenciais.
- **Entregáveis**:
  - **GAP-110**: `internal/harness/safepath/`: Resolução canônica de links simbólicos antes de mutações de arquivo, impedindo escape para diretórios do sistema (ex: `/etc`).
  - **GAP-111**: `internal/harness/security/`: Verificação estrita de URLs de provedores contra allowlist declarada, impedindo vazamento de chaves para endpoints de terceiros.
  - **GAP-112**: Saneamento de identificadores de execução (`run_id`) contra path traversal (`../`).
  - Testes de segurança: `TestSymlinkEscapeRejected`, `TestUntrustedEndpointRefused` e `TestRunIDTraversalRejected`.
- **DoD**: 100% dos testes de escape de sandbox e injeção de caminho passam com rejeição explícita.

---

### Linha v0.7.x — O Cliente de Terminal Completo (`prumo-tui`)

#### v0.7.0 (15/10/2026) — Entrega Canônica do Prumo TUI
- **Foco**: Lançamento oficial do `prumo-tui` como cliente de terminal autocontido, padronizado e com ergonomia inspirada no OpenCode.
- **Entregáveis**:
  - Binário dedicado `prumo-tui` empacotado em `prumo-tui/cmd/prumo-tui`, inicializado de forma transparente via `prumo agent` ou `prumo tui` (com resolução canônica de caminho de executável).
  - **Sistema de Tecla Líder (`Ctrl+X`)**: Ativação de atalhos mnemônicos do OpenCode (`Ctrl+X N` nova sessão, `Ctrl+X L` listar, `Ctrl+X U` undo, `Ctrl+X M` modelo) sem conflitos com GNU Readline.
  - **Prefixos de Entrada no Composer**:
    - `@`: Autocomplete fuzzy de arquivos e diretórios integrado ao input.
    - `!`: Pass-through shell imediato sem acionar o modelo (ex: `!git status`), exibindo a saída em card limpo.
    - `/`: Slash commands rápidos (`/help`, `/model`, `/provider`, `/sidebar`, `/init`).
  - **Trindade Operacional**: Paridade total entre modo interativo (`prumo tui`), CLI one-shot com silenciamento de animações (`prumo agent "<prompt>" -q` / `prumo run`) e daemon em background (`prumo daemon`).
  - Supervisão sem colisão de daemons locais existentes (GAP-086).
  - Gestão de múltiplas sessões com dobra de timeline em memória constante (GAP-052, GAP-080).
  - Inspetor de arquivos tocados (`Ctrl+G`) com visualização de diffs colorizados via operação `diff` (GAP-084).
  - Paleta de comandos do usuário em Markdown (`commands/*.md`) com placeholders `{{arg}}` (GAP-082).
  - Emblemas dinâmicos de capacidade de modelo (`[text reasoning vision tools audio]`, GAP-089).
  - Acessibilidade WCAG completa: 9 temas com contraste mínimo de 4.5:1 (GAP-070), modo de movimento reduzido (GAP-065), indicadores de foco por glifo (GAP-062) e modo de tela linear para leitores de tela (`--prompt`, `--plain`, `--json`, GAP-074).
  - Exportação de histórico em texto puro com divisão de tokens de cache e custos monetários (GAP-067, GAP-087).
- **DoD**: Navegação verificada ponta a ponta nos 22 estados de interface documentados em `docs/ui-ux/state-matrix.json` e execução de comandos `!` e `/` comprovada por testes unitários.

#### v0.7.1 (25/10/2026) — Ergonomia Avançada de Terminal, LSP & Telemetria
- **Foco**: Experiência visual refinada, monitoramento em tempo real e inteligência semântica via LSP.
- **Entregáveis**:
  - **Inteligência Semântica com LSP**: Integração do `internal/harness/lsp` no loop do TUI (`goToDefinition`, `findReferences` e `getDiagnostics`), exibindo diagnósticos do compilador antes da execução.
  - **Accordions de Pensamento (`<think>`)**: Componente colapsável interativo no `message.go` com cálculo de tempo decorrido e atalho `Ctrl+O`.
  - **Cards de Ferramenta Paginados**: Rodapé interativo `[Enter para expandir]` em saídas com mais de 10 linhas.
  - **Telemetria de Cache Hit Ratio**: Exibição da porcentagem de acerto de cache (`cache 82%`) na statusline.
  - **Gauge de Contexto**: Indicador visual de ocupação do contexto do modelo (`ctx: 35%`).
  - **Navegação Vim**: Modo normal opcional com teclas `j`/`k`/`gg`/`G` para navegação fluida no histórico de mensagens.
- **DoD**: Testes de frame snapshot atualizados confirmando renderização dos novos widgets colapsáveis e ferramenta de diagnóstico LSP validada em Go e Rust.

#### v0.7.2 (05/11/2026) — Ferramental Git-Nativo & Web no Terminal
- **Foco**: Operações de Git integradas e captura de contexto externo.
- **Entregáveis**:
  - **Comando `/undo`**: Reversão atômica do último turno do agente via `git reset --hard HEAD~1` e restauração de checkpoint.
  - **Verificação de Árvore Suja**: Prompt interativo ao iniciar edições sobre repositório com alterações não comitadas (`[s] Stash | [c] Commit | [a] Abort`).
  - **Auto-Commit Pós-Meta**: Criação automática de commits convencionais após passagem nos gates de verificação.
  - **Comando `/web <url>`**: Destilação de URLs externas em cápsulas limpas de Markdown sem scripts ou anúncios.
  - **Colar Imagens via `Ctrl+V`**: Suporte a captura da área de transferência nativa (`wl-paste`/`xclip`/`pbpaste`) com prévia em meio-bloco ANSI de 24-bit.
- **DoD**: Teste unitário e de integração comprovando reversão com `/undo` e leitura de imagem do clipboard.

#### v0.7.3 (15/11/2026) — Repo Map via Tree-sitter & PageRank
- **Foco**: Otimização máxima de contexto estrutural do repositório baseado em LPC.
- **Entregáveis**:
  - Reescrita do `internal/harness/repomap` utilizando as gramáticas de AST do Tree-sitter em Go.
  - Algoritmo de PageRank personalizado sobre o grafo de dependências e símbolos do repositório.
  - Injeção dinâmica da espinha estrutural otimizada por busca binária dentro do teto de tokens do modelo (`--map-tokens`).
- **DoD**: Testes de benchmark demonstrando aumento de 40% na precisão de localização de arquivos em tarefas complexas com menos de 2.000 tokens de contexto.

---

### Linha v0.8.x — Gateway Resiliente & Roteamento Inteligente com Cotas

#### v0.8.0 (25/11/2026) — Gateway em Produção & Resiliência
- **Foco**: Ativação do motor de roteamento inteligente e controle orçamentário.
- **Entregáveis**:
  - **GAP-102**: Conexão direta de `internal/harness/gateway` ao loop de execução em produção (`r.Svc.Models.Stream`, ADR 020).
  - **GAP-103**: Auto-recuperação de circuito aberto do disjuntor (*circuit breaker*) após o término do período de esfriamento (*cooldown*).
  - **GAP-108**: Backoff exponencial com jitter determinístico e respeito aos cabeçalhos `Retry-After` em erros HTTP 429 (`RESOURCE_EXHAUSTED`).
  - **GAP-098 & GAP-100**: Teto orçamentário rígido imposto pelo daemon e verificação de pré-voo antes de chamadas de inferência.
  - **GAP-104 & GAP-105**: Trava atômica de PID do daemon via `O_EXCL`/`flock` e encerramento gracioso (*drain*) ao receber `SIGTERM`.
  - **GAP-106 & GAP-107**: Recarga persistente de decisões de permissão humana por fingerprint entre reinicializações do daemon.
- **DoD**: Testes de estresse com simulação de 429 comprovam recuperação automática de disjuntor e bloqueio estrito de teto orçamentário.

#### v0.8.1 (05/12/2026) — Portfólio Multi-Provedor com Failover Dinâmico
- **Foco**: Roteamento automático de inferência baseado em prioridade, latência e disponibilidade.
- **Entregáveis**:
  - Regras de roteamento dinâmico: Tentativa primária em modelos locais/gratuitos (Ollama/OpenCode Free) com fallback transparente para modelos de alta capacidade (Claude 3.7 Sonnet / OpenAI o3-mini).
  - Monitoramento de cotas de provedores com transição em milissegundos sem interrupção da sessão.
- **DoD**: Teste de falha de provedor simulado demonstrando fallback com continuidade perfeita do turno.

#### v0.8.2 (15/12/2026) — Local Intel & Suporte a Modelos Locais
- **Foco**: Execução local-first e suporte offline com hardware heterogêneo.
- **Entregáveis**:
  - **GAP-025 & GAP-040**: Conectores para inferência local via `llama-server`, Ollama e vLLM.
  - Calibração de janela de contexto baseada na VRAM/RAM disponível e suporte a parsing de diffs de busca/substituição para modelos abertos menores (7B–14B).
- **DoD**: Teste de execução end-to-end com Ollama local desconectado da internet.

---

### Linha v0.9.x — Orquestração Multi-Agente em DAG & Delegação em Equipe

#### v0.9.0 (22/12/2026) — Equipes de Agentes em Produção & Worktrees Git
- **Foco**: Delegação autônoma em equipe com isolamento estrito.
- **Entregáveis**:
  - **GAP-003 & GAP-101**: Ativação do executor de equipes (`team.Runner` / `RunWork`) na CLI e no daemon em produção.
  - **ADR 017, GAP-022, GAP-109**: Ferramentas de subagentes `agent.delegate` e `agent.ask` com teto fixo de profundidade 1 (`max_delegation_depth: 0` para subagentes, prevenindo loops de custo).
  - **GAP-008 & GAP-168**: Criação automática de worktrees Git efêmeras para execução isolada de subagentes, com validação de merge de 3 vias e detecção de conflitos.
  - **Fase M9 / GAP-135**: Protocolo de handoff com sumários tipados entre diferentes provedores (Claude Code ↔ Codex ↔ Local).
  - **GAP-146 & DOC-GAP-024**: Resolução contextual de habilidades da força de trabalho com base na análise de impacto semântico da Meta.
- **DoD**: Execução de meta complexa distribuída entre subagentes paralelos em worktrees distintas com merge limpo no branch principal.

#### v0.9.1 (10/01/2027) — Padrão Dual Architect / Editor
- **Foco**: Divisão especializada de papéis para maximizar precisão e minimizar custos de API.
- **Entregáveis**:
  - Integração do modo `/architect` e `/code` no TUI com chaveamento por `Ctrl+M`.
  - O Arquiteto opera em modo somente-leitura produzindo especificações detalhadas de arquivo; o Editor implementa as edições via modelos rápidos.
  - Exibição de papéis no cabeçalho do TUI: `[ARQUITETO] Claude 3.7 Sonnet` $\rightarrow$ `[EDITOR] Qwen 2.5 Coder`.
- **DoD**: Redução medida de mais de 50% de custo de tokens em tarefas de refatoração multi-arquivo mantendo taxa de sucesso idêntica.

#### v0.9.2 (20/01/2027) — Loop de Reparo Guiado por Verificação (Test-Time Compute)
- **Foco**: Autocorreção adversarial estrita imposta por ferramentas externas.
- **Entregáveis**:
  - Injeção automática de saídas de erro de compilação, linters e testes unitários na fase `EvaluateStop` do runtime.
  - Ciclo de auto-reparo com limite estrito de $K$ tentativas configuráveis antes de emitir falha.
  - Cards de progresso em tempo real no TUI exibindo etapas de verificação e tentativas de correção.
- **DoD**: Prova de autocorreção em falha induzida de sintaxe em projeto Go e TypeScript sem intervenção humana.

---

### Linha v1.0.x — Lançamento Flagship: Prumo Native Workspace Viewer (Rust IDE GUI) & Ecossistema Maduro

#### v1.0.0-alpha (30/01/2027) — Workspace Viewer Nativo em Rust com Freya
- **Foco**: Primeira versão funcional da interface gráfica desktop integrada.
- **Entregáveis**:
  - Compilação e empacotamento do binário `prumo-native` / `prumo-viewer` em Rust com **Freya 0.4+**, Skia e motor de layout Torin.
  - Visualização de árvore de arquivos em tempo real com debounced watching via `notify`.
  - Editor leve com realce de sintaxe Tree-sitter para 8 linguagens principais e visualizador de diffs inline e side-by-side via `similar`.
  - Terminal embutido baseado em `portable-pty` e `vt100`.
- **DoD**: Aplicação gráfica inicializa em menos de 100ms e acompanha sessões de agentes em tempo real no Linux, macOS e Windows.

#### v1.0.0-beta (15/02/2027) — Extension SDK & Living Plan
- **Foco**: Extensibilidade out-of-process e planejamento iterativo.
- **Entregáveis**:
  - Barramento de extensões `prumo-extension-sdk` com suporte a servidores de linguagem (LSP) e depuradores (DAP) isolados em processos separados com concessão de permissões.
  - Motor do **Living Plan (Fase M6)**: Entrevista interativa conversacional (`prumo plan interview`) que leva projetos do zero ao estado pronto para implementação com atualização documental incremental automática.
- **DoD**: Conformance suite de extensões passa 100% (testes de pacotes, assinaturas e RPC).

#### v1.0.0-rc1 / rc2 (01/03/2027) — Daemon Global de Sistema & Rastreabilidade Ponta a Ponta
- **Foco**: Estabilidade em nível corporativo e governança total.
- **Entregáveis**:
  - Serviço de sistema no nível do usuário (**ADR 018**, Fases M10/M11) com cofre seguro de credenciais e isolamento de executores por projeto.
  - Grafo Tipado de Rastreabilidade Ponta a Ponta (**Fase M8**): Navegação bidirecional completa conectando Requisitos ↔ Decisões ↔ Código ↔ Testes ↔ Evidências (`prumo trace <referencia>`).
  - Verificação rigorosa do Docs Gauntlet (`prumo docs verify --strict`) com zero advertências.
- **DoD**: Relatório completo do Gauntlet verde e builds estáticos assinados para todos os sistemas operacionais.

#### v1.0.0 (15/03/2027) — Disponibilidade Geral (GA) 🚀
- **Foco**: Lançamento oficial de produção do Prumo Framework.
- **Entregáveis**:
  - Disponibilidade pública em gerenciadores oficiais de pacotes: Homebrew, WinGet, Scoop e Arch Linux (AUR).
  - Cobertura completa de documentação bilíngue (Português / Inglês) e contratos 100% aderentes a JSON Schema Draft 2020-12.
- **DoD**: Lançamento oficial aprovado por todos os release gates corporativos.

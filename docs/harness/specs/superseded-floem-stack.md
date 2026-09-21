# 21 — HISTÓRICO/SUPERSEDED: Stack H Floem + Bubble Tea v2

> Authority: canonical specification.
> Logical ID: PART2-CH-21
> Source: Notion Living Book (3d89bb7d023f8151842bf5c40e49c0b0)
> Status: Documento histórico: stack Floem superseded por Bubble Tea v2 e Oxi.


<aside>
⚠️

**Status: SUPERSEDED PARCIALMENTE / HISTÓRICO.** Esta página preserva a decisão Stack H e sua pesquisa como provenance. **Go no Core/Harness e Bubble Tea v2 na TUI permanecem vigentes. A escolha Rust + Floem para Desktop foi supersedida pela decisão mais recente do Livro Vivo: Oxi como base do Prumo Native / Workspace Viewer agent-aware.** Para implementação atual, consultar a página 82 e a Constituição 86. O conteúdo abaixo não deve ser recuperado como decisão desktop vigente sem esse contexto.

</aside>

# Decisão arquitetural

```
Prumo Core / Harness / Daemon     Go
            │
     Agent Client API
            │
    ┌───────┴────────┐
    │                │
Desktop           TUI / CLI
Rust              Go
Floem             Bubble Tea v2
```

A separação é deliberada: **Go concentra domínio, orchestration, runtime, daemon e TUI; Rust é usado onde o ecossistema entrega uma vantagem clara para uma GUI desktop editor-grade**. Desktop e TUI são clientes do mesmo Harness e não duplicam regras canônicas.

# Produto aprovado

Prumo Code será um **Agentic Development Workbench**, agent-first e não IDE-first, mas com capacidades editor-grade. A superfície padrão será Agentic Workspace, complementada por Code Workspace, Review Workspace e Terminal Workspace.

A linguagem visual será **Zed-like**, sem copiar identidade literal: alta densidade, panes limpos, tabs compactas, divisores sutis, command palette, sidebars retráteis, status bar fina e hierarquia baseada em planos/contraste em vez de caixas pesadas.

A interpretação não precisa ser cromaticamente idêntica nas duas superfícies. O **Desktop/Floem** tende a ser mais sóbrio e refinado; a **TUI/Bubble Tea** pode ser propositalmente **mais colorida, divertida e viva**, explorando true color, badges, agent identities, activity states, diffs e status sem cair em excesso de bordas/caixas.

# Desktop — Rust + Floem

## Fundação

- **Floem** — GUI principal;
- **Floem Editor / floem-editor-core** — editor desktop inicial;
- **lapce-xi-rope** — buffer/rope herdado do ecossistema;
- **prumo-ui** — design system próprio e camada canônica de componentes do Prumo.

## Componentes e styling

- **floem-shadcn** — referência e reutilização seletiva de componentes; não definir como API canônica do produto;
- **floem-tailwind** — referência/utilitários de styling, sempre atrás de `prumo-ui`;
- **floem-ui-kit** — REJECTED para caminho crítico: projeto arquivado;
- **mottes-floem-components** — REJECTED para caminho crítico: licença GPL-3.0-only.

## Editor e code intelligence local

- Floem Editor + `floem-editor-core`;
- `lapce-xi-rope`;
- `tree-sitter` + `tree-sitter-highlight` para syntax highlighting incremental;
- **Nucleo** para fuzzy file/symbol/command/run picking.

LSP/DAP permanecem serviços centralizados no Harness; o Desktop apenas consome seus modelos/eventos.

## Terminal

- **wezterm-term** — primeira escolha para modelo/emulação de terminal;
- `alacritty_terminal` — challenger/reference;
- processos PTY e lifecycle de sessões permanecem preferencialmente sob o Harness.

## Markdown, diff e native helpers

- **pulldown-cmark** — Markdown nativo, sem WebView;
- **similar** — inline/buffer diff local quando necessário;
- **rfd** — file/folder dialogs nativos;
- `arboard` — clipboard avançado se Floem não cobrir o requisito;
- **Lucide Icons** — família visual padrão;
- `icondata` — packs adicionais apenas seletivamente;
- `notify` — watcher somente em standalone/fallback local;
- `interprocess` — candidato para local IPC;
- `tokio-tungstenite` — candidato para remote WebSocket client.

# TUI — Go + Bubble Tea v2

## Charm Stack principal

- **Bubble Tea v2** — application/runtime model;
- **Bubbles v2** — componentes oficiais;
- **Lip Gloss v2** — layout/styling/theme;
- **Glamour v2** — Markdown → ANSI;
- **Huh v2** — setup/forms/configuração, não como framework da UI inteira;
- **Wish v2** — P1 para servir o Prumo TUI via SSH.

## Bubbles usados como primitives

- `textinput` — command/search filters;
- `textarea` — prompt/composer e edição leve inicial;
- `viewport` — logs, Markdown, output e traces;
- `list` — runs/agents/tasks;
- `table` — evidence/model/budget views;
- `filepicker` — operações simples;
- `help` — key hints/help;
- `spinner` — tool/agent activity;
- `progress` — context/build/budget progress;
- `paginator` — listas paginadas;
- `cursor` — input primitives.

## Terminal e panes

- `charmbracelet/x/vt` — virtual terminal primitive;
- `charmbracelet/x/xpty` — PTY primitive atrás de uma interface estável do Prumo;
- **Spanreed** — spike/reference para terminal embutível sobre Bubble Tea v2, não dependência canônica inicial;
- **TUIOS** — referência e possível fonte de padrões para BSP panes, terminal sessions, mouse resizing, scrollback e event-driven PTY rendering; não adotar seu window manager inteiro como arquitetura do Prumo.

`charmbracelet/x` é experimental; nenhuma API dessas libs pode vazar para contratos públicos do Prumo.

## Editor TUI

O TUI permanece **agent-first**. No primeiro release, edição complexa não deve bloquear o produto.

```
v0: Bubbles textarea + abrir em editor externo quando necessário
v1: Prumo Editor component próprio
v2: LSP / code actions / advanced editing
```

Para implementação futura, estudar **Chiquito** como referência de piece-table/editor Bubble Tea. Não adotar `glyph/editor` como foundation enquanto permanecer em Bubble Tea v1/limitações Unicode relevantes.

## Syntax e fuzzy

- `github.com/tree-sitter/go-tree-sitter` — Tree-sitter no cliente Go;
- `sahilm/fuzzy` — primeira opção para fuzzy matching local simples;
- filtros/índices mais complexos podem ser fornecidos pelo Harness.

# Serviços compartilhados no Harness Go

Essas capacidades **não** devem ser implementadas separadamente em Floem e Bubble Tea:

- LSP lifecycle e multiplexação — `go.lsp.dev/protocol` + `go.lsp.dev/jsonrpc2` como candidatos principais;
- DAP — `google/go-dap`;
- Git — system Git como autoridade;
- patch/diff parsing — `go-gitdiff`;
- workspace watching — `fsnotify/fsnotify` + `WorkspaceWatcher` próprio;
- PTY/process/session lifecycle — provider próprio, com `x/xpty` como primitive candidata;
- Agent Runtime, Models, Context, Tools/ACI, Permissions, Sandbox, Evidence e Gates;
- remote protocol/server — `coder/websocket` como candidato Go;
- canonical Run/Goal/Task/Workspace state.

# Strategic challengers

## GoGPU/UI

Status: **WATCH**. Reavaliar se o editor amadurecer, IME/Unicode/a11y e software backend forem comprovados, houver aplicação IDE-grade real e a API reduzir churn. A arquitetura do Agent Protocol deve tornar uma eventual migração Floem → GoGPU possível sem tocar no Harness.

## R3BL

Status: **WATCH**. Reavaliar se a TUI passar a exigir um editor/PTY multiplexer muito mais sofisticado do que Bubble Tea consegue entregar pragmaticamente. Hoje a decisão agent-first reduz esse benefício marginal.

# Invariantes de implementação

- uma engine, múltiplos clientes;
- semantic design tokens compartilhados, widgets não compartilhados;
- nenhuma API de toolkit pode virar contrato do Harness;
- nenhuma lógica canônica de Agent/Goal/Plan/Evidence fica na UI;
- Desktop e TUI devem poder evoluir independentemente;
- hardware modesto continua sendo performance target;
- libraries experimentais ficam sempre atrás de adapters próprios;
- licenças, supply chain e manutenção entram nos gates antes de adoção.

# Agent Connectivity relacionado

A stack de clientes consome o [22 — Agent Connectivity Runtime: Providers, OAuth, ACP, SDK, RPC e CLI](ch22.md). Desktop e TUI renderizam eventos/capabilities normalizados e não implementam autenticação/provider logic específica de vendors.

# Próximo passo

A stack está fechada o suficiente para iniciar **especificação de funcionalidades e interface**. Desktop e TUI serão detalhados separadamente, mas a partir de um catálogo comum de capacidades e state models para impedir divergência funcional e duplicação arquitetural.
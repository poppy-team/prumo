# 82 — Prumo Native com Freya: Workspace Viewer Agent-Aware, Editor Leve e Plano de Implementação

> Authority: canonical specification.
> Logical ID: CONST-82
> Source: Notion Living Book (3de9bb7d023f81c791d4e4145393fa10)
> Status: Especificação do Prumo Native Workspace Viewer com Freya.


<aside>
🧩

**Decisão:** o app nativo do Prumo é construído **do zero com Freya** (GUI declarativa em Rust, renderização Skia) e preserva sua natureza de coding-agent desktop leve. Não transformar o produto em uma IDE tradicional completa. O objetivo é um **Workspace Viewer agent-aware**: visualizar, acompanhar e editar os arquivos/pastas/código que os agentes criam e modificam sem abrir uma ferramenta externa.

A direção anterior de basear o viewer em Oxi (fork de editor) foi **superseded** em 22/09/2026. O crate `prumo-viewer` baseado em egui/eframe foi descartado e reimplementado do zero com Freya; a especificação derivada de Oxi foi removida do repositório. Esta página substitui a CONST-82 em sua totalidade.

</aside>

## Relação com a arquitetura do Prumo

Esta decisão preserva a separação já estabelecida no Livro Vivo:

- o **Prumo Core em Go** continua sendo a autoridade de protocolo, estado canônico, Runs, Agents, Context, Evidence, Quality, Skills e Experience;
- o app nativo é uma **surface/client**, não um segundo harness independente;
- `Run != GUI` e `Run != ExecutorSession` continuam invariantes;
- a GUI pode ser fechada/reaberta sem redefinir a identidade da Run;
- o editor/viewer existe para observabilidade, revisão e pequenas intervenções humanas — não para duplicar o Control Plane.

[02 — Arquitetura Atlas v0.4 e Boundaries do Core](../framework/specs/ch02.md)

[48 — Agent Runtime Control Plane: Arquitetura e Princípios](../framework/specs/ch48.md)

[26 — Native Desktop GUI Test Bridge](../framework/specs/ch26.md)

[79.T — Technology Skill Packs: egui, Bubble Tea v2 e Ecossistemas de Frameworks](../skills/specs/technology-skill-packs-egui-bubble-tea-v2.md)

## Por que Freya

O requisito real do Prumo não é competir com VS Code, Zed ou Lapce como IDE geral. É:

> **Enquanto o code agent trabalha, o usuário deve conseguir ver a árvore do workspace, abrir os arquivos modificados/criados, acompanhar mudanças, ler o código, comparar diffs e realizar pequenas edições sem sair do Prumo.**

Freya é a escolha para construir isso do zero porque:

1. **GUI nativa declarativa em Rust** — componentes reativos com estado tipado, sem web tech, sem fork de editor de terceiros para manter;
2. **Renderização Skia** — texto nítido, prerasterização de camadas e gradientes/transformações que o editor e o diff view exigem;
3. **Escopo controlado** — Freya oferece layout (Torque), eventos, scrolling, SVG, canvas e acessibilidade como primitivas; o que falta (syntax highlighting, tree view, diff engine) são bibliotecas Rust conhecidas e substituíveis, não um fork de 100k linhas;
4. **Sem dependência de projeto externo** — a direção Oxi exigia pin de revisão upstream, sincronização de fork e auditoria de dependências de terceiros. Construir com Freya elimina essa classe de risco: o limite do produto é o nosso próprio código;
5. **Licença MIT** — compatível com a distribuição do Prumo.

Trade-offs aceitos: Freya é mais jovem que egui/Iced; a equipe assume o custo de construir editor, explorer e diff view (em vez de herdar de um fork). Esse custo é deliberado — é exatamente a superfície que precisa ser agent-aware, e nenhuma base herdada entrega isso sem cirurgia profunda.

## Stack técnica aprovada

```
Prumo Native (Rust, Freya 0.4+)
├── freya            — componentes, estado reativo, janela, eventos
├── freya-elements   — parágrafo, imagem, canvas, SVG, scroll
├── torque           — layout engine (via Freya)
├── skia-safe        — rasterização (via Freya)
├── tree-sitter       — grammars via freya-code-editor (lazy por tab)
├── similar          — diff engine (inline + word-level)
├── notify           — file watching (debounced)
├── tokio            — runtime async compartilhado (1 runtime, workers limitados)
├── serde/serde_json — config, projeções e eventos do protocol
├── git2 (libgit2)   — git status/diff em worker de fundo
├── portable-pty + vt100 — terminal embutido (fase 2, implementada)
└── arboard/rfd      — clipboard e diálogos nativos
```

Bibliotecas não aprovadas neste recorte: editores prontos de terceiros, webview (Tauri/Wry), toolkits web-based. O editor é código do Prumo sobre Freya.

## Produto-alvo: Prumo Workspace Viewer

```
┌──────────────────────────────────────────────────────────────┐
│ Prumo                                                        │
├──────────────┬───────────────────────────────┬───────────────┤
│ WORKSPACE    │ CODE                          │ AGENT         │
│              │                               │               │
│ ▼ src        │ main.go                       │ ● Working     │
│   main.go M  │                               │               │
│   app.go  +  │ func main() {                 │ Created       │
│ ▼ internal   │     ...                       │ src/app.go    │
│   agent.go M │ }                             │               │
│              │                               │ Modified      │
│              │                               │ agent.go:42   │
├──────────────┴───────────────────────────────┴───────────────┤
│ TERMINAL / DIFF / EVIDENCE                                  │
└──────────────────────────────────────────────────────────────┘
```

O Workspace Viewer não deve ser uma IDE generalista. Ele deve otimizar quatro verbos humanos:

1. **ver** o que o agente está fazendo;
2. **entender** os arquivos e alterações;
3. **acompanhar** a execução em tempo real;
4. **corrigir** pequenas coisas sem mudar de aplicativo.

## Direção visual: Prumo Studio

A interface combina padrões de duas fontes reconhecidas com o modelo próprio do Prumo:

- [Zed](https://zed.dev/features): editor dominante, baixa ornamentação, divisórias finas, alta densidade, paleta neutra e foco no código;
- [Xcode](https://developer.apple.com/documentation/xcode/configuring-the-xcode-project-window): toolbar contextual, navigator, editor com tabs, inspector à direita, área inferior e hierarquia de informação por seção;
- [Xcode 27](https://developer.apple.com/videos/play/wwdc2026/258/): conversa de coding agent no fluxo do editor, activities persistentes, mudanças e artefatos associados à Run.

A referência é o modelo mental, não uma cópia de marca. A resultante deve parecer uma IDE profissional calma, técnica e densa, sem reproduzir a quantidade de controles do Xcode.

### Grid e proporção

| Região | Comportamento |
| --- | --- |
| Titlebar | 40 px, contexto do workspace e da Run sempre visível |
| Activity rail | 44 px, persistente e independente da sidebar |
| Sidebar | 240 a 360 px, com Explorer e Changes |
| Editor | mínimo de 480 px e maior área útil da tela |
| Right stack | 300 a 380 px, com Prumo Agent e inspector contextual |
| Bottom dock | 30 px recolhido; 160 a 260 px expandido, com Activity, Changes e Evidence real |
| Statusbar | 26 px, facts de conexão, Run, documento e contagens |

Em viewport largo, rail, sidebar, editor, right stack e dock coexistem. Em viewport médio, o dock recolhe antes da sidebar. Em viewport estreito, a sidebar e a right stack recolhem antes de comprimir o editor.

### Linguagem visual

- superfícies distinguem canvas, sidebar, editor, panel e área elevada sem depender apenas de bordas;
- texto principal, secundário e placeholder mantêm contraste suficiente nos dois temas;
- azul é reservado para foco, ação e estado ativo; verde, amarelo e vermelho representam hechos, não decoração;
- divisórias são suaves; cards e cantos arredondados ficam restritos a overlays e approvals;
- ícones têm hit area mínima, label acessível e estado visível;
- hierarquia usa posição, tamanho e peso antes de cor;
- animações são curtas, funcionais e desativadas em reduced motion.

### Navegação e inspetores

- o activity rail alterna Explorer, Changes e Prumo Agent;
- a sidebar preserva a seleção ao trocar de modo;
- Changes lista somente arquivos presentes na projeção da Run;
- o right stack mostra Run, phase, approvals e timeline;
- o bottom dock apresenta Activity, Changes e Evidence apenas quando existem dados;
- controles de debug, terminal e inspector não aparecem até possuírem operação real.

### Integrações Prumo

- explorer exibe `Created`, `Modified` e `Deleted` oriundos da RunProjection;
- agent panel inicia Runs, mostra eventos e responde approvals pelo Agent Protocol;
- diff e dirty state protegem edição humana contra alteração externa;
- Evidence e Quality aparecem no dock quando o Core fornecer esses dados;
- status bar mostra somente conexão, Run, documento e contagens verificáveis.

### Estado implementado do shell

- o editor usa o componente `CodeEditor` nativo do Freya com Rope, VirtualScrollView e Tree-sitter;
- Rust, Go, JSON, YAML, Markdown, TOML, TypeScript e JavaScript possuem gramáticas carregadas pelo editor;
- o Runner envia `start`, `steer`, `cancel`, `models`, `approve` e `deny` exclusivamente pelo Agent Protocol publicado;
- o terminal inferior usa `portable-pty` + `vt100`, inicia no workspace, mantém stdin/stdout, permite restart e termina o child junto com a sessão;
- o terminal não executa com privilégios elevados e a UI informa que herda os privilégios do usuário;
- preferências persistidas usam `schemas/viewer-config.schema.json` em `$PRUMO_VIEWER_CONFIG` ou, por padrão, no diretório de configuração do usuário sob `prumo/viewer.json`; segredos e API keys nunca são armazenadas nesse arquivo;
- socket e shell customizados exigem restart, enquanto provider, model, tema, whitespace e intervalo de polling entram em vigor na viewer;
- ResizableContainer horizontal usa contexto interno do Freya, chaves estáveis e `Content::flex`; o CodeEditor precisa ocupar a área central medida por teste headless.

## Agent-aware Explorer

O explorer é uma representação do estado da Run, não apenas uma árvore de arquivos.

Estados sugeridos:

- `+` — criado na Run atual;
- `M` — modificado na Run atual;
- `D` — removido;

Estados vêm das projeções do protocolo (Run/Agent/Tool events), nunca de heurísticas locais de filesystem como fonte de verdade. Git status complementa (worker de fundo, refresh debounced).

Em Freya: componente `ExplorerTree` com virtualização de linhas (apenas itens visíveis são renderizados), estado em `Signal` raiz derivado de eventos do protocol.

## Follow Agent

O Follow Agent mantém a UI sincronizada com a execução:

- abre automaticamente o arquivo que o agente está editando;
- destaca as linhas alteradas no momento da alteração;
- exibe timeline de eventos da Run (tool calls, edições, approvals pendentes);
- nunca bloqueia a edição humana (o usuário pode desanexar a qualquer momento).

## Alterações planejadas no editor

O editor é construído do zero com escopo restrito:

- multi-tab com abas virtuais (lazy: conteúdo carrega ao focar);
- syntax highlighting via Syntect em linha (não documento inteiro a cada frame);
- find/replace no arquivo aberto;
- go-to-file (fuzzy) com `nucleo-matcher`;
- save/reload com detecção de alteração externa;
- diff aberto como tab do editor (read-only, com gutter);
- sem minimap no MVP (reavaliar após medir custo de render);
- sem LSP no MVP.

## File watching

- `notify` com debounce (mínimo 100ms por batch de eventos);
- watchers recriados em rename/move de diretórios;
- eventos coalescidos antes de tocar estado da UI;
- **nenhuma I/O de filesystem durante o paint** — watchers escrevem em canal; uma task Tokio consolida e atualiza `Signal`s;
- ignore rules respeitadas (`.gitignore`, `.prumoignore`).

## Buffer e arquivos grandes

- buffer `String` com limite defensivo (default 8 MB por arquivo aberto);
- arquivos acima do limite abrem em modo truncado com aviso explícito e opção "abrir mesmo assim" (somente leitura);
- não migrar para Ropey por entusiasmo tecnológico. Primeiro medir:
  - latência de inserção/remoção típica de edição humana (<1k linhas alteradas por save);
  - tamanho real dos arquivos em workspaces-alvo;
- virtualização: apenas as linhas visíveis são medidas e rasterizadas; windows de contexto para scroll suave.

## Conflitos humano × agente

Contrato explícito:

- se o humano edita um arquivo que o agente modifica:
  - a UI não salva silenciosamente por cima;
  - o arquivo entra em estado `conflicted` visível;
  - o diff 3-way (base, agente, humano) é oferecido;
  - resolver é ação humana explícita (manter meu, aceitar do agente, mesclar manual);
- autosave é opt-in, nunca default;
- o Core continua sendo a autoridade do que foi aceito na Run; a GUI apenas projeta.

## Arquitetura recomendada

```
┌─────────────────────────────────────────────┐
│                  Freya UI                   │
│  (componentes declarativos, Signals)        │
├─────────────────────────────────────────────┤
│              Application Services           │
│  OpenDocument · SaveDocument · Diff         │
│  FollowAgent · Approvals · Search           │
├─────────────────────────────────────────────┤
│                 Projections                 │
│  RunProjection · WorkspaceProjection        │
│  (estado derivado, reconstituível)          │
├─────────────────────────────────────────────┤
│              PrumoClient (protocol)         │
│  connect/reconnect · replay · versioning    │
├─────────────────────────────────────────────┤
│         Prumo Core (Go, via socket)         │
│  autoridade canônica de estado e protocolo  │
└─────────────────────────────────────────────┘
```

Evitar que um componente `App` acumule responsabilidades. UI chama application services e lê projections; watchers/Git/eventos do Prumo não manipulam widgets diretamente — atualizam estado, e a reatividade do Freya propaga.

## Boundaries de implementação

- `ui/` — componentes Freya puros; sem I/O, sem protocol;
- `services/` — casos de uso; únicos que orquestram projections e client;
- `projections/` — estado derivado; reconstituível a partir de replay do protocol;
- `client/` — transporte do protocolo (socket, framing, reconexão);
- `platform/` — watchers, PTY, clipboard, diálogos;
- `state/` — Signals e stores compartilhados.

Nenhuma camada pula a anterior: componente não fala com o client; service não toca widget.

## Performance budget

Medido no hardware de referência da equipe, workspace médio (2–5k arquivos):

- idle CPU ≤ 1% (event-driven: repaint só quando estado muda ou animação ativa);
- tempo até primeira árvore visível ≤ 300ms;
- abertura de arquivo ≤ 80ms (arquivo ≤ 1MB);
- input latency do editor imperceptível (p95 ≤ 16ms por tecla em arquivo de 5k linhas);
- memória idle ≤ 250MB;
- scroll em árvore de 10k itens sem stutter.

Freya renderiza com Skia e só reavalia componentes cujos `Signal`s mudaram; qualquer violação de budget vira issue com benchmark reproduzível (ver matriz obrigatória).

## Testes obrigatórios

- unit: services, projections e parsing do protocol;
- golden: render de estados canônicos (explorer vazio, Run ativa, conflito, diff);
- integração: PrumoClient contra daemon de teste (reconnect, replay, cancel);
- perf: benchmarks com critério de reprova por budget;
- smoke e2e: abrir workspace → conectar à Run → observar edição do agente → resolver conflito.

### Gate funcional da interface

A linguagem visual pode ser de IDE, mas controles decorativos não contam como funcionalidade:

- todo controle visível executa uma ação real ou não aparece;
- conexão, Run, branch e contagens refletem dados verificáveis;
- salvar, trocar de aba e fechar não perdem edição humana;
- Quick Open, explorer e editor operam por teclado e mouse;
- falha de protocol, leitura ou escrita aparece em um caminho de correção;
- help e documentação só anunciam capacidades cobertas por teste ou integração real.

Esse gate vale antes de política visual, goldens e novas bibliotecas.

## Estratégia de implementação incremental

O recorte do zero exige disciplina de fatias verticais entregáveis:

- **Wave 0 — Esqueleto:** janela Freya + shell de layout (3 painéis) + config + logging. Critério: abre, fecha, layout responsivo.
- **Wave 1 — Workspace read-only:** explorer virtualizado + abertura de arquivos com highlighting + busca fuzzy de arquivos. Sem protocol: dados locais.
- **Wave 2 — Protocol:** PrumoClient (connect/reconnect/replay) + RunProjection + painel Agent com timeline de eventos.
- **Wave 3 — Edição:** save/reload + deteção de conflito humano×agente + diff view (Similar) como tab.
- **Wave 4 — Git:** worker de fundo (git2) + status no explorer + diff Git.
- **Wave 5 — Follow Agent:** auto-follow das edições do agente + highlight de linhas modificadas.
- **Wave 6 — Terminal:** PTY embutido (portable-pty + vt100) como painel inferior.
- **Wave 7 — Polish:** atalhos, acessibilidade Freya/AccessKit, golden tests visuais, tuning de performance.

Cada Wave termina com benchmark + golden tests atualizados + doc desta página se o contrato mudar.

## Critério de sucesso

- o usuário acompanha uma Run real sem abrir outra ferramenta;
- conflitos humano×agente nunca são resolvidos silenciosamente;
- budgets de performance respeitados em máquina modesta;
- nenhum fork de projeto externo no caminho crítico (zero manutenção de upstream);
- a GUI pode ser apagada e reconstruída a partir desta especificação sem consultar código morto.

## Decisão resumida

**Freya + bibliotecas Rust consolidadas** é a baseline oficial do app nativo do Prumo. O escopo permanece Workspace Viewer agent-aware, não IDE generalista. O investimento vai para agent-awareness, confiabilidade de workspace, diff/merge e observabilidade. A direção anterior (base Oxi, crate egui) foi descartada e o crate `prumo-viewer` foi reimplementado com Freya; esta página é a única fonte vigente.

# Fechamento aprovado — custo, tooling e programação agentic

## Estimativa de esforço

Reconstrução do zero com Freya (sem herdar código de editor externo):

- **160–260 horas humanas** para o MVP completo (Waves 0–5);
- **60–90 horas** adicionais para terminal embutido e polish (Waves 6–7);
- cenário agentic bom (Prumo orquestrando o próprio desenvolvimento): **redução de 35–50%** no esforço humano, confirmada empiricamente Wave a Wave.

Comparação com a direção anterior: Oxi reduzia esforço herdando editor (~51% no ponto médio), mas transferia risco de fork e limitava agent-awareness à estrutura herdada. A diferença de esforço estimada é de +15–25% em favor de Oxi no MVP; o custo é aceito em troca de ownership total da base.

## Decomposição recomendada

- **15–20%** — fundação: janela, layout, config, logging, build/CI;
- **20%** — explorer + abertura de arquivos + highlighting;
- **20%** — protocol/client/projections;
- **15%** — edição, save, conflitos, diff;
- **10%** — Git worker;
- **10%** — Follow Agent;
- **10%** — performance, testes, acessibilidade.

## Onde não economizar agressivamente

- camadas de boundaries (ui/services/projections/client);
- testes de protocol (reconnect/replay);
- benchmarks de performance desde a Wave 0;
- tratamento de conflito humano×agente;
- acessibilidade básica.

## Bibliotecas aprovadas para reduzir fricção

- `nucleo-matcher` — fuzzy matching de arquivos;
- `similar` — diff com inline;
- `notify` — watching;
- `git2` vendored — Git sem binário externo;
- `insta` — golden tests;
- `proptest` — propriedades de parsing/estado.

## Gauntlet Loop oficial para este recorte

Cada Wave passa pelo loop:

1. especificação da Wave revisada contra esta página;
2. implementação agentic com boundaries explícitos;
3. benchmark + golden tests + testes de protocol verdes;
4. revisão humana de conflitos de arquitetura;
5. atualização do changelog desta página se contrato mudou.

Waves não acumulam: uma Wave só começa quando a anterior fecha com critérios verdes.

## Non-goals reafirmados

- não virar IDE generalista (sem LSP no MVP, sem debugger, sem refactor engine);
- não implementar segundo harness ou estado canônico no client;
- não adicionar webview/web tech;
- não herdar fork de editor de terceiros;
- não otimizar prematuramente (Ropey, minimap, multi-janela) sem medição.

## Decisão consolidada

Construção do zero com Freya, bibliotecas auxiliares aprovadas e desenvolvimento agentic orientado pelo Prumo/Gauntlet Loop. Orçamento técnico seguro do MVP: **160–260 horas humanas** antes de assumir ganhos de programação agentic.

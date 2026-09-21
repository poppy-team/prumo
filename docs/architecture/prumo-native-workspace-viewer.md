# 82 — Prumo Native sobre Oxi: Workspace Viewer Agent-Aware, Editor Leve e Plano de Implementação

> Authority: canonical specification.
> Logical ID: CONST-82
> Source: Notion Living Book (3de9bb7d023f81c791d4e4145393fa10)
> Status: Especificação do Prumo Native Workspace Viewer sobre Oxi.


<aside>
🧩

**Decisão:** o app nativo do Prumo deve partir do **oxi** e preservar sua natureza de coding-agent desktop leve. Não transformar o produto em uma IDE tradicional completa. O objetivo é um **Workspace Viewer agent-aware**: visualizar, acompanhar e editar os arquivos/pastas/código que os agentes criam e modificam sem abrir uma ferramenta externa.

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

## Por que Oxi em vez de Lapce

A necessidade foi refinada durante a pesquisa. O Prumo não precisa, no primeiro recorte, competir com VS Code, Zed ou Lapce como IDE geral. O requisito real é:

> **Enquanto o code agent trabalha, o usuário deve conseguir ver a árvore do workspace, abrir os arquivos modificados/criados, acompanhar mudanças, ler o código, comparar diffs e realizar pequenas edições sem sair do Prumo.**
> 

Isso reduz drasticamente o escopo e favorece Oxi porque o projeto já possui a maior parte dessa superfície.

## Fundação já existente no Oxi

O fork deve preservar e aprofundar os componentes já existentes:

- workspace file explorer;
- editor multi-tab;
- syntax highlighting;
- minimap;
- find/replace;
- file create/rename/delete;
- save/reload;
- detecção de alterações externas;
- Git status no explorer;
- Git diff aberto como tab do editor;
- terminal PTY embutido;
- Tree-sitter/Syntect para múltiplas linguagens;
- Git worker em background;
- agent activity/tool blocks já estruturados;
- ACP/MCP e approvals existentes como referências de UX, ainda que a autoridade final seja substituída pelo Prumo.

O código do Oxi já separa `documents`, `editor_body`, `editor_logic`, `editor_paint`, `editor_tabs`, `explorer_tree`, `file_operations`, `find_replace`, `minimap` e `navigation_diff`. Essa modularização deve ser mantida e aprofundada, não substituída por um editor externo completo.

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

## Agent-aware Explorer

O explorer deve evoluir de árvore de arquivos para uma representação do estado da Run.

Estados sugeridos:

- `+` — criado na Run atual;
- `M` — modificado na Run atual;
- `D` — removido;
- `R` — renomeado;
- `●` — agente trabalhando nesse arquivo agora;
- `✓` — alteração verificada por gate/teste;
- `!` — alteração associada a falha de teste/gate.

A origem do estado deve ser estruturada: Prumo events + Git/workspace state, nunca heurística visual apenas.

## Follow Agent

Criar um modo explícito **Follow Agent**.

```
Agent tool event
      ↓
path + optional line/range
      ↓
WorkspaceEventAdapter
      ↓
OpenFile(path)
      ↓
GoToLine(range.start)
      ↓
highlight affected range
```

Com `Follow Agent` habilitado, o viewer acompanha automaticamente o arquivo em que o agente está atuando. O usuário pode desligá-lo a qualquer momento e navegar livremente.

### Eventos mínimos

- file read;
- file created;
- file modified;
- file deleted;
- file moved/renamed;
- diff produced;
- verification started/completed/failed;
- tool event com `path`, `line`, `range` ou `symbol` quando disponíveis.

## Alterações planejadas no editor

### P0 — obrigatório

- line numbers robustos;
- go-to-line;
- click em tool/event → abrir `path:line`;
- badges Created/Modified/Deleted/Renamed;
- Follow Agent;
- before/after diff integrado;
- changed/recent files;
- workspace search simples;
- file watcher real e debounced;
- conflitos entre edição humana e edição do agente tratados explicitamente;
- testes de reload/save/external modification;
- viewport eficiente para arquivos maiores.

### P1 — recomendado

- migrar backing buffer de `String` para rope caso profiling confirme benefício;
- split editor simples;
- command palette/quick open;
- fuzzy file search;
- inline diff refinement;
- breadcrumbs simples `folder/file[:line]`;
- pinned files;
- history de navegação back/forward.

### P2 — somente se uso real justificar

- LSP genérico;
- diagnostics;
- symbol navigation;
- rename/refactoring;
- DAP/debugger;
- extension marketplace.

LSP não é requisito do primeiro corte. O agente já executa search/read/edit/test; o humano precisa principalmente observar, compreender, revisar e corrigir.

## Bibliotecas auxiliares recomendadas

| Necessidade | Biblioteca/abordagem | Uso |
| --- | --- | --- |
| Filesystem events | `notify`  • `notify-debouncer-mini/full` | Substituir polling/metadata checks como mecanismo principal; manter fallback para filesystems sem eventos confiáveis. |
| Large text buffer | `ropey` | Rope UTF-8 para edição eficiente de arquivos maiores; adotar apenas depois de benchmark da implementação atual. |
| Text diff / merge | `similar` 3.x | Line/word/char diff, inline refinement, deadlines e three-way merge para conflitos humano×agente. |
| Undo/redo | `egui::util::undoer` ou camada própria sobre operações do editor | Evitar reimplementar histórico simples; para Ropey, preferir command log/transactions para não clonar buffers enormes. |
| Explorer tree | `egui_ltreeview` ou `egui_ailanthus` | Keyboard navigation, selection, drag/drop e badges; usar somente se integração reduzir código sem perder agent-aware rendering. |
| Fuzzy finder | `nucleo-matcher` | Quick Open e busca rápida de arquivos/comandos. |
| Parsing/highlighting | Tree-sitter + Syntect já existentes | Reaproveitar; não adicionar LSP para resolver syntax highlighting. |
| Git state/diff | `git2` já existente + worker atual | Status, deltas, hunks e rename detection sem bloquear UI. |
| Snapshot tests | `insta` | Snapshots de modelos de diff, explorer states, event projections e serializações. |
| Property tests | `proptest` | Paths, ranges, edit sequences, reload/merge e invariantes de workspace safety. |
| Test runner | `cargo-nextest` | Execução rápida/isolada da suite Rust no Gauntlet Loop. |
| Benchmark | `criterion`  • métricas de processo | Diff, viewport, search, large files e event bursts. |

## File watching

Preferir watcher nativo por plataforma através de `notify`, com debounce e fallback para polling quando o filesystem não oferece eventos confiáveis.

```
agent / user / git
      ↓
filesystem mutation
      ↓
notify
      ↓
WorkspaceWatcher
      ↓
WorkspaceEvent
      ├── ExplorerProjection
      ├── EditorProjection
      ├── DiffProjection
      └── RunActivityProjection
```

O watcher não deve atualizar a UI diretamente. Ele produz eventos tipados consumidos por projections/application state.

## Buffer e arquivos grandes

A implementação Oxi atual usa `String` e limite defensivo de tamanho. Não migrar para Ropey por entusiasmo tecnológico. Primeiro medir:

- arquivo 100 KiB;
- 1 MiB;
- 2 MiB;
- 5 MiB;
- 20 MiB;
- random edits;
- edits próximas e distantes;
- scroll/viewport;
- syntax refresh;
- diff refresh.

Migrar para Ropey somente se o custo atual for material. Se migrar, separar:

```
DocumentBuffer
├── text access
├── line/char translation
├── edit transaction
├── revision
└── snapshot/export
```

para impedir acoplamento direto do restante da UI ao tipo concreto `Rope`.

## Conflitos humano × agente

Não sobrescrever silenciosamente alterações humanas.

Modelo recomendado:

```
base = versão vista pelo agente
ours = versão atual do usuário
 theirs = nova edição proposta pelo agente
        ↓
three-way merge
        ↓
clean merge OR explicit conflict UI
```

`similar` pode fornecer o primitive de three-way line merge. A política final pertence ao Prumo/Workspace layer.

## Arquitetura recomendada

```
              PRUMO NATIVE
                  Oxi
                   │
    ┌──────────────┼──────────────┐
    │              │              │
Agent View     Workspace View   Terminal
    │              │
    │        ┌─────┴─────┐
    │        │           │
Timeline   Explorer    Editor
    │        │           │
    │    Run badges   Tabs/Highlight
    │    Git state    Find/Diff
    │        │           │
    └────────┴─────┬─────┘
                   │
              PrumoClient
                   │
              `prumo serve`
                   │
              PRUMO CORE / Go
```

## Boundaries de implementação

Recomendados:

```
WorkspaceService
WorkspaceWatcher
DocumentStore
DocumentBuffer
EditorNavigation
DiffService
RunFileProjection
AgentFollowController
PrumoClient
```

Evitar `OxiApp` acumulando novas responsabilidades. UI chama application services e projections; watchers/Git/Prumo events não manipulam widgets diretamente.

## Performance budget

Meta inicial do fork:

- preservar aproximadamente a ordem de grandeza atual de memória idle do Oxi;
- nenhuma operação de agent-follow pode bloquear a UI thread;
- file events devem ser coalescidos/debounced;
- syntax/diff em background quando exceder threshold pequeno;
- renderizar viewport, não documento inteiro, quando necessário;
- manter terminal lazy;
- LSP permanece ausente até existir necessidade demonstrada.

## Testes obrigatórios

1. agente cria arquivo → aparece no explorer sem refresh manual;
2. agente modifica arquivo aberto → atualização segura, sem perder edição humana;
3. Follow Agent abre arquivo + linha corretos;
4. Follow Agent desligado nunca rouba foco;
5. rename preserva tab/history quando possível;
6. delete de arquivo aberto produz estado explícito;
7. burst de centenas de filesystem events não congela UI;
8. Git checkout/branch switch reconcilia documents corretamente;
9. diff grande possui deadline/fallback;
10. watcher fallback funciona em ambiente sem native events;
11. paths fora do workspace nunca são abertos por eventos malformados;
12. accessibility tree continua coerente para explorer/editor controls;
13. tests com Unicode, CRLF/LF, symlinks, arquivos vazios e read-only;
14. crash/reopen preserva estado de tabs quando configurado;
15. memória e frame time permanecem dentro do budget documentado.

## Estratégia de implementação incremental

### Wave 1 — Integration Baseline

- fork limpo do Oxi;
- PrumoClient boundary;
- remover/encapsular autoridade do agent runtime próprio;
- preservar editor/explorer/terminal/Git funcionando;
- conformance tests baseline.

### Wave 2 — Agent-aware Workspace

- `RunFileProjection`;
- badges;
- event→path/range navigation;
- Follow Agent;
- recent/changed files;
- integration com Evidence/Quality state.

### Wave 3 — Workspace Reliability

- `notify` watcher + debounce;
- human×agent conflict handling;
- diff service com deadlines;
- reload/rename/delete reconciliation;
- property/snapshot tests.

### Wave 4 — Editor Hardening

- profiling de buffers;
- Ropey apenas se necessário;
- viewport/large-file improvements;
- undo/redo hardening;
- quick open via fuzzy matcher;
- split editor opcional.

### Wave 5 — Optional Intelligence

Somente após uso real:

- generic LSP;
- diagnostics;
- symbols;
- refactors;
- DAP.

## Critério de sucesso

O usuário deve conseguir executar uma Run longa e, sem abrir outro editor:

- acompanhar qual arquivo cada agente está lendo/editando;
- abrir imediatamente o local relevante;
- revisar before/after;
- navegar pelo workspace;
- fazer pequenas correções;
- ver Git/verification state;
- retornar ao Agent View sem perder contexto.

O projeto **não** precisa satisfazer o critério “substituir VS Code/Lapce”. Ele satisfaz o critério “não precisar abrir VS Code/Lapce para observar e corrigir o trabalho do agente”.

## Decisão resumida

**Oxi + editor existente reforçado** é o baseline oficial proposto para o app nativo do Prumo. Lapce permanece referência de IDE/editor e Lux Edit/OpenEdit podem servir como donors técnicos pontuais, mas não são dependências-base. O investimento deve ir para agent-awareness, confiabilidade de workspace, diff/merge e observabilidade — não para construir uma IDE generalista.

# Fechamento aprovado — custo, tooling e programação agentic

<aside>
✅

**Decisão fechada em 17 de setembro de 2026:** seguir com **Oxi + bibliotecas auxiliares + desenvolvimento agentic orientado pelo Prumo/Gauntlet Loop**. O orçamento técnico seguro do MVP permanece **130–220 horas humanas** antes de assumir ganhos de programação agentic. A faixa **78–165 horas humanas** é o cenário operacional bom após aplicar uma hipótese de redução de 25–40%, não uma garantia.

</aside>

## Comparação consolidada de custo

As estimativas representam **horas de engenharia efetiva**, incluindo implementação, integração, correção, review e testes. Não representam calendário nem promessa de duração.

| Recurso | Praticamente do zero | Oxi + bibliotecas | Oxi + libs + agentic — cenário bom |
| --- | --- | --- | --- |
| Watcher + reconciliação de mudanças | 24–40h | 10–18h | 6–14h |
| Go-to-line + highlight de range | 12–20h | 4–8h | 2–6h |
| Quick Open / fuzzy files | 20–32h | 6–12h | 4–9h |
| Badges agent-aware no Explorer | 24–40h | 12–20h | 7–15h |
| Tool-call → arquivo:linha | 24–40h | 12–20h | 7–15h |
| Follow Agent | 32–56h | 18–30h | 11–22h |
| Before/after diff | 32–56h | 12–24h | 7–18h |
| Conflito humano ↔ agent | 40–72h | 24–40h | 14–30h |
| Changed / recent files | 16–24h | 8–14h | 5–10h |
| Undo/redo hardening | 24–48h | 16–28h | 10–21h |
| Acessibilidade + testes semânticos | 40–64h | 24–40h | 14–30h |
| Performance/load/regression | 48–80h | 32–48h | 19–36h |
| **MVP consolidado** | **270–450h** | **130–220h** | **78–165h** |

O MVP desta tabela **não inclui** migração completa do backing buffer para Ropey nem split/docking livre.

## Leitura dos números

Pelos pontos médios das faixas:

```
construção praticamente do zero
≈ 360 h

        Oxi + crates
              ↓
≈ 175 h

        agentic — cenário bom
              ↓
≈ 122 h de esforço humano
```

Isso significa que **Oxi + bibliotecas reduz aproximadamente 51% do esforço no ponto médio** antes mesmo de assumir benefício de code agents. Se o cenário agentic bom for confirmado empiricamente no próprio projeto, o esforço humano pode ficar aproximadamente **66% abaixo da construção equivalente praticamente do zero**.

Esses percentuais são derivados das faixas de planejamento e **não devem ser apresentados como benchmark externo ou promessa**.

## Decomposição recomendada das 130–220h

Como baseline de planejamento interno:

- **70–115h** — implementação funcional nova;
- **40–65h** — correção, hardening, testes, regressão e edge cases;
- **20–40h** — integração/refatoração do Oxi com `PrumoClient`, boundaries e projections.

A divisão é aproximada e serve para evitar que todo o orçamento seja erroneamente tratado como “feature coding”.

## Onde não economizar agressivamente

Manter forte revisão humana e Independent Verification em:

- concorrência humano ↔ agent;
- reconciliação de filesystem events;
- race conditions;
- dirty buffers;
- focus e keyboard behavior;
- acessibilidade/AccessKit;
- profiling e frame-time;
- large diffs;
- path validation e workspace boundaries;
- conflict resolution.

A regra central permanece:

> **nenhum evento externo ou alteração agentic pode sobrescrever silenciosamente trabalho humano não salvo.**
> 

## Bibliotecas aprovadas para reduzir fricção

| Biblioteca | Status | Responsabilidade |
| --- | --- | --- |
| `notify`  • debouncer | **Aprovada / MVP** | Filesystem watcher, coalescing e fallback quando necessário |
| `similar` | **Aprovada / MVP** | Diff textual, inline refinement e primitive para merge/conflict |
| `nucleo` / `nucleo-matcher` | **Aprovada / MVP** | Quick Open e fuzzy file search |
| Tree-sitter + Syntect atuais | **Reutilizar** | Parsing/highlighting; não duplicar com LSP |
| `git2` atual | **Reutilizar** | Status/diff/rename detection em worker |
| `insta` | **Recomendada** | Snapshots de projections, diff states e serializações |
| `proptest` | **Recomendada** | Paths, ranges, edit sequences e invariantes |
| `cargo-nextest` | **Recomendada** | Suite rápida no Gauntlet Loop |
| `criterion` | **Recomendada** | Benchmarks de diff, search, viewport e event bursts |
| `Ropey` | **Posterior / evidence-driven** | Large-file buffer somente após profiling |
| `egui_dock` | **Opcional / posterior** | Splits/docking somente se requisito real surgir |

## Cenários agentic aprovados para planejamento

Não assumir um multiplicador mágico. O projeto deve medir o próprio uplift por Wave.

| Cenário | Redução humana assumida sobre 130–220h | Faixa humana | Uso |
| --- | --- | --- | --- |
| Conservador | 10–20% | 104–198h | Referência pessimista |
| Bom | 25–40% | 78–165h | Meta operacional |
| Excelente | 40–55% | 59–132h | Somente após evidência interna suficiente |

Agents tendem a oferecer mais vantagem em:

- scaffolding;
- adapters;
- wiring repetitivo de estado/eventos;
- testes e fixtures;
- snapshot/property tests;
- port/transplant de patterns existentes;
- documentação;
- refactors mecânicos.

Esperar menor ganho e maior revisão humana em:

- concurrency;
- filesystem edge cases;
- focus/input;
- acessibilidade;
- performance profiling;
- visual UX;
- conflitos humano ↔ agent.

## Medição de uplift pelo próprio Prumo

Cada Wave deve registrar pelo menos:

```
estimated_manual_effort
human_active_time
agent_active_time
agent_rework_time
review_time
failed_attempts
regressions_found
regressions_escaped
accepted_first_pass_ratio
```

Com isso, após algumas Waves o Prumo deixa de depender de uma faixa genérica e passa a ter um **multiplicador empírico próprio por classe de tarefa**.

Exemplo futuro:

```
Rust adapter/wiring       → uplift observado 38%
Snapshot/property tests   → uplift observado 52%
Filesystem concurrency    → uplift observado 11%
Accessibility semantics   → uplift observado 17%
UX refinement             → uplift observado 8%
```

Esses valores acima são apenas exemplo de estrutura de métricas, não expectativas.

## Gauntlet Loop oficial para este recorte

```
Spec
→ implementation slice
→ unit tests
→ integration fixture
→ snapshot/property tests quando aplicável
→ UI semantic test
→ performance/regression check
→ independent review
→ evidence
→ accepted OR rework
```

Evitar uma Wave grande que concentre novas responsabilidades em `editor_body.rs` ou `OxiApp`. Preservar e aprofundar os boundaries:

```
WorkspaceService
WorkspaceWatcher
DocumentStore
DocumentBuffer
EditorNavigation
DiffService
RunFileProjection
AgentFollowController
PrumoClient
```

## Waves finais aprovadas

### Wave 0 — Baseline e fork hygiene

- pin da revisão upstream do Oxi;
- baseline cold-start, idle RAM/CPU e frame-time;
- fixtures sintéticas de workspace;
- contract `PrumoClient`;
- regression baseline do editor/explorer;
- medir comportamento atual antes de alterar.

### Wave 1 — Observe

- `notify` watcher;
- changed-files model;
- agent-aware explorer badges;
- go-to-line/range;
- tool/event → path navigation;
- external-change safety.

### Wave 2 — Follow

- Follow Agent;
- timeline ↔ code;
- active-file indicator;
- no-focus-steal;
- pause/freeze durante edição humana.

### Wave 3 — Review

- `similar` before/after diff;
- human/agent conflict state e UI;
- Evidence/Gates annotations;
- review workflow.

### Wave 4 — Navigation

- `nucleo` Quick Open;
- changed/recent filters;
- workspace search quando necessário;
- keyboard shortcuts e command navigation.

### Wave 5 — Scale somente por evidência

- `TextBuffer` abstraction;
- Ropey;
- large-file fixtures;
- viewport optimization;
- optional split/docking.

## Non-goals reafirmados

O recorte não deve evoluir por inércia para:

- clone de VS Code/Zed/Lapce;
- LSP obrigatório no MVP;
- DAP/debugger;
- marketplace tradicional de extensões;
- refactoring engine próprio;
- duplicate agent runtime na GUI;
- policies ou canonical state dentro do frontend.

## Decisão consolidada

```
Oxi
+ editor/explorer existente fortalecido
+ agent-aware workspace semantics
+ approved helper crates
+ PrumoClient
+ prumo serve
+ Prumo Core em Go
+ Gauntlet Loop com evidence
```

**Lapce permanece donor/referência, não base principal deste recorte.**

A implementação deve otimizar o Prumo como **Code Agent Workspace observável, editável, leve, modular e evidence-driven**, e não como IDE generalista.

# Adendo — Auditoria de Performance do Oxi e Estratégia Dual Glow/WGPU — 20/09/2026

<aside>
⚙️

**Decisão atualizada:** manter **Glow e WGPU compilados como renderers selecionáveis**, com **Glow/OpenGL como baseline leve recomendado** para a interface 2D do Prumo Native. A escolha deve ser persistida nas configurações e aplicada no próximo startup. WGPU permanece disponível para compatibilidade, experimentação e futuros recursos que realmente justifiquem a abstração GPU mais pesada.

</aside>

## Síntese da auditoria do Oxi

A análise foi realizada sobre o branch `dev` do repositório [maziluiosif/oxi](https://github.com/maziluiosif/oxi), revisão `cdacb5efd38bec7c90f7200d671335c4bfb91d6d` de 18/09/2026.

O Oxi continua sendo uma excelente base funcional, mas o idle atual pode ser reduzido de forma relevante sem abandonar sua arquitetura inteira. Os principais pontos encontrados foram:

1. renderer WGPU como caminho padrão;
2. Explorer executando leitura de filesystem, ordenação e matching de ignore durante o caminho de renderização;
3. múltiplos runtimes Tokio independentes;
4. repaint periódico para estados de atenção que poderiam ser estáticos/event-driven;
5. ACP, SSH, voice e discovery inicializados mais cedo que o necessário;
6. fontes/default fonts e loaders mais amplos do que o recorte do Prumo exige;
7. dupla infraestrutura de syntax highlighting, Tree-sitter + Syntect;
8. refresh Git agregando trabalho que pode ser decomposto por demanda.

O objetivo não é remover funcionalidades indiscriminadamente, e sim transformar o aplicativo em uma arquitetura **zero-cost while unused**.

## Estratégia gráfica aprovada: Glow + WGPU persistidos

O eframe permite compilar os dois renderers e selecionar um deles na inicialização por `NativeOptions::renderer`. A documentação do eframe 0.35 confirma que Glow é habilitado pela feature `glow` e WGPU pela feature correspondente; quando ambos existem, o renderer usado é determinado no startup.

Referências:

- [eframe 0.35](https://docs.rs/crate/eframe/0.35.0)
- [NativeOptions](https://docs.rs/eframe/latest/eframe/struct.NativeOptions.html)
- [Renderer](https://docs.rs/eframe/latest/eframe/enum.Renderer.html)

### Modelo de configuração do Prumo

```rust
enum GraphicsRenderer {
    Auto,
    Glow,
    Wgpu,
}

enum WgpuBackendPreference {
    Auto,
    Gl,
    Vulkan,
    Dx12,
    Metal,
}
```

Configuração persistida proposta:

```toml
[graphics]
renderer = "glow"
wgpu_backend = "auto"
fallback = true
```

A configuração é lida **antes de `eframe::run_native`** e convertida para `NativeOptions`.

### Política recomendada

```
startup
  ↓
settings.graphics.renderer
  ├── glow
  │     └── egui_glow / OpenGL
  │
  ├── wgpu
  │     └── WGPU backend configurável
  │          ├── Auto
  │          ├── GL
  │          ├── Vulkan
  │          ├── DX12
  │          └── Metal
  │
  └── auto
        └── política da plataforma + fallback
```

### Default recomendado

**Glow deve ser o default inicial do Prumo Native**, pois a superfície é predominantemente UI 2D: editor, sidebar, diffs, terminal, timeline e painéis.

WGPU continua presente, mas não deve ser pago como custo obrigatório por todos os usuários.

### OpenGL persistente

A preferência por OpenGL faz sentido como política de compatibilidade e baixo consumo:

- `Glow` usa OpenGL diretamente por `egui_glow`;
- WGPU possui backend `GL`, além de Vulkan, DX12 e Metal;
- portanto podemos oferecer **Glow/OpenGL** como caminho leve e **WGPU/GL** como modo alternativo para benchmark/compatibilidade;
- não é recomendado forçar WGPU/GL em toda plataforma sem medição;
- em macOS, se WGPU for selecionado, Metal tende a ser a escolha nativa mais coerente;
- em Linux e Windows devemos medir Glow/OpenGL, WGPU/GL e WGPU nativo no mesmo hardware.

O setting é persistente, mas **a mudança de renderer exige restart**. Não tentar hot-swap entre Glow e WGPU durante a execução.

### Matriz proposta

| Modo | Uso | Política |
| --- | --- | --- |
| Glow / OpenGL | UI normal, editor, terminal, Git, agent view | **Default recomendado** |
| WGPU / Auto | Compatibilidade/futuras superfícies GPU | Opcional |
| WGPU / GL | Benchmark e fallback específico | Experimental até evidência |
| WGPU / Vulkan | Linux/Windows quando demonstrar vantagem | Opt-in / Auto |
| WGPU / DX12 | Windows | Opt-in / Auto |
| WGPU / Metal | macOS | Preferência WGPU nativa |

### Importante: não manter dois renderers ativos

Compilar suporte para ambos é aceitável. Manter simultaneamente dois contexts/render loops ativos **não** é.

```
GOOD
binary
├── Glow support
└── WGPU support

startup
└── escolhe exatamente um

BAD
runtime
├── Glow context ativo
└── WGPU device ativo
```

O segundo desenho desperdiçaria justamente RAM/VRAM que estamos tentando economizar.

## P0 — Substituir WGPU obrigatório por renderer configurável

O `Cargo.toml` atual do Oxi ativa `wgpu`, Wayland, X11, AccessKit e default fonts no eframe.

O fork deve testar:

```toml
eframe = {
    version = "0.35",
    default-features = false,
    features = [
        "accesskit",
        "glow",
        "wgpu",
        "wayland",
        "x11"
    ]
}
```

A lista exata de feature flags deverá acompanhar a versão pinada de eframe, mas a regra arquitetural é estável: **compilar ambos, selecionar um**.

Separar benchmark de:

- binary size;
- cold start;
- RSS;
- private memory quando disponível;
- VRAM;
- CPU idle;
- wakeups;
- frame time;
- scroll/editor latency.

## P0 — Explorer deve deixar de fazer filesystem work durante paint

O Explorer atual executa trabalho de I/O e transformação no caminho de renderização:

```
render_file_explorer
  ↓
load .gitignore
  ↓
read_dir
  ↓
collect Vec
  ↓
sort
  ↓
is_gitignored
  ↓
glob matching / Strings
  ↓
paint
```

Isso cria custo toda vez que a UI redesenha.

Arquitetura alvo:

```
filesystem
    ↓
notify
    ↓
WorkspaceIndexer
    ↓
CachedWorkspaceTree
    ↓
visible-row projection
    ↓
egui paint
```

### Implementação

Introduzir:

```
WorkspaceIndex
WorkspaceTree
WorkspaceWatcher
IgnoreMatcher
ExplorerProjection
```

Usar:

- `notify` para eventos;
- `ignore` para semântica Gitignore compilada;
- cache de entradas ordenadas por diretório;
- invalidação incremental;
- virtualização/painting apenas das linhas visíveis quando necessário.

Regra:

> **Paint nunca deve descobrir o filesystem. Paint só deve projetar state já preparado.**
> 

## P0 — Consolidar runtimes Tokio

A auditoria encontrou runtimes separados para pelo menos:

```
AgentExecutor
TunnelManager
AcpManager
```

Além disso, helpers de background criam threads e novos runtimes temporários para operações isoladas.

`Runtime::new()` usa runtime Tokio multi-thread; criar múltiplos runtimes multiplica workers, stacks e estruturas de scheduler.

Arquitetura alvo:

```
UI thread
    │
    └── SharedRuntime
          ├── agent
          ├── HTTP
          ├── model discovery
          ├── update check
          ├── ACP manager
          └── SSH/tunnels

Blocking pool
    ├── filesystem pesado
    ├── Git pesado
    └── parsing/indexing pesado
```

Começar com budget explícito de workers, por exemplo 2, e aumentar somente após benchmark.

Eliminar o pattern:

```
spawn std::thread
    ↓
Runtime::new
    ↓
uma única async task
    ↓
drop
```

## P0 — Idle realmente event-driven

Oxi solicita repaint em intervalos de aproximadamente:

- 50 ms durante waiting/streaming/transcribing;
- 80 ms quando existe atenção pendente na sidebar.

O primeiro pode ser aceitável durante animação/streaming.

O segundo deve ser removido quando o badge estiver estático.

Arquitetura alvo:

```
event
  ↓
state change
  ↓
request_repaint once
  ↓
sleep until next real event
```

Animações de atenção, se existirem, devem ter deadline curto:

```
event
→ pulse 0.5–1.0 s
→ badge estático
→ zero periodic repaint
```

Meta de idle:

> **sem input, sem stream e sem evento externo = event loop dormindo.**
> 

## P1 — Capabilities lazy

Hoje várias infraestruturas já são parcialmente lazy, mas o Prumo deve aprofundar o princípio.

```
Capability never used
        ↓
0 subprocess
0 dedicated runtime
0 network connection
0 loaded model
idealmente 0 dedicated thread
```

Aplicar a:

- ACP;
- SSH;
- voice;
- Whisper;
- local model runtime;
- terminal;
- model discovery;
- MCP servers;
- future LSP/DAP.

### ACP

Não aquecer subprocesso apenas para preencher dropdown de modelos.

Preferir:

1. cache de models conhecido;
2. refresh explícito;
3. spawn no primeiro uso real.

## P1 — Fontes

O Oxi habilita `default_fonts`, mas também instala fontes próprias:

- Noto Sans;
- Ubuntu Mono;
- Apple Symbols;
- Noto Emoji;
- Symbols Nerd Font Mono.

Testar remoção de `default_fonts` quando as fontes próprias cobrirem o necessário.

Manter AccessKit; não trocar acessibilidade por alguns megabytes sem benchmark.

Também estudar subset da Nerd Font contendo apenas os glyphs realmente utilizados.

## P1/P2 — Image loaders mínimos

Oxi usa `egui_extras` com `all_loaders`.

O fork deve habilitar apenas loaders necessários ao produto:

- PNG;
- JPEG;
- WebP;
- GIF, se realmente necessário;
- SVG somente se existir feature que o utilize.

Benefícios principais:

- menor grafo;
- build menor;
- binário menor;
- menor superfície de ataque;
- inicialização potencialmente mais simples.

## P2 — Tree-sitter e Syntect

Oxi usa vários grammars Tree-sitter e também Syntect.

Tree-sitter já possui estado incremental por documento.

Investigar a possibilidade de Tree-sitter ser a infraestrutura dominante para:

- editor;
- code blocks;
- diff;
- preview.

Não remover Syntect antes de medir custo/complexidade. Primeiro registrar memória e latência reais.

Qualquer cache de highlighting deve possuir limite por **bytes**, não apenas quantidade de entradas.

Exemplo:

```
max_entries = 64
max_bytes = 8 MiB
```

## Git — preservar worker, reduzir refresh monolítico

O worker Git atual é event-oriented e portanto é uma boa base.

O problema é que um refresh pode agregar:

- branch;
- branches;
- ahead/behind;
- staged;
- unstaged;
- line changes;
- log;
- diffs.

Decompor:

```
GitStatusSnapshot
GitBranchSnapshot
GitDiff(path)
GitLineChanges(path)
GitLog
```

Só calcular informação cara quando a superfície correspondente exigir.

Também manter aberta a hipótese de benchmarkar:

- `git2`;
- Git CLI subprocess.

A decisão deve ser evidence-driven.

## Arquitetura de performance proposta

```
                    PRUMO NATIVE
               Rust + egui/eframe
                       │
              ┌────────┴────────┐
              │ Renderer Policy │
              ├─────────────────┤
              │ Glow / OpenGL   │ ← default
              │ WGPU optional   │
              └────────┬────────┘
                       │
            ┌──────────┴──────────┐
            │ Application State   │
            └──────────┬──────────┘
                       │
       ┌───────────────┼────────────────┐
       │               │                │
WorkspaceIndex    DocumentStore    AgentProjection
       │               │                │
 notify/ignore    Tree-sitter       PrumoClient
       │                                │
       └──────────── Shared Runtime ─────┘
                       │
                  prumo serve
                       │
                   Go Core
```

## Performance budgets atualizados

As metas são budgets internos para benchmark, não garantias universais.

| Métrica | Budget inicial |
| --- | --- |
| CPU realmente idle | ~0–0,3% médio no hardware de referência |
| Periodic repaint em idle | 0 |
| Explorer idle I/O | 0 fora de eventos/watchers |
| RAM RSS — build leve | alvo exploratório &lt; 60–70 MB |
| VRAM — Glow baseline | alvo exploratório &lt; 50 MB |
| Threads base | alvo &lt; 10–12 |
| ACP nunca utilizado | 0 subprocessos ACP |
| Whisper nunca utilizado | 0 modelo carregado |
| SSH nunca utilizado | 0 conexão/túnel ativo |

Os valores devem ser medidos separadamente por Linux/Wayland, Linux/X11, Windows e macOS.

## Matriz obrigatória de benchmark gráfico

Cada release candidata do Prumo Native deve poder executar uma pequena matriz de smoke/perf:

| Renderer | Backend | Cold Start | RSS | VRAM | CPU Idle | Frame Time |
| --- | --- | --- | --- | --- | --- | --- |
| Glow | OpenGL | medir | medir | medir | medir | medir |
| WGPU | Auto | medir | medir | medir | medir | medir |
| WGPU | GL | medir | medir | medir | medir | medir |
| WGPU | Vulkan / DX12 / Metal | medir | medir | medir | medir | medir |

## Fallback e recovery de renderer

Persistir não apenas a escolha, mas a saúde do último boot.

Modelo:

```
graphics.renderer = glow
graphics.last_boot_ok = true
graphics.last_backend = opengl
```

Se ocorrer falha precoce de inicialização gráfica:

1. registrar `renderer-init-failure`;
2. no próximo startup oferecer ou executar fallback seguro;
3. nunca entrar em boot loop;
4. disponibilizar CLI:
    
    `prumo-native --renderer glow`
    
    `prumo-native --renderer wgpu`
    
    `prumo-native --wgpu-backend gl`
    
    `prumo-native --safe-graphics`.
    

`--safe-graphics` deve privilegiar o caminho mais amplamente compatível validado para a plataforma e ignorar temporariamente configuração persistida problemática.

## Build profiles

Manter dois conceitos separados:

### Build padrão

Inclui Glow + WGPU para que o usuário possa alternar sem instalar outro binário.

### Build lean opcional

Pode ser produzida futuramente somente com Glow/OpenGL para:

- máquinas antigas;
- distribuição mínima;
- ambientes controlados;
- comparação de binary/RSS.

Não criar matrix de builds excessiva antes de existir necessidade.

## Wave 0 revisada — Performance Baseline antes de features

Antes de acrescentar novas capacidades ao fork:

1. pin exato do Oxi upstream;
2. build WGPU upstream;
3. build Glow;
4. build dual-renderer;
5. medir cold-start/RSS/VRAM/threads/wakeups;
6. reproduzir idle com Explorer aberto e fechado;
7. medir com terminal aberto/fechado;
8. medir com Git panel aberto/fechado;
9. medir com sessão attention badge;
10. medir com ACP nunca usado versus warmed;
11. medir workspace pequeno, médio e grande;
12. salvar evidências no Gauntlet Loop.

Só após esse baseline devem entrar mudanças arquiteturais de performance.

## Ordem de implementação revisada

```
P0.1 dual renderer + persisted renderer preference
P0.2 benchmark Glow vs WGPU
P0.3 remover repaint periódico de atenção
P0.4 WorkspaceIndex + cached Explorer
P0.5 ignore matcher compilado
P0.6 runtime Tokio compartilhado

P1.1 ACP/SSH/voice fully lazy
P1.2 lazy model discovery
P1.3 decompor Git refresh
P1.4 remover default_fonts redundantes
P1.5 reduzir loaders

P2.1 avaliar Tree-sitter/Syntect
P2.2 caches por memory budget
P2.3 optional lean build
```

## Critério de decisão Oxi vs reimplementação

O Oxi deve primeiro ser tratado como um **laboratório mensurável de otimização**, não descartado por impressão subjetiva de peso.

```
Oxi upstream
    ↓
Oxi performance fork
    ↓
P0 applied
    ↓
benchmark
    ├── resultado satisfatório
    │     → continuar fork / modularização
    │
    └── limites arquiteturais persistem
          → extrair componentes comprovados
          → Prumo Native reimplementation
```

Não começar uma IDE nova sem antes medir a diferença que os P0 produzem.

## Princípios permanentes de performance

1. **Zero-cost while unused.**
2. **No filesystem discovery during paint.**
3. **No periodic repaint for static state.**
4. **One renderer active per process.**
5. **Glow/OpenGL is the lightweight baseline until evidence says otherwise.**
6. **WGPU is capability, not mandatory tax.**
7. **Background work uses shared bounded execution resources.**
8. **Expensive state is computed incrementally and on demand.**
9. **Every optimization claim must produce benchmark evidence.**
10. **Accessibility remains a first-class constraint, not a performance toggle.**

## Decisão consolidada deste adendo

<aside>
✅

**Aprovado para investigação/implementação:** o Prumo Native deve manter suporte compilado a **Glow + WGPU**, persistir a escolha do renderer e selecionar exatamente um no startup. **Glow/OpenGL passa a ser o baseline recomendado de baixo consumo.** WGPU permanece disponível, com backend configurável quando necessário. A decisão final por plataforma será baseada na matriz de benchmark do próprio Prumo.

</aside>

A auditoria reforça a decisão anterior de **não reimplementar a IDE do zero neste momento**. Primeiro devemos aplicar os P0 no Oxi, medir e então decidir se o fork continua como base ou se os componentes comprovados serão extraídos para uma arquitetura nova.

## Próxima auditoria aprovada — Dependências do Oxi por custo real

<aside>
📦

**Próxima etapa aprovada:** auditar as dependências diretas e os principais transitivos do Oxi com foco em custo mensurável, classificando cada uma como `Core`, `Lazy`, `Optional`, `Removable` ou `Replaceable`.

</aside>

A auditoria deve evitar decisões por tamanho do `Cargo.lock` ou percepção subjetiva. Cada dependência relevante será avaliada por impacto em:

- RSS / private memory;
- CPU idle;
- wakeups;
- cold start;
- tamanho do binário;
- VRAM quando aplicável;
- quantidade de threads;
- subprocessos;
- custo de compilação;
- superfície de dependências transitivas;
- superfície de segurança/manutenção;
- frequência real de uso no Prumo Native.

### Classificação

| Classe | Significado |
| --- | --- |
| Core | Necessária ao funcionamento base e justificada por uso constante. |
| Lazy | Necessária, mas não deve inicializar recursos até o primeiro uso. |
| Optional | Capability que pode ser desligada, feature-gated ou removida de builds lean. |
| Removable | Dependência sem benefício suficiente para o recorte do Prumo Native. |
| Replaceable | Função necessária, mas existe alternativa mais simples, leve ou já presente no stack. |

### Ordem inicial de investigação

Priorizar dependências e grupos que mais provavelmente alteram consumo ou complexidade:

1. `eframe / egui / wgpu / glow`;
2. `egui_extras` e image loaders;
3. `tokio`;
4. `reqwest / rustls`;
5. `git2 / libgit2`;
6. `tree-sitter` e grammars;
7. `syntect`;
8. `portable-pty / vt100`;
9. `russh`;
10. `cpal / whisper-rs`;
11. `fontdb` e fontes embarcadas;
12. `image`;
13. `walkdir / glob`;
14. keyring/secrets;
15. MCP/ACP-related dependencies.

### Entregável esperado

Produzir uma tabela por dependência:

| Dependência | Função | Classe | Idle RAM | CPU/Wakeups | Startup | Binário | Threads/Processos | Ação |
| --- | --- | --- | --- | --- | --- | --- | --- | --- |
| exemplo | renderer | Optional | medir | medir | medir | medir | medir | manter atrás de feature/config |

### Critério de decisão

Uma dependência só deve ser removida ou substituída quando houver evidência de pelo menos um dos seguintes:

- redução material de RAM/VRAM;
- redução material de wakeups/CPU idle;
- cold-start significativamente menor;
- simplificação estrutural relevante;
- redução de superfície de ataque/manutenção;
- eliminação de duplicação tecnológica;
- possibilidade de tornar capability zero-cost while unused.

A auditoria deve resultar em uma proposta de `Cargo.toml` reorganizado por features e build profiles, incluindo ao menos:

```
default
lean
full
```

ou nomenclatura equivalente, somente se os benchmarks justificarem a existência de múltiplos profiles.

### Regra permanente

> **Dependência não é custo apenas por existir no manifest; o custo deve ser medido no binário e no runtime. Remoções e substituições serão evidence-driven.**
>
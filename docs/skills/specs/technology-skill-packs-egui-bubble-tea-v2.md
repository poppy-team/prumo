# 79.T — Technology Skill Packs: egui, Bubble Tea v2 e Ecossistemas de Frameworks

> Authority: canonical specification.
> Logical ID: 79 T
> Source: Notion Living Book (3dd9bb7d023f81868bbde5eda5cce20f)
> Status: Skill Package de referência para 79 T — Technology Skill Packs egui, Bubble Tea v2.


<aside>
🧰

**Decisão proposta:** o Prumo deve suportar Technology Skill Packs componíveis para frameworks/toolkits recorrentes. Eles especializam uma `lang-*` sem duplicar a linguagem inteira e carregam conhecimento version-sensitive somente quando o projeto realmente usa a tecnologia.

</aside>

# Por que criar Technology Packs

Frameworks como egui e Bubble Tea possuem modelos arquiteturais, lifecycle, input/event semantics, layout, testing, accessibility, performance e ecossistemas próprios. Uma `lang-rust` ou `lang-go` não consegue orientar essas decisões sem ficar gigantesca. Em contrapartida, criar um agent por framework aumentaria workforce e duplicaria responsabilidades. O owner correto é uma família de skills/knowledge packs resolvida pelo stack detector.

# Technology Pack Contract

Todo pack candidato deve declarar:

- `technology_id`, upstream oficial e aliases;
- versões/ranges testados e `last_verified`;
- compatibility matrix com linguagem/toolchain/OS/targets;
- activation e negative triggers;
- dependencies em `lang-*` e cross-cutting skills;
- architecture/lifecycle/event model;
- state/resource/concurrency model;
- UI/layout/style/accessibility quando aplicável;
- official ecosystem modules e third-party status claramente separado;
- testing strategy e deterministic fixtures;
- performance failure modes e profiling;
- security/trust boundaries;
- build/package/cross-platform/deployment;
- migration/upgrade notes;
- source provenance por knowledge unit;
- good/bad examples + evals de implementação e review.

# Pack P0 — Bubble Tea v2 / Charm TUI stack

O próprio Prumo usa `charm.land/bubbletea/v2` e Lip Gloss v2, portanto este pack deve ser **P0**.

## Skill graph sugerido

```
framework-bubbletea-v2
├── requires: lang-go
├── composes: tui-design
├── composes: terminal-accessibility
├── composes: performance-analysis
├── knowledge: bubbletea-v2-core
├── knowledge: lipgloss-v2
├── knowledge: bubbles-v2
├── workflow: tui-architecture
├── workflow: tui-testing
└── workflow: tui-visual-qa
```

## `framework-bubbletea-v2` deve ensinar

- Elm Architecture / Model → Update → View sem transformar `Update` em god-function;
- Commands e side effects fora da lógica pura quando possível;
- mensagens/eventos tipados e ownership de long-running work;
- cancellation, timeouts, tick/subscription-like work e cleanup;
- resize handling e layouts derivados de terminal width/height;
- focus model e key maps centralizados;
- mouse apenas como enhancement, nunca requisito implícito;
- alt-screen vs normal-screen, cursor, paste, terminal capabilities e graceful degradation;
- deterministic `Update` tests por sequências de messages;
- render/golden tests e normalização de ANSI quando fizer sentido;
- no hidden goroutine leaks nem blocking I/O no update/render path.

## Lip Gloss v2 knowledge

Cobrir measurement real de terminal cells, Unicode/wide glyphs, wrapping/truncation, borders/padding/margins, compositing, placement, adaptive colors, no-color/fallback modes e style reuse. Definir um **Terminal Design Token Adapter** que transforme tokens sem impor CSS semantics ao TUI.

## Bubbles v2 knowledge

Componentes oficiais devem ser tratados como primitives reutilizáveis: cursor, file picker, help, list, paginator, progress, spinner, stopwatch, table, textarea, text input, timer e viewport. O pack precisa ensinar quando compor um Bubble existente vs escrever widget próprio e como manter keymaps/focus/help consistentes.

## TUI accessibility

- leitura linear e informação essencial em texto;
- não depender somente de cor, box drawing ou posição;
- shortcuts descobríveis com help/status affordance;
- focus explícito e reversível;
- reduced-motion/animation-light profile;
- narrow-terminal behavior documentado;
- copy/select/readback não deve ser sabotado sem motivo;
- testar combinações reais de terminal + screen reader quando o produto prometer suporte AT.

## Evals Bubble Tea

1. resize agressivo sem overflow/panic;
2. list/table/viewport com dados vazios, longos e Unicode;
3. cancellation de operation lenta;
4. focus traversal somente teclado;
5. 80x24 e terminal estreito;
6. no-color;
7. deterministic message sequence;
8. visual snapshot normalizado;
9. goroutine/resource leak fixture;
10. upgrade fixture v1→v2 onde houver código legado.

## Upstream roots

- [Charm v2 announcement](https://charm.land/blog/v2/)
- [Bubble Tea](https://github.com/charmbracelet/bubbletea)
- [Lip Gloss](https://github.com/charmbracelet/lipgloss)
- [Bubbles](https://github.com/charmbracelet/bubbles)

# Pack P0/P1 — egui / eframe ecosystem

## Skill graph sugerido

```
framework-egui
├── requires: lang-rust
├── composes: desktop-ui
├── composes: design-system
├── composes: accessibility
├── composes: visual-regression
├── composes: performance-analysis
├── knowledge: eframe
├── knowledge: egui-winit
├── knowledge: egui-wgpu
├── knowledge: egui-glow
├── knowledge: egui-extras
├── workflow: egui-custom-widget
├── workflow: egui-kittest
└── workflow: egui-accesskit
```

## `framework-egui` deve ensinar

- immediate-mode mental model e consequências para state, identity e redraw;
- stable widget IDs e colisões de ID;
- layout constraints, available space, clipping, scroll areas, panels/windows e nested UI;
- state persistence sem criar um retained-mode paralelo acidental;
- custom painting/widgets via response/sense/painter;
- style/visuals/spacing/fonts centralizados por design system adapter;
- não bloquear UI thread: async/background work via channels/watch/promise patterns e repaint signaling;
- separation de app/core para testabilidade;
- viewport/multi-window e integration boundaries quando relevante;
- renderer/backends `egui-wgpu`/`egui_glow` e `egui-winit` sem duplicar o conhecimento de wgpu/winit.

## Accessibility / AccessKit

O pack deve exigir semantics para custom widgets. Built-ins já alimentam a árvore; labels separados devem ser relacionados; widgets custom precisam role/name/state coerentes. `egui_kittest` consulta a mesma árvore AccessKit, então testes semânticos devem ser preferidos a coordinate-click tests sempre que possível.

## Testing

- `egui_kittest::Harness` para interações e state transitions;
- queries por role/name/label;
- screenshot/snapshot com renderer controlado;
- tolerâncias explícitas e por OS/backend quando inevitável;
- fixtures por window size/density/theme/locale;
- baseline jamais autoaceito só para “ficar verde”.

## Performance

- distinguir CPU UI construction, tessellation, texture upload e GPU render;
- evitar trabalho pesado por frame;
- caches/memoization somente medidos;
- virtualização/partial rendering para data-dense UI quando necessário;
- medir frame time e allocation hotspots antes de otimizar.

## Evals egui

1. custom widget com AccessKit correto;
2. modal/menu/popover/focus interactions;
3. data-dense inspector/outliner com resize;
4. async operation sem freeze;
5. theme/tokens light-dark-high-contrast;
6. snapshot deterministic;
7. keyboard-only flow;
8. 100%/200% scale e small-window layout;
9. renderer feature matrix;
10. migration fixture entre minor releases relevantes.

## Upstream roots

- [egui repository / architecture](https://github.com/emilk/egui)
- [egui accessibility](https://github.com/emilk/egui/blob/main/docs/accessibility.md)
- [egui_kittest](https://docs.rs/egui_kittest/latest/egui_kittest/)
- [AccessKit](https://github.com/AccessKit/accesskit)

# Technology packs candidatos P1

Não criar todos automaticamente. Usar Project Portfolio Scan + anti-duplication RFC. Candidatos recorrentes no portfólio podem incluir:

- `framework-slint` + Slint renderer/platform/testing/accessibility knowledge;
- `framework-gpui` + component-kit/ecosystem knowledge quando usado;
- `framework-floem` quando projeto adotar Floem;
- `graphics-wgpu`, `windowing-winit`, `graphics-vello` como packs infra reutilizáveis;
- `framework-wails` quando Go + webview desktop for stack real;
- `framework-avalonia` para C# desktop quando projeto ainda depender dele;
- web framework packs somente para stacks realmente detectadas (Svelte/React/etc.);
- engine/editor packs específicos apenas quando regras e tests diferirem materialmente da composição genérica.

# Detecção e activation

Stack detector deve combinar manifests (`Cargo.toml`, `go.mod`, package manifests), imports, lockfiles e project config. Import transitivo isolado não deve ativar skill automaticamente. Activation precisa distinguir **direct dependency / source usage / migration target / comparative research**.

# Anti-explosion rule

Crate/library pequena não ganha skill própria por padrão. Preferir `knowledge/<library>.md` dentro do pack pai. Promover para skill independente somente quando houver activation distinta, workflow próprio, outputs/evidence próprios e eval gain mensurável.

# Exit Gate

Technology Pack só chega a `recommended/verified` quando melhora implementação/review em evals reais, possui upstream roots versionados, não duplica `lang-*`, e seu context budget é inferior a carregar documentação indiscriminadamente.

# Portfolio Technology Matrix — expansão 2026-09-16

A priorização abaixo deriva de recorrência real nos projetos + necessidade de conhecimento próprio. `P0` significa uso direto/core do Prumo ou alto risco de orientação genérica incorreta; `P1` significa recorrente/importante; `P2` é condicionado a novo uso real.

| Pack / knowledge | Pri. | Forma recomendada | Motivo / foco |
| --- | --- | --- | --- |
| **framework-bubbletea-v2** | P0 | Skill + knowledge ecosystem | Prumo/TUIs: TEA, Msg/Cmd, View v2, focus/keymaps, resize, cancellation, terminal capabilities, Bubbles/Lip Gloss, tests. |
| **framework-egui** | P0/P1 | Skill + eframe/renderer/testing knowledge | Immediate mode, IDs/state, custom widgets, async/repaint, AccessKit, egui_kittest, visual/perf. |
| **graphics-wgpu** | P1 | Skill | Cross-platform GPU abstraction, adapter/device/surface lifecycle, capabilities/limits/features, errors/loss, tracing/debugging, shaders, uploads, synchronization, profiling. |
| **framework-slint** | P1 | Skill + renderer/platform/testing knowledge | Declarative `.slint`, Rust integration, backend/renderer matrix, accessibility feature, viewer/screenshot automation, system testing, desktop/embedded constraints. |
| **framework-gpui** | P1 experimental | Skill | Hybrid retained/immediate model, Entity/Context/View/Element, actions/focus/input, async executor, tests; pre-1.0 exige pin/freshness curto. |
| **framework-floem** | P1 experimental | Skill | Signals/effects, view tree, Taffy flex/grid, virtual lists, renderer matrix; pre-v1 e breaking changes exigem pin. |
| **graphics-vello** | P1 | Knowledge pack sob graphics/UI | 2D vector rendering sobre wgpu, scene construction, render-to-texture/surface, text/image/gradient pipeline e performance. |
| **windowing-winit** | P1 | Knowledge pack | Event loop/window/input/IME/scale-factor/platform lifecycle compartilhado por egui/Slint/graphics stacks. |
| **accessibility-accesskit** | P1 cross-cutting | Knowledge pack | Accessibility tree/adapters nativos e mapping semântico reutilizado por toolkits Rust custom-rendered. |
| **tooling-tree-sitter** | P1 | Skill | Incremental parsing, grammar/query lifecycle, highlighting/injections/locals, parser tests, editor embedding e language tooling. |
| **compiler-cranelift** | P1 | Skill | IR/codegen/frontend/module/JIT/object, ISA/ABI, verification, object emission, optimization boundaries e compiler differential tests. |
| **browser-wpe-webkit** | P1 | Skill | Embedded/low-footprint WebKit, engine/view lifecycle, accelerated rendering/media, JS/runtime integration, process/security/performance boundaries. |
| **browser-servo** | P1 experimental | Skill | Embedding WebView/ServoBuilder/EventLoopWaker/RenderingContext, delegates/permissions, evolving embedding API e compatibility evidence. |
| **graphics-opengl** | P1 | Skill/profile | Desktop/low-spec graphics projects: context/profile/extensions, resource lifetime, shader pipeline, debug output, state hazards, portability/perf. |
| **tooling-lsp** / **tooling-dap** | P1 | Skill/knowledge conforme catálogo existente | Editors/languages: protocol capabilities, lifecycle, cancellation, diagnostics, semantic tokens, incremental sync, test harnesses. |

## Slint: knowledge obrigatório

A documentação atual separa backend de renderer; Rust pode selecionar renderers como software, Skia, FemtoVG/wgpu e Vello experimental. A feature de accessibility integra APIs do SO e o `slint-viewer` consegue live reload e screenshots. Portanto `framework-slint` deve ter fixtures por backend/renderer, escala, accessibility e screenshot; no Web/Wasm, registrar explicitamente que a UI é canvas/WebGL e a própria documentação alerta que screen readers não estão disponíveis nesse modo.

## GPUI: freshness policy

GPUI é descrito pelo próprio upstream como **pre-1.0** e sujeito a breaking changes. Seu pack deve usar commit/version pin, `review_after_days` curto e exemplos extraídos da mesma versão. Cobrir `Application`, `Entity`, `Context`, `Render/View`, low-level `Element`, actions/keybindings, async contexts, virtualization e `#[gpui::test]`/`TestAppContext`.

## Floem: freshness policy

Floem também declara maturação pré-v1. O pack deve cobrir fine-grained reactivity via signals/effects, single-built view tree, virtualized lists, Taffy Flex/Grid e renderer selection (`vger`/Vello/Skia/tiny-skia conforme versão). Não congelar APIs em prosa sem range de versão.

## wgpu + Vello

`graphics-wgpu` deve ser independente porque serve múltiplos frameworks/produtos. Knowledge de Vello fica inicialmente subordinado a graphics/UI: Vello é um renderer 2D sobre wgpu, e não precisa repetir device/surface/resource contracts. Promover Vello a skill independente somente se workflows/evals específicos justificarem.

## WPE WebKit / Servo

`browser-wpe-webkit` deve ser production-oriented para browser/embedded use cases de baixo consumo; WPE declara explicitamente foco em dispositivos embedded/low-consumption, performance e footprint. `browser-servo` deve nascer experimental porque o embedding API segue em evolução; knowledge deve registrar breaking changes por release e separar `servoshell` de APIs embeddable.

## Cranelift / Tree-sitter

Esses packs são transversais aos projetos de linguagens e editores. Cranelift precisa compor `compiler-testing`, `abi-compatibility`, `object-format` e `lang-rust`; Tree-sitter compõe `parser-testing`, `editor-tooling`, `syntax-highlighting` e language packs sem duplicar a gramática específica de cada linguagem.

## Referências upstream adicionais

- [Slint documentation](https://docs.slint.dev/latest/docs/slint/)
- [GPUI README](https://github.com/zed-industries/zed/tree/main/crates/gpui)
- [Floem](https://github.com/lapce/floem)
- [wgpu documentation](https://wgpu.rs/doc/wgpu/)
- [Vello](https://docs.rs/vello/)
- [Servo Book](https://doc.servo.org/)
- [WPE WebKit](https://webkit.org/wpe/)
- [Tree-sitter](https://tree-sitter.github.io/tree-sitter/)
- [Cranelift](https://docs.rs/cranelift/)

# Novos technology families — 2026-09-16

Adicionar à matriz de Technology Packs: `platform-webassembly`, `graphics-webgpu`, `framework-threejs`, `advanced-linux-platform` e technology packs de bancos quando recorrentes. WebAssembly/WebGPU/Linux não são apenas bibliotecas: possuem runtime/platform/capability contracts próprios e devem compor language skills, não ser absorvidos por elas.
# 79.L — Frontend, Design System, UI Implementation, Visual Quality e Web

> Authority: canonical specification.
> Logical ID: 79 L
> Source: Notion Living Book (3d89bb7d023f818683bdeddd03d7852a)
> Status: Skill Package de referência para 79 L — Frontend, Design System, UI Implementation,.


<aside>
🎨

O objetivo desta família é separar **design intent**, **component contract**, **implementation**, **visual fidelity** e **UX/accessibility review**. Hoje algumas dessas responsabilidades se sobrepõem ou usam procedimentos genéricos.

</aside>

| Skill | Pri. | Aprimoramento | Aceite |
| --- | --- | --- | --- |
| **design-system** | P0 | Reestruturar em foundations, tokens, components, patterns, governance, versioning/migrations, accessibility e testing. | Design system possui lifecycle/API e coverage score. |
| **design-tokens** | P0 | Schema para primitive/semantic/component tokens, aliases, themes/modes, naming, transformations, migrations e contrast checks. | Tokens compiláveis e schema-valid. |
| **component-specification** | P1 | Anatomy, props/slots/events, variants/states, keyboard/focus/SR, responsive, tokens, errors e semver. | Engineer implementa sem decisões implícitas relevantes. |
| **frontend-web** | P0 | Browser/SSR/CSR/hydration, state/errors, responsive, a11y, security, performance, i18n e testing matrix. | Compatibility evidence por browser/runtime. |
| **ui-implementation** | P0 | Mapear component/states, responsive, keyboard/SR, i18n, loading/error/empty, perf e visual regression. | Nenhum state especificado fica sem implementation/test. |
| **ui-ux-review** | P1 | Aggregator de interaction/consistency/usability/fidelity; delegar WCAG e pixel regression. | Finding encaminhado ao specialist owner correto. |
| **visual-qa** | P0 | Layout/fidelity/overflow/z-index/font/render anomalies, device/browser matrix, severity/tolerances. | Annotated evidence + tolerance rule. |
| **visual-regression** | P0 | Baseline governance, deterministic rendering, masks/dynamic regions, thresholds, matrix e artifact retention. | Baseline nunca auto-atualizado apenas para passar. |
| **performance-web** | P1 | Lab + field/RUM, CWV, bundle/network/render budgets, device/network profiles e regression thresholds. | Baseline/target por page/flow. |
| **website-forensics** | P1 | Metodologia real para DOM/CSS/assets/fonts/layout/responsive/interaction/perf/a11y inference e confidence. | Observação objetiva separada de hipótese. |
| **competitor-analysis** | P1 | Feature normalization, date/version, source confidence, pricing/licensing where relevant, strengths/weaknesses e anti-copying. | Comparação temporalmente rastreável. |
| **design-research** | P1 | Research question, method, source/sample, bias, confidence, ethics/privacy e traceability. | Insight sem evidence não vira decision. |
| **visual-reference-research** | P1 | Source/licensing/date, visual taxonomy, pattern abstraction e non-copying guidance. | Reference pack rastreável. |

# UI State Contract

Para componente/flow crítico, considerar quando aplicável: default, hover, focus, active, selected, disabled, loading/pending, empty, error, success, offline, permission-denied, destructive confirmation, partial data, long text/locale expansion.

# Design System as API

Breaking token/component change deve possuir version bump/migration guidance. Deprecation precisa de replacement e janela. Visual regression e accessibility são consumidores do design system; não devem morar apenas como texto dentro dele.

# Visual QA vs Visual Regression

- **visual-qa:** pergunta “há problema visual?”; pode encontrar defeito novo sem baseline.
- **visual-regression:** pergunta “o output mudou em relação a baseline aprovado?”.
- **ui-ux-review:** pergunta “o comportamento/experiência é coerente e útil?”.
- **accessibility:** pergunta “a experiência atende requirements acessíveis?”.

# Internationalization integration

Mesmo antes de criar skill dedicada, frontend/UI contracts devem prever resource-based strings, text expansion, number/date formatting, RTL readiness quando aplicável e pseudo-localization em projetos multilíngues.

# Exit Gate

Design e implementation contracts são separados; P0 visual/UI deixam de ser boilerplate; baselines visuais têm governance; a11y/UX não são substituídos por screenshot review.

# Atualização de auditoria — 2026-09-16

<aside>
🎛️

**Objetivo revisado:** uma UI skill madura não pode encerrar em “criar layout consistente”. Ela deve produzir um pacote de especificação suficientemente preciso para implementação, review e regressão sem deixar decisões visuais/behaviorais críticas implícitas.

</aside>

## UI Documentation Pack v2 — artefatos obrigatórios quando aplicáveis

1. **Problem / users / tasks / constraints** com platform, input modalities, target devices, density e accessibility profile.
2. **Reference board rastreável:** referências web/imagem, data/fonte, versão/produto, o que foi observado, o que é apenas hipótese, padrões abstraídos e anti-copying note.
3. **Information architecture + user flows:** happy/alternate/error/recovery/cancel/offline/permission flows.
4. **Wireframes multi-state e adaptive:** regiões, hierarchy, min/preferred/max sizing, scrolling, overflow, split panes, resizable panels, collapse behavior e narrow-window fallback.
5. **Layout/Grid contract:** container strategy, columns, gutters, margins, alignment, spacing rhythm, density modes, baseline/grid quando útil e window-size classes. Evitar breakpoints universais fixos: derivar de conteúdo/plataforma.
6. **Layer system:** canvas/content/panels/popovers/tooltips/menus/modals/toasts/drag overlays/focus rings com stacking rules e clipping/portal behavior.
7. **Design tokens:** primitive → semantic → component; aliases, themes/modes, state tokens e transformations. Adotar DTCG 2025.10 como formato interoperável preferencial onde JSON tokens fizerem sentido.
8. **Typography:** families/fallbacks, scale, weights, line height, tracking, truncation/wrapping, numeric/tabular needs, locale expansion e font-loading behavior.
9. **Color:** surfaces/text/borders/icons/accent/status/data-viz, light/dark/high-contrast, contrast evidence e color-not-alone rules.
10. **Component contracts:** anatomy, variants, sizes, slots, tokens, content rules, events/actions, keyboard/pointer/touch behavior e state machine.
11. **State matrix:** default, hover, focus-visible, pressed/active, selected, checked, expanded/open, disabled, read-only, loading/pending, empty, error, warning, success, offline, permission-denied, drag/drop, destructive-confirmation e long-content states conforme componente.
12. **Forms:** label/help/placeholder policy, required/optional, validation timing, inline/global errors, submit/pending/success, autocomplete, keyboard ordering e recovery.
13. **Complex overlays:** dropdown/combobox/menu/context-menu/popover/tooltip/modal/drawer — trigger, anchoring, collision/flip, dismissal, focus, escape, nesting e viewport boundaries.
14. **Data-dense UI:** tables/trees/outliners/property inspectors/toolbars/tabs/docks/panels — resizing, virtualization, selection, sorting/filtering, hierarchy, disclosure, row density e bulk actions.
15. **Motion/feedback:** duration/easing intent, progress, skeleton/spinner, optimistic state, reduced-motion alternative, transient feedback/toast duration.
16. **Accessibility matrix + implementation mapping:** semantic roles/names/states/actions, keyboard map, focus graph, AT expectations e mapping para framework/toolkit.
17. **Visual QA plan:** reference screenshots, deterministic fixtures, view/window matrix, expected tolerances, baseline ownership e deviation log.

## Vision-aware visual research workflow

Quando o modelo/harness possuir vision e web/image retrieval, `visual-reference-research` deve: coletar múltiplas referências relevantes; registrar fonte/data/contexto; inspecionar screenshots; decompor layout, spacing, hierarchy, density, typography, color, components, interaction affordances e edge states; comparar padrões entre produtos; sintetizar decisões sem copiar assets/trade dress; e produzir confidence por inferência. Quando vision não estiver disponível, a skill deve declarar a limitação e nunca fingir ter inspecionado pixels.

## Component State Schema proposto

Cada component spec deve expor, no mínimo: `anatomy`, `variants`, `sizes`, `states`, `tokens`, `content_rules`, `actions`, `keyboard`, `focus`, `pointer`, `accessibility`, `responsive_adaptive`, `i18n`, `errors`, `performance_notes`, `test_matrix` e `visual_examples` quando aplicável.

## Design Tokens baseline

Usar a especificação estável **DTCG 2025.10** como referência de interoperabilidade; drafts posteriores podem ser pesquisados, mas não devem substituir silenciosamente o baseline estável. Transformações específicas (`CSS`, Rust/egui, Go/Lip Gloss, Slint etc.) são adapters/outputs, não a fonte de verdade do token model.

## Referências canônicas adicionadas

- [Design Tokens Community Group — stable 2025.10](https://www.designtokens.org/tr/2025.10/)
- [ARIA APG patterns](https://www.w3.org/WAI/ARIA/apg/patterns/)
- [WCAG 2.2](https://www.w3.org/TR/WCAG22/)

# Revisão 2026-09-16 — WebGPU e Three.js

`lang-wgsl` não deve ser owner da API WebGPU. Criar `graphics-webgpu` para adapter/device/queue/resources/pipelines/bind groups/features/limits/error scopes/device loss/CTS/performance e manter WGSL como linguagem. Three.js deve ganhar `framework-threejs` version-sensitive com perfis WebGLRenderer/WebGPURenderer, resource disposal, loaders/glTF, color management, PBR, scene graph, culling/instancing, TSL/node materials e migration knowledge. A migração atual para WebGPURenderer muda material/shader/post-processing APIs e exige knowledge ligado à release.

# Amendment 2026-09-21 — UI Interaction Manifest

[84 — Total Assurance Constitution: Gauntlet Loop, Evidence e Anti-False-Green](../../quality/gauntlet-84.md) e 84.E passam a definir o fechamento de UI. Todo controle visível/interativo precisa ser inventariado e ligado a behavior/evidence. Dead control é blocker. Premium quality inclui microinteractions, task-flow proof, clean-state usability, input/focus hierarchy, accessibility, localization, DPI/responsive e performance percebida; visual regression continua incapaz de substituir UX ou functional interaction tests.
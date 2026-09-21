# 79.E — Accessibility, UX e Cognitive Quality: Skills P0/P1

> Authority: canonical specification.
> Logical ID: 79 E
> Source: Notion Living Book (3d89bb7d023f81408b76eaac40d47b34)
> Status: Skill Package de referência para 79 E — Accessibility, UX e Cognitive Quality Skill.


<aside>
♿

A auditoria encontrou um dos maiores clusters de **especialização nominal com procedimento genérico** neste domínio. A prioridade não é adicionar mais checklists; é transformar cada micro-skill em uma unidade verificável com responsabilidade própria.

</aside>

# Arquitetura da família

`accessibility` deve funcionar como **orchestrator/umbrella skill**, delegando detalhes às micro-skills. `ui-ux-review` não deve reimplementar WCAG. `visual-qa` não deve decidir semântica de accessibility. Cada owner deve ser único.

| Skill | Pri. | Responsabilidade alvo | Aprimoramentos obrigatórios | Evidence / aceite |
| --- | --- | --- | --- | --- |
| **accessibility** | P0 | Orquestração WCAG e scorecard. | Mapear requisitos → critérios WCAG 2.2; selecionar subskills; consolidar severity, exceptions e retest; não duplicar procedimentos detalhados. | Scorecard por criterion + links para evidências especializadas. |
| **contrast** | P0 | Contraste textual/não textual e estados. | Normal/large text, UI graphics, focus/error/hover/disabled, gradients/images, token-level verification; APCA apenas complementar se usado. | Contrast matrix automatizada por token/componente + exceptions justificadas. |
| **focus-management** | P0 | Transições de foco. | Modals, popovers, route changes, focus trap, restoration, initial focus, async updates, roving tabindex, DOM removal. | Focus-transition graph exercitado por keyboard tests. |
| **keyboard-accessibility** | P0 | Operabilidade sem pointing device. | Tab order, native semantics, composite widgets, roving tabindex, shortcuts/chords, conflicts, traps e conventions por plataforma. | Critical flows completos somente com teclado. |
| **motion-accessibility** | P0 | Redução/remoção segura de movimento. | `prefers-reduced-motion`, essential vs decorative motion, parallax/autoplay/vestibular triggers, alternative transitions. | Motion inventory + reduced-motion behavior validado. |
| **screen-reader** | P0 | Accessibility tree e AT real. | Accessible name, role/state/value, reading order, live regions, announcements, dynamic changes; matriz NVDA/JAWS/VoiceOver conforme plataforma suportada. | Evidence de accessibility tree/AT, não inspeção visual genérica. |
| **zoom-reflow** | P0 | Zoom, text spacing e reflow. | 200/400%, horizontal-scroll exceptions, fixed/sticky UI, text spacing, orientation e responsive reflow. | Cenários reproduzíveis com viewport/zoom definidos. |
| **cognitive-clarity** | P1 | Clareza de colaboração e interface. | Separar regras de conversa humana de execução autônoma; decision budget, output density, plain error language, predictable structure, undo/recovery cues. | Não induz parada desnecessária e melhora comprehension evals. |
| **ux-architecture** | P0 | Arquitetura de navegação/interaction global. | Information hierarchy, navigation model, global patterns, cross-flow consistency, responsive/i18n/a11y NFRs e measurable UX criteria. | Architecture spec reutilizável por múltiplos flows/components. |
| **information-architecture** | P1 | Taxonomia, findability e crescimento. | Hierarchy, labels, navigation, search, card sorting/tree testing quando aplicável, content growth e ambiguity analysis. | Findability tests ou rationale documentada. |
| **interaction-design** | P1 | Estados e transições de interação. | State machines, modality, affordance, latency feedback, disabled/error/pending/cancel/retry e parity entre input methods. | State/transition table por interação crítica. |
| **interaction-research** | P1 | Pesquisa de usabilidade/interação. | Protocol, tasks, observation coding, confounders, confidence, qualitative/quant metrics, decision linkage. | Observação separada de interpretação. |
| **user-flows** | P1 | Jornadas completas. | Happy, alternate, error, cancel, retry, offline, permission denied, session expiry, recovery; entry/exit criteria. | Todo critical flow inclui recovery path. |
| **wireframing** | P1 | Especificação estrutural de tela. | Schema para regions, hierarchy, states, flows, content density, breakpoints e a11y/interaction annotations. | Wireframe passa linter/checklist determinístico. |
| **wireframe-styleguide** | P1 | Linguagem comum de wireframes. | Naming, region notation, state notation, responsive rules, a11y annotations, component references. | Dois agents geram specs compatíveis. |

# Matriz de ownership

| Pergunta | Owner | Não é owner |
| --- | --- | --- |
| “É operável por teclado?” | keyboard-accessibility | visual-qa |
| “O foco foi para o local correto?” | focus-management | interaction-design |
| “O AT anuncia corretamente?” | screen-reader | accessibility umbrella |
| “O contraste passa?” | contrast | design-system |
| “A animação respeita reduced motion?” | motion-accessibility | visual-regression |
| “O layout reflowa no zoom?” | zoom-reflow | responsive design genérico |
| “O fluxo é compreensível e recuperável?” | user-flows / ux-architecture | visual-qa |

# Coverage mínima de evals

- accessibility umbrella: seleção correta de subskills e ausência de over-activation;
- keyboard/focus/screen-reader: fixtures de widgets simples e complexos;
- motion/contrast/zoom: positive/negative boundary fixtures;
- UX skills: casos com requirements ambíguos para testar quando pedir decisão vs aplicar default documentado;
- false-positive cases: backend-only, CLI-only ou mudança sem UI não deve ativar toda a família.

# Exit Gate da família

Nenhuma micro-skill P0 permanece `stable` com procedimento genérico, nenhuma usa “visual or keyboard check” como substituto de evidence específica, e `accessibility` passa a agregar sem duplicar detalhes.

# Atualização de auditoria — 2026-09-16

<aside>
🧭

**Ampliação de escopo:** accessibility deixa de ser tratada primariamente como web. A família deve possuir profiles `web`, `native-desktop`, `terminal/TUI`, `mobile` e `closed/custom-rendered UI` quando aplicável. O baseline web continua WCAG 2.2; para software não-web, usar WCAG2ICT como guidance informativa e complementar com APIs/padrões da plataforma.

</aside>

## Accessibility Platform Matrix

- **Web:** WCAG 2.2 + ARIA/APG + semantic HTML; automated checks são somente uma camada.
- **Native desktop:** WCAG2ICT + accessibility tree da plataforma: Windows UI Automation, macOS NSAccessibility, Linux AT-SPI; toolkits custom-rendered devem expor roles, names, state, value, relationships e actions.
- **egui:** modelar AccessKit como primeira classe; custom widgets precisam `WidgetInfo`/roles/names; `egui_kittest` deve testar a mesma árvore acessível consumida por assistive technology.
- **Terminal/TUI:** keyboard-first, leitura linear coerente, não depender apenas de cor/símbolo/posição, modo sem animação/efeitos quando necessário, evitar cursor/focus traps e validar fluxos em terminais/AT reais para produtos que declarem suporte.

## Evidence mínima revisada

Automation (`axe`, linters, tree assertions) nunca substitui manual/AT evidence. Para fluxos críticos, o pacote deve conseguir emitir: criterion/profile, plataforma, tecnologia assistiva, input modality, viewport/zoom/density, passos reproduzíveis, resultado esperado/observado, artifact/screenshot/tree dump e retest status.

## Referências canônicas

- [WCAG 2.2](https://www.w3.org/TR/WCAG22/)
- [WCAG2ICT 2.2](https://www.w3.org/TR/wcag2ict-22/)
- [ARIA Authoring Practices Guide](https://www.w3.org/WAI/ARIA/apg/)
- [egui accessibility / AccessKit](https://github.com/emilk/egui/blob/main/docs/accessibility.md)

# Teaching integration — boundary com accessibility

Cognitive accessibility define **como apresentar** informação de forma acessível; Teaching define **como a aprendizagem progride**. O primeiro não deve virar tutor implícito em toda conversa. O segundo é ativado explicitamente pelo usuário ou por intents claros de aprender/entender/praticar. Ambos podem compor-se, com UDL/cognitive-accessibility como foundation e Teaching Runtime como controlador de progressão, hints e assessment.
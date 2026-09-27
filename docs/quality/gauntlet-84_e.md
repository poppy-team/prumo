# 84.E — UI/UX, Interaction Fidelity, Accessibility e Human-Factor Assurance

> Authority: canonical specification.
> Logical ID: 84 E
> Source: Notion Living Book (3e29bb7d023f8173848cc4fa2995af62)
> Status: Constituição 84: Perfis do Gauntlet / Total Assurance (84 E — UI UX, Interaction Fidelity, Accessibility).


<aside>
🎛️

**Princípio:** “a tela abriu” não é validação de UI. Toda superfície interativa deve ser tratada como contrato comportamental e humano.

</aside>

# Interaction Manifest

Cada controle crítico registra:

ID, role, label token, icon, tooltip, shortcut, focus semantics, enabled rule, disabled reason, states, command/action, side effect, undo behavior, persistence, test e evidence.

# State completeness

Cobrir quando aplicável:

default, hover, pressed, focused, selected, disabled, loading, empty, error, warning, success, readonly, permission-denied, offline, conflict, unsaved, partial data, dragging e destructive confirmation.

# Reachability

Feature implementada mas inacessível é falha. Cada capability user-facing precisa de entry point coerente e discoverability compatível com frequência/importância.

# Task-flow proof

Além de componente isolado, executar tarefas completas como usuário. Medir falhas de descoberta, contexto perdido, necessidade de ação secreta, ausência de feedback e recovery ruim.

# First-run and clean-state

Executar em config/profile limpo. Evitar validar UI dependente de cache, preferência ou layout de developer.

# Input matrix

Mouse, keyboard, mixed input, focus loss, window resize, high DPI, localization e platform-specific behavior quando aplicável.

# Microinteraction discipline

Validar menus, submenus, dropdowns, popovers, modals, splitters, scrollbars, tooltips, drag-and-drop, numeric fields, sliders, context menus, empty CTAs e icon-only controls.

# Accessibility

Keyboard path, focus visible/order, accessible labels, no color-only meaning, contrast, zoom/text scaling, reduced motion e assistive semantics quando stack suportar.

# Localization

Pseudo-locale, expansion, locale-sensitive number/date, truncation, fallback e missing token. UI não pode depender do tamanho do inglês.

# Visual quality

Separar:

- design intent;
- implementation fidelity;
- visual QA;
- regression;
- UX;
- accessibility.

Screenshot baseline não substitui UX review.

# Premium quality

“Premium” é consistência, hierarchy, spacing, typography, iconography, feedback, density, predictability e acabamento — não excesso de shadows/animation.

# Responsive and DPI

Testar dimensões contínuas, não só snapshots fixos. Splitters/panels devem respeitar min/max e evitar esmagar a tarefa primária.

# Error/recovery UX

Falhas precisam dizer o que aconteceu, impacto e ação possível. Evitar silent failure e modal indiscriminada.

# Dead-control gate

Controle visível sem comportamento real é blocker do profile UI. Disabled intencional precisa de semântica clara e, quando útil, motivo.

# Performance perception

Latency percebida, frame stalls, delayed feedback e blocking work fazem parte de UX e podem acionar performance specialist.
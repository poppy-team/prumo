# Catálogo de Skills de Precisão (189 Skills)

As **Skills** no Prumo são módulos de procedimento canônicos, estruturados com pré-condições, ferramentas permitidas, limites operacionais e critérios verificáveis de conclusão.

## Como Funcionam as Skills

Cada skill do catálogo define:
1. **Intenção Precisa**: Um propósito inequívoco (ex: refatoração de AST, criação de paleta de cores acessível, auditoria de lockfile).
2. **Contexto Requerido**: O subconjunto mínimo de arquivos que o agente deve examinar (princípio Lean Progressive Context).
3. **Restrições de Execução**: Quais ferramentas ACI podem ser invocadas.
4. **Critério de Saída (Done Definition)**: O teste ou evidência que comprova o sucesso da operação.

## Grupos Funcionais de Skills

### 🎨 Design, UI/UX, Vetores & Comunicação Visual

- **Design Systems & Tokens**: `design-system-architect`, `design-tokens-specifier`, `color-palette-generator`, `typography-scale-builder`, `spacing-system-designer`, `theme-switcher-implementer`.
- **Componentização & UI**: `accessible-component-builder`, `form-control-designer`, `data-table-designer`, `modal-dialog-builder`, `toast-notification-system`, `responsive-layout-grid`.
- **Arte Vetorial, Ícones & SVG**: `svg-icon-author`, `vector-path-optimizer`, `scalable-logo-generator`, `svg-illustration-composer`, `canvas-renderer-adapter`.
- **Motion & Microinterações**: `css-animation-choreographer`, `motion-physics-easing`, `page-transition-designer`, `skeleton-loader-animator`.
- **Acessibilidade & Ergonomia**: `wcag-aaa-auditor`, `screen-reader-announcer`, `focus-management-specifier`, `high-contrast-mode-checker`.
- **Marketing, Marca & Publicidade**: `brand-identity-guideline-author`, `hero-section-copywriter`, `social-share-card-generator`, `conversion-rate-optimizer-designer`.

### ⚙️ Engenharia de Compiladores, Runtimes & Sistemas

- **Análise Léxica & Parsing**: `lexer-tokenizer-builder`, `recursive-descent-parser`, `ast-node-definer`, `syntax-error-reporter`.
- **Semântica & Tipagem**: `type-inference-solver`, `scope-resolution-analyzer`, `contract-checker-static`, `borrow-checker-emulator`.
- **Geração de Código & Bytecode**: `bytecode-assembler-emitter`, `virtual-machine-dispatcher`, `js-es2022-transpiler`, `wasm-module-builder`.
- **Concorrência & Tempo**: `async-await-state-machine`, `cooperative-scheduler-designer`, `virtual-clock-tester`, `channel-mailbox-handler`.
- **Memória & Performance**: `arena-allocator-engineer`, `memory-leak-hunter`, `cache-locality-optimizer`, `flamegraph-profiler`.

### 🏗️ Arquitetura, Governança & Qualidade

- **Governança & Prumo Protocol**: `goal-lifecycle-manager`, `sha256-integrity-locker`, `plan-dag-validator`, `evidence-collector`, `gate-evaluator`.
- **Lean Progressive Context (LPC)**: `context-capsule-compiler`, `megaprompt-reducer`, `token-budget-enforcer`, `document-relevance-scorer`.
- **Testes & Conformance**: `differential-testing-runner`, `fuzz-harness-generator`, `golden-master-comparer`, `flaky-test-eliminator`.
- **Segurança & Hermeticidade**: `sandbox-boundary-enforcer`, `secret-leak-detector`, `dependency-cve-auditor`, `env-var-isolation-checker`.
- **Documentação Viva**: `living-book-sync`, `adr-author`, `contradiction-detector`, `authority-drift-checker`.

## Invocando Skills via CLI

As skills podem ser invocadas diretamente no harness ou inspecionadas:

```bash
# Listar todas as skills catalogadas
prumo workforce skills list

# Inspecionar os detalhes e ferramentas de uma skill
prumo workforce skill inspect design-tokens-specifier

# Executar uma tarefa guiada por skill
prumo run --skill accessible-component-builder --path ./src/components
```

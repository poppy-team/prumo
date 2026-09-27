# Prumo Harness vs. OpenCode, Claude Code & DeepSeek Harness

> **Authority:** Canonical engineering specification.  
> **Relates to:** `docs/harness/agent-runtime.md`, `docs/harness/gap-register.md`, `docs/security/security-control-plane.md`, `docs/systems/systems-engineering-layer.md`, `docs/adr/016-product-split-agent-and-harness.md`, `docs/adr/017-single-level-subagent-delegation.md`.

---

## 1. Princípios Inegociáveis da Filosofia do Prumo

Qualquer aprimoramento absorvido do mercado deve se submeter estritamente aos princípios fundacionais do Prumo:

1. **Neutralidade de Provedor (Provider-Neutral Core):** O framework nunca deve acoplar suas políticas ou arquitetura a um provedor proprietário específico. O mesmo loop deve operar com Claude, OpenAI, DeepSeek, Gemini ou modelos locais (llama.cpp, vLLM).
2. **Lean Progressive Context (LPC/PCA):** Menor contexto suficiente, expansão progressiva sob demanda e ponteiro sobre payload. O repositório nunca é pré-carregado no contexto.
3. **O LLM Nunca é a Raiz de Confiança:** O modelo propõe; políticas determinísticas decidem; o runtime impõe; testes falsificam; verificadores independentes avaliam; o release gate aprova. Autoaprovação pelo implementador é expressamente proibida.
4. **Delegação de Nível Único com Worktrees (ADR 017):** A delegação para subagentes é de profundidade fixa zero (`max_delegation_depth: 0`, `may_delegate: false` para filhos) com isolamento em worktrees Git dedicadas.
5. **Substrato Binário Nativo e Eficiente em Go:** Inicialização instantânea, processo compilado único, local-first, com zero dependência de runtimes pesados (Node.js/npm ou Python) para o core do harness.
6. **Evidência Imutável e Falsificabilidade:** Todo fechamento de tarefa ou gate exige evidências reproduzíveis registradas em Content-Addressed Storage.

---

## 2. Matriz Comparativa de Arquitetura

| Dimensão | **Prumo Harness** | **OpenCode** | **Claude Code** | **DeepSeek Harness (R1/V3)** |
| :--- | :--- | :--- | :--- | :--- |
| **Linguagem & Runtime do Core** | **Go nativo (binário estático)** | TypeScript / Node.js | TypeScript / Node.js (Ink TUI) | Python / C++ (PyTorch/vLLM) |
| **Arquitetura de Processo** | **Daemon + CLI + TUI + GUI (Rust GPUI)** | Client / Daemon desacoplado | CLI monolítico interativo | Scaffolding de inferência / TTC |
| **Independência de Provedor** | **100% Agnóstico** (cloud + local) | Agnóstico (múltiplos modelos) | Acoplado (Anthropic Claude) | Otimizado para DeepSeek R1/V3 |
| **Máquina de Estados do Loop** | **Reentrante, desenrolada por fases com safe-points** | Loop de eventos assíncrono | Loop REPL com tool calls sequenciais | Árvore de busca com MCTS / Best-of-N |
| **Isolamento de Subagentes** | **Worktree Git dedicada, profundidade 1 (ADR 017)** | Subagentes com flags estruturais | Subagentes com teto de profundidade fixo | Rollouts paralelos / branches de busca |
| **Controle de Permissões** | **Engine determinístico (fingerprint por ação/escopo)** | Tool guards com regex e bloqueio pré-tool | Prompts iterativos (Allow/Ask/Deny) | Não aplicável (ambiente de sandbox) |
| **Política de Contexto** | **Lean Progressive Context (LPC/PCA)** | Injeção de contexto por hooks de sessão | `CLAUDE.md` + compactação aos 75% | KV-cache prefix reuse + CoT tagging |
| **Segurança & Confiança** | **Axioma: LLM nunca é raiz de confiança (10 planos)** | Tool guards e sandbox de processos | Read-only vs Bash execution prompts | Recompensa verificável por testes (RLVR) |
| **Governança & Gates** | **Gauntlet de 13 estágios + Evidence Store** | Verificação pós-ferramenta | Validação humana contínua | Oráculo de testes unitários automatizados |

---

## 3. Análise Detalhada dos Concorrentes de Referência

### A. OpenCode (Referência em Extensibilidade e Hooks de Sessão)
* **Pontos Fortes:**
  * Excelente arquitetura cliente/daemon com protocolo IPC desacoplado.
  * Plugins TypeScript dinâmicos que interceptam o ciclo de vida da sessão (`onSessionStart`, `onSessionEnd`) e execução de ferramentas (`validateToolExecution`).
  * Ferramentas primárias e subagentes bem separados (`primary_agent` vs subagentes em pasta dedicada).
* **Limitações frente ao Prumo:**
  * Dependência do ecossistema Node.js/npm para execução de plugins, aumentando atrito de instalação e superfície de supply chain.
  * Falta de verificação formal por schemas (Draft 2020-12) em contratos de dados internos.
  * Ausência de evidências criptograficamente assinadas de proveniência (SLSA) ou matriz formal de multi-tenancy.

### B. Claude Code (Referência em Ergonomia de Terminal e Compacting)
* **Pontos Fortes:**
  * Ergonomia de terminal excepcional: saídas colapsáveis de ferramentas, feedback contínuo sem travar a interface.
  * Monitoramento de custos em tempo real: tokens de entrada, saída, cache e valor financeiro estimado em USD a cada interação.
  * Mecanismo de compactação agressivo: quando a janela atinge ~75% de ocupação, resume micro-históricos automaticamente.
  * Pausa instantânea via teclado (Ctrl+C) preservando o histórico e estado da sessão.
* **Limitações frente ao Prumo:**
  * Sistema proprietário e fechado, rigidamente atrelado à infraestrutura da Anthropic.
  * Não possui separação entre implementador e verificador (`Security Fixer == Verifier` na mesma sessão).
  * Sem suporte a builds reproduzíveis, contratos científicos de convergência ou auditoria de sanitizers de baixo nível.

### C. DeepSeek Harness (Referência em Test-Time Compute e Verification-Driven Search)
* **Pontos Fortes:**
  * Exploração guiada por verificação (TTC): o modelo formula hipóteses, executa o compilador/testes e utiliza a falha imediata como sinal de retroalimentação para autocorreção direcionada.
  * Separação explícita entre tokens de raciocínio (*Chain-of-Thought* / `<think>`) e a emissão final de comandos/patches.
  * Altíssima eficiência computacional através do reaproveitamento intensivo de prefixos em KV-cache.
* **Limitações frente ao Prumo:**
  * Focado em benchmarks de código e treino por reforço (SWE-bench), carecendo de gestão de ciclo de vida de produto (Goals, ADRs, Governança, CI/CD, Multi-tenant SaaS).

---

## 4. Gaps e Oportunidades de Aprimoramento para o Prumo

Sem violar a filosofia de leveza, neutralidade e rigor epistêmico do Prumo, incorporamos os seguintes aprimoramentos arquiteturais:

### Aprimoramento 1: Orçamento e Telemetria em Tempo Real no Loop de Execução
* **Inspiração:** Claude Code.
* **Mecanismo Prumo:** O `Runner` já possui a porta `ReserveBudget` ([`runtime.go:49`](file:///home/raillen/Documentos/Projetos/prumo/internal/harness/runtime/runtime.go#L49)). O aprimoramento conecta o `ReserveBudget` à contabilidade de custos por token e tempo de inferência, emitindo eventos estruturados `budget.tick` com tokens usados, tokens em cache e custo monetário acumulado.
* **Preservação da Filosofia:** Mantém o pre-flight de orçamento (GAP-001/GAP-098) fail-closed antes de chamar a API do modelo.

### Aprimoramento 2: Compactação Reativa de Observações de Ferramentas (Micro-Compacting)
* **Inspiração:** Claude Code.
* **Mecanismo Prumo:** Quando o histórico da sessão atinge 70% do orçamento da janela de contexto do modelo, o compilador de contexto (`internal/harness/contextv2`) substitui outputs extensos de ferramentas antigas por resumos pontuais com seus digests SHA-256 no Evidence Store (ponteiro sobre payload).
* **Preservação da Filosofia:** Fiel ao *Lean Progressive Context*: preserva o Active Goal, as decisões de ADR e o último turno íntegro, evitando degradação de atenção (*lost-in-the-middle*).

### Aprimoramento 3: Verification-Guided Repair Loop (Autocorreção Adversarial Cirúrgica)
* **Inspiração:** DeepSeek Harness (TTC).
* **Mecanismo Prumo:** Na fase `EvaluateStop` ([`runtime.go:852`](file:///home/raillen/Documentos/Projetos/prumo/internal/harness/runtime/runtime.go#L852)), se os testes ou sanitizers falharem, o harness injeta o erro de compilação ou log do sanitizer como uma observação sintética direta. O runner permite até $K$ tentativas de autocorreção direcionada (*bounded retry*) antes de acionar `PhaseFailed`.
* **Preservação da Filosofia:** O oráculo de sucesso continua sendo **estritamente externo e determinístico** (compilador, `go test`, ASan); o modelo nunca decide sozinho se o erro foi corrigido.

### Aprimoramento 4: Pausa Atômica e Safe-Yield no SIGINT (Ctrl+C)
* **Inspiração:** Claude Code & OpenCode.
* **Mecanismo Prumo:** A máquina de estados do runner já opera de forma reentrante com safe-points a cada `Step`. O interceptor de `SIGINT` sinaliza a transição imediata para `PhaseCheckpoint` e `PhaseYield`, gravando o checkpoint persistido e liberando o terminal de forma limpa, permitindo a retomada posterior via `prumo agent resume`.

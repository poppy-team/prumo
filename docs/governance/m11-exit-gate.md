# M11 Connector SDK — Exit Gate

Status: **CONCLUÍDO (Exit Gate do M11 Aprovado)**

## Critérios do exit gate

Exit gate do marco M11 (conforme `docs/development/phases.md:380`):
um novo conector pode ser construído sobre o Connector SDK e passa nos testes de contrato.

---

### 1. Protocolo de Negociação de Capacidades

- Implementado em `internal/connectors/negotiation.go`.
- Avalia as capacidades do contrato contra as capacidades de runtime solicitadas:
  - Supported: execução nativa.
  - Degraded: fallbacks graciosos (p. ex. `pre_tool_block` degrada para a pós-checagem consultiva `advise`; `session_hooks` degrada para checkpointing manual periódico).
  - Unsupported: incompatível no modo strict; avisos são gerados no modo não strict.
- Contrato de schema: `schemas/connector-negotiation.schema.json`.

---

### 2. Engine Padronizado de Manifestos de Limpeza

- Implementado em `internal/connectors/cleanup.go`.
- `SaveCleanup`, `LoadCleanup` e `ExecuteCleanup` unificados.
- Calcula o que foi removido versus o que sobrou, garantindo que modificações do usuário e dados do projeto nunca sejam apagados na desinstalação.

---

### 3. Connector Test Kit (`internal/connectors/testkit`)

- Suíte reutilizável de verificação de contrato:
  - `VerifyContract`: verifica schema, ID, versão, faixa de protocolo, capacidades e enforcement.
  - `VerifyCompilation`: verifica as saídas da compilação e o marcador de propriedade `.prumo-generated.json`.
  - `VerifyIdempotentInstall`: verifica a convergência de instalações repetidas e a criação do manifesto de limpeza.
  - `VerifySafeUninstall`: verifica que os arquivos do usuário são preservados e os artefatos gerenciados são removidos.
  - `VerifyNegotiation`: verifica a negociação de capacidades em condições strict e não strict.
  - `RunAll`: executa as 5 suítes em qualquer conector em uma única chamada.

---

### 4. Implementações e Elevações de Harness

- **Google Gemini CLI (`internal/connectors/gemini`)**: conector completo para o `gemini`, fornecendo `.gemini/config.json`, prompts, subagentes e comandos. Passa em 100% do testkit.
- **Anthropic Claude Code (`internal/connectors/claudecode`)**: conector elevado para o `claude-code`, que gerencia `CLAUDE.md`, subagentes, skills e configurações em `.claude/`. Passa em 100% do testkit.
- **OpenAI Codex CLI (`internal/connectors/codex`)**: conector elevado para o `codex`, que gerencia `AGENTS.md`, subagentes, skills e configuração em `.codex/`. Passa em 100% do testkit.
- **OpenCode Native Harness (`internal/connectors/opencode`)**: harness nativo completo, com plugin TS, tool guards e session hooks. Passa em 100% do testkit.

---

### 5. Integração e Verificação da CLI

- `prumo-agent connector list`: lista todos os conectores registrados, versões, enforcements e capacidades.
- `prumo-agent connector install <name>`: instala qualquer conector registrado e configura os manifestos de limpeza.
- `prumo-agent connector validate <name>`: verifica a integridade dos artefatos do conector e os marcadores de propriedade.
- `prumo-agent connector uninstall <name>`: realiza a remoção limpa.
- `prumo-agent connector negotiate <name> [--strict] [--caps <list>]`: avalia a compatibilidade e as degradações graciosas.
- Todos os testes unitários e de integração passam com detecção de race habilitada.

# M10 OpenCode Native Harness — Exit Gate

Status: **CONCLUÍDO (Exit Gate do M10 Aprovado)**

## Critérios do exit gate

Exit gate do marco M10 (conforme `docs/development/phases.md:365`):
`prumo-agent connector install opencode` produz uma integração nativa totalmente funcional.

---

### 1. OpenCode Native Compiler

- Implementado em `internal/connectors/opencode/opencode.go` e integrado ao `internal/cliops/compile.go`.
- Invocado via:
  - `prumo-agent compile opencode` ou `prumo-agent compile --target opencode`
  - `prumo-agent connector install opencode` (e o `prumo-agent install connector opencode`, mantido por retrocompatibilidade)
- Compila um workspace `.opencode/` completo:
  - `.opencode/opencode.json`: configuração do workspace aderente ao schema do OpenCode.
  - `.opencode/plugins/prumo.ts`: plugin nativo em TypeScript que fornece guards de ferramentas e hooks de ciclo de vida da sessão.
  - `.opencode/agents/prumo.md`: agente orquestrador principal do Prumo.
  - `.opencode/agents/architect.md`, `executor.md`, `verifier.md`: subagentes especializados.
  - `.opencode/skills/`: skills de domínio mapeadas (`code-review`, `goal-management`, `evidence-collection`).
  - `.opencode/commands/prumo.json`: slash commands registrados (`/goal`, `/plan`, `/trace`, `/experience`, `/adopt`, `/status`).
  - `.opencode/guards/tool-policy.json`: políticas de segurança dos guards de ferramentas.
  - `.opencode/.prumo-generated.json`: marcador de propriedade legível por máquina, com versão e flags de gerenciamento.

---

### 2. Tool Guards (Validação Pré-Ferramenta)

- Invariante: as execuções de ferramentas passam por um veto síncrono (`pre_tool_block`).
- Proíbe comandos de shell destrutivos (`rm -rf /`, `mkfs`, fork bombs).
- Protege caminhos sensíveis (`.git/`, `.prumo/credentials`, `.env`) contra escritas destrutivas.
- Sinaliza mutações de alto risco (`git push --force`, `git reset --hard`) que exigem confirmação explícita.

---

### 3. Hooks de Ciclo de Vida da Sessão

- `session.start`: injeta o Lean Progressive Context (LPC) e os detalhes do Goal ativo sem inflar com transcript conversacional.
- `session.end`: emite eventos estruturados de conclusão de sessão em `.prumo/experience/`.
- `tool.before_execute` / `tool.after_execute`: intercepta e registra a telemetria das ferramentas de forma determinística.

---

### 4. Connector Contract e Integridade da Limpeza

- Está em conformidade com `schemas/connector-contract.schema.json`.
- Aplica enforcement strict e suporta as capacidades padrão (`advise`, `restrict_tools`, `pre_tool_block`, `post_tool_verify`, `isolate_subagents`, `session_hooks`, `native_plugin`, `commands`, `subagents`).
- Gera o `CleanupManifest` em `PRUMO_HOME/connectors/opencode/cleanup.json`.
- Suporta limpeza atômica e segura via `prumo-agent connector uninstall opencode`, deixando os arquivos do usuário intactos.

---

### 5. Evidência de Verificação

- Testes unitários e de integração em `internal/connectors/opencode/opencode_test.go`:
  - `TestOpenCodeContract`: verifica o conjunto de capacidades, o modo de enforcement e os hooks de ciclo de vida.
  - `TestOpenCodeCompileAndValidate`: verifica a estrutura de diretórios, o código do plugin e o marcador de propriedade.
  - `TestOpenCodeInstallAndUninstall`: verifica o registro do manifesto, a criação do manifesto de limpeza e a desinstalação segura.
- Testes da CLI em `cmd/prumo/connector_commands_test.go`:
  - Valida `prumo-agent connector list`, `prumo-agent connector install`, `prumo-agent connector validate`, `prumo-agent connector uninstall` e `prumo-agent install connector`.
- Todos os testes passam com `-race` habilitado.

# Manual de Uso

## Fluxo recomendado

```text
profile → init → validate → doctor → Goal → context → compile → evidence → review
```

O Prumo organiza estado canônico no repositório. O harness executa ações; o Prumo mantém protocolo, políticas, Goals, evidências e adapters.

## 1. Criar um projeto

```bash
prumo-agent init ./my-project \
  --profile examples/brasa/project-profile.json \
  --non-interactive
```

O profile deve declarar `ai.preferred_models`. O resultado contém `prumo.json`, `.ai/`, `docs/PRUMO.md`, `PROJECT_STATE.md` e histórico derivado.

## 2. Validar e diagnosticar

```bash
prumo-agent validate ./my-project
prumo-agent doctor ./my-project
prumo-agent --json doctor ./my-project
prumo-agent framework-check
```

`validate` verifica estrutura e schemas. `doctor` verifica também versionamento, locks, dependencies, DAGs, workforce, policies, gates e evidence.

## 3. Criar e bloquear Goals

```bash
prumo-agent goal new P00-G01 "Foundation" \
  --phase P00 \
  --objective "Establish a tested project foundation." \
  --path ./my-project

prumo-agent goal list --path ./my-project
prumo-agent goal state P00-G01 PLANNED --path ./my-project
prumo-agent goal state P00-G01 LOCKED --path ./my-project
```

Estados válidos:

```text
DRAFT → PLANNED → LOCKED → EXECUTING → VERIFYING → REVIEWING → DONE
```

Estados podem ir para `BLOCKED` conforme as transições do protocolo. `DONE` exige evidence. Goal bloqueado deve ser alterado com amendment:

```bash
prumo-agent goal amend P00-G01 \
  --file amendment.json \
  --path ./my-project
```

## 4. Planejar contexto

```bash
prumo-agent context plan "debug authentication regression" \
  --path ./my-project \
  --json
```

O planner escolhe uma estratégia e budget conforme o risco sem carregar o repositório inteiro.

## 5. Resolver workforce

```bash
prumo-agent resolve examples/brasa/project-profile.json --json
prumo-agent explain workforce examples/brasa/project-profile.json --json
prumo-agent explain agent architect --json
prumo-agent explain skill clean-code --json
prumo-agent explain recipe web-feature --json
```

A resolução é determinística para os mesmos profile, catálogo e recursos.

## 6. Compilar adapters

```bash
prumo-agent compile --target generic --path ./my-project
prumo-agent compile --target codex --path ./my-project
prumo-agent compile --target claude-code --path ./my-project
prumo-agent compile --target traycer --path ./my-project
```

Targets disponíveis:

```text
generic, chatgpt, claude, kimi, codex, claude-code, traycer
```

Saídas são derivadas. Edite o catálogo/workforce canônico, não o adapter gerado.

## 7. Reports e inteligência

```bash
prumo-agent report add conformance/fixtures/task-report.json --path ./my-project
prumo-agent report summary --path ./my-project --json
```

Reports alimentam `.prumo/history/project-intelligence.json`, que é estado derivado e reconstruível.

## 8. Snapshot e migração

```bash
prumo-agent snapshot ./my-project --output ./my-project-backup.zip
prumo-agent migrate ./my-project --dry-run --json
prumo-agent migrate ./my-project
```

Sempre execute `--dry-run` antes de migrações. A migração cria snapshot prévio quando altera o projeto.

## 9. Saída JSON para automações

```bash
prumo-agent --json version
prumo-agent --json validate ./my-project
prumo-agent --json framework-check
prumo-agent --json compile --target generic --path ./my-project
```

Contrato comum:

```json
{
  "protocol_version": "1",
  "ok": true,
  "data": {},
  "diagnostics": [],
  "warnings": []
}
```

Códigos principais:

| Código | Uso |
|--------|-----|
| `0` | sucesso |
| `1` | validação ou gate falhou |
| `2` | uso/argumentos inválidos |
| `3` | configuração inválida |
| `4` | capability indisponível |
| `5` | erro interno |
| `6` | projeto não encontrado |

## 10. Instalação e estado global

```bash
prumo-agent --home ./prumo-home setup
prumo-agent --home ./prumo-home install connector opencode
prumo-agent --home ./prumo-home uninstall --connectors --purge-cache
```

Use `--home` em CI, testes, devboxes e cenários que não devem tocar `~/.prumo`.

## 11. Aposentadoria do Python (ADR 002)

O runtime e a suíte de testes em Python v0.3 (legacy) foram aposentados e removidos (ADR 002). O Prumo v0.6 é 100% Go nativo e autocontido. Conformance e validação são executadas diretamente pela suíte de testes em Go.

## Command reference

### Core

```text
prumo-agent version
prumo-agent status --path <path>
prumo-agent setup
prumo-agent install connector <id>
prumo-agent uninstall [--connectors] [--purge-cache] [--purge-global-config]
prumo-agent init <path> --profile <profile> --non-interactive
prumo-agent validate [path]
prumo-agent doctor [path] [--json]
prumo-agent framework-check
```

### Protocol

```text
prumo-agent goal new <id> <title> --phase <phase> [--objective <text>] [--path <path>]
prumo-agent goal state <id> <state> [--reason <text>] [--path <path>]
prumo-agent goal amend <id> [--file <path>] [--reason <text>] [--approved-by <actor>] [--path <path>]
prumo-agent goal list [--path <path>]
prumo-agent context plan <task> [--path <path>] [--json]
prumo-agent report add <file> [--path <path>]
prumo-agent report summary [--path <path>]
prumo-agent migrate [path] [--dry-run] [--json]
prumo-agent snapshot [path] [--output <path>]
prumo-agent docs delta propose --goal <goal> [--path <project>] [changed ...] [--json]
prumo-agent docs delta list [--path <project>] [--json]
prumo-agent docs delta show --id <delta> [--path <project>] [--json]
prumo-agent docs delta transition --id <delta> --state <state> [--evidence <id>]... [--path <project>] [--json]
```

### Resolution, explanation, and compiler

```text
prumo-agent resolve <profile> [--json]
prumo-agent explain workforce <profile> [--json]
prumo-agent explain agent <id> [--json]
prumo-agent explain skill <id> [--json]
prumo-agent explain recipe <id> [--json]
prumo-agent explain context <task-id> [--path <path>] [--json]
prumo-agent explain model <role> [--path <path>] [--json]
prumo-agent explain execution <profile> [--path <path>] [--json]
prumo-agent compile --target <target> [--path <path>] [--json]
```

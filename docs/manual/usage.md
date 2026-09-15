# Manual de Uso

## Fluxo recomendado

```text
profile → init → validate → doctor → Goal → context → compile → evidence → review
```

O Prumo organiza estado canônico no repositório. O harness executa ações; o Prumo mantém protocolo, políticas, Goals, evidências e adapters.

## 1. Criar um projeto

```bash
prumo init ./my-project \
  --profile examples/brasa/project-profile.json \
  --non-interactive
```

O profile deve declarar `ai.preferred_models`. O resultado contém `prumo.json`, `.ai/`, `docs/PRUMO.md`, `PROJECT_STATE.md` e histórico derivado.

## 2. Validar e diagnosticar

```bash
prumo validate ./my-project
prumo doctor ./my-project
prumo --json doctor ./my-project
prumo framework-check
```

`validate` verifica estrutura e schemas. `doctor` verifica também versionamento, locks, dependencies, DAGs, workforce, policies, gates e evidence.

## 3. Criar e bloquear Goals

```bash
prumo goal new P00-G01 "Foundation" \
  --phase P00 \
  --objective "Establish a tested project foundation." \
  --path ./my-project

prumo goal list --path ./my-project
prumo goal state P00-G01 PLANNED --path ./my-project
prumo goal state P00-G01 LOCKED --path ./my-project
```

Estados válidos:

```text
DRAFT → PLANNED → LOCKED → EXECUTING → VERIFYING → REVIEWING → DONE
```

Estados podem ir para `BLOCKED` conforme as transições do protocolo. `DONE` exige evidence. Goal bloqueado deve ser alterado com amendment:

```bash
prumo goal amend P00-G01 \
  --file amendment.json \
  --path ./my-project
```

## 4. Planejar contexto

```bash
prumo context plan "debug authentication regression" \
  --path ./my-project \
  --json
```

O planner escolhe uma estratégia e budget conforme o risco sem carregar o repositório inteiro.

## 5. Resolver workforce

```bash
prumo resolve examples/brasa/project-profile.json --json
prumo explain workforce examples/brasa/project-profile.json --json
prumo explain agent architect --json
prumo explain skill clean-code --json
prumo explain recipe web-feature --json
```

A resolução é determinística para os mesmos profile, catálogo e recursos.

## 6. Compilar adapters

```bash
prumo compile --target generic --path ./my-project
prumo compile --target codex --path ./my-project
prumo compile --target claude-code --path ./my-project
prumo compile --target traycer --path ./my-project
```

Targets disponíveis:

```text
generic, chatgpt, claude, kimi, codex, claude-code, traycer
```

Saídas são derivadas. Edite o catálogo/workforce canônico, não o adapter gerado.

## 7. Reports e inteligência

```bash
prumo report add conformance/fixtures/task-report.json --path ./my-project
prumo report summary --path ./my-project --json
```

Reports alimentam `.prumo/history/project-intelligence.json`, que é estado derivado e reconstruível.

## 8. Snapshot e migração

```bash
prumo snapshot ./my-project --output ./my-project-backup.zip
prumo migrate ./my-project --dry-run --json
prumo migrate ./my-project
```

Sempre execute `--dry-run` antes de migrações. A migração cria snapshot prévio quando altera o projeto.

## 9. Saída JSON para automações

```bash
prumo --json version
prumo --json validate ./my-project
prumo --json framework-check
prumo --json compile --target generic --path ./my-project
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
prumo --home ./prumo-home setup
prumo --home ./prumo-home install connector opencode
prumo --home ./prumo-home uninstall --connectors --purge-cache
```

Use `--home` em CI, testes, devboxes e cenários que não devem tocar `~/.prumo`.

## 11. Aposentadoria do Python (ADR 002)

O runtime e a suíte de testes em Python v0.3 (legacy) foram aposentados e removidos (ADR 002). O Prumo v0.6 é 100% Go nativo e autocontido. Conformance e validação são executadas diretamente pela suíte de testes em Go.

## Command reference

### Core

```text
prumo version
prumo status --path <path>
prumo setup
prumo install connector <id>
prumo uninstall [--connectors] [--purge-cache] [--purge-global-config]
prumo init <path> --profile <profile> --non-interactive
prumo validate [path]
prumo doctor [path] [--json]
prumo framework-check
```

### Protocol

```text
prumo goal new <id> <title> --phase <phase> [--objective <text>] [--path <path>]
prumo goal state <id> <state> [--reason <text>] [--path <path>]
prumo goal amend <id> [--file <path>] [--reason <text>] [--approved-by <actor>] [--path <path>]
prumo goal list [--path <path>]
prumo context plan <task> [--path <path>] [--json]
prumo report add <file> [--path <path>]
prumo report summary [--path <path>]
prumo migrate [path] [--dry-run] [--json]
prumo snapshot [path] [--output <path>]
prumo docs delta propose --goal <goal> [--path <project>] [changed ...] [--json]
prumo docs delta list [--path <project>] [--json]
prumo docs delta show --id <delta> [--path <project>] [--json]
prumo docs delta transition --id <delta> --state <state> [--evidence <id>]... [--path <project>] [--json]
```

### Resolution, explanation, and compiler

```text
prumo resolve <profile> [--json]
prumo explain workforce <profile> [--json]
prumo explain agent <id> [--json]
prumo explain skill <id> [--json]
prumo explain recipe <id> [--json]
prumo explain context <task-id> [--path <path>] [--json]
prumo explain model <role> [--path <path>] [--json]
prumo explain execution <profile> [--path <path>] [--json]
prumo compile --target <target> [--path <path>] [--json]
```

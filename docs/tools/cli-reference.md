# Referência Completa do CLI (`prumo`)

O executável `prumo` aceita comandos e flags estruturadas. Para automação em pipelines de CI ou integração com agentes, a flag `--json` garante saída padronizada em envelope.

## Tabela de Comandos Principais

| Comando | Descrição | Exemplo de Uso |
|---|---|---|
| `prumo version` | Exibe a versão do CLI, commit SHA e build time. | `prumo version --json` |
| `prumo init <caminho>` | Inicializa um repositório Prumo com perfil canônico. | `prumo init ./meu-projeto --profile ./profile.json` |
| `prumo goal new <id> <título>` | Cria uma nova Meta (Goal) no projeto. | `prumo goal new P01-G01 "Fundação"` |
| `prumo goal state <id> <estado>` | Altera o estado de uma Meta (`PLANNED`, `LOCKED`, etc.). | `prumo goal state P01-G01 LOCKED` |
| `prumo goal amend <id>` | Abre uma emenda formal para alterar uma Meta travada. | `prumo goal amend P01-G01 --reason "Novo escopo"` |
| `prumo goal list` | Lista todas as Metas do repositório com status e fase. | `prumo goal list` |
| `prumo plan new` | Cria um novo Plano de trabalho com DAG de tarefas. | `prumo plan new --goal P01-G01` |
| `prumo run` | Executa tarefas ou diretivas através do Prumo Harness. | `prumo run --path .` |
| `prumo compile` | Compila e sincroniza adaptadores para harnesses. | `prumo compile --target claude-code` |
| `prumo doctor` | Executa bateria de diagnósticos de integridade do projeto. | `prumo doctor` |
| `prumo validate` | Valida todos os arquivos canônicos contra JSON Schemas. | `prumo validate` |
| `prumo adopt` | Executa motor de adoção em repositórios legados. | `prumo adopt --path ./legado --scan` |
| `prumo agent` | Dispara o cliente interativo prumo-agent (ou `pa`). | `prumo agent` |
| `prumo docs audit` | Audita conformidade e integridade da documentação. | `prumo docs audit` |
| `prumo docs authority` | Valida hierarquia de autoridade e links de roteamento. | `prumo docs authority` |

## Envelope de Saída JSON (`--json`)

Todas as invocações de comando com a flag `--json` produzem um JSON determinístico na saída padrão (`stdout`), enquanto avisos e logs de depuração são emitidos em `stderr`:

```json
{
  "protocol_version": "1",
  "ok": true,
  "data": {
    "goal_id": "P01-G01",
    "status": "LOCKED",
    "lock_digest": "sha256:7f83b1657ff1fc53b92dc18148a1d65dfc2d4b1fa3d677284addd200126d9069"
  },
  "diagnostics": [],
  "warnings": []
}
```

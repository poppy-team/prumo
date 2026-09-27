# Conectores & Adapters de Harness

O Prumo adota uma arquitetura de conectores e adaptadores agnóstica de provedor. Nenhum ambiente de execução ou provedor de LLM possui autoridade direta sobre a verdade canônica do projeto.

## O que são Conectores?

Os **Conectores** são pontes bidirecionais entre o plano de controle canônico do Prumo e os ambientes onde os agentes operam no dia a dia. Ao usar o comando `prumo compile`, o Prumo gera ou atualiza cirurgicamente as configurações exigidas por cada ferramenta, sem sobrescrever instruções customizadas mantidas por desenvolvedores humanos.

## Plataformas Suportadas

| Plataforma / Ferramenta | Identificador `--target` | Arquivo / Diretório Gerado | Estratégia de Atualização |
|---|---|---|---|
| **Google Antigravity** | `antigravity` | `.gemini/` e regras de contexto | Injeção cirúrgica de contexto |
| **Claude Code** | `claude-code` | `CLAUDE.md` | Regiões gerenciadas `doccompile` |
| **OpenAI Codex** | `codex` | `CODEX.md` / `.codex/` | Regiões gerenciadas |
| **Cursor** | `cursor` | `.cursorrules` | Injeção de regras de arquitetura |
| **Windsurf** | `windsurf` | `.windsurfrules` | Injeção de regras de arquitetura |
| **OpenCode** | `opencode` | `.opencode/` | Manifesto nativo e skills |
| **Cline / Roo Code** | `cline` | `.clinerules` | Regras de orquestração e contexto |
| **Generic (Padrão)** | `generic` | `AGENTS.md` | Padrão canônico neutro |

## Exemplo de Compilação

Para compilar e sincronizar os adaptadores para seu ambiente preferido:

```bash
# Sincronizar regras para Claude Code
prumo compile --target claude-code

# Sincronizar regras para Google Antigravity
prumo compile --target antigravity

# Sincronizar para múltiplos ambientes
prumo compile --target generic --target cursor
```

## Regiões Gerenciadas (`doccompile`)

Quando o Prumo atualiza um arquivo de instruções (como `CLAUDE.md` ou `AGENTS.md`), ele utiliza marcadores delimitadores:

```markdown
<!-- prumo:begin agents-core -->
<!-- prumo:generated adapter=agents-md scope=root fingerprint=... sources=... -->
- Instruções geradas automaticamente pelo compilador Prumo...
<!-- prumo:end agents-core -->
```

Qualquer texto fora desses marcadores pertence aos desenvolvedores do projeto e é preservado integralmente durante compilações, instalações ou desinstalações.

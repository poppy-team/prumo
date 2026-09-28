# Conectores & Adapters de Harness

O Prumo adota uma arquitetura agnóstica de provedor. Nenhum ambiente de execução ou fornecedor de LLM possui autoridade direta sobre a verdade canônica do projeto.

---

## Conectores Globais vs. Adaptadores de Projeto

É fundamental distinguir os dois níveis de integração do Prumo:

1. **Conector Global (`prumo connector ...`)**: 
   Opera no seu computador (`~/.prumo/connectors/`). Gerencia a instalação, status de disponibilidade da ferramenta no `$PATH` e capacidades negociadas com o executável daquele agente (ex: Claude Code, OpenCode CLI, Antigravity).
2. **Adaptador de Projeto (`prumo compile --target ...`)**: 
   Opera no repositório ativo (`./`). Gera ou injeta cirurgicamente regras de contexto, skills e instruções que aquele agente lerá ao ser aberto nesta pasta (ex: `CLAUDE.md`, `.gemini/`, `.opencode/`, `.cursorrules`).

---

## Gerenciamento de Conectores (`prumo connector`)

### Listar Conectores Disponíveis e Registrados
```bash
prumo connector list
```

### Inspecionar o Status e Saúde de um Conector
```bash
prumo connector status claude
prumo connector status opencode
prumo connector status antigravity
```
O comando informa:
- Versão e modelo de imposição (strict vs standard).
- Se está instalado em `~/.prumo/connectors/`.
- Se o binário da ferramenta foi detectado no seu `$PATH`.
- Lista completa de capacidades suportadas (advise, restrict_tools, session_hooks, etc.).

### Instalar um Conector Globalmente
```bash
prumo connector install opencode
prumo connector install claude-code
```

---

## Compilação de Adaptadores de Projeto (`prumo compile`)

Para sincronizar as regras de contexto do seu projeto ativo com seus agentes preferidos:

```bash
# Compilar adaptadores para todos os conectores suportados
prumo compile --all

# Sincronizar regras específicas para o Claude Code (gera CLAUDE.md)
prumo compile --target claude-code

# Sincronizar regras para o Google Antigravity (gera .gemini/)
prumo compile --target antigravity

# Sincronizar para o Cursor (gera .cursorrules)
prumo compile --target cursor
```

### Regiões Gerenciadas (`doccompile`)

Quando o Prumo atualiza um arquivo de instruções (como `CLAUDE.md` ou `AGENTS.md`), ele utiliza marcadores delimitadores:

```markdown
<!-- prumo:begin agents-core -->
<!-- prumo:generated adapter=agents-md scope=root fingerprint=... sources=... -->
- Instruções geradas automaticamente pelo compilador Prumo...
<!-- prumo:end agents-core -->
```

Qualquer texto fora desses marcadores pertence aos desenvolvedores do projeto e é preservado integralmente durante compilações, instalações ou desinstalações.

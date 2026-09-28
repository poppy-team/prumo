# Manual de Uso da CLI

## Fluxo Recomendado de Engenharia

```text
init (ou adopt) → validate → doctor → Goal → context → compile → evidence → review
```

O Prumo mantém a verdade canônica no repositório Git. O harness executa ações deterministicamente; o Prumo impõe o protocolo, políticas, Goals, evidências e adapters.

---

## 1. Inicializar um Projeto Novo (`prumo init`)

O Prumo suporta inicialização zero-configuração com detecção automática de stack:

```bash
# Zero-configuração (detecta linguagem e cria workspace Protocolo v3)
prumo init

# Com presets específicos
prumo init --preset web --name portal-web
prumo init --preset service --stack go
prumo init --preset cli --name minha-ferramenta

# Visualizar o perfil inferido antes de aplicar
prumo init --print-profile
```

---

## 2. Adotar um Repositório Existente (`prumo adopt`)

Para transformar qualquer base de código existente em um projeto governado pelo Prumo:

```bash
# Auditar e gerar propostas não-destrutivas
prumo adopt

# Aplicar a migração e configurar prumo.json Protocolo v3
prumo adopt --apply

# Subcomandos específicos de inspeção
prumo adopt scan .       # Indexar arquivos
prumo adopt facts .      # Extrair fatos técnicos
prumo adopt classify .   # Classificação arquitetural
prumo adopt scaffold .   # Dry-run das alterações propostas
```

---

## 3. Validar e Diagnosticar

```bash
# Validação estrutural de esquemas e contratos do Protocolo v3
prumo validate

# Diagnóstico completo de saúde do projeto, locks e conectores
prumo doctor

# Saída em envelope JSON estruturado para pipelines
prumo --json doctor
```

`validate` verifica schemas JSON e integridade dos arquivos obrigatórios. `doctor` audita integridade de Goals, versionamento, DAGs, integridade da workforce e ferramentas de IA no ambiente.

---

## 4. Criar e Bloquear Metas (Goals)

```bash
# Criar uma nova meta
prumo goal new P00-G01 "Foundation" \
  --phase P00 \
  --objective "Establish a tested project foundation."

# Listar metas do projeto
prumo goal list

# Transicionar estado da meta
prumo goal state P00-G01 PLANNED
prumo goal state P00-G01 LOCKED
```

Ciclo de vida dos estados:

```text
DRAFT → PLANNED → LOCKED → EXECUTING → VERIFYING → REVIEWING → DONE
```

Metas bloqueadas (`LOCKED`) possuem digest criptográfico SHA-256 e exigem emenda formal (`goal amend`) para qualquer alteração de escopo:

```bash
prumo goal amend P00-G01 --file amendment.json
```

---

## 5. Gerenciamento de Conectores (`prumo connector`)

```bash
# Listar todos os conectores suportados
prumo connector list

# Inspecionar status, ferramentas no PATH e capacidades
prumo connector status claude
prumo connector status opencode
prumo connector status antigravity

# Instalar conector no ambiente global (~/.prumo/connectors/)
prumo connector install opencode
```

---

## 6. Compilar Adaptadores para Agentes de IA (`prumo compile`)

```bash
# Compilar todos os adaptadores configurados
prumo compile --all

# Compilar para agentes específicos
prumo compile --target claude-code
prumo compile --target antigravity
prumo compile --target cursor
prumo compile --target opencode
```

Os arquivos gerados (`CLAUDE.md`, `.gemini/`, `.cursorrules`) utilizam marcadores delimitadores `doccompile` e preservam qualquer instrução manual dos desenvolvedores.

---

## 7. Prumo Code Agent (TUI Interativo) e Prumo IDE

O Prumo disponibiliza duas interfaces avançadas para desenvolvimento: o **Prumo Code Agent** (interface terminal rica) e o **Prumo IDE** (interface gráfica desktop).

### Invocando as Interfaces

```bash
# Iniciar o Prumo Code Agent (TUI interativo)
prumo code-agent

# Aliases equivalentes
prumo agent
prumo tui

# Iniciar o Prumo IDE (GUI Desktop nativa)
prumo native

# Executar uma meta específica de forma autônoma (Headless Harness)
prumo run
```

### Comandos de Teclado (Keybindings) no Prumo Code Agent

| Tecla / Atalho | Ação |
|---|---|
| `Ctrl+T` | Abre uma nova aba (tab) de conversação no projeto atual |
| `Ctrl+W` | Fecha a aba de conversação atual |
| `Alt+1` a `Alt+9` | Alterna instantaneamente entre as abas abertas |
| `Ctrl+B` | Alterna a visibilidade da barra lateral (salvo de forma persistente) |
| `Ctrl+K` | Abre o menu de comandos (Command Palette) |
| `Ctrl+L` | Alterna para a visualização de logs internos |
| `Ctrl+P` | Seletor rápido de arquivos para anexar ao contexto |
| `Ctrl+Q` | Abre o diálogo de saída com opção de renomear e salvar a sessão |
| `Esc` | Fecha diálogos modais e cancela pré-visualizações sem persistir |

### Comandos Slash na Linha de Conversa

Dentro da conversa do Prumo Code Agent, utilize comandos iniciados por barra (`/`):

- `/agents` ou `/workforce`: Lista todos os agentes especialistas da força de trabalho ativa.
- `/agent <nome>`: Exibe a especificação, skills e restrições de um agente específico.
- `/subagents` ou `/subagent`: Exibe a árvore e estado dos subagentes delegados em execução.
- `/tab` ou `/tabs`: Lista e gerencia as abas de conversação ativas.
- `/gates`: Exibe os gates de verificação (testes, linters, segurança) e seu status.
- `/audit` ou `/telemetry`: Exibe o resumo da telemetria e custos acumulados do projeto.
- `/theme`: Abre o seletor de temas com **pré-visualização em tempo real**. Ao navegar com `↑`/`↓`, a interface é repintada imediatamente com a nova paleta. Pressione `Enter` para persistir ou `Esc` para restaurar o tema anterior.
- `/clear`: Limpa o histórico visual da conversa na aba ativa.
- `/help`: Exibe o guia completo de atalhos e comandos.
- `/quit`: Inicia o encerramento com diálogo de salvamento.

---

### Sistema de Auditoria Determinística por Projeto e Sessão

Todas as informações coletadas pelo harness durante a execução do Prumo Code Agent são persistidas deterministicamente em disco ao finalizar cada sessão. Isso permite auditar consumo de tokens, custos financeiros, arquivos impactados e decisões tomadas.

#### Localização dos Arquivos de Auditoria
- **Auditoria do Projeto**: `.prumo/runtime/audit/telemetry.json` (agregado de todas as sessões e tarefas).
- **Auditoria por Sessão**: `.prumo/runtime/audit/sessions/<session-id>.json` (detalhes isolados de cada conversa).

#### Estrutura do JSON de Auditoria

O arquivo gerado segue uma ordenação determinística de chaves para compatibilidade total com Git diffs e inspeções automatizadas:

```json
{
  "schema_version": "1.0.0",
  "project_name": "meu-projeto",
  "project_dir": "/caminho/para/meu-projeto",
  "git_branch": "feature/auth-jwt",
  "updated_at": "2026-09-28T09:30:00Z",
  "total_sessions": 3,
  "telemetry": {
    "total_tokens": 154200,
    "prompt_tokens": 112000,
    "completion_tokens": 32200,
    "cache_read_tokens": 10000,
    "cache_write_tokens": 5000,
    "cost_usd": 0.3845,
    "active_model": "claude-3-7-sonnet",
    "active_provider": "anthropic",
    "session_count": 3
  },
  "workforce": {
    "agents_count": 39,
    "skills_count": 189,
    "recipes_count": 20,
    "active_subagents": [
      {
        "id": "subagent-1",
        "agent_name": "architect",
        "role": "Interface & Schema Specialist",
        "status": "completed",
        "tokens": 42000,
        "cost_usd": 0.105
      }
    ]
  },
  "sessions": [
    {
      "id": "sess-20260928-091522",
      "title": "feature/auth-jwt",
      "created_at": "2026-09-28T09:15:22Z",
      "ended_at": "2026-09-28T09:30:00Z",
      "status": "completed",
      "git_branch": "feature/auth-jwt",
      "active_model": "claude-3-7-sonnet",
      "active_provider": "anthropic",
      "telemetry": {
        "total_tokens": 54200,
        "prompt_tokens": 42000,
        "completion_tokens": 12200,
        "cache_read_tokens": 5000,
        "cache_write_tokens": 2000,
        "cost_usd": 0.135
      },
      "tasks": [
        {
          "id": "task-1",
          "title": "Implementar token validator",
          "status": "done",
          "tokens": 24000,
          "cost_usd": 0.06
        }
      ],
      "changed_files": [
        "internal/auth/jwt.go",
        "internal/auth/jwt_test.go"
      ],
      "verification_gates": [
        {
          "name": "lint",
          "status": "passed",
          "details": "golangci-lint: 0 errors"
        },
        {
          "name": "tests",
          "status": "passed",
          "details": "100% passing (12 tests)"
        }
      ]
    }
  ]
}
```

Cada gravação de sessão recalcula e consolida os totais do projeto de maneira atômica e determinística.

---

## 8. Relatórios de Evidência e Inteligência

```bash
# Registrar evidência de execução de tarefa
prumo report add task-report.json

# Consultar métricas e histórico de inteligência do projeto
prumo report summary
prumo --json report summary
```

---

## 9. Contrato JSON Unificado para Agentes de Código

Todos os comandos suportam a flag global `--json`:

```bash
prumo --json version
prumo --json validate
prumo --json doctor
prumo --json status
```

Envelope padronizado retornado na saída padrão:

```json
{
  "protocol_version": "1",
  "ok": true,
  "data": {},
  "diagnostics": [],
  "warnings": []
}
```

Códigos de saída da CLI:

| Código | Significado |
|---|---|
| `0` | Sucesso / Operação concluída |
| `1` | Validação de schema, gate ou evidência falhou |
| `2` | Erro de uso ou argumentos inválidos |
| `3` | Erro de configuração |
| `4` | Capability ou serviço indisponível |
| `5` | Erro interno |
| `6` | Projeto Prumo não encontrado |

---

## 10. Atualização do Prumo (`prumo upgrade`)

```bash
# Verificar novas versões
prumo upgrade --check

# Atualizar binário global
prumo upgrade
```

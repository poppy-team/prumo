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

## 7. Agente de Código Interativo (`prumo agent`)

```bash
# Iniciar a interface terminal interativa (TUI)
prumo agent

# Executar uma meta específica de forma autônoma
prumo run
```

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

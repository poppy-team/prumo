# Conceitos Fundamentais e Modelo Mental

Para utilizar o Prumo com máxima eficiência, humanos e agentes de código devem compreender seu **modelo mental em dois planos**: o **Plano do Host (Ambiente Global)** e o **Plano do Repositório (Workspace Local)**.

---

## O Modelo em Dois Planos

```text
┌─────────────────────────────────────────────────────────────┐
│                 PLANO DO HOST (Ambiente Global)              │
│  Diretório: ~/.prumo                                        │
│                                                             │
│  • Executável prumo (/usr/local/bin/prumo)                  │
│  • Daemon de execução e background services (prumo serve)   │
│  • Conectores globais instalados (~/.prumo/connectors/)     │
│  • Subsistema de auto-update (prumo upgrade)                │
│  • Auditoria global do sistema (prumo doctor)               │
└──────────────────────────────┬──────────────────────────────┘
                               │
            opera sobre e sincroniza regras com
                               │
                               ▼
┌─────────────────────────────────────────────────────────────┐
│               PLANO DO REPOSITÓRIO (Workspace Local)        │
│  Diretório do projeto (./)                                  │
│                                                             │
│  • Configuração Canônica: prumo.json (Protocolo v3)         │
│  • Documentação Canônica: docs/PRUMO.md, PROJECT_STATE.md   │
│  • Força de Trabalho Ativa: .ai/ (agentes, skills, receitas)│
│  • Adaptadores Compilados: AGENTS.md, CLAUDE.md, .opencode  │
│  • Metas e Planos com Trava Criptográfica SHA-256           │
└─────────────────────────────────────────────────────────────┘
```

---

## As 3 Superfícies do Ecossistema Prumo

O ecossistema Prumo organiza suas interfaces de forma estrita e especializada:

1. **Prumo** (CLI & Daemon):
   - **Binário/Comando**: `prumo`.
   - **Papel**: Motor central de governança, compilação de adaptadores (`prumo compile`), validação de esquemas (`prumo validate`), diagnósticos (`prumo doctor`), travamento de metas (`prumo goal`) e execução em sandbox (`prumo run`).
2. **Prumo Code Agent** (TUI - Terminal User Interface):
   - **Comando de Ativação**: `prumo code-agent` (ou aliases `prumo agent`, `prumo tui`).
   - **Papel**: Interface rica de terminal para desenvolvimento assistido por IA em par com o desenvolvedor. Suporta sessões multi-tab, barra lateral informativa estilo OpenCode v2 com telemetria profunda do harness (provider/modelo sempre visíveis, branch ativa, tokens, custos USD, arquivos alterados, árvore de subagentes e gates de qualidade), preview de temas em tempo real e salvamento persistente determinístico.
3. **Prumo IDE** (GUI Desktop Nativa):
   - **Comando de Ativação**: `prumo native` (ou binário `prumo-viewer`).
   - **Papel**: Ambiente gráfico desktop de altíssimo desempenho para navegação visual em repositórios massivos, inspeção interativa de grafos de tarefas (DAGs) e painéis executivos de evidência.

---

## Os 11 Princípios Canônicos do Prumo

1. **Repositório sobre Memória de Conversa**: A verdade canônica do projeto vive exclusivamente em arquivos no Git, não na janela efêmera de chat de uma LLM.
2. **Protocolo sobre Harness**: Claude Code, OpenCode, Codex, Gemini e Cursor são clientes intercambiáveis; o Prumo fornece o protocolo neutro comum.
3. **Canônico antes de Gerado**: Markdown, JSON e JSON Schema são fontes da verdade mantidas por humanos e agentes autorizados; arquivos de adapters (`AGENTS.md`, `.cursorrules`), caches e índices são derivados e compilados.
4. **Metas (Goals) como Unidade de Entrega**: Metas representam entregas mensuráveis de engenharia. Após entrarem no estado `LOCKED`, seus critérios de aceitação tornam-se imutáveis via digest criptográfico.
5. **Lean Progressive Context (LPC)**: Os agentes de IA recebem apenas o menor contexto suficiente para a tarefa em execução. A expansão de contexto ocorre progressivamente e apenas mediante evidência comprovada.
6. **Evidência sobre Afirmação**: Uma tarefa só é declarada concluída se apresentar evidências verificáveis (testes executados, linters verdes, builds aprovados, revisões concluídas).
7. **Determinístico antes de Probabilístico**: Invariantes arquiteturais, checagens de sintaxe e gates de qualidade são avaliados por regras determinísticas em código Go, nunca pela intuição probabilística da LLM.
8. **Incompatibilidade Estrita contra Deriva (Anti-Drift)**: Modificações que quebrem contratos arquiteturais ou esquemas JSON falham imediatamente no `prumo validate`.
9. **Desacoplamento e Independência de Provedor**: O núcleo do framework não impõe nenhum modelo LLM ou provedor proprietário como obrigatório.
10. **Adoção Não-Destrutiva**: Projetos existentes mantêm seus diretórios e arquivos de código intactos durante todo o ciclo de vida.
11. **Auditoria Determinística e Rastreabilidade Integral**: Toda sessão executada no Prumo Code Agent gera registros estruturados e ordenados deterministicamente em `.prumo/runtime/audit/telemetry.json` e `.prumo/runtime/audit/sessions/<id>.json`. Isso permite auditar tokens de entrada/saída, reaproveitamento de cache, custos acumulados em dólares, arquivos tocados e status de gates de verificação por projeto e por tarefa.

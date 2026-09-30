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

## Os 10 Princípios Canônicos do Prumo

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

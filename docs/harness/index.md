# Prumo Harness — Plano de Controle & Execução

O **Prumo Harness** é o subsistema de execução headless e plano de controle autônomo do Prumo v0.6. Ele conecta o modelo cognitivo (LLM) aos recursos reais do sistema operacional e do repositório através de contratos determinísticos e sandboxing seguro.

## Papel na Arquitetura

Enquanto a camada de protocolo gerencia Metas, Planos, Evidências e Documentação, o **Harness** é responsável por transformar intenções compiladas em ações verificadas:

```
[ Usuário / Agente ]
        │
        ▼
[ Protocolo & Metas ] ── (Compilação LPC) ──► [ Diretivas Executáveis ]
                                                      │
                                                      ▼
                                              [ Prumo Harness ]
                                                      ├── Tool Gateway & ACI
                                                      ├── Sandboxing & Timeouts
                                                      ├── AST & Code Analysis
                                                      └── Verification Envelopes
```

## Capacidades do Harness

- **[Ferramental ACI & Sandboxing](/harness/aci-tools)**: Inspeção de sintaxe, parsers de AST para múltiplas linguagens, runner de comandos isolados com limites estritos de memória e tempo de execução.
- **[Diretivas & Task DAGs](/harness/directives)**: Modelo de execução formal onde cada tarefa possui pré-condições, ações prescritas e critérios de saída verificáveis.
- **Gateway de Modelos & Orçamento**: Roteamento inteligente de modelos (Claude 3.7 Sonnet, GPT-4o, DeepSeek R1/V3, Kimi), monitoramento de custos por token e fallback automático.
- **Daemon de Fundo & Sessões Resumíveis**: Capacidade de retomar tarefas longas interrompidas através do log de eventos e checkpoints em `.prumo/`.

## Executando com o Harness

Para disparar uma execução orquestrada pelo Harness:

```bash
# Executar a próxima tarefa pendente do plano ativo
prumo run

# Executar uma diretiva específica em modo estrito
prumo run --directive ./directives/task-01.json --sandbox pure

# Iniciar o daemon de supervisão em background
prumo daemon start
```

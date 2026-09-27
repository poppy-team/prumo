# Reference Project — Prumo

Este é o **projeto canônico de referência** do Prumo v0.4. Ele exemplifica a arquitetura padrão, o modelo documental agnóstico, as políticas de segurança e a hierarquia onde **cada pasta contém um `README.md` detalhado** com sua finalidade, justificativa e inventário de conteúdo.

## Estrutura do Projeto

```text
reference-project/
├── prumo.json                       # Manifesto canônico do projeto (protocolo v3)
├── ENTRYPOINT.md                    # Roteador de contexto Lean Progressive Context
├── PROJECT_STATE.md                 # Estado operacional, fase ativa e recuperação
├── CHANGELOG.md                     # Histórico estruturado de alterações
├── README.md                        # Visão geral do projeto e guia de navegação
├── .ai/                             # Recursos e orquestração de IA (agentes, metas, planos)
│   └── README.md
├── .prumo/                          # Metadados de runtime, cache e inteligência
│   └── README.md
└── docs/                            # Documentação canônica orientada a papéis
    ├── README.md
    ├── PRUMO.md                     # Roteador de intenção (mapa de documentação)
    ├── architecture/                # Visão de sistemas, Clean Code e ADRs
    ├── product/                     # Visão de produto e escopo
    ├── development/                 # Padrões de código e estratégia exaustiva de testes
    ├── operations/                  # Guias de deploy e observabilidade
    ├── security/                    # Modelo de ameaças e contrato de segurança
    └── governance/                  # Políticas de repositório e branches
```

## Como Usar este Exemplo

1. **Validação do Projeto**:
   ```bash
   prumo-agent validate .
   ```
2. **Diagnóstico do Projeto**:
   ```bash
   prumo-agent doctor .
   ```
3. **Compilar Adaptador para seu Agente**:
   ```bash
   prumo-agent compile --target antigravity .
   prumo-agent compile --target codex .
   prumo-agent compile --target claude-code .
   ```
4. **Ciclo de Desenvolvimento com Metas**:
   Consulte [`docs/development/testing-strategy.md`](docs/development/testing-strategy.md) para o ciclo TDD/BDD com testes exaustivos.

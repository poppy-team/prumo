# Goals 101: Minha Primeira Goal

Goal (objetivo) é a unidade fundamental de planejamento no Prumo. Uma Goal define **o que precisa ser feito**, com critérios de aceitação, dependências e gates de qualidade.

## Conceitos Básicos

### O que é uma Goal?

Uma Goal é um trabalho entregável com:
- **Identificador** único (ex: `P00-G01`)
- **Título** descritivo
- **Fase** de desenvolvimento (ex: `P00` = Foundation, `P01` = Features)
- **Objetivo** mensurável
- **Critérios de aceitação** (acceptance criteria)
- **Gates** de qualidade (build, tests, review, documentação)
- **Estado** (DRAFT, PLANNED, LOCKED, EXECUTING, DONE)

### Estados de uma Goal

```
DRAFT          (inicial, criada)
  ↓
PLANNED        (planejada, pronta para travar)
  ↓
LOCKED         (traçada, pronta para execução)
  ↓
EXECUTING      (em progresso)
  ↓
VERIFYING      (testes rodando)
  ↓
REVIEWING      (revisão de código)
  ↓
DONE           (concluída)
```

## Passo 1: Criar um Projeto

```bash
prumo-agent init my-project --profile examples/brasa/project-profile.json --non-interactive
cd my-project
```

Isso cria:
- `prumo.json` — configuração do projeto
- `.ai/` — diretório de Goals, workforce, etc
- `PROJECT_STATE.md` — estado do projeto

## Passo 2: Criar Sua Primeira Goal

```bash
prumo-agent goal new P00-G01 "Setup CI/CD pipeline" --phase P00
```

**Output esperado:**
```
Created Goal P00-G01 in /path/to/my-project/.ai/goals/P00-G01.goal.json
```

Isso cria um arquivo JSON:

```json
{
  "id": "P00-G01",
  "title": "Setup CI/CD pipeline",
  "phase": "P00",
  "state": "DRAFT",
  "objective": "Define the measurable outcome for Setup CI/CD pipeline.",
  "acceptance": [
    "Replace this placeholder with objective acceptance criteria."
  ],
  "gates": {
    "build": "required",
    "tests": "required",
    "review": "required",
    "documentation_impact": "required",
    "project_intelligence": "required"
  },
  "dependencies": [],
  "evidence": []
}
```

## Passo 3: Detalhar a Goal

Abra `.ai/goals/P00-G01.goal.json` e edite:

```json
{
  "id": "P00-G01",
  "title": "Setup CI/CD pipeline",
  "phase": "P00",
  "state": "DRAFT",
  "objective": "Implementar pipeline CI/CD com testes, linting e builds automáticos",
  "constraints": [
    "Usar GitHub Actions",
    "Testar em Python 3.10+"
  ],
  "non_goals": [
    "Monitoramento de produção",
    "Docker image distribution"
  ],
  "acceptance": [
    "GitHub Actions workflow existe e passa",
    "Testes executam em 3.10, 3.11, 3.12",
    "Coverage ≥ 80%",
    "Linting (ruff, mypy) passa sem erros"
  ],
  "gates": {
    "build": "required",
    "tests": "required",
    "review": "required",
    "documentation_impact": "required",
    "project_intelligence": "required"
  },
  "dependencies": [],
  "evidence": []
}
```

## Passo 4: Planejar (DRAFT → PLANNED)

Quando estiver satisfeito com os detalhes, mude para PLANNED:

```bash
prumo-agent goal state P00-G01 PLANNED --reason "CI/CD details finalized"
```

**Output:**
```
Goal P00-G01 transitioned to PLANNED
```

## Passo 5: Travar (PLANNED → LOCKED)

Quando confirmar que não vai mudar, trave para executar:

```bash
prumo-agent goal state P00-G01 LOCKED
```

**Output:**
```
Goal P00-G01 transitioned to LOCKED
```

Isso computa um digest SHA256 das aceitações e gates. Se alguém mudar a Goal depois, o digest não vai combinar e você verá um erro.

## Passo 6: Executar (LOCKED → EXECUTING)

Comece o trabalho:

```bash
prumo-agent goal state P00-G01 EXECUTING --reason "Started implementation"
```

## Passo 7: Adicionar Evidência

Conforme você completa testes, builds e reviews, registre a evidência:

```bash
# Crie um arquivo de report
cat > report.json << 'EOF'
{
  "id": "P00-G01-TEST-1",
  "status": "success",
  "type": "test",
  "tokens": {
    "input": 1000,
    "output": 200,
    "cached": 0
  },
  "cost": {
    "direct": {
      "amount": 0.12,
      "currency": "USD",
      "provenance": "observed"
    }
  }
}
EOF

# Registre a evidência
prumo-agent report add report.json
```

## Passo 8: Revisar e Completar

Quando tudo estiver pronto:

```bash
prumo-agent goal state P00-G01 VERIFYING
prumo-agent goal state P00-G01 REVIEWING --reason "Code reviewed, all gates green"
prumo-agent goal state P00-G01 DONE
```

## Listar Todas as Goals

```bash
prumo-agent goal list
```

**Output:**
```
Phase P00 — Foundation
  P00-G01  Setup CI/CD pipeline                          [DONE]
```

## Adicionar Dependências

Se Goal B depende de Goal A:

```bash
# Edite .ai/goals/P00-G02.goal.json
{
  "dependencies": ["P00-G01"],
  ...
}
```

## Emendar uma Goal Travada

Se traçou mas precisa mudar depois:

```bash
# Crie um arquivo de emenda
cat > amendment.json << 'EOF'
{
  "id": "amendment-1",
  "goal_id": "P00-G01",
  "changes": {
    "add_acceptance": [
      "Performance: CI/CD completes in <5 minutes"
    ]
  },
  "reason": "New acceptance criterion from stakeholder",
  "approved_by": "pm@example.com"
}
EOF

# Aplique a emenda
prumo-agent goal amend P00-G01 --file amendment.json
```

Isso incrementa a revisão de 1 → 2 e recomputa o digest.

## Dicas Práticas

1. **Comece com objetivos claros** — Não deixe placeholder nas aceitações
2. **Divida em fases** — `P00` (Foundation), `P01` (Features), `P02` (Polish)
3. **Use nomenclatura consistente** — `P{phase}-G{number}`, ex: `P00-G01`, `P01-G03`
4. **Trave quando estiver certo** — LOCKED garante que ninguém muda sem formal amendment
5. **Registre evidência** — Testes, builds, reviews deixam rastreabilidade
6. **Documente não-goals** — Deixa claro o que NÃO está no escopo

## Troubleshooting

**Erro: "Invalid goal transition: DRAFT → EXECUTING"**
- Goals devem passar por estados válidos: DRAFT → PLANNED → LOCKED → EXECUTING

**Erro: "Lock digest mismatch"**
- Você editou a Goal depois de travar. Use `prumo-agent goal amend` para mudanças formais.

**Erro: "Goal is in locked/executing state but lacks a lock record"**
- Salve o arquivo sem editar os campos criticos (objective, acceptance, gates)

## Próximos Passos

- [Goals Avançadas](./goals.md) — Locking, amendments, audit trail
- [Plans & DAGs](./plans-and-tasks.md) — Criar planos de execução
- [Evidence & Gates](./evidence-and-gates.md) — Registrar resultados
- [Project Intelligence](./project-intelligence.md) — Custos e métricas

# Política de Autoridade e Projeção

> Autoridade: canônica do repositório. Fonte única da verdade para a ordem de
> autoridade da documentação, papéis e regras de drift (W0.6, W0.8, W0.11).

## Ordem de autoridade

1. Specs, schemas, contratos e ADRs aceitas canônicos do repositório
2. Documentação de engenharia aceita em `docs/`
3. Livro Vivo do Prumo (Notion) — insumo de design até ser promovido
4. Inferência do agente
5. Fontes externas

Depois que uma decisão de design é promovida para uma especificação canônica do
repositório, a versão do repositório tem autoridade sobre a versão do Notion para
a implementação.

## Papéis

| Papel | Significado | Verificado quanto a drift |
|-------|-------------|---------------------------|
| `canonical` | Detém os fatos do projeto | sim |
| `projection` | Superfície derivada; deve resolver para uma fonte canônica | sim |
| `historical` | Registro (ADR, nota de migração, log de progresso); nunca reescrito | não |

A classificação legível por máquina é o `docs/AUTHORITY_MAP.json`, verificado por
`prumo-agent docs authority` e pelo gate de CI `docs-authority`.

## Política de projeção (W0.11)

- Adapters de agentes e providers — `AGENTS.md`, arquivos de regras de Copilot/Cursor/Claude,
  manifestos de conectores, SDKs gerados — são **projeções**: não podem
  introduzir fatos de projeto independentes.
- Uma projeção declara seu `canonical_source`. Uma projeção cuja fonte é
  ela mesma uma projeção é um **ciclo** e falha no gate.
- Adapters gerados carregam metadados de proveniência/fingerprint assim que o Agent
  Surface Compiler (W16) for entregue.
- Índices derivados, resumos e Working Context Capsules vivem em runtime/cache
  e nunca se tornam canônicos.

## Regras de drift (W0.8)

- Um documento **ativo** (canônico ou projeção) não pode referenciar uma linha de
  release do projeto mais antiga que a versão atual da CLI, a menos que a linha traga uma
  anotação histórica (`historical`, `legacy`, `migration`, `retired`,
  `superseded`, `deprecated`, `no longer`, `compatibility oracle`).
- Superfícies de roteamento (`AGENTS.md`, `ENTRYPOINT.md`, `README.md`, `FRAMEWORK.md`,
  `docs/PRUMO.md`) devem ter links relativos resolvíveis.
- Isenções por documento exigem um `drift_exempt_reason` explícito no
  mapa de autoridade — nunca um enfraquecimento silencioso do gate.

## Verificação

```bash
prumo-agent docs authority            # saída para humanos
prumo-agent docs authority --json     # envelope para máquinas
go test ./internal/documentation/ -run TestAuthority
```

Grupo de gates do CI: `docs-authority` (veja `docs/development/waves.md`).

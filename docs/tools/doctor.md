# Diagnóstico do Repositório (`prumo doctor`)

O comando `prumo doctor` é o médico do repositório. Ele realiza uma inspeção minuciosa em múltiplos níveis para garantir que o projeto está íntegro e pronto para execução autônoma segura.

## Categorias de Checagens

1. **Integridade de Metas & Planos**:
   - Verifica se todas as Metas no estado `LOCKED` possuem digests SHA-256 válidos e não foram modificadas manualmente no Git.
   - Valida se o DAG de tarefas dos Planos ativos não possui ciclos ou nós órfãos.

2. **Conformidade de JSON Schemas**:
   - Valida `prumo.json`, perfis e manifestos contra o Draft 2020-12 dos JSON Schemas canônicos embutidos no binário.

3. **Ambiente & Ferramental**:
   - Detecta presença do Git e integridade do repositório local.
   - Checa executáveis requeridos por perfis e receitas ativas (compiladores, linters, test runners).

4. **Conectores & Adaptadores**:
   - Verifica se os adaptadores compilados em `CLAUDE.md`, `.cursorrules`, etc., estão sincronizados com as definições mais recentes de `prumo.json`.

## Exemplo de Execução

```bash
prumo doctor
```

Saída típica em modo texto:

```text
[OK] Git repository clean and initialized
[OK] prumo.json valid against schema v1
[OK] 14 Goals verified, SHA-256 locks intact
[OK] Task DAGs verified without cycles
[OK] Adapters synchronized: claude-code, generic
Prumo repository health check passed (100% operational).
```

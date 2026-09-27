# 62 — Repository Change Governance, GitHub Policy e Agent SCM Safety

> Authority: canonical specification.
> Logical ID: PHASE-62
> Source: Notion Living Book (3d69bb7d023f8170be73f4caeb575771)
> Status: Fase/Gate de implementação (62 — Repository Change Governance, GitHub Policy e).


<aside>
🔀

**Status:** novo prerequisite arquitetural antes do M5. O Atlas precisa governar mudanças de repositório como parte do protocolo, não apenas confiar em convenções de agentes ou configurações manuais do GitHub.

</aside>

# Objetivo

Formalizar uma camada provider-agnostic de **Repository Change Governance** para branches, commits, pushes, issues, pull requests, reviews, merges, tags/releases, automações e exceções operacionais. GitHub é o primeiro adapter, não a autoridade semântica.

# Princípio

```
Desired Repository Policy
        ↓
Atlas Repository Governance
        ↓
SCM Adapter
        ↓
GitHub / GitLab / outros
```

As políticas pertencem ao Atlas. O adapter converte estado desejado em configuração/verificação específica do host.

# Contratos principais

- `BranchPolicy`
- `CommitPolicy`
- `PushPolicy`
- `IssuePolicy`
- `PullRequestPolicy`
- `ReviewPolicy`
- `MergePolicy`
- `TagReleasePolicy`
- `AutomationIdentityPolicy`
- `EmergencyBypassPolicy`

# Estratégia de branches

Adotar **trunk-based development leve**. `main` representa estado integrado, validado e potencialmente liberável. Não introduzir `develop`/`staging` permanentes sem necessidade comprovada.

Branches curtas sugeridas: `feat/<goal>-<slug>`, `fix/<issue>-<slug>`, `docs/<slug>`, `refactor/<slug>`, `test/<slug>`, `chore/<slug>`, `release/<version>`, `hotfix/<slug>`.

Agents não fazem push direto para `main` por padrão.

# Commits

Adotar Conventional Commits simplificado: `type(scope): resumo imperativo`.

Tipos: `feat`, `fix`, `refactor`, `perf`, `test`, `docs`, `build`, `ci`, `chore`, `revert`.

Commits devem ser logicamente atômicos, reversíveis e explicar o motivo quando não for óbvio. Evitar mensagens `update`, `fix`, `wip`, `final` e equivalentes sem semântica.

# Pull requests

Todo merge em `main` deve ocorrer via PR. PR é unidade lógica de mudança e deve registrar no mínimo: why, what, scope, non-goals, Goal/Issue, risk, validation, evidence, documentation delta, migration impact e rollback.

Para mudanças originadas por agents, registrar provenance operacional estruturada — Goal, Run, Evidence, harness/agent — sem armazenar chain-of-thought.

# Review por risco

- **Solo/low:** PR + CI + conversas resolvidas; merge humano, sem approval externo obrigatório.
- **Team/medium:** PR + CI + pelo menos um review conforme política do projeto.
- **High:** CI + independent verifier + review.
- **Critical:** CI + verifier independente + aprovação humana explícita + gates específicos de segurança/evidência.

A política deve ser configurável por profile, mas invariantes como proibição de rewrite silencioso de `main` são hard-coded.

# Merge

**Squash merge é o padrão recomendado.** Um PR lógico deve virar um commit lógico em `main`. Merge commit/rebase podem existir como capacidade do adapter, mas não devem ser default do Atlas.

Após merge, branches descartáveis devem ser removidas automaticamente quando o host permitir.

# Push e operações privilegiadas

Classificar operações SCM no Tool Gateway:

- read-only: status, log, diff, fetch metadata;
- reversible: criar branch, commit local;
- side-effecting: push em feature branch, criar Issue/PR;
- destructive/privileged: force push, delete remote branch, merge em default branch, mover tag publicada;
- forbidden por default: force push/rewrite de `main`, mover tag de release imutável.

`--force-with-lease` em branch de trabalho só pode existir quando policy explicitamente permitir. `--force` cego deve ser negado.

# Issues

Issues são **work intake**, bugs e propostas; não são fonte canônica de arquitetura. Decisão aceita deve ser promovida para ADR/spec/Goal/documentação canônica.

Templates mínimos: bug, feature, documentation, proposal. Segurança privada deve apontar para `SECURITY.md`, não para issue pública.

# GitHub profile

Criar um GitHub adapter/profile que consiga verificar e futuramente aplicar estado desejado. Para `main`, o desired state deve incluir:

- PR obrigatório;
- required status checks;
- conversas resolvidas;
- force push bloqueado;
- exclusão da branch protegida bloqueada;
- histórico linear quando compatível com merge policy;
- direct push bloqueado;
- squash merge habilitado e recomendado como único método;
- branch deletion após merge;
- auto-merge desabilitado inicialmente;
- merge queue adiado até existir concorrência real de múltiplos agents/colaboradores.

# Assinaturas

Signed commits não são requisito inicial universal. Profiles `strict`/`release` podem exigir assinatura, especialmente para tags/releases. Não criar fricção obrigatória para todos os harnesses antes de existir suporte de identidade automatizada coerente.

# Releases e tags

SemVer para releases: `v0.4.0-alpha.1`, `v0.4.0-beta.1`, `v0.4.0-rc.1`, `v0.4.0`.

Tags publicadas são imutáveis. Agent não pode mover tag de release silenciosamente. Release deve passar gates definidos, gerar provenance/evidence e changelog/release narrative conforme contratos do Atlas.

# Emergency bypass

Toda exceção privilegiada deve exigir autorização humana explícita, motivo estruturado, identidade, timestamp, operação, evidência e post-event review. Bypass nunca é solução automática para CI inconveniente.

# Arquivos de implementação esperados

```
CONTRIBUTING.md
.github/
├── CODEOWNERS
├── PULL_REQUEST_TEMPLATE.md
└── ISSUE_TEMPLATE/
    ├── bug.yml
    ├── feature.yml
    ├── documentation.yml
    ├── proposal.yml
    └── config.yml

docs/governance/
├── repository-governance.md
└── release-policy.md
```

Não criar documentação fragmentada sem necessidade; consolidar quando melhorar manutenção.

# Repository Policy Contract

O Atlas deve evoluir para uma configuração machine-readable, provider-agnostic, capaz de expressar: default branch, strategy, branch naming, direct-push rules, commit convention, PR requirement, review profile, merge strategy, required checks, agent permissions, release/tag immutability e bypass policy.

O formato final deve possuir JSON Schema e ser validável pelo Core.

# CLI alvo

Planejar sem necessariamente implementar toda a superfície imediatamente:

```
atlas repo policy check
atlas repo policy explain
atlas repo policy plan
atlas repo policy apply
```

`check` compara desired vs actual; `plan` produz delta sem side effect; `apply` exige permissões e política de aprovação adequadas.

# Conformance

Adicionar fixtures/testes para:

- branch name válida/inválida;
- commit message válida/inválida;
- tentativa de push direto à main;
- force push;
- PR sem metadata obrigatória;
- review profile por risco;
- merge strategy inválida;
- tag mutation;
- emergency bypass registrado;
- desired GitHub state vs actual state.

# Agent permission matrix

Agents devem poder ler Git livremente dentro do escopo do projeto e criar commits/branches de trabalho. Push remoto, Issue e PR são side effects auditáveis. Merge em `main`, alterações de ruleset, releases e operações destrutivas exigem capacidade explícita. Nenhum harness recebe autoridade implícita apenas por possuir credencial técnica.

# Gate antes do M5

M5 Documentation System v2 só deve avançar após a fundação mínima desta governança existir:

- Repository Policy Contract documentado;
- convenções de branch/commit/PR/review/merge/release aprovadas;
- `CONTRIBUTING.md` e templates GitHub;
- policy check determinístico inicial;
- CI/required-check mapping conhecido;
- agent SCM permission matrix;
- estratégia de enforcement no GitHub definida.

# Não-objetivos desta fase

- construir merge queue própria;
- suportar todos os SCMs;
- servidor central de equipes;
- auto-merge agentic irrestrito;
- obrigar assinatura universal de commits;
- criar um Git workflow complexo sem necessidade.

# Regra arquitetural

**Hard-code invariants; configure policies; evaluate heuristics.** Rewrite silencioso de `main` e mutação de tags release são invariantes; quantidade de reviews e auto-merge são policies; quando abrir Issue automaticamente pode ser heurística avaliada.
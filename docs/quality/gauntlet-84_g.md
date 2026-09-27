# 84.G — Release Candidate Exit Gates, 10/10 Sem Média e Continuous Assurance

> Authority: canonical specification.
> Logical ID: 84 G
> Source: Notion Living Book (3e29bb7d023f81e58e99e8e58035fd93)
> Status: Constituição 84: Perfis do Gauntlet / Total Assurance (84 G — Release Candidate Exit Gates, 10 10 Sem Méd).


<aside>
🏁

**Regra:** score é resumo; gate é autoridade. Nenhum “overall 9.8” pode compensar data loss, dead control, security finding ou acceptance sem evidence.

</aside>

# Scorecard

Áreas aplicáveis recebem status e evidência, não opinião. Exemplos:

Architecture, Code Quality, Functional, UI/UX, Performance, Security, Persistence, Compatibility, Accessibility, i18n, Error Handling, Observability, Packaging, Recovery, Documentation e Regression.

# 10/10 semantics

Para uma área ser 10/10 dentro do scope:

- required checks = pass;
- failed = 0;
- untested required = 0;
- unknown required = 0;
- stale required evidence = 0;
- known unwaived defect = 0;
- required review completed;
- scope/non-goals frozen.

Não representa perfeição metafísica; representa satisfação comprovada do contrato aceito.

# Defect severity

Baseline:

BLOCKER, CRITICAL, HIGH, MEDIUM, LOW, POLISH.

Policy define quais podem receber waiver. Blocker/data-loss/security-critical normalmente não.

# Issue ledger

Cada finding:

ID, severity, subsystem, reproduction, root cause, affected surface, fix, regression test, evidence, owner e status.

# Root-cause gate

Repeated superficial patches indicam failure de review. Para findings recorrentes, exigir análise de ownership/invariant/boundary.

# Fresh-state final run

Release candidate precisa de execução em estado limpo ou environment reproduzível equivalente: instalar/launch, executar workflows, save/reopen quando aplicável, uninstall/cleanup quando relevante.

# Artifact validation

Testar o artefato que será entregue, não somente source tree. Verificar package contents, versions, signatures/hashes, runtime dependencies e startup.

# Compatibility matrix

Release deve declarar supported platforms/runtime/project formats. O que não foi testado não pode ser implicitamente chamado suportado.

# Post-release assurance

Continuous assurance observa regressões, incidents, crash/telemetry consentida, performance drift, dependency vulnerability, provider/model drift e docs staleness. Release não encerra o lifecycle.

# Canary/rollback

Mudanças de resolver, model, provider, migration ou updater críticas passam por shadow/canary quando aplicável e mantêm rollback real.

# Final questions

Antes de promover, responder com evidence:

- há surface pública não inventariada?
- há acceptance sem prova?
- há teste que não detectaria implementação errada?
- há failed/skipped mascarado?
- há stale evidence?
- há unowned state/resource?
- há failure mode sem recovery?
- há performance sem budget/baseline?
- há security assumption não testada?
- há incompatibilidade não declarada?
- há UI reachability quebrada?
- há doc/code drift?
- há known defect sem owner/waiver?

Qualquer YES relevante mantém o Gauntlet ativo.

# Continuous Assurance

A versão final do Prumo deve conseguir reexecutar somente a porção necessária do Gauntlet usando change impact + risk + evidence dependencies, sem cair no extremo de rodar tudo sempre. Rigor e eficiência coexistem por seleção inteligente, nunca por omissão silenciosa.
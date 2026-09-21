# 84.F — Documentation Refactoring, Drift, NFR Coverage e Living Truth Assurance

> Authority: canonical specification.
> Logical ID: 84 F
> Source: Notion Living Book (3e29bb7d023f8139a32ee1744fe2802d)
> Status: Constituição 84: Perfis do Gauntlet / Total Assurance (84 F — Documentation Refactoring, Drift, NFR Cover).


<aside>
📚

**Princípio:** documentação madura descreve não apenas o que existe, mas como provar, operar, degradar, recuperar, evoluir e invalidar o conhecimento.

</aside>

# Documentation Contract v3 additions

Cada semantic role deve declarar quando aplicável:

- positive behavior;
- negative guarantees;
- state ownership;
- lifecycle;
- failure modes;
- recovery;
- concurrency;
- persistence;
- compatibility;
- migration;
- security assumptions;
- privacy/data classes;
- performance budgets;
- resource ownership;
- observability;
- debugging;
- release/update/uninstall;
- accessibility/i18n;
- evidence/oracles;
- known unknowns;
- update triggers;
- staleness dependencies.

# Normative language

Documentos de contrato devem distinguir:

MUST, MUST NOT, SHOULD, MAY ou equivalentes definidos pelo projeto.

Evitar “idealmente”, “deve funcionar bem”, “performance boa” sem critério.

# Decision versus speculation

Marcar:

accepted decision;

proposal;

experiment;

hypothesis;

future;

deprecated;

rejected.

Agente não deve promover hipótese a requirement por repetição.

# Canonicality

Cada fato crítico precisa de owner/source canônica. Docs narrativas podem referenciar, não duplicar sem strategy de sync.

# Drift types

- code ahead of docs;
- docs ahead of code;
- tests ahead/behind behavior;
- UI spec diverges from implementation;
- performance budget sem benchmark;
- security model sem current architecture;
- shortcut/menu drift;
- dependency/version drift;
- stale screenshot;
- stale compatibility matrix.

# Documentation Gauntlet

Para material project:

inventory → semantic coverage → contradiction → source linkage → implementation comparison → test/evidence linkage → NFR coverage → failure/recovery → user-facing surfaces → freshness → refactor → verify.

# Documentation quality evidence

Pass não depende apenas de existência de headings. Deve verificar conteúdo semanticamente suficiente e, quando possível, evidence pointers reais.

# Runnable documentation

Commands/examples/configs públicos devem ser testados ou claramente marcados illustrative. Preferir doc-tests, fixtures ou smoke verification onde custo permitir.

# Documentation for performance/security

Performance docs precisam de workload, environment, baseline, target e regression policy.

Security docs precisam de assets, boundaries, threat assumptions, permission model, residual risks e refresh triggers.

# Operational documentation

Para software complexo, incluir diagnostics, logs, crash reports, recovery, cache reset, safe mode quando aplicável, backup/migration e troubleshooting.

# Known limitations

Limitação conhecida não pode ficar escondida no transcript. Deve ter scope, impact, workaround e trigger de revisão.

# Refactoring rule

Refatorar documentação não significa apagar história útil. Preservar decisions/ADRs, rejeitados e migration context; reduzir duplicação, contradictions e informação sem owner.

# Exit

Documentation ready significa: implementação pode ocorrer sem decisões críticas implícitas; verification pode provar critérios; operação/recovery têm orientação suficiente; future maintainer entende boundaries e trade-offs.
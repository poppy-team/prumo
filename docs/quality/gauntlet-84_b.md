# 84.B — Blind-Spot Hunting, Oracle Quality, Negative Space e Adversarial Assurance

> Authority: canonical specification.
> Logical ID: 84 B
> Source: Notion Living Book (3e29bb7d023f810e9bdfc5ff8506ea9c)
> Status: Constituição 84: Perfis do Gauntlet / Total Assurance (84 B — Blind-Spot Hunting, Oracle Quality, Negativ).


<aside>
🕳️

**Princípio:** requisitos conhecidos não cobrem automaticamente classes de falha desconhecidas. Toda run material deve reservar uma fase para caçar pontos cegos.

</aside>

# Blind-spot lenses

O reviewer deve percorrer lenses independentes:

- architecture;
- state ownership;
- concurrency;
- persistence;
- cancellation;
- recovery;
- resource exhaustion;
- security/trust;
- compatibility;
- platform/driver;
- user error;
- malformed input;
- accessibility;
- localization;
- observability;
- release/update;
- uninstall/cleanup;
- long-session degradation;
- external dependency failure;
- data loss/corruption;
- legal/license/provenance quando relevante.

# Pre-mortem

Antes do gate final, perguntar: “se esta mudança causar incidente em produção, quais seriam cinco causas plausíveis que nossos testes atuais não detectariam?”. Cada causa deve ser classificada como already-covered, new-fixture, documented-risk ou not-applicable.

# Oracle quality

Todo teste depende de um oracle. Classificar:

- exact;
- invariant;
- metamorphic;
- differential;
- reference implementation;
- structured human review;
- probabilistic eval;
- weak/unknown.

Critical tests com oracle weak/unknown não podem ser tratados como prova forte.

# Mutation sensitivity

Quando risco justificar, usar mutation testing ou fault seeding para verificar se a suíte realmente detecta mudança errada. Não perseguir 100% universal; usar para código crítico, parsers, authorization, state machines e invariants.

# Negative space

Testar coisas que NÃO devem acontecer:

- ausência de side effect;
- ausência de network call;
- ausência de secret em log;
- ausência de mutation após cancel;
- ausência de duplicate external action após retry;
- ausência de stale selection/reference;
- ausência de unauthorized fallback;
- ausência de silent data migration;
- ausência de hidden telemetry.

# Sequence bugs

Gerar testes de ordem:

A→B, B→A, A→Undo→C, cancel midway, switch context, reopen, retry, resume, repeated operation. Muitas falhas agentic aparecem em sequência, não em operação isolada.

# Fault injection

Profiles podem acionar falhas controladas:

- filesystem error;
- disk full;
- read-only;
- network timeout;
- provider error;
- allocation failure quando viável;
- corrupted cache;
- killed process;
- cancelled task;
- unavailable GPU/device;
- stale lock;
- concurrent edit.

# Nondeterminism

Detectar:

- race;
- flaky timing;
- unstable ordering;
- unseeded randomness;
- timestamp dependence;
- locale/timezone dependence;
- filesystem ordering;
- GPU nondeterminism.

Quando necessário, repetir critical scenario e registrar seed/environment.

# Inconclusive como resultado legítimo

Se não há oracle confiável ou ambiente suficiente, resultado é inconclusive. O Prumo deve preferir honestidade a falso pass.

# Regression promotion

Todo bug significativo corrigido deve ser candidato a regression fixture. Incidentes reais, quando sanitizados, alimentam corpus e blind-spot taxonomy.
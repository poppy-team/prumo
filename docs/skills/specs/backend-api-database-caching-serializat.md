# 79.K — Backend, API, Database, Caching, Serialization e RPC Skills

> Authority: canonical specification.
> Logical ID: 79 K
> Source: Notion Living Book (3d89bb7d023f81358874c144d514c488)
> Status: Skill Package de referência para 79 K — Backend, API, Database, Caching, Serializat.


<aside>
🗄️

Backend/Data hardening deve cobrir não apenas correctness local, mas **contratos públicos, consistência, evolução, operação, recuperação e limites de recurso**.

</aside>

| Skill | Pri. | Lacuna / hardening | Evidence/aceite |
| --- | --- | --- | --- |
| **backend-api** | P1 | Idempotency, pagination, limits, versioning, consistency, cancellation/timeouts, observability, authz e error semantics. | API contract cobre success/error/limit/retry cases. |
| **api-contract-testing** | P1 | Consumer/provider tests, schema diff, negative/boundary cases, backward/forward compatibility e version gates. | Breaking contract detectado pre-integration. |
| **database-review** | P1 | Migration safety, locks/isolation, EXPLAIN/indexes, N+1, backup/restore, retention, access e growth. | Deploy-time DB risk explicitado. |
| **caching** | P1 | TTL/eviction, invalidation, stampede, stale-while-revalidate, negative cache, multi-node consistency e poisoning. | Hit/miss/stale/invalidation/concurrent refill tests. |
| **serialization** | P1 | Versioning, unknown fields, canonicalization, depth/size limits, unsafe polymorphism e fuzz. | Compatibility + adversarial fixtures. |
| **rpc-protocols** | P1 | Deadlines, retries, idempotency, streaming, backpressure, auth/version negotiation e tracing. | N/N-1 protocol matrix. |
| **dependency-management** | P1 | Pinning/locks, changelog/API diff, CVEs, transitive risk, licenses, cadence e rollback. | Dependency addition/update possui rationale+risk evidence. |

# API Change Classification

- additive non-breaking;
- behavior change without schema change;
- deprecated;
- breaking schema/semantic;
- security-sensitive;
- quota/performance-sensitive.

A classificação determina compatibility/security/performance gates.

# Database migration lifecycle

```
preflight
→ backup/restore confidence
→ expand
→ dual-read/write when needed
→ backfill
→ verify
→ cutover
→ contract
→ observe
→ close rollback window
```

Não aplicar esse fluxo completo a migrations triviais; usar risk profile.

# Consistency contract

Toda operação stateful relevante deve deixar explícito: transaction boundary, isolation/locking assumptions, retry semantics, idempotency, concurrency conflicts e recovery behavior.

# Cache as derived state

Cache nunca deve virar implicitamente fonte canônica. Definir authoritative source, invalidation trigger, stale tolerance e behavior on cache failure.

# Backend observability

Critical endpoints/jobs devem definir logging context, tracing/correlation, relevant counters/histograms, error classification e alert/SLO only where operationally justified.

# Exit Gate

Public/stateful changes recebem classification e gates adequados; migrations possuem recovery semantics; API/RPC evolution é testada; caches não escondem consistência indefinida.
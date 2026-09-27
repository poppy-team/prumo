# 84.D — Security, Resilience, Data Integrity, Recovery e Failure Containment Assurance

> Authority: canonical specification.
> Logical ID: 84 D
> Source: Notion Living Book (3e29bb7d023f811b9968e49ab5de7d7e)
> Status: Constituição 84: Perfis do Gauntlet / Total Assurance (84 D — Security, Resilience, Data Integrity, Recov).


<aside>
🔐

**Princípio:** segurança e resiliência são properties de comportamento sob abuso, erro e falha, não uma checklist de linters.

</aside>

# Security delta

Toda mudança material deve responder:

- novos assets;
- novos entry points;
- novas trust boundaries;
- novos permissions;
- novo egress;
- novo parser/input;
- novo filesystem/process behavior;
- nova dependency;
- novo updater/plugin path;
- mudança de secrets;
- mudança de privilege.

Isso determina specialists e testes.

# Data integrity

Validar:

- atomicity;
- consistency;
- partial writes;
- crash during save;
- corrupted file detection;
- versioning/migration;
- backup/restore quando aplicável;
- duplicate mutation;
- idempotency;
- stale revision protection;
- concurrency conflicts.

# Filesystem adversarial set

Quando aplicável:

path traversal, absolute/relative confusion, symlink, TOCTOU, permission denied, readonly, special files, invalid names, long paths, case sensitivity, path normalization, temp cleanup e archive extraction.

# Parser/input robustness

Malformed, truncated, oversized, duplicate fields, unknown versions, invalid encoding, NaN/Infinity, integer bounds e deeply nested input quando relevante.

# Resource exhaustion

Threat model inclui CPU, RAM, VRAM, disk, file descriptors, process count, queue size, recursion/depth e unbounded collections. Graceful refusal é preferível a host failure.

# Supply chain

Para release-critical paths:

dependency pin/lock, provenance, SBOM quando aplicável, vulnerability review, license policy, signing/attestation e artifact hash.

# Secrets and privacy

Secret nunca vira evidence textual por conveniência. Testar redaction. Telemetry/export/debug bundles devem aplicar data classification e egress policy.

# Recovery

Failure behavior precisa ser documentado:

retryable? idempotent? rollback? resume? cleanup? user action?

Retry automático em permanent failure é finding.

# Crash containment

Plugin/provider/tool/worker failure não deve necessariamente derrubar host. Quando isolation não existe, risco precisa ser explícito.

# Security verifier independence

High/critical security change exige review independente conforme policy e evidence vinculada à revision final.

# Abuse-case corpus

Além de CVEs/checklists, manter fixtures de abuso por capability. Security skill deve mapear threat → check/evidence → residual risk.

# Secure default

Quando state/policy/evidence está ausente em decisão de alto impacto, default não deve ser permissivo silencioso. Fail-safe behavior deve ser declarado por classe.
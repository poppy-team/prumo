# Caching Architecture & Invalidation — Verification Checklist

## 1. Capacity Limits & Eviction Mechanics
- [ ] In-process (L1) cache specifies an explicit upper bound (maximum item count or maximum byte size).
- [ ] Eviction algorithm is explicitly defined (LRU, LFU, ARC, or W-TinyLFU) and executes in $O(1)$ amortized time.
- [ ] Continuous insertion beyond capacity triggers eviction without memory leaks or unbounded growth.
- [ ] Memory fragmentation and GC overhead (for managed runtimes) are bounded by reusing buffers or using off-heap / arena memory where appropriate.

## 2. Failure Mode Mitigations
- [ ] **Cache Stampede (Thundering Herd)**:
  - [ ] Concurrent requests for the same cold or expired key are coalesced via SingleFlight, distributed mutex, or lease.
  - [ ] Origin database receives exactly 1 call during concurrent lookups for an uncached key.
- [ ] **Cache Avalanche (Mass Expiration)**:
  - [ ] Keys configured with uniform or normal jitter added to base TTL ($\pm 10\% - 25\%$).
  - [ ] Cold-start pre-warming scripts or staggered population strategies are documented for mass cache reboot.
- [ ] **Cache Penetration (Non-Existent Keys)**:
  - [ ] Negative lookups (entity not found) are cached as sentinel values with short TTL ($30\text{s} - 120\text{s}$) or guarded by a Bloom/Cuckoo filter.
- [ ] **Cache Breakdown (Hot Key Expiry)**:
  - [ ] Critical hot keys employ background probabilistic refresh (XFetch) or dedicated background refresh workers.

## 3. Invalidation & Consistency Invariants
- [ ] Invalidation occurs **after** the primary database transaction successfully commits.
- [ ] No uncommitted dirty reads: cache writes do not execute within an uncommitted relational transaction.
- [ ] Distributed cache mutations use transactional outbox, CDC (Change Data Capture), or publish-subscribe invalidation channels.
- [ ] Dual-write race conditions are mitigated by using key deletion (`DEL`) instead of overwriting (`SET`) during updates, forcing subsequent reads to re-fetch canonical state.

## 4. Key Namespacing & Serialization
- [ ] Key naming format follows canonical hierarchical taxonomy: `<service>:<domain>:<version>:<entity_id>`.
- [ ] Zero unescaped characters or user-controlled injection vulnerabilities in cache keys.
- [ ] Serialization format is versioned, compact, and fast (Protobuf, MessagePack, or optimized JSON).
- [ ] Schema evolution rules support forward/backward compatibility without deserialization crashes.

## 5. Resilience & Operational Monitoring
- [ ] Cache client implements circuit breaking and timeouts on all remote network calls (timeout $\le 100\text{ms}$).
- [ ] Fail-open behavior: cache cluster outage degrades to origin database with rate limiting rather than cascading system failure.
- [ ] Metrics instrumentation exposes: Hit Rate ($H$), Miss Rate ($1-H$), Eviction Count, Origin Fetch Latency, and Memory Utilization.
- [ ] Automated verification script `scripts/verify.sh` executes with exit code 0.

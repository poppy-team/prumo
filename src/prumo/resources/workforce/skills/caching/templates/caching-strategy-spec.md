# Caching Architecture & Invalidation Strategy Specification Template

## 1. System Overview & Caching Tiers
- **Service Identifier**: [e.g., UserService / InventoryService]
- **Target Read/Write Ratio**: [e.g., 90% Read / 10% Write]
- **Peak Read QPS**: [e.g., 15,000 requests/second]
- **L1 In-Process Cache**: [Go In-Memory LRU / Java Caffeine / Rust Moka] (Max Heap: `256 MB`)
- **L2 Distributed Cache**: [Redis 7.2 Cluster / Dragonfly] (Cluster Size: `3 Nodes`, Max Memory: `8 GB`)
- **L3 Edge CDN**: [Cloudflare / Fastly] (Static or SWR endpoints)

---

## 2. Key Namespace Taxonomy & Serialization

```
<environment>:<service>:<entity>:<version>:<primary_key>
```

| Entity Type | Cache Key Pattern | Serialization Format | Target Max Payload Size |
|---|---|---|---|
| `[User Profile]` | `prod:user:profile:v1:{user_id}` | Protocol Buffers (`proto3`) | $\le 2.0\text{ KB}$ |
| `[Product Details]` | `prod:catalog:product:v2:{sku}` | MessagePack | $\le 4.5\text{ KB}$ |
| `[Session Token]` | `prod:auth:session:v1:{token_hash}` | Raw JSON | $\le 512\text{ bytes}$ |

---

## 3. Entity Caching Lifecycle & Eviction Policies

| Entity | Eviction Algorithm | Base TTL | Jitter Window | Invalidation Trigger | Staleness Window ($\Delta t_{\text{stale}}$) |
|---|---|---|---|---|---|
| `UserProfile` | LRU (Bounded $50{,}000$ items) | `15 min` | $\pm 90\text{ s}$ | DB Post-Commit Hook / CDC Event | $\le 500\text{ ms}$ |
| `CatalogSKU` | W-TinyLFU (Bounded $100{,}000$ items)| `2 hours` | $\pm 10\text{ min}$ | Product Updated Kafka Message | $\le 2.0\text{ s}$ |
| `SiteMetadata` | Static / Refresh-Ahead | `24 hours`| $\pm 30\text{ min}$ | Admin CMS Webhook | $\le 5.0\text{ s}$ |

---

## 4. Failure Mode Mitigations & Defense Strategy

### 4.1 Cache Stampede (Thundering Herd)
- **Mechanism**: `SingleFlight` request coalescing (`singleflight.Group`).
- **Guaranteed Invariant**: For any simultaneous burst of requests for an uncached key, the origin data fetcher executes exactly 1 query. All concurrent callers receive the same coalesced result.

### 4.2 Cache Avalanche Prevention
- **Mechanism**: Randomized uniform TTL jitter applied at insertion:
  $$\text{TTL}_{\text{effective}} = \text{TTL}_{\text{base}} + \text{rand}(-J, +J)$$
  where $J = 0.15 \times \text{TTL}_{\text{base}}$.

### 4.3 Cache Penetration Defense
- **Mechanism**: Negative / Null-Object Caching + Bloom Filter.
- **Rule**: If origin database returns `RecordNotFound`, write sentinel value `__NOT_FOUND__` to cache with short TTL ($60\text{ seconds}$). Guard public ID lookups with a 100,000-capacity Scalable Bloom Filter ($FPR \le 0.01$).

### 4.4 Cache Breakdown (Hotspot) Defense
- **Mechanism**: XFetch probabilistic early refresh applied for keys where access frequency $> 500\text{ QPS}$.

---

## 5. Invalidation Order & Consistency Invariants
1. **Relational Transaction Commit First**: The database transaction must successfully commit before evicting the cache key.
2. **Key Deletion Over In-Place Overwrites**: Use `DELETE / DEL` instead of `SET` during data updates to prevent stale race overwrites.
3. **Dual-Write Error Policy**: If cache deletion fails after DB commit, publish an asynchronous eviction retry message to the Dead Letter Queue (DLQ).

---

## 6. Circuit Breaking & Fallback Policy
- **Cache Connection Timeout**: $50\text{ ms}$.
- **Circuit Breaker Threshold**: 5 consecutive timeouts or connection errors within $10\text{ s} \implies$ Open circuit for $30\text{ s}$.
- **Fallback Action**: Route queries directly to origin database with an active token bucket rate-limiter ($1{,}000\text{ QPS}$ max) to protect DB from overload.

---

## 7. Verification Evidence & Metrics
- [ ] LRU capacity bounds and eviction verified under continuous $100{,}000$ key insertions.
- [ ] Singleflight coalescing verified under simulated $1{,}000$ concurrent goroutines with 1 origin query.
- [ ] Race detector (`go test -race`) shows zero race conditions across concurrent Get/Put/Delete.

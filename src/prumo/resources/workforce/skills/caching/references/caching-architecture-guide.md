# Caching Architecture, Eviction Algorithms & Resilience Reference Guide

## 1. Mathematical Foundations & Performance Modeling

### 1.1 Effective Memory Latency & Hit Ratio
The performance gain of a caching hierarchy depends strictly on the Hit Ratio $H$:
$$H = \frac{N_{\text{hits}}}{N_{\text{hits}} + N_{\text{misses}}}, \quad 0 \le H \le 1$$
The effective average latency $T_{\text{eff}}$ observed by the client is:
$$T_{\text{eff}} = H \cdot T_{\text{cache}} + (1 - H) \cdot (T_{\text{cache}} + T_{\text{origin}})$$
Where:
- $T_{\text{cache}}$: Time to query the cache (e.g., $10\,\mu\text{s}$ for local L1 heap, $1\,\text{ms}$ for Redis L2).
- $T_{\text{origin}}$: Time to query the underlying database or upstream microservice (e.g., $50\,\text{ms}$).

Even a small degradation in $H$ dramatically increases database load:
$$\text{Database Load Multiplier} = \frac{1 - H_{\text{degraded}}}{1 - H_{\text{nominal}}}$$
If $H$ drops from $99\%$ to $95\%$, the origin database receives $\frac{0.05}{0.01} = 5\times$ (500%) more queries, which frequently triggers cascading database outages.

### 1.2 Zipfian Distribution of Access
Real-world data access follows a power-law / Zipfian distribution:
$$P(k) = \frac{1 / k^s}{\sum_{n=1}^N (1 / n^s)}$$
where $k$ is the popularity rank of an entity and $s \approx 0.8 - 1.2$.
According to the 80/20 rule, caching the top $20\%$ of items satisfies $\sim 80\%$ of read requests.

### 1.3 Probabilistic Early Refresh: The XFetch Algorithm
To prevent **Cache Breakdown** (when a high-frequency hot key expires, causing thousands of concurrent requests to hit the database simultaneously), systems use the **XFetch** algorithm (Vattani et al.):
$$\Delta \cdot \beta \cdot (-\ln(r)) > \text{TTL}_{\text{remaining}}$$
Where:
- $\Delta$: Time in seconds required to recompute/fetch the value from the origin.
- $\beta > 0$: Aggressiveness parameter (typically set to $1.0$).
- $r \sim \text{Uniform}(0, 1)$: A random floating-point number.
- $\text{TTL}_{\text{remaining}}$: Seconds remaining until the cached object expires.

As $\text{TTL}_{\text{remaining}}$ approaches 0, the probability of early background refresh approaches 1. Exactly one worker asynchronously re-warms the key while other clients continue receiving the current cached item without latency spikes.

---

## 2. Eviction Algorithms Comparison

| Algorithm | Time Complexity (Get/Put) | Space Overhead | Resistance to Scan Bursts | Recommended Use Case |
|---|---|---|---|---|
| **LRU** (Least Recently Used) | $O(1) / O(1)$ | 2 pointers per node | Poor (sequential table scans flush entire cache) | General-purpose, low-memory footprints |
| **LFU** (Least Frequently Used) | $O(1) / O(1)$ | Frequency buckets + nodes | High (preserves hot keys) | Read-heavy static references, stable workloads |
| **ARC** (Adaptive Replacement Cache) | $O(1) / O(1)$ | Double cache size (ghost lists) | High (self-tuning between recency and frequency) | File systems, database page buffers |
| **W-TinyLFU** | $O(1) / O(1)$ | Count-Min sketch (4-bit counters) | Exceptional (Count-Min sketch filters one-hit wonders) | High-throughput web applications (Caffeine, Ristretto) |

### 2.3 LRU Implementation Mechanics
A thread-safe $O(1)$ LRU consists of:
1. **Hash Table**: Maps key `K` to node pointer `*Node`.
2. **Doubly Linked List**: Maintains chronological order of accesses.
   - `Head`: Most Recently Used (MRU).
   - `Tail`: Least Recently Used (LRU).
- On `Get(K)`: Look up node in map. Move node to `Head`. Return value.
- On `Put(K, V)`: If key exists, update value and move node to `Head`. If key is new:
  - If `len >= Capacity`: Evict `Tail` node, delete key from map.
  - Insert new node at `Head`.

---

## 3. The 4 Systemic Cache Failure Modes & Remedies

```
+--------------------------------------------------------------------------------+
|                        CACHE HAZARDS & DEFENSIVE PATTERNS                      |
|                                                                                |
| 1. THUNDERING HERD (STAMPEDE) ----> [ SingleFlight Request Coalescing ]        |
|    Concurrent misses on single key     Only 1 query goes to origin DB          |
|                                                                                |
| 2. CACHE AVALANCHE --------------> [ TTL Jitter + Staggered Warming ]          |
|    Mass simultaneous expiration        TTL = BaseTTL +/- Rand(0, Jitter)       |
|                                                                                |
| 3. CACHE PENETRATION ------------> [ Bloom Filter / Negative Caching ]         |
|    Querying non-existent keys          Fast reject before reaching origin DB   |
|                                                                                |
| 4. CACHE BREAKDOWN --------------> [ XFetch Early Probabilistic Refresh ]      |
|    Hot key sudden expiration           Background re-warm before TTL expires   |
+--------------------------------------------------------------------------------+
```

### 3.1 SingleFlight Coalescing
When $1{,}000$ concurrent goroutines request key `"user:100"` which is currently cold:
- Without SingleFlight: All $1{,}000$ requests miss cache simultaneously and execute $1{,}000$ expensive SQL queries against the DB.
- With SingleFlight (`singleflight.Group`): The first request launches the fetcher function. The other $999$ requests detect an ongoing flight for key `"user:100"`, register as listeners on the pending channel, and await the single result. The DB handles exactly $1$ query.

---

## 4. Multi-Tiered Caching Topology (L1 + L2)

```
[ Client Request ]
       |
       v
+-----------------------------+
| L1 Cache: In-Process Heap   |  (Latency: < 1 µs, Zero Serialization)
+-----------------------------+
       | Miss
       v
+-----------------------------+
| L2 Cache: Redis / Memcached |  (Latency: < 1 ms, Shared Cluster State)
+-----------------------------+
       | Miss
       v
+-----------------------------+
| Origin: Relational DB / API |  (Latency: 10 - 100 ms)
+-----------------------------+
```

### Invalidation Ordering Rule
In a Cache-Aside architecture, always invalidate cache keys **after** the database commit:
```go
// Correct Invalidation Sequence:
tx := db.Begin()
tx.Exec("UPDATE users SET name = ? WHERE id = ?", newName, id)
if err := tx.Commit(); err != nil {
    return err
}
// Evict after successful commit:
cache.Delete(ctx, fmt.Sprintf("user:%d", id))
```
*Why?* If the cache is invalidated *before* the transaction commits, a concurrent read will query the DB, read the old uncommitted state, populate the cache with stale data, and overwrite the invalidation indefinitely!

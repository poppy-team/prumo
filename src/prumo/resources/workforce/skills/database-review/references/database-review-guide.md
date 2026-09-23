# Database & Schema Review Reference Guide

## 1. Core Concepts

### 1.1 Reading EXPLAIN ANALYZE
EXPLAIN ANALYZE BUFFERS executes the query and reports actual time, rows, loops, and buffer hits versus
disk reads. Key math: total cost is planner fiction, actual time in milliseconds is ground truth. A nested
loop joining 1 M outer rows with an inner index scan executes the inner side 1 M times; prefer hash joins
for large equijoins and verify with Buffers: shared hit ratios above 99 percent on hot queries. Estimates
off by more than 10x demand ANALYZE or rewritten predicates (functions on indexed columns defeat B-tree use).

### 1.2 B-Tree Index Design
B-tree indexes serve equality and range predicates in O(log n). Order composite columns by selectivity and
query pattern: (tenant_id, created_at) serves WHERE tenant_id = 42 ORDER BY created_at far better than the
reverse. Partial indexes (WHERE status = 'pending') shrink hot working sets by 90 percent or more. Covering
indexes with INCLUDE columns turn index scans into index-only scans, eliminating heap fetches. GIN indexes
serve JSONB containment and full-text search; pg_trgm GIN/GiST indexes serve LIKE '%term%' patterns.

### 1.3 Transaction Isolation Levels
READ COMMITTED (PostgreSQL default) takes a new snapshot per statement: lost updates are possible without
SELECT FOR UPDATE. REPEATABLE READ gives a statement-stable snapshot and raises serialization failures on
concurrent writes instead of silently winning. SERIALIZABLE adds predicate locking for ledger-grade safety
at measurable throughput cost. Cost model: stronger isolation trades throughput for correctness; choose the
weakest level that keeps the invariant, and enforce balance mutations with explicit row locks plus retries.

### 1.4 Constraint-Driven Integrity
Databases outlive applications, so invariants must live in DDL: PRIMARY KEY, FOREIGN KEY with explicit
ON DELETE policy, UNIQUE on natural keys, NOT NULL on required columns, CHECK (balance_cents >= 0).
Deferrable constraints allow bulk loads to validate at COMMIT. Application validation is UX; database
constraints are truth. Reviews reject any design where two concurrent app servers could jointly violate an
invariant the database does not enforce.

### 1.5 Zero-Downtime Migration Patterns
Expand-contract: add nullable column, backfill in batches, add constraint with NOT VALID then VALIDATE,
cut reads over, drop old column. Never combine rename plus type change in one deploy. CREATE INDEX
CONCURRENTLY avoids ACCESS EXCLUSIVE locks but cannot run inside a transaction block. Measure pg_locks
wait events during staging runs; any lock queue above 2 s on the orders table blocks the release.

## 2. Patterns and Anti-Patterns

| Area | Pattern (do) | Anti-Pattern (avoid) |
|---|---|---|
| Backfill | Batches of 1,000–5,000 with sleeps | Single UPDATE over 40 M rows holding locks for minutes |
| Indexes | Composite (tenant_id, created_at), partial on hot subset | Single-column index per column, 14 indexes on one table |
| Queries | Cursor pagination on indexed key | OFFSET 500000 on the hot listing endpoint |
| Writes | 500 ms transactions, 3 s lock_timeout | Open transaction across Stripe API call |
| Types | timestamptz everywhere, numeric for money | timestamp without zone, float for currency |
| Deletes | Soft-delete plus partitioned archival | Hard DELETE cascading 9 tables in request path |
| Evidence | EXPLAIN ANALYZE on production volume | It works on my empty laptop database |

## 3. Code Example: Safe Batched Backfill Migration

```sql
-- Migration 0047: backfill orders.search_vector in batches, then index concurrently.
-- Step 1 (transactional): add nullable column
ALTER TABLE orders ADD COLUMN IF NOT EXISTS search_vector tsvector;

-- Step 2 (batched, outside long transaction; repeat until 0 rows affected)
UPDATE orders
SET search_vector = to_tsvector('portuguese', coalesce(customer_name, '') || ' ' || coalesce(notes, ''))
WHERE id IN (
  SELECT id FROM orders
  WHERE search_vector IS NULL
  ORDER BY id
  LIMIT 2000
);

-- Step 3 (concurrent, cannot run in transaction block)
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_orders_search
  ON orders USING GIN (search_vector);

-- Rollback: DROP INDEX CONCURRENTLY IF EXISTS idx_orders_search;
--           ALTER TABLE orders DROP COLUMN IF EXISTS search_vector;
```

Batches of 2,000 rows keep each write under 300 ms, the concurrent GIN build avoids blocking order
inserts, and the rollback path drops artifacts in dependency order.

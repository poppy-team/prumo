# Database & Schema Review — Verification Checklist

## 1. Migration Safety & Reversibility
- [ ] Every migration ships a tested down path or a documented forward-only repair with equivalent effect.
- [ ] DDL runs inside explicit transactions; multi-statement migrations are split so a mid-flight failure leaves a known state.
- [ ] Large-table rewrites use batched backfills of 1,000–5,000 rows with CONCURRENTLY index builds, never bare locking rewrites.
- [ ] Migration and rollback both complete on a production-sized staging clone, rollback in under 60 s.

## 2. Indexes & Query Plans
- [ ] EXPLAIN ANALYZE BUFFERS plans are attached for every affected query, run against production volume.
- [ ] Hot-path queries (above 100 QPM) use index-only or index scans; no sequential scan over 100 k rows remains.
- [ ] Composite index column order follows predicate selectivity; hot subsets use partial indexes.
- [ ] Row-count estimates in plans fall within one order of magnitude of actual rows; stale statistics trigger ANALYZE.

## 3. Constraints & Data Integrity
- [ ] Foreign keys exist for every cross-table reference with explicit ON DELETE behavior (RESTRICT, CASCADE, or SET NULL chosen deliberately).
- [ ] Uniqueness, NOT NULL, and check constraints are enforced in DDL, including email format and non-negative balance rules.
- [ ] Enum or lookup-table values are versioned; application-only validation without a database constraint is rejected.
- [ ] Sensitive columns (CPF, card tokens) use column-level grants and pgcrypto encryption, never plaintext storage.

## 4. Transactions & Concurrency
- [ ] Transactions complete in under 500 ms; lock_timeout is 3 s and statement_timeout is 10 s on OLTP roles.
- [ ] Balance-style mutations use SELECT FOR UPDATE with retry on serialization failures capped at 3 attempts.
- [ ] No transaction holds locks across network calls, job queues, or user interaction.
- [ ] Connection pool sizing follows (2 x CPU cores) + spindle count with PgBouncer in transaction mode where applicable.

## 5. Operations & Evidence
- [ ] pg_stat_statements deltas prove p95 query time within 10 percent of baseline after the change.
- [ ] Backup RPO under 15 minutes and restore drill completed within the last 90 days for the affected cluster.
- [ ] scripts/verify.sh executes with exit code 0.

# Database Migration Review Specification Template

## 1. Change Overview
- Migration: 0048_add_orders_idempotency_key, author Marina Duarte, reviewed 2026-09-20
- Database: PostgreSQL 16.4 primary with 1 standby, orders table 38.5 M rows growing 400 k per day
- Goal: add idempotency_key citext column with per-tenant uniqueness to support the backend-api replay guarantee
- Window: online migration, zero downtime, rollback budget under 60 s

## 2. DDL Under Review
- ALTER TABLE orders ADD COLUMN idempotency_key citext NULL, backfilled NULL for historical rows
- CREATE UNIQUE INDEX CONCURRENTLY uq_orders_tenant_idem ON orders (tenant_id, idempotency_key) WHERE idempotency_key IS NOT NULL
- ALTER TABLE orders ADD CONSTRAINT chk_orders_total CHECK (total_cents >= 0) NOT VALID, then VALIDATE CONSTRAINT after backfill audit
- Timeouts for migration role: lock_timeout 3 s, statement_timeout 600 s for the concurrent build only

## 3. Plan & Metrics Evidence
- EXPLAIN ANALYZE BUFFERS on lookup by (tenant_id, idempotency_key): index-only scan, 0.4 ms, buffers hit 4
- pg_stat_statements: orders lookup p95 1.8 ms baseline, 1.9 ms after index (+5.5 percent, within 10 percent gate)
- Staging clone 40 M rows: concurrent build 11 min 20 s, zero lock waits above 1 s in pg_locks, rollback 34 s
- Slow query log: zero new entries above 1 s during soak; autovacuum caught up in 6 minutes

## 4. Rollback & Operations
- Rollback order: drop constraint chk_orders_total, drop index uq_orders_tenant_idem concurrently, drop column idempotency_key
- Forward-only alternative accepted if backfill passes 50 percent: complete backfill instead of reverting
- Backup: base backup 2026-09-19 03:00 UTC plus WAL archiving, RPO 5 minutes, restore drill 2026-08-02 succeeded in 22 minutes
- verify.sh: exit code 0 on 2026-09-23 run by data-platform pipeline job 1190

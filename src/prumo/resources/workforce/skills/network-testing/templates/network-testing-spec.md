# Network Resilience Test Specification Template

## 1. System & Objectives
- System: OrdersAPI sync client plus Starfall Arena session service, owner connectivity guild, contact net@acme.example
- RTO: reconnect plus catch-up within 5 minutes under hotel Wi-Fi; duplicate effect rate under 0.01 percent
- Harness: Toxiproxy 2.9 in CI, tc netem on bare-metal soak runners, seeds recorded per run
- Baseline: clean LAN p99 API latency 120 ms, session tick delivery 99.99 percent, zero backlog at rest

## 2. Fault Profiles Applied
- Hotel Wi-Fi: 60 ms + 40 ms jitter, 2 percent loss, drop every 90 s, seed 7741
- 4G handoff: 300 ms spike for 5 s then 10 s partition, seed 7742
- Region partition: full gru1/mia1 split 60 s asymmetric return, seed 7743
- Reconnect storm: 500 clients dropped simultaneously, staged backoff with full jitter

## 3. Retry & Queue Configuration
- Orders client: 5 attempts, base 200 ms, cap 10 s, full jitter, Idempotency-Key on all POST retries
- Session service: heartbeat 2 s, detection after 3 misses (6 s), reconnect p95 measured 4.1 s
- Offline queue capped at 10,000 ops with drop-oldest-plus-warning; drain rate 5,000 ops per minute
- Timeouts: client 8 s, server 5 s, payment downstream 2 s, hierarchy verified in config audit

## 4. Recovery Evidence
- Hotel Wi-Fi soak 60 min: zero data loss, duplicate effect rate 0.004 percent, frozen UI p95 3.2 s
- Region partition 60 s: both sides queued, converged in 22 s after heal, RTO 5 min met with margin
- Reconnect storm: peak accept 1.6x provisioned, no cascading 503, backlog drained in 2.4 min
- verify.sh: exit code 0 on 2026-09-23 run by connectivity pipeline job 145

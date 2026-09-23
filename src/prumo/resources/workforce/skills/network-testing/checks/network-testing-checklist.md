# Network Resilience Testing — Verification Checklist

## 1. Fault Profiles & Injection
- [ ] Profiles defined: clean LAN, hotel Wi-Fi (60 ms + 40 ms jitter, 2 percent loss), 4G handoff spike, 60 s region partition.
- [ ] Faults injected via Toxiproxy toxics or tc netem qdisc with committed seeds for deterministic CI replay.
- [ ] Packet loss, latency, jitter, duplication, corruption, and full partition are each covered by at least one scenario.
- [ ] Staging hop count, TLS termination, and NAT behavior match production within one documented delta.

## 2. Retries, Backoff & Timeouts
- [ ] Every dependency declares attempts (max 5), base 200 ms, cap 10 s, full-jitter exponential backoff.
- [ ] POST retries carry Idempotency-Key; unsafe replays without keys are absent from the codebase.
- [ ] Timeout hierarchy holds: client > server > downstream, each with explicit margin and logged expiry.
- [ ] Retry storms measured: 500-client reconnect stays under 2x provisioned accept rate via staged backoff.

## 3. Partitions, Queues & Dedup
- [ ] 60 s split-brain test shows both sides queueing with zero corruption and convergence within 30 s RTO.
- [ ] Offline queues bounded at 10,000 operations with drop-oldest-plus-warning and a visible drain gauge.
- [ ] Duplicate rate after dedup IDs stays under 0.01 percent across loss and redelivery scenarios.
- [ ] Circuit breakers open after 5 consecutive failures and half-open probe before full restore.

## 4. Recovery Evidence
- [ ] Detection time (3 missed 2 s heartbeats), reconnect p95, and backlog drain rate (5,000 ops/min) measured.
- [ ] User-visible frozen time reported per scenario with a 5 s budget on the critical path.
- [ ] Recovery dashboard or log export attached to the release with seed numbers.
- [ ] scripts/verify.sh executes with exit code 0.

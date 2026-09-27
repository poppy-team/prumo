# Network Resilience Testing Reference Guide

## 1. Core Concepts

### 1.1 The Eight Fallacies as Test Plan
The network is not reliable, latency is not zero, bandwidth is not infinite, the topology changes, and
the transport cost is not zero. Each fallacy becomes a scenario: kill packets (loss 2 percent), add delay
spikes (300 ms for 5 s), halve bandwidth mid-transfer, repartition the region, and bill the retry storm.
A system tested only on clean LAN encodes all eight fallacies as production incidents.

### 1.2 Fault Injection with netem and Toxiproxy
Linux tc netem shapes real packets: `loss 2%`, `delay 60ms 40ms distribution normal`, `duplicate 1%`,
`corrupt 0.5%` on the egress qdisc reproduce hotel Wi-Fi faithfully. Toxiproxy works at TCP level with
named toxics (latency, bandwidth, slow_close, timeout, slicer) toggled per test over HTTP, ideal for CI
where root qdisc access is unavailable. Rule: netem for packet truth on dedicated runners, Toxiproxy for
portable deterministic CI; both seeded, both committed, never ad-hoc manual chaos.

### 1.3 Exponential Backoff with Full Jitter
Without jitter, 500 clients retrying every 2 s arrive as a synchronized hammer. Full jitter spreads them:
sleep = random_between(0, min(cap, base × 2^attempt)) with base 200 ms and cap 10 s gives attempt sleeps
bounded by 0.2 s, 0.4 s, 0.8 s, 1.6 s, 3.2 s windows. Expected wait stays under 7 s total while the thunder
dissolves into uniform noise. Decorrelated jitter (sleep = min(cap, random(base, sleep × 3))) converges
faster under persistent failure and is preferred for reconnect storms.

### 1.4 Idempotency, Dedup, and At-Least-Once Delivery
Networks deliver at-least-once: anything unacknowledged may be redelivered. Safety comes from receivers,
not wishes: Idempotency-Key headers with 24-hour stores for APIs, dedup IDs (wh_7d02) with exactly-once
effect for webhooks, idempotent consumers (set-add, upsert-by-key) for queues. Test the guarantee
directly: deliver every message twice under 2 percent loss and assert downstream effects occur exactly once.

### 1.5 Recovery Arithmetic and RTO Budgeting
Recovery time = detection + reconnect + catch-up. Detection is 3 missed heartbeats at 2 s intervals (6 s).
Reconnect p95 under 5 s on hotel Wi-Fi with staged backoff. Catch-up is backlog divided by drain rate:
12,000 queued ops at 5,000 ops/min needs 2.4 minutes. RTO of 5 minutes therefore demands detection plus
reconnect under 2.6 minutes plus bounded queues; unbounded queues make every RTO a fiction.

## 2. Patterns and Anti-Patterns

| Area | Pattern (do) | Anti-Pattern (avoid) |
|---|---|---|
| Retries | 5 attempts, 200 ms base, 10 s cap, full jitter | Infinite immediate retry hot-looping a dead peer |
| Keys | Idempotency-Key on every POST retry | Blind POST replay double-charging cards |
| Timeouts | Client 8 s > server 5 s > downstream 2 s | 30 s client hanging on 60 s server fantasy |
| Queues | Bounded 10 k with drain gauge and warning | Unbounded in-memory list OOMing the phone |
| Breakers | Open after 5 failures, half-open probe | Hammering a down region for 40 minutes |
| Storms | Staged reconnect backoff with jitter | 500 clients reconnecting on the same second |
| Evidence | Seeded scripts plus RTO dashboard | Worked once on my machine, no logs |

## 3. Code Example: Toxiproxy Hotel Wi-Fi Scenario with Jittered Retry

```json
{
  "name": "hotel-wifi-profile",
  "listen": "0.0.0.0:8474",
  "upstream": "orders-api.internal:443",
  "toxics": [
    { "name": "base-latency", "type": "latency", "attributes": { "latency": 60, "jitter": 40 } },
    { "name": "lossy-link", "type": "loss", "attributes": { "probability": 0.02 } },
    { "name": "slow-drain", "type": "bandwidth", "attributes": { "rate": 512 } }
  ]
}
```

```python
import random, time

def call_with_backoff(fn, attempts=5, base=0.2, cap=10.0):
    for n in range(attempts):
        try:
            return fn()
        except TransientError:
            if n == attempts - 1:
                raise
            ceiling = min(cap, base * (2 ** n))
            time.sleep(random.uniform(0, ceiling))
```

The proxy reproduces 60 ms latency with 40 ms jitter, 2 percent loss, and a 512 KB/s ceiling, while the
caller spreads retries across full-jitter windows so the reconnect storm dissolves instead of amplifying.

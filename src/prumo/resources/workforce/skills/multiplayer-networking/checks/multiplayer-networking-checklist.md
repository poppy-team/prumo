# Multiplayer Networking — Verification Checklist

## 1. Server Authority & Cheat Resistance
- [ ] Server is the sole writer of gameplay state; clients transmit only timestamped input commands.
- [ ] All client-originated state writes are rejected server-side and logged as cheat signals.
- [ ] Sensitive rolls (damage, loot, spawn) execute server-side only; seeds never leave the server.
- [ ] Packet validation rejects malformed datagrams; senders exceeding 120 packets/s are throttled then disconnected.
- [ ] Traffic is encrypted (DTLS 1.3 or equivalent) with per-session keys and replay-protection windows.

## 2. Prediction, Reconciliation & Interpolation
- [ ] Local input applies to the predicted pawn within one client frame; inputs buffer in a ring of at least 64 entries.
- [ ] Reconciliation rewinds to the ACKed tick and re-simulates on errors above 8 cm; smaller errors smooth-damp within 150 ms.
- [ ] Remote entities render via two-snapshot interpolation at 80-120 ms delay; jitter buffer holds at 5% packet loss.
- [ ] No visible rubber-banding in 10-minute soak at 80 ms RTT; reconciliation events logged with tick and error magnitude.

## 3. Bandwidth, Packets & Delta Compression
- [ ] Only fields changed since the last ACKed snapshot replicate (delta bitmask verified in packet captures).
- [ ] Transforms quantized (position 1 mm fixed point, yaw 12-bit); mean packet size recorded per frequency class.
- [ ] Zero packets exceed 1200 bytes in soak captures; oversized snapshots split across ticks by priority.
- [ ] Mean downstream per client stays within budget (64 kbps broadband floor) with 8 players active.
- [ ] Relevancy culling enforced (40 m radius, 12-entity cap) with graceful degradation order documented.

## 4. Lag Compensation & Hit Registration
- [ ] Server keeps a 1-second position history buffer per entity for rewind queries.
- [ ] Hitscan tests rewind hitboxes to shooter time (`server_time - RTT/2`) before capsule tests.
- [ ] Shooter-vs-server hit agreement is at or above 98% below 120 ms RTT, reported per weapon class.
- [ ] Melee and projectile paths use the same rewind clock; no weapon bypasses compensation silently.

## 5. Reconnection & Operational Evidence
- [ ] Drops up to 30 s resume from full-state baseline plus delta stream with no scene reload.
- [ ] Session migration (host handoff or relay failover) preserves entity IDs and input sequence numbers.
- [ ] Five-drop reconnect drill logged with resume times; p95 resume under 4 s.
- [ ] Automated verification script `scripts/verify.sh` executes with exit code 0.

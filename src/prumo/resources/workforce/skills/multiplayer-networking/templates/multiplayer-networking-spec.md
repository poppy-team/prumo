# Multiplayer Networking Specification — Harbor Siege Arena (8-Player Skirmish)

## 1. Topology and Tick Contract
- **Authority model**: Dedicated authoritative server (Linux, 4 vCPU), clients are input terminals with local prediction.
- **Tick rates**: Server simulation 20 Hz (50 ms fixed step); client render 60 Hz; snapshot broadcast 20 Hz for pawns, 10 Hz for projectiles, on-change for pickups.
- **Transport**: UDP with custom reliability (seq + 32-bit ACK bitfield), DTLS 1.3 sessions, per-client rate limit 120 packets/s.
- **Test date and owner**: 2026-09-12, netcode pod (Ilya Sorin), engine custom Rust server plus Godot 4.3 client.

## 2. Entity Replication Classes

| Class | Examples | Frequency | Quantization | Priority |
|---|---|---|---|---|
| Pawn | 8 player captains | 20 Hz | pos 1 mm fixed, yaw 12-bit | 1, never deferred |
| Projectile | cannonballs, max 24 live | 10 Hz | pos 1 cm fixed, vel 10-bit | 2 |
| Pickup | 14 crates, 6 repair kits | on-change | id + state byte | 3, deferred first |

## 3. Bandwidth Budget (Per Client)

| Item | Value |
|---|---|
| Header per snapshot | 28 bytes (seq, ack, bitfield, timestamp) |
| Mean snapshot payload | 216 bytes (12 relevant entities typical) |
| Mean downstream | 20 Hz * 244 B = 39 kbps against 64 kbps ceiling |
| p99 packet observed in soak | 812 bytes, zero packets above 1200 bytes |
| Relevancy | 40 m radius, 12-entity cap, line-of-sight required for class 3 |

## 4. Prediction, Interpolation, and Compensation Tuning

| Mechanism | Setting | Measured result |
|---|---|---|
| Prediction ring | 64 inputs, 8 cm snap gate | 3.1 reconciliations/min at 80 ms RTT |
| Interpolation delay | 100 ms render offset | smooth at 5% loss, no underflow |
| Rewind history | 20 entries (1 s at 20 Hz) | hit agreement 98.6% below 120 ms RTT |
| Melee window | 200 ms acceptance | agreement 97.9% in dock map |

## 5. Reconnect Contract
- Drops up to 30 s resume from full-state baseline plus delta stream; entity IDs and input seq preserved.
- Drill of 2026-09-12: 5 drops (12 s, 18 s, 24 s, 29 s, 30 s), p95 resume 3.4 s, zero scene reloads.
- Beyond 30 s the session issues a fresh baseline and the client shows a 2-second "resyncing" veil.

## 6. Regression Evidence
- [x] 10-minute 8-player soak: 20 Hz tick held, mean packet 244 B, zero packets above 1200 B.
- [x] RTT matrix (80/150/250 ms): prediction error histograms and reconciliation counts archived.
- [x] Hitscan agreement 98.6% below 120 ms RTT across 2,400 fired rounds.
- [x] `scripts/verify.sh` exits 0 on the skill package; netcode soak gate green on server commit a71f09.

# Realtime Synchronization Specification Template

## 1. Game & Topology
- Game: Starfall Arena 3v3 arena battler, authoritative dedicated servers in São Paulo (gru1) plus Miami (mia1)
- Model: server-authoritative snapshot interpolation, 20 Hz ticks, 20 Hz delta snapshots, full snapshot every 30 ticks
- Targets: RTT under 80 ms p75, jitter under 15 ms, loss under 1 percent, 6 players plus 4 spectators per room
- Clocks: NTP discipline with monotonic tick accumulator, drift alarm at 50 ms skew

## 2. Simulation & Bandwidth Budget
- Tick rate 20 Hz (50 ms), input packets 24 bytes with triple redundancy, upstream 4 kbps per player
- Snapshot deltas average 400 bytes at 20 Hz, downstream 64 kbps cap per client, network LOD beyond 60 m halves rate
- Positions quantized to 12-bit grid cells, health sent on change only, state hash FNV-1a per tick logged
- Interpolation delay 100 ms, jitter buffer 100 ms, extrapolation hard-capped at 250 ms with freeze plus icon

## 3. Reconnect & Fairness
- Reconnect grace 10 s with snapshot plus input-log catch-up; late join downloads full state (9 KB) then deltas
- Host migration not applicable (dedicated servers); server failover reassigns room in under 5 s with score preserved
- Anti-cheat: server decides hits, deaths, pickups; client damage reports ignored; speed-hack detection at 15 percent tick-rate deviation

## 4. Test Evidence
- Soak: 6 players plus 20 bots across 4 rooms, 30 minutes at 80 ms RTT, 15 ms jitter, 1 percent loss via netem, zero desyncs
- Mispredict analogue (reconciliation snaps): 2.1 percent of snapshots, max correction 1.4 m, no visible teleport complaints
- Bandwidth p99 58 kbps downstream, snapshot p99 460 bytes, tick hash agreement 100 percent across 36,000 ticks
- verify.sh: exit code 0 on 2026-09-23 run by netcode pipeline job 512

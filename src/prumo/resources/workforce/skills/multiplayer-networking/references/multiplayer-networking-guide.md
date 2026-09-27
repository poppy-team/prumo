# Multiplayer Networking Reference Guide

## 1. Core Concepts

### 1.1 Server Authority and Tick Pipeline
The server owns truth. One canonical loop runs at fixed tick `dt = 50 ms` (20 Hz): collect validated inputs, simulate physics and gameplay, snapshot entity state, broadcast. Clients are dumb terminals with a prediction fast-path. Any design letting clients dictate positions is a speed-hack waiting for release day.

### 1.2 Client Prediction and Server Reconciliation
Round-trip latency (80-250 ms) would make movement feel drunk without prediction. The client applies its own input instantly and stores `(seq, input)` pairs in a ring buffer. Each server snapshot carries the last processed `seq`; the client discards ACKed inputs, compares authoritative vs predicted position, and re-simulates the remainder when divergence exceeds the 8 cm gate. Bandwidth cost: zero extra bytes beyond sequence numbers.

### 1.3 Delta Compression and Quantization
Full-state snapshots scale as `O(entities x fields)`; deltas scale as `O(changed fields)`. Each snapshot carries a per-entity changed-bitmask against the client's ACKed baseline, followed by only changed values. Quantization compounds the win: 32-bit float position becomes 24-bit fixed point at 1 mm resolution over a 16 km map; yaw becomes 12 bits (0.09 degree steps). Combined savings on a quiet scene routinely exceed 10:1.

### 1.4 Bandwidth Budget Model
Per-client downstream follows:
```
BW = tick_rate * (header + entities_relevant * fields_changed_avg * bytes_per_field)
```
Example: 20 Hz * (28 B header + 12 entities * 6 fields * 3 B) = 20 * 244 B = 4.88 kB/s ≈ 39 kbps, inside a 64 kbps floor budget. Relevancy radius and entity caps are the control knobs.

### 1.5 Snapshot Interpolation vs Extrapolation
Interpolation renders the past (`now - 100 ms`) by lerping between two buffered snapshots: smooth but adds latency. Extrapolation renders the predicted present: responsive but explodes under loss. Standard practice: interpolate remote entities, predict only the local pawn. Never extrapolate remote pawns beyond 150 ms; freeze and show a connection icon instead.

### 1.6 Lag Compensation by Rewind
A shooter with 120 ms RTT aims at a ghost 60 ms old. The server stores per-entity position history (20 Hz * 1 s = 20 entries) and rewinds hitboxes to `fire_time = server_now - RTT/2` before raycasts. Melee uses the same clock with a wider 200 ms acceptance window.

## 2. Patterns and Anti-Patterns

| Pattern (do) | Anti-pattern (do not) |
|---|---|
| Fixed 1200-byte packet ceiling; priority-split snapshots across ticks | Trust path-MTU discovery and fragment 4 KB snapshots |
| ACK bitfields (last seq + 32-bit mask) drive delta baselines | Resend full state every tick "for simplicity" |
| Interest management by distance + line-of-sight + cap | Replicate the whole 64-player roster to everyone |
| Encrypted session with replay window and rate limits | Raw UDP with plaintext positions and no sequence validation |
| Reconnect via baseline-plus-delta with preserved entity IDs | Full scene reload and new entity IDs on every drop |

## 3. Minimal Example: Delta Snapshot Packing (Rust)

```rust
// Snapshot delta against the client's last ACKed baseline.
// Packet layout: [seq: u16][ack: u16][entity_count: u8][entity deltas...]
// Hard ceiling: PACKET_CEILING = 1200 bytes (safe internet MTU floor).
const PACKET_CEILING: usize = 1200;

pub struct DeltaWriter {
    buf: Vec<u8>,
}

impl DeltaWriter {
    pub fn push_entity(&mut self, id: u16, mask: u8, fields: &[u8]) -> bool {
        // 2 B id + 1 B changed-bitmask + changed field bytes.
        let need = 2 + 1 + fields.len();
        if self.buf.len() + need > PACKET_CEILING {
            return false; // Defer to next tick by priority order.
        }
        self.buf.extend_from_slice(&id.to_le_bytes());
        self.buf.push(mask);
        self.buf.extend_from_slice(fields);
        true
    }
}
```

Callers sort entities by priority (local relevance, then change age), push until `push_entity` returns false, and defer the remainder to the next tick. The soak harness asserts no emitted datagram exceeds `PACKET_CEILING` and reports mean size per frequency class.

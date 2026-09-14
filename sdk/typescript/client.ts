// Prumo Harness client for Node.js (GAP-041 remainder).
// Mirrors sdk/prumo (Go): same ops, same JSONL wire, typed results.
// Unix-socket default; TCP+TLS+token via connectRemote. No dependencies.
import net from "node:net";
import tls from "node:tls";

export interface RunStatus {
  run_id: string;
  status: string;
  phase: string;
  stop_reason: string;
  active: boolean;
}

export interface AgentEvent {
  id: string;
  run_id: string;
  turn_id?: string;
  kind: string;
  payload?: Record<string, unknown>;
  created_at: string;
}

export interface StartRequest {
  goal: string;
  provider?: string;
  model?: string;
  base_url?: string;
  max_turns?: number;
  run_id?: string;
  workspace?: string;
}

export interface RemoteOptions {
  addr: string;
  token: string;
  caCert?: string;
}

type OpEnvelope = Record<string, unknown>;

function splitLines(buf: string): { lines: string[]; rest: string } {
  const parts = buf.split("\n");
  return { lines: parts.slice(0, -1), rest: parts[parts.length - 1] };
}

export class Client {
  private socketPath?: string;
  private remote?: RemoteOptions;
  private timeoutMs: number;

  constructor(socketPath?: string, timeoutMs = 30000) {
    this.socketPath = socketPath;
    this.timeoutMs = timeoutMs;
  }

  static remote(opts: RemoteOptions, timeoutMs = 30000): Client {
    const c = new Client(undefined, timeoutMs);
    c.remote = opts;
    return c;
  }

  private connect(): Promise<net.Socket> {
    return new Promise((resolve, reject) => {
      const onError = (err: Error): void => reject(err);
      if (this.remote) {
        const [host, portRaw] = this.remote.addr.split(":");
        const port = Number(portRaw);
        const sock = tls.connect(
          {
            host,
            port,
            ca: this.remote.caCert,
            minVersion: "TLSv1.2",
            servername: host,
          },
          () => {
            sock.write(JSON.stringify({ auth: this.remote?.token }) + "\n");
            const chunks: string[] = [];
            const onData = (d: Buffer): void => {
              chunks.push(d.toString("utf8"));
              const { lines } = splitLines(chunks.join(""));
              if (lines.length > 0) {
                const first = JSON.parse(lines[0]) as { ok?: boolean; error?: string };
                sock.off("data", onData);
                if (first.ok) {
                  resolve(sock);
                } else {
                  sock.destroy();
                  reject(new Error(`auth: ${String(first.error)}`));
                }
              }
            };
            sock.on("data", onData);
          },
        );
        sock.once("error", onError);
        return;
      }
      if (!this.socketPath) {
        reject(new Error("no socket path or remote configured"));
        return;
      }
      const sock = net.createConnection(this.socketPath);
      sock.once("connect", () => resolve(sock));
      sock.once("error", onError);
    });
  }

  async call(msg: OpEnvelope): Promise<OpEnvelope> {
    const op = String(msg["op"] ?? "?");
    const sock = await this.connect();
    try {
      const reply = await new Promise<OpEnvelope>((resolve, reject) => {
        const timer = setTimeout(() => reject(new Error("daemon timeout")), this.timeoutMs);
        let buf = "";
        sock.on("data", (d: Buffer) => {
          buf += d.toString("utf8");
          const { lines, rest } = splitLines(buf);
          buf = rest;
          if (lines.length > 0) {
            clearTimeout(timer);
            try {
              resolve(JSON.parse(lines[0]) as OpEnvelope);
            } catch (e) {
              reject(e instanceof Error ? e : new Error(String(e)));
            }
          }
        });
        sock.once("error", (e: Error) => {
          clearTimeout(timer);
          reject(e);
        });
        sock.write(JSON.stringify(msg) + "\n");
      });
      if (!reply["ok"]) {
        throw new Error(`daemon op ${op}: ${String(reply["error"])}`);
      }
      return reply;
    } finally {
      sock.destroy();
    }
  }

  async start(r: StartRequest): Promise<string> {
    const res = await this.call({ op: "start", max_turns: 5, ...r });
    return String(res["run_id"]);
  }

  async status(runID: string): Promise<RunStatus> {
    const res = await this.call({ op: "status", run_id: runID });
    return res as unknown as RunStatus;
  }

  async list(): Promise<RunStatus[]> {
    const res = await this.call({ op: "list" });
    return (res["runs"] ?? []) as RunStatus[];
  }

  async events(runID: string): Promise<AgentEvent[]> {
    const res = await this.call({ op: "events", run_id: runID });
    return (res["events"] ?? []) as AgentEvent[];
  }

  async cancel(runID: string): Promise<void> {
    await this.call({ op: "cancel", run_id: runID });
  }

  async steer(runID: string, message: string): Promise<void> {
    await this.call({ op: "steer", run_id: runID, message });
  }

  async protocol(): Promise<Record<string, unknown>> {
    return this.call({ op: "protocol" });
  }

  async wait(runID: string, pollMs = 100): Promise<RunStatus> {
    for (;;) {
      const st = await this.status(runID);
      if (st.status !== "running") {
        return st;
      }
      await new Promise((r) => setTimeout(r, pollMs));
    }
  }
}

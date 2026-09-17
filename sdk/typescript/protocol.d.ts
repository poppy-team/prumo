// Generated from schemas/protocol-manifest.json — DO NOT EDIT.
// Regenerate: go run ./sdk/typescript/gen

export const PROTOCOL_VERSION = "0.3.0";
export const PROTOCOL_MIN_COMPATIBLE = "0.1.0";

export type OpName =
  | "start" |
  | "status" |
  | "list" |
  | "events" |
  | "cancel" |
  | "steer" |
  | "schedule" |
  | "unschedule" |
  | "jobs" |
  | "protocol" |
  | "approve" |
  | "deny" |
  | "models";

export interface OpArgs {
  "start": ["goal"];
  "status": ["run_id"];
  "list": [];
  "events": ["run_id"];
  "cancel": ["run_id"];
  "steer": ["run_id", "message"];
  "schedule": ["goal"];
  "unschedule": ["job_id"];
  "jobs": [];
  "protocol": [];
  "approve": ["run_id", "request_id"];
  "deny": ["run_id", "request_id"];
  "models": [];
}

export interface RunStatus {
  run_id: string;
  status: string;
  phase: string;
  stop_reason: string;
  active: boolean;
  pending_permissions: string[];
}

export interface AgentEvent {
  id: string;
  run_id: string;
  turn_id?: string;
  kind: string;
  payload?: Record<string, unknown>;
  created_at: string;
}

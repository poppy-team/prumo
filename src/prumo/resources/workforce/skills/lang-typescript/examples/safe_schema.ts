/**
 * Modern TypeScript 5.x+ Soundness, Branded Types & Boundary Validation Reference
 * Self-contained: Zero external dependencies, 100% strict type soundness.
 */

// 1. Branded Nominal Types
declare const BrandSymbol: unique symbol;
export type Brand<Base, Tag extends string> = Base & {
  readonly [BrandSymbol]: Tag;
};

export type UserId = Brand<string, "UserId">;
export type ProjectId = Brand<string, "ProjectId">;

export function parseUserId(raw: unknown): UserId {
  if (typeof raw !== "string" || !raw.startsWith("usr_") || raw.length < 8) {
    throw new TypeError(`Invalid UserId format: ${String(raw)}`);
  }
  return raw as UserId;
}

export function parseProjectId(raw: unknown): ProjectId {
  if (typeof raw !== "string" || !raw.startsWith("prj_") || raw.length < 8) {
    throw new TypeError(`Invalid ProjectId format: ${String(raw)}`);
  }
  return raw as ProjectId;
}

// 2. Discriminated Union with Exhaustiveness Invariant
export type JobStatus =
  | { readonly kind: "PENDING"; readonly queuedAt: number }
  | { readonly kind: "RUNNING"; readonly startedAt: number; readonly progress: number }
  | { readonly kind: "COMPLETED"; readonly finishedAt: number; readonly result: string }
  | { readonly kind: "FAILED"; readonly error: string };

export function assertNever(x: never): never {
  throw new Error(`Unexpected variant encountered: ${JSON.stringify(x)}`);
}

export function describeJob(job: JobStatus): string {
  switch (job.kind) {
    case "PENDING":
      return `Job queued at ${new Date(job.queuedAt).toISOString()}`;
    case "RUNNING":
      return `Job running: ${(job.progress * 100).toFixed(1)}% complete`;
    case "COMPLETED":
      return `Job finished: ${job.result}`;
    case "FAILED":
      return `Job failed with error: ${job.error}`;
    default:
      return assertNever(job);
  }
}

// 3. The `satisfies` Operator: Verification without Widening
export interface RouteConfig {
  readonly path: string;
  readonly method: "GET" | "POST" | "PUT" | "DELETE";
  readonly requiresAuth: boolean;
}

export const AppRoutes = {
  health: { path: "/health", method: "GET", requiresAuth: false },
  users: { path: "/api/users", method: "POST", requiresAuth: true },
} as const satisfies Record<string, RouteConfig>;

// 4. Pure Runtime Boundary Validator (Zero-dependency schema validator pattern)
export interface UserRecord {
  readonly id: UserId;
  readonly email: string;
  readonly role: "admin" | "member";
}

export function validateUserRecord(raw: unknown): UserRecord {
  if (typeof raw !== "object" || raw === null) {
    throw new TypeError("Payload must be a non-null object");
  }

  const record = raw as Record<string, unknown>;

  const id = parseUserId(record["id"]);
  const email = record["email"];
  const role = record["role"];

  if (typeof email !== "string" || !email.includes("@")) {
    throw new TypeError("Invalid or missing email address");
  }

  if (role !== "admin" && role !== "member") {
    throw new TypeError(`Invalid role: ${String(role)}`);
  }

  return { id, email, role };
}

// 5. Verification Run
function main(): void {
  console.log("=== TypeScript 5.x+ Strict Soundness Demonstration ===");

  const rawInput = {
    id: "usr_9981240",
    email: "architect@prumo.dev",
    role: "admin",
  };

  const validatedUser = validateUserRecord(rawInput);
  console.log(`Validated User: ${validatedUser.id} (${validatedUser.email}) [${validatedUser.role}]`);

  const job: JobStatus = {
    kind: "RUNNING",
    startedAt: Date.now(),
    progress: 0.75,
  };
  console.log(`Job Status: ${describeJob(job)}`);

  console.log(`Route verified: ${AppRoutes.users.path} [${AppRoutes.users.method}]`);
}

main();

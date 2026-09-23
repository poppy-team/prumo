# TypeScript Soundness, Nominal Types & Boundary Validation Reference

## 1. Type Soundness & The Danger of `any`
TypeScript's design goals prioritize pragmatic developer ergonomics over strict mathematical soundness, meaning potential holes exist (e.g. mutable array covariance, method bivariance). However, adopting strict compiler invariants eliminates accidental runtime type crashes.

### The Top Types: `any` vs `unknown`
```
                    any (Disables Type Checker)
                   /   \
         Top Type /     \ Bottom Type (Can be assigned TO anything!)
                 v       v
         Bypasses all safety checks; infectious across call graphs!

                   unknown (Sound Top Type)
                   /
         Top Type / (Can accept ANY value)
                 v
         Cannot be operated on without explicit type narrowing!
```

---

## 2. Branded Types (Nominal Typing Simulation)
TypeScript's type system is structural, meaning two types with the same shape are identical. To enforce nominal separation for domain IDs:
```typescript
declare const BrandSymbol: unique symbol;

export type Brand<Base, Tag extends string> = Base & {
  readonly [BrandSymbol]: Tag;
};

export type UserId = Brand<string, "UserId">;
export type ProjectId = Brand<string, "ProjectId">;

function deleteUser(id: UserId): void { ... }

const uid = "usr_123" as UserId;
const pid = "prj_456" as ProjectId;

deleteUser(uid); // COMPILES
deleteUser(pid); // COMPILER ERROR: Type '"ProjectId"' is not assignable to type '"UserId"'!
```

---

## 3. Exhaustive Discriminated Unions with `never`
A discriminated union utilizes a shared single-value property (discriminant) to allow TypeScript's control flow analysis to narrow union variants:
```typescript
type Command =
  | { type: "INIT"; path: string }
  | { type: "BUILD"; target: string; release: boolean }
  | { type: "CLEAN" };

function assertNever(val: never): never {
  throw new Error(`Unhandled command variant: ${JSON.stringify(val)}`);
}

function execute(cmd: Command): void {
  switch (cmd.type) {
    case "INIT":  /* cmd is { type: "INIT"; path: string } */ break;
    case "BUILD": /* cmd is { type: "BUILD"; target: string; release: boolean } */ break;
    case "CLEAN": /* cmd is { type: "CLEAN" } */ break;
    default:
      // If a new Command variant is added and not handled, this line produces a compile error:
      assertNever(cmd);
  }
}
```

---

## 4. The `satisfies` Operator vs Type Annotations
When annotating a variable (`const x: Type = ...`), TypeScript widens the inferred type to `Type`.
With `satisfies` (TypeScript 4.9+), TypeScript validates that the expression conforms to the type **without widening**:
```typescript
type Palette = Record<string, string | [number, number, number]>;

// Using satisfies:
const colors = {
  primary: "#1e88e5",
  secondary: [255, 0, 128],
} satisfies Palette;

// Retains exact tuple type:
colors.secondary[0]; // Number (TS knows this is a tuple, not string | [number, number, number])
```

---

## 5. Runtime Validation: Deriving Types from Schemas
In TypeScript, compile-time types disappear at runtime. Schemas act as single sources of truth that bridge compile-time checking with runtime guarantee:
```typescript
import { z } from "zod";

export const ServerConfigSchema = z.object({
  port: z.number().int().min(1024).max(65535).default(8080),
  host: z.string().ip().default("127.0.0.1"),
  corsOrigins: z.array(z.string().url()).default([]),
});

export type ServerConfig = z.infer<typeof ServerConfigSchema>;
```
Parsing untrusted configuration with `ServerConfigSchema.parse(rawEnv)` guarantees that the object conforms to `ServerConfig` with zero manual assertions (`as ServerConfig`).

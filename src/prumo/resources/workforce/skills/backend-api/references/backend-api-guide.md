# Backend API Design Reference Guide

## 1. Core Concepts

### 1.1 Resources, Methods & Safety
Model endpoints as resources with nouns, never verbs: POST /v1/orders creates, GET /v1/orders lists,
GET /v1/orders/ord_9f31 retrieves. Safe methods (GET, HEAD) never mutate; idempotent methods
(GET, PUT, DELETE) produce the same server state on replay. POST is neither safe nor idempotent,
which is why mutation endpoints require explicit Idempotency-Key handling.

### 1.2 Cursor Pagination Model
Offset pagination (`?offset=10000&limit=25`) forces the database to scan and discard 10,000 rows and
drifts when rows are inserted concurrently. Cursor pagination encodes the last seen sort key in an
opaque token (`?cursor=eyJsYXN0X2lkIjo5MzEwLCJsYXN0X3RzIjoxNzI5NTk0NjAwfQ&limit=25`), giving stable
O(log n) index seeks. Rule: default limit 25, max 100, cursors opaque and signed, total counts served
from a separate cached estimate, never from COUNT(*) on the hot path.

### 1.3 Token-Bucket Rate Limiting
Each client holds a bucket of capacity B tokens refilled at rate R per second. A request consumes one
token; empty buckets yield 429 with Retry-After. Example: R = 10 req/s sustained, B = 120 burst absorbs
traffic spikes without starving steady clients. Apply per API key plus per IP fallback, and exempt
health checks. Communicate limits via RateLimit-Limit, RateLimit-Remaining, and RateLimit-Reset headers.

### 1.4 Versioning Strategy
URL path versioning (/v1, /v2) is preferred for public APIs because it is visible, cacheable, and
routable at the gateway. Additive changes (new optional fields, new endpoints) stay in the same major
version; renames, removals, type changes, and tightened validation require a new major version with a
90-day Sunset window. Never version with magic query flags or content negotiation alone.

### 1.5 Authentication & Authorization Models
Prefer short-lived JWT access tokens (5–15 minutes) with refresh-token rotation for user traffic, and
mutual TLS or signed service tokens for machine traffic. Authorize with scopes (orders:write) checked
at the handler plus tenant predicates pushed into every query. Authentication answers who, scopes answer
what, tenant predicates answer whose data.

## 2. Patterns and Anti-Patterns

| Area | Pattern (do) | Anti-Pattern (avoid) |
|---|---|---|
| Errors | RFC 9457 problem-details with stable codes | Free-text messages and HTTP 200 with error bodies |
| Pagination | Opaque cursors, max 100 per page | Unbounded offset with limit=100000 |
| Mutations | Idempotency-Key with 24 h replay | Blind retries creating duplicate charges |
| Auth | Fail-closed scope and tenant checks | Client-supplied tenant_id trusted without verification |
| Timeouts | 2 s downstream timeout + circuit breaker | Unbounded waits cascading into gateway 504 storms |
| Payloads | Field selection (?fields=id,total) | Always returning full 2 MB nested graphs |
| Webhooks | HMAC-SHA256 signatures + dedup IDs | Unsigned callbacks re-playable by anyone |

## 3. Code Example: FastAPI Cursor Pagination with Idempotency

```python
from fastapi import FastAPI, Header, HTTPException
from pydantic import BaseModel, Field

app = FastAPI()
idempotency_store: dict[str, dict] = {}
orders: list[dict] = [{"id": f"ord_{i:05d}", "total_cents": 1999 + i} for i in range(1, 500)]

class OrderCreate(BaseModel):
    total_cents: int = Field(gt=0, le=100_000_00)
    currency: str = Field(pattern=r"^[A-Z]{3}$")

@app.post("/v1/orders", status_code=201)
def create_order(body: OrderCreate, idempotency_key: str = Header(alias="Idempotency-Key")):
    if idempotency_key in idempotency_store:
        return idempotency_store[idempotency_key]
    order = {"id": f"ord_{len(orders) + 1:05d}", **body.model_dump()}
    orders.append(order)
    idempotency_store[idempotency_key] = order
    return order

@app.get("/v1/orders")
def list_orders(cursor: str = "", limit: int = 25):
    limit = min(max(limit, 1), 100)
    start = int(cursor) if cursor.isdigit() else 0
    page = orders[start:start + limit]
    next_cursor = str(start + limit) if start + limit < len(orders) else ""
    return {"data": page, "next_cursor": next_cursor}
```

The replay branch guarantees duplicate POST deliveries return the original stored order instead of
creating a second charge, and the cursor slice keeps list queries bounded regardless of table size.

# API Contract Verification Specification Template

## 1. Release Pair
- Provider: OrdersAPI 1.4.3, spec openapi/orders-1.4.0.yaml, deployed to staging at 2026-09-22 14:05 UTC
- Consumers: AcmeWeb 2.14.0, PartnerSync 1.9.2, MobileApp 5.3.0
- Broker: https://broker.acme.example, matrix export attached as broker-matrix-2026-09-22.json
- Policy: additive-only within 1.x, removals sunset 90 days, can-i-deploy must be green for production

## 2. Static Validation & Diff
- Spectral: zero errors, 3 warnings waived (info contact URL uses http in dev example)
- Examples validation: 47 of 47 request/response examples valid against schemas
- oasdiff breaking vs 1.3.2: 0 breaking, 1 dangerous (new optional webhook event order.refunded documented and acked by PartnerSync), 6 safe additions
- buf breaking on proto billing/v2: zero breaking findings

## 3. Pact Verification Matrix
- AcmeWeb 2.14.0: 31 interactions verified against OrdersAPI 1.4.3, all green, states isolated per run
- PartnerSync 1.9.2: 12 interactions green, including HMAC webhook redelivery with dedup id wh_7d02
- MobileApp 5.3.0: 18 interactions green; 2 pending pacts for 1.5.0 preview excluded from gate
- can-i-deploy to production: green for all three pairs on 2026-09-22 15:40 UTC

## 4. Mock & Suite Evidence
- Prism mock from openapi/orders-1.4.0.yaml in validating mode: 96 partner tests green, zero spec violations
- Provider contract suite: 214 passed; consumer suites: AcmeWeb 88, PartnerSync 41, all deterministic reruns
- verify.sh: exit code 0 on 2026-09-23 run by contracts pipeline job 663

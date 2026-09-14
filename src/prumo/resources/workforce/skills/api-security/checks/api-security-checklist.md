# API Security & Hardening Checklist

- [ ] BOLA check: Object ownership verified on every entity query by ID
- [ ] Request payloads validated against strict schema; unknown properties rejected
- [ ] DTOs used for all responses; no raw database models exposed to clients
- [ ] Token bucket rate limiting active with HTTP 429 and Retry-After headers
- [ ] Security headers (HSTS, nosniff, no-store) configured; CORS origin explicitly whitelisted
- [ ] RFC 7807 format used for all API errors; stack traces completely masked from client
- [ ] Request body size limited (max 1MB) and collection endpoints enforce max pagination bounds

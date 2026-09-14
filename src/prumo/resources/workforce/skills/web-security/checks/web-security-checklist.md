# Web Application Security & Hardening Checklist

- [ ] Strict nonce-based CSP header active; unsafe-eval and inline scripts banned
- [ ] Output encoding verified; DOMPurify used for user HTML; innerHTML prohibited
- [ ] Cookies set with Secure, HttpOnly, SameSite=Lax/Strict, and __Host- prefix
- [ ] SSRF protection validates IP addresses against private/cloud metadata ranges
- [ ] File uploads validate magic bytes, randomize names, and disable in-browser execution
- [ ] CORS configured with explicit domain whitelist; wildcard with credentials banned
- [ ] Open redirects blocked via strict relative path validation

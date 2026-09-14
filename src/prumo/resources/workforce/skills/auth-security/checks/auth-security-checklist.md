# Authentication & Authorization Security Checklist

- [ ] OAuth 2.1 code flows enforce PKCE with cryptographically random state and nonce
- [ ] Passwords hashed using Argon2id or bcrypt (cost >= 12) with unique per-user salts
- [ ] Access tokens expire in <= 15 minutes; refresh tokens use rotation with reuse detection
- [ ] Server-side revocation denylist immediately terminates sessions upon password change
- [ ] Rate limiting and progressive delay enforced on all authentication endpoints
- [ ] Account recovery tokens expire in <= 30 minutes and prevent username enumeration
- [ ] Audit logs record all auth events with correlation IDs and zero credential leakage

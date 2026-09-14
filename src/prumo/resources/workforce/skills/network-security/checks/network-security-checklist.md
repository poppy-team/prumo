# Network Transport Security & Hardening Checklist

- [ ] TLS 1.3 enforced; deprecated ciphers (CBC, RC4, 3DES) disabled
- [ ] HTTP clients configured with strict connect, read, and write timeouts
- [ ] Outbound egress governed by explicit domain and port allowlist
- [ ] Host and X-Forwarded-Host headers validated to block DNS rebinding
- [ ] Slowloris mitigated via aggressive read-header and keep-alive timeouts
- [ ] WebSockets validate Origin on handshake and cap message frame sizes
- [ ] Internal network communication encrypted with mTLS and zero implicit trust

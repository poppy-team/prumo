# Model Context Protocol (MCP) Security Checklist

- [ ] Tool operations restricted to domain-specific functions; arbitrary shell denied
- [ ] Input arguments strictly validated against JSON Schema definitions
- [ ] Tool outputs treated as untrusted data with delimiters blocking prompt injection
- [ ] Protected paths (.git/, .env, credentials) completely blocked from tool access
- [ ] Mutating tool executions recorded in side-effect journal with hashes
- [ ] Subprocess sandboxing active with stripped parent environment variables
- [ ] High-impact operations require explicit human approval before execution

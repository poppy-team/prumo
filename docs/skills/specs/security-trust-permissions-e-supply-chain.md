# 79.F — Security, Trust, Permissions e Supply Chain: Skills P0/P1

> Authority: canonical specification.
> Logical ID: 79 F
> Source: Notion Living Book (3d89bb7d023f811eb481dedc79ff4353)
> Status: Skill Package de referência para 79 F — Security, Trust, Permissions e Supply Chain.


<aside>
🛡️

Security é a família com maior necessidade de **negative/adversarial testing**. Texto declarativo como “validar rigorosamente” não é evidence; cada threat relevante precisa de checks, fixtures, policy ou roteiro de review reproduzível.

</aside>

# Princípio de composição

`secure-coding` atua durante implementação. `security-review` atua como auditoria independente. Skills de domínio (`api-security`, `mcp-security`, `filesystem-security` etc.) especializam threats e checks. `threat-modeling` antecede risco relevante; `security-reviewer` não substitui o threat model.

| Skill | Pri. | Escopo alvo | Aprimoramentos | Aceite |
| --- | --- | --- | --- | --- |
| **acp-security** | P0 | Comunicação entre agents. | Identity, message provenance, capability delegation, replay, confused deputy, injection, cross-agent trust e protocol/version boundaries. | Adversarial ACP corpus. |
| **api-security** | P0 | APIs HTTP/RPC públicas/privadas. | BOLA/BFLA, authz, rate/quotas, input/content, pagination abuse, replay, SSRF where applicable, CORS/CSRF context, resource exhaustion. | Threat→fixture/check mapping. |
| **auth-security** | P0 | Identity/session/token lifecycle. | OAuth/OIDC/passkey/MFA as applicable, token/session fixation, rotation/revocation, reset/recovery, privilege changes, CSRF. | State-machine tests de auth flows. |
| **desktop-security** | P0 | Apps desktop/native. | IPC, URI handlers, updater signing, local secrets, shell/process, filesystem, native plugins, sandbox e privilege boundaries. | Desktop-specific abuse corpus. |
| **filesystem-security** | P0 | Paths/files/archives/temp. | Traversal, symlink race, TOCTOU, canonicalization, permissions, atomic writes, temp files, archive extraction, special files. | Adversarial filesystem fixtures. |
| **mcp-security** | P0 | MCP servers/tools/resources. | Poisoned descriptions/schemas, tool injection, malicious outputs, confused deputy, scope escalation, data exfiltration, trust classification. | Adversarial MCP corpus + permission assertions. |
| **network-security** | P0 | Transport/network trust. | TLS/cert validation, downgrade, DNS, MITM, replay, mTLS when applicable, DoS/resource exhaustion, secret transport. | Protocol-specific attack fixtures. |
| **plugin-security** | P0 | Third-party extension boundary. | Signing/trust, permissions, sandbox, FS/network/process scopes, update/revocation, dependency isolation, crash containment. | Malicious-plugin corpus. |
| **process-execution-security** | P0 | Spawn/shell/process. | argv vs shell, quoting, PATH hijack, env sanitization, cwd, limits, timeout, privilege drop, sandbox. | Untrusted input nunca chega a shell concatenado. |
| **secrets-security** | P0 | Credential lifecycle. | Detection, storage, transit, redaction, rotation, revocation, scope, fixture secrets e history cleanup. | Secret fixture não vaza em logs/evidence/artifacts. |
| **secure-coding** | P0 | Baseline de implementação segura. | CWE-oriented base, safe defaults, boundary validation, negative examples, dispatch por linguagem/domínio; não duplicar specialists. | Specialist rules têm precedência e implementação não se autoaprova. |
| **security-review** | P0 | Auditoria independente. | Attack-surface delta, manual review, SAST/SCA/secrets/DAST as applicable, exploitability, severity, false positives, waiver lifecycle e remediation proof. | Coverage statement explícito. |
| **supply-chain-security** | P0 | Dependencies/build/artifacts. | Pin/lock, SBOM, provenance, signing, build isolation, typosquat/malicious packages, license policy e artifact trust. | Ingress e release possuem trust evidence. |
| **threat-modeling** | P0 | Threat model vivo. | Assets, actors, entry points, trust boundaries, abuse cases, mitigations, residual risk, assumptions, refresh triggers. | Architecture/trust change marca model stale. |
| **untrusted-project-security** | P0 | Repos não confiáveis. | Prompt injection em docs/repo, hooks, scripts, binaries, symlinks, submodules, package scripts, generated content e safe-exploration tiers. | Index/open não executa conteúdo implicitamente. |
| **web-security** | P0 | Browser/web application. | XSS, CSRF, CSP, clickjacking, cookie/session, redirects, uploads, CORS, cache poisoning e SSRF server-side quando aplicável. | OWASP-oriented adversarial fixtures. |

# Threat-to-evidence rule

Toda seção normativa de security deve responder cinco perguntas:

1. **Qual ameaça?**
2. **Qual trust boundary/asset?**
3. **Qual mitigação/invariante?**
4. **Como validar?** check, static analysis, dynamic fixture, review ou policy.
5. **Qual evidence prova o resultado?**

# Waiver Contract

Waiver nunca é booleano solto. Deve conter owner/authority, finding ID, rationale, compensating control, scope, expiration/review date e residual risk. High/critical não pode ser silenciosamente ignorado.

# Specialist activation examples

- change em auth → auth-security + threat-model delta + independent security review;
- read/write filesystem de input externo → filesystem-security;
- MCP/tool integration → mcp-security + process/filesystem/network conforme capabilities;
- third-party plugin → plugin-security + supply-chain;
- web UI sem backend → web-security subset adequado, sem ativar API security automaticamente;
- package/dependency update → supply-chain-security sem forçar full application threat model se risk profile não justificar.

# Exit Gate da família

Todos os P0 possuem negative/adversarial fixtures ou checks quando tecnicamente possíveis; skills deixam de repetir boilerplate; activation evita carregar segurança irrelevante; waivers e evidence tornam-se auditáveis.

# Atualização de auditoria — 2026-09-16

<aside>
🛡️

**Mudança de modelo:** security skills devem deixar de depender de frases absolutas como “100% clean” e passar a produzir findings contextualizados, rastreáveis e retestáveis. O Prumo precisa distinguir ausência de achados da ausência de cobertura.

</aside>

## Security Evidence Contract vNext

Cada finding deve possuir, quando aplicável: `finding_id`, standard/control ID, asset/trust boundary, source→sink ou attack path, preconditions, reachability, exploitability, severity, confidence, affected targets/versions, proof/evidence, remediation, compensating controls, waiver lifecycle e retest evidence.

## Standards / knowledge roots

- **NIST SSDF 1.1 (SP 800-218):** organizar práticas de secure development ao longo do SDLC.
- **OWASP ASVS 5.0:** requirements verificáveis para application security; preservar IDs/versionamento e consumir material machine-readable quando útil.
- **OWASP SAMM:** maturity/process improvement de segurança sem misturar com finding-level verification.
- **SLSA 1.2:** provenance e supply-chain integrity de source/build/artifacts.
- **OWASP Agentic Security Initiative:** knowledge root para riscos de agentes autônomos e multi-step tool use.

## Nova especialização proposta: `agentic-security`

Deve cobrir indirect prompt injection, poisoned repo/docs/skills, malicious tool descriptions/schemas/outputs, MCP/tool poisoning, data exfiltration, capability confusion, cross-agent confused deputy, unsafe delegation, excessive agency, persistent-memory poisoning, secret exposure via context/evidence e command/process injection mediada por tool arguments. Ela compõe `untrusted-project-security`, `mcp-security`, `process-execution-security`, `secrets-security`, `plugin-security` e `supply-chain-security`; não deve duplicá-las.

## Regra para scripts e scanners

Semgrep/CodeQL/SCA/secrets/sanitizers/fuzzers são providers de evidence, não definição da skill. Provider selection deve ser stack/risk-aware e falhas não podem ser neutralizadas por `|| true` em gates críticos. Findings precisam de triage, confidence e coverage statement explícitos.

## Referências canônicas

- [NIST SSDF 1.1](https://csrc.nist.gov/pubs/sp/800/218/final)
- [OWASP ASVS](https://owasp.org/www-project-application-security-verification-standard/)
- [OWASP Agentic Security Initiative](https://genai.owasp.org/initiatives/agentic-security-initiative/)
- [SLSA 1.2](https://slsa.dev/spec/v1.2/)

# Amendment 2026-09-21 — Security Assurance Expansion

[84 — Total Assurance Constitution: Gauntlet Loop, Evidence e Anti-False-Green](../../quality/gauntlet-84.md) e 84.D passam a compor o exit gate. Acrescentar data-integrity/crash consistency, malformed-input resilience, resource-exhaustion, recovery/idempotency, negative-space privacy e failure-containment aos threat→evidence mappings. Finding de segurança sem reproduction/evidence ou waiver governada não pode desaparecer por classificação “baixo risco” do próprio implementador.
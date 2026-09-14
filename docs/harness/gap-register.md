# Harness Gap Register (canônico)

Single source of truth for everything missing. Authority: task DoD (§35) +
snapshot `2026-09-11-d41f18bb65f5` (págs. 18, 19, 22–27, 29–35, 40–41) +
code/tests reality. `ACCEPTED != implemented`; environmental and
approval-gated items are labeled, never disguised.

**Disciplina:** todo incremento atualiza os Status aqui e imprime a tabela
no relatório da rodada. IDs são estáveis — nunca reutilizar um ID para outro
item. Novos gaps entram no fim com o próximo número livre.

**Legenda:** ✅ done · 🟡 partial · ⬜ open · 🛑 blocked · ⏸️ deferred ·
🚫 out-of-scope (deste Goal).

## 1. Capability gaps

| ID | Item | Fonte | Status | Evidência / limite | Próximo passo |
|----|------|-------|--------|--------------------|---------------|
| GAP-001 | Budget envelope no path real | DoD §35, HA4 | ✅ done | runlayer Tracker + flags + persist budget-<run>.json, CLI e daemon | — |
| GAP-002 | Evidence/Gates do protocolo no loop | HA2/HA4, pág. 05 | ✅ done | evidence-<run>.json + `--strict` + políticas declarativas (`--gates`) | motores externos futuros |
| GAP-003 | Team binding real (Runner aninhado por role) | HA10/HA14 | ✅ done | team/bind.go: runs aninhados + budget do role + checkpoints | — |
| GAP-004 | Egress + segredos no path local | HA-seg, págs. 07/19 | ✅ done | redação default + EgressPolicy fail-closed c/ allowlist (`--egress-deny/--allow`) | postura legacy quando nil (explícito) |
| GAP-005 | Roteamento por custo/latência/privacidade/quota | HA6, pág. 24 | ✅ done | Policy + QuotaState c/ cooldown 429 + exclusão; pricing via caller | billing vivo futuro |
| GAP-006 | Retrieval estrutural (FTS/BM25, símbolos/LSP, repo map) | HA9, págs. 08/31 | ✅ done nos limites | BM25 + repo-map + LSP (symbols/hover/definition, fallback) | typed graph + embeddings |
| GAP-007 | Regiões gerenciadas + JSON Patch no doc compiler | HD-base, pág. 26 | ✅ done nos limites | regions + RFC6902 + seções Markdown estruturais | AST pleno (listas/tabelas) |
| GAP-008 | Merge/conflict explícito entre worktrees | HA10 | ✅ done | three-way + deleções propagadas/conflitadas, base intacta | — |
| GAP-009 | Steering + compaction | HA2, pág. 05 | ✅ done | Inject/op/CLI/SDK + CompactKeep/Budget auto + ACP bridge | — |
| GAP-010 | ACP Agent Server (expor Runtime a editores) | H11, págs. 06/22 | ✅ done nos limites | servidor ACP v1 (spec oficial) + `agent acp` + bridge testada vs daemon | verificação c/ cliente real (Zed) |
| GAP-011 | MCP client (transporte + integração tools) | H5 | ✅ done nos limites | stdio + HTTP c/ sessão + Adapter + Fanout + `--mcp` | SSE streams futuros |
| GAP-012 | Benchmarks (packing, compilação, Runner) | relatório | ✅ done | compile ~6.8ms, BM25 ~0.78ms, run ~7µs (i7-3632QM) | — |
| GAP-013 | Operação do daemon (PID lock, rotação, unit, stop) | daemon | ✅ done | lock/stale-takeover + stop + rotação + prune + unit doc | — |
| GAP-014 | Kill -9 real com side effect pendente | HA4/HA8 | ✅ done | TestDaemonKillRecovery: SIGKILL + takeover + store íntegro | kill mid-side-effect em CI |
| GAP-015 | Approvals persistidas | HA3/pág. 07 | ✅ done | permissions-<run>.jsonl em CLI+daemon | — |
| GAP-016 | Bridge AgentEvent → observability.Event | HA4 | ✅ done | obs-<run>.jsonl dual-write CLI+daemon | — |
| GAP-017 | Planning→Build (PlanningSession ⇒ Run) | pág. 25 | ✅ done | handoff/promote.go + `agent promote [--start]` | — |
| GAP-018 | Memory Atlas | págs. 25/27-G12 | ✅ done nos limites | Atlas local + recall + Promote c/ gates (restricted/confidential nunca cruzam) | freshness/TTL automáticos |
| GAP-019 | Agent writes KnowledgeDelta-first (G15) | pág. 27-G15 | ✅ done | seeding via Commit com Author | — |
| GAP-020 | Retention/GC (checkpoints, eventos, knowledge) | pág. 27-G23 | ✅ done | RetentionPolicy + GC (prune 5, órfãos 30d, provenance intacta) + `agent gc` | política por projeto futura |
| GAP-021 | Retry com backoff no Gateway | HA6 | ✅ done | classes (rate 5x/servidor 2x) + jitter determinístico | — |
| GAP-022 | Child runs/subagentes com ownership (H14) | H14 | ✅ done | RunWork aninhado + ChildHandoff + budgets | — |
| GAP-023 | Scheduled runs (H16) | H16 | ✅ done | retry linear + dead-letter + last-status | supervisão externa (systemd doc) |
| GAP-024 | Provedores sandbox adicionais (H13) | H13 | 🟡 partial | StrongProvider detect + `--sandbox-runtime` (ex. runsc) | execução verificada + remota |
| GAP-025 | Contratos Local Intel (KnowledgeTask, Router, ResourceManager, Supervisor) | págs. 29–30 | 🟡 partial | tipos + router + supervisão c/ backoff + detecção llama-server | workers de inferência + benchmarks |
| GAP-026 | Research Ledger first-class (G11) | pág. 27-G11 | ✅ done | Add/Resolve/Open Delta-first | — |
| GAP-027 | Token-estimate index (G18) | pág. 27-G18 | ✅ done | tabela tokens-v1 + uso no Context | calibração medida |
| GAP-028 | Lint dos agent docs + doc-evals (G19/G21) | pág. 27 | ✅ done | humandocs.Lint (presença/fiação/higiene) | lint de agent-docs legados |
| GAP-029 | Schema evolution/migrations (G20) | pág. 27 | ✅ done | schemareg (parse-all + Migrate por versão) | migrações quando houver v2 |
| GAP-030 | Transporte remoto do daemon (+auth) | split gate | ✅ done nos limites | TCP+TLS + token (subtle), SDK+CLI, teste live local | CA corporativa + hardening de exposição |
| GAP-031 | Decisão: MCP Go SDK e transports | pág. 19 | ✅ decided | stdlib JSON-RPC registrado em mcp.go (troca sem mudar superfície) | reavaliar com benchmark |
| GAP-032 | Decisão: subset ACP + matriz oficial | pág. 19 | ✅ decided | subset v1 registrado no código (init/new/load/resume/list/delete/close/prompt/cancel) | registry/mCP-per-session futuros |
| GAP-033 | Decisão: Docker vs Podman padrão/rootless | pág. 19 | 🛑 decision | detecção honesta pronta | medir + ADR |
| GAP-034 | Decisão: driver SQLite derived runtime | pág. 19 | 🛑 decision | — | medir + ADR |
| GAP-035 | Decisão: isolamento de plugins | pág. 19 | 🛑 decision | — | ADR quando houver plugins |
| GAP-036 | Decisão: budgets de performance + TTL/memory-pressure | pág. 19 | 🛑 decision | — | medir + ADR |
| GAP-037 | Prova live do container | HA5 | 🛑 env | `TestContainerLive` pronto; daemon inacessível aqui | `PRUMO_LIVE_DOCKER=1` onde houver daemon |
| GAP-038 | Chaves live de models | HA6 | 🛑 env | adapters prontos + stub-testados | `PRUMO_MODEL_API_KEY`/BASE_URL |
| GAP-039 | Sends live externos (opencode/codex) | HA7/HA8 | 🛑 approval | **gastam sua quota**; tudo ao redor live-verificado | sua aprovação explícita de spend |
| GAP-040 | Modelos locais + thresholds (benchmark-driven) | pág. 19/29 | 🛑 env+decision | No-LLM first-class mantido | hardware + corpus + aprovação |
| GAP-041 | Bindings não-Go (TS types do IDL) | HA11 | ✅ done nos limites | protocol.d.ts + client.ts c/ roundtrip live vs daemon Go | mais linguagens sob demanda |
| GAP-042 | Checkpoint retention no daemon store | HA4 | ✅ done | coberto por GAP-020 (mesmo store) | — |
| GAP-043 | Descoberta de modelos nos adapters reais | HA1 | ✅ done nos limites | OpenAI lista `/models`; Anthropic não expõe listagem (limite do vendor) | — |
| GAP-044 | Structured-output enforcement | HA1 | ✅ done | validador subset + enforcement nos adapters + response_format | subset documentado |
| GAP-045 | Impact analysis lexical (G5, legado M5) | pág. 27-G5 | ✅ done | matchers tipados path:/ext:/contract:/rel: + legado rotulado; entrada antiga preservada | — |

## 2. Definition of Done (§35) — estado por item

| # | Item DoD | Status |
|---|----------|--------|
| 1 | docs reconciled/promoted | ✅ |
| 2 | AgentRuntime contracts | ✅ |
| 3 | FakeProvider conformance | ✅ |
| 4 | ≥1 real ModelProvider works | ✅ código+discovery (live: GAP-038) |
| 5 | NativeAgent end-to-end | ✅ |
| 6 | tools via ToolGateway | ✅ |
| 7 | permission lifecycle | ✅ persistida (GAP-015) |
| 8 | Environment/Sandbox baseline | ✅ +redação+strong-detect (live: GAP-037) |
| 9 | checkpoint/restart/resume | ✅ +kill+SIGHUP-safe lock (GAP-014) |
| 10 | duplicate side effects prevented | ✅ |
| 11 | budget enforcement | ✅ fiação CLI/daemon + persist (GAP-001) |
| 12 | observability/events | ✅ bridge dual-write (GAP-016) |
| 13 | gateway routing/fallback | ✅ retry + policy (quota: GAP-005) |
| 14 | ≥1 external AgentProvider works | ✅ nos limites (send: GAP-039) |
| 15 | typed Handoff | ✅ |
| 16 | Context Compiler v2 baseline | ✅ repo-map+LSP+FTS+Atlas (graph futuro) |
| 17 | Knowledge Runtime baseline | ✅ (Atlas: GAP-018) |
| 18 | KnowledgeDelta validate/commit | ✅ Delta-first + ledger (GAP-019/026) |
| 19 | Coverage/Readiness baseline | ✅ |
| 20 | Documentation Compiler baseline | ✅ regions+patch (GAP-007) |
| 21 | multi-agent/worktree baseline | ✅ binding+merge aninhados |
| 22 | compatibility/eval suite | ✅ +kill-test+benches+TS (GAP-012/014/041) |
| 23 | `prumo agent` headless usable | ✅ |
| 24 | docs describe reality | ✅ |
| 25 | no P0 contradictions | ✅ (1 não-P0 registrada: regra de imports do `cmd`) |

## 3. Split gate — estado por item

| Item | Status |
|------|--------|
| HA0 contracts | ✅ |
| HA1 FakeProvider + first ModelProvider | ✅ |
| HA2 Native Agent | ✅ |
| HA3 Tool/Permission | ✅ |
| HA4 checkpoint/restart/resume | ✅ |
| HA5 ACI + Sandbox baseline | ✅ c/ limites |
| headless coding Run end-to-end | ✅ |
| versioned public protocol | ✅ (IDL + SDK Go; TS: GAP-041) |
| reconnect/replay | ✅ local + remoto TLS c/ token (GAP-030 nos limites) |
| **Veredito** | **NOT READY** — GAP-030 + GAP-041 + provas live |

## 4. Fora deste Goal (não entra na conta)

Desktop/TUI (prumo-code), execução cloud/microVM (H18), federação A2A (H17), site público/i18n (HD5+), polish visual, `prumo-code` em si.

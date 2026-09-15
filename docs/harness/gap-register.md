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
| GAP-010 | ACP Agent Server (expor Runtime a editores) | H11, págs. 06/22 | ✅ done nos limites | servidor ACP v1 (spec oficial) + `agent acp` + bridge testada vs daemon + cliente ACPClient no extagent (dogfood live Prumo→ACP→Prumo 2026-09-14, 0 API key) | verificação c/ cliente real (Zed) |
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
| GAP-032 | Decisão: subset ACP + matriz oficial | pág. 19 | ✅ decided | subset v1 registrado no código (init/new/load/resume/list/delete/close/prompt/cancel); cliente stdio implementado (ACPClient) cobrindo o mesmo subset | registry/mCP-per-session futuros |
| GAP-033 | Decisão: Docker vs Podman padrão/rootless | pág. 19 | ✅ decided | ADR 006: podman-preferred na detecção, docker como runtime de referência testado (GAP-037), rootless obrigatório quando disponível, fail-fast sem fallback silencioso | medir podman rootless quando houver ambiente |
| GAP-034 | Decisão: driver SQLite derived runtime | pág. 19 | 🟡 partial | ADR 007 (Proposed): modernc.org/sqlite puro-Go atrás da port DerivedIndex; cgo só opt-in | medir workload real do primeiro componente SQLite para Promote |
| GAP-035 | Decisão: isolamento de plugins | pág. 19 | ✅ decided | ADR 008: sem loader in-process; extensões out-of-process via contratos existentes (MCP/ACP/CLI providers); plugin futuro = processo próprio com handshake tipado | YAGNI até uso real |
| GAP-036 | Decisão: budgets de performance + TTL/memory-pressure | pág. 19 | 🟡 partial | ADR 009 (Proposed): política de baselines medidos agora; thresholds só de distribuições medidas; TTL default 7d + prune-on-start | thresholds após primeiro componente SQLite (ADR 007) |
| GAP-037 | Prova live do container | HA5 | ✅ done | TestContainerLive PASS 2026-09-14 (docker, alpine:latest, limits + rede negada) | — |
| GAP-038 | Chaves live de models | HA6 | 🛑 env | adapters prontos + stub-testados | `PRUMO_MODEL_API_KEY`/BASE_URL |
| GAP-039 | Sends live externos (opencode/codex/cursor) | HA7/HA8 | ✅ done nos limites | spend aprovado: cursor-agent turno real PASS 2026-09-14 (auth própria, 0 API key); opencode Send reach server mas OpenRouter s/ saldo e opencode/* free em 429 (rate limit vendor); codex ChatGPT em usage limit até 19/09; caminho CLI opencode run também validado até o boundary (mesmo 429 vendor em mimo-v2.5-free) — bloqueio é 100% vendor-side, adapters e protocolo OK | re-testar opencode/codex quando quota voltar (após 19/09) |
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
| 25 | no P0 contradictions | ✅ (contradição não-P0 da regra de imports do `cmd` resolvida por ADR 005) |

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
| versioned public protocol | ✅ (IDL + SDK Go + SDK TS com roundtrip live: GAP-041) |
| reconnect/replay | ✅ local + remoto TLS c/ token (GAP-030 nos limites: CA corporativa) |
| ADRs do split | ✅ ADR 005–009 escritos (005/006/008 Accepted; 007/009 Proposed pendentes de medição) |
| **Veredito** | **READY (condicional)** — falta: PR/merge, re-testar opencode/codex sends quando quota voltar, Promote dos ADRs 007/009 com medição. Provas live fechadas 2026-09-14 (GAP-037/039/010) |

## 4. Fora deste Goal (não entra na conta)

Desktop/TUI (prumo-code), execução cloud/microVM (H18), federação A2A (H17), site público/i18n (HD5+), polish visual, `prumo-code` em si.

## 5. Documentation Control Plane Gaps (DOC-GAP-001 – DOC-GAP-030)

Source: `PRUMO_DOCUMENTATION_CONTROL_PLANE_DEEP_AUDIT_AND_WAVES.md` (2026-09-14).
Mapeados diretamente para as Waves W0–W21 em `docs/development/waves.md`.

| ID | Item | Tier | Wave | Status | Próximo passo |
|----|------|------|------|--------|---------------|
| DOC-GAP-001 | Authority/version drift não prevenido mecanicamente | P0 | W0, W1 | ⬜ open | Regra de drift e gate CI em W0 |
| DOC-GAP-002 | Semantic coverage/readiness excessivamente léxico (ANY-word bug) | P0 | W15 | 🟡 fixing | Substituir por matching semântico/estruturado em W15 (fix imediato em coverage.go) |
| DOC-GAP-003 | Verificação de evidência baseada em mera contagem | P0 | W15 | ⬜ open | Vincular evidência por ID, tipo e revisão |
| DOC-GAP-004 | Contradições entre documentos não barram completion | P0 | W15, W19 | ⬜ open | Ligar motor de contradições ao gate |
| DOC-GAP-005 | Fiação Goal → DocumentationPlan incompleta | P0 | W17 | ⬜ open | Preflight em Goal lock e postflight em delta |
| DOC-GAP-006 | Lint de HumanDocs valida apenas estrutura, não verdade/semântica | P0 | W10, W19 | ⬜ open | Verificador em camadas em W19 |
| DOC-GAP-007 | Instruções de agente como verdade paralela e obsoleta (context-rot) | P0 | W4, W16 | ⬜ open | AgentInstructionIR e compilador de superfícies |
| DOC-GAP-008 | Análise de impacto não cobre tipos semânticos suficientes | P0 | W17 | ⬜ open | Triggers por símbolo, API, schema e UI |
| DOC-GAP-009 | Pacotes de documentação fragmentados no runtime | P0 | W15 | ⬜ open | Coordenador do control plane em W15 |
| DOC-GAP-010 | Documentação "ready" diverge da verdade do código | P0 | W15 | ⬜ open | Relatório de dogfood semântico |
| DOC-GAP-011 | Modelo DocumentationUnit necessita semântica mais rica | P1 | W3, W10 | ⬜ open | Schema expandido em W10 |
| DOC-GAP-012 | Unidades de contexto e docs humanos não unificados | P1 | W3 | ⬜ open | Grafo único de KnowledgeUnits |
| DOC-GAP-013 | Publicação HD5+ ainda deferida | P1 | W18 | ⬜ open | Adaptador Starlight e gerador de rotas |
| DOC-GAP-014 | Ausência de projeções AI-native (llms.txt, MCP) | P1 | W18 | ⬜ open | Gerar /llms.txt e expor MCP de docs |
| DOC-GAP-015 | Documentação de UI/UX necessita ciclo de vida | P1 | W5, W20 | ⬜ open | Contratos especializados de UI em W5 |
| DOC-GAP-016 | Evidência visual sem semântica de frescor | P1 | W20 | ⬜ open | MediaRecord vinculado a estado de UI |
| DOC-GAP-017 | Fundação de tradução necessita ciclo de QA completo | P1 | W6, W20 | ⬜ open | Terminology, glossário e pseudo-localização |
| DOC-GAP-018 | Temas/personalização sem contratos explícitos | P1 | W9 | ⬜ open | Design tokens como API versionada |
| DOC-GAP-019 | Documentação de referência de API/CLI não orientada a contratos | P1 | W18 | ⬜ open | Adaptadores de referência de contratos de máquina |
| DOC-GAP-020 | Exemplos não são unidades de documentação executáveis | P1 | W19 | ⬜ open | Execução e teste de exemplos no pipeline |
| DOC-GAP-021 | Documentação de release/depreciação/migração sem ciclo estruturado | P1 | W20 | ⬜ open | Política de versão e plano de release |
| DOC-GAP-022 | Observabilidade documental subespecificada | P1 | W21 | ⬜ open | Métricas integradas ao Project Intelligence |
| DOC-GAP-023 | Feedback de usuário/suporte não alimenta planning documental | P2 | W21 | ⬜ open | Rastrear falhas de intenção documental |
| DOC-GAP-024 | Workforce documental necessita especialização por impacto | P2 | W21 | ⬜ open | Roteamento dinâmico de skills por tipo de impacto |
| DOC-GAP-025 | Segurança e visibilidade documental necessitam política de projeção | P2 | W21 | ⬜ open | Filtro de visibilidade público vs interno |
| DOC-GAP-026 | Edição com AST completo de Markdown (listas/tabelas) | P2 | W21 | ⬜ open | Parser AST completo no doccompile |
| DOC-GAP-027 | Performance de build em monorepos/grandes repositórios | P2 | W21 | ⬜ open | Benchmarking de invalidação e build |
| DOC-GAP-028 | Adoção e migração brownfield de documentação | P2 | W21 | ⬜ open | Descoberta semântica e proposta de bindings |
| DOC-GAP-029 | Débito documental como classe de débito tipada | P2 | W21 | ⬜ open | Integração com registro de dívida técnica |
| DOC-GAP-030 | Rastreabilidade e explicabilidade no nível de consulta | P2 | W21 | ⬜ open | Proveniência de query registrada no ContextManifest |

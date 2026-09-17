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
| GAP-047 | Critérios 4 e 5 do H10 não medidos | pág. 13/H10 | ✅ done | Critério 4: `prumo tui --remote/--token/--token-file/--remote-tls-cert` monta o **mesmo** cliente SDK em outro endereço (não um segundo caminho de código) e não supervisiona daemon local; `TestLiveRemoteDaemonFlow` roda palette→goal→stream→**aprovação**→evidência sobre TCP+TLS+token e confirma que token errado é recusado. Critério 5: baseline medido na hardware de referência do ADR 009 (i7-3632QM), janela de 200 rows — `View` 4,38 ms/frame (83 KB, 1846 allocs), `Timeline.Append` 146 µs/op (2 allocs), `Session.Poll` com log cheio e cursor avançado 45 µs/op (1 alloc). **Nenhum threshold inventado**: os números estão no ADR 009, como a metodologia exige, e o critério 5 era explicitamente não-bloqueante | — |
| GAP-046 | Op de aprovação no protocolo (H10 critério 2) | pág. 13/H10 | ✅ done | Protocolo 0.2.0 ganha `approve`/`deny` (`run_id` + `request_id`). Três defeitos encadeados foram consertados: (1) `perm.Engine.Evaluate` consulta a decisão gravada antes da política — antes `Approve` era registrado na trilha e **ignorado**, então aprovar não mudava nada; (2) o yield para aprovação persiste checkpoint com `pending_tools`/`pending_permissions` (o schema do checkpoint não descrevia nenhum dos dois) e o evento `permission_wait` passa a nomear o `request_id`, sem o qual o cliente não tem o que responder; (3) o daemon mantém o run vivo em `awaiting_approval` e reexecuta o turno em vez de descartar o runner — `yielded` (run parado) e `awaiting_approval` são estados distintos, e confundi-los faria o cliente ou travar ou sair do painel cedo. Superfície: `agent approve\|deny` + `agent ps` nomeando o pendente, SDK Go/TS, e a TUI (`a`/`d` no painel de run). Provas: `TestDaemonApprovePermission`, `TestDaemonDenyPermission`, `TestAgentApprovePendingPermission`, `TestVerticalSliceFlow` e `TestLiveRemoteDaemonFlow` (aprovação sobre TLS+token) | — |
| GAP-050 | Inventário de interface com posição e interligações | W5/W22, págs. 13/24 | ✅ done | `ui-component-contract` existia desde W5 e **nenhum produtor existia**: as quatro obrigações (`ui.component-contracts`, `ui.information-architecture`, `ui.screen-inventory`, `ui.layout`) estavam satisfeitas por waiver, e o gate ficava verde enquanto nenhum inventário de componentes existia. Agora: `docs/ui-ux/interface-map.json` (24 elementos, 10 interligações tipadas) + `internal/uimap` + `prumo ui map\|verify\|impact`, com projeções para desenvolvedor, site e code agent. Declarado vence, derivado preenche, divergência é erro (ADR 012) | — |
| GAP-048 | Categorias ocultas na ajuda de topo | CLI | ✅ done | `helpCategoryOrder` não listava `Harness`, `Adoption & Migration` nem `Platform & Automation`, então `agent`, `tui`, `adopt` e `automation` **nunca apareciam** em `prumo help` — inclusive `agent`, a porta de entrada do harness. Um comando que ninguém descobre não existe. Ordem corrigida e `TestEveryCommandCategoryIsPrinted` falha se alguma categoria do registry deixar de ser impressa | — |
| GAP-049 | Projeção de tokens: resolução no-op e no-color incompleto | Stack H | ✅ done | Dois defeitos da mesma família ("uma afirmação que nenhum gate verificava"). `Palette.resolveReferences` tinha receiver por valor e remapeava o mapa, então a resolução era **descartada** e todo consumidor lia referências cruas (`token:color.panel`); passou despercebido porque o único tema com valor literal era o `no-color`. E `theme.no-color` neutralizava 4 de 10 tokens de cor: o tema prometia monocromia e ainda pintava. Corrigido com receiver por ponteiro e `validateNoColorIsComplete`, que falha fechado quando um tema suprime a cor pela accent mas deixa outros tokens de cor resolvidos | — |

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

## 5. Documentation Control Plane Gaps (DOC-GAP-001 – DOC-GAP-033)

Source: `PRUMO_DOCUMENTATION_CONTROL_PLANE_DEEP_AUDIT_AND_WAVES.md` (2026-09-14,
artefato de entrada gitignored). Mapeados para as Waves W0–W21 em
`docs/development/waves.md`. Decisão arquitetural: ADR 010 (Proposed).

> Nota de status: o fix **léxico** imediato do DOC-GAP-002/003 aterrissou em
> `internal/documentation/coverage.go` em 2026-09-14 (matching por todas as
> palavras-chave + stopwords; evidência distinta e não-vazia) com regressão
> `TestFalseGreenKnowledgeMatchingPrevented` / `TestEvidenceDistinctAndNonEmpty`.
> Em 2026-09-15 o alvo chegou: `internal/documentation/semantic.go` implementa
> requirement → claim → evidence e o avaliador léxico foi **rebaixado a
> diagnóstico** (`mode: lexical`, `authoritative: false`, estado máximo
> `unverified`). W15.10 também aterrou: os 7 contratos do próprio repositório
> carregam claims aceitas com evidência verificada e `prumo docs readiness`
> reporta `ready: true` sem warnings. Vale registrar **por que** demorou: a
> primeira versão dos bindings citava `docs/migration/conformance-strategy.md`,
> que o authority map classifica como **historical**, e o avaliador rejeitou como
> `non-canonical-evidence` em vez de aceitar o verde. Foi o caso D002 do corpus
> léxico (misbinding de evidência) disparando no repositório real.
>
> Em 2026-09-15 as waves W17–W21 também aterraram: impacto semântico tipado e
> preflight/postflight de Goal (W17), plano de publicação e recuperação por IA
> (W18), verificador determinístico, gate `--strict` e Gauntlet (W19), ciclo de
> vida de tradução/mídia/release (W20) e inteligência documental + adoção
> brownfield (W21).

| ID | Item | Tier | Wave | Status | Próximo passo |
|----|------|------|------|--------|---------------|
| DOC-GAP-001 | Authority/version drift não prevenido mecanicamente | P0 | W0, W1 | ✅ done | W0: `docs/governance/authority.md` + `docs/AUTHORITY_MAP.json` + `prumo docs authority` (gate `docs-authority`); W1: `conformance/schema-runtime` (identidade, draft, enums, integridade referencial, round-trip) |
| DOC-GAP-002 | Semantic coverage/readiness excessivamente léxico (ANY-word bug) | P0 | W15 | ✅ done | `semantic.go`: requirement → claim → evidence; léxico virou diagnóstico não-autoritativo (W15.3–W15.7); corpus D001–D011 |
| DOC-GAP-003 | Verificação de evidência baseada em mera contagem | P0 | W15 | ✅ done | Evidência exige `id` estável, `target` = claim, revisão atual, `verified` e artefato existente; dedup por ID (`TestEvidenceDistinctAndNonEmpty`) |
| DOC-GAP-004 | Contradições entre documentos não barram completion | P0 | W15, W19 | ✅ done | W15: findings `contradiction` e `superseded-claim` bloqueiam readiness (D010/D011); W19: `VerifyDocsStrict` + Gauntlet contínuo sobre documentos |
| DOC-GAP-005 | Fiação Goal → DocumentationPlan incompleta | P0 | W17 | ✅ done | Preflight em Goal lock (`BuildPlanForContracts`), postflight em delta (`Reconcile`), `DocumentationGap`/`DocumentationPlan` populados em `internal/app/goalplan.go` |
| DOC-GAP-006 | Lint de HumanDocs valida apenas estrutura, não verdade/semântica | P0 | W10, W19 | ✅ done | W10: schemas de quality-policy e evidence-requirement; W19: verificador em camadas (authority → managed regions → links → claim-drift) + `semantic-readiness` sob `--strict` |
| DOC-GAP-007 | Instruções de agente como verdade paralela e obsoleta (context-rot) | P0 | W4, W16 | ✅ done | W16: IR canônico (`docs/agents/instruction-ir.json`) → adapters (agents-md/copilot/cursor/claude/skill-md) com fingerprint e orçamento de tokens; gate de context-rot (`prumo docs agents verify` + `TestAgentSurfaceContextRotGate`) falha em referências obsoletas |
| DOC-GAP-008 | Análise de impacto não cobre tipos semânticos suficientes | P0 | W17 | ✅ done | `impact_semantic.go`: triggers por símbolo, schema, token, UI, API, evento, permissão, locale, mídia, Goal, Wave, ADR, release e evidência |
| DOC-GAP-009 | Pacotes de documentação fragmentados no runtime | P0 | W15 | ✅ done | Fronteira e direção de dependência definidas em `docs/architecture/documentation-control-plane.md`; composição no CLI, sem acoplar docengine ao harness |
| DOC-GAP-010 | Documentação "ready" diverge da verdade do código | P0 | W15 | ✅ done | `TestSemanticReadinessDogfood` + estado `unverified`; `prumo docs readiness` não passa por palavras e hoje reporta `semantic=7 lexical=0` com `ready: true` porque os claims existem de fato |
| DOC-GAP-011 | Modelo DocumentationUnit necessita semântica mais rica | P1 | W3, W10 | ✅ done | `documentation-unit` schema completo (fontes, evidência, visibilidade, sensibilidade, locale, freshness, supersede, projeções, token estimate) |
| DOC-GAP-012 | Unidades de contexto e docs humanos não unificados | P1 | W3 | ✅ done | `knowledge-unit` + 13 relações tipadas + identidade estável; o baseline do Knowledge Runtime já semeia por IDs estáveis (`knowledge.DeriveID`), com manifest derivado (`internal/harness/knowledge/manifest.go`) e CLI `prumo knowledge manifest\|search` |
| DOC-GAP-013 | Publicação HD5+ ainda deferida | P1 | W18 | ✅ done | `internal/docpublish`: interface de renderer, adaptador Starlight, rotas versionadas/localizadas, `prumo docs site build\|verify` |
| DOC-GAP-014 | Ausência de projeções AI-native (llms.txt, MCP) | P1 | W18 | ✅ done | `/llms.txt`, `/llms-full.txt`, Markdown cru, índice de busca, referência de API/schema e recursos MCP de documentação: leitura sempre disponível (`prumo://docs/...`, `docs resources read`) e mutação negada por padrão (`docs mutations`, `allowed: []`) |
| DOC-GAP-015 | Documentação de UI/UX necessita ciclo de vida | P1 | W5, W20, H10 | ✅ done | W5 entregou os 16 contratos `ui.*`/`tui.*` e a evidência visual chegou em W20. **Correção de registro:** a entrada anterior afirmava que W5 havia entregado também "state matrix (22 estados) e component contract" — o artefato **não existia**, e ninguém percebeu porque os contratos `ui.*` não tinham binding e a capability `tui` nunca era composta, então nunca foram avaliados. `docs/ui-ux/state-matrix.json` foi escrito em H10 e agora é verificado por `TestStateMatrixIsCompleteAndExplicit`; o component contract segue dispensado com motivo em `bindings.json` (`ui.component-contracts`) |
| DOC-GAP-016 | Evidência visual sem semântica de frescor | P1 | W20 | ✅ done | `media-record` schema + `doclifecycle.MediaStatuses`: staleness por digest, ligada a estado/tema/locale/plataforma |
| DOC-GAP-017 | Fundação de tradução necessita ciclo de QA completo | P1 | W6, W20 | ✅ done | W6: lifecycle, locale-key, placeholders, reviewer, fallback, RTL; W20: `VerifyPseudoLoc` + staleness por digest de conteúdo |
| DOC-GAP-018 | Temas/personalização sem contratos explícitos | P1 | W9 | ✅ done | `design-token-set` schema + `docs/ui-ux/design-tokens.json` (tiers, temas no-color/high-contrast/reduced-motion, contraste medido, fallback TUI) + contrato `ui.personalization` |
| DOC-GAP-019 | Documentação de referência de API/CLI não orientada a contratos | P1 | W18 | ✅ done | `apiReferenceRenderer` gera as páginas de referência a partir de `docs/contracts/builtin.json` e de `schemas/*.schema.json`, marcadas como geradas (`Graph.Reference`) para nunca serem confundidas com páginas autorais |
| DOC-GAP-020 | Exemplos não são unidades de documentação executáveis | P1 | W19 | ⬜ open | Execução e teste de exemplos no pipeline |
| DOC-GAP-021 | Documentação de release/depreciação/migração sem ciclo estruturado | P1 | W20 | ✅ done | `docs/lifecycle.json` com release gates, `DeprecationFindings` (substituto ou remoção + anotação no documento) e `CheckRelease` → `prumo docs release` |
| DOC-GAP-022 | Observabilidade documental subespecificada | P1 | W21 | ✅ done | `internal/docintel.Collect` + `prumo docs metrics`: tokens, split léxico/semântico, traduções/mídia, links, claim drift, débito nomeado e `freshness_score` limitado |
| DOC-GAP-023 | Feedback de usuário/suporte não alimenta planning documental | P2 | W21 | 🚫 refused by design | Rastrear intenção exigiria um canal de telemetria; as regras de autoridade e trust do framework proíbem coletar o que usuários consultam. Decisão registrada em **ADR 011**; métricas vêm apenas do estado do repositório |
| DOC-GAP-024 | Workforce documental necessita especialização por impacto | P2 | W21 | ⬜ open | Roteamento dinâmico de skills por tipo de impacto (o impacto tipado existe em W17; o roteamento de workforce não) |
| DOC-GAP-025 | Segurança e visibilidade documental necessitam política de projeção | P2 | W21 | ✅ done | `Graph.Human()` vs `Current()` separa público/interno e superfícies de agente; a saída publicada é derivada, sob `.prumo/runtime/` |
| DOC-GAP-026 | Edição com AST completo de Markdown (listas/tabelas) | P2 | W21 | ⬜ open | Parser AST completo no doccompile |
| DOC-GAP-027 | Performance de build em monorepos/grandes repositórios | P2 | W21 | ✅ done | `TestLargeRepositoryIncrementalInvalidationIsBounded` (build frio escreve tudo, rebuild sem mudança não escreve nada, uma edição invalida um subconjunto) + `BenchmarkBuildLargeRepository` (200 docs: frio ~64,5 ms, inalterado ~14,5 ms; i7-3632QM) |
| DOC-GAP-028 | Adoção e migração brownfield de documentação | P2 | W21 | ✅ done | `Inspect`/`Propose`: sinais, inventário, duplicatas, propostas com confiança `inferred` e caminho de promoção; nada é escrito no repositório alvo |
| DOC-GAP-029 | Débito documental como classe de débito tipada | P2 | W21 | ✅ done | `documentationDebt` nomeia cada obrigação pendente; contrato não verificado sempre aparece como débito e reduz o score |
| DOC-GAP-030 | Rastreabilidade e explicabilidade no nível de consulta | P2 | W21 | ✅ done | `prumo docs query` retorna rota/heading/excerto rastreáveis e `prumo docs explain` explica unidade ou finding; a proveniência de consulta registrada (W21.2) foi recusada com DOC-GAP-023/ADR 011 |
| DOC-GAP-031 | Terminologia/glossário sem registro canônico | P1 | W20 | ✅ done | `docs/glossary.json` + `schemas/glossary.schema.json` + `internal/doclifecycle/glossary.go`: status fechado (preferred/deprecated/forbidden), termo deprecado deve nomear um substituto preferido, match por palavra inteira, `prumo docs glossary`, e verificação dentro de `prumo docs verify` |
| DOC-GAP-032 | Ausência de política explícita de versão documental | P1 | W20 | ✅ done | `version_policy` em `docs/lifecycle.json` + `schemas/doc-lifecycle.schema.json` + `internal/doclifecycle/versionpolicy.go`: buckets exclusivos (current/supported/deprecated/removed), drift contra `protocol.CLIVersion`, referência a linha removida reprovada salvo em documento histórico ou linha histórica anotada, `prumo docs version` |
| DOC-GAP-033 | Telemetria de consulta/intenção documental | P2 | W21 | 🚫 refused by design | W21.2/W21.3 não implementados: não existe canal de telemetria e registrar consultas armazenaria conteúdo sensível por construção. **ADR 011** registra a decisão, o limite e a condição para revertê-la |

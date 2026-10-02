# Prumo — Regression + Improvement Benchmark

> Commit medido: `dbc14203f258a50a0567a30b141e847c8246af6c` (2026-10-02T00:28:50-03:00) em `main`. Versao declarada e reportada: `0.6.1` / `0.6.1`.
> Todos os numeros deste relatorio sao calculados por `build_report.py` a partir de `audit-data.json`. Nenhuma nota foi digitada a mao.

## 1. Condicoes do teste

| Atributo | Valor |
|---|---|
| Repositorio canonico | `https://github.com/poppy-team/prumo` |
| Branch | `main (detached at origin/main)` |
| Commit SHA | `dbc14203f258a50a0567a30b141e847c8246af6c` |
| Data do commit | `2026-10-02T00:28:50-03:00` |
| Subject | `fix(ci): take the Go toolchain from go.mod instead of a stale pin (#85)` |
| Versao declarada | `0.6.1` |
| Versao reportada pelo CLI | `0.6.1` |
| go.mod exige | `1.26.6` |
| Working tree | `clean (0 modified paths); benchmark run from an isolated worktree at origin/main` |
| Projeto do experimento | `PulseSync — B2B SaaS webhook routing/automation platform with metering` |
| Data do benchmark | `2026-10-02` |

> **Caveat de medicao:** An older prumo 0.6.0 is installed at ~/.local/bin/prumo and shadows the built binary on PATH. Every measurement in this report uses the absolute path to the built binary.

> Baseline: commit `807343c1f1da0fbc8f5dceaea17d807e7377e38a` (2026-10-01), branch `feat/prumo-ide-workspace-viewer`.

## 2. Dashboard antes vs depois

| | Anterior | Atual | Delta |
|---|---:|---:|---:|
| Nota composta (ponderada + end-to-end) | 4.18 | **6.26** | **+2.08** |
| Media da Health Matrix (22 subsistemas) | 5.0 | 6.68 | +1.6799999999999997 |
| Mediana da Health Matrix | 4.0 | 7.0 | +3.0 |
| End-to-End Engineering Loop | 2.0 | 5.0 | +3.0 |
| Score medio da documentacao (21 docs com nota nos dois lados) | 4.77 | 4.96 | +0.1900000000000004 |
| Findings anteriores corrigidos | — | 23/32 | |
| Findings novos | — | 14 | |
| Bridges working | 0/12 | 5/12 | +5 |

Faixa da nota composta atual: **6.0-7.9 Funcional**.

### 2.1 Notas por familia (ponderadas)

| Familia | Peso | Antes | Agora | Delta |
|---|---:|---:|---:|---:|
| ux | 20% | 3.25 | 8.0 | +4.75 |
| docs | 25% | 4.33 | 6.67 | +2.34 |
| governance | 20% | 6.5 | 6.0 | -0.5 |
| harness | 20% | 5.06 | 5.5 | +0.4400000000000004 |
| reliability | 15% | 5.75 | 7.38 | +1.63 |
| **composto** | 100% | **4.91** | **6.67** | **+1.7599999999999998** |

## 3. Health Matrix comparativa

| Subsistema | Antes | Agora | Delta | Status | Evidencia |
|---|---:|---:|---:|:---:|---|
| Init / Scaffolding | 3.5 | **8.0** | +4.5 | PARTIAL | `prumo init` sem argumentos em projeto vazio: exit 0, 'Initialized Prumo v0.6.1', 59 arquivos, 23 docs .md, autodetecção de stack (go, javascript). Document Control Plane e repository policy |
| Living Plan / Interview | 3.0 | **8.5** | +5.5 | PARTIAL | `plan resume --goal` semeou 8 open questionsderived de docs readiness; `plan questions` listou 8; 2 respostas resolveram (8->7->6); `plan decisions` listou as 2 decisões aceitas [accepted].  |
| Answer Classification / Authority | 9.0 | **9.0** | +0.0 | PASS | `plan answer --classification agent-suggestion` => 'recorded as a proposed agent suggestion; not promoted to an accepted decision'; `plan decisions` permaneceu vazio. Garantia de autoridade  |
| Documentation Control Plane | 3.0 | **7.5** | +4.5 | PARTIAL | docs audit (exit 0, 7 contratos aplicáveis), docs authority (exit 0, 27 arquivos, 0 findings), docs readiness e docs verify executam. Dedução: funcional, porém projeto recém-criado FALHA em  |
| Docs Authority | 6.0 | **7.0** | +1.0 | PARTIAL | `docs authority` => current_version 0.6.1, 27 arquivos, findings: []. Dedução: sem regressão; a validação de conteúdo de projeção continua ausente. |
| Docs Contradictions | 4.0 | **5.5** | +1.5 | PARTIAL | Detecção genérica FUNCIONA para claims em forma config: docs/architecture/gateway-a.md 'listen_port: 3000' vs gateway-b.md 'listen_port: 8080' => 1 finding DOC_CONTRADICTION. NÃO detecta pro |
| Goal Lifecycle / Locks | 9.0 | **5.0** | -4.0 | FAIL | Detecção de adulteração INTACTA: `doctor` => exit 1 'Lock digest mismatch'; `goal state EXECUTING` => exit 1 'Illegal goal mutation detected'. PORÉM `goal amend` aceitou o arquivo adulterado |
| Goal Amendments | 4.0 | **7.0** | +3.0 | PARTIAL | Payload plano aceito (objective+acceptance aplicados); flags venceram o arquivo (reason=REASON_FROM_FLAG, approved_by=FLAG_AUTHOR). Faltam: validação de tipo (aceitou acceptance:string), e o |
| Plan / Task DAG | 7.5 | **7.0** | -0.5 | PARTIAL | `plan blueprint --plan --json` produziu 2 tasks com dependência real (T-2-review depende de T-1-security-trust). Validação de DAG não regrediu. Dedução: o parser funciona, mas nada consome a |
| Workforce Resolution | 7.0 | **6.5** | -0.5 | PARTIAL | 5 perfis por domínio: agents 8-9, skills 16-25, recipes 1-6. database=>database-engineer e frontend=>ui-component-engineer (especialização parcial). security=>NENHUM agente de segurança apes |
| Lean Progressive Context | 3.0 | **3.5** | +0.5 | FAIL | Recursão FUNCIONA: 25 de 36-38 itens são docs/*/*.md. MAS 5 tarefas semanticamente distintas (security/PostgreSQL/React/API/billing) deram overlap Jaccard 0.95-1.00, 35 de 37 refs idênticas, |
| Connectors / Adapters | 4.0 | **5.5** | +1.5 | PARTIAL | Regiões gerenciadas preservam conteúdo humano (BUG-007 FIXED, teste md5) e recompilação é idempotente. Subagents do Antigravity agora recebem o AGENT.md completo (4176 bytes, igual à fonte). |
| ACI / Tool Gateway | 6.5 | **6.5** | +0.0 | PARTIAL | Permissões aplicadas e registradas (perm-c1 decision=allow com fingerprint). A observação de ferramenta aparece na conversa restaurada do checkpoint, mas não é persistida no obs-*.jsonl (NEW |
| Evidence | 3.0 | **4.5** | +1.5 | PARTIAL | Registro rico e válido contra o schema: id, type, producer, timestamp, status, goal_id, run_id, confidence, metadata{failed_tools,phase,report_count,stop_reason,usage}. Dedução: shape corret |
| Gates | 6.5 | **3.5** | -3.0 | FAIL | O gate existe e é estrito, porém INALCANÇÁVEL pelo caminho real: após um agent run bem-sucedido, `goal state DONE` => exit 1 'no such evidence record', porque AppendRunEvidence grava id 'ev- |
| Agent Execution Loop | 4.0 | **4.5** | +0.5 | PARTIAL | Tool calls reais executam e o context manifest é materializado (36 itens, 7145 tokens, 2 excluídos por budget). Workspace NÃO foi modificado (src/limiter.js intacto). `prumo run`, anunciado  |
| Recovery / Resume | 3.0 | **8.0** | +5.0 | PARTIAL | Checkpoint persiste messages (5 e depois 12), tool_queue, after_side_effects, turns_done; `agent resume` executou RunUntilDone de verdade (12 passos, checkpoint r3 criado, evidência reescrit |
| Traceability | 2.0 | **8.0** | +6.0 | PARTIAL | `trace goal-m8` => exit 1 trace_not_found com a lista de nós reais (P00-G01, contract:*, doc:*). `trace docs/security/security-contract.md` => raiz + downstream (contract:security.trust). `t |
| Tooling de Verificação | 8.5 | **8.5** | +0.0 | PASS | Não re-testado em profundidade nesta rodada; sem evidência de regressão observada nos comandos exercitados. |
| Framework Self-Check | 8.0 | **8.5** | +0.5 | PASS | main @ dbc1420: 86 pacotes de teste ok, 0 falhas; race detector limpo nos pacotes tocados. framework-check é repo-scoped: fora da árvore de fontes retorna exit 1 listando schemas/workforce/a |
| Portabilidade do Binário | 2.5 | **7.0** | +4.5 | PARTIAL | Em diretório limpo e fora da árvore: version/init/validate/doctor/resolve/context/plan/goal/agent/trace todos funcionam (exit 0). schemas embutidos são usados (BUG-001 FIXED). Dedução: porta |
| Repository Governance | 3.0 | **8.0** | +5.0 | PARTIAL | init semeia .prumo/repository/policy.json (2657 bytes); `repo policy check --json` executa (falha apenas por não ser repositório git, diagnóstico correto). Todos os 45 comandos de top-level  |

Melhoraram: **15**. Pioraram: **4** (Goal Lifecycle / Locks, Plan / Task DAG, Workforce Resolution, Gates). Iguais: **3**.

## 4. Findings anteriores: re-testados um a um

| ID | Finding | Sev. | Antes | Agora | Status |
|---|---|---|---:|---:|:---:|
| BUG-001 | Schema registry quebra fora da árvore de fontes | P0 | P0 | corrigido | **FIXED** |
| BUG-002 | adopt --apply grava bindings.json incompatível com o engine | P0 | P0 | corrigido | **FIXED** |
| BUG-003 | Documentation Impact Analyzer inoperante | P0 | P0 | corrigido | **FIXED** |
| BUG-004 | LPC não enxerga documentos canônicos em profundidade >= 3 | P0 | P0 | corrigido | **FIXED** |
| BUG-005 | prumo trace fabrica grafo com dados do framework | P0 | P0 | corrigido | **FIXED** |
| BUG-006 | agent resume é stub e força conclusão sem executar trabalho | P0 | P0 | corrigido | **FIXED** |
| BUG-007 | compile sobrescreve regiões mantidas por humanos | P0 | P0 | corrigido | **FIXED** |
| BUG-008 | goal amend descarta flags e ignora payload sem 'changes' | P1 | P1 | corrigido | **FIXED** |
| BUG-009 | init não gera arquivos do Documentation Control Plane | P1 | P1 | corrigido | **FIXED** |
| BUG-010 | Ajuda de CLI documenta subcomandos inexistentes | P1 | P1 | corrigido | **FIXED** |
| BUG-011 | core-software força cli.reference e installation.lifecycle | P1 | P1 | corrigido | **FIXED** |
| BUG-012 | docs verify --strict não detecta documento vinculado removido | P1 | P1 | corrigido | **FIXED** |
| BUG-013 | Templates de scaffold contradizem profile e locale | P1 | P1 | parcial | **PARTIALLY_FIXED** |
| BUG-014 | Versões divergentes entre artefatos gerados | P2 | P2 | corrigido | **FIXED** |
| BUG-015 | Adapter Antigravity perde agentes e emite stubs | P2 | P2 | corrigido | **FIXED** |
| BUG-016 | Adapters triplicam 5,1 MB byte-idênticos | P2 | P2 | parcial | **PARTIALLY_FIXED** |
| BUG-017 | goal list engole silenciosamente meta corrompida | P2 | P2 | corrigido | **FIXED** |
| BUG-018 | docs contradictions só detecta números de porta | P2 | P2 | parcial | **PARTIALLY_FIXED** |
| BUG-019 | Evidence registra término do loop, não atingimento do objetivo | P2 | P2 | parcial | **PARTIALLY_FIXED** |
| UX-001 | Zero-argument init e modo interativo não existem | P2 | P2 | corrigido | **FIXED** |
| UX-002 | Ajuda de repo e bootstrap de policy.json | P2 | P2 | corrigido | **FIXED** |
| DOC-001 | Conjunto documental não cobre o mínimo profissional de um SaaS | P2 | P2 | presente | **STILL_BROKEN** |
| BR-01 | docs readiness não alimenta plan questions | P0 | P0 | corrigido | **FIXED** |
| BR-02 | adopt --apply incompatível com docengine.LoadBindings | P0 | P0 | corrigido | **FIXED** |
| BR-03 | contract.update_triggers não chega a AnalyzeImpact | P0 | P0 | corrigido | **FIXED** |
| BR-04 | context compiler não alcança docs/*/*.md | P0 | P0 | corrigido | **FIXED** |
| BR-05 | prumo trace desconectado do estado real | P0 | P0 | corrigido | **FIXED** |
| BR-06 | Goal não alimenta agent run / evidência | P1 | P1 | parcial | **PARTIALLY_FIXED** |
| BR-07 | plan blueprint não executa tasks | P1 | P1 | presente | **STILL_BROKEN** |
| BR-08 | workforce resolution não chega ao runtime | P1 | P1 | presente | **STILL_BROKEN** |
| BR-09 | Evidence não governa Gates | P1 | P1 | presente | **STILL_BROKEN** |
| BR-10 | checkpoint não alimenta resume real | P0 | P0 | corrigido | **FIXED** |

### 4.1 FIXED (23)

**BUG-001 — Schema registry quebra fora da árvore de fontes** (P0)

- Antes: cd /tmp/clean && prumo validate => error: schema registry: open schemas: no such file or directory
- Agora: cd /tmp/opencode/v2/portable && prumo validate => 'Prumo validation passed.' exit 0
- Evidencia: Portabilidade testada fora da árvore de fontes com o binário standalone; schemas embutidos são consumidos.

**BUG-002 — adopt --apply grava bindings.json incompatível com o engine** (P0)

- Antes: adopt apply && docs audit => json: cannot unmarshal object into Go value of type []docengine.Binding
- Agora: adopt apply grava array bare; docs audit => exit 0, 7 contratos; docs readiness => exit 0
- Evidencia: Produtor e consumidor concordam sobre o schema.

**BUG-003 — Documentation Impact Analyzer inoperante** (P0)

- Antes: 6/6 consultas de docs impact retornaram []
- Agora: docs/architecture/overview.md => [architecture.system]; docs/product/vision.md => [product.vision, project.scope, ui.product-ux]; docs/security/* => [security.trust, accessibility.accessible-auth]. Falsos positivos: README.md, src/web/app.ts, package.json, bindings.json => [] em todos.
- Evidencia: Analyzer Typed + detecção de trigger path: functioning com true positives e sem false positives.

**BUG-004 — LPC não enxerga documentos canônicos em profundidade >= 3** (P0)

- Antes: 4 tarefas distintas produziram o mesmo conjunto de 12 arquivos; zero docs de docs/*/*.md
- Agora: 25 de 36-38 itens incluídos são docs/*/*.md em todas as 5 tarefas compiladas
- Evidencia: Recursão corrigida. Ver NEW-007 para a falha remanescente de discriminação.

**BUG-005 — prumo trace fabrica grafo com dados do framework** (P0)

- Antes: prumo trace goal-m8 devolvia nós do marco M8 do Prumo
- Agora: prumo trace goal-m8 --json => exit 1, code trace_not_found, 'known nodes: P00-G01, contract:architecture.system, ...'
- Evidencia: Nenhum nó fabricado; o sistema declara honestamente a ausência de dados.

**BUG-006 — agent resume é stub e força conclusão sem executar trabalho** (P0)

- Antes: resume criava provider fake e mudava a fase para PhaseComplete sem chamar RunUntilDone; NativeAgentState sem Messages/ToolQ
- Agora: checkpoint persiste messages(12), tool_queue(1), after_side_effects, turns_done(12); agent resume => 'Resumed R-agent-1 from R-agent-1-r2: complete (12 message(s) restored, 12 step(s) taken)', checkpoint r3 criado e evidência reescrita
- Evidencia: Resume continua trabalho real; a mensagem de role 'tool' no checkpoint contém conteúdo real de arquivo lido.

**BUG-007 — compile sobrescreve regiões mantidas por humanos** (P0)

- Antes: seção customizada em CLAUDE.md era apagada no compile
- Agora: regiões prumo:begin/end presentes; conteúdo humano appended fora da região sobreviveu a 2 recompilações (md5 estável); recompilação é idempotente
- Evidencia: Preservação de região gerenciada e idempotência confirmadas por hash.

**BUG-008 — goal amend descarta flags e ignora payload sem 'changes'** (P1)

- Antes: com --file, --reason e --approved-by eram descartados; payload sem 'changes' ignorado
- Agora: payload plano aplicado (objective+acceptance); flags venceram o arquivo: reason=REASON_FROM_FLAG, approved_by=FLAG_AUTHOR; digest recalculado
- Evidencia: Ambas as metades do finding corrigidas. Ver NEW-001 para a falha nova no mesmo caminho.

**BUG-009 — init não gera arquivos do Documentation Control Plane** (P1)

- Antes: docs audit falhava por falta de builtin.json
- Agora: init gera docs/contracts/builtin.json (25487 B), docs/contracts/bindings.json (1035 B), docs/profiles/builtin.json (4398 B), docs/AUTHORITY_MAP.json (990 B); docs audit exit 0
- Evidencia: Control Plane semeado no primeiro uso.

**BUG-010 — Ajuda de CLI documenta subcomandos inexistentes** (P1)

- Antes: help plan listava init/show/update inexistentes; help repo listava branch-check/commit-check/pr-check inexistentes
- Agora: help lista 45 comandos de top-level e todos despacham (verificados por invocação). Subcomandos de plan (status/questions/resume/answer/decisions/delta/blueprint/seed) existem. repo branch-check/commit-check/pr-check caem no handler de policy e retornam erro de uso (exit 2), mas não são mais anunciados no help.
- Evidencia: Help e parser alinhados no nível de top-level.

**BUG-011 — core-software força cli.reference e installation.lifecycle** (P1)

- Antes: todo projeto herdava contratos de linha de comando
- Agora: docs/profiles/builtin.json: core-software => [product.vision, project.scope, architecture.system, testing.strategy, security.trust]; cli => [cli.reference, installation.lifecycle]
- Evidencia: Contratos de CLI isolados no perfil cli. Ver NEW-012 para inconsistência de inferência de tipo.

**BUG-012 — docs verify --strict não detecta documento vinculado removido** (P1)

- Antes: remover threat-model.md produzia os mesmos 9 findings com e sem o arquivo
- Agora: baseline 8 findings; após remover docs/security/threat-model.md => 9 findings, incluindo 'binding:missing-source ... contract security.trust is bound to this document, but it does not exist'
- Evidencia: Integridade referencial verificada.

**BUG-014 — Versões divergentes entre artefatos gerados** (P2)

- Antes: CLI 0.6.0, init anunciava v0.5, CLAUDE.md v0.2, GEMINI.md v0.5
- Agora: CLI => 0.6.1; prumo.json framework.version => 0.6.1; adapters gerados usam 'Prumo v0.6' derivado de CLIVersion
- Evidencia: Versão única e coerente.

**BUG-015 — Adapter Antigravity perde agentes e emite stubs** (P2)

- Antes: 9 de 12 agentes perdidos; subagents eram stubs de uma linha
- Agora: subagents Antigravity emitem o AGENT.md completo (architect.md = 4176 bytes = tamanho da fonte); roster padrão de 13 preserva os nomes de arquivo esperados (executor.md, verifier.md) mapeando para implementer e quality-reviewer
- Evidencia: Contratos reais entregues.

**BUG-017 — goal list engole silenciosamente meta corrompida** (P2)

- Antes: JSON inválido => continue silencioso e tabela vazia com exit 0
- Agora: goal list => exit 0 com 'warning: goal file ... is corrupt/unreadable' e linha 'P00-G02 ?? CORRUPT ? ⚠ corrupt or unreadable goal file'; doctor => exit 1 com 2 erros
- Evidencia: Corrupção visível e diagnosticada.

**UX-001 — Zero-argument init e modo interativo não existem** (P2)

- Antes: prumo init sem flags => error: --profile is required (exit 2)
- Agora: prumo init < /dev/null em projeto vazio => exit 0, 'Initialized Prumo v0.6.1', autodetecção de stack e projeto
- Evidencia: Bootstrap sem atrito; main implementa via detecção + preset (não via prompt interativo).

**UX-002 — Ajuda de repo e bootstrap de policy.json** (P2)

- Antes: repo policy exigia policy.json que nada criava; sem repo policy init
- Agora: init semeia .prumo/repository/policy.json (2657 bytes); `repo policy check --json` executa e reporta diagnóstico correto quando não é repositório git
- Evidencia: Bootstrap e comando presentes.

**BR-01 — docs readiness não alimenta plan questions** (P0)

- Antes: docs readiness reportava 2 contratos unverified; plan resume retornava questions: []
- Agora: docs readiness => 8 blocking_questions; plan resume => 'open questions 8'; plan questions lista 8 com contract, priority e blocking
- Evidencia: Ponte成本中transformada em intake real.

**BR-02 — adopt --apply incompatível com docengine.LoadBindings** (P0)

- Antes: unmarshal error
- Agora: array bare aceito pelo consumidor; docs audit e docs readiness funcionam pós-adoption
- Evidencia: Compatibilidade producer/consumer.

**BR-03 — contract.update_triggers não chega a AnalyzeImpact** (P0)

- Antes: triggerMatches não removia prefixo path:; 6/6 queries vazias
- Agora: triggers path: são casados; reason das findings cita 'trigger:path:docs/security/'
- Evidencia: Análise de impacto operational.

**BR-04 — context compiler não alcança docs/*/*.md** (P0)

- Antes: fileItems varria apenas raiz e um nível
- Agora: 25 docs de subdiretórios incluídos por tarefa
- Evidencia: Recursão presente. Discriminação continua ausente (NEW-007).

**BR-05 — prumo trace desconectado do estado real** (P0)

- Antes: grafo mock do M8; goal real não encontrada
- Agora: grafo derivado do projeto; not-found honesto com nós conhecidos
- Evidencia: Rastreabilidade sem fabricação.

**BR-10 — checkpoint não alimenta resume real** (P0)

- Antes: checkpoint sem Messages/ToolQ; resume forçava conclusão
- Agora: checkpoint persiste messages, tool_queue, after_side_effects, turns_done; resume executa RunUntilDone (12 passos) e cria checkpoint r3
- Evidencia: Recuperação funcional.

### 4.2 PARTIALLY_FIXED (5)

**BUG-013 — Templates de scaffold contradizem profile e locale** (P1)

- Antes: docs em português para projeto com source_locale=en; doc fixava 85% de cobertura contra 80% do profile
- Agora: A contradição 85%/80% desapareceu: prumo.json gerado não declara quality.coverage, enquanto docs/development/testing-strategy.md e docs/product/scope.md ainda afirma 85%. Todos os 23 docs gerados continuam exclusivamente em português, sem template por locale.
- Evidencia: Metade corrigida (contradição removida); metade pendente (localização e parametrização por domínio).

**BUG-016 — Adapters triplicam 5,1 MB byte-idênticos** (P2)

- Antes: 224 arquivos de skills idênticos, 5,1 MB de 5,4 MB
- Agora: Total de skills caiu para 12,7 KiB, mas por truncamento e não por deduplicação: skills emitidas têm 167 bytes contra 2997 bytes da fonte, e apenas SKILL.md é emitido (sem references/scripts/checks/templates/manifest.json). claude-code e codex continuam byte-idênticos em 18/18 arquivos.
- Evidencia: Volume eliminado por perda de conteúdo, não por arquitetura de distribuição. Ver NEW-003.

**BUG-018 — docs contradictions só detecta números de porta** (P2)

- Antes: regex fixa em 4 arquivos; FM-1 (SQLite vs PostgreSQL) não detectado
- Agora: Detector genérico FUNCIONA para claims config-style em docs/*.md e *.json: listen_port 3000 vs 8080 => DOC_CONTRADICTION. NÃO funciona para prosa, .yaml/.yml (walkAllDocs aceita apenas .md/.json) nem contradições semânticas (SQLite vs PostgreSQL => 0 findings).
- Evidencia: Generalização com perda de recall em prosa.
- Nota de regressao: O detector antigo detectava portas em prosa; o novo perdeu esse caso.

**BUG-019 — Evidence registra término do loop, não atingimento do objetivo** (P2)

- Antes: evidence sem Goal, plano ou teste; summary 'run complete (max turns reached)'
- Agora: Registro agora satisfaz evidence.schema.json (id, type, producer, timestamp, status, goal_id, run_id, confidence, metadata). Mantém o defeito semântico: status 'passed' com stop_reason 'max turns reached' e workspace inalterado; goal_id recebe o texto do goal em vez do id; nenhum vínculo com critério de aceite.
- Evidencia: Forma correta, conteúdo ainda não prova nada.

**BR-06 — Goal não alimenta agent run / evidência** (P1)

- Antes: agent run recebia texto e não atualizava evidence da meta
- Agora: Caminho direto: agent run com --goal casando objective => .ai/goals/P00-G01.goal.json recebe 1 entrada de evidência (artifact .prumo/runtime/harness/evidence-R-agent-1.json). Caminho de resume: checkpoint não persiste state.goal => evidência sai com goal_id 'unassigned' e sem vínculo (NEW-009).
- Evidencia: Ligação funciona no run direto e se perde no resume.

### 4.3 STILL_BROKEN (4)

**DOC-001 — Conjunto documental não cobre o mínimo profissional de um SaaS** (P2)

- Antes: ausência de contratos de API, modelo de dados, tenancy e filas
- Agora: Busca por api/data-model/tenancy/queues/billing/webhooks em docs/ => ABSENT em todos. Árvore gerada: architecture, contracts, development, governance, operations, product, profiles.
- Evidencia: Nenhuma mudança; o scaffold continua sendo genérico.

**BR-07 — plan blueprint não executa tasks** (P1)

- Antes: DAG válida sem componente que despeje tasks no executor
- Agora: blueprint produz T-1-security-trust e T-2-review (dependência real); `prumo run` — anunciado como 'Execute the autonomous goal implementation loop' — aceita apenas --json, sem --goal/--task/--plan, e produz apenas 'Created Run R-bootstrap' com status 'created' e uma continuation de texto genérico. Nenhuma busca por 'task'/'dag'/'blueprint' no harness.
- Evidencia: Não existe ponte Plan->Executor. NEW-006 detailha o stub.

**BR-08 — workforce resolution não chega ao runtime** (P1)

- Antes: 12 agentes resolvidos estaticamente; runner sem papéis/permissões por tarefa
- Agora: Resolução é por profile e não por tarefa; para a tarefa 'Harden authentication: add MFA...' o perfil de segurança não resolve nenhum agente de segurança (NEW-010). No run observado, todas as decisões de permissão foram 'allow' por default policy, idênticas para qualquer objetivo, e o context manifest não referencia agentes.
- Evidencia: Nenhuma especialização por tarefa observável no runtime.

**BR-09 — Evidence não governa Gates** (P1)

- Antes: evidence registrava só término do loop; sem vínculo com aceite
- Agora: Bloqueio duro: após agent run + 4 transições, `goal state DONE` => exit 1 'A goal cannot be marked DONE: ev-run-17909169...: no such evidence record'. Causa: AppendRunEvidence grava id 'ev-run-<unixnano>' no array evidence da meta, e loadEvidenceRecords indexa por record.ID ('ev-R-agent-1-complete'). Além disso o gate exige record.GoalID == goal.id enquanto o harness grava o texto do goal.
- Evidencia: Happy path não pode ser concluído; regressão de gate.

### 4.4 REGRESSED (0)

## 5. Findings novos

| ID | Sev. | Finding | Impacto |
|---|---|---|---|
| NEW-001 | P0 | goal amend re-legitimiza conteúdo adulterado (bypass de integridade) | Anula a garantia de integridade que sustentava a nota 9.0 de Goal Lifecycle. Um humano pode editar a meta em disco e torná-la legitimamente travada. |
| NEW-002 | P0 | Gate DONE inalcançável: id de evidência diverge do registro | Nenhuma meta pode ser concluída pelo caminho do harness; o gate de aceite existe mas nunca aprova. |
| NEW-003 | P1 | Skills embutidas inalcançáveis: prefixo 'workforce' duplicado | Todo adapter compilado com binário instalado recebe skills stubs não funcionais; o payload real (references/scripts/checks/templates/manifest.json) nu |
| NEW-004 | P1 | init imprime 'prumo compile --all', flag inexistente | Primeira acción sugerida ao usuário falha; 3 locaisaffected. |
| NEW-005 | P1 | Projeto recém-criado falha na própria docs verify | Nenhum projeto novo passa na verificação documental; o gate é inalcançável por construção. |
| NEW-006 | P1 | prumo run é stub que se apresenta como loop autônomo | O comando que representa a execução autônoma não executa nada e reporta sucesso; é a maior false bridge remanescente. |
| NEW-007 | P2 | LPC permanece independente da tarefa | Noise ratio ~69% (25 docs genéricos de 36 itens); domain recall baixo; o arquivo-alvo do goal pontua abaixo de README de diretório. |
| NEW-008 | P2 | Evidência declara 'passed' sem mudança no workspace | Registro de evidência afirma sucesso sem que nenhum objetivo tenha sido atingido; confidence 'low' não compensa. |
| NEW-009 | P2 | Resume perde a associação com a meta | A correção de BR-06 funciona no run direto e se dissolve no resume. |
| NEW-010 | P2 | Perfil de segurança não resolve nenhum agente de segurança | Especialização por domínio é inconsistente e falha justamente no domínio de maior risco. |
| NEW-011 | P2 | Saída de ferramenta não é persistida em Observations | Observation não é auditável fora do processo; o backlop do benchmark anterior permanece. |
| NEW-013 | P2 | Docs de operations gerados ficam fora de todo trigger de contrato | O Control Plane cria documentos que ele próprio não governa; alterações em deploy/observabilidade passam sem impacto. |
| NEW-014 | P2 | Contradições e impacto ignoram .yaml/.yml | Contratos e ADRs em YAML — formato usado pelo próprio repositório — são invisíveis ao detector. |
| NEW-012 | P3 | Inferência de project.type inconsistente | Perfil e contratos aplicados variam de forma pouco previsível. |

**NEW-001 — goal amend re-legitimiza conteúdo adulterado (bypass de integridade)** (P0)

- Reproducao: 1) lock P00-G01; 2) editar acceptance no disco para 'TAMPERED: skip all security requirements'; 3) doctor => exit 1 'Lock digest mismatch'; goal state EXECUTING => exit 1 'Illegal goal mutation detected'; 4) goal amend --file => exit 0, revisão 2, digest recalculado = digest adulterado; 5) doctor => exit 0, validate => exit 0, goal state EXECUTING => exit 0.
- Evidencia: Causa raiz: internal/protocol/goals/state.go:237 AmendGoal carrega, muta, recalcula ComputeDigest e salva sem chamar VerifyLock, ao contrário de SetGoalState (state.go:140) que verifica antes de mutar.
- Impacto: Anula a garantia de integridade que sustentava a nota 9.0 de Goal Lifecycle. Um humano pode editar a meta em disco e torná-la legitimamente travada.

**NEW-002 — Gate DONE inalcançável: id de evidência diverge do registro** (P0)

- Reproducao: agent run fake-tools => evidence-R-agent-1.json com id 'ev-R-agent-1-complete'; meta recebe evidence[0].id 'ev-run-1790916709252982619'; loadEvidenceRecords indexa por record.ID; `goal state DONE` => exit 1 'no such evidence record'.
- Evidencia: Causa raiz: internal/protocol/goals/state.go AppendRunEvidence gera id 'ev-run-%d' com time.Now().UnixNano(), incompatível com o id do registro (ev-<runID>-<phase>) que internal/cliops/ops.go:410 usa como chave.
- Impacto: Nenhuma meta pode ser concluída pelo caminho do harness; o gate de aceite existe mas nunca aprova.

**NEW-003 — Skills embutidas inalcançáveis: prefixo 'workforce' duplicado** (P1)

- Reproducao: Compile standalone (fora da árvore): .claude/skills com 16 arquivos e 3412 bytes, cada SKILL.md com 167 bytes. Com PRUMO_REPO_ROOT apontando para a árvore: 119 arquivos e 271152 bytes. Verificação direta do FS embutido: ReadDir('.') => agents, recipes, skills; ReadDir('workforce/skills/architecture-quality') => 'file does not exist'.
- Evidencia: Causa raiz: EmbeddedWorkforce() já é subFS com raiz em src/prumo/resources/workforce, mas internal/cliops/compile.go:306 (copyWorkforceSkill) e :289 (readWorkforceFile) prefixam 'workforce' novamente. Agentes não são afetados porque antigravity.go:79 usa path.Join('agents', ...) sem o prefixo.
- Impacto: Todo adapter compilado com binário instalado recebe skills stubs não funcionais; o payload real (references/scripts/checks/templates/manifest.json) nunca é emitido fora da árvore de fontes.

**NEW-004 — init imprime 'prumo compile --all', flag inexistente** (P1)

- Reproducao: prumo compile --all => exit 2, 'error: --target is required'. String presente em cmd/prumo/main.go:337, cmd/prumo/adopt_commands.go:272 e cmd/prumo/dashboard.go:86. `prumo help compile` documenta apenas --target.
- Evidencia: Reproduzido em projeto recém-inicializado, seguindo exatamente as instruções do próprio init.
- Impacto: Primeira acción sugerida ao usuário falha; 3 locaisaffected.

**NEW-005 — Projeto recém-criado falha na própria docs verify** (P1)

- Reproducao: Em diretório vazio: prumo init => exit 0; prumo docs verify => exit 1 com finding 'version-policy:version-policy-incomplete' em 'docs/lifecycle.json' ('the policy must declare a current version'). init não cria docs/lifecycle.json.
- Evidencia: Causa raiz: internal/doclifecycle/versionpolicy.go:81 exige policy.Current não vazio, mas LifecyclePath (internal/doclifecycle/lifecycle.go:26) nunca é semeado pelo scaffold.
- Impacto: Nenhum projeto novo passa na verificação documental; o gate é inalcançável por construção.

**NEW-006 — prumo run é stub que se apresenta como loop autônomo** (P1)

- Reproducao: prumo run => exit 0, 'Created Run R-bootstrap'; run record {status: created}; events.jsonl com 1 evento run.created; continuation.json com current:['start work'], next_steps:['inspect Goal and repository state']; budget.json com usage vazio. Nenhum artefato de harness, nenhum tool call, workspace inalterado.
- Evidencia: `prumo help run` descreve 'Execute the autonomous goal implementation loop, enforcing Red-Green-Refactor cycles and quality gates', e aceita apenas --json.
- Impacto: O comando que representa a execução autônoma não executa nada e reporta sucesso; é a maior false bridge remanescente.

**NEW-007 — LPC permanece independente da tarefa** (P2)

- Reproducao: 5 objetivos semânticos distintos (MFA/segurança, migração PostgreSQL, dashboard React, REST API v2, billing) => 36-38 itens cada; Jaccard 0.95-1.00; 35 de 37 refs em todas as 5 tarefas; apenas 2 refs discriminam, e nenhum semanticamente correto (testing-strategy em frontend+security; profiles/builtin.json em api+billing+database). Nenhum arquivo src/ incluído em nenhuma tarefa. No caso 'Implement rate limiting in src/limiter.js', src/limiter.js recebeu score 0.5 contra 0.7 de docs genéricos.
- Evidencia: Recalculado a partir dos 5 manifests de contexto persistidos.
- Impacto: Noise ratio ~69% (25 docs genéricos de 36 itens); domain recall baixo; o arquivo-alvo do goal pontua abaixo de README de diretório.

**NEW-008 — Evidência declara 'passed' sem mudança no workspace** (P2)

- Reproducao: agent run --goal 'Implement rate limiting in src/limiter.js' --provider fake-tools => evidence status 'passed', stop_reason 'max turns reached', e src/limiter.js permanece com 'return "TODO"'.
- Evidencia: Artefato evidence-R-agent-1.json vs conteúdo do arquivo.
- Impacto: Registro de evidência afirma sucesso sem que nenhum objetivo tenha sido atingido; confidence 'low' não compensa.

**NEW-009 — Resume perde a associação com a meta** (P2)

- Reproducao: checkpoints r2 e r3 ambos com state.goal = None; após resume, evidence goal_id = 'unassigned' e nenhum vínculo em .ai/goals.
- Evidencia: O texto do goal sobrevive em messages[0].role='user' mas não no campo usado pela ligação de evidência.
- Impacto: A correção de BR-06 funciona no run direto e se dissolve no resume.

**NEW-010 — Perfil de segurança não resolve nenhum agente de segurança** (P2)

- Reproducao: profile features=['security'], risk=critical, quality.security=strict => agents [architect, debugger, documentation-maintainer, explorer, implementer, release-verifier, reviewer, tester]; security-reviewer e security-architect existem no catálogo mas não são selecionados. database=>database-engineer e frontend=>ui-component-engineer são selecionados.
- Evidencia: Comparação de 5 perfis por domínio contra src/prumo/resources/workforce/agents/.
- Impacto: Especialização por domínio é inconsistente e falha justamente no domínio de maior risco.

**NEW-011 — Saída de ferramenta não é persistida em Observations** (P2)

- Reproducao: obs-R-agent-1.jsonl contém apenas {text_delta: 4, tool_call_ready: 4, side_effect_skipped: 3}; nenhum evento carrega o output da ferramenta. O conteúdo lido aparece apenas dentro de messages no checkpoint.
- Evidencia: Contagem de tipos de evento no artefato de observação.
- Impacto: Observation não é auditável fora do processo; o backlop do benchmark anterior permanece.

**NEW-013 — Docs de operations gerados ficam fora de todo trigger de contrato** (P2)

- Reproducao: O scaffold cria docs/operations/deployment.md e docs/operations/observability.md; docs impact --changed docs/operations/deployment.md => 0 contratos. Nenhum dos 34 contratos em docs/contracts/builtin.json possui trigger que mencione 'operations'.
- Evidencia: Varredura dos update_triggers.
- Impacto: O Control Plane cria documentos que ele próprio não governa; alterações em deploy/observabilidade passam sem impacto.

**NEW-014 — Contradições e impacto ignoram .yaml/.yml** (P2)

- Reproducao: walkAllDocs (internal/documentation/impact.go:228) aceita apenas .md e .json. Conflito real em YAML (listen_port: 3000 vs 8080 em dois .yaml) => docs contradictions retorna 0 findings.
- Evidencia: Teste com gateway-a.yaml e gateway-b.yaml sob docs/architecture/.
- Impacto: Contratos e ADRs em YAML — formato usado pelo próprio repositório — são invisíveis ao detector.

**NEW-012 — Inferência de project.type inconsistente** (P3)

- Reproducao: Projeto React (package.json + src/App.tsx) => type ['cli','web']. Projeto full-stack (go.mod + package.json + docker-compose + src/api + src/web) => type ['cli'] apenas.
- Evidencia: Dois projetos de mesma natureza, resultados distintos.
- Impacto: Perfil e contratos aplicados variam de forma pouco previsível.

## 6. End-to-End Engineering Loop

Nota: **2.0 -> 5.0 (+3.0)** — avaliada exclusivamente na cadeia, independente da contagem de features.

| Estagio | Antes | Agora | Evidencia |
|---|:---:|:---:|---|
| Intent -> Questions | FAIL | **PASS** | plan resume semeou 8 questions de docs readiness sem edição manual de JSON. |
| Questions -> Decisions | FAIL | **PASS** | 2 respostas resolveram perguntas e geraram decisões [accepted]. |
| Documentation | FAIL | **PASS** | docs audit/authority/readiness/verify executam; verify falha por arquivo não gerado (NEW-005). |
| Governance (Goal lock) | PASS | **PARTIAL** | Digest SHA-256 funciona e bloqueia adulteração, mas amend a re-legitimiza (NEW-001). |
| Planning (DAG) | PARTIAL | **PARTIAL** | Blueprint produz DAG válida; nenhum consumidor no harness (BR-07). |
| Workforce | PARTIAL | **PARTIAL** | Resolução por profile parcial; não chega ao runtime de forma especializada (NEW-010). |
| LPC | FAIL | **FAIL** | Overlap 0.95-1.00 entre tarefas; nenhum arquivo de código incluído (NEW-007). |
| Execution | FAIL | **FAIL** | agent run executa ferramentas mas não altera o workspace; prumo run é stub (NEW-006). |
| Observation | FAIL | **PARTIAL** | ToolCall e permissões persistidos; saída da ferramenta ausente do obs-*.jsonl (NEW-011). |
| Evidence | FAIL | **PARTIAL** | Registro válido e rico, porém status passed sem mudança no workspace (NEW-008). |
| Gate | PARTIAL | **FAIL** | DONE inalcançável por id mismatch (NEW-002). |
| Trace | FAIL | **PASS** | Grafo real; not-found honesto. |
| Resume | FAIL | **PASS** | 12 mensagens e tool queue restauradas; 12 passos executados. |

Estagios PASS 5 / PARTIAL 5 / FAIL 3 de 13.

## 7. False Bridges

| Bridge | Antes | Agora | Evidencia |
|---|:---:|:---:|---|
| docs readiness -> plan questions | BROKEN | **WORKING** | 8 blocking_questions propagadas para open_questions na sessão; 2 respostas as resolveram. |
| adoption -> bindings | BROKEN | **WORKING** | bindings.json array bare aceito por docs audit e docs readiness. |
| update_triggers -> impact | BROKEN | **WORKING** | Triggers path: casados; sem falsos positivos em 4 arquivos não-documentais. |
| docs -> LPC | BROKEN | **PARTIAL** | Recursão alcança docs/*/*.md (25 docs), mas a seleção é task-independent (NEW-007). |
| project state -> trace | BROKEN | **WORKING** | Nós reais; not-found honesto com lista de nós conhecidos. |
| Goal -> agent | BROKEN | **PARTIAL** | Evidência entra no array da meta no run direto; perdida no resume (NEW-009). |
| Plan -> executor | BROKEN | **BROKEN** | prumo run é stub que não aceita task/plan e não executa nada (NEW-006). |
| workforce -> runtime | BROKEN | **BROKEN** | Sem especialização por tarefa; perfil de segurança sem agente de segurança (NEW-010); permissões idênticas em qualquer objetivo. |
| evidence -> gates | BROKEN | **BROKEN** | Gate bloqueia permanentemente por divergência de id (NEW-002). |
| checkpoint -> resume | BROKEN | **WORKING** | 12 mensagens e tool queue restauradas; 12 passos executados. |
| tools -> observations | BROKEN | **PARTIAL** | ToolCall e permissão persistidos; saída da ferramenta não persistida (NEW-011). |
| init -> onboarding | BROKEN | **PARTIAL** | Init funciona e semeia o Control Plane, mas o next-step impresso é inválido (NEW-004) e o projeto novo falha na docs verify (NEW-005). |

Working 5 / Partial 4 / Broken 3 de 12.

## 8. Failure Modes

| ID | Injetado | Antes | Agora | Nota |
|---|---|:---:|:---:|---|
| FM-1 | Documento canônico contraditório (SQLite-only vs PostgreSQL) | False | **False** | Detecção genérica funciona para claims config-style (listen_port 3000 vs 8080 => 1 finding) mas não para semântica nem para prosa. |
| FM-2 | Meta bloqueada alterada manualmente | True | **True** | doctor exit 1 e goal state EXECUTING exit 1. MAS amend re-legitimiza (NEW-001). |
| FM-3 | Plano com dependência cíclica | True | **True** | Não re-testado diretamente nesta rodada; nenhum caminho de código relacionado foi alterado desde o benchmark anterior. |
| FM-4 | Schema inválido no manifesto | True | **True** | Reproduzido indiretamente: goal com acceptance:string fez validate => exit 1 '<root>.acceptance: expected type array'. |
| FM-5 | Adapter modificado manualmente | False | **True** | Melhorado: conteúdo humano fora da região gerenciada sobrevive ao recompile (md5 estável) em vez de ser apagado. Detecção formal continua ausente. |
| FM-6 | Informação requerida ausente | True | **True** | docs verify detecta fonte de binding removida (binding:missing-source). |
| FM-7 | Decisão inferida sem confirmação | True | **True** | agent-suggestion não promovido a decisão; pergunta permanece aberta. |
| FM-8 | Arquivo documental vinculado removido | False | **True** | 8 findings => 9 findings com binding:missing-source explícito. |
| FM-9 | Estado parcialmente corrompido | PARTIAL | **PARTIAL** | goal list mostra CORRUPT + warning; doctor exit 1 com 2 erros. Executado com sucesso. |
| FM-10 | Execução interrompida e retomada | PARTIAL | **True** | Resume real: 12 mensagens/1 tool_queue restauradas, 12 passos, checkpoint r3. |
| FM-11 | Evidence falsa | False | **False** | NOVO: evidence com status 'passed' e workspace inalterado não é rejeitada por nenhum comando (NEW-008). |
| FM-12 | tool failure | False | **True** | Runtime emite tool.failed quando res.ExitCode != 0; side_effect_skipped é registrado com motivo. |
| FM-13 | timeout | False | **True** | Run com max-turns reporta stop_reason 'max turns reached' e evidence status 'incomplete' quando phase=yield. |
| FM-14 | Alteração concorrente no workspace | False | **False** | Não exercitado; side-effects journal usa idempotency key e reportou 'already recorded as applied', o que sugere rudimentary support. |
| FM-15 | Contexto documental removido | False | **False** | Remover threat-model.md altera o manifest de contexto, mas não há sinal explícito de staleness no manifest. |
| FM-16 | Stale projection | False | **False** | docs authority reporta findings: [] com 26 canônicos + 1 projeção; não valida conteúdo de projeção. |
| FM-17 | Bindings inválido | True | **True** | binding:missing-source para fonte inexistente; tipo de payload inválido não foi exercitado. |
| FM-18 | Goal sem acceptance evidence | True | **True** | Gate bloqueia DONE — mas por divergência de id (NEW-002), o que é um bloqueio por bug e não poranker. |

Detectados 13 / nao detectados 5 de 18.

## 9. Promise vs Reality

| Promessa | Antes | Agora | Evidencia |
|---|:---:|:---:|---|
| Zero dependências de runtime; assets embutidos no binário | NÃO CONFIRMADA | **PARCIALMENTE CONFIRMADA** | schemas, contracts, profiles, catalog e adapters embutidos funcionam fora da árvore (validate/doctor/init exit 0). Skills NÃO: o fallback embutido usa prefixo d |
| Init de zero argumentos | DIVERGENTE | **CONFIRMADA** | exit 0 com detecção de stack. |
| Integridade de Goal com digest SHA-256 | CONFIRMADA | **NÃO CONFIRMADA** | Detecção de adulteração funciona, porém amend re-legitimiza conteúdo adulterado (NEW-001). Regressão da promessa central. |
| Nenhuma sugestão de agente promovida a decisão | CONFIRMADA | **CONFIRMADA** | agent-suggestion registrado como proposed. |
| Documentação como projeção governada, jamais inventada | NÃO CONFIRMADA | **CONFIRMADA** | trace not-found honesto; nenhum nó fabricado. |
| Documentation Impact Analysis | NÃO CONFIRMADA | **CONFIRMADA** | True positives corretos e ausência de falsos positivos em 4 controles. |
| Living Plan de ponta a ponta sem megaprompt | DIVERGENTE | **PARCIALMENTE CONFIRMADA** | Ingestão e resolução de perguntas funcionam sem edição manual; blueprint e execução não se conectam. |
| Regiões humanas preservadas em projeções | NÃO CONFIRMADA | **CONFIRMADA** | md5 estável após recompile com conteúdo humano appended. |
| Recuperação e retomada de execução | NÃO CONFIRMADA | **CONFIRMADA** | Resume executa RunUntilDone com estado completo. |
| Lean Progressive Context: smallest sufficient context | PARCIALMENTE CONFIRMADA | **NÃO CONFIRMADA** | 5 tarefas distintas com overlap 0.95-1.00 e nenhum arquivo de código incluído. |
| Rastreabilidade Requirement -> Claim -> Evidence | NÃO CONFIRMADA | **PARCIALMENTE CONFIRMADA** | Grafo real, porém sem arestas de evidência porque o gate nunca conclui. |
| Evidence over assertion | PARCIALMENTE CONFIRMADA | **NÃO CONFIRMADA** | Evidence declara passed sem mudança no workspace (NEW-008) e o gate é inalcançável (NEW-002). |
| Adoption Engine para brownfield | PARCIALMENTE CONFIRMADA | **PARCIALMENTE CONFIRMADA** | Scan e apply funcionam e bindings são consumíveis; continua sem contrato de tenancy/API. |
| Documentação por audiência | PARCIALMENTE CONFIRMADA | **PARCIALMENTE CONFIRMADA** | Estrutura por audiência mantida, conteúdo monolítico em português. |
| Governança de repositório (branch, commit, PR risk) | NÃO CONFIRMADA | **CONFIRMADA** | policy.json semeado no init; repo policy check funcional; help alinhado ao parser. |
| Loop autônomo de implementação (Red-Green-Refactor + quality gates) | NÃO CONFIRMADA | **NÃO CONFIRMADA** | prumo run é stub que cria registro e sai com exit 0 (NEW-006). |

Confirmadas 7/16 (antes: 2); nao confirmadas 4.

## 10. Documentacao

Media 4.77 -> 4.96 (+0.1900000000000004) em 21 documentos com nota nos dois lados; 24 linhas no total.

| Documento | Tipo | Antes | Agora | Delta | Nota |
|---|---|---:|---:|---:|---|
| `prumo.json` | json-canonical | 8.2 | **8.8** | +0.6 | Versão coerente com o CLI; type inferido de forma inconsistente (NEW-012); sem claims de coverage que contradigam docs. |
| `docs/architecture/overview.md` | markdown-canonical | 3.5 | **3.5** | +0.0 | Genérico; sem topologia do domínio. |
| `docs/architecture/adr/001-architecture-baseline.md` | markdown-canonical | 4.0 | **4.0** | +0.0 | Genérico. |
| `docs/architecture/clean-code-contract.md` | markdown-canonical | 4.7 | **4.7** | +0.0 | Genérico. |
| `docs/development/testing-strategy.md` | markdown-canonical | 5.2 | **4.8** | -0.4 | Afirma 85% de cobertura sem lastro no profile; português apenas. |
| `docs/development/coding-standards.md` | markdown-canonical | 4.8 | **4.8** | +0.0 | Genérico. |
| `docs/governance/repository-governance.md` | markdown-canonical | 4.3 | **4.3** | +0.0 | Genérico. |
| `docs/operations/deployment.md` | markdown-canonical | 2.9 | **2.9** | +0.0 | Genérico e fora de todo trigger de contrato (NEW-013). |
| `docs/operations/observability.md` | markdown-canonical | 2.9 | **2.9** | +0.0 | Genérico e fora de todo trigger. |
| `docs/product/scope.md` | markdown-canonical | 3.0 | **3.0** | +0.0 | Genérico; afirma 85% sem lastro. |
| `docs/product/vision.md` | markdown-canonical | 3.2 | **3.2** | +0.0 | Genérico. |
| `docs/PRUMO.md` | markdown-router | 5.3 | **5.8** | +0.5 | Melhorou por servir de fonte para o seed de perguntas. |
| `docs/README.md` | markdown-index | 4.6 | **4.6** | +0.0 | Genérico. |
| `docs/security/security-contract.md` | markdown-canonical | 4.4 | **4.4** | +0.0 | Genérico; não menciona tenancy/RBAC. |
| `docs/security/threat-model.md` | markdown-canonical | 4.0 | **4.0** | +0.0 | Genérico. |
| `docs/contracts/builtin.json` | json-canonical | n/a | **7.0** | novo | NOVO nesta rodada: 34 contratos semeados no init; ver NEW-013 (sem trigger de operations). |
| `docs/contracts/bindings.json` | json-canonical | 2.9 | **7.5** | +4.6 | Array bare consumido sem erro; binding-integrity detecta fonte removida. |
| `docs/profiles/builtin.json` | json-canonical | n/a | **7.0** | novo | NOVO: separa core-software de cli (BUG-011). |
| `docs/AUTHORITY_MAP.json` | json-canonical | 5.0 | **5.5** | +0.5 | Semeado no init. |
| `ENTRYPOINT.md` | markdown-router | 6.7 | **6.7** | +0.0 | Score 0.9 no manifest de contexto em todas as tarefas. |
| `PROJECT_STATE.md` | markdown-canonical | 6.3 | **6.3** | +0.0 | Genérico. |
| `.ai/goals/P00-G01.goal.json` | json-canonical | 8.6 | **5.0** | -3.6 | Lock e histórico functioning, mas vulnerável a laundering via amend (NEW-001) e a gate inalcançável (NEW-002). |
| `.prumo/repository/policy.json` | json-canonical | n/a | **8.0** | novo | NOVO: semeado no init e validado por repo policy check. |
| `AGENTS.md / CLAUDE.md` | agent-projection | 5.6 | **7.5** | +1.9 | Regiões gerenciadas preservam escrita humana e recompilação é idempotente. |

## 11. Maturity gates

| Eixo | Classificacao | Justificativa reproduzivel |
|---|:---:|---|
| Document System | **Beta** | docs audit/authority/readiness/impact/verify/verify --strict executam com dados reais e sem falsos positivos de impacto. Não é Production-Capable: projeto recém-criado falha na própria verificação (NEW-005), docs/operations/* não é coberto por trigger (NEW-013 |
| Governance | **Alpha** | Verificação de lock SHA-256 é genuína e原本 blindada contra edição direta, e autoridade de decisão é respeitada (agent-suggestion nunca promovido). Não é Beta porque existe um caminho P0 que contorna a garantia central: goal amend re-legitimiza conteúdo adultera |
| Agent Runtime | **Alpha** | A máquina de fases executa, ferramentas são chamadas com permissões e fingerprint, checkpoints e resume funcionam de verdade. Não é Beta porque nenhuma execução_available altera o workspace, prumo run é stub, e o único provider observável em CI é fake/fake-too |
| End-to-End Harness | **Prototype** | A cadeia quebra em três pontos consecutivos: LPC não discrimina tarefa, Execution não modifica o workspace e Gate é inalcançável. Um harness cujo happy path não chega a DONE não é Alpha. |
| Developer UX | **Beta** | init zero-arg funciona, autodetecção de stack funciona, help é consistente com o parser, exit codes são corretos. Não é Production-Capable porque o próprio next-step impresso por init é um comando inválido (NEW-004). |

## 12. Conclusao — respostas objetivas

**1. Quanto o Prumo melhorou?** Nota composta 4.18 -> 6.26 (+2.08); Health Matrix 5.0 -> 6.68. A melhoria e real e concentrado em子系统 deterministas (portabilidade, trace, resume, init, help).

**2. Quantos findings foram realmente corrigidos?** 23 de 32 findings anteriores, cada um re-testado com comando e observed behavior.

**3. Quantos ficaram parciais?** 5: BUG-013, BUG-016, BUG-018, BUG-019, BR-06.

**4. Quantos permanecem?** 4: DOC-001, BR-07, BR-08, BR-09.

**5. Quantos regrediram?** 0 findings anteriores. 4 subsistemas pontuais perderam nota (Goal Lifecycle / Locks, Plan / Task DAG, Workforce Resolution, Gates), porem por causa de achados novos e nao por perda de correcao anterior.

**6. Quantos bugs novos?** 14 findings novos, sendo 2 P0, 4 P1, 7 P2, 1 P3.

**7. O end-to-end funciona?** Nao. A cadeia quebra em 3 de 13 estagios: LPC, Execution e Gate. Nota 2.0 -> 5.0.

**8. O Agent Runtime executa trabalho real?** Parcialmente. Ferramentas sao-called com permissoes e fingerprint, checkpoints e resume funcionam; nenhuma execucao alterou o workspace, e `prumo run` e stub.

**9. Evidence prova acceptance criteria?** Nao. O registro satisfaz o schema, mas declara status 'passed' com stop_reason 'max turns reached' e workspace inalterado, e nao referencia criterio de aceite.

**10. Gates governam conclusao?** Nao de forma utilizavel. O gate DONE e inalcancavel pelo caminho real por divergencia de id (NEW-002).

**11. Trace representa estado real?** Sim. Nenhum no fabrication; not-found honesto com lista de nos conhecidos.

**12. Resume continua execucao?** Sim. 12 mensagens e tool queue restauradas, 12 passos executados, checkpoint r3. Perde apenas a associacao com a meta (NEW-009).

**13. LPC seleciona contexto dependente da tarefa?** Nao. 5 tarefas semanticas distintas deram overlap Jaccard 0.95-1.00 e nenhum arquivo de codigo incluido.

**14. A documentacao ficou substancialmente melhor?** Pouco em conteudo: media 4.77 -> 4.96. Muito em governanca: o Control Plane e semeado, audit/authority/impact/verify operam sobre dados reais. O conjunto documental continua generico e monolitico.

**15. Pode ser usado de forma confiavel em projeto real?** Nao ainda. O bootstrap e o tracking documental sao utilizaveis; conclusion automatica de meta, execucao autonoma e contexto por tarefa nao sao.

**16. Cinco maiores bloqueadores** NEW-001 (goal amend re-legitimiza conteúdo adulterado (bypass de integridade)); NEW-002 (Gate DONE inalcançável: id de evidência diverge do registro); BR-06 (Goal não alimenta agent run / evidência); BR-07 (plan blueprint não executa tasks); BR-08 (workforce resolution não chega ao runtime).

## 13. Backlog repriorizado

### P0 (2)

- [ ] **NEW-001** goal amend re-legitimiza conteúdo adulterado (bypass de integridade)
- [ ] **NEW-002** Gate DONE inalcançável: id de evidência diverge do registro

### P1 (9)

- [ ] **BR-06** Goal não alimenta agent run / evidência
- [ ] **BR-07** plan blueprint não executa tasks
- [ ] **BR-08** workforce resolution não chega ao runtime
- [ ] **BR-09** Evidence não governa Gates
- [ ] **BUG-013** Templates de scaffold contradizem profile e locale
- [ ] **NEW-003** Skills embutidas inalcançáveis: prefixo 'workforce' duplicado
- [ ] **NEW-004** init imprime 'prumo compile --all', flag inexistente
- [ ] **NEW-005** Projeto recém-criado falha na própria docs verify
- [ ] **NEW-006** prumo run é stub que se apresenta como loop autônomo

### P2 (11)

- [ ] **BUG-016** Adapters triplicam 5,1 MB byte-idênticos
- [ ] **BUG-018** docs contradictions só detecta números de porta
- [ ] **BUG-019** Evidence registra término do loop, não atingimento do objetivo
- [ ] **DOC-001** Conjunto documental não cobre o mínimo profissional de um SaaS
- [ ] **NEW-007** LPC permanece independente da tarefa
- [ ] **NEW-008** Evidência declara 'passed' sem mudança no workspace
- [ ] **NEW-009** Resume perde a associação com a meta
- [ ] **NEW-010** Perfil de segurança não resolve nenhum agente de segurança
- [ ] **NEW-011** Saída de ferramenta não é persistida em Observations
- [ ] **NEW-013** Docs de operations gerados ficam fora de todo trigger de contrato
- [ ] **NEW-014** Contradições e impacto ignoram .yaml/.yml

### P3 (1)

- [ ] **NEW-012** Inferência de project.type inconsistente

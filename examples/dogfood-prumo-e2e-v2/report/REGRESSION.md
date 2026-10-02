# Prumo — Relatorio de Regressao

Baseline `807343c1f1da0fbc8f5dceaea17d807e7377e38a` -> medido `dbc14203f258a50a0567a30b141e847c8246af6c`.

**Nota composta: 4.18 -> 6.26 (+2.08)**

## Matriz before -> after

| ID | Finding antigo | Antes | Agora | Status | Evidencia |
|---|---|---:|---:|:---:|---|
| BUG-001 | Schema registry quebra fora da árvore de fontes | P0 | corrigido | **FIXED** | Portabilidade testada fora da árvore de fontes com o binário standalone; schemas embutidos são consumidos. |
| BUG-002 | adopt --apply grava bindings.json incompatível com o engine | P0 | corrigido | **FIXED** | Produtor e consumidor concordam sobre o schema. |
| BUG-003 | Documentation Impact Analyzer inoperante | P0 | corrigido | **FIXED** | Analyzer Typed + detecção de trigger path: functioning com true positives e sem false positives. |
| BUG-004 | LPC não enxerga documentos canônicos em profundidade >= 3 | P0 | corrigido | **FIXED** | Recursão corrigida. Ver NEW-007 para a falha remanescente de discriminação. |
| BUG-005 | prumo trace fabrica grafo com dados do framework | P0 | corrigido | **FIXED** | Nenhum nó fabricado; o sistema declara honestamente a ausência de dados. |
| BUG-006 | agent resume é stub e força conclusão sem executar trabalho | P0 | corrigido | **FIXED** | Resume continua trabalho real; a mensagem de role 'tool' no checkpoint contém conteúdo real de arquivo lido. |
| BUG-007 | compile sobrescreve regiões mantidas por humanos | P0 | corrigido | **FIXED** | Preservação de região gerenciada e idempotência confirmadas por hash. |
| BUG-008 | goal amend descarta flags e ignora payload sem 'changes' | P1 | corrigido | **FIXED** | Ambas as metades do finding corrigidas. Ver NEW-001 para a falha nova no mesmo caminho. |
| BUG-009 | init não gera arquivos do Documentation Control Plane | P1 | corrigido | **FIXED** | Control Plane semeado no primeiro uso. |
| BUG-010 | Ajuda de CLI documenta subcomandos inexistentes | P1 | corrigido | **FIXED** | Help e parser alinhados no nível de top-level. |
| BUG-011 | core-software força cli.reference e installation.lifecycle | P1 | corrigido | **FIXED** | Contratos de CLI isolados no perfil cli. Ver NEW-012 para inconsistência de inferência de tipo. |
| BUG-012 | docs verify --strict não detecta documento vinculado removido | P1 | corrigido | **FIXED** | Integridade referencial verificada. |
| BUG-013 | Templates de scaffold contradizem profile e locale | P1 | parcial | **PARTIALLY_FIXED** | Metade corrigida (contradição removida); metade pendente (localização e parametrização por domínio). |
| BUG-014 | Versões divergentes entre artefatos gerados | P2 | corrigido | **FIXED** | Versão única e coerente. |
| BUG-015 | Adapter Antigravity perde agentes e emite stubs | P2 | corrigido | **FIXED** | Contratos reais entregues. |
| BUG-016 | Adapters triplicam 5,1 MB byte-idênticos | P2 | parcial | **PARTIALLY_FIXED** | Volume eliminado por perda de conteúdo, não por arquitetura de distribuição. Ver NEW-003. |
| BUG-017 | goal list engole silenciosamente meta corrompida | P2 | corrigido | **FIXED** | Corrupção visível e diagnosticada. |
| BUG-018 | docs contradictions só detecta números de porta | P2 | parcial | **PARTIALLY_FIXED** | Generalização com perda de recall em prosa. |
| BUG-019 | Evidence registra término do loop, não atingimento do objetivo | P2 | parcial | **PARTIALLY_FIXED** | Forma correta, conteúdo ainda não prova nada. |
| UX-001 | Zero-argument init e modo interativo não existem | P2 | corrigido | **FIXED** | Bootstrap sem atrito; main implementa via detecção + preset (não via prompt interativo). |
| UX-002 | Ajuda de repo e bootstrap de policy.json | P2 | corrigido | **FIXED** | Bootstrap e comando presentes. |
| DOC-001 | Conjunto documental não cobre o mínimo profissional de um SaaS | P2 | presente | **STILL_BROKEN** | Nenhuma mudança; o scaffold continua sendo genérico. |
| BR-01 | docs readiness não alimenta plan questions | P0 | corrigido | **FIXED** | Ponte成本中transformada em intake real. |
| BR-02 | adopt --apply incompatível com docengine.LoadBindings | P0 | corrigido | **FIXED** | Compatibilidade producer/consumer. |
| BR-03 | contract.update_triggers não chega a AnalyzeImpact | P0 | corrigido | **FIXED** | Análise de impacto operational. |
| BR-04 | context compiler não alcança docs/*/*.md | P0 | corrigido | **FIXED** | Recursão presente. Discriminação continua ausente (NEW-007). |
| BR-05 | prumo trace desconectado do estado real | P0 | corrigido | **FIXED** | Rastreabilidade sem fabricação. |
| BR-06 | Goal não alimenta agent run / evidência | P1 | parcial | **PARTIALLY_FIXED** | Ligação funciona no run direto e se perde no resume. |
| BR-07 | plan blueprint não executa tasks | P1 | presente | **STILL_BROKEN** | Não existe ponte Plan->Executor. NEW-006 detailha o stub. |
| BR-08 | workforce resolution não chega ao runtime | P1 | presente | **STILL_BROKEN** | Nenhuma especialização por tarefa observável no runtime. |
| BR-09 | Evidence não governa Gates | P1 | presente | **STILL_BROKEN** | Happy path não pode ser concluído; regressão de gate. |
| BR-10 | checkpoint não alimenta resume real | P0 | corrigido | **FIXED** | Recuperação funcional. |

## Subsistemas que regrediram

| Subsistema | Antes | Agora | Motivo |
|---|---:|---:|---|
| Goal Lifecycle / Locks | 9.0 | 5.0 | A garantia central de integridade é contornável pelo comando de amendment. |
| Plan / Task DAG | 7.5 | 7.0 | Pontuação reduzida por ausência de execução, não por falha do validador. |
| Workforce Resolution | 7.0 | 6.5 | Nenhuma. |
| Gates | 6.5 | 3.5 | Pontuação caiu: o gate passou de 'existe mas não é consumido' para 'bloqueia permanentemente o happy path'. |

## Resumo numerico

- Findings anteriores re-testados: **32**
  - FIXED: **23**
  - PARTIALLY_FIXED: **5**
  - STILL_BROKEN: **4**
- Findings novos: **14**
- Severidade atual: P0=15, P1=14, P2=16, P3=1
- Bridges: working 5, partial 4, broken 3 (de 12)
- Failure modes detectados: 13 de 18
- End-to-End: 5 PASS / 5 PARTIAL / 3 FAIL
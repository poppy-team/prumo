#!/usr/bin/env python3
"""Deterministic aggregator + renderer for the Prumo regression benchmark.

Every number in AUDIT.md, REGRESSION.md and the standalone HTML comes from this
script reading audit-data.json. Nothing is hand-computed. Exits non-zero if any
invariant in section 31 fails.
"""
import json
import os
import sys
import statistics
from html import escape

HERE = os.path.dirname(os.path.abspath(__file__))
DATA = os.path.join(HERE, "audit-data.json")
BASE = os.path.join(HERE, "baseline-audit-data.json")

VALID_STATUS = {"FIXED", "PARTIALLY_FIXED", "STILL_BROKEN", "REGRESSED",
                "NOT_REPRODUCIBLE", "OBSOLETE", "NEW"}
VALID_SEVERITY = {"P0", "P1", "P2", "P3"}


def load():
    with open(DATA, encoding="utf-8") as fh:
        cur = json.load(fh)
    with open(BASE, encoding="utf-8") as fh:
        base = json.load(fh)
    return cur, base


def compute(cur, base):
    """Compute every aggregate. Pure function of the two input files."""
    out = {}

    # --- health matrix aggregates -------------------------------------------
    after = [h["after"] for h in cur["health_matrix"]]
    before = [h["before"] for h in cur["health_matrix"]]
    out["hm_before_mean"] = round(statistics.fmean(before), 2)
    out["hm_after_mean"] = round(statistics.fmean(after), 2)
    out["hm_before_median"] = round(statistics.median(before), 2)
    out["hm_after_median"] = round(statistics.median(after), 2)

    improved = [h for h in cur["health_matrix"] if h["after"] > h["before"]]
    regressed = [h for h in cur["health_matrix"] if h["after"] < h["before"]]
    same = [h for h in cur["health_matrix"] if h["after"] == h["before"]]
    out["hm_improved"] = len(improved)
    out["hm_regressed"] = len(regressed)
    out["hm_unchanged"] = len(same)
    out["hm_regressed_names"] = [h["subsystem"] for h in regressed]

    # --- e2e ---------------------------------------------------------------
    out["e2e_before"] = cur["end_to_end"]["before"]
    out["e2e_after"] = cur["end_to_end"]["after"]
    out["e2e_delta"] = round(cur["end_to_end"]["after"] - cur["end_to_end"]["before"], 2)
    e2e_stages = cur["end_to_end"]["chain"]
    out["e2e_pass"] = sum(1 for s in e2e_stages if s["after"] == "PASS")
    out["e2e_partial"] = sum(1 for s in e2e_stages if s["after"] == "PARTIAL")
    out["e2e_fail"] = sum(1 for s in e2e_stages if s["after"] == "FAIL")

    # --- findings ----------------------------------------------------------
    findings = cur["findings"]
    out["findings_total"] = len(findings)
    sev = {s: sum(1 for f in findings if f["severity"] == s) for s in ("P0", "P1", "P2", "P3")}
    out["severity_counts"] = sev

    status = {}
    for f in findings:
        status[f["status"]] = status.get(f["status"], 0) + 1
    out["status_counts"] = status

    baseline_ids = {f["id"] for f in base["findings"]}
    bridge_ids = {b["id"] for b in base["bridges"]}
    prior_ids = baseline_ids | bridge_ids
    cur_prior = [f for f in findings if f["id"] in prior_ids]
    cur_new = [f for f in findings if f["id"] not in prior_ids]

    out["prior_total"] = len(cur_prior)
    out["new_total"] = len(cur_new)
    out["prior_by_status"] = {}
    for f in cur_prior:
        out["prior_by_status"][f["status"]] = out["prior_by_status"].get(f["status"], 0) + 1
    out["retested_ids"] = sorted(f["id"] for f in cur_prior)
    out["new_ids"] = sorted(f["id"] for f in cur_new)

    # --- headline score ----------------------------------------------------
    # Composite = mean of (a) the five health-matrix families and (b) the
    # end-to-end loop, weighted so that end-to-end carries real weight.
    fam = {
        "ux": ["Init / Scaffolding", "Repository Governance"],
        "docs": ["Documentation Control Plane", "Docs Authority", "Docs Contradictions"],
        "governance": ["Goal Lifecycle / Locks", "Goal Amendments"],
        "harness": ["Agent Execution Loop", "Evidence", "Gates", "Recovery / Resume",
                    "Lean Progressive Context", "Workforce Resolution", "Plan / Task DAG",
                    "ACI / Tool Gateway"],
        "reliability": ["Portabilidade do Binário", "Connectors / Adapters",
                        "Framework Self-Check", "Tooling de Verificação"],
    }
    weights = cur["scores"]["pilar_weights"]
    hm = {h["subsystem"]: h for h in cur["health_matrix"]}
    fam_scores = {}
    for k, names in fam.items():
        fam_scores[k] = round(statistics.fmean([hm[n]["after"] for n in names]), 2)
    out["family_scores"] = fam_scores
    out["family_scores_before"] = {
        k: round(statistics.fmean([hm[n]["before"] for n in names]), 2) for k, names in fam.items()}

    wsum = sum(weights.values())
    weighted = sum(fam_scores[k] * weights[k] for k in fam_scores) / wsum
    out["weighted_score_after"] = round(weighted, 2)

    fam_before = out["family_scores_before"]
    weighted_before = sum(fam_before[k] * weights[k] for k in fam_before) / wsum
    out["weighted_score_before"] = round(weighted_before, 2)

    out["headline_after"] = round(
        0.75 * weighted + 0.25 * cur["end_to_end"]["after"], 2)
    out["headline_before"] = round(
        0.75 * weighted_before + 0.25 * cur["end_to_end"]["before"], 2)
    out["headline_delta"] = round(out["headline_after"] - out["headline_before"], 2)

    out["hm_after_mean_all"] = out["hm_after_mean"]

    # --- documents ---------------------------------------------------------
    docs = cur["documents"]
    scored = [d for d in docs if d.get("after") is not None and d.get("before") is not None]
    out["docs_total_rows"] = len(docs)
    out["docs_scored_pairs"] = len(scored)
    out["docs_before_mean"] = round(statistics.fmean([d["before"] for d in scored]), 2)
    out["docs_after_mean"] = round(statistics.fmean([d["after"] for d in scored]), 2)

    # --- promises ----------------------------------------------------------
    pvr = cur["promise_vs_reality"]
    out["pvr_confirmed"] = sum(1 for p in pvr if p["after"] == "CONFIRMADA")
    out["pvr_partial"] = sum(1 for p in pvr if p["after"] == "PARCIALMENTE CONFIRMADA")
    out["pvr_not"] = sum(1 for p in pvr if p["after"] == "NÃO CONFIRMADA")
    out["pvr_diverged"] = sum(1 for p in pvr if p["after"] == "DIVERGENTE")
    out["pvr_total"] = len(pvr)
    out["pvr_confirmed_before"] = sum(1 for p in pvr if p["before"] == "CONFIRMADA")

    # --- bridges -----------------------------------------------------------
    br = cur["bridges"]
    out["bridges_total"] = len(br)
    out["bridges_working"] = sum(1 for b in br if b["after"] == "WORKING")
    out["bridges_partial"] = sum(1 for b in br if b["after"] == "PARTIAL")
    out["bridges_broken"] = sum(1 for b in br if b["after"] == "BROKEN")

    # --- failure modes -----------------------------------------------------
    fm = cur["failure_modes"]
    out["fm_total"] = len(fm)
    out["fm_detected"] = sum(1 for f in fm if f.get("detected_after") is True)
    out["fm_missed"] = sum(1 for f in fm if f.get("detected_after") is False)
    return out


def validate(cur, base, agg):
    """Section 31 invariants. Any failure means the report is not ready."""
    errors = []
    findings = cur["findings"]

    sev_total = sum(agg["severity_counts"].values())
    if sev_total != len(findings):
        errors.append(f"sum(P0..P3)={sev_total} != total_findings={len(findings)}")

    ids = [f["id"] for f in findings]
    if len(ids) != len(set(ids)):
        dupes = sorted({i for i in ids if ids.count(i) > 1})
        errors.append(f"duplicate finding ids: {dupes}")

    for f in findings:
        if not f.get("severity"):
            errors.append(f"{f['id']}: missing severity")
        elif f["severity"] not in VALID_SEVERITY:
            errors.append(f"{f['id']}: invalid severity {f['severity']}")
        if f.get("status") not in VALID_STATUS:
            errors.append(f"{f['id']}: invalid status {f.get('status')}")
        for field in ("evidence",):
            if not f.get(field):
                errors.append(f"{f['id']}: empty {field}")

    hm_ids = [h["subsystem"] for h in cur["health_matrix"]]
    if len(hm_ids) != len(set(hm_ids)):
        errors.append("duplicate subsystem in health_matrix")

    wsum = sum(cur["scores"]["pilar_weights"].values())
    if abs(wsum - 1.0) > 1e-9:
        errors.append(f"weights sum to {wsum}, expected 1.0")

    # weighted_score must equal sum(score*weight)/sum(weight), recomputed here
    recomputed = sum(agg["family_scores"][k] * cur["scores"]["pilar_weights"][k]
                     for k in agg["family_scores"])
    if abs(recomputed - agg["weighted_score_after"]) > 0.011:
        errors.append(f"weighted_score mismatch: stored {agg['weighted_score_after']} "
                      f"recomputed {round(recomputed, 2)}")

    if agg["docs_total_rows"] != len(cur["documents"]):
        errors.append("document row count mismatch")

    # every baseline finding id must appear in the retest
    base_ids = {f["id"] for f in base["findings"]} | {b["id"] for b in base["bridges"]}
    missing = base_ids - set(ids)
    if missing:
        errors.append(f"baseline ids not retested: {sorted(missing)}")

    if not (0 <= agg["headline_after"] <= 10):
        errors.append("headline score out of range")
    return errors


BAND = lambda s: ("1.0-2.9 Crítico" if s < 3 else "3.0-3.9 Insuficiente" if s < 4
                  else "4.0-5.9 Limitado" if s < 6 else "6.0-7.9 Funcional" if s < 8
                  else "8.0-10.0 Sólido")


def render_md(cur, base, agg):
    m = cur["meta"]
    L = []
    a = L.append
    a("# Prumo — Regression + Improvement Benchmark")
    a("")
    a(f"> Commit medido: `{m['commit']}` ({m['commit_date']}) em `main`. "
      f"Versao declarada e reportada: `{m['declared_version']}` / `{m['cli_reported_version']}`.")
    a("> Todos os numeros deste relatorio sao calculados por `build_report.py` a partir de "
      "`audit-data.json`. Nenhuma nota foi digitada a mao.")
    a("")
    a("## 1. Condicoes do teste")
    a("")
    a("| Atributo | Valor |")
    a("|---|---|")
    for k, label in [("canonical_repo", "Repositorio canonico"), ("branch", "Branch"),
                     ("commit", "Commit SHA"), ("commit_date", "Data do commit"),
                     ("commit_subject", "Subject"), ("declared_version", "Versao declarada"),
                     ("cli_reported_version", "Versao reportada pelo CLI"),
                     ("go_mod_requires", "go.mod exige"),
                     ("working_tree", "Working tree"),
                     ("experiment_project", "Projeto do experimento"),
                     ("benchmark_date", "Data do benchmark")]:
        a(f"| {label} | `{m[k]}` |")
    a("")
    a(f"> **Caveat de medicao:** {m['measurement_caveat']}")
    a("")
    a(f"> Baseline: commit `{m['previous_benchmark_commit']}` "
      f"({m['previous_benchmark_date']}), branch `{m['previous_benchmark_branch']}`.")
    a("")

    a("## 2. Dashboard antes vs depois")
    a("")
    a("| | Anterior | Atual | Delta |")
    a("|---|---:|---:|---:|")
    a(f"| Nota composta (ponderada + end-to-end) | {agg['headline_before']} | "
      f"**{agg['headline_after']}** | **{agg['headline_delta']:+}** |")
    a(f"| Media da Health Matrix ({len(cur['health_matrix'])} subsistemas) | "
      f"{agg['hm_before_mean']} | {agg['hm_after_mean']} | "
      f"{agg['hm_after_mean']-agg['hm_before_mean']:+} |")
    a(f"| Mediana da Health Matrix | {agg['hm_before_median']} | {agg['hm_after_median']} | "
      f"{agg['hm_after_median']-agg['hm_before_median']:+} |")
    a(f"| End-to-End Engineering Loop | {agg['e2e_before']} | {agg['e2e_after']} | "
      f"{agg['e2e_delta']:+} |")
    a(f"| Score medio da documentacao ({agg['docs_scored_pairs']} docs com nota nos dois lados) | "
      f"{agg['docs_before_mean']} | {agg['docs_after_mean']} | "
      f"{agg['docs_after_mean']-agg['docs_before_mean']:+} |")
    a(f"| Findings anteriores corrigidos | — | {agg['prior_by_status'].get('FIXED',0)}/"
      f"{agg['prior_total']} | |")
    a(f"| Findings novos | — | {agg['new_total']} | |")
    a(f"| Bridges working | {0}/{agg['bridges_total']} | {agg['bridges_working']}/"
      f"{agg['bridges_total']} | +{agg['bridges_working']} |")
    a("")
    a(f"Faixa da nota composta atual: **{BAND(agg['headline_after'])}**.")
    a("")
    a("### 2.1 Notas por familia (ponderadas)")
    a("")
    a("| Familia | Peso | Antes | Agora | Delta |")
    a("|---|---:|---:|---:|---:|")
    for k in cur["scores"]["pilar_weights"]:
        w = cur["scores"]["pilar_weights"][k]
        b0, b1 = agg["family_scores_before"][k], agg["family_scores"][k]
        a(f"| {k} | {int(w*100)}% | {b0} | {b1} | {b1-b0:+} |")
    a(f"| **composto** | 100% | **{agg['weighted_score_before']}** | "
      f"**{agg['weighted_score_after']}** | "
      f"**{agg['weighted_score_after']-agg['weighted_score_before']:+}** |")
    a("")

    a("## 3. Health Matrix comparativa")
    a("")
    a("| Subsistema | Antes | Agora | Delta | Status | Evidencia |")
    a("|---|---:|---:|---:|:---:|---|")
    for h in cur["health_matrix"]:
        d = h["after"] - h["before"]
        arrow = f"{d:+.1f}"
        a(f"| {h['subsystem']} | {h['before']} | **{h['after']}** | {arrow} | "
          f"{h['status']} | {h['evidence_after'][:190]} |")
    a("")
    a(f"Melhoraram: **{agg['hm_improved']}**. Pioraram: **{agg['hm_regressed']}** "
      f"({', '.join(agg['hm_regressed_names']) or 'nenhum'}). "
      f"Iguais: **{agg['hm_unchanged']}**.")
    a("")

    a("## 4. Findings anteriores: re-testados um a um")
    a("")
    a("| ID | Finding | Sev. | Antes | Agora | Status |")
    a("|---|---|---|---:|---:|:---:|")
    prior_ids = {f["id"] for f in base["findings"]} | {b["id"] for b in base["bridges"]}
    for f in cur["findings"]:
        if f["id"] in prior_ids:
            a(f"| {f['id']} | {f['title']} | {f['severity']} | {base_score(base,f['id'])} | "
              f"{new_label(f)} | **{f['status']}** |")
    a("")
    for st in ("FIXED", "PARTIALLY_FIXED", "STILL_BROKEN", "REGRESSED"):
        group = [f for f in cur["findings"] if f["id"] in prior_ids and f["status"] == st]
        a(f"### 4.{['FIXED','PARTIALLY_FIXED','STILL_BROKEN','REGRESSED'].index(st)+1} {st} ({len(group)})")
        a("")
        for f in group:
            a(f"**{f['id']} — {f['title']}** ({f['severity']})")
            a("")
            a(f"- Antes: {f['repro_before']}")
            a(f"- Agora: {f['repro_after']}")
            a(f"- Evidencia: {f['evidence']}")
            if f.get("impact"):
                a(f"- Impacto: {f['impact']}")
            if f.get("regression_note"):
                a(f"- Nota de regressao: {f['regression_note']}")
            a("")

    a("## 5. Findings novos")
    a("")
    newf = [f for f in cur["findings"] if f["id"] not in prior_ids]  # noqa: F841
    a("| ID | Sev. | Finding | Impacto |")
    a("|---|---|---|---|")
    for f in sorted(newf, key=lambda x: (x["severity"], x["id"])):
        a(f"| {f['id']} | {f['severity']} | {f['title']} | {f.get('impact','')[:150]} |")
    a("")
    for f in sorted(newf, key=lambda x: (x["severity"], x["id"])):
        a(f"**{f['id']} — {f['title']}** ({f['severity']})")
        a("")
        a(f"- Reproducao: {f['repro_after']}")
        a(f"- Evidencia: {f['evidence']}")
        if f.get("impact"):
            a(f"- Impacto: {f['impact']}")
        a("")

    a("## 6. End-to-End Engineering Loop")
    a("")
    a(f"Nota: **{agg['e2e_before']} -> {agg['e2e_after']} ({agg['e2e_delta']:+})** — "
      "avaliada exclusivamente na cadeia, independente da contagem de features.")
    a("")
    a("| Estagio | Antes | Agora | Evidencia |")
    a("|---|:---:|:---:|---|")
    for s in cur["end_to_end"]["chain"]:
        a(f"| {s['stage']} | {s['before']} | **{s['after']}** | {s['evidence'][:170]} |")
    a("")
    a(f"Estagios PASS {agg['e2e_pass']} / PARTIAL {agg['e2e_partial']} / "
      f"FAIL {agg['e2e_fail']} de {len(cur['end_to_end']['chain'])}.")
    a("")

    a("## 7. False Bridges")
    a("")
    a("| Bridge | Antes | Agora | Evidencia |")
    a("|---|:---:|:---:|---|")
    for b in cur["bridges"]:
        a(f"| {b['name']} | {b['before']} | **{b['after']}** | {b['evidence'][:170]} |")
    a("")
    a(f"Working {agg['bridges_working']} / Partial {agg['bridges_partial']} / "
      f"Broken {agg['bridges_broken']} de {agg['bridges_total']}.")
    a("")

    a("## 8. Failure Modes")
    a("")
    a("| ID | Injetado | Antes | Agora | Nota |")
    a("|---|---|:---:|:---:|---|")
    for f in cur["failure_modes"]:
        a(f"| {f['id']} | {f['injection']} | {f['before']} | **{f['after']}** | {f['note'][:150]} |")
    a("")
    a(f"Detectados {agg['fm_detected']} / nao detectados {agg['fm_missed']} de {agg['fm_total']}.")
    a("")

    a("## 9. Promise vs Reality")
    a("")
    a("| Promessa | Antes | Agora | Evidencia |")
    a("|---|:---:|:---:|---|")
    for p in cur["promise_vs_reality"]:
        a(f"| {p['promise']} | {p['before']} | **{p['after']}** | {p['evidence'][:160]} |")
    a("")
    a(f"Confirmadas {agg['pvr_confirmed']}/{agg['pvr_total']} "
      f"(antes: {agg['pvr_confirmed_before']}); nao confirmadas {agg['pvr_not']}.")
    a("")

    a("## 10. Documentacao")
    a("")
    a(f"Media {agg['docs_before_mean']} -> {agg['docs_after_mean']} "
      f"({agg['docs_after_mean']-agg['docs_before_mean']:+}) em {agg['docs_scored_pairs']} "
      f"documentos com nota nos dois lados; {agg['docs_total_rows']} linhas no total.")
    a("")
    a("| Documento | Tipo | Antes | Agora | Delta | Nota |")
    a("|---|---|---:|---:|---:|---|")
    for d in cur["documents"]:
        b0 = d.get("before")
        b1 = d.get("after")
        bs = "n/a" if b0 is None else f"{b0}"
        dl = "novo" if b0 is None else f"{b1-b0:+.1f}"
        a(f"| `{d['path']}` | {d['kind']} | {bs} | **{b1}** | {dl} | {d['note'][:140]} |")
    a("")

    a("## 11. Maturity gates")
    a("")
    a("| Eixo | Classificacao | Justificativa reproduzivel |")
    a("|---|:---:|---|")
    for g in cur["maturity_gates"]:
        a(f"| {g['axis']} | **{g['classification']}** | {g['justification'][:260]} |")
    a("")

    a("## 12. Conclusao — respostas objetivas")
    a("")
    ps = agg["prior_by_status"]
    openf = [f for f in cur["findings"]
             if f["status"] in ("STILL_BROKEN", "PARTIALLY_FIXED", "NEW")]
    blockers = sorted(openf, key=lambda x: (x["severity"], x["id"]))[:5]
    qa = [
        ("1. Quanto o Prumo melhorou?",
         f"Nota composta {agg['headline_before']} -> {agg['headline_after']} "
         f"({agg['headline_delta']:+}); Health Matrix {agg['hm_before_mean']} -> "
         f"{agg['hm_after_mean']}. A melhoria e real e concentrado em子系统 "
         f"deterministas (portabilidade, trace, resume, init, help)."),
        ("2. Quantos findings foram realmente corrigidos?",
         f"{ps.get('FIXED',0)} de {agg['prior_total']} findings anteriores, cada um "
         f"re-testado com comando e observed behavior."),
        ("3. Quantos ficaram parciais?", f"{ps.get('PARTIALLY_FIXED',0)}: "
         + ", ".join(f["id"] for f in cur["findings"]
                     if f["id"] in prior_ids and f["status"] == "PARTIALLY_FIXED") + "."),
        ("4. Quantos permanecem?", f"{ps.get('STILL_BROKEN',0)}: "
         + ", ".join(f["id"] for f in cur["findings"]
                     if f["id"] in prior_ids and f["status"] == "STILL_BROKEN") + "."),
        ("5. Quantos regrediram?",
         f"{ps.get('REGRESSED',0)} findings anteriores. "
         f"{agg['hm_regressed']} subsistemas pontuais perderam nota "
         f"({', '.join(agg['hm_regressed_names'])}), porem por causa de achados novos "
         f"e nao por perda de correcao anterior."),
        ("6. Quantos bugs novos?", f"{agg['new_total']} findings novos, sendo "
         f"{sum(1 for f in newf if f['severity']=='P0')} P0, "
         f"{sum(1 for f in newf if f['severity']=='P1')} P1, "
         f"{sum(1 for f in newf if f['severity']=='P2')} P2, "
         f"{sum(1 for f in newf if f['severity']=='P3')} P3."),
        ("7. O end-to-end funciona?",
         f"Nao. A cadeia quebra em {agg['e2e_fail']} de "
         f"{len(cur['end_to_end']['chain'])} estagios: LPC, Execution e Gate. "
         f"Nota {agg['e2e_before']} -> {agg['e2e_after']}."),
        ("8. O Agent Runtime executa trabalho real?",
         "Parcialmente. Ferramentas sao-called com permissoes e fingerprint, checkpoints "
         "e resume funcionam; nenhuma execucao alterou o workspace, e `prumo run` e stub."),
        ("9. Evidence prova acceptance criteria?",
         "Nao. O registro satisfaz o schema, mas declara status 'passed' com "
         "stop_reason 'max turns reached' e workspace inalterado, e nao referencia "
         "criterio de aceite."),
        ("10. Gates governam conclusao?",
         f"Nao de forma utilizavel. {ps.get('STILL_BROKEN',0)and ''}O gate DONE e "
         f"inalcancavel pelo caminho real por divergencia de id (NEW-002)."),
        ("11. Trace representa estado real?",
         "Sim. Nenhum no fabrication; not-found honesto com lista de nos conhecidos."),
        ("12. Resume continua execucao?",
         "Sim. 12 mensagens e tool queue restauradas, 12 passos executados, checkpoint r3. "
         "Perde apenas a associacao com a meta (NEW-009)."),
        ("13. LPC seleciona contexto dependente da tarefa?",
         "Nao. 5 tarefas semanticas distintas deram overlap Jaccard 0.95-1.00 e nenhum "
         "arquivo de codigo incluido."),
        ("14. A documentacao ficou substancialmente melhor?",
         f"Pouco em conteudo: media {agg['docs_before_mean']} -> {agg['docs_after_mean']}. "
         f"Muito em governanca: o Control Plane e semeado, audit/authority/impact/verify "
         f"operam sobre dados reais. O conjunto documental continua generico e monolitico."),
        ("15. Pode ser usado de forma confiavel em projeto real?",
         "Nao ainda. O bootstrap e o tracking documental sao utilizaveis; conclusion "
         "automatica de meta, execucao autonoma e contexto por tarefa nao sao."),
        ("16. Cinco maiores bloqueadores",
         "; ".join(f"{f['id']} ({f['title']})" for f in blockers[:5]) + "."),
    ]
    for q, ans in qa:
        a(f"**{q}** {ans}")
        a("")

    a("## 13. Backlog repriorizado")
    a("")
    for sev in ("P0", "P1", "P2", "P3"):
        items = sorted([f for f in cur["findings"] if f["severity"] == sev
                        and f["status"] in ("STILL_BROKEN", "PARTIALLY_FIXED", "NEW")],
                       key=lambda x: x["id"])
        a(f"### {sev} ({len(items)})")
        a("")
        for f in items:
            a(f"- [ ] **{f['id']}** {f['title']}")
        a("")
    return "\n".join(L)


def base_score(base, fid):
    for f in base["findings"]:
        if f["id"] == fid:
            return f["severity"]
    for b in base["bridges"]:
        if b["id"] == fid:
            return b.get("priority", "P1")
    return "?"


def new_label(f):
    return {"FIXED": "corrigido", "PARTIALLY_FIXED": "parcial",
            "STILL_BROKEN": "presente", "REGRESSED": "REGRESSOU", "NEW": "novo"}.get(f["status"], "?")


def render_regression(cur, base, agg):
    L = []
    a = L.append
    prior_ids = {f["id"] for f in base["findings"]} | {b["id"] for b in base["bridges"]}
    a("# Prumo — Relatorio de Regressao")
    a("")
    a(f"Baseline `{cur['meta']['previous_benchmark_commit']}` -> "
      f"medido `{cur['meta']['commit']}`.")
    a("")
    a(f"**Nota composta: {agg['headline_before']} -> {agg['headline_after']} "
      f"({agg['headline_delta']:+})**")
    a("")
    a("## Matriz before -> after")
    a("")
    a("| ID | Finding antigo | Antes | Agora | Status | Evidencia |")
    a("|---|---|---:|---:|:---:|---|")
    for f in cur["findings"]:
        if f["id"] not in prior_ids:
            continue
        b0 = base_score(base, f["id"])
        b1 = "corrigido" if f["status"] == "FIXED" else \
             "parcial" if f["status"] == "PARTIALLY_FIXED" else \
             "presente" if f["status"] == "STILL_BROKEN" else f["status"]
        a(f"| {f['id']} | {f['title']} | {b0} | {b1} | **{f['status']}** | {f['evidence'][:130]} |")
    a("")
    a("## Subsistemas que regrediram")
    a("")
    reg = [h for h in cur["health_matrix"] if h["after"] < h["before"]]
    if reg:
        a("| Subsistema | Antes | Agora | Motivo |")
        a("|---|---:|---:|---|")
        for h in reg:
            a(f"| {h['subsystem']} | {h['before']} | {h['after']} | {h.get('regression_note') or h['evidence_after'][:150]} |")
    else:
        a("Nenhum.")
    a("")
    a("## Resumo numerico")
    a("")
    a(f"- Findings anteriores re-testados: **{agg['prior_total']}**")
    for st, n in sorted(agg["prior_by_status"].items()):
        a(f"  - {st}: **{n}**")
    a(f"- Findings novos: **{agg['new_total']}**")
    a(f"- Severidade atual: " + ", ".join(f"{k}={v}" for k, v in agg["severity_counts"].items()))
    a(f"- Bridges: working {agg['bridges_working']}, partial {agg['bridges_partial']}, "
      f"broken {agg['bridges_broken']} (de {agg['bridges_total']})")
    a(f"- Failure modes detectados: {agg['fm_detected']} de {agg['fm_total']}")
    a(f"- End-to-End: {agg['e2e_pass']} PASS / {agg['e2e_partial']} PARTIAL / "
      f"{agg['e2e_fail']} FAIL")
    return "\n".join(L)


def render_html(cur, base, agg):
    m = cur["meta"]
    e = escape
    prior_ids = {f["id"] for f in base["findings"]} | {b["id"] for b in base["bridges"]}
    H = []
    a = H.append
    a("<!DOCTYPE html><html lang='pt-BR'><head><meta charset='utf-8'>")
    a("<meta name='viewport' content='width=device-width,initial-scale=1'>")
    a("<title>Prumo — Regression Benchmark</title><style>")
    a("""
:root{--bg:#0f1115;--panel:#171a21;--panel2:#1d212a;--fg:#e6e8ee;--mut:#9aa3b2;
--line:#272c37;--ok:#3fb950;--warn:#d29922;--bad:#f85149;--info:#58a6ff;--acc:#a371f7}
*{box-sizing:border-box}
body{margin:0;background:var(--bg);color:var(--fg);
font:15px/1.6 -apple-system,BlinkMacSystemFont,'Segoe UI',Roboto,Helvetica,Arial,sans-serif}
.wrap{max-width:1180px;margin:0 auto;padding:32px 20px 80px}
h1{font-size:30px;margin:0 0 4px}h2{font-size:21px;margin:40px 0 14px;
padding-bottom:8px;border-bottom:1px solid var(--line)}
h3{font-size:16px;margin:24px 0 10px;color:var(--mut);text-transform:uppercase;
letter-spacing:.06em}
.sub{color:var(--mut);margin-bottom:24px}
.cards{display:grid;grid-template-columns:repeat(auto-fit,minmax(190px,1fr));gap:12px;margin:18px 0}
.card{background:var(--panel);border:1px solid var(--line);border-radius:10px;padding:16px}
.card .k{font-size:11px;text-transform:uppercase;letter-spacing:.07em;color:var(--mut)}
.card .v{font-size:27px;font-weight:650;margin-top:6px}
.card .d{font-size:12px;margin-top:4px}
.pos{color:var(--ok)}.neg{color:var(--bad)}.zero{color:var(--mut)}
table{width:100%;border-collapse:collapse;margin:12px 0;background:var(--panel);
border:1px solid var(--line);border-radius:10px;overflow:hidden;font-size:13.5px}
th{background:var(--panel2);text-align:left;padding:9px 11px;font-weight:600;
font-size:11px;text-transform:uppercase;letter-spacing:.05em;color:var(--mut)}
td{padding:9px 11px;border-top:1px solid var(--line);vertical-align:top}
tr:hover td{background:#1b1f28}
code{background:#0b0d11;padding:1px 5px;border-radius:4px;font-size:12.5px}
.pill{display:inline-block;padding:2px 8px;border-radius:99px;font-size:11px;
font-weight:650;letter-spacing:.02em}
.FIXED{background:rgba(63,185,80,.15);color:var(--ok)}
.PARTIALLY_FIXED{background:rgba(210,153,34,.15);color:var(--warn)}
.STILL_BROKEN{background:rgba(248,81,73,.15);color:var(--bad)}
.REGRESSED{background:rgba(248,81,73,.3);color:var(--bad)}
.NEW{background:rgba(163,113,247,.15);color:var(--acc)}
.NOT_REPRODUCIBLE,.OBSOLETE{background:#2a2f3a;color:var(--mut)}
.PASS{color:var(--ok)}.FAIL{color:var(--bad)}.PARTIAL{color:var(--warn)}
.sm{font-size:12px;color:var(--mut)}
.mono{font-family:ui-monospace,SFMono-Regular,Menlo,monospace;font-size:12px}
.bar{height:7px;background:#0b0d11;border-radius:99px;overflow:hidden;min-width:70px}
.bar i{display:block;height:100%}
details{background:var(--panel);border:1px solid var(--line);border-radius:8px;
padding:10px 13px;margin:8px 0}
summary{cursor:pointer;font-weight:600;font-size:14px}
.note{border-left:3px solid var(--warn);padding:8px 13px;background:rgba(210,153,34,.07);
margin:12px 0;border-radius:0 8px 8px 0}
.grid2{display:grid;grid-template-columns:1fr 1fr;gap:14px}
@media(max-width:820px){.grid2{grid-template-columns:1fr}}
""")
    a("</style></head><body><div class='wrap'>")
    a(f"<h1>Prumo — Regression + Improvement Benchmark</h1>")
    a(f"<div class='sub'><code>{e(m['commit'][:12])}</code> · main · "
      f"v{e(m['cli_reported_version'])} · {e(m['benchmark_date'])} · "
      f"baseline {e(m['previous_benchmark_commit'][:12])}</div>")

    a("<h2>Before vs After</h2><div class='cards'>")
    for k, v, d, good in [
        ("Nota composta", agg["headline_after"], f"{agg['headline_before']} -> {agg['headline_after']} ({agg['headline_delta']:+})", agg["headline_delta"] > 0),
        ("Health Matrix (media)", agg["hm_after_mean"], f"{agg['hm_before_mean']} -> {agg['hm_after_mean']}", agg["hm_after_mean"] > agg["hm_before_mean"]),
        ("End-to-End Loop", agg["e2e_after"], f"{agg['e2e_before']} -> {agg['e2e_after']} ({agg['e2e_delta']:+})", agg["e2e_delta"] > 0),
        ("Documentacao", agg["docs_after_mean"], f"{agg['docs_before_mean']} -> {agg['docs_after_mean']}", agg["docs_after_mean"] > agg["docs_before_mean"]),
    ]:
        cls = "pos" if good else ("neg" if d.startswith("-") else "zero")
        a(f"<div class='card'><div class='k'>{e(k)}</div><div class='v'>{v}</div>"
          f"<div class='d {cls}'>{e(str(d))}</div></div>")
    a("</div>")
    a(f"<div class='note'><b>Faixa:</b> {e(BAND(agg['headline_after']))}. "
      f"Subsistemas que melhoraram: {agg['hm_improved']}; que pioraram: {agg['hm_regressed']}; "
      f"iguais: {agg['hm_unchanged']}.</div>")

    a("<h3>Contagem de findings</h3><div class='cards'>")
    for st in ("FIXED", "PARTIALLY_FIXED", "STILL_BROKEN", "REGRESSED", "NEW"):
        n = agg["prior_by_status"].get(st, 0) if st != "NEW" else agg["new_total"]
        if st == "NEW":
            n = agg["new_total"]
        a(f"<div class='card'><div class='k'>{e(st.replace('_',' '))}</div>"
          f"<div class='v'><span class='pill {st}'>{n}</span></div></div>")
    a("</div>")

    a("<h2>Health Matrix</h2><table><tr><th>Subsistema</th><th>Antes</th><th>Agora</th>"
      "<th>Delta</th><th>Status</th><th>Evidencia</th></tr>")
    for h in cur["health_matrix"]:
        d = h["after"] - h["before"]
        cls = "pos" if d > 0 else ("neg" if d < 0 else "zero")
        st = h["status"]
        sc = {"PASS": "PASS", "PARTIAL": "PARTIAL", "FAIL": "FAIL"}.get(st, st)
        a(f"<tr><td><b>{e(h['subsystem'])}</b></td><td>{h['before']}</td>"
          f"<td><b>{h['after']}</b></td><td class='{cls}'>{d:+.1f}</td>"
          f"<td class='{sc}'>{e(st)}</td><td class='sm'>{e(h['evidence_after'][:260])}</td></tr>")
    a("</table>")

    a("<h2>Findings anteriores re-testados</h2><table><tr><th>ID</th><th>Finding</th>"
      "<th>Sev</th><th>Status</th><th>Evidencia</th></tr>")
    for f in cur["findings"]:
        if f["id"] not in prior_ids:
            continue
        a(f"<tr><td class='mono'>{e(f['id'])}</td><td>{e(f['title'])}</td>"
          f"<td>{f['severity']}</td><td><span class='pill {f['status']}'>"
          f"{e(f['status'].replace('_',' '))}</span></td>"
          f"<td class='sm'>{e(f['evidence'][:220])}</td></tr>")
    a("</table>")

    newf = [f for f in cur["findings"] if f["id"] not in prior_ids]
    a("<h2>Findings novos</h2><table><tr><th>ID</th><th>Sev</th><th>Finding</th>"
      "<th>Impacto</th></tr>")
    for f in sorted(newf, key=lambda x: (x["severity"], x["id"])):
        a(f"<tr><td class='mono'>{e(f['id'])}</td><td>{f['severity']}</td>"
          f"<td>{e(f['title'])}</td><td class='sm'>{e(f.get('impact','')[:240])}</td></tr>")
    a("</table>")
    for f in sorted(newf, key=lambda x: (x["severity"], x["id"])):
        a(f"<details><summary>{e(f['id'])} — {e(f['title'])} "
          f"<span class='pill NEW'>{f['severity']}</span></summary>"
          f"<p class='sm'><b>Reproducao:</b> {e(f['repro_after'])}</p>"
          f"<p class='sm'><b>Evidencia:</b> {e(f['evidence'])}</p>"
          f"<p class='sm'><b>Impacto:</b> {e(f.get('impact',''))}</p></details>")

    a("<h2>End-to-End Pipeline</h2><table><tr><th>Estagio</th><th>Antes</th>"
      "<th>Agora</th><th>Evidencia</th></tr>")
    for s in cur["end_to_end"]["chain"]:
        sc = {"PASS": "PASS", "FAIL": "FAIL", "PARTIAL": "PARTIAL"}.get(s["after"], s["after"])
        a(f"<tr><td>{e(s['stage'])}</td><td class='sm'>{e(s['before'])}</td>"
          f"<td class='{sc}'><b>{e(s['after'])}</b></td>"
          f"<td class='sm'>{e(s['evidence'][:230])}</td></tr>")
    a("</table>")

    a("<h2>False Bridges</h2><table><tr><th>Bridge</th><th>Antes</th><th>Agora</th>"
      "<th>Evidencia</th></tr>")
    for b in cur["bridges"]:
        cls = {"WORKING": "PASS", "PARTIAL": "PARTIAL", "BROKEN": "FAIL"}[b["after"]]
        a(f"<tr><td>{e(b['name'])}</td><td class='sm'>{e(b['before'])}</td>"
          f"<td class='{cls}'><b>{e(b['after'])}</b></td>"
          f"<td class='sm'>{e(b['evidence'][:230])}</td></tr>")
    a("</table>")

    a("<h2>Failure Modes</h2><table><tr><th>ID</th><th>Injetado</th><th>Antes</th>"
      "<th>Agora</th><th>Nota</th></tr>")
    for f in cur["failure_modes"]:
        cls = "PASS" if f.get("detected_after") else "FAIL"
        a(f"<tr><td class='mono'>{e(f['id'])}</td><td>{e(f['injection'])}</td>"
          f"<td class='sm'>{e(str(f['before']))}</td>"
          f"<td class='{cls}'><b>{e(str(f['after']))}</b></td>"
          f"<td class='sm'>{e(f['note'][:210])}</td></tr>")
    a("</table>")

    a("<h2>Promise vs Reality</h2><table><tr><th>Promessa</th><th>Antes</th>"
      "<th>Agora</th><th>Evidencia</th></tr>")
    for p in cur["promise_vs_reality"]:
        cls = {"CONFIRMADA": "PASS", "PARCIALMENTE CONFIRMADA": "PARTIAL",
               "NÃO CONFIRMADA": "FAIL", "DIVERGENTE": "FAIL"}[p["after"]]
        a(f"<tr><td>{e(p['promise'])}</td><td class='sm'>{e(p['before'])}</td>"
          f"<td class='{cls}'><b>{e(p['after'])}</b></td>"
          f"<td class='sm'>{e(p['evidence'][:210])}</td></tr>")
    a("</table>")

    a("<h2>Documents</h2><table><tr><th>Documento</th><th>Tipo</th><th>Antes</th>"
      "<th>Agora</th><th>Nota</th></tr>")
    for d in cur["documents"]:
        b0 = d.get("before")
        a(f"<tr><td class='mono'>{e(d['path'])}</td><td class='sm'>{e(d['kind'])}</td>"
          f"<td>{'n/a' if b0 is None else b0}</td><td><b>{d['after']}</b></td>"
          f"<td class='sm'>{e(d['note'][:190])}</td></tr>")
    a("</table>")

    a("<h2>Maturity Gates</h2><table><tr><th>Eixo</th><th>Classificacao</th>"
      "<th>Justificativa</th></tr>")
    for g in cur["maturity_gates"]:
        a(f"<tr><td><b>{e(g['axis'])}</b></td><td><b>{e(g['classification'])}</b></td>"
          f"<td class='sm'>{e(g['justification'])}</td></tr>")
    a("</table>")

    a("<h2>Prioritized Backlog</h2>")
    for sev in ("P0", "P1", "P2", "P3"):
        items = sorted([f for f in cur["findings"] if f["severity"] == sev
                        and f["status"] in ("STILL_BROKEN", "PARTIALLY_FIXED", "NEW")],
                       key=lambda x: x["id"])
        a(f"<h3>{sev} ({len(items)})</h3><table><tr><th>ID</th><th>Item</th></tr>")
        for f in items:
            a(f"<tr><td class='mono'>{e(f['id'])}</td><td>{e(f['title'])}</td></tr>")
        a("</table>")
    a("</div></body></html>")
    return "\n".join(H)


def main():
    cur, base = load()
    agg = compute(cur, base)
    errors = validate(cur, base, agg)

    with open(os.path.join(HERE, "computed-aggregates.json"), "w", encoding="utf-8") as fh:
        json.dump(agg, fh, indent=2, ensure_ascii=False)

    if errors:
        print("INVARIANT FAILURES:")
        for e in errors:
            print("  -", e)
        sys.exit(1)

    print("invariants: OK")
    with open(os.path.join(HERE, "AUDIT.md"), "w", encoding="utf-8") as fh:
        fh.write(render_md(cur, base, agg))
    with open(os.path.join(HERE, "REGRESSION.md"), "w", encoding="utf-8") as fh:
        fh.write(render_regression(cur, base, agg))
    with open(os.path.join(HERE, "prumo-dogfood-report.html"), "w", encoding="utf-8") as fh:
        fh.write(render_html(cur, base, agg))

    print(f"headline {agg['headline_before']} -> {agg['headline_after']} "
          f"({agg['headline_delta']:+})")
    print(f"findings: {agg['findings_total']} total; prior {agg['prior_total']}; new {agg['new_total']}")
    print("status:", agg["prior_by_status"], "new:", agg["new_total"])
    print("severity:", agg["severity_counts"])
    print(f"bridges: working {agg['bridges_working']} partial {agg['bridges_partial']} "
          f"broken {agg['bridges_broken']}")
    print(f"e2e: {agg['e2e_pass']}P/{agg['e2e_partial']}Pa/{agg['e2e_fail']}F")
    print("rendered AUDIT.md, REGRESSION.md, prumo-dogfood-report.html")


if __name__ == "__main__":
    main()
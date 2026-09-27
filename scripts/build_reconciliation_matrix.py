import os
import json

with open('docs/migration/notion-final-docs-inventory.json') as f:
    notion_inv = json.load(f)
with open('docs/migration/repo-docs-inventory.json') as f:
    repo_inv = json.load(f)

with open('docs/SOURCE_MAP.json') as f:
    source_map = json.load(f)

reconciliation = []

for ndoc in notion_inv:
    lid = ndoc['logical_id']
    title = ndoc['title']
    cat = ndoc['category']
    uuid = ndoc['uuid']
    
    action = 'UNKNOWN'
    canonical_target = ''
    rationale = ''
    superseded = False
    
    if 'SUPERSEDED' in ndoc['status_raw'].upper() or 'SUPERSEDED' in title.upper():
        superseded = True
        action = 'PROMOTED_HISTORICAL'
        canonical_target = 'docs/harness/specs/superseded-floem-stack.md'
        rationale = 'Documento histórico: stack Floem superseded por Bubble Tea v2 e Freya.'
    elif lid == 'CONST-86':
        action = 'PROMOTED_CONSTITUTIONAL'
        canonical_target = 'docs/architecture/unified-architecture.md'
        rationale = 'Constituição 86: Arquitetura Unificada de Framework, Harness, Code Agent e Surfaces.'
    elif lid == 'CONST-87':
        action = 'PROMOTED_CONSTITUTIONAL'
        canonical_target = 'docs/runtime/model-portfolio-and-routing.md'
        rationale = 'Constituição 87: Roteamento por Custo/Quota, Budget e Fallback.'
    elif lid == 'CONST-88':
        action = 'PROMOTED_CONSTITUTIONAL'
        canonical_target = 'docs/architecture/documentation-architecture-v2.md'
        rationale = 'Constituição 88: Arquitetura Documental v2, Canonical Graph, IR e Rebuild Determinístico.'
    elif lid == 'CONST-89':
        action = 'PROMOTED_CONSTITUTIONAL'
        canonical_target = 'docs/development/consolidation-program.md'
        rationale = 'Constituição 89: Programa de Consolidação e Implementação do Prumo Unificado.'
    elif lid == 'CONST-90':
        action = 'PROMOTED_CONSTITUTIONAL'
        canonical_target = 'docs/governance/consolidation-crosswalk.md'
        rationale = 'Constituição 90: Crosswalk de Consolidação, Ownership e Supersession.'
    elif lid == 'CONST-91':
        action = 'PROMOTED_CONSTITUTIONAL'
        canonical_target = 'docs/reference/canonical-glossary.md'
        rationale = 'Constituição 91: Glossário Canônico e Ontologia Operacional.'
    elif lid == 'CONST-92':
        action = 'PROMOTED_CONSTITUTIONAL'
        canonical_target = 'docs/contracts/directive-compiler.md'
        rationale = 'Constituição 92: Directive Compiler e Regras Executáveis.'
    elif lid == 'CONST-93':
        action = 'PROMOTED_CONSTITUTIONAL'
        canonical_target = 'docs/harness/capability-skill-gap-register.md'
        rationale = 'Constituição 93: Capability & Skill Gap Register do Harness.'
    elif lid.startswith('85') or lid.startswith('PART2_85') or '85 A' in title:
        action = 'PROMOTED_CONSTITUTIONAL'
        canonical_target = 'docs/contracts/llm-agent-execution-contract.md'
        rationale = 'Constituição 85.A: Execution Contract Grounding, Tools, Evidence e Handoff.'
    elif lid.startswith('84') or lid.startswith('PART2_84'):
        action = 'PROMOTED_CONSTITUTIONAL'
        sub_id = lid.replace(' ', '_').lower()
        canonical_target = f'docs/quality/gauntlet-{sub_id}.md'
        rationale = f'Constituição 84: Perfis do Gauntlet / Total Assurance ({title}).'
    elif lid == '79 AA' or '79 AA' in title:
        action = 'PROMOTED_SKILL_SPEC'
        canonical_target = 'docs/skills/specs/grounding-spec-compliance.md'
        rationale = 'Skill Package P0: Grounding, Spec Compliance e Surface Conformance.'
    elif lid.startswith('79') or lid.startswith('PART2_79'):
        pack_name = title.split('—')[-1].strip().lower().replace(' ', '-').replace(',', '').replace('/', '-')
        canonical_target = f'docs/skills/specs/{pack_name}.md'
        action = 'PROMOTED_SKILL_SPEC'
        rationale = f'Skill Package de referência para {title}.'
    elif lid.startswith('PART2-CH'):
        action = 'PROMOTED_HARNESS_SPEC'
        ch_num = lid.split('-')[-1]
        canonical_target = f'docs/harness/specs/ch{ch_num}.md'
        rationale = f'Especificação especializada do Harness / Code Agent (Capítulo {ch_num}).'
    elif lid.startswith('PART1-CH'):
        action = 'PROMOTED_FRAMEWORK_SPEC'
        ch_num = lid.split('-')[-1]
        canonical_target = f'docs/framework/specs/ch{ch_num}.md'
        rationale = f'Especificação de engenharia do Framework (Capítulo {ch_num}).'
    elif lid.startswith('PHASE'):
        action = 'PROMOTED_PHASE_SPEC'
        canonical_target = f'docs/development/phases/{lid.lower()}.md'
        rationale = f'Fase/Gate de implementação ({title}).'
    elif lid.startswith('HUB'):
        action = 'PROMOTED_HUB'
        canonical_target = 'docs/LIVRO_VIVO_HUB.md' if 'PART-I' in lid else 'docs/CODE_AGENT_HUB.md'
        rationale = 'Hub / Sumário estrutural do Livro Vivo.'
    elif lid == 'CONST-80':
        action = 'PROMOTED_CONSTITUTIONAL'
        canonical_target = 'docs/reference/prumo-ask-headless.md'
        rationale = 'Especificação do Prumo Ask Headless One-Shot Interface.'
    elif lid == 'CONST-81':
        action = 'PROMOTED_CONSTITUTIONAL'
        canonical_target = 'docs/runtime/global-learning-layer.md'
        rationale = 'Especificação da Global Learning Layer e Padrões Cross-Project.'
    elif lid == 'CONST-82':
        action = 'PROMOTED_CONSTITUTIONAL'
        canonical_target = 'docs/architecture/prumo-native-workspace-viewer.md'
        rationale = 'Especificação do Prumo Native Workspace Viewer com Freya.'
    elif lid == 'CONST-83':
        action = 'PROMOTED_CONSTITUTIONAL'
        canonical_target = 'docs/runtime/decision-intelligence-runtime.md'
        rationale = 'Especificação do Decision Intelligence Runtime.'
    else:
        action = 'PROMOTED_SPECIALIZED'
        canonical_target = f'docs/reference/notion-{uuid[:8]}.md'
        rationale = f'Documento especializado: {title}.'

    reconciliation.append({
        'logical_id': lid,
        'title': title,
        'category': cat,
        'uuid': uuid,
        'action': action,
        'canonical_target': canonical_target,
        'superseded': superseded,
        'rationale': rationale
    })

with open('docs/migration/reconciliation-matrix.json', 'w', encoding='utf-8') as f:
    json.dump(reconciliation, f, indent=2, ensure_ascii=False)

md_lines = [
    '# Matriz de Reconciliação Semântica — Notion Final Docs ↔ Repositório',
    '',
    f'**Total de documentos analisados:** {len(reconciliation)}',
    '**Regra normativa:** Documento 90 (Crosswalk de Consolidação) + Documento 86 (Arquitetura Unificada).',
    '**Princípio de zero perda de informação:** preserve before transform; classify before delete; reconcile before promote.',
    '',
    '| ID Lógico | Título | Categoria | Ação de Reconciliação | Alvo Canônico no Repositório | Rationale |',
    '|---|---|---|---|---|---|'
]

for r in reconciliation:
    t = r['title'][:50].replace('|', '/')
    act = r['action']
    tgt = f"`{r['canonical_target']}`" if r['canonical_target'] else '-'
    rat = r['rationale'][:60].replace('|', '/')
    md_lines.append(f"| `{r['logical_id']}` | {t} | {r['category']} | `{act}` | {tgt} | {rat} |")

with open('docs/migration/reconciliation-matrix.md', 'w', encoding='utf-8') as f:
    f.write('\n'.join(md_lines) + '\n')

print(f'Generated reconciliation matrix: {len(reconciliation)} documents mapped cleanly.')

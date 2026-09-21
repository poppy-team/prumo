import json
import os

with open('docs/SOURCE_MAP.json') as f:
    sm = json.load(f)

with open('docs/migration/reconciliation-matrix.json') as f:
    recon = json.load(f)

existing_docs = {d['local_path']: d for d in sm.get('documents', [])}

# Update SOURCE_MAP metadata
sm['generated_at'] = '2026-09-21'
sm['source'] = 'Prumo — Livro Vivo Unificado Framework, Harness & Code Agent (Snapshot 2026-09-21)'
sm['source_url'] = 'https://app.notion.com/p/raillen/Project-Prumo-Framework-Livro-Vivo-v0-4-3d59bb7d023f8168a69ac57326eddc85'
sm['snapshot_id'] = '2026-09-21-e3a56193'
sm['snapshot_archive_sha256'] = 'e3a561937bfc00209f968f4ead96b84c76a5544b0eb40f50c504dc7037501b4d'

for r in recon:
    tgt = r['canonical_target']
    if not tgt or r['action'] == 'SUPERSEDED_HISTORICAL':
        continue
        
    entry = existing_docs.get(tgt)
    if not entry:
        entry = {
            'local_path': tgt,
            'notion_pages': [],
            'semantic_role': r['rationale']
        }
        existing_docs[tgt] = entry
        
    # Append notion page mapping if not present
    page_entry = {
        'title': r['title'],
        'uuid': r['uuid'],
        'logical_id': r['logical_id'],
        'role': r['rationale']
    }
    # Check if page already in notion_pages
    found = False
    for np in entry['notion_pages']:
        if np.get('uuid') == r['uuid'] or np.get('title') == r['title']:
            np.update(page_entry)
            found = True
            break
    if not found:
        entry['notion_pages'].append(page_entry)

sm['documents'] = sorted(list(existing_docs.values()), key=lambda d: d['local_path'])

with open('docs/SOURCE_MAP.json', 'w', encoding='utf-8') as f:
    json.dump(sm, f, indent=2, ensure_ascii=False)

print(f'Updated docs/SOURCE_MAP.json with {len(sm["documents"])} mapped paths.')

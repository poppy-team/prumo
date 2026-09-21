import os
import re
import json
from urllib.parse import quote

snapshot_dir = '.prumo/imports/notion/prumo-unified-livro-vivo/2026-09-21-e3a56193'
with open('docs/migration/reconciliation-matrix.json') as f:
    recon = json.load(f)

# Build a mapping from Notion filename / UUID to canonical target
filename_to_target = {}
uuid_to_target = {}

for r in recon:
    tgt = r['canonical_target']
    if tgt:
        uuid_to_target[r['uuid']] = tgt
        for fn in os.listdir(snapshot_dir):
            if r['uuid'] in fn:
                filename_to_target[fn] = tgt
                filename_to_target[quote(fn)] = tgt
                break

print(f'Configured {len(uuid_to_target)} UUID mappings.')

promoted_count = 0

for item in recon:
    target_path = item['canonical_target']
    uuid = item['uuid']
    if not target_path or item['action'] == 'SUPERSEDED_HISTORICAL':
        continue
        
    src_file = None
    for fn in os.listdir(snapshot_dir):
        if uuid in fn:
            src_file = os.path.join(snapshot_dir, fn)
            break
            
    if not src_file or not os.path.exists(src_file):
        print(f"Warning: source file for UUID {uuid} not found!")
        continue
        
    with open(src_file, 'r', encoding='utf-8') as sf:
        content = sf.read()
        
    def replace_link(match):
        text = match.group(1)
        url = match.group(2)
        for fn, tgt in filename_to_target.items():
            if fn in url:
                rel = os.path.relpath(tgt, os.path.dirname(target_path))
                return f'[{text}]({rel})'
        for u, tgt in uuid_to_target.items():
            if u in url:
                rel = os.path.relpath(tgt, os.path.dirname(target_path))
                return f'[{text}]({rel})'
        return match.group(0)
        
    updated_content = re.sub(r'\[([^\]]+)\]\(([^)]+\.md)\)', replace_link, content)
    
    header = f"""> Authority: canonical specification.
> Logical ID: {item['logical_id']}
> Source: Notion Living Book ({uuid})
> Status: {item['rationale']}

"""
    h1_idx = updated_content.find('\n', updated_content.find('# '))
    if h1_idx != -1:
        final_doc = updated_content[:h1_idx+1] + '\n' + header + updated_content[h1_idx+1:]
    else:
        final_doc = header + updated_content

    os.makedirs(os.path.dirname(target_path), exist_ok=True)
    with open(target_path, 'w', encoding='utf-8') as df:
        df.write(final_doc)
        
    promoted_count += 1

print(f'Successfully promoted {promoted_count} canonical documents in total.')

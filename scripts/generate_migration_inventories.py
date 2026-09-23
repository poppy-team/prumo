import os
import re
import json
import hashlib

# 1. Build Markdown for Notion Final Docs Inventory
with open('docs/migration/notion-final-docs-inventory.json') as f:
    notion_inv = json.load(f)

md_lines = [
    '# Inventário Completo — Documentação Final Prumo (Snapshot 2026-09-21)',
    '',
    f'**Total de documentos extraídos:** {len(notion_inv)}',
    '**Origem:** `prumo-final-docs.zip` (SHA256: `e3a561937bfc00209f968f4ead96b84c76a5544b0eb40f50c504dc7037501b4d`)',
    '**Localização de proveniência:** `.prumo/imports/notion/prumo-unified-livro-vivo/2026-09-21-e3a56193/`',
    '',
    '| ID Lógico | Título | Categoria | Status | Tamanho (bytes) | Linhas | UUID Notion |',
    '|---|---|---|---|---|---|---|'
]

for doc in notion_inv:
    status = doc['status_raw'][:30].replace('|', '/')
    title = doc['title'][:60].replace('|', '/')
    lid = doc['logical_id']
    uuid = doc['uuid']
    md_lines.append(f"| `{lid}` | {title} | {doc['category']} | {status} | {doc['size_bytes']} | {doc['line_count']} | `{uuid}` |")

with open('docs/migration/notion-final-docs-inventory.md', 'w', encoding='utf-8') as f:
    f.write('\n'.join(md_lines) + '\n')

print('Generated docs/migration/notion-final-docs-inventory.md cleanly.')

# 2. Inventory of existing repository documentation
GENERATED_OUTPUTS = {
    os.path.join('docs', 'migration', 'repo-docs-inventory.json'),
    os.path.join('docs', 'migration', 'repo-docs-inventory.md'),
    os.path.join('docs', 'migration', 'notion-final-docs-inventory.md'),
}

repo_docs = []
for root, dirs, files in os.walk('docs'):
    for file in sorted(files):
        if file.endswith('.md') or file.endswith('.json'):
            rel_path = os.path.relpath(os.path.join(root, file), '.')
            if rel_path in GENERATED_OUTPUTS:
                continue
            with open(rel_path, 'rb') as fh:
                data = fh.read()
            digest = hashlib.sha256(data).hexdigest()
            text = data.decode('utf-8', errors='replace')
            first_h1 = ''
            for line in text.splitlines():
                if line.startswith('# '):
                    first_h1 = line[2:].strip()
                    break
            repo_docs.append({
                'path': rel_path,
                'type': 'markdown' if file.endswith('.md') else 'json',
                'size_bytes': len(data),
                'lines': len(text.splitlines()),
                'sha256': digest,
                'first_h1': first_h1
            })

# Also include root Markdown files
for file in sorted(os.listdir('.')):
    if file.endswith('.md'):
        with open(file, 'rb') as fh:
            data = fh.read()
        digest = hashlib.sha256(data).hexdigest()
        text = data.decode('utf-8', errors='replace')
        first_h1 = ''
        for line in text.splitlines():
            if line.startswith('# '):
                first_h1 = line[2:].strip()
                break
        repo_docs.append({
            'path': file,
            'type': 'markdown',
            'size_bytes': len(data),
            'lines': len(text.splitlines()),
            'sha256': digest,
            'first_h1': first_h1
        })

with open('docs/migration/repo-docs-inventory.json', 'w', encoding='utf-8') as f:
    json.dump(repo_docs, f, indent=2, ensure_ascii=False)

repo_md_lines = [
    '# Inventário da Documentação Preexistente no Repositório',
    '',
    f'**Total de arquivos documentais no repositório:** {len(repo_docs)}',
    '',
    '| Path | Tipo | Tamanho (bytes) | Linhas | Primeiro Título H1 | SHA256 (prefixo) |',
    '|---|---|---|---|---|---|'
]
for rd in repo_docs:
    h1 = rd['first_h1'][:60].replace('|', '/')
    p = rd['path']
    s = rd['sha256'][:10]
    repo_md_lines.append(f"| `{p}` | {rd['type']} | {rd['size_bytes']} | {rd['lines']} | {h1} | `{s}` |")

with open('docs/migration/repo-docs-inventory.md', 'w', encoding='utf-8') as f:
    f.write('\n'.join(repo_md_lines) + '\n')

print(f'Generated docs/migration/repo-docs-inventory.json and docs/migration/repo-docs-inventory.md ({len(repo_docs)} files) cleanly.')

# Adoção de Projetos Existentes (Brownfield)

Você tem um repositório existente em Go, Node/TypeScript, Python, Rust ou outra linguagem e deseja integrá-lo ao Prumo sem reescrever código ou perder histórico?

O **Motor de Adoção (`prumo adopt`)** foi desenhado especificamente para isso: ele inspeciona fatos técnicos, classifica a arquitetura e gera uma configuração canônica não-destrutiva.

---

## 1. O Fluxo de Adoção em 2 Passos

### Passo 1: Inspecionar e Auditar

Execute o comando `adopt` na raiz do seu repositório:

```bash
cd meu-projeto-legado
prumo adopt
```

O Prumo executará:
1. **Varredura de artefatos**: detecta manifestos (`go.mod`, `package.json`, `Cargo.toml`, `pyproject.toml`, Dockerfiles, CI/CD).
2. **Extração de Fatos Técnicos**: linguagens, frameworks utilizados, bancos de dados, suítes de teste.
3. **Classificação Arquitetural**: determina os tipos de aplicação e sugere um perfil de capacidades.
4. **Proposta de Migração**: exibe uma prévia do `prumo.json` gerado **sem alterar nenhum arquivo no disco**.

### Passo 2: Aplicar a Adoção

Quando estiver satisfeito com o relatório de auditoria, aplique a adoção:

```bash
prumo adopt --apply
```

Este comando:
- Cria o `prumo.json` compatível com o **Protocolo v3**.
- Inicializa a workforce correspondente sob `.ai/`.
- Gera a documentação de roteamento de intenção em `docs/PRUMO.md`, `PROJECT_STATE.md` e `ENTRYPOINT.md`.
- **Preserva 100% dos seus arquivos de código-fonte e documentações existentes**.

---

## 2. Subcomandos do `prumo adopt`

Para agentes de código e pipelines de CI/CD que precisam de inspeção granular:

| Comando | Descrição |
|---|---|
| `prumo adopt scan [caminho]` | Indexa arquivos, identifica artefatos conhecidos e quantifica o repositório. |
| `prumo adopt facts [caminho]` | Extrai e exibe o ledger de fatos técnicos observados com níveis de confiança. |
| `prumo adopt classify [caminho]` | Exibe a classificação arquitetural pura (linguagens, frameworks, toolchains). |
| `prumo adopt scaffold [caminho]` | Simulação em dry-run das mutações propostas com preview dos diffs. |
| `prumo adopt apply [caminho]` | Aplica as propostas e configura o repositório no Protocolo v3. |

### Exemplo em JSON para Code Agents:

```bash
prumo --json adopt classify .
```

---

## 3. Verificando o Projeto Adotado

Após a execução do `prumo adopt --apply`:

```bash
# Validar conformidade
prumo validate

# Diagnosticar estado e dependências
prumo doctor

# Compilar adapters de contexto para seus agentes de IA
prumo compile --all
```

Seu repositório existente agora está plenamente protegido contra deriva de contexto e apto a trabalhar com agentes autônomos de engenharia.

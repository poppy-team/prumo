# Instalação do Prumo

O Prumo é distribuído como um executável estático único compilado em Go, sem dependências externas de runtime (não exige Python, Node.js ou compiladores instalados).

## Instalação Rápida (Recomendada)

### Linux & macOS

```bash
# Baixar o executável oficial v0.6.0
curl -fsSL https://github.com/poppy-team/prumo/releases/latest/download/prumo-linux-amd64 -o /usr/local/bin/prumo
chmod +x /usr/local/bin/prumo

# Verificar instalação
prumo version
```

### Inicialização do Ambiente Global

Ao instalar pela primeira vez, execute o `setup` para registrar o manifesto do sistema e detectar ferramentas de IA no seu `$PATH`:

```bash
prumo setup
```

O comando inicializará o diretório global `~/.prumo`, contendo o registro de conectores e o cache de atualizações.

---

## Verificação de Integridade e Diagnóstico

Para auditar o ambiente host, ferramentas de IA detectadas e integridade do executável:

```bash
prumo doctor
```

Se executado fora de um projeto, o `prumo doctor` analisa o ambiente global; dentro de um projeto, analisa também a conformidade dos esquemas e metas do repositório.

---

## Autoupdate (Atualização Contínua)

O Prumo inclui um subsistema de auto-atualização integrado com verificação criptográfica SHA-256 diretamente dos lançamentos oficiais do GitHub:

```bash
# Verificar se há novas versões disponíveis
prumo upgrade --check

# Atualizar para a versão mais recente
prumo upgrade

# Atualizar para uma versão específica
prumo upgrade --version v0.6.1
```

---

## Desinstalação Reversível e Segura

Caso precise remover a instalação global ou redefinir caches:

```bash
# Remove conectores e binários globais
prumo uninstall

# Limpa caches sem afetar dados do projeto
prumo uninstall --purge-cache
```

> [!IMPORTANT]
> A desinstalação **nunca** muta ou remove diretórios de repositórios locais (`prumo.json`, `.ai/`, `docs/` e arquivos de metas permanecem 100% intactos).

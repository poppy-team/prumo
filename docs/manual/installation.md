# Manual de Instalação

## Escolha o método e caminhos (Installation paths)

| Cenário | Método |
|---------|--------|
| Usuário final | Binário de uma release publicada |
| Desenvolvimento do Prumo | `go build` ou `go run` |
| CI, testes e USB/devbox | `--home` ou `PRUMO_HOME` |
| Automação e scripts | Binário Go compilado ou empacotado |

## Instalação com uma linha (One-Link Install)

### Linux e macOS
O script detecta o sistema operacional e a arquitetura (`amd64`/`arm64`), baixa o binário e os checksums da release, valida a integridade SHA-256, instala em `~/.local/bin/prumo`, configura automaticamente o `PATH` nos arquivos de inicialização do seu shell (`~/.bashrc`, `~/.zshrc`, `~/.config/fish/config.fish` ou `~/.profile`) e executa `prumo setup`.

```bash
curl -fsSL https://raw.githubusercontent.com/raillen/prumo/main/scripts/install.sh | sh
```

Para revisar antes de executar:
```bash
curl -fsSL https://raw.githubusercontent.com/raillen/prumo/main/scripts/install.sh -o install.sh
less install.sh
sh install.sh
```

### Windows (PowerShell)
O script PowerShell detecta a arquitetura (`amd64`/`arm64`), baixa o binário `prumo-windows-*.exe`, valida a integridade SHA-256, instala em `%LOCALAPPDATA%\Programs\prumo\prumo.exe`, configura permanentemente o `PATH` do usuário no Registro do Windows e executa `prumo setup`.

```powershell
powershell -ExecutionPolicy ByPass -c "irm https://raw.githubusercontent.com/raillen/prumo/main/scripts/install.ps1 | iex"
```

Variáveis opcionais para customização:
```bash
PRUMO_VERSION=v0.6.0 PRUMO_INSTALL_DIR="$HOME/.local/bin" sh install.sh
PRUMO_HOME="$HOME/.prumo" sh install.sh
```


## Desenvolvimento a partir do repositório

```bash
git clone git@github.com:raillen/prumo.git
cd prumo
go version
go run ./cmd/prumo version
```

Compilar um binário local:

```bash
go build -trimpath -o ./prumo ./cmd/prumo
./prumo version
```

O binário Go não requer Python, Node ou CGO.

## Build de release

O script oficial gera Linux amd64/arm64, macOS amd64/arm64 e Windows amd64/arm64:

```bash
VERSION=0.5.0 sh scripts/release.sh
cat dist/checksums.txt
cat dist/release.json
```

A instalação de release deve verificar o checksum antes de substituir o binário. A rollback strategy preserva o binário anterior até o novo passar `version`, `framework-check` e smoke tests; em falha, restaure o binário anterior sem alterar dados do projeto. Assinaturas de release permanecem um gate operacional antes da distribuição pública final.

## Estado global e modo portátil

Por padrão, o Prumo usa `~/.prumo` em sistemas Unix-like. Use `PRUMO_HOME` ou `--home` para isolar o estado:

```bash
PRUMO_HOME="$PWD/.prumo-home" ./prumo setup
./prumo --home "$PWD/.prumo-home" setup
```

O estado global contém:

```text
PRUMO_HOME/
├── config/installation.json
├── cache/
├── logs/
├── connectors/
└── runtimes/
```

Não confunda esse estado com `.prumo/` dentro de um projeto. O segundo pertence ao projeto e pode conter estado derivado local.

## Setup

```bash
prumo --home ~/.prumo setup
```

`setup` é idempotente e registra:

- versão do Prumo;
- caminho do binário;
- harnesses detectados no `PATH`;
- estado de conectores;
- caminhos gerenciados pelo Prumo.

O comando não altera configurações globais de harnesses automaticamente.

## Instalar um conector

```bash
prumo --home ~/.prumo install connector opencode
```

O estado do conector deve possuir um cleanup manifest antes de qualquer remoção automática.

## Verificar instalação

```bash
prumo version
prumo --json version
prumo framework-check
prumo --json framework-check
```

Verificação de projeto:

```bash
prumo validate ./my-project
prumo doctor ./my-project
```

## Depreciação e Aposentadoria do Python (ADR 002)

O runtime e os testes em Python v0.3 (legacy) foram completamente removidos (consulte [ADR 002](../adr/002-retire-python-runtime.md)). O Prumo v0.6 é distribuído exclusivamente em binário único compilado em Go, sem dependência de interpretadores externos, ambientes virtuais ou gerenciadores de pacotes Python.

## Problemas comuns

### `go: command not found`
Instale Go 1.22 ou superior e verifique `go version`.

### `prumo: command not found`
Use o caminho absoluto do binário ou coloque o diretório do binário no `PATH`:

```bash
export PATH="$PWD/bin:$PATH"
```

### `missing prumo.json`
O comando foi executado fora de um projeto Prumo. Informe o caminho do projeto ou execute `prumo init` primeiro.

### Falha de checksum
Não use o artefato. Baixe novamente e compare com `dist/checksums.txt` ou com o checksum publicado pela release.

### Estado global corrompido
Use um home portátil novo para isolar o diagnóstico:

```bash
prumo --home "$PWD/prumo-recovery-home" setup
```

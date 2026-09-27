# Viewer Extensions

O viewer aceita pacotes de extensão sem linkar código de terceiros ao Freya ou ao Core.

## Formato atual

Um pacote pode ser um diretório com `manifest.json` ou um archive `.prumoext` TAR+GZ com assinatura Ed25519 destacada:

```text
my-extension/
├── manifest.json
└── payload/
```

O archive contém `metadata.json`, `manifest.json` e `payload/*`; a assinatura, a trust store e o lockfile são arquivos separados.

O viewer procura pacotes em:

1. `<workspace>/.prumo/extensions/`;
2. `PRUMO_VIEWER_EXTENSIONS`, usando o separador de paths do sistema;
3. `prumo-viewer/extensions/` durante o desenvolvimento local.

A pasta precisa estar em um desses roots. Cada subpasta é um pacote. O pacote de referência `rust-analyzer` exige `rust-analyzer` no `PATH` e um grant explícito para `editor.lsp`; o viewer não concede essa capability automaticamente.

## Contrato

O manifest segue `schemas/viewer-extension-manifest.schema.json` e declara:

- identidade, versão e publisher;
- `prumo.viewer.extensions/v1`;
- faixa do host;
- runtime `declarative` ou `process`;
- capabilities e permissions;
- contributions com IDs namespaced: commands, views, panels, themes, keybindings, LSP, tasks e debuggers.

IDs de contribution devem começar com o namespace do pacote. O viewer rejeita IDs duplicados, entrypoints fora do diretório e ações desconhecidas.

## Instalar um pacote de terceiros

```bash
mkdir -p "$PRUMO_WORKSPACE/.prumo/extensions"
cp -R ./my-extension "$PRUMO_WORKSPACE/.prumo/extensions/"
PRUMO_VIEWER_EXTENSIONS=/path/to/extensions prumo-native --workspace "$PRUMO_WORKSPACE"
```

Abra `Ctrl+Shift+P` para ver os comandos declarativos contributionados pelo pacote.

Para instalar um archive, use `ExtensionPackage::read_archive`, valide a assinatura com `TrustStore` e chame `install_to`; o SDK grava o lockfile em `extensions.lock.json` com o digest do pacote.

Para ativar um pacote processual, forneça um array JSON de `ExtensionGrant` separado:

```bash
PRUMO_VIEWER_EXTENSIONS=/path/to/extensions \
PRUMO_VIEWER_EXTENSION_GRANTS=/path/to/grants.json \
prumo-native --workspace "$PRUMO_WORKSPACE"
```

O grant precisa repetir `extension_id`, `extension_version`, `manifest_digest`, capabilities e scopes. O campo legado `manifest_digest` representa o digest do pacote completo (manifest + payload); o host recusa digest, versão ou capability divergentes. Comandos processuais são executados em worker e o resultado volta para a UI sem bloquear o paint.

## developing with the SDK

O contrato compartilhado vive em `prumo-viewer/extension-sdk/`. Ele não depende de Freya, do viewer ou do Core.

```toml
[dependencies]
prumo-extension-sdk = { path = "../prumo-viewer/extension-sdk" }
```

Para preparar o SDK para publicação:

```bash
cargo package --manifest-path prumo-viewer/extension-sdk/Cargo.toml --allow-dirty
```

O pacote de extensão continua sendo um diretório independente; o SDK é apenas a biblioteca de contratos e runtime.

O SDK fornece:

- tipos e validação do manifest;
- IDs, versões, capabilities e permissões;
- `ExtensionGrant`, autorização separada e vinculada ao digest do pacote;
- framing JSON-RPC 2.0 delimitado por newline;
- limite de 1 MiB por mensagem;
- helpers para escrever e ler mensagens.

## Broker methods

A extensão pode pedir ao host somente operações concedidas:

- `ui/notify`: mostra uma notificação, com limite de 2.000 caracteres;
- `workspace/read`: lê um arquivo dentro do workspace, com limite de 512 KiB por resposta;
- `workspace/search`: busca textual com regex opcional e limite de 200 resultados;
- `editor/reveal`: abre um arquivo e seleciona um intervalo;
- `editor/set-line-decorations` e `editor/clear-line-decorations`: pinta linhas do editor, com limite de 500 decorations;
- LSP: um pacote declarativo pode declarar `contributions.lsp`; após receber a capability `editor.lsp`, o host inicia o servidor com transporte stdio `Content-Length`, sincroniza `didOpen`/`didChange`/`didSave`/`didClose`, consome `publishDiagnostics` e oferece símbolos, hover, definition, completion, code actions e formatting. Cada servidor tem uma fila própria; requests que excedem 10 s enviam `$/cancelRequest` e não mantêm o lock global do host. `Ctrl+Space` abre completion no editor, `Ctrl+.` abre code actions para a seleção ou diagnostic sob o cursor e `Shift+Alt+F` formata o documento ativo; edits são aplicados somente ao documento ativo e entram no histórico do Rope.
- Panels/views: `contributions.panels` e `contributions.views` aceitam conteúdo estático limitado e são abertos pelo Command Palette.
- Themes: `contributions.themes` aceita tokens semânticos (`accent`, `background`, `surface-*`, `text-*`, `border*`, estados) e é aplicado sem acoplar o tema ao código de UI.
- Keybindings: `contributions.keybindings` usa `Ctrl`, `Shift` e `Alt`, e o dispatch global executa o comando associado.
- Workspace index: cada refresh atualiza metadados incrementais; `workspace/index` permite consultar até 500 arquivos.
- Filesystem: `workspace/create-file`, `workspace/create-folder`, `workspace/rename` e `workspace/delete` usam `workspace.write`, rejeitam path absoluto e nunca removem a raiz ou `.git`.
- Git: `git/status` e `git/diff` usam `git.read`; stage, unstage, discard, commit, push e pull usam `git.write`, sempre com path relativo validado.
- Tasks: `contributions.tasks` executa argv sem shell, com diretório limitado ao workspace, timeout de 120s e output máximo de 1 MiB.
- Debugger: `contributions.debuggers` inicializa adaptadores DAP stdio sob a capability `debugger`; o `launch` só ocorre por ação explícita do usuário e requests DAP permanecem bounded.
- Distribuição: `ExtensionPackage` lê/gera `.prumoext` TAR+GZ, assina Ed25519, valida revogação via `TrustStore` e grava `extensions.lock.json`; a instalação é atômica e nunca segue paths inseguras.
- `command/invoke`: executa um comando contributionado pelo próprio processo.

O host valida capability, boundary de path, tamanho e digest antes de responder. Ele nunca repassa socket, token ou handle interno.

## Limites da v1

- o viewer inicia processos com grant válido e conecta os brokers de workspace, editor, índice, Git, filesystem, LSP e DAP; operações arbitrárias de filesystem e execução de comandos do Core ainda exigem contratos próprios;
- completion, code actions e formatting cobrem o documento ativo; references, folding, split panes, workspace edits multi-arquivo e snippets ainda são waves futuras; diagnostics são publicados pelo LSP e exibidos no Problems do viewer;
- quarantine, provenance de marketplace e confinement de processos ainda são stages de release;
- extensões não recebem socket do Core, tokens, pointers Rust ou widgets.

## Onda seguinte

1. **Editor:** references, folding e split panes.
2. **Language:** syntax tree publishing, references, code actions remotas e snippets.
3. **Workspace:** file renderers, Git broker e operações de filesystem.
4. **Agent surfaces:** timeline, evidence e quality visualizers.
5. **Release:** quarantine, provenance e política de marketplace.

A política completa está em [ADR 021](../adr/021-viewer-extension-contract.md).

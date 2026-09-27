# Viewer Extension Capability Catalog

Cada linha é uma extensão potencial independente. A coluna **Estado** não indica que a capacidade já está pronta.

| Família | Capability ID | Estado | Primeiro host method |
|---|---|---|---|
| Comandos | `ui.command` | implementado v1 | `command/invoke` |
| Notificações | `ui.notify` | broker v1 implementado | `ui/notify` |
| Editor navigation | `editor.navigate` | broker v1 implementado | `editor/reveal` |
| Editor decorations | `editor.decorations` | line decorations v1 implementadas | `editor/set-line-decorations` |
| Workspace index | `workspace.index` | índice incremental em memória por refresh | `workspace/index` |
| Workspace write | `workspace.write` | create/rename/delete com boundary check | `workspace/create-*`, `workspace/rename`, `workspace/delete` |
| Git | `git.read`, `git.write` | status/diff e mutações via CLI | `git/*` |
| Tasks | `workspace.tasks` | execução sem shell, timeout e output limitado | Command Palette + `services/tasks` |
| Debugger | `debugger` | DAP stdio inicializado sob grant; launch explícito | host DAP + ação de debug |
| Panels and views | `ui.panel`, `ui.view` | contribution estática v1 implementada | palette + overlay |
| Themes | `ui.theme` | tokens semânticos v1 implementados | palette + aplicação de cores |
| Keybindings | `ui.keybinding` | parser e dispatch v1 implementados | dispatch global para comandos |
| Language intelligence | `editor.lsp` | lifecycle parcial: initialize, didOpen/didChange/didSave, diagnostics, símbolos, hover, definition, completion, code actions e formatting do documento ativo | `LspClient` + Problems + popup/editor actions |
| Diagnostics via LSP | `editor.lsp` | `publishDiagnostics` consumido pelo host | Problems + decorations inline |
| Search/index | `workspace.search` | busca e índice em memória implementados | `workspace/search`, `workspace/index` |
| File operations | `workspace.files` | `workspace/read` v1; create/rename/delete implementados | `workspace/read`, `workspace/create-*`, `workspace/rename`, `workspace/delete` |
| Git | `git.read`, `git.write` | host-only; extensão planejada | `git/status`, `git/stage` |
| Terminal/tasks | `terminal.*`, `task.*` | host-only; extensão planejada | `terminal/create`, `task/run` |
| Panels/views | `ui.panel`, `ui.view` | contrato reservado | `ui/contribute` |
| Themes | `ui.theme` | contrato reservado | `ui/theme` |
| Keybindings | `ui.keybinding` | contrato reservado | `ui/keybinding` |
| Diagnostics | `diagnostics.publish` | planejado | `diagnostics/publish` |
| Diff/renderers | `document.renderer` | planejado | `document/render` |
| Debugger | `debugger.*` | planejado | `debugger/launch` |
| Evidence/quality | `run.visualization` | host-only; extensão planejada | `run/snapshot` |
| Evidence e Quality no dock | `run.visualization` | v1 implementada no host a partir de eventos reais do protocolo; `evidence-<run>.json` e quality gates do Core ainda não têm operação | `DockView::Evidence`, `DockView::Quality` |

## Regra

Uma extensão só aparece como contribution quando seu método host existe e possui teste. O catálogo não é uma lista de UI decorativa.

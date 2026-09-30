# Manual do Usuário do Prumo

Este manual documenta o uso abrangente do CLI `prumo`, seus comandos, ciclo de vida de instalação e interoperabilidade com ambientes de agentes.

## Seções do Manual

- **[Instalação Oficial](/manual/installation)**: Requisitos de sistema, instalação via script curl/PowerShell, compilação a partir do código-fonte e verificação de integridade via checksum SHA-256.
- **[Desinstalação Limpa & Reversível](/manual/uninstallation)**: Modos de desinstalação segura (`pure`, `preserve`), remoção de binários e garantias de preservação de dados canônicos do repositório.
- **[Guia de Uso & Comandos](/manual/usage)**: Referência exaustiva de todos os comandos do `prumo` (`init`, `goal`, `plan`, `run`, `compile`, `docs`, `doctor`, `validate`, `adopt`, `agent`), flags e formatos de saída JSON.
- **[Conectores & Adapters](/manual/connectors)**: Como compilar e injetar adaptadores do Prumo em ferramentas como Google Antigravity, Claude Code, Cursor, Windsurf, OpenCode e Roo Code.

## Estrutura Operacional

O Prumo opera segundo três princípios estritos:

1. **Determinismo**: Toda saída gerada com `--json` segue o envelope estável canônico `protocol_version: "1"`, sem formatações ANSI ou saídas imprevistas.
2. **Isolamento de Estado**: Metas e planos de trabalho residem em `.prumo/` e `prumo.json`, sendo perfeitamente versionáveis em Git.
3. **Imutabilidade Auditada**: Mutações em Metas travadas (LOCKED) exigem o comando `prumo goal amend`, gerando um rastro explícito de justificativa.

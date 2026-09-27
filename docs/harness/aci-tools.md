# Ferramental ACI & Sandboxing

A interface **ACI (Agent-Computer Interface)** do Prumo Harness é o conjunto de ferramentas de precisão fornecidas aos agentes durante a execução de diretivas.

## Princípios de Design da ACI

Diferente de abordagens ingênuas que fornecem um shell irrestrito ao modelo, o Prumo implementa proteções fundamentais:

1. **Limites Rígidos de Saída (Bounded Output)**: Comandos executados pelo agente têm limites estritos de bytes capturados em stdout/stderr para impedir poluição de contexto e estouro de tokens.
2. **Timeouts Inegociáveis**: Qualquer processo filho possui limite de tempo (default 30 segundos, configurável por diretiva) antes de ser terminado com segurança.
3. **Isolamento de Escopo (Working Directory Sandboxing)**: Operações de escrita em arquivos só podem atingir alvos dentro da raiz permitida pelo workspace.
4. **Análise Estática Nativa (AST Tools)**: O agente pode inspecionar nós de sintaxe sem carregar arquivos gigantescos em memória.

## Ferramentas Disponíveis na Caixa

| Ferramenta | Identificador | Função |
|---|---|---|
| **Execução Segura** | `execute_process` | Dispara binários com timeout, variáveis de ambiente controladas e captura delimitada de logs. |
| **Edição Atômica** | `replace_file_content` | Substituição contígua de blocos de texto com validação prévia de correspondência única. |
| **Criação de Arquivos** | `write_to_file` | Escrita de artefatos novos com opção de criação automática de diretórios pais. |
| **Leitura Delimitada** | `view_file` | Leitura de trechos paginados de código (fatiamento 1-indexed) com bound de bytes. |
| **Inspeção de AST** | `ast_inspect` | Busca semântica de funções, structs, interfaces e declarações de tipos. |
| **Verificação de Gate** | `gate_eval` | Avaliação de comandos de teste e suítes de conformidade determinísticas. |

## Exemplo de Definição de Sandboxing

No arquivo de configuração do projeto ou perfil (`prumo.json`), as políticas de execução são declaradas:

```json
{
  "harness": {
    "sandbox": {
      "mode": "restricted",
      "allowed_commands": ["go", "git", "cargo", "pnpm", "make"],
      "max_timeout_seconds": 60,
      "max_output_bytes": 65536,
      "deny_network_for_tests": true
    }
  }
}
```

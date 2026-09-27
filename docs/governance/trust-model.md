# Modelo de Confiança & Segurança

O Prumo opera sob um modelo de **privilégio mínimo e isolamento estrito**. Agentes autônomos recebem apenas as permissões explicitamente concedidas pelo perfil do projeto.

## Níveis de Confiança

1. **Repositório Canônico (Nível Máximo)**: Arquivos versionados em Git com hashes íntegros. Nenhuma IA pode mutar esses arquivos sem passar por validação de portão (gate).
2. **Ambiente de Desenvolvimento Local**: O usuário humano detém autoridade suprema e pode revogar ou abortar qualquer execução do harness a qualquer momento.
3. **Agentes de IA (Nível Restrito)**: Executam em sandboxes de diretivas, sem acesso a segredos ou variáveis de ambiente de produção, com comandos restritos à whitelist do perfil.
4. **Fontes Externas & Web (Não Confiável)**: Qualquer dado recuperado da internet ou de pacotes terceiros é considerado não confiável até validação explícita de conformidade.

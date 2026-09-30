# Decisões Arquiteturais (ADRs)

Os Registros de Decisão de Arquitetura (**ADRs — Architectural Decision Records**) capturam o contexto, as opções consideradas e as justificativas técnicas para decisões fundamentais tomadas pela equipe de engenharia do Prumo.

## Lista Canônica de Decisões

| ADR | Título | Status | Resumo |
|---|---|---|---|
| **001** | Go Core | Aceito | Migração completa do Core para Go garantindo binário único nativo e tipagem estática. |
| **002** | Aposentadoria do Runtime Python | Aceito | Retirada definitiva de qualquer dependência de runtime Python para execução do Prumo. |
| **003** | Subsistema de Segurança de Linguagem | Aceito | Sandboxing de processos, quotas de CPU/memória e limites estritos de execução. |
| **004** | Renomeação para Prumo | Aceito | Transição de marca e identidade formal para o nome Prumo. |
| **005** | Política de Importação da Camada CLI | Aceito | Proibição de acoplamento entre comandos CLI e detalhes internos não expostos. |
| **006** | Contêineres como Runtime Padrão | Aceito | Execução de suítes de teste pesadas em ambientes isolados por padrão. |
| **007** | Driver SQLite Puro Go | Aceito | Adoção de driver CGO-free para persistência de cache local e histórico. |
| **008** | Isolamento de Plugins | Aceito | Protocolo IPC seguro com canais delimitados para extensões externas. |
| **009** | Orçamentos de Desempenho | Aceito | Metas de latência sub-50ms para comandos CLI padrão e benchmarks em CI. |
| **010** | Plano de Controle de Documentação | Aceito | O repositório como autoridade máxima de governança e documentação viva. |
| **011** | Recusa de Telemetria Oculta | Aceito | O Prumo é 100% livre de telemetria oculta ou rastreamento remoto invasivo. |
| **012** | Mapa de Interfaces e Projeções | Aceito | Regras estritas contra ciclos de projeção em adaptadores. |
| **013** | Fundação da Interface TUI | Aceito | Arquitetura reativa baseada em Bubble Tea para o cliente terminal interativo. |
| **014** | Leituras de Protocolo e Push | Aceito | Semântica de persistência Git atômica sem mutações corrompidas. |
| **015** | Nomenclatura de Comandos | Aceito | Padronização dos verbos e substantivos no CLI `prumo`. |
| **016** | Separação entre Harness e Agent | Aceito | O `prumo` (harness/CLI) e o `prumo-agent` (`pa`, TUI/cliente interativo) são binários distintos. |
| **017** | Delegação de Subagentes em Nível Único | Aceito | Limitação da recursão de subagentes a um único nível de profundidade para evitar explosão de custos. |
| **018** | Daemon Global | Aceito | Serviço em background leve para supervisão e sincronização contínua. |
| **019** | Ciclo de Vida de Invocação de Ferramentas | Aceito | Protocolo padronizado de validação prévia, execução e envelope de resultado para ACI. |
| **020** | Roteamento de Modelos Ciente de Quota | Aceito | Fallback automático entre provedores de LLM baseado em cotas e latência. |
| **021** | Contrato de Extensões do Viewer | Aceito | Especificação do barramento de comunicação Rust para o Native Workspace Viewer. |

# Catálogo de Agentes Especialistas (39 Agentes)

O Prumo inclui **39 agentes autônomos pré-configurados**, projetados para atuar com foco cirúrgico e limites claros de responsabilidade e autoridade.

## Divisão por Áreas de Domínio

### 1. Engenharia de Sistemas, Compiladores & Runtime

| Agente | Papel | Responsabilidade Principal |
|---|---|---|
| **`compiler-engineer`** | Engenheiro de Compiladores | Frontend léxico, parser sintático, AST, checagem estática de tipos e emissores de bytecode/Wasm. |
| **`systems-architect`** | Arquiteto de Sistemas | Arquitetura de baixo nível, concorrência, alocação de memória e padrões de desempenho de hardware. |
| **`engine-engineer`** | Engenheiro de Game Engines | Loops de execução determinísticos, física, gerenciamento de entidades e tempo virtual. |
| **`renderer-engineer`** | Engenheiro de Renderização | Pipelines gráficos, shaders, backends de desenho por software/hardware e canvas. |
| **`debugger`** | Engenheiro de Depuração | Isolamento de panics, análise de core dumps, profiling e eliminação de vazamentos de memória. |
| **`performance-agent`** | Agente de Otimização | Benchmarks comparativos, profiling de CPU/memória e mitigação de regressões de performance. |
| **`scientific-computing-agent`** | Computação Científica | Algoritmos numéricos, precisão de ponto flutuante, matrizes e simulações físicas. |

### 2. Arquitetura de Software, Backend & Plataforma

| Agente | Papel | Responsabilidade Principal |
|---|---|---|
| **`architect`** | Arquiteto de Software | Coesão macro, DDD, preservação de fronteiras entre módulos e conformidade com ADRs. |
| **`backend-engineer`** | Engenheiro de Backend | APIs, contratos gRPC/REST, serviços de domínio e integridade transacional de dados. |
| **`database-engineer`** | Engenheiro de Banco de Dados | Modelagem relacional, migrações determinísticas, índices e transações ACID. |
| **`networking-engineer`** | Engenheiro de Redes | Protocolos TCP/UDP, WebSockets, serialização binária e tolerância a partições de rede. |
| **`devops-engineer`** | Engenheiro de Infraestrutura & CI | Pipelines de automação, builds herméticos, empacotamento em contêineres e deploy seguro. |
| **`editor-engineer`** | Engenheiro de IDE & Editores | Protocolo LSP, realce de sintaxe, plugins de IDE e ergonomia do ambiente de edição. |

### 3. UI/UX, Design Systems, Vetores & Motion

| Agente | Papel | Responsabilidade Principal |
|---|---|---|
| **`design-system-engineer`** | Engenheiro de Design System | Especificação de design tokens, arquitetura de temas, semântica de cores e variáveis CSS. |
| **`ui-component-engineer`** | Engenheiro de Componentes UI | Implementação de componentes de interface modulares, acessíveis e reativos. |
| **`ux-architect`** | Arquiteto de UX | Fluxos de usuário, arquitetura de informação, acessibilidade (WCAG AAA) e usabilidade. |
| **`design-researcher`** | Pesquisador de Design | Benchmarking de mercado, testes heurísticos e validação científica de padrões visuais. |
| **`svg-artist`** | Artista SVG & Ícones | Criação matemática de artes vetoriais, ilustrações SVG escaláveis e famílias de ícones. |
| **`motion-designer`** | Designer de Animação & Motion | Microinterações fluidas, curvas de easing físicas e transições de tela performáticas. |
| **`accessibility-reviewer`** | Revisor de Acessibilidade | Auditoria de contraste, suporte a leitores de tela, navegação por teclado e semântica ARIA. |

### 4. Comunicação, Marca, Publicidade & Propaganda

| Agente | Papel | Responsabilidade Principal |
|---|---|---|
| **`brand-designer`** | Designer de Marca | Identidade corporativa, tipografia de marca, paleta conceitual e manual de uso de logo. |
| **`advertising-designer`** | Designer de Campanhas & Anúncios | Peças visuais para marketing, banners, materiais de conversão e comunicação persuasiva. |
| **`creative-director`** | Diretor Criativo | Coerência estética transversal entre engenharia, design, produto e tom de voz institucional. |
| **`visual-identity-auditor`** | Auditor de Identidade Visual | Checagem de conformidade de marca e integridade estilística em todos os pontos de contato. |

### 5. Qualidade, Testes, Auditoria & Segurança

| Agente | Papel | Responsabilidade Principal |
|---|---|---|
| **`tester`** | Engenheiro de Testes | Criação de testes unitários, testes de integração e suites de conformidade diferencial. |
| **`quality-reviewer`** | Revisor de Qualidade | Auditoria de Clean Code, cobertura de testes e prevenção de dívida técnica. |
| **`security-architect`** | Arquiteto de Segurança | Modelagem de ameaças, superfícies de ataque, controle de privilégios e criptografia. |
| **`security-reviewer`** | Revisor de Segurança | Varredura de vulnerabilidades, auditoria de dependências terceiras e vetores de injeção. |
| **`isolation-auditor`** | Auditor de Isolamento | Garantia de hermeticidade de sandbox e proteção contra vazamento de variáveis de ambiente. |
| **`release-verifier`** | Verificador de Release | Checagem de critérios de release, integridade de checksums e conformidade de tags. |

### 6. Orquestração, Triagem & Exploração

| Agente | Papel | Responsabilidade Principal |
|---|---|---|
| **`explorer`** | Agente Explorador | Análise exploratória de bases de código complexas e mapeamento semântico sem mutação. |
| **`implementer`** | Implementador | Execução pragmática de diretivas de implementação guiadas por critérios de aceitação. |
| **`reviewer`** | Revisor de Código | Análise de pull requests, detecção de regressões e sugestões embasadas em evidências. |
| **`issue-triager`** | Triador de Demandas | Classificação, deduplicação e roteamento de tickets e chamados de engenharia. |
| **`issue-author`** | Autor de Issues | Elaboração de especificações detalhadas de bugs e propostas de funcionalidades. |
| **`documentation-maintainer`** | Mantenedor de Documentação | Sincronização entre código e especificações, prevenindo obsolescência de documentação. |
| **`technology-decision-agent`** | Decisões de Tecnologia | Apoio formal na redação e avaliação de Trade-offs e Registros de Decisão de Arquitetura (ADRs). |
| **`prototyper`** | Prototipador Rápido | Criação ágil de provas de conceito (PoCs) e validações experimentais de hipóteses técnicas. |

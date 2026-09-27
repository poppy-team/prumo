# Arquitetura do Prumo Framework

A arquitetura do Prumo foi projetada para garantir **longevidade, determinismo e independência de fornecedor** em ambientes de engenharia de software complexos.

## Pilares Arquiteturais

1. **Repositório Git como Fonte Durável da Verdade**: Nenhum banco de dados proprietário ou serviço em nuvem é obrigatório. O estado do projeto reside em artefatos versionáveis (Markdown, JSON, Git commits).
2. **Separação Rígida de Camadas**:
   - **Camada de Protocolo**: Metas (Goals), Planos (Plans), Tarefas (Tasks), Evidências (Evidence), Portões (Gates).
   - **Camada de Serviços & Resolução**: Resolução de modelos, riscos, skills, receitas e aplicabilidade de documentação.
   - **Camada de Harness & Execução**: Tool gateway, sandboxing de processos, análise de código e AST.
   - **Camada de Adaptadores & Projeções**: Conectores finos para Claude Code, Antigravity, Codex, Cursor, etc.
3. **Puro Go, Zero Dependências**: Implementação autossuficiente sem dependências de interpretadores de script ou ambientes virtuais.
4. **Lean Progressive Context (LPC)**: Otimização cognitiva permanente para agentes e humanos.

## Documentos de Arquitetura

- **[Visão Geral do Sistema](/architecture/overview)**: Princípios e decomposição conceitual do framework.
- **[Regras de Dependência](/architecture/dependency-rules)**: Isolamento entre pacotes e invariantes de Clean Code.
- **[Native Workspace Viewer (Rust)](/architecture/viewer)**: Substrato de alto desempenho para visualização do repositório.

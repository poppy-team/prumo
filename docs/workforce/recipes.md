# Receitas Determinísticas (20 Receitas)

As **Receitas** no Prumo são sequências operacionais padronizadas que orquestram múltiplos passos e agentes em uma esteira de execução determinística com validação de portão (gate).

## Catálogo de Receitas

| Receita | Descrição | Agentes Envolvidos |
|---|---|---|
| **`architecture-change`** | Alteração estrutural ou de dependências com redação prévia de ADR e auditoria de contratos. | `architect`, `quality-reviewer` |
| **`brand-creation`** | Criação completa de identidade de marca: estética, logo SVG, paleta OKLCH, tipografia fluida e manual. | `creative-director`, `brand-designer`, `svg-artist` |
| **`bug-fix`** | Correção de defeito orientada a evidências (reprodução determinística com teste falho antes da correção). | `debugger`, `implementer`, `tester` |
| **`compiler-change`** | Evolução de sintaxe, AST ou regras de compilação com verificação diferencial de paridade. | `compiler-engineer`, `tester` |
| **`compiler-conformance`** | Execução de suítes exaustivas de conformidade e testes contra baselines canônicos. | `compiler-engineer`, `release-verifier` |
| **`design-system-foundation`** | Arquitetura de design system: tokens W3C DTCG, paletas OKLCH, componentes atômicos e styleguide vivo. | `design-system-engineer`, `ui-component-engineer`, `accessibility-reviewer` |
| **`documentation-refactor`** | Atualização de documentação canônica, auditoria de autoridade e mitigação de drifts. | `documentation-maintainer`, `reviewer` |
| **`engine-renderer`** | Modificações no subsistema de renderização gráfica, shaders ou loop principal. | `engine-engineer`, `renderer-engineer` |
| **`feature-standard`** | Implementação de funcionalidade padrão com Metas, Planos, testes e documentação. | `architect`, `implementer`, `tester` |
| **`github-issue`** | Ciclo completo de resolução de issue de GitHub: triagem, plano, implementação e PR. | `issue-triager`, `implementer`, `reviewer` |
| **`icon-library-creation`** | Produção e engenharia de biblioteca de ícones: grid canônico, otimização de paths SVG e empacotamento. | `svg-artist`, `design-system-engineer`, `accessibility-reviewer` |
| **`marketing-campaign`** | Esteira de campanhas publicitárias com metodologia AIDA: criativos multiformato e landing view responsiva. | `creative-director`, `advertising-designer`, `frontend-engineer` |
| **`multiplayer-feature`** | Desenvolvimento de mecânica em rede com tratamento de sincronização e latência. | `networking-engineer`, `tester` |
| **`project-bootstrap`** | Inicialização canônica de repositório, configuração de perfis e primeiras Metas. | `architect`, `devops-engineer` |
| **`release`** | Verificação formal de pré-requisitos de release, changelog, tags e geração de binários. | `release-verifier`, `security-reviewer` |
| **`security-audit-release`** | Auditoria formal de segurança prévia ao lançamento com checagem de privilégios. | `security-architect`, `security-reviewer` |
| **`security-review`** | Revisão periódica de superfícies de ataque, dependências e isolamento de segredos. | `security-reviewer`, `isolation-auditor` |
| **`ui-feature`** | Construção de componentes de interface com design tokens e revisão de usabilidade. | `ui-component-engineer`, `ux-architect` |
| **`ui-review`** | Auditoria visual, testes de responsividade e conformidade de acessibilidade (WCAG). | `accessibility-reviewer`, `visual-identity-auditor` |
| **`web-feature`** | Implementação de serviço web, endpoints REST/HTTP e validação de payloads. | `backend-engineer`, `frontend-engineer` |

## Como Executar uma Receita

```bash
# Executar a receita de bug-fix para um caso específico
prumo run --recipe bug-fix --param issue=142

# Executar a esteira de verificação de release
prumo run --recipe release --param version=0.6.0
```

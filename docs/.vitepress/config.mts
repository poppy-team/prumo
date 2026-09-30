import { defineConfig } from 'vitepress'

export default defineConfig({
  cleanUrls: true,
  lastUpdated: true,
  ignoreDeadLinks: true,
  srcExclude: [
    '**/framework/**',
    '**/harness/open-work.md',
    '**/harness/gap-register.md',
    '**/harness/canonical-implementation-matrix.md',
    '**/harness/market-comparison-and-enhancements.md',
    '**/harness/roadmap-status.md',
    '**/harness/autoupdate.md',
    '**/harness/reconnect-replay.md',
    '**/harness/providers.md',
    '**/harness/daemon.md',
    '**/harness/human-docs.md',
    '**/harness/security.md',
    '**/harness/context-knowledge.md',
    '**/harness/workforce-handoff.md',
    '**/harness/agent-runtime.md',
    '**/harness/promotion-report.md',
    '**/harness/capability-skill-gap-register.md',
    '**/harness/overview.md',
    '**/harness/specs/**',
    '**/governance/m*-exit-gate.md',
    '**/governance/documentation-dogfood.md',
    '**/governance/documentation-system.md',
    '**/governance/consolidation-crosswalk.md',
    '**/governance/release-policy.md',
    '**/governance/repository-governance.md',
    '**/architecture/accessibility.md',
    '**/architecture/documentation-architecture-v2.md',
    '**/architecture/documentation-compiler.md',
    '**/architecture/documentation-control-plane.md',
    '**/architecture/prumo-native-workspace-viewer.md',
    '**/architecture/shared-product-contract.md',
    '**/architecture/unified-architecture.md',
    '**/architecture/visual-constitution.md',
    '**/canon/**',
    '**/contracts/**',
    '**/conformance/**',
    '**/journal/**',
    '**/operations/**',
    '**/installation/**',
    '**/product/**',
    '**/reference/**',
    '**/security/**',
    '**/stdlib/**',
    '**/testing/**',
    '**/waves/**',
    '**/development/**',
    '**/diagnostics/**',
    '**/adr/**',
    '**/systems/**',
    '**/ui-ux/**',
    '**/skills/**',
    '**/migration/**',
    '**/runtime/**',
    '**/quality/**',
    '**/authoring/**',
    '**/integration/**',
    '**/profiles/**',
    '**/user-guide/**',
    '**/README.md',
    '**/PRUMO.md',
    '**/AI_WORKFORCE.md',
    '**/ARCHITECTURE.md',
    '**/DOCUMENTATION_SYSTEM.md',
    '**/EVENT_PROTOCOL.md',
    '**/FRAMEWORK_CHANGE_PROPOSAL.md',
    '**/GOAL_SYSTEM.md',
    '**/LEAN_PROGRESSIVE_CONTEXT.md',
    '**/LIVRO_VIVO_HUB.md',
    '**/PORTABILITY.md',
    '**/PROJECT_INTELLIGENCE.md',
    '**/PROJECT_ORCHESTRATION_PROTOCOL.md',
    '**/QUALITY.md',
    '**/TRUST_MODEL.md',
    '**/USAGE.md'
  ],

  head: [
    ['link', { rel: 'icon', type: 'image/svg+xml', href: '/assets/logo.svg' }],
    ['meta', { name: 'theme-color', content: '#2563eb' }],
    ['meta', { name: 'og:type', content: 'website' }],
    ['meta', { name: 'og:site_name', content: 'Prumo Engineering Framework' }]
  ],

  themeConfig: {
    logo: '/assets/logo.svg',
    siteTitle: 'Prumo',
    socialLinks: [
      { icon: 'github', link: 'https://github.com/poppy-team/prumo' }
    ],
    search: {
      provider: 'local',
      options: {
        locales: {
          root: {
            translations: {
              button: {
                buttonText: 'Pesquisar documentação...',
                buttonAriaLabel: 'Pesquisar documentação'
              },
              modal: {
                noResultsText: 'Nenhum resultado encontrado para',
                resetButtonTitle: 'Limpar pesquisa',
                footer: {
                  selectText: 'para selecionar',
                  navigateText: 'para navegar',
                  closeText: 'para fechar'
                }
              }
            }
          },
          en: {
            translations: {
              button: {
                buttonText: 'Search documentation...',
                buttonAriaLabel: 'Search documentation'
              },
              modal: {
                noResultsText: 'No results found for',
                resetButtonTitle: 'Reset search',
                footer: {
                  selectText: 'to select',
                  navigateText: 'to navigate',
                  closeText: 'to close'
                }
              }
            }
          }
        }
      }
    }
  },

  locales: {
    root: {
      label: 'Português',
      lang: 'pt-BR',
      title: 'Prumo',
      description: 'Harness de Engenharia e Protocolo Nativo Git para Humanos & Agentes de IA',
      themeConfig: {
        nav: [
          { text: 'Início', link: '/' },
          { text: 'Começando', link: '/getting-started/' },
          {
            text: 'Manual',
            items: [
              { text: 'Visão Geral do Manual', link: '/manual/' },
              { text: 'Uso & Comandos', link: '/manual/usage' },
              { text: 'Instalação Oficial', link: '/manual/installation' },
              { text: 'Desinstalação Reversível', link: '/manual/uninstallation' },
              { text: 'Conectores & Adapters', link: '/manual/connectors' }
            ]
          },
          {
            text: 'Harness & Execução',
            items: [
              { text: 'Arquitetura do Harness', link: '/harness/' },
              { text: 'Ferramental ACI & Sandboxing', link: '/harness/aci-tools' },
              { text: 'Diretivas & Task DAGs', link: '/harness/directives' }
            ]
          },
          {
            text: 'Workforce',
            items: [
              { text: 'Catálogo da Força de Trabalho', link: '/workforce/' },
              { text: '189 Skills de Precisão', link: '/workforce/skills' },
              { text: '39 Agentes Especialistas', link: '/workforce/agents' },
              { text: '20 Receitas Determinísticas', link: '/workforce/recipes' }
            ]
          },
          { text: 'Ferramentas CLI', link: '/tools/' },
          {
            text: 'Engenharia & Governança',
            items: [
              { text: 'Macroarquitetura do Sistema', link: '/architecture/' },
              { text: 'Regras de Dependência', link: '/architecture/dependency-rules' },
              { text: 'Native Workspace Viewer (Rust)', link: '/architecture/viewer' },
              { text: 'Governança & Livro Vivo', link: '/governance/' },
              { text: 'Autoridade & Drift de Docs', link: '/governance/authority' },
              { text: 'Lean Progressive Context (LPC)', link: '/governance/lpc' },
              { text: 'Decisões Arquiteturais (ADRs)', link: '/decisions/' }
            ]
          }
        ],

        sidebar: {
          '/getting-started/': [
            {
              text: 'Começando com Prumo',
              items: [
                { text: 'Visão Geral', link: '/getting-started/' },
                { text: 'Instalação Rápida', link: '/getting-started/installation' },
                { text: 'Primeiro Projeto (5 min)', link: '/getting-started/first-project' },
                { text: 'Adoção Brownfield', link: '/getting-started/adoption' },
                { text: 'Conceitos Fundamentais', link: '/getting-started/concepts' }
              ]
            }
          ],
          '/manual/': [
            {
              text: 'Manual do Usuário',
              items: [
                { text: 'Visão Geral do Manual', link: '/manual/' },
                { text: 'Instalação Detalhada', link: '/manual/installation' },
                { text: 'Desinstalação Limpa', link: '/manual/uninstallation' },
                { text: 'Guia de Uso & Comandos', link: '/manual/usage' },
                { text: 'Conectores & Plataformas', link: '/manual/connectors' }
              ]
            }
          ],
          '/harness/': [
            {
              text: 'Harness & Execução Autônoma',
              items: [
                { text: 'Visão Geral do Harness', link: '/harness/' },
                { text: 'Ferramentas ACI & Sandbox', link: '/harness/aci-tools' },
                { text: 'Diretivas & Execução', link: '/harness/directives' }
              ]
            }
          ],
          '/workforce/': [
            {
              text: 'Workforce & Catálogo Canônico',
              items: [
                { text: 'Visão Geral da Força de Trabalho', link: '/workforce/' },
                { text: '189 Skills Catalogadas', link: '/workforce/skills' },
                { text: '39 Agentes Especialistas', link: '/workforce/agents' },
                { text: '20 Receitas Determinísticas', link: '/workforce/recipes' }
              ]
            }
          ],
          '/tools/': [
            {
              text: 'Ferramentas do Desenvolvedor',
              items: [
                { text: 'Visão Geral do CLI', link: '/tools/' },
                { text: 'Referência de Comandos', link: '/tools/cli-reference' },
                { text: 'Diagnóstico & Doctor', link: '/tools/doctor' }
              ]
            }
          ],
          '/architecture/': [
            {
              text: 'Arquitetura do Prumo',
              items: [
                { text: 'Macroarquitetura do Sistema', link: '/architecture/' },
                { text: 'Regras de Dependência', link: '/architecture/dependency-rules' },
                { text: 'Native Workspace Viewer (Rust)', link: '/architecture/viewer' }
              ]
            }
          ],
          '/governance/': [
            {
              text: 'Governança & Livro Vivo',
              items: [
                { text: 'Visão Geral de Governança', link: '/governance/' },
                { text: 'Autoridade & Drift de Docs', link: '/governance/authority' },
                { text: 'Metodologia Lean Progressive Context', link: '/governance/lpc' },
                { text: 'Modelo de Confiança & Segurança', link: '/governance/trust-model' }
              ]
            }
          ],
          '/decisions/': [
            {
              text: 'Decisões Arquiteturais (ADRs)',
              items: [
                { text: 'Índice de Decisões', link: '/decisions/' }
              ]
            }
          ]
        },

        footer: {
          message: 'Distribuído sob licença MIT. Governança com Prumo v0.6.0.',
          copyright: 'Copyright © 2026 Poppy Team & Contribuidores do Prumo'
        },

        docFooter: {
          prev: 'Página anterior',
          next: 'Próxima página'
        },

        outline: {
          label: 'Nesta página',
          level: [2, 3]
        }
      }
    },

    en: {
      label: 'English',
      lang: 'en-US',
      link: '/en/',
      title: 'Prumo',
      description: 'Engineering Harness and Git-Native Protocol for Humans & AI Agents',
      themeConfig: {
        nav: [
          { text: 'Home', link: '/en/' },
          { text: 'Getting Started', link: '/en/getting-started/' },
          { text: 'Manual', link: '/en/manual/usage' },
          { text: 'Harness', link: '/en/harness/' },
          { text: 'Workforce', link: '/en/workforce/' },
          { text: 'Tools', link: '/en/tools/' },
          { text: 'Architecture', link: '/en/architecture/' },
          { text: 'Governance', link: '/en/governance/' }
        ],

        sidebar: {
          '/en/getting-started/': [
            {
              text: 'Getting Started',
              items: [
                { text: 'Overview', link: '/en/getting-started/' },
                { text: 'Quick Installation', link: '/en/getting-started/installation' },
                { text: 'First Project (5 min)', link: '/en/getting-started/first-project' },
                { text: 'Brownfield Adoption', link: '/en/getting-started/adoption' },
                { text: 'Core Concepts', link: '/en/getting-started/concepts' }
              ]
            }
          ],
          '/en/manual/': [
            {
              text: 'User Manual',
              items: [
                { text: 'Detailed Installation', link: '/en/manual/installation' },
                { text: 'Clean Uninstallation', link: '/en/manual/uninstallation' },
                { text: 'Usage & Commands Guide', link: '/en/manual/usage' },
                { text: 'Connectors & Platforms', link: '/en/manual/connectors' }
              ]
            }
          ],
          '/en/harness/': [
            {
              text: 'Harness & Execution',
              items: [
                { text: 'Harness Overview', link: '/en/harness/' },
                { text: 'ACI Tooling & Sandboxing', link: '/en/harness/aci-tools' },
                { text: 'Directives & Task DAGs', link: '/en/harness/directives' }
              ]
            }
          ],
          '/en/workforce/': [
            {
              text: 'Workforce Catalog',
              items: [
                { text: 'Overview', link: '/en/workforce/' },
                { text: '189 Precision Skills', link: '/en/workforce/skills' },
                { text: '39 Specialist Agents', link: '/en/workforce/agents' },
                { text: '20 Deterministic Recipes', link: '/en/workforce/recipes' }
              ]
            }
          ],
          '/en/tools/': [
            {
              text: 'Developer Tools',
              items: [
                { text: 'CLI Reference', link: '/en/tools/' },
                { text: 'Command Reference', link: '/en/tools/cli-reference' },
                { text: 'Diagnostics & Doctor', link: '/en/tools/doctor' }
              ]
            }
          ],
          '/en/architecture/': [
            {
              text: 'Architecture',
              items: [
                { text: 'Macroarchitecture', link: '/en/architecture/' },
                { text: 'System Overview', link: '/en/architecture/overview' },
                { text: 'Dependency Rules', link: '/en/architecture/dependency-rules' },
                { text: 'Native Workspace Viewer (Rust)', link: '/en/architecture/viewer' }
              ]
            }
          ],
          '/en/governance/': [
            {
              text: 'Governance',
              items: [
                { text: 'Authority & Living Book', link: '/en/governance/' },
                { text: 'Authority & Docs Drift', link: '/en/governance/authority' },
                { text: 'Lean Progressive Context (LPC)', link: '/en/governance/lpc' },
                { text: 'Trust & Security Model', link: '/en/governance/trust-model' }
              ]
            }
          ]
        },

        footer: {
          message: 'Released under the MIT License. Governed by Prumo v0.6.0.',
          copyright: 'Copyright © 2026 Poppy Team & Prumo Contributors'
        },

        docFooter: {
          prev: 'Previous page',
          next: 'Next page'
        },

        outline: {
          label: 'On this page',
          level: [2, 3]
        }
      }
    }
  }
})

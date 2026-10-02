package cliops

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"time"

	prumo "github.com/raillen/prumo"
	"github.com/raillen/prumo/internal/protocol"
)

type scaffoldFile struct {
	relPath  string
	content  func(projectName, date string) string
	optional bool
}

func scaffoldBaselineDocuments(root, projectName string) error {
	nowDate := time.Now().UTC().Format("2006-01-02")
	files := getScaffoldTemplates()

	for _, f := range files {
		target := filepath.Join(root, f.relPath)
		if f.optional {
			if _, err := os.Stat(target); err == nil {
				continue
			}
		}
		if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
			return err
		}
		content := f.content(projectName, nowDate)
		if err := os.WriteFile(target, []byte(content), 0644); err != nil {
			return err
		}
	}
	return nil
}

// scaffoldDocumentationControlPlane seeds the machine-readable documentation
// contracts a project needs before `docs audit`, `docs readiness`,
// `docs authority` and the Living Plan loop can run at all.
//
// This was the single largest greenfield blocker: `prumo init` produced a
// project where every documentation command failed with a missing-file error,
// and the only recovery path (the adoption engine) wrote a bindings format the
// engine could not read. A scaffolded project now arrives with a working
// control plane and an honest, explicit "not yet answered" readiness state
// instead of a crash.
func scaffoldDocumentationControlPlane(root string) error {
	for _, dir := range []string{
		filepath.Join(root, "docs", "contracts"),
		filepath.Join(root, "docs", "profiles"),
	} {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return err
		}
	}

	// Materialize the framework registry locally so a project is self-describing
	// and auditable without relying on the binary's embedded copy.
	for _, pair := range []struct {
		name string
		data []byte
	}{
		{filepath.Join(root, "docs", "contracts", "builtin.json"), embeddedFile(prumo.EmbeddedDocContracts(), "builtin.json")},
		{filepath.Join(root, "docs", "profiles", "builtin.json"), embeddedFile(prumo.EmbeddedDocProfiles(), "builtin.json")},
	} {
		if len(pair.data) == 0 {
			continue
		}
		if _, err := os.Stat(pair.name); err == nil {
			// A project that already carries its own registry keeps it.
			continue
		}
		if err := os.WriteFile(pair.name, pair.data, 0644); err != nil {
			return err
		}
	}

	// Bind the canonical documents the scaffold actually produced. Without
	// bindings, every applicable contract reads as unbound and readiness can
	// never move; with them, `docs audit` reports real, actionable coverage.
	bindings := []map[string]any{
		{
			"contract_id": "product.vision",
			"sources":     []string{"docs/product/vision.md"},
			"ownership":   "human-maintained",
			"authority":   "canonical-documentation",
		},
		{
			"contract_id": "project.scope",
			"sources":     []string{"docs/product/scope.md"},
			"ownership":   "human-maintained",
			"authority":   "canonical-documentation",
		},
		{
			"contract_id": "architecture.system",
			"sources":     []string{"docs/architecture/overview.md", "docs/architecture/clean-code-contract.md"},
			"ownership":   "human-maintained",
			"authority":   "canonical-documentation",
		},
		{
			"contract_id": "testing.strategy",
			"sources":     []string{"docs/development/testing-strategy.md"},
			"ownership":   "human-maintained",
			"authority":   "canonical-documentation",
		},
		{
			"contract_id": "security.trust",
			"sources":     []string{"docs/security/threat-model.md", "docs/security/security-contract.md"},
			"ownership":   "human-maintained",
			"authority":   "canonical-documentation",
		},
	}
	bindingsPath := filepath.Join(root, "docs", "contracts", "bindings.json")
	if _, err := os.Stat(bindingsPath); err != nil {
		data, err := json.MarshalIndent(bindings, "", "  ")
		if err != nil {
			return err
		}
		if err := os.WriteFile(bindingsPath, append(data, '\n'), 0644); err != nil {
			return err
		}
	}

	// Authority map: the classification `prumo docs authority` verifies. Scaffolding
	// it here is what makes the documented CI gate runnable on a new project.
	authorityPath := filepath.Join(root, "docs", "AUTHORITY_MAP.json")
	if _, err := os.Stat(authorityPath); err != nil {
		authority := map[string]any{
			"version":         1,
			"current_version": protocol.CLIVersion,
			"generated_by":    "prumo init",
			"documents": []map[string]any{
				{"path": "docs/**", "role": "canonical"},
				{"path": "docs/governance/**", "role": "historical", "drift_exempt": true, "drift_exempt_reason": "Governance records reference the versions that were current when they were written."},
				{"path": "docs/adr/**", "role": "historical", "drift_exempt": true, "drift_exempt_reason": "ADRs are immutable decision records and legitimately reference superseded versions."},
				{"path": "AGENTS.md", "role": "projection", "canonical_source": "docs/PRUMO.md"},
				{"path": "CLAUDE.md", "role": "projection", "canonical_source": "docs/PRUMO.md"},
				{"path": "GEMINI.md", "role": "projection", "canonical_source": "docs/PRUMO.md"},
				{"path": "*.md", "role": "canonical"},
			},
		}
		data, err := json.MarshalIndent(authority, "", "  ")
		if err != nil {
			return err
		}
		if err := os.WriteFile(authorityPath, append(data, '\n'), 0644); err != nil {
			return err
		}
	}
	return nil
}

func embeddedFile(fsys fs.FS, name string) []byte {
	data, err := fs.ReadFile(fsys, name)
	if err != nil {
		return nil
	}
	return data
}

func getScaffoldTemplates() []scaffoldFile {
	return []scaffoldFile{
		{
			relPath:  "README.md",
			optional: true,
			content: func(projectName, _ string) string {
				return fmt.Sprintf(`# %s — Prumo

Este projeto utiliza o **Prumo v%s** para colaboração humano-agente com governança, orquestração e contexto enxuto (Lean Progressive Context).

## Estrutura do Projeto

`+"```text"+`
├── prumo.json                       # Manifesto canônico do projeto (protocolo v3)
├── ENTRYPOINT.md                    # Roteador de contexto Lean Progressive Context
├── PROJECT_STATE.md                 # Estado operacional, fase ativa e recuperação
├── CHANGELOG.md                     # Histórico estruturado de alterações
├── README.md                        # Visão geral do projeto e guia de navegação
├── .ai/                             # Recursos e orquestração de IA (agentes, metas, planos)
│   └── README.md
├── .prumo/                          # Metadados de runtime, cache e inteligência
│   └── README.md
└── docs/                            # Documentação canônica orientada a papéis
    ├── README.md
    ├── PRUMO.md                     # Roteador de intenção (mapa de documentação)
    ├── architecture/                # Visão de sistemas, Clean Code e ADRs
    ├── product/                     # Visão de produto e escopo
    ├── development/                 # Padrões de código e estratégia exaustiva de testes
    ├── operations/                  # Guias de deploy e observabilidade
    ├── security/                    # Modelo de ameaças e contrato de segurança
    └── governance/                  # Políticas de repositório e branches
`+"```"+`

## Como Usar

1. **Validação do Projeto**:
   `+"```bash"+`
   prumo validate .
   `+"```"+`
2. **Diagnóstico do Projeto**:
   `+"```bash"+`
   prumo doctor .
   `+"```"+`
3. **Compilar Adaptador para seu Agente**:
   `+"```bash"+`
   prumo compile --target antigravity .
   prumo compile --target codex .
   prumo compile --target claude-code .
   `+"```"+`
4. **Ciclo de Desenvolvimento com Metas**:
   Consulte [`+"`docs/development/testing-strategy.md`"+`](docs/development/testing-strategy.md) para o ciclo TDD/BDD com testes exaustivos.
`, projectName, protocol.CLIVersion)
			},
		},
		{
			relPath: "CHANGELOG.md",
			content: func(_, date string) string {
				return fmt.Sprintf(`# Changelog

Todas as alterações notáveis deste projeto são documentadas neste arquivo.
O formato baseia-se no [Keep a Changelog](https://keepachangelog.com/pt-BR/1.0.0/) e adere ao [Semantic Versioning](https://semver.org/lang/pt-BR/).

## [0.1.0] - %s

### Adicionado
- Inicialização da estrutura canônica do Prumo v%s.
- Configuração do manifesto `+"`prumo.json`"+` e orquestração `+"`.ai/`"+`.
- Definição da hierarquia de documentação canônica em `+"`docs/`"+`.
- Contrato estrito de arquitetura em `+"`docs/architecture/clean-code-contract.md`"+`.
- Estratégia de testes exaustivos em `+"`docs/development/testing-strategy.md`"+` (unitários, integração, conformidade, segurança SAST/secrets, performance/stress, UI).
- Política de documentação mandatória com `+"`README.md`"+` explicativo em cada diretório do projeto.
`, date, protocol.CLIVersion)
			},
		},
		{
			relPath: ".ai/README.md",
			content: func(_, _ string) string {
				return `# Diretório de Recursos de IA (` + "`.ai/`" + `)

## O que é este diretório?
A pasta ` + "`.ai/`" + ` armazena as definições canônicas de workforce (agentes, habilidades, receitas de execução) e as metas operacionais (Goals e Planos) vinculadas ao projeto.

## Para que serve?
Permite que o Prumo resolva deterministicamente quais agentes, skills e receitas de trabalho estão autorizados e disponíveis para executar tarefas no projeto, mantendo políticas de modelos, integridade de locks e rastreabilidade de evidências.

## Inventário de Arquivos e Subdiretórios
- ` + "`agents/`" + `: Agentes resolvidos e manifestos de papéis.
- ` + "`skills/`" + `: Habilidades e procedimentos atribuídos aos agentes.
- ` + "`recipes/`" + `: Fluxos de trabalho e receitas de automação multi-etapas.
- ` + "`goals/`" + `: Metas do projeto organizadas por fases (` + "`P00`" + `, ` + "`P01`" + `, etc.) com hashes de integridade SHA-256.
- ` + "`orchestration/`" + `: Políticas de roteamento de LLMs, tolerância a falhas e scorecards de desempenho.
`
			},
		},
		{
			relPath: ".ai/agents/README.md",
			content: func(_, _ string) string {
				return `# Diretório de Agentes (` + "`.ai/agents/`" + `)

## O que é este diretório?
Contém o registro dos agentes de IA designados para atuar neste projeto de acordo com o perfil resolvido.

## Para que serve?
Define os limites de atuação, papéis, responsabilidades e permissões de cada agente que colabora com a equipe humana.

## Inventário
- ` + "`manifest.json`" + `: Manifesto com os IDs dos agentes ativos e a justificativa técnica para a inclusão de cada um no projeto.
`
			},
		},
		{
			relPath: ".ai/skills/README.md",
			content: func(_, _ string) string {
				return `# Diretório de Habilidades (` + "`.ai/skills/`" + `)

## O que é este diretório?
Registra o catálogo de habilidades operacionais (skills) habilitadas para este projeto.

## Para que serve?
As skills fornecem instruções procedurais, referências, checklists e ferramentas práticas para os agentes executarem tarefas específicas (como ` + "`clean-code`" + `, ` + "`testing-quality`" + `, ` + "`secure-coding`" + `).

## Inventário
- ` + "`manifest.json`" + `: Manifesto com os IDs das skills habilitadas e suas justificativas de inclusão no escopo do projeto.
`
			},
		},
		{
			relPath: ".ai/recipes/README.md",
			content: func(_, _ string) string {
				return `# Diretório de Receitas (` + "`.ai/recipes/`" + `)

## O que é este diretório?
Armazena os fluxos de trabalho estruturados e multi-etapas (Recipes) aplicáveis às operações do projeto.

## Para que serve?
As receitas coordenam múltiplos agentes e habilidades para resolver objetivos complexos com gates de aprovação, exigência de evidências e paradas seguras (ex: ` + "`feature-standard`" + `, ` + "`bug-fix`" + `, ` + "`security-review`" + `).

## Inventário
- ` + "`manifest.json`" + `: Manifesto com as receitas aprovadas e ativas para uso no projeto.
`
			},
		},
		{
			relPath: ".ai/goals/README.md",
			content: func(_, _ string) string {
				return `# Diretório de Metas (` + "`.ai/goals/`" + `)

## O que é este diretório?
Contém as metas formais de desenvolvimento (Goals) do projeto, organizadas hierarquicamente por fases macro (` + "`P00`" + `, ` + "`P01`" + `, etc.).

## Para que serve?
No Prumo, nenhum código é escrito sem um Goal explicitamente definido e medido. Metas no estado LOCKED possuem um hash de integridade SHA-256 e só podem ser modificadas através do comando formal ` + "`prumo goal amend`" + `.

## Inventário
- ` + "`P00/`" + `: Fase de Fundação e Arquitetura do projeto.
- ` + "`P01/`" + ` (quando aplicável): Fases subsequentes de entrega de valor e funcionalidades.
`
			},
		},
		{
			relPath: ".ai/goals/P00/README.md",
			content: func(_, _ string) string {
				return `# Fase P00: Fundação e Arquitetura (` + "`.ai/goals/P00/`" + `)

## O que é este diretório?
Armazena os Goals pertencentes à Fase P00 (Foundation).

## Para que serve?
A Fase P00 estabelece as bases invioláveis do projeto: estrutura de repositório, contratos de interface, infraestrutura de testes, pipelines de CI e definição de segurança antes de qualquer implementação de funcionalidade de usuário.

## Inventário
- Arquivos de metas no formato ` + "`<id>.json`" + ` (ex: ` + "`P00-G01.json`" + `), contendo objetivo, escopo, critérios de aceitação e digest criptográfico.
`
			},
		},
		{
			relPath: ".ai/orchestration/README.md",
			content: func(_, _ string) string {
				return `# Diretório de Orquestração (` + "`.ai/orchestration/`" + `)

## O que é este diretório?
Concentra as diretrizes de governança e parametrização do ecossistema de inteligência artificial do projeto.

## Para que serve?
Garante previsibilidade, controle orçamentário e resiliência nas chamadas aos modelos de linguagem através de regras formais de fallback, limites de tokens e avaliação empírica de acurácia.

## Inventário
- ` + "`model-policy.json`" + `: Mapeamento de papéis (geral, arquitetura, testes, revisão) para modelos recomendados.
- ` + "`orchestrator.json`" + `: Parâmetros de autonomia e metodologia de contexto (POP / LPC).
- ` + "`fallbacks.json`" + `: Regras de degradação graciosa e escalação humana em caso de falha de provedores.
- ` + "`model-scorecard.json`" + `: Registro histórico de desempenho empírico dos modelos testados no projeto.
`
			},
		},
		{
			relPath: ".prumo/README.md",
			content: func(_, _ string) string {
				return `# Diretório Operacional do Prumo (` + "`.prumo/`" + `)

## O que é este diretório?
Contém metadados de execução, caches de desempenho, histórico analítico e estado derivado mantidos pelo Prumo CLI.

## Para que serve?
Isola todo estado temporário ou derivado do código-fonte canônico. Dados em ` + "`.prumo/runtime/`" + ` e ` + "`.prumo/cache/`" + ` são voláteis e não devem ser versionados no Git (com exceção do histórico estruturado em ` + "`history/`" + `).

## Inventário
- ` + "`cache/`" + `: Cache de resoluções de esquemas e pacotes para acelerar comandos do CLI.
- ` + "`history/`" + `: Histórico cumulativo de inteligência do projeto, tokens consumidos e métricas.
- ` + "`runtime/`" + `: Checkpoints de execução, sessões ativas e cápsulas de contexto.
`
			},
		},
		{
			relPath: ".prumo/runtime/README.md",
			content: func(_, _ string) string {
				return `# Estado de Execução (` + "`.prumo/runtime/`" + `)

## O que é este diretório?
Área de estado volátil onde o motor de execução (Control Plane) mantém checkpoints e planos dinâmicos (Living Plan).

## Para que serve?
Permite pausas, retentativas e retomadas resilientes de sessões de agentes sem perda de contexto ou corrupção do repositório.

## Inventário
- Checkpoints de execução, logs de sessão e artefatos de compilação temporários. Todos são derivados e não canônicos.
`
			},
		},
		{
			relPath: ".prumo/cache/README.md",
			content: func(_, _ string) string {
				return `# Cache Local (` + "`.prumo/cache/`" + `)

## O que é este diretório?
Área de cache local de disco utilizada pelo Prumo CLI.

## Para que serve?
Armazena dados indexados de compilação e esquemas para evitar acessos repetitivos ao disco e manter execuções de comandos sub-milissegundo.

## Inventário
- Arquivos de cache derivados. Este diretório é ignorado pelo Git e pode ser apagado a qualquer momento com segurança (` + "`prumo uninstall --purge-cache`" + `).
`
			},
		},
		{
			relPath: ".prumo/history/README.md",
			content: func(_, _ string) string {
				return `# Histórico e Inteligência (` + "`.prumo/history/`" + `)

## O que é este diretório?
Repositório de inteligência durável e telemetria operacional mantido pelo Prumo.

## Para que serve?
Registra custos de LLM (tokens de entrada/saída), número de iterações por tarefa, débitos técnicos identificados e histórico de decisões tomadas.

## Inventário
- ` + "`project-intelligence.json`" + `: Base de dados estruturada em JSON com métricas consolidadas de tarefas e consumo de recursos.
`
			},
		},
		{
			relPath: ".prumo/repository/README.md",
			content: func(_, _ string) string {
				return `# Governança de Repositório (` + "`.prumo/repository/`" + `)

## O que é este diretório?
Contém a política de governança do repositório usada pelos gates de branch, commit, PR e merge.

## Arquivos
- ` + "`policy.json`" + `: Política de governança do repositório avaliada por ` + "`prumo repo policy`" + `.
`
			},
		},
		{
			relPath: "docs/README.md",
			content: func(_, _ string) string {
				return `# Documentação Canônica do Projeto (` + "`docs/`" + `)

## O que é este diretório?
A pasta ` + "`docs/`" + ` é a fonte canônica da verdade para todas as especificações de engenharia, produto, arquitetura, testes e governança do projeto.

## Para que serve?
Implementa o princípio de **Documentação Canônica Viva**: o repositório é autossuficiente e todo o conhecimento técnico essencial reside diretamente no código e em arquivos Markdown padronizados.

## Roteador Central
Consulte [` + "`PRUMO.md`" + `](PRUMO.md) como ponto de entrada para navegação guiada por intenção (usuário, desenvolvedor, operador, agente).

## Inventário de Subdiretórios
- ` + "`architecture/`" + `: Arquitetura do sistema, boundaries, contratos de Clean Code e ADRs.
- ` + "`product/`" + `: Visão de produto, proposta de valor e escopo delimitado.
- ` + "`development/`" + `: Padrões de codificação e estratégia exaustiva de testes (TDD/BDD).
- ` + "`operations/`" + `: Procedimentos de deploy, runbooks e observabilidade.
- ` + "`security/`" + `: Modelagem de ameaças (STRIDE), políticas e contrato de segurança.
- ` + "`governance/`" + `: Políticas de branches, pull requests e regras de conformidade.
`
			},
		},
		{
			relPath: "docs/architecture/README.md",
			content: func(_, _ string) string {
				return `# Arquitetura de Sistemas (` + "`docs/architecture/`" + `)

## O que é este diretório?
Contém as definições de topologia, limites de domínio (boundaries), regras de composição modular e registros de decisões arquiteturais (ADRs).

## Para que serve?
Garante que o sistema mantenha alta coesão e baixo acoplamento durante todo o seu ciclo de evolução, prevenindo dependências circulares e degradação arquitetural.

## Inventário
- ` + "`overview.md`" + `: Visão geral da topologia, camadas e fluxo de dados.
- ` + "`clean-code-contract.md`" + `: Contrato estrito de engenharia de software e Clean Code.
- ` + "`adr/`" + `: Registros formais de decisões arquiteturais (Architectural Decision Records).
`
			},
		},
		{
			relPath: "docs/architecture/clean-code-contract.md",
			content: func(_, _ string) string {
				return `# Contrato de Arquitetura e Clean Code

Este documento define o **contrato mandatório de engenharia de software** para todas as implementações deste projeto.

## 1. Princípios de Clean Code

1. **Responsabilidades Explícitas (SRP)**: Cada módulo, classe ou pacote possui uma única razão para mudar.
2. **Alta Coesão e Baixo Acoplamento**: Módulos devem ser auto-contidos e interagir apenas através de interfaces abstratas ou contratos tipados.
3. **Nomes Expressivos de Domínio**: Variáveis, funções e tipos devem refletir a linguagem ubíqua do negócio. Não utilize nomes genéricos como ` + "`manager`" + `, ` + "`helper`" + `, ` + "`utils`" + ` ou ` + "`data`" + `.
4. **Funções Pequenas e Focadas**: Funções devem realizar apenas uma ação lógica e caber idealmente em uma tela de visualização.
5. **Erros Explícitos**: Proibido suprimir exceções silenciosamente (` + "`bare except`" + `, ignorar erros). Todo erro deve ser tratado, encapsulado ou propagado com contexto.
6. **Zero Abstração Especulativa (YAGNI)**: Implemente abstrações somente quando houver dois ou mais casos de uso concretos comprovados.

## 2. Direção das Dependências (Clean Architecture)

- O fluxo de dependência aponta sempre **para dentro**, em direção às regras de negócio essenciais.
- Mecanismos externos (bancos de dados, frameworks web, CLI, bibliotecas de terceiros) são detalhes de infraestrutura encapsulados por adapters.
- O core da aplicação desconhece protocolos externos ou fornecedores específicos de nuvem.

## 3. Modularidade e Desacoplamento

- Nenhum pacote ou módulo pode importar seu consumidor.
- Ciclos de dependência são estritamente proibidos e checados no pipeline de CI.
- Toda pasta do repositório deve ser autoexplicativa e conter seu respectivo ` + "`README.md`" + `.
`
			},
		},
		{
			relPath: "docs/architecture/overview.md",
			content: func(_, _ string) string {
				return `# Visão Geral de Arquitetura

## Topologia do Sistema

O projeto é estruturado em três planos concêntricos:

1. **Plano de Domínio**: Entidades essenciais, regras de negócio e validações puras.
2. **Plano de Aplicação**: Casos de uso, orquestração de operações e fluxos de tarefas.
3. **Plano de Adaptadores e Infraestrutura**: Drivers de persistência, conectores externos, interfaces de CLI e mensageria.

## Diagrama Conceitual

` + "```mermaid" + `
graph TD
    UI[Interfaces / CLI / Web] --> App[Plano de Aplicação]
    Infra[Bancos de Dados / Serviços] --> App
    App --> Domain[Plano de Domínio e Contratos]
` + "```" + `
`
			},
		},
		{
			relPath: "docs/architecture/adr/README.md",
			content: func(_, _ string) string {
				return `# Architectural Decision Records (` + "`docs/architecture/adr/`" + `)

## O que é este diretório?
Contém os registros formais e imutáveis de decisões arquiteturais significativas tomadas ao longo do projeto.

## Para que serve?
Preserva o contexto histórico, as opções avaliadas, as consequências aceitas e os critérios que justificaram cada decisão estrutural.

## Inventário
- ` + "`001-architecture-baseline.md`" + `: Decisão que estabelece os limites arquiteturais iniciais e o modelo de portas e adaptadores.
`
			},
		},
		{
			relPath: "docs/architecture/adr/001-architecture-baseline.md",
			content: func(_, _ string) string {
				return `# ADR 001: Linha de Base Arquitetural e Princípios de Clean Architecture

## Status
Aceito

## Contexto
O projeto requer alta manutenibilidade, isolamento rigoroso de regras de negócio em relação a frameworks e dependências externas, e suporte a testes unitários e de integração sem dependências de infraestrutura pesada.

## Decisão
Adotamos a Clean Architecture (Portas e Adaptadores). Todas as dependências devem apontar para o domínio central. Comunicações com infraestrutura externa devem ocorrer exclusivamente por meio de interfaces de portas.

## Consequências
- **Positivas**: Testabilidade completa em memória; facilidade de substituição de drivers de persistência; independência de frameworks.
- **Negativas**: Introdução de camadas intermediárias de mapeamento de dados (DTOs e entidades).
`
			},
		},
		{
			relPath: "docs/product/README.md",
			content: func(_, _ string) string {
				return `# Especificações de Produto (` + "`docs/product/`" + `)

## O que é este diretório?
Centraliza a visão estratégica do produto, os problemas resolvidos, os perfis de usuários atendidos e as delimitações de escopo.

## Para que serve?
Evita o desperdício de engenharia garantindo que desenvolvedores e agentes compreendam o *porquê* de cada funcionalidade antes da implementação.

## Inventário
- ` + "`vision.md`" + `: Visão do produto, proposta de valor e personas.
- ` + "`scope.md`" + `: O que está dentro e fora do escopo da versão atual.
`
			},
		},
		{
			relPath: "docs/product/vision.md",
			content: func(_, _ string) string {
				return `# Visão de Produto

## Proposta de Valor
Fornecer um sistema altamente confiável, modular e auto-documentado, projetado para colaboração fluida e segura entre humanos e agentes autônomos.

## Personas
- **Engenheiro de Software**: Procura código modular, limpo, com testes determinísticos e documentação viva.
- **Líder Técnico**: Requer conformidade com padrões de arquitetura, segurança e rastreabilidade de decisões.
- **Agente de IA**: Necessita de contexto enxuto, instruções explícitas de escopo e feedback automatizado de testes.
`
			},
		},
		{
			relPath: "docs/product/scope.md",
			content: func(_, _ string) string {
				return `# Escopo do Projeto

## Em Escopo (In-Scope)
- Implementação modular de regras de negócio fundamentais.
- Estrutura completa de testes automatizados com cobertura mínima de 85%.
- Verificação automatizada de segurança (SAST, secrets scanning e dependências).
- Documentação exaustiva com ` + "`README.md`" + ` explicativo em todas as pastas.

## Fora de Escopo (Out-of-Scope)
- Criação de abstrações genéricas sem caso de uso concreto.
- Implementação de funcionalidades sem Goal formalmente aceito.
- Modificações silenciosas de contratos ou decisões arquiteturais.
`
			},
		},
		{
			relPath: "docs/development/README.md",
			content: func(_, _ string) string {
				return `# Guia de Desenvolvimento (` + "`docs/development/`" + `)

## O que é este diretório?
Contém os padrões de codificação, guias de onboarding e a estratégia exaustiva de testes automatizados do projeto.

## Para que serve?
Assegura consistência estilística, disciplina de implementação orientada a testes e altos padrões de qualidade entre todos os contribuidores.

## Inventário
- ` + "`coding-standards.md`" + `: Padrões e boas práticas de código.
- ` + "`testing-strategy.md`" + `: O ciclo exaustivo de testes (unitários, integração, segurança, performance, stress e UI).
`
			},
		},
		{
			relPath: "docs/development/coding-standards.md",
			content: func(_, _ string) string {
				return `# Padrões de Codificação e Engenharia

1. **Formatação e Estilo**:
   - Todo código deve passar por formatadores oficiais da linguagem (` + "`gofmt`" + `, ` + "`prettier`" + `, ` + "`black`" + `, ` + "`rustfmt`" + `) sem exceção.
   - Linhas mantidas em até 100 caracteres quando razoável.

2. **Tipagem e Erros**:
   - Tipagem estrita em todas as assinaturas públicas.
   - Retornos de erro devem ser explícitos e incluir contexto da operação.

3. **Documentação no Código**:
   - Comentários explicam o *porquê*, nunca o *o quê*.
   - Todas as funções e interfaces públicas devem conter docstrings/comentários descritivos.
`
			},
		},
		{
			relPath: "docs/development/testing-strategy.md",
			content: func(_, _ string) string {
				return `# Estratégia de Testes Exaustivos e Ciclo de Qualidade

Este documento codifica o **ciclo mandatório e rigoroso de testes e qualidade** aplicado a todas as implementações deste projeto. Nenhum código é integrado sem evidências automatizadas de aprovação em todos os níveis da pirâmide.

---

## 1. O Loop Rigoroso de Implementação (TDD / BDD)

Para cada nova funcionalidade ou correção de defeito:

` + "```mermaid" + `
flowchart LR
    A[1. Definir Contrato / Goal] --> B[2. Escrever Teste com Falha]
    B --> C[3. Implementação Mínima]
    C --> D[4. Validação & Refatoração]
    D --> E[5. Bateria de Segurança & Estresse]
    E --> F[6. Evidência & Atualização de Docs/Changelog]
` + "```" + `

1. **Definição de Contratos**: Especificar a interface, tipos e critérios de aceitação.
2. **Teste Inicial**: Criar teste que reproduza a falha ou verifique o comportamento esperado antes de codificar a solução.
3. **Implementação**: Escrever código limpo, modular e desacoplado que satisfaça o teste.
4. **Refatoração**: Aplicar princípios de Clean Code sem quebrar nenhum teste existente.
5. **Varredura Completa**: Executar suíte de conformidade, segurança, performance e estresse.
6. **Evidência e Changelog**: Atualizar documentação afetada e registrar a mudança no ` + "`CHANGELOG.md`" + `.

---

## 2. As Camadas da Pirâmide de Testes

### A. Funcionalidade
- **Testes Unitários**: Determinísticos, rápidos (milissegundos), sem chamadas de rede ou I/O real. Cobertura mínima exigida: **85%**.
- **Testes de Integração**: Testam interações reais entre módulos, adaptadores de banco de dados e sistemas de arquivos com fixtures isoladas.
- **Testes de Contrato**: Validação de comunicação entre serviços e interfaces públicas.

### B. Conformidade e Validação Estática
- **Linters**: Verificação estrita de sintaxe, tipos e convenções.
- **Schemas**: Validação de todas as estruturas de entrada/saída contra esquemas JSON Draft 2020-12.
- **Formatação**: Zero divergência em relação ao padrão canônico.

### C. Segurança (DevSecOps)
- **Varredura de Segredos**: Proibição de chaves, senhas ou tokens no código (verificação automatizada pré-commit).
- **SAST (Static Application Security Testing)**: Análise estática contra injeção de comandos, XSS, SSRF e vulnerabilidades OWASP Top 10.
- **Auditoria de Dependências**: Detecção de CVEs e bibliotecas vulneráveis ou obsoletas na cadeia de suprimentos.
- **Princípio do Menor Privilégio**: Testes de permissões restritas em tempo de execução.

### D. Performance, Carga e Estresse
- **Benchmarks**: Medição contínua de latência e consumo de memória por operação.
- **Testes de Concorrência & Deadlock**: Detecção de condições de corrida (` + "`-race`" + `), bloqueios mútuos e vazamentos de recursos/goroutines.
- **Testes de Estresse & Carga**: Submissão a picos de tráfego e limites extremos para verificar comportamento de degradação graciosa.

### E. Frontend, UI e Experiência do Usuário
- **Testes de Componentes**: Renderização isolada e testes de interação com frameworks de teste modernos.
- **Testes End-to-End (E2E)**: Simulação de jornadas reais de usuário via navegadores headless (Playwright / Cypress).
- **Regressão Visual**: Comparação de screenshots para evitar desvios visuais de layout.
- **Acessibilidade**: Varredura automatizada contra as diretrizes WCAG 2.1 AA (axe-core).

---

## 3. Comandos de Execução Recomendados

| Categoria | Exemplo de Ferramenta | Comando Padrão |
|-----------|------------------------|----------------|
| Unitário & Corrida | Go test / Vitest / Pytest | ` + "`go test -v -race ./...`" + ` |
| Formatação | Official formatter | ` + "`gofmt -d .`" + ` ou ` + "`prettier --check .`" + ` |
| Linter | Staticcheck / ESLint / Ruff | ` + "`staticcheck ./...`" + ` |
| Segurança | Secret scanner / Trivy | ` + "`prumo tool scan-secrets .`" + ` |
| UI & E2E | Playwright | ` + "`npx playwright test`" + ` |
| Carga | k6 / Vegeta | ` + "`k6 run load-test.js`" + ` |
`
			},
		},
		{
			relPath: "docs/operations/README.md",
			content: func(_, _ string) string {
				return `# Operações e Infraestrutura (` + "`docs/operations/`" + `)

## O que é este diretório?
Contém runbooks, instruções de implantação, observabilidade e planos de recuperação de desastres.

## Para que serve?
Garante que a operação do software em produção seja previsível, auditável e resiliente a falhas.

## Inventário
- ` + "`deployment.md`" + `: Procedimentos de build, empacotamento e entrega contínua.
- ` + "`observability.md`" + `: Métricas, logs estruturados e rastreamento distribuído.
`
			},
		},
		{
			relPath: "docs/operations/deployment.md",
			content: func(_, _ string) string {
				return `# Guia de Implantação (Deployment)

## Requisitos de Release
1. Suíte completa de testes passando 100% com detector de condições de corrida.
2. Análise de segurança SAST e de dependências limpas.
3. Versão atualizada no manifesto e notas adicionadas no ` + "`CHANGELOG.md`" + `.

## Procedimento
- Compilação do binário ou container autocontido com verificação de checksums.
- Execução de smoke test em ambiente de staging antes da promoção para produção.
`
			},
		},
		{
			relPath: "docs/operations/observability.md",
			content: func(_, _ string) string {
				return `# Observabilidade e Telemetria

## Pilares
1. **Logs Estruturados**: Logs em formato JSON com timestamp UTC, nível de severidade e ID de correlação (` + "`trace_id`" + `).
2. **Métricas**: Contadores, medidores e histogramas de latência por endpoint ou operação.
3. **Traces**: Rastreamento de chamadas distribuídas com propagação de contexto.
`
			},
		},
		{
			relPath: "docs/security/README.md",
			content: func(_, _ string) string {
				return `# Segurança e Confiança (` + "`docs/security/`" + `)

## O que é este diretório?
Contém o modelo de confiança, análise de riscos e o contrato pesado de segurança do projeto.

## Para que serve?
Garante que a segurança não seja um pensamento tardio, estabelecendo controles rígidos contra vazamento de credenciais, abuso de privilégios e ataques à cadeia de suprimentos.

## Inventário
- ` + "`threat-model.md`" + `: Modelagem de ameaças (metodologia STRIDE).
- ` + "`security-contract.md`" + `: O contrato estrito de segurança de código e execução.
`
			},
		},
		{
			relPath: "docs/security/threat-model.md",
			content: func(_, _ string) string {
				return `# Modelo de Ameaças (STRIDE)

| Ameaça | Vetor Potencial | Mitigação Mandatória |
|--------|-----------------|----------------------|
| **Spoofing** | Falsificação de chamadas ou payloads | Autenticação mútua e validação estrita de assinaturas. |
| **Tampering** | Modificação maliciosa de metas ou código | Verificação de integridade via digest criptográfico SHA-256 e regras de branch protegida. |
| **Repudiation** | Ações executadas sem rastreabilidade | Registro imutável de eventos e evidências vinculadas a Goals. |
| **Information Disclosure** | Exposição acidental de credenciais ou dados | Varredura automatizada pré-commit e redação de segredos em logs. |
| **Denial of Service** | Consumo excessivo de recursos ou loops | Limites orçamentários de contexto (LPC), timeouts explícitos e ausência de recursão irrestrita. |
| **Elevation of Privilege** | Escape de isolamento de subagentes | Conectores com políticas de execução estritas e isolamento de permissões. |
`
			},
		},
		{
			relPath: "docs/security/security-contract.md",
			content: func(_, _ string) string {
				return `# Contrato Pesado e Firme de Segurança

Este documento estabelece as **obrigações invioláveis de segurança** que devem ser satisfeitas por todo contribuinte (humano ou agente de IA) antes de qualquer merge:

1. **Tolerância Zero para Segredos**:
   - Nenhuma chave privada, token de API, segredo de nuvem ou credencial pode ser commitada, mesmo que temporária.
   - Qualquer ocorrência aciona reprovação imediata do commit e rotação compulsória da credencial.

2. **Isolamento e Menor Privilégio**:
   - Subagentes operam exclusivamente com as permissões e ferramentas declaradas em seus contratos de harness.
   - Chamadas de subprocessos devem escapar todos os parâmetros e rejeitar interpolação de shell desprotegida.

3. **Validação Estrita de Limites (Boundaries)**:
   - Todo dado recebido de fontes externas (HTTP, arquivos, variáveis de ambiente, ferramentas MCP) é tratado como não-confiável.
   - Sanitização e validação de schema ocorrem na borda de entrada antes de atingir as camadas de domínio.

4. **Auditoria de Cadeia de Suprimentos**:
   - Dependências externas devem ser auditadas contra vulnerabilidades conhecidas (CVEs).
   - Bloqueio imediato de bibliotecas com avisos críticos ou licenças incompatíveis.
`
			},
		},
		{
			relPath: "docs/governance/README.md",
			content: func(_, _ string) string {
				return `# Governança do Repositório (` + "`docs/governance/`" + `)

## O que é este diretório?
Documenta as políticas operacionais de versionamento, fluxo de trabalho no Git, revisões de código e regras de merge.

## Para que serve?
Mantém histórico linear, rastreabilidade auditável e proteção de branches principais contra modificações não autorizadas.

## Inventário
- ` + "`repository-governance.md`" + `: Regras de branches, convenções de commits e processos de Pull Request.
`
			},
		},
		{
			relPath: "docs/governance/repository-governance.md",
			content: func(_, _ string) string {
				return `# Governança de Repositório

1. **Branches**:
   - ` + "`main`" + `: Branch protegida e imutável diretamente. Pushes diretos são proibidos.
   - Padrão de branches de trabalho: ` + "`feat/*`" + `, ` + "`fix/*`" + `, ` + "`chore/*`" + `, ` + "`docs/*`" + `, ` + "`refactor/*`" + `.

2. **Commits**:
   - Padrão Conventional Commits obrigatório: ` + "`tipo(escopo): descrição imperativa`" + `.

3. **Pull Requests & Merge**:
   - Todo código entra em ` + "`main`" + ` via PR.
   - Estratégia de merge: ` + "`squash`" + ` com branch de trabalho deletada após o merge.
   - Quality gates do CI (` + "`gofmt`" + `, ` + "`govet`" + `, ` + "`gotest -race`" + `) devem passar 100%.
`
			},
		},
		{
			relPath: "docs/contracts/README.md",
			content: func(_, _ string) string {
				return `# Contratos de Documentação (` + "`docs/contracts/`" + `)

## O que é este diretório?
Armazena a definição formal dos contratos de documentação e seus vínculos (bindings) com fontes canônicas.

## Arquivos
- ` + "`builtin.json`" + `: Contratos e obrigações canônicas de documentação.
- ` + "`bindings.json`" + `: Vínculo entre cada contrato e seus documentos fontes mantidos pelo projeto.
`
			},
		},
		{
			relPath: "docs/profiles/README.md",
			content: func(_, _ string) string {
				return `# Perfis de Documentação (` + "`docs/profiles/`" + `)

## O que é este diretório?
Declara os perfis que agrupam contratos aplicáveis de documentação conforme o tipo e capacidades do projeto.

## Arquivos
- ` + "`builtin.json`" + `: Perfis de documentação suportados e os contratos que compõem cada um.
`
			},
		},
	}
}

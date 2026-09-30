---
layout: home

hero:
  name: Prumo
  text: Harness de Engenharia & Protocolo Nativo Git
  tagline: Plano de controle executável para desenvolvimento de software com humanos e agentes autônomos. 189 skills de precisão, 39 agentes especialistas, integridade SHA-256 e Lean Progressive Context (LPC) sem poluição de contexto.
  image:
    src: /assets/logo.svg
    alt: Prumo Framework Logo
  actions:
    - theme: brand
      text: Começar Agora (5 min)
      link: /getting-started/first-project
    - theme: alt
      text: Workforce Catalog
      link: /workforce/
    - theme: alt
      text: Harness & ACI
      link: /harness/
    - theme: alt
      text: Ferramentas (CLI)
      link: /tools/

features:
  - icon: 🎯
    title: Clareza Cognitiva & Metodologia LPC
    details: Lean Progressive Context (LPC) — forneça aos modelos apenas o contexto estritamente necessário para cada tarefa. Expansão progressiva, cápsulas de trabalho imutáveis e zero fadiga por megaprompts.
    link: /governance/lpc
  - icon: 🛠️
    title: Harness Executável & ACI Completo
    details: Tool Gateway integrado com sandbox determinístico, AST parsers, isolamento de processos, controle de orçamentos e execução segura de comandos de terminal com bounds rígidos.
    link: /harness/
  - icon: 🛡️
    title: Governança Estrita & Integrity Locks
    details: Metas (Goals) e Planos assinados com digests SHA-256 invioláveis. Emendas formais rastreadas em Git que impedem qualquer mutação silenciosa por agentes em execução.
    link: /getting-started/concepts
  - icon: 🤖
    title: Workforce Canônica Completa
    details: 189 skills catalogadas, 39 agentes especialistas (sistemas, arquitetura, design visual, UI/UX, SVG/motion, publicidade, segurança) e 20 receitas determinísticas na caixa.
    link: /workforce/
  - icon: 🔌
    title: Conectores Universais Sem Lock-in
    details: Adapters cirúrgicos para Google Antigravity, Claude Code, OpenAI Codex, Cursor, Windsurf, OpenCode, Cline e Roo Code sem ceder autoridade do projeto a nenhum harness.
    link: /manual/connectors
  - icon: 📜
    title: Memória Perene & Livro Vivo
    details: Rastreabilidade ponta a ponta (Requisitos ↔ ADRs ↔ Código ↔ Testes ↔ Evidências), histórico episódico, checkpoints de sessão e sincronização contínua com o Living Book.
    link: /governance/
---

<div class="vp-doc">

## Engenharia Determinística Orientada a Evidências

O **Prumo** transforma repositórios de código em ambientes cooperativos estruturados para desenvolvedores e agentes autônomos. Em vez de prompts caóticos e conversas efêmeras que perdem contexto entre sessões, o Prumo estabelece um **plano de controle nativo em Git** onde metas possuem integridade criptográfica, cada ação gera evidências verificáveis e a verdade canônica do projeto é inegociável.

```bash
# 1. Inicializar repositório com perfil canônico e modelo preferido
prumo init ./meu-projeto --profile ./profiles/web-service.json

# 2. Criar uma Meta (Goal) com objetivo mensurável e fase formal
prumo goal new P01-G01 "Autenticação JWT" \
  --phase P01 \
  --objective "Implementar autenticação JWT sem dependências de terceiros"

# 3. Travar a Meta com digest SHA-256 (garantia contra mutações arbitrárias)
prumo goal state P01-G01 LOCKED

# 4. Executar tarefas em sandbox determinístico com agentes especialistas
prumo run --goal P01-G01 --agent systems-core

# 5. Auditar conformidade de documentação, autoridade e drift
prumo docs audit
```

---

## Todas as Ferramentas em um Único Binário Puro Go

O Prumo v0.6 é distribuído como um binário compilado único em Go, sem dependências de interpretadores externos, pip ou virtualenvs. Todas as capacidades do harness estão disponíveis imediatamente:

<div class="tool-grid">

<div class="tool-card">
  <div>
    <h3>▶️ Executar (Run Engine)</h3>
    <p>Execução de diretivas, tarefas e planos com sandbox hermético, tool gateway e captura determinística de logs.</p>
  </div>
  <div class="tool-cmd">prumo run --plan ./plan.json</div>
</div>

<div class="tool-card">
  <div>
    <h3>🎯 Planejar (Goals & DAGs)</h3>
    <p>Criação e transição de Metas, resolução de dependências em grafo direcionado acíclico e trava com hash SHA-256.</p>
  </div>
  <div class="tool-cmd">prumo goal new P01-G02 "Cache"</div>
</div>

<div class="tool-card">
  <div>
    <h3>🔌 Compilar Adaptadores</h3>
    <p>Projeção cirúrgica de contexto e instruções para Google Antigravity, Claude Code, Codex, Cursor e Windsurf.</p>
  </div>
  <div class="tool-cmd">prumo compile --target claude-code</div>
</div>

<div class="tool-card">
  <div>
    <h3>🤖 Agente TUI Interativo</h3>
    <p>Terminal interativo com prumo-agent (pa) para conduzir sessões assistidas por IA com preservação de memória.</p>
  </div>
  <div class="tool-cmd">prumo agent</div>
</div>

<div class="tool-card">
  <div>
    <h3>🔍 Auditoria de Documentação</h3>
    <p>Verificação estrita da hierarquia de autoridade, detecção de drift entre releases e conformidade de contratos.</p>
  </div>
  <div class="tool-cmd">prumo docs audit</div>
</div>

<div class="tool-card">
  <div>
    <h3>🩺 Diagnóstico do Repositório</h3>
    <p>Inspeção profunda de integridade do projeto, consistência de schemas e prontidão para execução autônoma.</p>
  </div>
  <div class="tool-cmd">prumo doctor</div>
</div>

</div>

---

## Primeiros Passos

<div class="tip custom-block">
  <p class="custom-block-title">💡 Como começar em menos de 5 minutos</p>
  <ol>
    <li><strong>Instale o binário oficial</strong>: Baixe via release do GitHub ou use nosso instalador de linha única para Linux, macOS e Windows.</li>
    <li><strong>Inicialize seu projeto</strong>: Execute <code>prumo init</code> para gerar os contratos canônicos e a estrutura <code>prumo.json</code>.</li>
    <li><strong>Explore o Catálogo Workforce</strong>: Descubra as <strong>189 skills</strong> e os <strong>39 agentes especialistas</strong> disponíveis nativamente no <a href="/workforce/">Catálogo Workforce</a>.</li>
    <li><strong>Consulte o Manual</strong>: Aprofunde-se no <a href="/manual/usage">Guia de Uso</a> e nas <a href="/architecture/">Especificações de Arquitetura</a>.</li>
  </ol>
</div>

---

## Engenharia Aberta, Rigorosa e Transparente

O desenvolvimento do Prumo Framework é 100% público e orientado pela verdade canônica do repositório:

- **[Macroarquitetura do Sistema](/architecture/)**: Visão detalhada do plano de controle, separação de camadas e barramentos de eventos.
- **[Decisões Arquiteturais (ADRs)](/decisions/)**: Registro formal e imutável de todas as decisões técnicas tomadas pela engenharia.
- **[Governança & Autoridade](/governance/authority)**: Como o Prumo resolve ambiguidades de conhecimento sem alucinações.
- **[Native Workspace Viewer (Rust)](/architecture/viewer)**: Interface de alta performance para visualização de repositórios complexos.

</div>

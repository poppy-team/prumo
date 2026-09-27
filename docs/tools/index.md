# Ferramentas do Desenvolvedor (CLI)

O ecossistema Prumo concentra todas as ferramentas necessárias para desenvolvimento, governança, diagnóstico e execução autônoma em um único binário executável (`prumo`).

## Ferramentas Disponíveis

<div class="tool-grid">

<div class="tool-card">
  <div>
    <h3>▶️ Executar</h3>
    <p>Execução de diretivas, tarefas de plano e pipelines autônomos com limites rígidos.</p>
  </div>
  <div class="tool-cmd">prumo run --path .</div>
</div>

<div class="tool-card">
  <div>
    <h3>🎯 Metas & Planos</h3>
    <p>Gestão formal de ciclo de vida de Metas, emendas auditadas e travas criptográficas.</p>
  </div>
  <div class="tool-cmd">prumo goal list</div>
</div>

<div class="tool-card">
  <div>
    <h3>🔌 Compilar</h3>
    <p>Geração de adaptadores para Claude Code, Antigravity, Cursor, Codex e Windsurf.</p>
  </div>
  <div class="tool-cmd">prumo compile --target all</div>
</div>

<div class="tool-card">
  <div>
    <h3>🩺 Diagnóstico (Doctor)</h3>
    <p>Verificação estrita de consistência de schemas, dependências e saúde do projeto.</p>
  </div>
  <div class="tool-cmd">prumo doctor</div>
</div>

<div class="tool-card">
  <div>
    <h3>📚 Auditoria de Documentação</h3>
    <p>Inspeção de autoridade, detecção de links quebrados e prevenção de drifts.</p>
  </div>
  <div class="tool-cmd">prumo docs audit</div>
</div>

<div class="tool-card">
  <div>
    <h3>🤖 Agente TUI</h3>
    <p>Interface de terminal interativa com o cliente prumo-agent (pa).</p>
  </div>
  <div class="tool-cmd">prumo agent</div>
</div>

</div>

## Páginas Detalhadas

- **[Referência Completa de Comandos](/tools/cli-reference)**: Todas as flags, argumentos, subcomandos e formatos de envelope JSON.
- **[Diagnóstico & Doctor](/tools/doctor)**: Como interpretar e solucionar avisos do comando `prumo doctor`.
- **[Conectores & Adapters](/manual/connectors)**: Detalhes de compilação para plataformas de agentes.

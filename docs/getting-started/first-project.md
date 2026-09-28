# Primeiro Projeto com o Prumo (5 minutos)

Inicializar um projeto no Prumo é instantâneo e não exige escrever arquivos de configuração manuais.

---

## 1. Inicialização Zero-Configuração (Greenfield)

Para criar um novo projeto governado pelo Prumo, basta criar o diretório e rodar `prumo init`:

```bash
mkdir my-app && cd my-app
git init
prumo init
```

O Prumo detectará o diretório e criará a estrutura canônica do **Protocolo v3**:

```text
my-app/
├── prumo.json                            # Configuração canônica do projeto (v3)
├── PROJECT_STATE.md                      # Estado atual, fase e metadados
├── ENTRYPOINT.md                         # Guia de onboarding para agentes e humanos
├── docs/
│   └── PRUMO.md                          # Roteador de documentação e intenções
├── .ai/
│   ├── agents/manifest.json              # Workforce de agentes resolvidos
│   ├── skills/manifest.json              # Skills necessárias
│   ├── recipes/manifest.json             # Receitas de engenharia
│   └── orchestration/                    # Políticas de fallback e scorecards
└── .prumo/                               # Runtime local, histórico de inteligência e cache
```

---

## 2. Inicialização com Presets e Flags

Se desejar personalizar a stack ou o foco do projeto imediatamente:

```bash
# Preset para aplicações Web & Frontend
prumo init --preset web --name portal-cliente

# Preset para Microsserviços e APIs de Alta Performance
prumo init --preset service --stack go

# Preset para Ferramentas de Linha de Comando (CLI)
prumo init --preset cli --name gerador-relatorios

# Inspecionar o perfil gerado antes de aplicar
prumo init --print-profile
```

### Presets Disponíveis:
- **`standard`** (padrão): Engenharia de software completa com balanceamento de qualidade e cobertura.
- **`cli`**: Foco em experiência de terminal, interfaces TUI e flags de linha de comando.
- **`web`**: Foco em design responsivo, componentes visuais e acessibilidade WCAG.
- **`service`**: APIs resilientes, microservices e segurança reforçada.
- **`library`**: Pacotes reutilizáveis e código com zero dependências externas.
- **`minimal`**: Configuração leve com consumo mínimo de tokens para prototipagem rápida.

---

## 3. Verificar Conformidade

Após inicializar, valide seu projeto com o comando de auditoria:

```bash
prumo validate
```

O comando confirmará que todos os esquemas JSON Schema, arquivos canônicos e contratos do Protocolo v3 estão 100% em conformidade.

---

## 4. Compilar Adaptadores para seus Agentes de IA

Para que o Claude Code, Google Antigravity, OpenCode ou Cursor entendam as diretrizes do seu projeto:

```bash
# Compilar adaptadores para todos os ambientes suportados
prumo compile --all

# Ou compilar para um ambiente específico
prumo compile --target claude-code
prumo compile --target antigravity
```

---

## 5. Iniciar o Prumo Code Agent (TUI Interativo)

O **Prumo Code Agent** é o cliente interativo de terminal projetado para desenvolvimento assistido por IA em máxima harmonia com o desenvolvedor:

```bash
prumo code-agent
```

*(Você também pode utilizar os atalhos `prumo agent` ou `prumo tui`)*.

### Recursos do Prumo Code Agent

1. **Sessão com Nome Automático por Branch**:
   - Cada nova sessão é automaticamente inicializada com o nome da **branch Git ativa** (ex: `feature/auth-jwt`).
   - Caso o repositório não tenha branch ou esteja em estado *detached HEAD*, o nome é gerado deterministicamente no formato `YYYY-MM-DD-<nome-do-projeto>`.

2. **Multi-Tabs (Conversas Simultâneas)**:
   - Execute múltiplos fluxos de trabalho ou investigações em paralelo no mesmo repositório:
     - `Ctrl+T`: Cria uma nova aba de conversação.
     - `Ctrl+W`: Fecha a aba ativa.
     - `Alt+1` até `Alt+9`: Alterna diretamente para a aba desejada.

3. **Barra Lateral com Telemetria Profunda do Harness**:
   - Inspirada na ergonomia do OpenCode v2, mas alimentada com dados profundos que apenas o harness do Prumo fornece:
     - **Provider & Modelo**: Sempre visíveis no topo da barra lateral e na status line.
     - **Projeto & Branch**: Caminho absoluto e branch atual rastreados em tempo real.
     - **Consumo de Tokens**: Contagem detalhada de tokens de entrada (prompt), saída (completion) e leitura/escrita de cache.
     - **Custo Acumulado (USD)**: Estimativa precisa de custo da sessão e do projeto.
     - **Arquivos Modificados**: Lista de arquivos criados ou editados na sessão.
     - **Workforce & Subagentes**: Árvore hierárquica de subagentes invocados em execução.
     - **Verification Gates**: Status de checagens de linters, testes e integridade de metas.
     - **Visibilidade Persistente**: Pressione `Ctrl+B` para alternar a exibição da barra lateral (o estado é salvo no arquivo de configuração do usuário).

4. **Visualização de Temas em Tempo Real**:
   - Abra a seleção de temas pelo comando `/theme` ou pelo menu de comandos (`Ctrl+K` → *Theme*).
   - Ao navegar com as setas `↑`/`↓` ou `j`/`k`, o tema é **aplicado instantaneamente na tela** para pré-visualização, acompanhado de swatches de cor (`[■ ■ ■]`).
   - Pressione `Enter` para confirmar e salvar nas preferências ou `Esc` para restaurar o tema anterior sem alterar nada no disco.

5. **Finalização com Salvamento Determinístico e Auditoria**:
   - Ao pressionar `Ctrl+Q`, um modal intuitivo permite escolher entre **Save & Quit**, **Quit (Sem Salvar)** ou **Cancel**.
   - Você pode renomear a sessão antes de salvá-la.
   - Ao salvar, a telemetria completa da sessão é gravada deterministicamente em `.prumo/runtime/audit/telemetry.json` e `.prumo/runtime/audit/sessions/<session-id>.json`.

---

## 6. Prumo IDE (Interface Desktop Gráfica)

Para inspecionar o repositório, grafos de tarefas e planos através de uma interface visual desktop de alta performance:

```bash
prumo native
```

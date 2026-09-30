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

## 5. Iniciar o Coding Agent Interativo (TUI)

Prumo inclui um agente terminal com interface rica integrado:

```bash
prumo agent
```

Você pode conversar com o agente, solicitar implementações e acompanhar as decisões arquiteturais com total rastreabilidade.

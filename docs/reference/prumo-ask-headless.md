# 80 — Prumo Ask: Headless One-Shot Interface e Machine-Friendly CLI

> Authority: canonical specification.
> Logical ID: CONST-80
> Source: Notion Living Book (3de9bb7d023f81f1a5c2e6d44111461b)
> Status: Especificação do Prumo Ask Headless One-Shot Interface.


<aside>
💬

**Status:** proposta consolidada para uma interface headless de pergunta/resposta rápida no Prumo. A implementação deve reutilizar o core já existente e permanecer desacoplada da TUI.

</aside>

# Objetivo

`prumo ask` será a superfície **one-shot, headless e machine-friendly** do Prumo para perguntas, análise rápida, pipelines de shell e automações. O comando deve responder diretamente no terminal sem iniciar a TUI e sem transformar uma pergunta simples em uma execução agentic completa.

A intenção é tornar o Prumo útil em qualquer terminal, shell, SSH, CI, script ou pipeline, mantendo o mesmo core de providers, contexto, políticas e observabilidade.

# Princípio arquitetural

**Headless é uma superfície de primeira classe, não uma TUI escondida.**

`prumo ask` deve chamar diretamente os serviços do Prumo Core. A TUI, `ask`, `agent run` e futuros clientes compartilham os mesmos boundaries, mas nenhuma interface depende da outra.

```
               PRUMO CORE
                  │
    ┌─────────────┼─────────────┐
    │             │             │
    ▼             ▼             ▼
Prumo TUI     prumo ask    prumo agent
    │             │             │
humano        one-shot       agentic
    │             │             │
    └─────────────┼─────────────┘
                  │
                  ▼
             prumo serve
           daemon / API local
```

# Relação com o runtime atual

O Prumo já possui uma base headless no comando `prumo agent` e um boundary canônico de `ModelProvider`, incluindo streaming e adapters OpenAI-compatible e Anthropic. Portanto, `prumo ask` deve ser implementado como uma camada fina de UX e contrato de saída sobre o core existente, evitando duplicação de runtime.

# Semântica principal

| Superfície | Uso primário | Ferramentas | Mutação | Persistência |
| --- | --- | --- | --- | --- |
| `prumo ask` | Pergunta, explicação, análise e transformação rápida | Desabilitadas por padrão | Read-only por padrão | Efêmera por padrão; sessão opcional |
| `prumo agent run` | Trabalho agentic completo | Permitidas conforme policy | Pode editar/executar | Run, checkpoints, evidence e state |
| `prumo` | Experiência humana interativa | Conforme modo/policy | Conforme modo/policy | Sessão interativa |

# Exemplos de uso

```bash
prumo ask "explique esse erro"

cargo test 2>&1 | prumo ask "explique por que os testes falharam"

prumo ask --context repo "onde está implementada a autenticação?"

prumo ask --model openrouter/deepseek-v4.1-flash \
  "explique este código"

prumo ask --file internal/harness/model/model.go \
  "analise esta implementação"

prumo ask --raw "gere somente um nome para esta função"

prumo ask --json "analise este erro"

prumo ask --session backend \
  "agora considere também o banco de dados"
```

# Contrato de entrada

O comando deve aceitar, de forma composável:

- prompt posicional;
- `stdin` via pipe;
- um ou mais arquivos explicitamente informados;
- contexto compilado do repositório;
- contexto de sessão opcional;
- system instruction explícita;
- provider/model selecionado por flag ou configuração.

Quando prompt posicional e `stdin` coexistirem, o conteúdo do pipe deve ser tratado como material/contexto e o argumento como instrução principal, salvo opção explícita em contrário.

# Contrato de saída

A separação de canais é obrigatória.

**stdout** deve conter somente a resposta solicitada ou o stream estruturado escolhido.

**stderr** deve receber diagnósticos, modelo selecionado, warnings, retries, status, métricas, logs e mensagens operacionais.

Isso garante composição correta:

```bash
RESULT=$(prumo ask --raw "retorne somente um número")

git diff | prumo ask --raw "resuma" | glow
```

# Formatos de saída

Superfície inicial recomendada:

- `--raw` — apenas texto final, sem decoração;
- `--json` — envelope estruturado único;
- `--jsonl` — stream de eventos/mensagens em NDJSON;
- `--quiet` — reduz diagnósticos não essenciais em stderr;
- saída Markdown/texto humana como default quando conectado a TTY.

O formato estruturado deve permanecer estável e versionável para CI e integrações.

# Flags propostas

```
--model <id>
--provider <provider>
--context <none|repo|auto|level>
--file <path>                repetível
--session <name-or-id>
--continue
--system <text-or-file>
--raw
--json
--jsonl
--quiet
--no-tools                   default
--tools <profile-or-list>
--timeout <duration>
```

Aliases curtos podem ser adicionados depois que o contrato principal estabilizar.

# Segurança e permissões

`prumo ask` deve nascer **read-only e sem tools por padrão**.

Uma pergunta rápida não pode inesperadamente modificar o workspace, executar comandos ou produzir efeitos externos. Capacidades adicionais exigem opt-in explícito, por exemplo:

```bash
prumo ask --tools read "analise a estrutura deste projeto"
```

A policy engine continua sendo a autoridade final. Uma flag de tool nunca deve contornar deny rules, trust boundaries ou políticas superiores.

Operações mutáveis ou agentic complexas devem orientar o usuário para `prumo agent run` em vez de ampliar silenciosamente a autoridade de `ask`.

# Sessões

O comportamento padrão deve ser **stateless/efêmero** para preservar previsibilidade e facilitar scripting.

Continuidade é opt-in:

```bash
prumo ask --session api "explique o fluxo de autenticação"
prumo ask --session api "agora compare com o refresh token"
```

`--continue` pode retomar a sessão one-shot mais recente dentro do escopo atual.

Sessões devem reutilizar o modelo canônico de session/event log do Prumo em vez de criar um formato paralelo.

# Contexto

O comando deve reutilizar Context v2 e a filosofia Least Context.

Modos sugeridos:

- `--context none` — somente prompt/stdin/arquivos explícitos;
- `--context repo` — contexto do workspace compilado sob budget;
- `--context auto` — seleção progressiva baseada na intenção;
- níveis futuros podem mapear diretamente para os context levels já definidos pelo Prumo.

O comando deve conseguir explicar o que entrou no contexto quando solicitado em modo diagnóstico.

# Daemon e warm path

A primeira versão pode operar diretamente no processo do CLI.

Em evolução posterior, `prumo ask` pode detectar ou se conectar a `prumo agent serve`/`prumo serve` para reutilizar:

- conexões/provider clients;
- discovery de modelos;
- índices do workspace;
- caches seguros;
- sessões;
- context manifests.

Isso deve ser uma otimização transparente, nunca uma dependência obrigatória.

# Implementação sugerida

```
cmd/prumo/ask_commands.go
        │
        ▼
internal/ask/ApplicationService
        │
        ├── Context Compiler
        ├── Session Adapter
        ├── ModelProvider
        ├── Permission/Tool Policy
        └── Output Renderer
```

O serviço de aplicação deve ser utilizável também por testes, daemon e futuros clientes sem depender de `os.Stdout`, parsing de flags ou TUI.

# Pipeline esperado

```
argv / stdin / files
        │
        ▼
Input Normalization
        │
        ▼
Context Assembly
        │
        ▼
Provider Resolution
        │
        ▼
Model Stream
        │
        ▼
Output Contract
        ├── stdout
        └── stderr
```

# Conformance e testes obrigatórios

- prompt simples retorna resposta em stdout;
- pipe/stdin preserva bytes/texto esperado;
- stdout nunca recebe logs operacionais;
- stderr nunca contamina `--raw`;
- JSON e JSONL permanecem válidos durante streaming;
- Ctrl+C cancela provider e retorna exit code coerente;
- erro de provider produz erro estruturado e código não-zero;
- ausência de tools é garantida por default;
- `--tools` continua subordinado à permission policy;
- arquivos/contexto não escapam do workspace sem autorização;
- budget de contexto é respeitado;
- sessão opcional retoma corretamente;
- comportamento é idêntico com e sem TTY onde o contrato exige;
- execução via daemon e execução direta devem ser conformantes.

# Exit codes

O contrato deve diferenciar pelo menos:

```
0  sucesso
2  uso/argumentos inválidos
3  configuração/provider inválido
4  policy/permission denied
5  upstream/model failure
130 cancelado pelo usuário
```

Os valores definitivos devem ser harmonizados com a taxonomia global de exit codes da CLI do Prumo.

# Não objetivos

`prumo ask` não deve:

- virar um terminal emulator;
- duplicar a TUI;
- possuir runtime agentic paralelo;
- contornar policies para conveniência;
- editar arquivos por default;
- depender de Bubble Tea;
- criar um protocolo de sessão incompatível com o restante do Prumo.

# Roadmap incremental

1. **Ask v1:** prompt + stdin + provider/model + streaming + stdout/stderr + raw/json.
2. **Ask v1.1:** arquivo explícito, Context v2 e cancelamento completo.
3. **Ask v1.2:** sessão opcional e `--continue`.
4. **Ask v1.3:** profiles de ferramentas read-only e JSONL/event stream.
5. **Ask v1.4:** warm path via daemon e otimizações de startup/cache.

# Decisão de produto

O Prumo não deve criar ou forkear um terminal somente para fornecer IA no shell. `prumo ask` deve funcionar em qualquer terminal moderno, SSH, tmux, CI ou script. A especialização do Prumo permanece no harness, contexto, governança, qualidade, documentação e orquestração — não em PTY/renderização de terminal.
# O que está aberto, explicado

> Companheiro de `docs/harness/gap-register.md`. O registro diz **o que** falta e
> em que estado; este documento diz **o que cada coisa é**, por que ela importa e
> o que exatamente precisa acontecer para destravá-la. Onde o registro tem um
> código (`GAP-078`), este documento tem a frase que explica o código.

## Como ler, e o que é "prioridade"

Os itens abertos estão em uma de quatro situações, e a situação é mais útil que a
prioridade:

| Situação | O que significa | Quem destrava |
|---|---|---|
| ⛔ **Espera outro trabalho** | Existe uma fila: este item não começa antes de outro terminar | quem fizer o item da frente |
| ⬜ **Espera uma decisão** | Há dois caminhos bons e escolher é de quem decide o produto, não de quem escreve o código | você |
| 🟡 **Espera a última milha** | A base já existe e funciona; falta o pedaço final | eu, quando a decisão existir |
| 🧑 **Espera uma pessoa** | Não é código: alguém precisa sentar e usar | alguém com o equipamento certo |

E o tamanho, que é quanto trabalho *depois* de destravado: **P** cabe em uma
sessão, **M** leva alguns dias, **G** leva uma semana ou mais e mexe em contrato
gravado ou em várias camadas ao mesmo tempo.

---

## 1. Anexos: um arquivo é *citado*, não enviado ✅ **fechado** (GAP-090)

**O que é.** Perguntar sobre uma imagem ou um print: "o que está errado nesta
tela?" com a imagem à mão.

**A decisão (2026-09-17).** O arquivo entra na conversa como **referência**, do
jeito que a maioria dos agentes de código já faz: `@caminho/da/imagem.png` na
frase, ou o caminho escolhido no seletor `ctrl+f`, ou um link, se a imagem está
na web. Não existe "anexo" — existe uma frase que aponta para um arquivo.

**Implementação e fechamento.**
1. `agent.Message` ganhou partes (`Parts []ContentPart`) com tipos `text` e `image` (base64) mantendo `Content` como string para retrocompatibilidade.
2. O runtime (`internal/harness/runtime/images.go`) escaneia o texto por referências `@caminho/imagem.ext` (`png`, `jpg`, `jpeg`, `webp`, `gif`).
3. Gate de visão: a imagem só é resolvida e codificada se o modelo **declarar `vision`** (`capabilities.go` / `.prumo/models.json`). Se o modelo não declara vision ou o arquivo não existe, a menção permanece texto puro no prompt.
4. Teto explícito: arquivos acima de 10 MB (`MaxImageSize`) são rejeitados com erro explícito em vez de truncamento silencioso.
5. Adapters: tanto o Anthropic quanto o OpenAI-compat empacotam blocos de imagem em base64 / data URL quando `Parts` estiverem presentes.
6. Provas: `images_test.go` (5 testes), `anthropic_test.go` (payload multipart) e `model_test.go` (payload multipart OpenAI-compat).

---

## 2. Ver o diff de um arquivo depois que o run terminou ✅ **fechado** (GAP-084)

**O que é.** Ao final de um run, saber **o que mudou** em cada arquivo, não só
que ele mudou. Hoje `ctrl+g` diz `modified  internal/x.go`; o que faltava era o
conteúdo da mudança, do jeito que um `git diff` mostra.

**Implementação e fechamento (ADR 014, Protocolo 0.4.0).**
1. O protocolo subiu para **0.4.0** registrando a operação `diff` (`{"op": "diff", "run_id": "...", "path": "..."}`).
2. O daemon armazena diffs das ferramentas executadas em `diffs-<runID>.json`, mantendo o payload da timeline mínimo (não re-trafega patches inteiros a cada reconexão).
3. O SDK expõe `Client.Diff(ctx, runID, path)` e o runner expõe `Runner.Diff(ctx, sessionID, path)`.
4. O diálogo de arquivos no TUI (`ctrl+g` ou paleta) permite inspecionar o diff de qualquer arquivo tocado via `Enter` ou `d`, renderizando syntax diff numa viewport rolável e retornando à lista com `Esc` ou `q`.
5. Provas: `TestDaemonDiff` (`daemon_test.go`), `TestRunnerDiff` (`streaming_test.go`), `TestFilesDialogListAndDiff` e `TestFilesDialogDiffError` (`files_test.go`).

---

## 3. O daemon avisar em vez de o cliente perguntar ✅ **fechado** (GAP-085)

**O que é.** O cliente perguntava ao daemon "tem novidade?" **quatro vezes por
segundo** enquanto um run roda (`PollInterval = 250ms`). Um canal push permite
que o daemon escreva à medida que eventos acontecem e o cliente apenas leia.

**Implementação e fechamento (ADR 014, Protocolo 0.4.0).**
1. Operação `subscribe` no protocolo 0.4.0 (`{"op": "subscribe", "run_id": "...", "from": cursor}`).
2. O daemon mantém a conexão aberta, responde com o ack inicial `{"op": "subscribed", "run_id": "...", "from": cursor}` e transmite eventos JSONL à medida que ocorrem com latência 0 ms, fechando ao atingir estados terminais (`run.finished`, `run.completed`, `run.failed`, etc.).
3. O SDK expõe `Client.Subscribe(ctx, runID, from) <-chan Event`.
4. O runner do TUI (`Runner.observe`) conecta ao stream push por padrão. Se o daemon não suportar a operação ou a conexão cair, há **fallback transparente para o loop de polling (`PollInterval`)**.
5. Provas: `TestDaemonSubscribe` (`daemon_test.go`) e `TestRunnerSubscribePush` (`streaming_test.go`).

---

## 4. As quatro verificações de acessibilidade que exigem uma pessoa 🧑

**O que já está feito.** 51 telas de referência (uma por estado por largura de
terminal), o contraste medido em todos os nove temas com o piso de 4.5:1, o
indicador de foco por glifo (não só por cor), o modo de movimento reduzido, o
modo linear sem interface (`--prompt`), e nove dos quinze contratos de
acessibilidade fechados com prova.

**O que falta.** Quatro obrigações que **nenhum teste substitui**, porque cada
uma é "como isto é na prática, para uma pessoa":

| O que verificar | Como fazer | Onde registrar |
|---|---|---|
| Leitor de tela | Ligar NVDA/VoiceOver/Orca, rodar `prumo-agent tui --prompt "liste os arquivos"` e ouvir: o que é falado faz sentido sozinho? Depois percorrer os atalhos da interface normal | `accessibility.screen-reader` |
| Contraste renderizado | Olhar os temas num terminal de verdade, incluindo um sem cor | `accessibility.contrast` |
| Movimento | Ligar `--reduced-motion`, confirmar que nada anima e que a informação continua na tela | `accessibility.motion` |
| Foco | Percorrer a interface só de teclado, incluindo um diálogo aberto e fechado | `accessibility.focus` |

**Quem destrava.** Uma pessoa, com o leitor de tela instalado. É uma sessão.

**Por que o documento insiste nisso.** Porque a alternativa é escrever "verificado"
sem ninguém ter verificado — e foi exatamente esse tipo de afirmação que este
repositório passou a tratar como defeito.

---

## 5. As quatro decisões herdadas: o que são e o que eu recomendo 🟡

Estas não são do cliente de terminal: são decisões do framework que estão
paradas. Elas ficam aqui porque "o que está aberto" não pode parecer menor do que
é — e porque duas delas bloqueiam uma à outra.

### SQLite: qual biblioteca o runtime usa para índices derivados — **decidido: aceito**

**O que é.** O runtime guarda coisas que podem ser recalculadas (índices de
busca, contexto compilado). A decisão é *com que biblioteca* de SQLite ele faz
isso: uma escrita em Go puro (sem depender de compilador C) ou uma que usa C e é
mais rápida.

**Por que está parada.** O ADR 007 está `Proposed (pending measurement)`: espera
medir o primeiro componente que realmente precise dela.

**Recomendo aceitar agora**, e manter a medição como acompanhamento em vez de
pré-requisito. Três razões: a escolha é reversível **por construção** (o driver
vive atrás de uma porta, então trocar é mexer em um arquivo, não em quem chama);
o custo de errar é baixo e conhecido; e o item **bloqueia o ADR 009**, que espera
"o primeiro componente SQLite" para fixar teto de memória. Esperar por algo que
só existe depois da decisão é circular.

**Ação:** mudar o status do ADR 007 para `Accepted`, mantendo a cláusula de
emenda (medir e trocar atrás da porta, se a medição mostrar vantagem).

### Budgets de performance: quanto cada coisa pode gastar — **decidido: aceito**

**O que é.** Duas coisas: quanto tempo e quanta memória cada operação do harness
pode gastar antes de virar defeito, e por quanto tempo guardar estado derivado.

**Por que está parada.** O ADR 009 está `Proposed (thresholds pending benchmark
corpus)`: os limites numéricos esperam um corpus de medições.

**Recomendo aceitar a política já**, porque o próprio ADR decide *não* inventar
número ("thresholds are set only from measured distributions"). Aceitar não
inventa precisão: torna vinculante o que já é decidível — reportar latência
honesta contra baselines gravados, tratar regressão em benchmark como falha, e
guardar estado derivado por 7 dias com limpeza no start. Os números entram quando
houver distribuição medida, e aí o ADR é emendado.

**Ação:** status para `Accepted`; a parte numérica continua explicitamente em
aberto, e o ADR já diz por quê.

### Sandbox: onde as ferramentas do agente rodam — **decidido: medir podman rootless**

**O que é.** Rodar as ferramentas (comando, edição de arquivo) dentro de uma
caixa isolada, com limites de memória/CPU e sem rede.

**Por que está parada.** O ADR 006 está aceito e o docker funciona; o que falta é
o modo **rootless** medido (o preferido por ser menos privilégio) e "provedores
adicionais".

**Evidência fresca (2026-09-17, esta máquina):** rodei a verificação live e ela
**passa** — docker 29.7.2, sandbox aplicando os limites, 1,6 s. **Podman não está
instalado** aqui, e é justamente o ponto que o ADR 006 deixou aberto.

**Recomendo duas coisas, e uma recusa.** (1) Instalar podman e rodar o mesmo
teste: fecha o ponto aberto **sem escrever código**. (2) **Não** adotar
runsc/gVisor agora: é um provedor a mais sem necessidade demonstrada, e o preço é
um runtime privilegiado na máquina de quem usa.

**Ação:** `sudo apt install podman` e, depois,
`PRUMO_LIVE_DOCKER=1 go test ./internal/harness/aci/ -run TestContainerLive`.
Passando, registrar a medição no ADR 006.

### Local Intel: modelo local — **decidido: deferir até haver caso de uso**

**O que é.** Contratos para rodar modelo local (roteamento, supervisão, medição de
recurso). Os contratos existem; faltam os workers de inferência e os benchmarks.

**Por que está parada.** Os benchmarks dependem de hardware, corpus e aprovação —
o GAP-040 está marcado como bloqueado por ambiente e por decisão.

**Recomendo deferir de propósito.** Sem decidir *qual* caso de uso local importa
(privacidade? custo? funcionar offline?) e com que qualidade mínima, um worker de
inferência é código especulativo que envelhece enquanto a decisão não vem. O
gatilho para retomar é a decisão, não mais código.

**Ação:** nenhuma agora. Manter os contratos e marcar o item como "espera decisão
de caso de uso".

## 6. Providers gratuitos, e o opencode que já está construído ✅ **fechado** (GAP-093)

**O que você pediu.** Que o harness e o cliente facilitem provider gratuito —
principalmente o opencode, que "já deveria ser facilmente acoplável, não apenas
por api-key".

**Implementação e fechamento (GAP-093).**
Opção (a) implementada: `--provider opencode` executa turnos delegados, delegando
ferramentas e autenticação ao opencode instalado (`Capabilities().ToolCalls=false`),
com medição de uso/cache/custo capturada do protocolo JSON e timeline declarando
a execução delegada.

---

## 7. Imagem no run: o que falta, agora que o gate existe ✅ **fechado** (GAP-090)

**O que você pediu.** Marcar uma imagem com `@` e, se o modelo tiver vision, ele
entende a imagem — sem sair do TUI.

**Implementação e fechamento.**
Fechado e verificado pelo item 1 (GAP-090): partes em `agent.Message`, resolução
de `@path` em `internal/harness/runtime/images.go` sob gate de `vision` declarado
no workspace (`.prumo/models.json`), limite explícito de 10 MB, e codificação
multipart nos adapters Anthropic e OpenAI-compat.

---

## 9. Pesquisar capacidades de modelos na internet ⬜ (desenho, não implementado)

**Sua ideia.** Quando um modelo entra nos nossos registros, um modelo barato ou
gratuito pesquisaria o que ele oferece e atualizaria a ficha.

**Dá, e o encaixe já existe.** A ficha (`.prumo/models.json`) alimenta o
`model_info`, que já chega ao TUI; o harness já tem rede e uma ferramenta de
busca/fetch. Não falta cano — faltam **duas decisões**, e as duas são sobre
confiança, não sobre código:

1. **Procedência.** O que a pesquisa produz é uma *afirmação sobre um modelo*
   vinda de uma fonte que ninguém conferiu. Ela precisa ser gravada **com** a
   fonte, a data e quem pesquisou — e em arquivo próprio, para que a declaração
   escrita à mão **sempre vença** no merge. Sem isso, a ficha do modelo deixa de
   distinguir "o fabricante diz" de "alguém escreveu".

2. **Quem escreve.** O texto que voltar da web é **entrada externa**: uma página
   pode conter instruções dirigidas ao modelo que está pesquisando. Se a pesquisa
   gravar direto no arquivo, isso deixa de ser uma pesquisa e passa a ser um
   caminho para escrever na sua configuração. Então a pesquisa **propõe** — mostra
   o que achou, com a fonte, e grava só com confirmação explícita.

**O que eu recomendo.** Fazer, nessa forma: um comando que pesquisa, mostra a
proposta com as fontes e grava em `.prumo/models.researched.json` só depois de um
"sim"; o merge nunca deixa o pesquisado sobrescrever o declarado; e a ficha
mostra a diferença ("declarado" x "pesquisado em <data>"). É meia hora de
máquina e uma decisão sua.

**Tamanho.** M, depois da decisão.

## 10. Por que o registro dizia "feito" para coisas que não funcionavam

> Auditoria de 2026-09-23. GAP-001/003/005/013/015/021/022 e o veredito de split.

**O que é.** Você está lendo um documento que falava. Não é problema de texto:
é um padrão repetido sete vezes no registro oficial do projeto.

Cada uma dessas sete linhas dizia que uma capacidade estava pronta, com prova de
teste ao lado. As provas eram reais — os testes existiam e passavam. O que não
existia era o **caminho que um programa real percorre para chegar lá**.

O exemplo mais claro: roteamento de modelo por custo e cota. Existe um motor
completo de escolha, com política, cooldown, retry e fallback. Tem testes. O
documento dizia "feito". Ele está no caminho real de alguma execução? Não. O
programa de agente pede um modelo direto a um fornecedor, como um interruptor
com posições — e o motor de escolha fica ao lado, desligado, usado só pelos
próprios testes dele.

O mesmo aconteceu com subagentes, orçamento no daemon, aprovações depois de
reinício, trava do processo do daemon, e o registro de tentativas.

**Por que isso importa mais do que parece.** Um registro que erra a direção faz
duas coisas ruins ao mesmo tempo: esconde o que falta, e faz com que o trabalho
próximo seja escolhido errado. A regra nova é explícita — **"feito" exige que
um programa de produção use aquilo, não que um teste consiga**.

**O que destrava.** Onda 0 (evidência e exit code honestos) e Onda 2 (runtime e
daemon). A regra passa a ser vigiada por teste, para não voltar a apodecer.

**Tamanho.** G.

---

## 11. Um comando que falha aparece como sucesso

> GAP-097, GAP-123. A onda mais urgente do projeto.

**O que é.** Quando o agente roda `go test ./...` e os testes falham, o Prumo
registra que o comando **passou**. Não é uma imprecisão: o código de execução
ignora o erro do comando e escreve "código de saída: 0" de volta. O comando de
teste ainda passa por um `head` no meio do caminho, que troca o status real pelo
status do `head`.

**Por que importa.** A regra do próprio framework é que evidência decide
conclusão, não a confiança do modelo. Se a evidência mente, essa regra inverte:
o sistema passa a aprovar trabalho quebrado com confiança total, e a fraude é
invisível porque o registro está todo verde.

**O que destrava.** Ler o erro real do comando e repassar o código de saída real.
Adicionar testes que falham quando um comando falha. E, no mesmo passo, fazer o
`resume` continuar o trabalho em vez de declarar concluído.

**Tamanho.** P para a correção. G para fechar a cadeia de evidência inteira.

---

## 12. Três furos de segurança que um programa externo consegue abrir

> GAP-110, GAP-111, GAP-112. Onda 1.

**O que é.** Três coisas separadas que, juntas, significam que o agente não pode
confiar que está preso no seu projeto:

1. **Link simbólico.** A trava de pasta só olha o texto do caminho, não o destino
   real. Um link para `/etc` dentro do projeto passa.
2. **Endereço do fornecedor.** O daemon aceita o endereço do fornecedor que o
   cliente pede e, se a chave não vier junto, usa a chave **do processo servidor**
   e envia para aquele endereço. Quem consegue falar com o socket local escolhe
   para onde a sua chave vai.
3. **Nome de execução.** O identificador do run entra no caminho do arquivo sem
   verificação. Um identificador com `../` escapa da pasta de estado.

**Por que importa.** A promessa de que o agente trabalha dentro do seu workspace
é a base de tudo — sem ela, nenhum modo de permissão faz sentido.

**O que destrava.** Uma camada única de contenção de caminho, resolvendo o
destino real antes de qualquer operação; lista de destinos permitidos para o
fornecedor; validação de identificador antes de virar caminho.

**Tamanho.** M para os três. G para o ambiente filho saneado e o resto.

---

## Se a pergunta é "por onde começo"

1. **Onda 0 — o comando que falha aparecer como sucesso (GAP-097).** É a
   única coisa aqui que faz o resto do trabalho mentir. Comece por esta.
2. **Onda 1 — os três furos de segurança (GAP-110, GAP-111, GAP-112).** Antes de
   rodar o agente em qualquer máquina que não seja a sua de teste.
3. **Onda 4 — roteamento com cota.** A resposta direta à sua pergunta sobre o
   roteador inteligente: ele existe, funciona quando chamado, e **não é chamado**.
   É a peça com mais código pronto e menos ligado.
4. **Onda 6 — instalação global.** Sete coisas concretas faltam, listadas no
   ADR 018: serviço de usuário, registro de projetos, executor por projeto,
   cofre de credenciais, identidade, rolover de custo, e parar de apagar
   arquivos de outro projeto na desinstalação.
5. **Onda 5 — subagentes de um nível.** Decidido no ADR 017. O que falta é o
   caminho de produção e o isolamento de verdade.
6. **Item 4 (acessibilidade)** — não depende de código: depende de uma sessão
   com o leitor de tela ligado e conferência manual em terminal real.
7. **Item 9 (pesquisa assistida de modelos)** — aguarda decisão de produto.

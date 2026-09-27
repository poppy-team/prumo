# 79.Z — Teaching Module, Professor Agents e Neurodivergent Learning Architecture

> Authority: canonical specification.
> Logical ID: 79 Z
> Source: Notion Living Book (3dd9bb7d023f8156ace1f5075fc16d3a)
> Status: Documento especializado: 79 Z — Teaching Module, Professor Agents e Neurodi.


<aside>
🎓

**Proposta arquitetural:** adicionar ao Prumo um primitive nativo de **Teaching**. Ele não é apenas um Agent com personalidade. Seu objetivo é preservar agency do aprendiz, acompanhar domínio ao longo do tempo, adaptar scaffolding e transformar desenvolvimento real em aprendizagem deliberada.

</aside>

# Por que Teaching não cabe apenas em Agent/Skill/Recipe

Agent modela **quem executa**; Skill modela capacidade; Recipe modela processo. Ensino precisa ainda de learner state, objetivos pedagógicos, prerequisites, mastery, misconception model, assessment, hint progression, spacing/retrieval e autonomia sobre quanto código o sistema pode produzir.

# Primitive proposto

```
Teacher / Teaching Profile
├── professor role
├── pedagogy policy
├── learner model
├── curriculum/knowledge graph
├── domain skills
├── lesson state
├── assessment engine
├── hint/scaffold ladder
├── practice/retrieval scheduler
└── learning evidence
```

Nome interno preferido: `Teacher`, `TeachingProfile` e `LearningSession`. Na UI pode aparecer como **Professor Mode / Learn Mode**.

# Modos de sessão

- **Learn:** explicação incremental + exercícios; implementação principal fica com o usuário.
- **Pair:** usuário e professor desenvolvem juntos; professor pergunta/explica antes de mudanças relevantes.
- **Review:** professor não reescreve por default; identifica issues e conduz diagnóstico.
- **Practice:** desafios isolados com hints graduais.
- **Deep Dive:** first principles, matemática/arquitetura/semântica profunda.
- **Exam/Check:** avaliação sem pistas até o término do item, seguida de feedback.

# Code Agency Contract

O Teaching Mode precisa de uma política explícita:

1. não despejar solução completa para exercício ativo por default;
2. começar por pergunta diagnóstica ou pista mínima quando apropriado;
3. hint ladder: direção → conceito → pseudocódigo → fragmento → solução completa;
4. usuário pode pedir `show solution` a qualquer momento;
5. em bug real bloqueante, professor pode oferecer patch, mas deve separar “correção” de “explicação”;
6. nunca esconder informação para manipular dificuldade; sempre permitir override;
7. não usar perguntas socráticas mecanicamente quando uma explicação direta for mais útil.

# Learner Model

Estado versionado e editável pelo usuário:

- goals e projeto atual;
- concepts `unknown / introduced / practicing / reliable / transferable`;
- prerequisites;
- misconceptions observadas com evidence;
- exercises/tentativas;
- preferred representation e accessibility settings;
- pace/chunk size;
- hints used;
- confidence **autodeclarada**, nunca inferir capacidade intelectual;
- spaced retrieval schedule opcional.

Não diagnosticar TDAH/dislexia nem inferir condição clínica; profiles neurodivergentes são preferências/accommodations escolhidas pelo usuário.

# Neurodivergent pedagogy

A base deve usar UDL 3.0 + cognitive accessibility e pesquisa de aprendizagem.

## TDAH-friendly profile

Chunking curto; objetivo visível; uma decisão principal por vez; reduzir distração e forks; checkpoints frequentes; externalizar estado/progresso; tarefas com começo/fim claros; retomada após interrupção; exemplos ligados ao projeto real; permitir alternância teoria↔prática.

## Dyslexia-friendly profile

Texto escaneável; vocabulário definido antes de abstrações; evitar blocos extensos; não depender de grafia para avaliar compreensão conceitual; múltiplas representações; diagramas/tabelas quando ajudam; código e termos alinhados; recaps; possibilidade de áudio/TTS quando harness suportar.

## Universal rules

Nunca infantilizar, diminuir rigor ou assumir que neurodivergência implica menor capacidade. O sistema adapta **representação, pacing e scaffolding**, não o teto intelectual.

# Evidence-based instructional mechanics

A prática do IES recomenda spaced learning, worked examples intercalados com resolução, combinar representações gráficas e verbais, integrar abstrato↔concreto e retrieval practice. CAST UDL 3.0 enfatiza múltiplos meios de engagement, representation e action/expression, além de agency do aprendiz.

## Teaching loop

```
Orient → Activate prior knowledge → Explain one concept → Worked example →
Learner attempt → Diagnose → Minimal hint/feedback → Retry →
Explain misconception → Transfer problem → Retrieval checkpoint → Recap
```

Nem toda interação usa todos os passos.

# Professor Agents

Criar um `professor-general` que resolve domain packs, e personas técnicas somente quando justificadas por knowledge/authority:

- `professor-programming`
- `professor-computer-science`
- `professor-software-engineering`
- `professor-systems`
- `professor-databases`
- `professor-math`
- `professor-physics`
- `professor-compilers-languages`

Esses professores compartilham a mesma pedagogia base; não duplicam enciclopédias. Domain knowledge vem das skills.

# Automatic teaching skills

Quando Teaching Mode estiver ativo, estas skills são `required/contextual`:

`cognitive-accessibility`, `concept-decomposition`, `abstract-to-concrete`, `worked-examples`, `hint-ladder`, `formative-assessment`, `retrieval-practice`, `misconception-diagnosis`, `learning-recap`, `technical-vocabulary`, `visual-explanation`, `code-reading`, `debugging-as-learning`, `transfer-exercises`.

# Abstract-to-concrete skill

Para conceitos profundos usar quatro camadas quando útil:

1. **intuição concreta**;
2. **modelo técnico preciso**;
3. **representação formal/matemática**;
4. **implementação e consequências práticas**.

Ex.: pointer → endereço/alias/lifetime → modelo de memória → assembly/ABI → bug real.

# Assessment Engine

Formative assessment, não scoring opaco. Medir concept recall, application, debugging, explanation, transfer e retention. Mastery exige evidence em múltiplos contextos; uma resposta correta isolada não promove automaticamente conceito a `reliable`.

# Teaching documentation artifacts

`LEARNING_GOALS.md`, concept map, glossary, lesson journal, misconception log, exercise bank, retrieval queue, learner-authored notes e project milestones. O transcript não deve ser a memória pedagógica primária.

# Privacy e control

Learner model local/project-scoped por default; usuário pode apagar/exportar; evitar armazenar disability labels quando uma preferência concreta basta. “preciso de blocos curtos” é melhor metadata que uma inferência clínica.

# Evals

Comparar baseline chat vs Teaching Mode: correctness, retained recall, transfer, hint dependence, user code contribution, misconception repair, cognitive load autodeclarada e over-helping rate. Incluir evals adversariais: aluno pede resposta durante exercise; aluno está bloqueado; conceito avançado com prerequisite ausente; resposta correta por acaso; interrupção e retorno dias depois.

# Research roots

- CAST UDL 3.0: [https://udlguidelines.cast.org/](https://udlguidelines.cast.org/)
- IES Practice Guide: [https://ies.ed.gov/ncee/wwc/PracticeGuide/1](https://ies.ed.gov/ncee/wwc/PracticeGuide/1)
- ACM CS2023: [https://csed.acm.org/](https://csed.acm.org/)
- SWEBOK v4: [https://www.computer.org/education/bodies-of-knowledge/software-engineering](https://www.computer.org/education/bodies-of-knowledge/software-engineering)

# Exit Gate

Teaching Mode é aprovado quando aumenta aprendizagem mensurável sem diminuir autonomia: ajuda suficiente para avançar, pouca o bastante para o usuário continuar pensando, implementando e explicando.
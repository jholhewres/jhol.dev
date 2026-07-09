---
title: "Meu Fluxo de Trabalho com IA: Dois Modelos, o Anchored e um Plano Que Não é /plan"
date: 2026-07-08
tags: ["ai", "workflow", "anchored", "claude-code", "glm", "productivity"]
summary: "Como eu codo hoje: dois modelos (Opus 4.8 e GLM 5.2), três Claude Codes, memória de mais de 60 projetos no Anchored e um fluxo de quatro etapas — levantamento de contexto, plano com agentes, execução com code review por fase e testes reais — que substituiu o /plan que eu nunca curti."
image: cover.png
reading_time: 12
---

Desde o começo do ano eu venho testando de tudo: modelos, IDEs, CLIs, skills prontas, skills que escrevi, PRDs, templates, padrões de documentação. O objetivo era um só — dar conta de dezenas de projetos ao mesmo tempo, entre trabalho e projetos pessoais, sem perder o fio. Começar uma task, terminar, e conseguir retomar depois numa sessão nova ou resumida com o mesmo controle e a mesma agilidade.

Depois de meses trocando ideia quase todo dia com o [Aron](https://github.com/aronpc) — o fluxo dele é bem parecido com o meu —, cheguei num workflow que finalmente parou de me atrapalhar. Este post é sobre ele: os modelos que uso, o que torna tudo isso possível (spoiler: [Anchored](/blog/anchored-cross-tool-ai-memory-mcp)), e o fluxo de quatro etapas que substituiu o `/plan` que eu nunca gostei.

## A Base: Dois Modelos, Três Claude Codes

Passei o ano testando os modelos chineses — Qwen, Kimi, e por aí vai. No fim, tudo convergiu pra **dois**:

- **Claude Opus 4.8** — o modelo primário. Análise, arquitetura, planejamento, o trabalho que não pode errar.
- **GLM 5.2** (conta Coding da Z.ai) — pra segurar as barras menores sem torrar meus tokens primários.

E rodo isso em três Claude Codes, cada um com um comando no terminal:

```
Comando     Modelo               Uso
-------     ------               ---
claude   →  Claude Opus 4.8      profissional (plano da empresa)
claudep  →  Claude Opus 4.8      pessoal (meu próprio plano)
glm      →  GLM 5.2 (Z.ai)       barra menor / poupar tokens primários
```

A lógica é de economia de contexto e de custo, não de capacidade. O GLM 5.2, na minha opinião, está no nível do Opus — perde por alguns detalhes. Então tarefa que não exige o topo da régua vai pro `glm`, e eu reservo os tokens primários do Opus pro que realmente pesa. Dois modelos, três terminais, uma régua clara de quando usar cada um.

Rodar três sessões em paralelo cria um problema imediato: saber qual terminou, qual ainda está processando, e qual está parada há dez minutos te esperando aprovar algo. Pra isso eu uso — e recomendo — o **[AI Traffic Lights](https://github.com/aronpc/ai-traffic-lights)**, outro projeto do Aron: um overlay que mostra cada sessão de agente como um farol (🟢 pronta · 🟡 trabalhando · 🔴 precisa de você), com clique pra pular direto pro terminal (e a aba) certos, e ainda medidores de uso por agente — inclusive do GLM Coding Plan, o que fecha bem com a lógica de não torrar os tokens primários.

## O Que Torna Tudo Isso Possível: Anchored

Nada disso funcionaria se cada sessão nova começasse do zero. O que segura o fluxo é o [Anchored](/blog/anchored-oss-team-memory-server) rodando no dia a dia, com memória de **mais de 60 projetos** e múltiplos servidores remotos separados por contexto:

```
                        ANCHORED (memória cross-tool)
        ┌───────────────────┬───────────────────┬───────────────────┐
   remote PESSOAL      remote EXTERNO       remote EMPRESA
   projetos próprios   projetos externos    compartilhado com colegas
        └───────────────────┴───────────────────┴───────────────────┘
                                   │
              60+ projetos: stack · acessos · comandos
              learnings · decisões · knowledge graph · histórico
```

O remote da empresa é o que muda o jogo no corporativo: eu vejo o que os colegas estão fazendo e aproveito o contexto do que eles já implementaram. O aprendizado circula sem ninguém escrever documentação.

O melhor exemplo é puxar uma task que eu não conheço. Antes de qualquer planejamento, num único levantamento de contexto eu consigo:

- Puxar **memórias do recurso** que a task toca — quem implementou, quando, e o que ficou registrado.
- Ver **tarefas anteriores** relacionadas, de qualquer status.
- Saber se a minha task **depende de outra** ou se ela **destrava um frontend**.
- Avaliar se ela **tem contexto suficiente pra ser entregue** — ou se falta informação antes mesmo de começar.

Isso tudo sai da memória remota, boa parte vinda de colegas. É a diferença entre começar cego e começar com o mapa na mão.

## Como Eu Codo Hoje

Muita gente ainda vive no `/plan` das ferramentas. Eu nunca curti — sempre achei que faltava alguma coisa. E faltava mesmo: contexto de verdade antes do plano, e verificação de verdade depois dele. O fluxo que desenvolvi tem quatro etapas:

```
   TASK  ──────────►  [1] LEVANTAMENTO DE CONTEXTO
   (Jira / pessoal)         memória remota (Anchored) + colegas
                            logs · CloudWatch · Datadog
                            deps/bloqueios · quem fez · quando
                            "tem contexto suficiente pra entregar?"
                                   │
                                   ▼
                         [2] PLANO  (OMC — sem /plan)
                             arquiteto · designer · frontend
                             └─ spawn de Sonnet 5 pro trabalho pesado
                                de leitura de código
                                   │
                                   ▼
                         [3] EXECUÇÃO  (omc ralph)
                             fase 1 ─► code review ─► correções
                             fase 2 ─► code review ─► correções
                             fase N ─► code review ─► correções
                                   │
                                   ▼
                         [4] TESTES REAIS
                             endpoints · jobs · filas · workers
                             cenários de erro · validações
                             ┌──── ajuste pontual + teste unitário
                             │           (loop até cobrir tudo)
                             ▼
                            ENTREGA  (code review costuma vir limpo)
```

### 1. Levantamento de contexto

Feature, task simples ou defect — não importa: levantamento de contexto é o que garante que a task carrega tudo que precisa. E fazê-lo com informação precisa da **memória remota vinda de outros colegas** deixa isso muito melhor.

Mas não paro na memória. Eu dou acesso a **logs**, entro no **CloudWatch**, no **Datadog**, e monto todo o contexto necessário antes de pensar em plano. Se falta informação pra entregar, isso aparece aqui — não no meio da implementação.

### 2. O plano (sem /plan)

Em vez do `/plan`, uso o plugin **[Oh My Claude Code (OMC)](https://github.com/yeachan-heo/oh-my-claudecode)**. Nele tenho os agentes certos pra escrever o plano: dependendo da complexidade, entra um **arquiteto**, um **designer**, um **frontend**. E quando um deles precisa de mais informação do código, ele mesmo faz spawn de modelos menores — **Claude Sonnet 5** — pra fazer o trabalho "pesado" de leitura.

Repare na divisão: Sonnet faz o braçal de vasculhar o código, mas **análise e planejamento ficam com o Opus**. Sonnet pra planejar não rola — perde a mão nos trade-offs.

### 3. Execução com Ralph + code review por fase

Com o plano na mão — e eu costumo escrever planos com mais de duas fases de execução — parto pra execução com o **Ralph**, via OMC (`omc ralph`). Ele segue com tudo até o fim.

O detalhe que faz diferença: **eu incluo um ciclo de code review no plano, por fase.** Por quê? Code review de muitos arquivos de uma vez não é útil — a não ser que você consiga spawnar vários agentes pra dividir. Então a cada fase concluída eu faço o code review, aplico as correções, e só então sigo pra próxima. É o jeito que me dá código confiável sem afogar o review numa fase gigante.

### 4. Testes reais

Acabou? Não. A última etapa são os **testes reais**.

Testes unitários são essenciais e requisito de entrega — mas seu colega vai testar a sua task de verdade. Você vai confiar numa entrega só porque tem teste unitário gerado por IA? Eu não.

Então eu executo **tudo** que foi implementado: endpoints, jobs, filas, workers. Testo vários cenários, cenários de erro, validações, tentando cobrir o máximo. **Umas 80% das vezes** algo precisa de ajuste ou melhoria pontual pra deixar todos os cenários mapeados e cobertos. Quando corrijo ou melhoro, **o teste unitário vai junto** — e aí sim tenho a entrega pronta.

Explicando assim parece que leva um dia inteiro. Não leva: são poucas horas mesmo em tasks maiores. Cada projeto tem a burocracia dele pra começar ou pra testar, claro, mas no fim os code reviews dificilmente voltam com comentários e eu sigo entregando.

## Autonomia Sob Guardrails

O mesmo fluxo roda em qualquer projeto, mas o quanto eu solto a rédea do agente muda conforme o risco do ambiente.

Nos **meus projetos pessoais** eu dou mais autonomia: gerenciador de senhas, **guardrails** de proteção contra vazamento e contra comandos de detonação, e a partir daí o agente faz deploys, correções, leitura de logs, chamadas de API e acesso aos bancos — tudo rápido, porque **está tudo no Anchored**.

É a memória que sustenta isso. Nela está registrado, por projeto: como buscar os acessos e como usá-los, onde cada projeto vive, qual a stack, quais comandos precisam rodar, e se houve erros antes. A categoria `learning` é a que mais rende — toda sessão nova começa com um aprendizado que já foi pago numa sessão passada.

## O Que Aprendi

**Dois modelos com uma régua clara batem cinco modelos sem régua.** Passei o ano testando de tudo; o ganho não veio de achar o modelo perfeito, veio de reduzir pra dois (Opus 4.8 e GLM 5.2) e saber exatamente quando usar cada um. Menos decisão por sessão, mais tokens primários pro que importa.

**Contexto antes do plano vale mais que o plano.** O `/plan` me incomodava porque atacava a etapa errada. O gargalo nunca foi escrever o plano — era chegar no plano sem saber se a task depende de algo, quem mexeu ali antes, ou se ela sequer tem contexto pra ser entregue. Anchored + logs + Datadog resolvem isso antes da primeira linha.

**Planejamento é trabalho de modelo caro; leitura de código é trabalho de modelo barato.** Deixar o Sonnet vasculhar arquivos e o Opus decidir arquitetura é a divisão certa. O contrário — Sonnet planejando — é onde os trade-offs se perdem.

**Code review por fase, não no fim.** Revisar uma fase de cada vez mantém o review pequeno e útil. Revisar tudo junto no final, sem um exército de agentes, é onde os bugs se escondem.

**Teste real é o que separa "passou no CI" de "entregue".** Teste unitário gerado por IA valida a lógica que a IA imaginou. Executar endpoint, job, fila e worker de verdade, com cenários de erro, é o que revela o que faltou — e em 80% das vezes falta algo. Só depois disso o unitário vira verdade.

**A memória é o que faz o fluxo escalar pra 60 projetos.** Nada disso — dois modelos, três terminais, quatro etapas — sobrevive ao terceiro projeto sem uma memória cross-tool que lembre stack, acessos, comandos e erros passados por você. O Anchored é a peça que transforma "processo bonito" em "processo que eu de fato uso todo dia".

## Agradecimentos

De novo ao [Aron](https://github.com/aronpc), amigo e colega de trabalho, com quem eu troco ideia sobre esse workflow quase todo dia. Metade do que está aqui nasceu dessas conversas — o fluxo dele é primo do meu, e é comparando os dois que cada um foi afinando o próprio.

---

**Links:**

- [Anchored (cliente local) no GitHub](https://github.com/jholhewres/anchored)
- [Anchored OSS (servidor de time) no GitHub](https://github.com/jholhewres/anchored_oss)
- [AI Traffic Lights (Aron) no GitHub](https://github.com/aronpc/ai-traffic-lights)
- [Post: Anchored — Uma Memória Pra Todas as Ferramentas](/blog/anchored-cross-tool-ai-memory-mcp)
- [Post: Anchored OSS — Memória de Time Pra Quem Programa com IA](/blog/anchored-oss-team-memory-server)
- [Model Context Protocol](https://modelcontextprotocol.io/)

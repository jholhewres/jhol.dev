---
title: "devpit: Um Espaço de Trabalho Por Projeto Pra Quem Programa com Agentes"
date: 2026-09-23
tags: ["ai", "devpit", "ade", "claude-code", "rust", "tauri", "open-source"]
summary: "Cada projeto que eu tocava gerava planos, revisões, pesquisas e artefatos que não iam pro git, e eu vivia criando diretórios invisíveis pra dar conta. O devpit é a minha tentativa de uma ADE (Agentic Development Environment): um espaço por projeto com terminal, chat, agentes, um quadro que executa o trabalho, code review, testes, notas e Excalidraw."
image: cover.png
reading_time: 9
---

Todo projeto que eu toco com agente deixa um rastro. É plano, refinamento do plano, revisão, code review de cada fase, pesquisa de referência, decisão que eu tomei e o motivo, print de bug, rascunho de prompt. Boa parte disso não deveria subir pro repositório — é material interno, é a história de como o código nasceu. Mas também não pode se perder, porque é justamente o que eu preciso quando volto no projeto dois dias depois.

A minha solução, por muito tempo, foi criar diretório invisível pro git. Um `.omc/` aqui, um `.project/` ali, um `.claude/`, cada ferramenta com a sua pasta e a sua convenção, tudo jogado no `.gitignore`. Funcionava, até eu ter vários projetos abertos ao mesmo tempo e não lembrar mais em qual pasta de qual projeto estava aquele plano que eu tinha revisado na semana anterior.

E tem as ideias. No meio de uma task sempre aparece alguma coisa que não é da task — uma feature, um refactor, um "e se a gente fizesse assim". Anotar onde? Num arquivo solto, no Jira, numa conversa com o agente que daqui a pouco vai ser compactada? Eu cheguei a escrever uma skill só pra ir apendando ideia num `ideias.md`.

Com o tempo ficou claro que o problema não era falta de ferramenta. Era que o lugar onde eu pensava o trabalho, o lugar onde ele rodava e o lugar onde eu revisava eram três lugares diferentes, e manter os três em dia era trabalho manual que eu simplesmente não fazia.

Foi daí que nasceu o **[devpit](https://github.com/jholhewres/devpit)**.

## O Que é uma ADE

IDE é *Integrated Development Environment*: o centro é o editor e tudo gira em volta do arquivo aberto. Quando o trabalho passa a ser conduzido por agente, esse centro muda. Eu não passo mais o dia editando arquivo — passo descrevendo o que precisa ser feito, acompanhando sessão, revisando diff, rodando teste e decidindo o próximo passo.

Uma **ADE — Agentic Development Environment** — é um ambiente pensado em volta disso. No devpit, o centro não é o arquivo, é o **projeto**, e dentro de cada projeto fica tudo que eu preciso pra trabalhar com agente:

```
                         devpit
  ┌──────────────────────────────────────────────────────┐
  │  projeto A      projeto B      projeto C      ...    │
  └──────┬───────────────────────────────────────────────┘
         │
         ├── terminal           o Claude Code de sempre
         ├── chat               com custo, perguntas e checklist na tela
         ├── quadro             mover o card é o que dispara o trabalho
         ├── agentes            os markdowns que já estão na sua máquina
         ├── arquivos + diff    o code review sem sair dali
         ├── runs               o que rodou, quanto custou, o que provou
         ├── navegador          a página que o projeto está servindo
         ├── notas              markdown salvo na pasta do projeto
         └── excalidraw         desenhos salvos na pasta do projeto
```

A ideia é que eu abra um projeto e tenha tudo ali. Troquei de projeto, troquei de espaço inteiro — terminal, quadro, notas, conversas — e quando eu volto, está do jeito que eu deixei.

Também tomei cuidado com o outro lado: não queria construir mais uma IDE. Aba aberta, layout de painel, arquivo em edição, isso é detalhe de tela. O que importa é o projeto, o trabalho e o que foi feito nele.

## O Que Me Incomodava

Antes de começar, eu escrevi o problema do meu jeito. Eram vários pedidos, mas no fundo eram três incômodos.

**Eu perdia a visão das tasks.** Não por falta de kanban — eu já tinha kanban. O problema é que o card descrevia o trabalho num lugar e o trabalho acontecia em outro. Um quadro que só descreve fica desatualizado no primeiro dia corrido. No devpit eu inverti isso: mover o card é o que faz o trabalho acontecer. Se o card está em "revisar", é porque a revisão rodou ali.

**A orquestração era vaga.** Quando eu chamava um comando que subia agentes sozinho, eu não sabia quantos iam subir, quanto ia custar, onde o estado tinha ficado nem em que passo estava. Os prompts eram bons, o que faltava era contorno: uma etapa por vez, visível, com teto de gasto.

**Eram terminais demais.** Cinco frentes em cinco terminais com cinco Claude Codes parece produtivo, mas na prática vira uma sala de alarme. Eu queria um terminal em foco por projeto, e o fluxo inteiro conversando com ele.

## Um Terminal Por Projeto

Cada projeto tem o seu terminal, e é ali que o Claude Code roda do jeito que sempre rodou. Você abre o devpit, digita, e é o Claude Code de sempre. Não precisa criar card nem configurar nada pra usar — o quadro é opcional.

Trocar de projeto troca o que está na tela, mas a sessão anterior continua rodando em segundo plano. Os terminais são sessões do **tmux**, então fechar o devpit não mata nada: abro de novo e está tudo lá, inclusive depois que o app se atualiza sozinho.

Dá pra rodar a mesma CLI com contas diferentes usando **perfis**. É o meu `claude` / `claudep` / `glm` do [post sobre o meu workflow](/blog/ai-coding-workflow-two-models-anchored), só que dentro do app. Codex, Gemini, Cursor, Aider e outras CLIs o devpit reconhece e abre no terminal; conduzir de ponta a ponta, por enquanto, só o Claude Code.

## O Quadro Que Executa

Essa é a parte que substituiu os meus comandos de orquestração. Cada projeto tem um quadro, e as colunas são minhas: renomeio, reordeno, crio as que eu quiser. Uma coluna pode não fazer nada, ou pode executar uma **etapa** quando um card chega nela. As etapas são de três tipos:

- **agent** — um agente roda em segundo plano, sem ocupar o terminal, e devolve uma resposta estruturada com o custo. Bom pra refinar, revisar e verificar.
- **session** — uma sessão no terminal do projeto, que eu conduzo. É onde a implementação acontece.
- **command** — um comando meu: testes, build, lint, deploy.

Um quadro montado fica mais ou menos assim:

```
   [inbox]      [refinar]     [revisar]     [fazendo]     [conferir]    [entregar]
      │             │             │             │             │             │
  eu escrevo     planner       critic       executor      verifier      comando
    o card       · opus        · opus       · sonnet      · sonnet       meu
                  agent         agent        session       agent        command
```

Cada coluna guarda uma receita: qual agente, em qual modelo, com que parte do contexto do card e com quanto ele pode gastar. Isso resolve uma coisa que sempre me incomodou — ter vinte agentes disponíveis em todo lugar significa carregar vinte agentes em todo lugar e pagar por eles em toda chamada. Aqui a coluna diz que *esta* etapa é *aquele* agente, e só isso vai.

E o card só anda até onde eu deixo. Cada coluna pode ser manual (nada se move sozinho), perguntar antes de mover, ou mover automaticamente e disparar a próxima etapa. Não tem nada rodando escondido: se uma sequência de colunas anda sozinha, foi porque eu configurei assim.

Os agentes são os arquivos markdown que já estão na minha máquina. O devpit não vem com agente nenhum e não copia nada — ele lê os que eu já tenho.

Quando estou com vários projetos ao mesmo tempo, o **Manager** junta o quadro de todos numa tela só. A coluna "revisar" de três projetos vira uma coluna só, com os cards dos três, e eu vejo de uma vez o que está esperando por mim.

## Code Review e Testes Sem Sair Dali

Revisar o que o agente fez não deveria me obrigar a abrir outra ferramenta. Do lado do terminal fica o painel de arquivos: a árvore do projeto com busca, o que mudou com o diff de cada arquivo, e o que já foi commitado. Stage e unstage saem dali mesmo.

Nos testes, eu quis resolver uma coisa que já me pegou mais de uma vez: **exit code zero não significa que passou.** Um runner que não encontrou nenhum arquivo de teste sai com zero e não testou nada. Então cada execução no devpit responde três perguntas separadas: o que o check disse (passou, falhou, não rodou ou não deixou resultado legível), se esse resultado ainda vale pro código que está na minha frente ou se o código já mudou depois, e o que foi encontrado.

Cada chamada de agente também escreve no card quanto custou quando termina. "O agente está fazendo alguma coisa" vira uma informação de verdade.

## Notas, Desenhos e o Fim dos Diretórios Invisíveis

Aqui está a parte que resolve o problema do começo do post. Cada projeto ganha uma pasta própria dentro de `~/.devpit`, fora do repositório, e tudo que é do projeto mas não é código vai pra lá:

```
~/.devpit/projects/meu-projeto/
├── agents/          os agentes que o quadro deste projeto usa
├── conversations/   as conversas, com as imagens coladas
├── attachments/     anexos dos cards
├── data/notes/      notas em .md, que abrem no Obsidian
├── data/excalidraw/ desenhos em .excalidraw, que abrem no excalidraw.com
└── local/           layout da tela e worktrees
```

Hoje são dois extras que você liga por projeto: **Notas**, em markdown com um editor de blocos, e **Excalidraw**, pra desenhar à mão — e dá pra fixar um desenho num card. Os dois gravam arquivo comum, no formato de cada ferramenta. Nada fica preso dentro do devpit: se um dia eu parar de usar, as notas continuam sendo markdown e os desenhos continuam abrindo no Excalidraw.

E pras ideias que aparecem no meio do trabalho, a coluna "inbox" do quadro existe exatamente pra isso: um card que não executa nada até eu decidir o que fazer com ele, com as notas logo do lado.

## Algumas Escolhas Pelo Caminho

O devpit é um app desktop em **Rust com Tauri**. Ele fica aberto o dia inteiro, então consumo de memória e tempo de abertura importam. Os tipos do front são gerados a partir do Rust, pra que o contrato entre as duas pontas nunca fique fora de sincronia.

Pro terminal, eu preferi apoiar no **tmux** a escrever um daemon próprio pra segurar os processos. O tmux já entrega exatamente o que eu queria — a sessão sobrevive à janela — e é uma peça que todo mundo conhece.

E tudo é **local**. Projetos, conversas, histórico de terminal e arquivos ficam no seu computador. Existe uma conta, mas ela é opcional e hoje só te identifica.

Essa também não é a minha primeira tentativa de resolver isso. Já tinha começado outras vezes, em Go e em Rust, e dessa vez eu aproveitei o que já estava pronto — terminal, integração com git, a casca desktop — em vez de começar do zero de novo.

## Onde Ele Está Hoje

É um MVP, e ainda tem bastante coisa pra fazer. Só o Claude Code é conduzido de ponta a ponta. Linux é a plataforma principal; o macOS tem build pra Apple silicon, ainda sem assinatura, e Windows não existe porque o tmux não existe lá. O modo foco está pela metade. E tem coisas que eu mesmo quero logo, como gerenciar os artefatos direto da sidebar e do explorador de arquivos, e abrir projetos que não são repositório git.

O projeto é open source (Apache 2.0). Se o objetivo fizer sentido pra você também, testa, abre uma issue ou manda um PR.

Pra instalar no Linux ou no macOS:

```sh
curl -fsSL https://raw.githubusercontent.com/jholhewres/devpit/main/install.sh | sh
```

Precisa do `tmux` e do `claude` no PATH.

---

**Links:**

- [devpit no GitHub](https://github.com/jholhewres/devpit)
- [Site e documentação](https://devpit.jhol.dev)
- [Post: Meu Fluxo de Trabalho com IA](/blog/ai-coding-workflow-two-models-anchored)
- [Post: Anchored — Uma Memória Pra Todas as Ferramentas](/blog/anchored-cross-tool-ai-memory-mcp)

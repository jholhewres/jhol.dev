---
title: "Um Orquestrador Pra Todos os Projetos: Como Eu Uso o devpit no Dia a Dia"
date: 2026-10-08
tags: ["ai", "devpit", "ade", "claude-code", "orchestration", "workflow"]
summary: "Duas semanas depois do lançamento do devpit, o jeito que eu trabalho mudou de novo. Um Orquestrador que separa empresa e projetos pessoais, e sessões que começam do zero com um prompt completo e reportam a cada etapa. Eu discuto no chat, a CLI fica só pro desenvolvimento, e nada vai pra produção sem eu aprovar."
reading_time: 5
---

No [post de lançamento do devpit](/blog/devpit-agentic-development-environment) eu falei do problema de ter terminais demais: cinco frentes, cinco Claude Codes, uma sala de alarme. O devpit resolveu isso dentro de cada projeto. Faltava resolver por cima deles.

É isso que o **Orquestrador** faz hoje no meu dia a dia. Ele é um chat que enxerga todos os meus projetos, abre sessões neles, recebe o que elas reportam e me devolve o que precisa de decisão. Este post é sobre como eu uso.

## Empresa de Um Lado, Pessoal do Outro

Eu uso o Orquestrador com a minha conta do Claude, e a primeira coisa que ele resolveu foi separar a empresa dos projetos pessoais.

```
                    ORQUESTRADOR
          ┌──────────────┴──────────────┐
      EMPRESA                        PESSOAL
   projetos do trabalho           POCs e MVPs
          │                              │
   artefatos · contextos          artefatos · contextos
   informações úteis              informações úteis
```

Cada lado tem os seus artefatos, os seus contextos e as informações úteis de cada projeto, organizados. Não tem contexto da empresa vazando pra uma POC pessoal nem o contrário.

## Discutir no Chat, Desenvolver na CLI

A divisão que mais mudou o meu ritmo foi essa: **eu discuto no chat do Orquestrador e uso a CLI só pro desenvolvimento.**

No chat eu penso em voz alta, comparo abordagens, peço recomendação, decido o escopo. Quando chega a hora de mexer em código, o Orquestrador abre uma sessão no projeto e manda o pedido. A sessão não precisa ouvir a conversa inteira; ela precisa saber o que fazer.

## Cada Sessão Começa do Zero

Isso é o que mais faz diferença. Toda sessão nasce limpa, com um prompt completo:

- **o pedido**: o que precisa ser feito;
- **as minhas preferências**: testes visuais, rodar as suítes, validar a ideia antes de implementar, buscar recomendações, entender o contexto do projeto antes de mexer;
- **as informações necessárias**: só o que aquela tarefa precisa.

As preferências vão em todo comando, então eu não repito "roda os testes" ou "tira um screenshot" a cada sessão. E como a sessão começa do zero, o contexto não enche com o que não é útil: nada de trinta mensagens de discussão que já viraram uma decisão de uma linha.

Dá pra ter **várias sessões no mesmo repositório ou na mesma worktree**, cada uma com o seu pedido.

## Um Fluxo de Ponta a Ponta

Um exemplo genérico: um bug de mensagens não enviadas num cliente.

```
  eu ──► chat do Orquestrador
           │  "mensagens não estão saindo nesse cliente, investiga"
           ▼
         brief = pedido + preferências + contexto do projeto
           │
           ▼
         sessão no projeto  (começa do zero)
           │
           ├── reporte: reproduzi, a causa é X
           ├── reporte: correção pronta, suíte passando
           └── reporte: draft pronto pra revisão
           ▼
         Orquestrador ──► me resume o que precisa de decisão
           │
           ▼
         eu aprovo (ou não). Nada vai pra produção sem isso.
```

1. **O pedido.** Eu descrevo o problema no chat, do jeito que ele chegou.
2. **O brief.** O Orquestrador monta o prompt da sessão com o pedido, as minhas preferências e o que ele sabe do projeto.
3. **A sessão.** Ela investiga, debuga, corrige, roda os testes, do jeito que eu faria.
4. **Os reportes.** A cada etapa a sessão manda pro Orquestrador um resumo, uma informação ou um pedido pra avançar.
5. **A decisão.** O que vai pra fora sai como rascunho. Eu leio e aprovo. Commit, deploy, mensagem pra alguém: nada disso acontece sem o meu ok.

O mesmo fluxo serve pra uma feature, pra um debug ou pra começar uma POC.

## A Conversa Entre o Orquestrador e a Sessão

A sessão não trabalha isolada até o fim. No meio do caminho ela pede contexto ou informação: um detalhe do projeto, uma decisão que ficou em aberto, um dado que não estava no brief. Esses pedidos chegam no Orquestrador, e eu tenho dois jeitos de atender:

- **deixar definido antes**: o que já está combinado, o Orquestrador responde sozinho, sem me chamar;
- **explicar e ele responde**: quando é algo que só eu sei, eu explico pro Orquestrador do meu jeito e ele responde pra sessão.

Na volta, a sessão deixa o Orquestrador sempre atualizado sobre o andamento. Eu não preciso perguntar como está; o estado de cada frente já está lá quando eu olho.

## Visão Sem Perder a Decisão

O que eu ganhei com isso foi **visão**. As sessões conversam com o Orquestrador e reportam a cada passo, então eu sei em que ponto cada frente está sem abrir terminal nenhum. E como cada etapa é bem definida, eu não perco a capacidade de decidir: eu sei o que foi feito, o que falta e o que está esperando por mim.

Posso deixar decisões pré-ordenadas ("se a suíte passar, abre o draft") ou decidir no meio de outro assunto, quando o reporte chega. Hoje eu estou conseguindo controlar muito melhor e ter visão de todos os projetos e POCs ao mesmo tempo.

E é por isso que agora dá pra fazer coisas que antes eu adiava:

- **iniciar POCs e MVPs**, como uma POC de SaaS que virou MVP em um dia;
- **abrir sessões pra resolver problemas**, debugar ou criar features sem largar o que eu estava fazendo.

## Os Planos Renderam Mais

Uma consequência que eu não esperava: o meu plano 5x do Claude rendeu mais nas últimas semanas, e o da empresa também. Uso os dois separados, cada um com o seu contexto.

## Depois do Lançamento

O devpit passou de **100 votos no Product Hunt** e de **50 cadastros** pelo OAuth da aplicação. Obrigado a quem testou, votou e mandou feedback.

## O Que Vem Por Aí

Estou preparando a próxima atualização, com muitas coisas legais e úteis, sempre pensando no desenvolvedor. Sem data ainda. Quando sair, eu conto aqui.

Se você quiser testar:

```sh
curl -fsSL https://raw.githubusercontent.com/jholhewres/devpit/main/install.sh | sh
```

---

**Links:**

- [devpit no GitHub](https://github.com/jholhewres/devpit)
- [Site e documentação](https://devpit.jhol.dev)
- [Post: devpit — Um Espaço de Trabalho Por Projeto](/blog/devpit-agentic-development-environment)
- [Post: Meu Fluxo de Trabalho com IA](/blog/ai-coding-workflow-two-models-anchored)

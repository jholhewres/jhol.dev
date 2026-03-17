---
title: "Implementando Gerenciamento de Contexto Lossless no DevClaw"
date: 2026-03-16
tags: ["ai", "devclaw", "lcm", "context-management", "research"]
summary: "Como implementei a arquitetura determinística de contexto do paper LCM no DevClaw — compressão hierárquica em DAG, escalação em três níveis e continuidade de custo zero"
reading_time: 10
---

Ontem, a Voltropy PBC publicou um paper que imediatamente chamou minha atenção: **Lossless Context Management (LCM)**. Em poucas horas, já tinha começado a implementar suas ideias centrais no DevClaw. Aqui está o que o paper propõe, por que importa, e como eu adaptei.

## O Problema que o LCM Resolve

Todo agente de IA para coding bate na mesma parede: limites de contexto. Até modelos com janelas de 1M+ tokens degradam bem antes de atingir seu limite nominal — um fenômeno que o paper chama de "context rot" (apodrecimento de contexto). Quando você está no meio de uma refatoração multi-arquivo, seu agente começa a esquecer decisões tomadas 30 minutos atrás.

As abordagens convencionais todas têm trade-offs:
- **Janela deslizante**: perde contexto antigo completamente
- **RAG/embeddings**: retorna fragmentos descontextualizados, sem estrutura conversacional
- **Grep em arquivos**: requer saber exatamente o que você está procurando

O LCM toma uma abordagem fundamentalmente diferente.

## A Arquitetura Central

O LCM introduz um **sistema de memória de estado duplo**:

1. **Store Imutável** — cada mensagem, resultado de ferramenta e resposta é persistida verbatim e nunca modificada. Esta é a fonte de verdade.
2. **Contexto Ativo** — a janela efetivamente enviada ao LLM, montada a partir de mensagens recentes brutas e nós de resumo pré-computados.

O insight chave: resumos são *views materializadas* sobre o histórico imutável. São um cache derivado. Os dados originais sempre estão lá.

```
Contexto Ativo (o que o LLM vê)
├── Mensagens recentes (brutas)
├── Nó de Resumo A (visão compactada das mensagens 1-50)
│   └── Ponteiro → Mensagens originais no Store Imutável
├── Nó de Resumo B (visão compactada das mensagens 51-120)
│   └── Ponteiro → Mensagens originais no Store Imutável
└── Conversa atual (bruta)
```

## O DAG Hierárquico

A estrutura de dados do paper é um **Grafo Acíclico Direcionado** de resumos mantido em um store persistente. Conforme o contexto vai enchendo, mensagens mais antigas são compactadas em nós de resumo enquanto as originais são preservadas. Nós de resumo podem ser compactados em resumos de nível superior, criando um mapa multi-resolução de todo o histórico da sessão.

Isso é o que achei mais elegante: não é apenas compressão, é compressão *navegável*. O agente pode aprofundar de um resumo de alto nível até as mensagens originais exatas quando precisa de detalhes.

## Escalação em Três Níveis

Um problema conhecido com sumarização baseada em LLM é "falha de compactação" — o resumo acaba maior que a entrada. O LCM resolve isso com um protocolo de escalação rigoroso:

```
Nível 1: Sumarização normal (preservar detalhes)
    ↓ se saída ≥ tokens de entrada
Nível 2: Sumarização agressiva (bullet points, metade do alvo)
    ↓ se saída ≥ tokens de entrada
Nível 3: Truncamento determinístico (512 tokens, sem LLM)
    → Convergência garantida
```

O Nível 3 é a rede de segurança. Não importa o que o LLM faça, a compactação *vai* reduzir tokens. Esse é o tipo de engenharia que eu aprecio — espere o melhor, engenheiro para o pior.

## Continuidade de Custo Zero

Este foi o fator decisivo para mim. A maioria das arquiteturas recursivas de contexto adiciona overhead a *toda* interação, mesmo as curtas. O LCM introduz três regimes:

| Tamanho do Contexto | Overhead |
|---|---|
| Abaixo do threshold suave | **Nenhum** — latência pura do LLM |
| Entre suave e rígido | **Assíncrono** — compactação roda em background |
| Acima do threshold rígido | **Bloqueante** — compactação antes do próximo turno |

Para a grande maioria das interações, o sistema adiciona zero latência. A maquinaria de compactação só ativa quando realmente necessário.

## Como Adaptei o LCM para o DevClaw

O DevClaw já tinha um gerenciador de contexto simples com janela deslizante. Substituir por uma arquitetura inspirada no LCM exigiu mudanças em três áreas:

### 1. O Store Imutável

Implementei o store usando um banco de dados chave-valor embarcado (Bolt em Go), mantendo alinhamento com a filosofia de binário único do DevClaw. Cada mensagem recebe um ID monotônico, tag de role, contagem de tokens e timestamp.

```go
type Message struct {
    ID        uint64
    Role      string
    Content   string
    Tokens    int
    Timestamp time.Time
    SummaryID *uint64 // qual resumo cobre esta mensagem
}
```

### 2. O DAG de Resumos

Nós de resumo formam a estrutura DAG. Cada nó sabe quais mensagens (ou outros resumos) ele cobre, permitindo tanto travessia ascendente (zoom out) quanto expansão descendente (zoom in até os originais).

```go
type SummaryNode struct {
    ID       uint64
    Kind     string // "leaf" ou "condensed"
    Content  string
    Tokens   int
    Covers   []uint64 // IDs de mensagens ou IDs de resumos filhos
    ParentID *uint64  // resumo pai, se condensado
}
```

### 3. O Loop de Controle

A montagem do contexto segue o algoritmo do paper. A cada turno:

1. Persistir a nova mensagem no store imutável
2. Anexar ao contexto ativo
3. Se acima do threshold suave, disparar compactação assíncrona
4. Se acima do threshold rígido, bloquear e compactar o bloco mais antigo

A escalação em três níveis roda dentro do passo de compactação, garantindo convergência.

### Integração de Segurança

Uma coisa que o paper LCM não aborda — porque é um paper de pesquisa, não uma ferramenta de produção — é segurança. No DevClaw, o store imutável é criptografado em repouso usando o mesmo cofre AES-256-GCM que protege as API keys. Seu histórico de conversa, que pode conter trechos de código, decisões de arquitetura e sessões de debugging, recebe a mesma proteção de nível militar que suas credenciais.

## O que Não Implementei (Ainda)

O paper também introduz **LLM-Map** e **Agentic-Map** — primitivas de recursão em nível de operador que processam datasets em paralelo fora do contexto do modelo. São poderosas para tarefas de agregação mas menos críticas para o caso de uso primário do DevClaw de assistência interativa de coding. Estão no roadmap.

O **invariante de redução de escopo** para prevenir delegação infinita também é interessante. Quando um sub-agente dispara outro sub-agente, ele deve declarar qual trabalho está delegando e qual está mantendo. Se tentar delegar tudo, o engine rejeita a chamada. Esta é uma garantia estrutural de terminação — sem necessidade de limites arbitrários de profundidade.

## Resultados Até Agora

Testes iniciais mostram melhorias significativas em sessões longas de coding:

- Utilização de contexto é mais eficiente — o agente mantém consciência de decisões tomadas anteriormente na sessão
- Sem mais "esquecimentos" de mudanças em arquivos de 20 minutos atrás
- A escalação em três níveis nunca precisou do Nível 3 na prática, mas ele está lá como rede de segurança
- Continuidade de custo zero significa que interações curtas se sentem exatamente iguais a antes

## O Quadro Geral

O paper LCM enquadra sua contribuição como um ponto em um espectro: RLM (Recursive Language Models) dá ao modelo autonomia total sobre sua estratégia de memória, enquanto LCM fornece primitivas determinísticas gerenciadas pelo engine. A analogia com programação estruturada substituindo GOTO é pertinente.

Para o DevClaw, a abordagem centrada em arquitetura é a escolha certa. A filosofia do DevClaw é sobre confiabilidade e previsibilidade — as mesmas razões pelas quais escolhemos Go, as mesmas razões pelas quais construímos o cofre de segurança. O gerenciamento determinístico de contexto do LCM se alinha perfeitamente.

O paper está disponível em [papers.voltropy.com/LCM](https://papers.voltropy.com/LCM) e vale a leitura completa. A implementação do DevClaw é open source em [github.com/jholhewres/devclaw](https://github.com/jholhewres/devclaw).

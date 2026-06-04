---
title: "Anchored OSS: Memória de Time Pra Quem Programa com IA"
date: 2026-06-03
tags: ["ai", "anchored", "mcp", "memory", "go", "team", "self-hosted"]
summary: "Estou abrindo o código do servidor que sincroniza memória de projeto entre times de devs que programam com IA. Como ele funciona por dentro, a dor que resolve — dezenas de markdowns, tokens queimados, conhecimento que não circula — e como faz um time inteiro errar cada erro uma vez só."
reading_time: 12
---

No [post anterior](/blog/anchored-scoped-memory-curation-team-sync) eu contei que tinha começado a compartilhar memória de projeto com alguns amigos através de um servidor opcional. O que era um experimento entre amigos virou um produto de verdade — e hoje estou abrindo o código dele: o **[Anchored OSS](https://github.com/jholhewres/anchored_oss)**, o servidor self-hosted de memória de time pra quem programa com IA.

A premissa é a mesma de sempre: o [Anchored local](/blog/anchored-cross-tool-ai-memory-mcp) resolve a memória **de um dev** — um banco único, acessível de qualquer ferramenta via MCP, 100% offline. O Anchored OSS resolve a memória **de um time** — o que o agente do seu colega aprendeu ontem, o seu sabe hoje. Sem ninguém escrever markdown.

## A Dor: Times Documentando Pra IA na Mão

Todo time que adotou agentes de IA pra valer conhece esse ciclo. Começa com um `CLAUDE.md`. Depois vem o `AGENTS.md`, porque outra ferramenta lê outro arquivo. Aí o projeto cresce e nascem `docs/architecture.md`, `docs/auth-flow.md`, mais um markdown por feature explicando como ela funciona. Cada decisão nova, cada fato novo, cada "descobrimos que a lib X quebra quando Y" depende de alguém lembrar de escrever no lugar certo.

E mesmo fazendo tudo certo, três coisas continuam acontecendo:

**Você queima tokens em contexto irrelevante.** O agente recarrega arquivos inteiros a cada sessão, mesmo quando 90% daquilo não tem nada a ver com a tarefa. Você paga — em custo e em janela de contexto — por conhecimento que não está usando.

**A documentação mente.** Docs-como-contexto envelhecem mal. O markdown do fluxo de autenticação foi escrito há três meses; o fluxo mudou duas vezes. O agente lê, confia, e gera código em cima de informação errada. Documentação desatualizada é pior que nenhuma: dá confiança ao modelo na direção errada.

**O aprendizado não circula.** Aquele bug sutil que o dev A passou duas horas debugando ontem? O agente do dev B vai tropeçar nele hoje, do zero. Cada dev mantém seu próprio contexto, por ferramenta, por máquina. O conhecimento do time existe — mas não flui.

No fundo é um problema só: **estamos usando arquivos estáticos pra resolver um problema de memória.** Markdown é ótimo pra documentação humana. É um péssimo banco de conhecimento pra agentes.

## O Desenho: Local-First + Camada de Time

O Anchored OSS não substitui o cliente local — ele se soma. A divisão é deliberada e é o que torna o compartilhamento viável:

```
              MÁQUINA DO DEV (local-first)
  [Claude Code]  [Cursor]  [OpenCode]  [Gemini CLI]
        |            |          |           |
        +-------- MCP STDIO ----+-----------+
                       |
            anchored (cliente, Go)
            SQLite + FTS5 + ONNX local
            busca híbrida · KG · curadoria
                       |
                       |  HTTPS (opt-in, por repositório)
                       |  Bearer anc_live_…  ·  roteado por git origin
                       v
        +----------------------------------------+
        |     anchored_oss (servidor do time)     |
        |----------------------------------------|
        |  HTTP /v1 · auth · rate limit · CORS   |
        |  Guardrails (filtro item a item)       |
        |  Postgres (pgx)  ou  SQLite (puro Go)  |
        |  Worker de curadoria (score + embed)   |
        |  Dashboard web embutido no binário     |
        +----------------------------------------+
```

O cliente continua sendo a fonte da verdade na máquina de cada dev. O servidor guarda só o que pertence ao time: **fatos, decisões, aprendizados, planos, sumários e o knowledge graph do projeto**. Preferência pessoal, contexto de máquina, handoff de sessão — nada disso sobe. Por design, não por disciplina.

O modelo de produto é simples:

```
Organização
├── Times (membros + permissões)
├── Guardrails (regras de sync, admin-managed)
├── Audit log
└── Projetos
    ├── Memórias compartilhadas
    └── Knowledge graph
```

API keys têm escopo (`admin`, `sync`, `readonly`), todo acesso passa por times, e tudo fica registrado em auditoria. É o mínimo de governança que uma empresa precisa pra deixar agentes escreverem num banco compartilhado sem ninguém perder o sono.

## O Vínculo É o Git Origin, Não a Pasta

A decisão de design que mais simplifica o dia a dia: **a identidade de um projeto é o git origin do repositório.** Não é o nome da pasta, não é o usuário, não é a ferramenta.

Conectar um repo ao servidor é isso:

```bash
anchored remote configure --server https://memoria.suaempresa.dev --key anc_live_…
cd seu-repo
anchored remote sync
```

O cliente resolve o git origin, deriva uma `remote_key`, e o servidor mapeia pro projeto certo. O mesmo repositório clonado em dez máquinas diferentes — em `~/work/api`, em `/home/dev/projetos/api-v2`, tanto faz — cai automaticamente no mesmo projeto remoto. Dev novo no time não configura nada além da API key: clonou, conectou, o agente dele já conhece o projeto.

## Como o Sync Funciona

O caminho de uma memória, do save até chegar no colega:

```
anchored save "…"  (ou a IA salva sozinha durante a sessão)
   |
   v
[1] grava LOCAL (SQLite + embedding)        <- sempre; offline-first
   |
   |  repo conectado + auto_sync?
   v
[2] classificação de segurança NO CLIENTE
   |    syncable  ·  blocked (secret, scope pessoal)  ·  review
   |    o que é blocked NUNCA sai da máquina
   v
[3] POST /api/v1/sync/push   (Bearer anc_live_…)
   |
   v        servidor
[4] resolve o projeto pelo git origin
[5] guardrails da org, item a item  ->  accepted / rejected (com a regra)
[6] dedup por content_hash           ->  re-push nunca duplica
[7] worker de curadoria (async)      ->  score + embedding, tick de 5s
   |
   v
[8] os outros devs puxam deltas por watermark no próximo sync/pull
```

Dois detalhes que importam na prática:

**Idempotência por `content_hash`.** O hash é byte-idêntico entre versões do cliente. Sincronizar duas vezes, de duas máquinas, com clientes em versões diferentes — nunca duplica. Sync que duplica é sync que ninguém liga.

**Pull por watermark.** Cada cliente busca só o que mudou desde a última vez. O servidor não reenvia o corpus inteiro; o delta é proporcional ao que o time produziu, não ao tamanho do banco.

## Guardrails: A Parte Que Deixa Uma Empresa Ligar Isso

A primeira objeção a qualquer memória compartilhada é óbvia: *"e se vazar algo que não devia?"* Um agente vê de tudo numa sessão — token, path local, dado pessoal. Se tudo subisse pro banco do time, seria um incidente esperando pra acontecer.

A resposta é defesa em camadas — a mesma lição do [post anterior](/blog/anchored-scoped-memory-curation-team-sync), agora com a segunda camada formalizada no servidor:

**Camada 1, no cliente:** o filtro de segurança classifica tudo antes de qualquer byte sair. Secret, scope pessoal, memória operacional — bloqueados na origem. E o `anchored remote preview` mostra offline, antes de qualquer rede, o que sairia.

**Camada 2, no servidor:** cada organização tem um conjunto de **guardrails** aplicados no momento do sync, item a item. Toda org nasce com um set padrão:

| Guardrail | O que pega |
|---|---|
| Detecção de secrets | Tokens Stripe/GitHub/Slack/AWS, chaves Google, chaves PEM, URIs com credencial (`postgres://user:pass@…`) |
| Paths locais | `/home/…`, `/Users/…`, `C:\Users\…`, `~/`, `/tmp/…` — força path relativo ao repo |
| Scope pessoal | Memória de um dev só não pertence ao banco do time |
| Categorias local-only | `event` e `preference` não sobem por padrão |

E o admin estende pelo dashboard: bloquear mais categorias, keywords (case-insensitive) ou regex RE2 — codinome interno, ID de ticket, o que fizer sentido pra empresa. Cada rejeição volta pro cliente com a regra que barrou, e tudo fica no audit log.

Por que duas camadas, se o cliente já filtra? Porque **o servidor não pode confiar no cliente.** Um cliente desatualizado, um fork, um bug — a última linha de defesa tem que ser de quem é dono do dado. Filtro único sempre tem furo.

## O Dia a Dia: Como o Time Escala o Aprendizado

A mecânica acima é invisível. O que o time sente é outra coisa.

Exemplo real, do desenvolvimento do próprio Anchored OSS: o servidor compila sem CGO, e o driver SQLite puro-Go (`modernc.org/sqlite`) retorna colunas `DATETIME` como **string**, não `time.Time`. Scan direto pra `*time.Time` funciona no Postgres e dá panic em runtime no SQLite. Custou uma sessão de debug — e virou um `learning` no projeto:

```
dev A: 2h de debug --> learning salvo --> sync --> servidor do time
                                                        |
dev B abre sessão no mesmo repo                         |
   anchored_context / anchored_search  <----------------+
   "SQLite puro-Go retorna DATETIME como string;
    use scanTime/scanNullTime em todo scan de timestamp"
```

Semanas depois, qualquer dev (ou o agente dele) que tocar num arquivo de store recebe esse contexto automaticamente. Ninguém escreveu doc, ninguém avisou no Slack, ninguém repetiu o erro. **É isso que significa escalar o aprendizado: o time inteiro erra cada erro uma vez só.**

O efeito composto aparece em quatro frentes:

**Menos tokens, mais precisão.** Em vez de despejar markdowns inteiros no contexto, o agente recupera por busca — texto e semântica (KNN vetorial) por projeto — só o que é relevante pra tarefa. Sessão mais barata e resposta mais certeira, porque o modelo trabalha com fatos atuais, não com doc de três meses atrás.

**Onboarding que não depende de ninguém.** Dev novo clona o repo, conecta o Anchored, e o agente dele já conhece as decisões de arquitetura, as convenções, as pegadinhas e o histórico. O conhecimento que vivia na cabeça dos seniores está disponível desde o primeiro `git clone`.

**Conhecimento que se mantém sozinho.** Decisão, fato e aprendizado são capturados na hora em que acontecem, como subproduto do trabalho — não como tarefa de documentação que fica pra depois. A memória do projeto cresce enquanto o time trabalha.

**Consistência entre ferramentas e pessoas.** Metade do time no Claude Code, metade no Cursor — todos os agentes consultam a mesma memória. As respostas param de divergir por diferença de contexto.

## Por Dentro do Servidor

Pra quem vai hospedar, o desenho segue a mesma filosofia do cliente: o mínimo de operação possível.

**Um binário Go estático, CGO-free.** O dashboard (React + shadcn/ui) é embedado no próprio binário — sem Node em produção, sem CDN, sem processo separado. As rotas de API têm precedência; o resto cai no SPA. O servidor serve até os próprios scripts de instalação em `/install`, então deploy interno não depende nem do GitHub.

**Postgres ou SQLite, à escolha.** Time pequeno roda com SQLite num VPS de R$ 20. Empresa roda com Postgres gerenciado. A interface de store é a mesma, com as duas implementações em paridade — e foi mantê-las em paridade que rendeu o learning do `scanTime` ali em cima.

**Worker de curadoria assíncrono.** O mesmo conceito da curadoria local, agora no servidor: cada memória sincronizada recebe score de qualidade e embedding num tick de 5 segundos, fora do caminho quente do sync. O provider de embedding é plugável — local-hash, ONNX ou OpenAI — e o `-reindex` refaz o corpus inteiro quando você troca de provider.

**Dashboard pra quem administra.** Overview, projetos, devs, times, API keys, guardrails, audit e health — tudo na mesma UI servida pelo binário.

Subir o servidor do time é isso:

```bash
git clone https://github.com/jholhewres/anchored_oss && cd anchored_oss
docker compose up -d
docker compose run --rm server -bootstrap   # cria org, admin e a API key
```

E o onboarding tem dois trilhos que se encontram na API key:

```
Trilho A — admin (uma vez)             Trilho B — cada dev
--------------------------             ------------------------------
sobe o servidor                        instala o anchored local
wizard web (ou -bootstrap)             anchored init --tool claude-code
   org -> admin -> projetos            funciona LOCAL, offline
recebe anc_live_…          ----+
                               +--->   anchored remote configure --key …
                                       cd repo && anchored remote sync
```

A partir daí o dev não pensa mais nisso. O local continua funcionando offline; quando tem rede, o sync corre por trás.

## O Que Aprendi

**A unidade de compartilhamento certa é o projeto, e a identidade certa é o git origin.** Não é o usuário, não é a ferramenta, não é a pasta. O repositório é a única identidade que todas as máquinas do time já têm em comum — usar ela elimina uma classe inteira de configuração.

**Privacidade em camadas é o que destrava o time.** O filtro no cliente protege o dev; os guardrails no servidor protegem a org. Nenhum dos dois sozinho basta — o servidor não pode confiar no cliente, e o dev não pode depender do admin ter configurado a regra certa.

**Idempotência é o que torna sync confiável.** `content_hash` byte-idêntico entre versões significa que re-push, retry e máquinas múltiplas nunca duplicam. Toda a confiabilidade do sistema descansa nesse invariante.

**Dois backends de store custam paridade, mas compram adoção.** SQLite pro time de três pessoas, Postgres pra empresa. O preço é implementar cada query duas vezes — e descobrir que o driver puro-Go devolve `DATETIME` como string. O ganho é ninguém precisar de infra pra testar.

**Memória compartilhada muda a economia do aprendizado.** Localmente, o Anchored fazia *uma pessoa* parar de repetir contexto entre ferramentas. Com o servidor, o *time* para de repetir erro entre pessoas. O custo de um debug difícil é pago uma vez e amortizado pelo time inteiro — é o mesmo argumento de uma biblioteca interna, aplicado a conhecimento.

## Agradecimentos

Um agradecimento especial ao [Aron](https://github.com/aronpc), amigo e colega de trabalho que esteve junto desde o início — nas discussões de lógica, nos testes do fluxo entre máquinas, e que continua auxiliando no projeto.

---

**Links:**

- [Anchored OSS no GitHub](https://github.com/jholhewres/anchored_oss)
- [Anchored (cliente local) no GitHub](https://github.com/jholhewres/anchored)
- [Post anterior: Anchored 0.5 — Escopo, Curadoria e Memória de Time](/blog/anchored-scoped-memory-curation-team-sync)
- [Post: Anchored — Uma Memória Pra Todas as Ferramentas](/blog/anchored-cross-tool-ai-memory-mcp)
- [Model Context Protocol](https://modelcontextprotocol.io/)

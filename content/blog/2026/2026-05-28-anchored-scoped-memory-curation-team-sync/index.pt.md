---
title: "Anchored 0.5: Escopo, Curadoria e Memória de Time"
date: 2026-05-28
tags: ["ai", "anchored", "mcp", "memory", "knowledge-graph", "go"]
summary: "O que mudou desde o primeiro Anchored: separação de escopo (pessoal vs projeto vs time) no mesmo banco, curadoria automática que mantém a memória limpa sem apagar nada, sync por projeto pra compartilhar contexto com amigos, e um knowledge graph bitemporal que sabe o que ainda é verdade."
reading_time: 11
---

No [primeiro post sobre o Anchored](/blog/anchored-cross-tool-ai-memory-mcp) eu resolvi o problema de memória ilhada: um único banco, acessível de qualquer ferramenta de IA, via MCP. Funcionou. Mas usar de verdade — todo dia, em vários projetos, e agora com alguns amigos no mesmo fluxo — expôs três problemas novos que a versão inicial não resolvia.

O primeiro: **preferência pessoal e convenção de projeto não são a mesma coisa.** "Eu gosto de commits temáticos pequenos" é minha, vale em qualquer repo. "Esse projeto usa Go 1.25 e deploy via VPS" é do projeto, não me segue pra outro lugar. Jogar as duas no mesmo balde polui o contexto.

O segundo: **memória cresce e apodrece.** Depois de milhares de memórias, aparece duplicata, fragmento, fato velho que já não é verdade. Sem manutenção, a busca começa a devolver lixo.

O terceiro: **eu queria compartilhar contexto de projeto com amigos** — sem vazar minhas preferências pessoais, meus paths locais, minhas secrets.

As versões 0.5.0 a 0.5.8 são a resposta pra esses três. E o ponto que mais importa pra mim no dia a dia: **nada disso exige escrever markdown.** Eu parei de editar `CLAUDE.md` na mão. As ferramentas salvam sozinhas, e o Anchored se encarrega de classificar, pontuar e manter.

## Escopo: Tudo no Mesmo Banco, Separado por Quem Manda

A mudança conceitual da 0.5.0 foi parar de tratar memória como uma lista plana. Agora cada memória carrega metadados de ciclo de vida — `scope`, `kind`, `importance`, `confidence`, `expires_at`, `pinned` — sem complicar a API que a IA enxerga.

O `scope` é o que separa o pessoal do projeto. Toda **preferência** nasce com um de três escopos:

| Scope | Pra quê | Exemplo |
|---|---|---|
| `user` | Preferência pessoal, vale em qualquer projeto | "Commits temáticos pequenos, sem co-author" |
| `project` | Convenção daquele repo | "Single binary, sem Postgres/Redis salvo se pedido" |
| `team` | Regra compartilhada com o time | "Token comparado em tempo constante" |

São a mesma tabela, o mesmo banco vetorial, a mesma busca. O escopo é só um campo — mas é o campo que decide o que entra no contexto quando eu abro um projeto, e o que pode ou não ser sincronizado pro time.

Na prática: minhas preferências pessoais me seguem pra todo lugar. As convenções do `jhol.dev` ficam no `jhol.dev`. Quando um amigo puxa o contexto do projeto, ele recebe as regras do projeto — nunca o meu "eu gosto de X".

```bash
anchored save "Commits temáticos pequenos, conventional commits, sem co-author" \
  --scope user
anchored save "Single binary Go, sem serviços externos salvo se pedido" \
  --scope project
```

Mas eu quase nunca rodo isso na mão. A IA salva sozinha, com o escopo certo, durante a conversa — é o ponto principal do "sem escrever markdown".

## Lifecycle v2: A Memória Sabe o Que É

Por baixo do escopo, cada memória ganhou um struct de metadados de verdade:

```go
type MetadataV2 struct {
    Kind        string     // decision, learning, rule, handoff...
    Scope       string     // user, project, team
    MemoryType  string     // semantic, operational, episodic
    Origin      string     // de onde veio
    Importance  float64    // peso manual/curado
    ContextTier string     // L0, L1, L2
    Pinned      bool        // nunca demovida, nunca curada
    ExpiresAt   *time.Time // TTL pra memória operacional
    Supersedes  []string   // substitui memórias antigas
    Consolidates []string  // funde memórias
    Confidence  float64
    ContentHash string     // dedup
}
```

Isso alimenta o que eu chamei de **lifecycle boost** na busca híbrida — antes do decay temporal, memórias importantes/pinadas ganham peso (decisão e learning +1.15×, handoff ativo +1.2×, semântica +1.1×), e memórias superadas apanham (-0.7×). O resultado: o que importa sobe, o que ficou obsoleto afunda — sem ninguém apagar nada.

Memória `semantic` (uma decisão, um fato) é permanente. Memória `operational` (um handoff de sessão, um snapshot de pré-compactação) tem TTL e é varrida depois. Misturar os dois era um erro silencioso da versão antiga.

## Curadoria: Manutenção Que Roda Sozinha e Não Apaga Nada

Esse é o recurso que mais mudou meu dia a dia. A partir da 0.5.7, o `anchored serve` sobe um **worker de curadoria ligado por padrão**. Ele roda em passes pequenos e incrementais (a cada 15 min, no máximo 50 memórias por vez, mais novas primeiro) e faz só uma coisa: cuida da saúde dos metadados.

```bash
anchored curation status     # versão do scorer, estado do corpus, candidatos pendentes
anchored config set curation.interval_minutes 5
anchored config set curation.max_updates_per_run 25
```

O scorer dá uma nota de qualidade a cada memória (tamanho, categoria, sinais de conteúdo, associação a projeto). Memória de baixo sinal recebe `curation_status=low_signal` e é **demovida na busca, não deletada**. Memória pinada é sempre isenta.

O ponto crucial de segurança: **a curadoria nunca reescreve conteúdo, nunca soft-deleta, nunca hard-deleta automaticamente.** Ela só ajusta `quality_score`, `importance` e `curation_status`. Quem deleta sou eu, explicitamente, com `anchored curation clean --dry-run` primeiro.

Isso é diferente do `dream`, que continua manual e é o caminho pra operações destrutivas — dedup, merge, supersede, revisão de contradição:

| Caminho | Default | O que faz | Segurança |
|---|---|---|---|
| `curation` | Ligado | Pontua e marca low-signal | Não-destrutivo |
| `dream` | Manual | Acha duplicata/contradição, propõe merge/supersede | Ação destrutiva exige apply explícito |

Na 0.5.8 eu versionei o scorer (`scorer_version` nos metadados). Quando eu mudo a fórmula, o worker re-flui a nota pelo corpus inteiro em vez de tocar só nas memórias novas — então corrigir o algoritmo conserta retroativamente o que a fórmula velha tinha marcado errado. O `anchored curation reconcile` faz isso num passe só.

> Caça-bug honesto da 0.5.8: a demoção de busca empilhava. `low_signal` (×0.03) e a faixa de qualidade sub-threshold (×0.15) estavam sendo multiplicadas (~0.0045), o que enterrava resultado legítimo. Agora são mutuamente exclusivas. E um segundo: o embedding não persistia porque o ID era atribuído a uma cópia por valor *depois* do save — o `embedAsync` rodava com ID vazio e a coluna ficava em branco. Atribuir o ID antes do save resolveu. Os dois eram do tipo que só aparece quando você usa de verdade.

## Memória de Time: Compartilhar Projeto Sem Vazar o Pessoal

Aqui entra o "alguns amigos". O local sempre foi a fonte da verdade e o caminho quente de leitura. Mas a 0.5.5 trouxe **sync por projeto** pra um servidor opcional (`anchored_oss`), e o desenho importa:

```bash
anchored remote sync-per-project --min-memories 5
```

Ele agrupa as memórias locais por `project_id`, cria **um projeto remoto por projeto local**, e empurra cada subconjunto separado. Os projetos não colapsam num balde só — `devclaw`, `anchored`, `jhol.dev` continuam distintos no servidor do time.

O que **não** sobe é a parte que me deixa tranquilo de ligar isso:

- Preferência de escopo `user` (pessoal) — bloqueada.
- Memória episódica/operational (handoff, precompact) — bloqueada.
- Path local, secret, credencial — bloqueado pelo sanitizer.
- Memória `low_signal` ou abaixo do `quality_score` mínimo (0.55) — bloqueada.

A detecção de secret foi endurecida na 0.5.5 com matchers explícitos: Stripe (`sk_live_`), GitHub (`ghp_`, `gho_`…), Slack (`xoxb-`), AWS (`AKIA[0-9A-Z]{16}`), Google (`AIza…`), connection strings com `user:pass@`, e chaves PEM. E tem `anchored remote preview` pra ver **offline**, antes de qualquer rede, o que classifica como syncable / blocked / needs-review.

O knowledge graph também sincroniza: `PushTriples` manda os triples do projeto, e o servidor é idempotente (único lógico em subject+predicate+object+project) com supersessão funcional e resolução de alias.

Resultado: eu e um amigo abrimos o mesmo projeto, em ferramentas diferentes, e os dois pegam as mesmas convenções, as mesmas decisões arquiteturais, o mesmo grafo de "X roda em Y". As minhas manias pessoais ficam comigo.

## Bitemporal: O Grafo Sabe o Que Ainda É Verdade

Isso já existia, mas é o que dá liga ao resto, então vale reforçar. Cada relação no knowledge graph tem `valid_from` e `valid_to`. Predicados funcionais como `deployed_on` só admitem um valor vigente — quando o deploy do `jhol.dev` mudou de servidor, o triple antigo ganhou `valid_to` e um novo nasceu. As consultas pedem sempre `valid_to IS NULL`.

```go
if isFunctional {
    tx.ExecContext(ctx,
        "UPDATE kg_triples SET valid_to = CURRENT_TIMESTAMP "+
        "WHERE subject_id = ? AND predicate_id = ? AND valid_to IS NULL",
        subjectID, predicateID,
    )
}
```

Bitemporal não é luxo: sem `valid_to`, o grafo vira um cemitério de fatos contraditórios. Com ele, a IA sempre responde com o estado atual — e o histórico fica disponível se eu precisar saber o que era verdade mês passado.

## 10 Tools, 10 Ferramentas

A superfície MCP cresceu mas continua simples pra IA: `anchored_context`, `_search`, `_save`, `_update`, `_forget`, `_list`, `_stats`, `_session_end`, `_kg_query`, `_kg_add`. O fluxo é o mesmo de sempre — `context` no início, `search` no meio, `save` no fim — só que agora cada `save` carrega escopo e ciclo de vida sem a IA precisar pensar nisso.

E o `anchored init` agora registra em 10 ferramentas: Claude Code, Cursor, OpenCode, Gemini CLI, Antigravity, Windsurf, Cline, VS Code Copilot, Codex CLI e Devin — cada uma com seu formato de config (JSON, TOML, a chave `servers` do VS Code).

## O Que Aprendi

**Escopo é o que torna memória compartilhável.** Sem separar pessoal de projeto, ou você não compartilha nada (e perde o ganho de time) ou vaza tudo (e perde a privacidade). O `scope` num único campo resolve os dois sem dois bancos.

**Manutenção tem que ser não-destrutiva por padrão.** Um worker que apaga sozinho é um worker que eventualmente apaga o que você precisava. Demover na busca em vez de deletar dá todo o ganho sem o risco — e deixa o delete como decisão explícita.

**Versionar o scorer compensa.** Mudar a fórmula de qualidade sem re-fluir o corpus conserta só o futuro e deixa o passado errado. `scorer_version` + reconcile faz a correção valer retroativamente.

**Privacidade no sync precisa ser camada, não confiança.** Filtro de escopo + sanitizer de secret + threshold de qualidade + preview offline. Defesa em profundidade, porque um único filtro sempre tem um furo.

**Parar de escrever markdown foi o objetivo certo.** O valor não é o banco — é que eu não administro mais a memória. As ferramentas salvam, a curadoria mantém, o sync compartilha. Eu só trabalho.

---

**Links:**

- [Anchored no GitHub](https://github.com/jholhewres/anchored)
- [Post anterior: Anchored — Uma Memória Pra Todas as Ferramentas](/blog/anchored-cross-tool-ai-memory-mcp)
- [Post: MemPalace, TurboQuant e KG Bitemporal](/blog/devclaw-mempalace-turboquant-kg-memory)
- [Post: Busca Semântica Sem API Key — ONNX + WordPiece em Go Puro](/blog/devclaw-semantic-search-onnx-wordpiece-go)
- [Model Context Protocol](https://modelcontextprotocol.io/)

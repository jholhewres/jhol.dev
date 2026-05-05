---
title: "Anchored: Uma Memória Pra Todas as Ferramentas"
date: 2026-05-04
tags: ["ai", "anchored", "mcp", "memory", "embeddings", "knowledge-graph"]
summary: "Como criei um servidor MCP em Go que centraliza a memória de todas as ferramentas de IA — deploy em qualquer projeto, preferências, decisões arquiteturais — acessível de qualquer tool, com busca híbrida, embeddings locais e knowledge graph."
reading_time: 12
---

As memórias dos meus projetos estavam todas no Claude Code. Preferências, decisões arquiteturais, detalhes de deploy, infra — tudo acumulado ao longo de meses de uso. Quando eu tentava acessar essas mesmas informações pelo OpenCode ou Cursor, conseguia só um fragmento aqui, outro ali. O grosso ficava preso no ecossistema do Claude.

Eu tinha duas opções: explicar tudo de novo toda vez que trocava de ferramenta, ou sempre voltar pro Claude Code pra recuperar contexto. Ficar dependente de uma ferramenta não é opção pra mim. Hoje eu estou me permitindo testar modelos como Qwen 3.6 Plus e Kimi K2.6, e junto com GLM 5.1, os resultados são satisfatórios. O que percebi é que não é só a LLM que importa — é o contexto e a memória que fazem ela trabalhar melhor. Uma LLM barata com contexto rico bate uma LLM cara sem contexto.

Foi aí que trouxe os conceitos que aprendi e implementei no [DevClaw](/blog/devclaw-claude-code-reverse-engineering) — [MemPalace, Knowledge Graph bitemporal](/blog/devclaw-mempalace-turboquant-kg-memory), [ONNX local, tokenização WordPiece](/blog/devclaw-semantic-search-onnx-wordpiece-go) — e empacotei tudo num MCP chamado Anchored. Na primeira execução, ele importa todas as memórias das outras ferramentas, e todos os projetos ficam acessíveis de onde você estiver usando esse MCP, localmente. Vetorização, quantização, sanitização, dream system, busca bitemporal — tudo que foi possível encapsular num único binário Go que vira uma aplicação inteira.

## O Problema: Memória Ilhada

Meu fluxo diário usa várias ferramentas de IA pra coding:

- **Claude Code** (Opus) — desenvolvimento principal, refactoring pesado
- **OpenCode** (GLM + GPT) — exploração, tarefas paralelas, plugins
- **Cursor** — edição em massa, planejamento visual
- E mais: Qwen 3.6 Plus, Kimi K2.6 — testando modelos emergentes

Cada uma tem seu próprio sistema de memória:

| Tool | Formato | Localização |
|---|---|---|
| Claude Code | `CLAUDE.md`, `memory/*.md`, JSONL de sessões | `~/.claude/projects/` |
| OpenCode | SQLite (sessions, messages, todos) | `.opencode/` |
| Cursor | `.mdc` rules files | `.cursor/rules/` |
| DevClaw | SQLite (chunks, memory) | `data/memory.db` |

Quatro silos. Quatro formatos. Zero comunicação entre eles.

O pior: quando eu aprendia algo num projeto — detalhes de deploy, infra, preferências — essa informação ficava presa na ferramenta onde foi descoberta. Trocar de tool era voltar à estaca zero.

## Um Servidor MCP Como Infraestrutura

Em vez de criar "mais um plugin", criei um **servidor MCP** que funciona como infraestrutura compartilhada. Todas as ferramentas falam com o mesmo processo, acessam o mesmo banco, compartilham o mesmo conhecimento. O modelo que roda embaixo tanto faz — Anchored funciona com qualquer LLM.

```
[Claude Code]  [Cursor]  [OpenCode]  [DevClaw]
      |            |          |          |
      +------ MCP STDIO ------+----------+
                     |
            Anchored (single binary)
                     |
         +-----------+-----------+
         v           v           v
    SQLite FTS5   ONNX      Knowledge
    + Vectors    Embed      Graph
         |
    ~/.anchored/data/anchored.db
```

Um único binário Go. Sem daemon. Sem API keys. Todas as ferramentas acessam o mesmo knowledge base.

## Memory Stack: Identidade, Projeto e Sob Demanda

A primeira decisão de design: não jogar toda a memória no contexto de uma vez. O orçamento de tokens é precioso. O Anchored usa um stack em três camadas com budget enforcement:

**L0 — Identity (~100 tokens):** Quem é o usuário. Arquivo `~/.anchored/identity.md` com preferências globais, stack, estilo de comunicação. Sempre carregado.

**L1 — Project Essentials (~400 tokens):** Fatos essenciais do projeto atual — deploy, infra, convensões, dependências. Cache com TTL de 6 horas. Se o projeto não mudou, não precisa recarregar.

**L2 — On-Demand Retrieval (~400 tokens):** Busca sob demanda quando a conversa precisa de contexto específico. Entity detection + hybrid search.

Total: ~900 tokens. Menos de 1% de um contexto de 200K, mas suficiente pra dar continuidade entre sessões e ferramentas.

O impacto prático: abro qualquer projeto em qualquer ferramenta, e o Anchored já sabe que o deploy é via `sshpass`, que as secrets vêm do GCP, e que o admin token é necessário pro endpoint `/api/admin/stats`. Não preciso repetir nada.

## Busca Híbrida: Por Que Vector Só Não Serve

A busca no Anchored combina dois métodos complementares via RRF (Reciprocal Rank Fusion):

**Vector search** captura similaridade semântica. "Como faço deploy" encontra uma memória sobre "deploy manual via SCP pro servidor de produção". O significado é o mesmo, as palavras são diferentes.

**BM25 (FTS5)** captura matches exatos e parciais. O hostname do servidor encontra a memória com o endereço exato. "pm2 restart" encontra o comando de deploy. Termos técnicos, nomes de servidores, paths — coisas que embeddings diluem.

O fusion usa RRF com pesos 70/30 (vector/BM25):

```go
func (h *HybridSearcher) rrfFuse(vecResults, bm25Results []SearchResult,
    vectorWeight, bm25Weight float64) []SearchResult {
    
    scoreMap := make(map[string]*scored)
    
    merge := func(results []SearchResult, weight float64) {
        for i, r := range results {
            key := r.Memory.ID
            if existing, ok := scoreMap[key]; ok {
                existing.score += weight * (1.0 / float64(i+1))
            } else {
                scoreMap[key] = &scored{
                    memory: r.Memory,
                    score:  weight * (1.0 / float64(i+1)),
                }
            }
        }
    }
    
    merge(vecResults, vectorWeight)   // 0.7
    merge(bm25Results, bm25Weight)    // 0.3
    // ... filter, sort, return
}
```

Além do RRF, três refinamentos:

1. **Temporal decay** — memórias recentes pontuam mais. Half-life de 30 dias (`e^(-λ * age)`)
2. **MMR diversification** — evita resultados repetitivos usando Jaccard similarity entre tokens
3. **Project boost** — memórias do projeto atual ganham 1.3x, globais 1.1x

Na prática, quando eu pergunto "como deployo o anchored?", o sistema prioriza memórias recentes do projeto `anchored`, funde resultados semânticos ("deploy", "release", "publish") com exatos ("goreleaser", "Makefile"), e diversifica pra não me dar 5 memórias dizendo a mesma coisa.

## Embeddings Locais: ONNX Sem API Key

Essa parte eu já [cubri em detalhes no post anterior](/blog/devclaw-semantic-search-onnx-wordpiece-go) sobre ONNX + WordPiece em Go puro. O Anchored usa o mesmo conceito, mas com um modelo diferente: `paraphrase-multilingual-MiniLM-L12-v2` (384 dimensões, suporte a 50+ línguas).

A escolha foi deliberada. Meus projetos usam PT-BR e EN. O `all-MiniLM-L6-v2` do DevClaw era inglês-only — buscas em português perdiam nuance semântico. O modelo multilingual mantém paridade de qualidade entre PT e EN.

```go
type ONNXEmbedder struct {
    session   *onnxruntime.AdvancedSession
    tokenizer *WordPieceTokenizer
    modelDir  string
    modelDim  int       // 384
    maxSeqLen int       // 128
}
```

Tudo roda localmente via ONNX Runtime. Sem Python, sem serviços externos, sem chaves de API. O binário auto-baixa o runtime (21MB), o modelo (~470MB) e o vocabulário no primeiro uso, com verificação SHA-256.

A quantização uint8 reduz o armazenamento em 4x com correlação >= 0.98 vs float32. Pra um knowledge base que cresce com cada sessão, isso faz diferença.

## Knowledge Graph: Relações Sem LLM

O knowledge graph armazena relações estruturadas entre entidades — "projeto X usa tecnologia Y", "deploy de Z roda em servidor W". Sem chamar LLM pra extrair. Pattern-based, direto no SQL.

```go
type Triple struct {
    ID         string     `json:"id"`
    Subject    string     `json:"subject"`
    Predicate  string     `json:"predicate"`
    Object     string     `json:"object"`
    Confidence float64    `json:"confidence"`
    ProjectID  *string    `json:"project_id,omitempty"`
    ValidFrom  time.Time  `json:"valid_from"`
    ValidTo    *time.Time `json:"valid_to,omitempty"`
}
```

Dois conceitos importantes:

**Bitemporal:** Cada triple tem `valid_from` e `valid_to`. Se o deploy do jhol.dev mudou de Fly.io pra VPS, o triple antigo ganha `valid_to` e um novo é criado. Consultas sempre pedem `valid_to IS NULL` — só o estado atual.

**Functional predicates:** Predicados como `deployed_on` são funcionais — só pode haver um valor vigente. Ao adicionar que um projeto está `deployed_on` num servidor, o sistema automaticamente invalida o triple anterior. Um UPDATE em vez de INSERT:

```go
if isFunctional {
    tx.ExecContext(ctx,
        "UPDATE kg_triples SET valid_to = CURRENT_TIMESTAMP "+
        "WHERE subject_id = ? AND predicate_id = ? AND valid_to IS NULL",
        subjectID, predicateID,
    )
}
```

Alias resolution permite buscar por nomes variantes. "jhol.dev" e "jholdev" resolvem pra mesma entidade. A query faz JOIN com `kg_entity_aliases` automaticamente.

Na prática: se eu pergunto "onde roda o projeto X?", o Anchored responde com o servidor correto — não porque alguém ensinou, mas porque a ferramenta de IA salvou essa relação durante uma sessão anterior, e o graph preservou.

## Sanitização: Segurança Sem LLM

Memória persistente armazena tudo que passa por ela. Incluindo acidentes — um debug que vazou uma API key, uma connection string num log, um token no output de um comando.

O Anchored sanitiza antes de persistir com um conjunto de regexes:

```go
defs := []ruleDef{
    // API keys
    {`(?i)(api[_-]?key|apikey)\s*[:=]\s*['"]?[a-zA-Z0-9_\-./+=]{20,}['"]?`,
     `$1=[REDACTED]`},
    // JWTs
    {`eyJ[a-zA-Z0-9_-]*\.eyJ[a-zA-Z0-9_-]*\.[a-zA-Z0-9_-]*`,
     `[REDACTED]`},
    // AWS keys
    {`(?i)AKIA[0-9A-Z]{16}`,
     `[REDACTED]`},
    // GitHub tokens
    {`(?i)gh[pouscr]_[a-zA-Z0-9]{36}`,
     `[REDACTED]`},
    // Connection strings
    {`(?i)((?:mongodb|postgres|mysql|redis))://[^@\s]*:[^@\s]+@`,
     `$1://[REDACTED]@`},
    // Private keys
    {`-----BEGIN\s+(RSA\s+|EC\s+|OPENSSH\s+)?PRIVATE\s+KEY-----[\s\S]*?-----END\s+`,
     `[REDACTED]`},
}
```

Por que regex e não LLM? Velocidade e determinismo. Regex roda em microssegundos. LLM pode falhar, alucinar, ou ser bypassado com prompt injection. Pra segurança, determinístico sempre vence.

O sanitizer roda no pipeline de save — antes da categorização, antes do embedding, antes de qualquer coisa tocar o SQLite. Se um `AWS_ACCESS_KEY` aparecer no conteúdo, vira `api_key=[REDACTED]` antes de chegar no banco.

## Import Multi-Source: Unificando o Passado

Criar a memória centralizada é meio caminho. O outro meio é importar o que já existe. O Anchored suporta importação de quatro fontes:

| Fonte | Formato | O que importa |
|---|---|---|
| Claude Code | JSONL de sessões | Decisões, preferências, fatos extraídos de conversas |
| OpenCode | SQLite | Sessions, mensagens, todos |
| Cursor | `.mdc` rules | Regras por projeto |
| DevClaw | SQLite (chunks) | Toda memória existente |

O pipeline é compartilhado: detecta formato → parse → sanitiza → categoriza → salva no SQLite. Dedup por hash de conteúdo (primeiros 200 chars) — re-importar não duplica.

```bash
# Import de todas as fontes detectadas
anchored import all

# Import específico
anchored import claude-code
anchored import opencode
```

Depois do import, qualquer ferramenta já tem acesso a anos de contexto acumulado — sem precisar reaprender nada.

## Single Binary, Zero Dependencies

O Anchored é um único binário Go. Sem Node.js, sem Python, sem npm, sem daemon.

```
~/.anchored/
├── data/
│   ├── anchored.db        # SQLite (FTS5 + vectors + KG)
│   └── onnx/              # embedding model (~470MB)
└── config.yaml
```

Instalação:

```bash
curl -fsSL https://raw.githubusercontent.com/jholhewres/anchored/main/install/install.sh | bash
```

Primeira execução auto-baixa o modelo ONNX. Depois é só adicionar como MCP server:

```json
{
  "mcpServers": {
    "anchored": {
      "command": "anchored"
    }
  }
}
```

Mesma config funciona em Claude Code, Cursor e OpenCode. O binário roda sob demanda via STDIO — não há processo de background consumindo RAM. Quando a ferramenta fecha, o Anchored fecha junto.

SQLite com WAL mode permite concorrência — múltiplas ferramentas podem ler simultaneamente sem lock.

## MCP Tools: O Que a IA Vê

O Anchored expõe 8 tools via MCP:

| Tool | O que faz |
|---|---|
| `anchored_context` | Carrega L0+L1+L2 no início da conversa |
| `anchored_search` | Busca híbrida (vector + BM25) |
| `anchored_save` | Persiste fato/decisão/preferência |
| `anchored_list` | Lista memórias por categoria/projeto |
| `anchored_forget` | Remove uma memória |
| `anchored_update` | Atualiza memória in-place |
| `kg_query` | Consulta o knowledge graph |
| `kg_add` | Adiciona relação ao knowledge graph |

O fluxo típico: ferramenta chama `anchored_context` no início da sessão → recebe identidade + projeto essentials → durante a conversa, chama `anchored_search` pra contexto adicional → ao final, chama `anchored_save` pra persistir aprendizados.

Nenhuma configuração por ferramenta. O Anchored detecta o projeto via CWD → `git rev-parse --show-toplevel` e resolve automaticamente.

## O Que Aprendi

**Centralizar > Distribuir.** Quatro silos de memória são piores que um knowledge base centralizado. A informação que fica presa numa ferramenta é informação perdida pra todas as outras.

**Híbrido vence.** Vector search sozinho perde termos exatos. BM25 sozinho perde semântica. RRF fusion com pesos calibrados cobre ambos. Não é mais complexo — é um `merge()` com pesos.

**Segurança precisa ser determinística.** Sanitização via LLM é arriscada e lenta. Regex cobre 95% dos padrões de secrets em microssegundos, sem risco de prompt injection.

**Bitemporal não é luxo.** Sem `valid_to`, o knowledge graph acumula lixo histórico. Com ele, consultas são sempre sobre o estado atual, e o histórico fica disponível quando necessário.

**Budget enforcement é essencial.** Jogar 5K tokens de contexto "só pra ter certeza" desperdiça janela. 900 tokens bem selecionados (identity + essentials + on-demand) entregam mais valor que 5K de dump não filtrado.

**Single binary é o deploy ideal pra tools de developer.** Sem Docker, sem config complexa, sem daemon. `curl | bash` e funciona. Isso é praticidade que nenhuma arquitetura distribuída bate.

---

**Links:**

- [Anchored no GitHub](https://github.com/jholhewres/anchored)
- [Post anterior: Busca Semântica Sem API Key — ONNX + WordPiece em Go Puro](/blog/devclaw-semantic-search-onnx-wordpiece-go)
- [Post: MemPalace, TurboQuant e KG Bitemporal — A Memória de Longo Prazo do DevClaw](/blog/devclaw-mempalace-turboquant-kg-memory)
- [Post: O que Aprendi Fazendo Engenharia Reversa do Claude Code](/blog/devclaw-claude-code-reverse-engineering)
- [Model Context Protocol](https://modelcontextprotocol.io/)

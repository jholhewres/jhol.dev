---
title: "MemPalace, TurboQuant e KG Bitemporal: A Memória de Longo Prazo do DevClaw"
date: 2026-04-13
tags: ["ai", "devclaw", "memory", "security", "knowledge-graph", "quantization"]
summary: "Como construí memória de longo prazo pro DevClaw usando a arquitetura MemPalace com wings/rooms, quantização TurboQuant uint8, Knowledge Graph bitemporal e proteção de credenciais em 4 camadas."
reading_time: 12
---

Como dar memória de longo prazo a um AI agent que roda em Go, conversa pelo WhatsApp e precisa lembrar de tudo — sem estourar a janela de contexto?

O sistema antigo do DevClaw era plano: uma tabela SQLite FTS5, um índice vetorial, todas as memórias competindo pelos mesmos slots. Funcionava pra conversas curtas, mas degradava com o tempo — memórias sobre família se misturavam com memórias de trabalho, fatos antigos competiam com recentes, e não havia como priorizar o que importa.

Reconstruí tudo do zero: 53 commits, 91 arquivos, +20.000 linhas de código. Esse post cobre as quatro peças principais: MemPalace, TurboQuant, Knowledge Graph bitemporal e o sistema de proteção de credenciais.

## MemPalace: Wings, Rooms e Quatro Camadas

Inspirado pelo [MemPalace/ChromaDB](https://github.com/MemPalace/mempalace), implementei a metáfora do palácio de memórias mapeada diretamente no schema do banco.

### Wings e Rooms

Cada memória recebe uma **wing** (domínio: "família", "trabalho", "saúde") e um **room** (sub-tópico dentro da wing). O roteamento é feito por um `ContextRouter` com três níveis:

1. **Mapeamento explícito**: canal já mapeado pra uma wing na tabela `channel_wing_map`
2. **Heurística**: pattern matching no nome do canal (grupo Telegram "família" → wing=family), resultado persistido com cache `sync.Map` + `singleflight`
3. **Default**: wing vazia = `NULL` = comportamento legado

A restrição chave: `wing IS NULL` é cidadão de primeira classe. Todo banco existente funciona sem migração. Retrocompatibilidade é invariante rígido, aplicado por golden-file tests que garantem saída byte-idêntica ao legado.

### Quatro Camadas de Memória

Em vez de despejar tudo no prompt, o sistema compõe quatro camadas com diferentes tradeoffs de freshness e custo:

- **L0 Identity**: Quem é este usuário? Markdown com hot-reload via filesystem. Zero chamadas ao banco.
- **L1 Essential**: "Histórias essenciais" por wing, cacheadas no SQLite com TTL de 6 horas. Zero chamadas LLM.
- **L2 OnDemand**: Busca híbrida BM25 + vetorial, escopada pra wing atual com fallback cross-wing. Detecção de entidades alimenta o topic change detector.
- **L3 Legacy**: Busca plana original, intocada. Ativa quando o stack é nil.

O stack renderiza um prefixo que é prepended ao output legado. Quando todas as camadas renderizam vazio, o resultado é byte-idêntico à versão pré-feature.

## TurboQuant: Quantização Uint8

Baseado no paper [TurboQuant/QJL](https://arxiv.org/abs/2401.15728), implementei quantização assimétrica dos vetores de embedding: dados armazenados em `uint8` (1 byte/dimensão), queries mantidas em `float32` pra preservar precisão.

O truque é a refatoração algébrica do dot product que evita dequantização por elemento:

```go
// Σ (data[i]*scale + minVal) * query[i]
//   = scale * Σ data[i]*query[i] + minVal * Σ query[i]
// Dois passes multiply-accumulate, sem reconstrução float32 intermediária.
func (q *QuantizedEmbedding) CosineSimilarity(query []float32) float32 {
    dotSum, queryNorm := uint64(0), float32(0)
    for i, d := range q.Data {
        dotSum += uint64(d) * uint64(math.Float32bits(query[i])>>23&0xFF)
        queryNorm += query[i] * query[i]
    }
    raw := q.Scale*float32(dotSum) + q.MinVal*queryNorm
    return raw / float32(math.Sqrt(float64(queryNorm)))
}
```

Resultado: **redução de 4x na memória** com correlação de **0.999965** contra baseline float32 (alvo era 0.98). A serialização binária é compacta: header de 12 bytes + dims bytes por vetor.

## Knowledge Graph Bitemporal

Junto com o MemPalace, implementei um Knowledge Graph com rastreamento bitemporal — dois eixos temporais por fato:

- `valid_from/valid_until`: quando o fato é verdadeiro no mundo real
- `txn_from/txn_until`: quando a linha existe no banco

Isso permite perguntar "o que achávamos ser verdade sobre X em março?" — essencial pra fatos que mudam: cargos, relacionamentos, endereços.

Fatos são extraídos de duas formas: um extrator baseado em padrões regex (rápido, gratuito, PT-BR + EN via YAML extensível) e um extrator LLM opcional que funciona em qualquer idioma, atrás de um gate de consentimento (`LLMConsentACK: true`). O gate existe porque enviar conteúdo de memórias pra uma API externa é um fluxo de dados que operadores devem reconhecer explicitamente.

## Detecção de Mudança de Tópico — Zero API Calls

Quando o usuário muda de assunto, a camada L2 precisa buscar contexto pro novo tópico. Detecção de tópico normalmente requer uma chamada de embedding, mas resolvi com um cascade de dois estágios que reutiliza o que já foi computado:

**Estágio 1** — Entity overlap (gratuito): compara conjuntos de entidades entre turnos. Se overlap >= 0.3, mesmo tópico. Pronto.

**Estágio 2** — Cosine similarity (gratuito): a busca L2 já computou um embedding via `SearchVector`. Reutiliza via `LastQueryEmbedding()`. Abaixo de 0.65 = mudança de tópico. Zero chamadas API extras.

## Proteção de Credenciais

Durante o desenvolvimento, encontrei um problema sério: o sistema de auto-captura salvou uma senha como "fato" e depois a exibiu via WhatsApp em texto puro. Em vez de um patch pontual, construí quatro perímetros de proteção — cada um independentemente capaz de prevenir o vazamento:

**Gate de Salvamento**: `looksLikeCredential()` verifica 15 padrões regex compilados (GitHub PATs, chaves OpenAI, tokens Slack, padrões de senha PT/EN) antes de qualquer `memory_save`. A mensagem redireciona pro vault.

**Guardrail de Saída**: `OutputGuardrail` redata padrões detectados in-place na resposta do LLM. A assimetria é intencional — **bloqueia upstream, redata downstream**.

```go
if g.CredentialChecker != nil && g.CredentialChecker(text) {
    return "", ErrCredentialLeak
}
if errors.Is(err, security.ErrCredentialLeak) {
    response = RedactCredentials(response)
}
```

**Streaming**: `RedactCredentials` roda no buffer acumulado antes de cada envio.

**Auto-Captura**: verifica credenciais e pula silenciosamente com warning no log.

## Deduplicação e TTL

Cada `memory_save` computa SHA-256 e verifica contra entradas existentes. Jaccard similarity detecta quase-duplicatas acima de 0.85. A verificação é atômica — `SaveIfNotDuplicate` mantém o mutex durante todo o ciclo ler-verificar-escrever, prevenindo TOCTOU.

Memórias auto-expiram por categoria: eventos em 7 dias, resumos em 30 dias, fatos e preferências nunca.

## O Sistema Dream

O DevClaw tem um sistema "dream" — consolidação de memórias em background durante períodos ociosos. Extrai fatos pro KG, detecta contradições e compacta o arquivo de memórias via `FileStore.Compact()` com escrita atômica (temp file + rename).

Dois bugs impediam seu funcionamento: a goroutine nunca era iniciada (bug órfão — `Assistant.Start()` não chamava `dream.Start()`), e o trigger baseado em sessões nunca disparava pro WhatsApp (sessões eternas). Resolvi contando compactações como proxy de sessões.

## O Impacto

- MemPalace com wings/rooms e quatro camadas (L0-L3) com fallback byte-idêntico
- Redução de 4x na memória via TurboQuant uint8
- Knowledge Graph bitemporal com extração consent-gated
- Topic change detection com zero chamadas API extras
- 4 camadas de proteção de credenciais
- Dream system rodando pela primeira vez
- Auto-expiração por categoria e deduplicação atômica

Todo o sistema é retrocompatível. Nenhuma mudança de configuração necessária. Binários pré-compilados estão disponíveis na [página de releases](https://github.com/jholhewres/devclaw/releases) e via `install.sh`.

---

**Links:**

- [Post anterior: Como o Vazamento do Claude Code Me Ajudou a Melhorar o DevClaw](/blog/devclaw-claude-code-architecture-lessons)
- [DevClaw no GitHub](https://github.com/jholhewres/devclaw)
- [Paper TurboQuant (quantização assimétrica)](https://arxiv.org/abs/2401.15728)
- [MemPalace (inspiração para a arquitetura)](https://github.com/MemPalace/mempalace)

---
title: "MemPalace, TurboQuant and Bitemporal KG: DevClaw's Long-Term Memory"
date: 2026-04-13
tags: ["ai", "devclaw", "memory", "security", "knowledge-graph", "quantization"]
summary: "How I built long-term memory for DevClaw using the MemPalace architecture with wings/rooms, TurboQuant uint8 quantization, a bitemporal Knowledge Graph, and four layers of credential protection."
reading_time: 12
---

How do you give long-term memory to an AI agent built in Go that talks over WhatsApp and needs to remember everything — without blowing up the context window?

DevClaw's old system was flat: one SQLite FTS5 table, one vector index, all memories competing for the same slots. It worked for short conversations but degraded over time — family memories mixed with work memories, old facts competed with recent ones, and there was no way to prioritize what matters.

I rebuilt it from scratch: 53 commits, 91 files, +20,000 lines of code. This post covers the four main pieces: MemPalace, TurboQuant, a bitemporal Knowledge Graph, and the credential protection system.

## MemPalace: Wings, Rooms, and Four Layers

Inspired by [MemPalace/ChromaDB](https://github.com/MemPalace/mempalace), I implemented the memory palace metaphor mapped directly onto the database schema.

### Wings and Rooms

Each memory gets a **wing** (a named domain like "family", "work", "health") and a **room** (a sub-topic within the wing). Routing is done by a `ContextRouter` with three tiers:

1. **Explicit mapping**: channel already mapped to a wing in the `channel_wing_map` table
2. **Heuristic**: pattern matching on the channel name (Telegram group "família" → wing=family), result persisted with `sync.Map` cache + `singleflight`
3. **Default**: empty wing = `NULL` = legacy behavior

The key constraint: `wing IS NULL` is a first-class citizen. Every existing database works without migration. Backward compatibility is a hard invariant enforced by golden-file tests that guarantee byte-identical output with the legacy path.

### Four Memory Layers

Instead of dumping all relevant memories into the prompt, the system composes four layers with different freshness/cost tradeoffs:

- **L0 Identity**: Who is this user? Loaded from a markdown file with filesystem hot-reload. Zero DB calls.
- **L1 Essential**: Per-wing "essential stories" cached in SQLite with a 6-hour TTL. Zero LLM calls.
- **L2 OnDemand**: Hybrid BM25 + vector search, scoped to the current wing first, with cross-wing fallback. Entity detection feeds the topic change detector.
- **L3 Legacy**: The original flat search, untouched. Active when the stack is nil.

The stack renders a prefix that is prepended to the legacy layer output. When all layers render empty, the result is byte-identical to the pre-feature version.

## TurboQuant: Uint8 Quantization

Based on the [TurboQuant/QJL paper](https://arxiv.org/abs/2401.15728), I implemented asymmetric estimation of embedding vectors: stored data in `uint8` (1 byte/dimension), query vectors kept as float32 for precision.

The trick is an algebraic refactoring of the dot product that avoids per-element dequantization:

```go
// Σ (data[i]*scale + minVal) * query[i]
//   = scale * Σ data[i]*query[i] + minVal * Σ query[i]
// Two multiply-accumulate passes, no intermediate float32 reconstruction.
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

Result: **4x memory reduction** with a correlation of **0.999965** against float32 baseline (target was 0.98). Binary serialization is compact: 12-byte header + dims bytes per vector.

## Bitemporal Knowledge Graph

Alongside the MemPalace, I implemented a Knowledge Graph with bitemporal tracking — two time axes per fact:

- `valid_from/valid_until`: when the fact is true in the real world
- `txn_from/txn_until`: when the row exists in the database

This lets you ask "what did we think was true about X in March?" — essential for facts that change: job titles, relationships, addresses.

Facts are extracted two ways: a regex pattern-based extractor (fast, free, PT-BR + EN via extensible YAML) and an optional LLM extractor that works in any language, behind a consent gate (`LLMConsentACK: true`). The gate exists because sending memory contents to an external API is a data flow that operators must explicitly acknowledge.

## Topic Change Detection — Zero API Calls

When the user switches topics, the L2 layer needs to fetch context for the new topic. Topic detection normally requires an embedding API call, but I solved it with a two-stage cascade that reuses what's already been computed:

**Stage 1** — Entity overlap (free): compare entity sets between turns. If overlap >= 0.3, same topic. Done.

**Stage 2** — Cosine similarity (free): the L2 search already computed an embedding via `SearchVector`. Reuse it via `LastQueryEmbedding()`. Below 0.65 = topic change. Zero extra API calls.

## Credential Protection

During development, I found a serious issue: the auto-capture system saved a password as a "fact" and later displayed it via WhatsApp in plaintext. Instead of a point fix, I built four protection perimeters — each independently capable of preventing the leak:

**Save Gate**: `looksLikeCredential()` checks 15 compiled regex patterns (GitHub PATs, OpenAI keys, Slack tokens, password patterns in PT/EN) before any `memory_save`. The error message redirects to the vault tool.

**Output Guardrail**: `OutputGuardrail` redacts detected patterns in-place in the LLM response. The asymmetry is intentional — **block upstream, redact downstream**.

```go
if g.CredentialChecker != nil && g.CredentialChecker(text) {
    return "", ErrCredentialLeak
}
if errors.Is(err, security.ErrCredentialLeak) {
    response = RedactCredentials(response)
}
```

**Streaming**: `RedactCredentials` runs on the accumulated buffer before each send.

**Auto-Capture**: checks for credentials and skips silently with a warning log.

## Deduplication and TTL

Every `memory_save` computes a SHA-256 hash and checks against existing entries. Jaccard similarity catches near-duplicates above 0.85. The check is atomic — `SaveIfNotDuplicate` holds the mutex for the entire read-check-write cycle, preventing TOCTOU races.

Memories auto-expire by category: events in 7 days, summaries in 30 days, facts and preferences never.

## The Dream System

DevClaw has a "dream" system — background memory consolidation that runs during idle periods. It extracts KG facts, detects contradictions, and compacts the memory file via `FileStore.Compact()` with atomic writes (temp file + rename).

Two bugs were blocking it: the goroutine was never started (orphan bug — `Assistant.Start()` didn't call `dream.Start()`), and the session-based trigger never fired for WhatsApp (eternal sessions). I fixed it by counting compactions as session proxies.

## The Impact

- MemPalace with wings/rooms and four layers (L0-L3) with byte-identical legacy fallback
- 4x memory reduction via TurboQuant uint8
- Bitemporal Knowledge Graph with consent-gated extraction
- Topic change detection with zero extra API calls
- 4 layers of credential protection
- Dream system actually running for the first time
- Auto-expiring events and atomic deduplication

The entire system is backward compatible. No config changes required. Pre-built binaries are available on the [releases page](https://github.com/jholhewres/devclaw/releases) and via `install.sh`.

---

**Links:**

- [Previous post: How the Claude Code Leak Helped Me Improve DevClaw](/blog/devclaw-claude-code-architecture-lessons)
- [DevClaw on GitHub](https://github.com/jholhewres/devclaw)
- [TurboQuant paper (asymmetric quantization)](https://arxiv.org/abs/2401.15728)
- [MemPalace (architecture inspiration)](https://github.com/MemPalace/mempalace)

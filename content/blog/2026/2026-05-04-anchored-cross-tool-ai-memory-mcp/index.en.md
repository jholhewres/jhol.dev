---
title: "Anchored: One Memory for All AI Tools"
date: 2026-05-04
tags: ["ai", "anchored", "mcp", "memory", "embeddings", "knowledge-graph"]
summary: "How I built an MCP server in Go that centralizes memory across all AI coding tools — deploy info, preferences, architecture decisions — accessible from any tool, with hybrid search, local embeddings, and a knowledge graph."
reading_time: 12
---

All my project memories were locked inside Claude Code. Preferences, architecture decisions, deploy details, infrastructure — accumulated over months of use. When I tried to access that same information from OpenCode or Cursor, I'd get a fragment here, a scrap there. Most of it stayed trapped in Claude's ecosystem.

I had two options: explain everything from scratch every time I switched tools, or always go back to Claude Code to recover context. Being locked into a single tool isn't an option for me. Right now I'm testing models like Qwen 3.6 Plus and Kimi K2.6, and alongside GLM 5.1, the results are solid. What I realized is that it's not just the LLM that matters — it's the context and memory that make it work better. A cheap LLM with rich context beats an expensive one without it.

So I took the concepts I learned and implemented in [DevClaw](/blog/devclaw-claude-code-reverse-engineering) — [MemPalace, bitemporal Knowledge Graph](/blog/devclaw-mempalace-turboquant-kg-memory), [local ONNX, WordPiece tokenization](/blog/devclaw-semantic-search-onnx-wordpiece-go) — and packed them all into an MCP called Anchored. On first run, it imports all memories from your other tools, and every project becomes accessible from wherever you're using that MCP, locally. Vectorization, quantization, sanitization, dream system, bitemporal search — everything I could encapsulate into a single Go binary that becomes an entire application.

## The Problem: Stranded Memory

My daily workflow uses multiple AI coding tools:

- **Claude Code** (Opus) — main development, heavy refactoring
- **OpenCode** (GLM + GPT) — exploration, parallel tasks, plugins
- **Cursor** — bulk edits, visual planning
- And more: Qwen 3.6 Plus, Kimi K2.6 — testing emerging models

Each has its own memory system:

| Tool | Format | Location |
|---|---|---|
| Claude Code | `CLAUDE.md`, `memory/*.md`, JSONL sessions | `~/.claude/projects/` |
| OpenCode | SQLite (sessions, messages, todos) | `.opencode/` |
| Cursor | `.mdc` rules files | `.cursor/rules/` |
| DevClaw | SQLite (chunks, memory) | `data/memory.db` |

Four silos. Four formats. Zero communication between them.

Worse: when I learned something in a project — "jhol.dev's deploy uses gcloud secrets for SSH and admin token" — that information stayed trapped in the tool where it was discovered. Switching tools meant starting from scratch.

## An MCP Server as Infrastructure

Instead of building "yet another plugin," I built an **MCP server** that works as shared infrastructure. All tools talk to the same process, access the same database, share the same knowledge. The LLM running underneath doesn't matter — Anchored works with any model.

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

A single Go binary. No daemon. No API keys. All tools access the same knowledge base.

## Memory Stack: Identity, Project, and On-Demand

The first design decision: don't dump all memory into context at once. Token budgets are precious. Anchored uses a three-layer stack with budget enforcement:

**L0 — Identity (~100 tokens):** Who the user is. A `~/.anchored/identity.md` file with global preferences, tech stack, communication style. Always loaded.

**L1 — Project Essentials (~400 tokens):** Core facts about the current project — deploy, infra, conventions, dependencies. Cached with a 6-hour TTL. If the project hasn't changed, no need to reload.

**L2 — On-Demand Retrieval (~400 tokens):** Search on demand when the conversation needs specific context. Entity detection + hybrid search.

Total: ~900 tokens. Less than 1% of a 200K context window, but enough to provide continuity across sessions and tools.

The practical impact: I open any project in any tool, and Anchored already knows the deploy uses `sshpass`, that secrets come from GCP, and that the admin token is needed for the `/api/admin/stats` endpoint. Nothing to repeat.

## Hybrid Search: Why Vector Alone Doesn't Cut It

Anchored's search combines two complementary methods via RRF (Reciprocal Rank Fusion):

**Vector search** captures semantic similarity. "How do I deploy" finds a memory about "manual deploy via SCP to production server." Same meaning, different words.

**BM25 (FTS5)** captures exact and partial matches. The server hostname finds the memory with the exact address. "pm2 restart" finds the deploy command. Technical terms, server names, paths — things embeddings dilute.

The fusion uses RRF with 70/30 weights (vector/BM25):

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

Beyond RRF, three refinements:

1. **Temporal decay** — recent memories score higher. 30-day half-life (`e^(-λ * age)`)
2. **MMR diversification** — avoids repetitive results using Jaccard similarity between tokens
3. **Project boost** — memories from the current project get 1.3x, global ones 1.1x

In practice, when I ask "how do I deploy anchored?", the system prioritizes recent memories from the `anchored` project, fuses semantic results ("deploy", "release", "publish") with exact ones ("goreleaser", "Makefile"), and diversifies so I don't get five memories saying the same thing.

## Local Embeddings: ONNX Without API Keys

I covered this [in detail in a previous post](/blog/devclaw-semantic-search-onnx-wordpiece-go) about ONNX + WordPiece in pure Go. Anchored uses the same concept, but with a different model: `paraphrase-multilingual-MiniLM-L12-v2` (384 dimensions, 50+ language support).

The choice was deliberate. My projects use PT-BR and EN. DevClaw's `all-MiniLM-L6-v2` was English-only — Portuguese searches lost semantic nuance. The multilingual model maintains quality parity between PT and EN.

```go
type ONNXEmbedder struct {
    session   *onnxruntime.AdvancedSession
    tokenizer *WordPieceTokenizer
    modelDir  string
    modelDim  int       // 384
    maxSeqLen int       // 128
}
```

Everything runs locally via ONNX Runtime. No Python, no external services, no API keys. The binary auto-downloads the runtime (21MB), the model (~470MB), and the vocabulary on first use, with SHA-256 verification.

Uint8 quantization reduces storage by 4x with correlation >= 0.98 vs float32. For a knowledge base that grows with every session, that adds up.

## Knowledge Graph: Relations Without an LLM

The knowledge graph stores structured relations between entities — "project X uses technology Y", "deploy of Z runs on server W". No LLM calls for extraction. Pattern-based, straight in SQL.

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

Two important concepts:

**Bitemporal:** Each triple has `valid_from` and `valid_to`. If jhol.dev's deploy moved from Fly.io to a VPS, the old triple gets a `valid_to` and a new one is created. Queries always request `valid_to IS NULL` — current state only.

**Functional predicates:** Predicates like `deployed_on` are functional — only one active value can exist. When adding that a project is `deployed_on` a server, the system automatically invalidates the previous triple. An UPDATE instead of an INSERT:

```go
if isFunctional {
    tx.ExecContext(ctx,
        "UPDATE kg_triples SET valid_to = CURRENT_TIMESTAMP "+
        "WHERE subject_id = ? AND predicate_id = ? AND valid_to IS NULL",
        subjectID, predicateID,
    )
}
```

Alias resolution lets you search by variant names. "jhol.dev" and "jholdev" resolve to the same entity. The query JOINs with `kg_entity_aliases` automatically.

In practice: if I ask "where does project X run?", Anchored answers with the correct server — not because someone taught it, but because an AI tool saved that relation during a previous session, and the graph preserved it.

## Sanitization: Security Without an LLM

Persistent memory stores everything that passes through it. Including accidents — a debug session that leaked an API key, a connection string in a log, a token in a command's output.

Anchored sanitizes before persisting with a set of regexes:

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

Why regex and not an LLM? Speed and determinism. Regex runs in microseconds. An LLM can fail, hallucinate, or get bypassed via prompt injection. For security, deterministic always wins.

The sanitizer runs in the save pipeline — before categorization, before embedding, before anything touches SQLite. If an `AWS_ACCESS_KEY` shows up in the content, it becomes `api_key=[REDACTED]` before reaching the database.

## Multi-Source Import: Unifying the Past

Building centralized memory is half the battle. The other half is importing what already exists. Anchored supports importing from four sources:

| Source | Format | What it imports |
|---|---|---|
| Claude Code | JSONL sessions | Decisions, preferences, facts extracted from conversations |
| OpenCode | SQLite | Sessions, messages, todos |
| Cursor | `.mdc` rules | Per-project rules |
| DevClaw | SQLite (chunks) | All existing memory |

The pipeline is shared: detect format → parse → sanitize → categorize → save to SQLite. Dedup by content hash (first 200 chars) — re-importing doesn't duplicate.

```bash
# Import from all detected sources
anchored import all

# Specific import
anchored import claude-code
anchored import opencode
```

After import, any tool immediately has access to years of accumulated context — no relearning required.

## Single Binary, Zero Dependencies

Anchored is a single Go binary. No Node.js, no Python, no npm, no daemon.

```
~/.anchored/
├── data/
│   ├── anchored.db        # SQLite (FTS5 + vectors + KG)
│   └── onnx/              # embedding model (~470MB)
└── config.yaml
```

Install:

```bash
curl -fsSL https://raw.githubusercontent.com/jholhewres/anchored/main/install/install.sh | bash
```

First run auto-downloads the ONNX model. Then just add it as an MCP server:

```json
{
  "mcpServers": {
    "anchored": {
      "command": "anchored"
    }
  }
}
```

Same config works in Claude Code, Cursor, and OpenCode. The binary runs on demand via STDIO — no background process eating RAM. When the tool closes, Anchored closes with it.

SQLite with WAL mode handles concurrency — multiple tools can read simultaneously without locks.

## MCP Tools: What the AI Sees

Anchored exposes 8 tools via MCP:

| Tool | What it does |
|---|---|
| `anchored_context` | Loads L0+L1+L2 at conversation start |
| `anchored_search` | Hybrid search (vector + BM25) |
| `anchored_save` | Persists fact/decision/preference |
| `anchored_list` | Lists memories by category/project |
| `anchored_forget` | Removes a memory |
| `anchored_update` | Updates memory in-place |
| `kg_query` | Queries the knowledge graph |
| `kg_add` | Adds relation to the knowledge graph |

The typical flow: tool calls `anchored_context` at session start → receives identity + project essentials → during the conversation, calls `anchored_search` for additional context → at the end, calls `anchored_save` to persist learnings.

No per-tool configuration. Anchored detects the project via CWD → `git rev-parse --show-toplevel` and resolves automatically.

## What I Learned

**Centralized > Distributed.** Four memory silos are worse than one centralized knowledge base. Information trapped in one tool is information lost to all others.

**Hybrid wins.** Vector search alone loses exact terms. BM25 alone loses semantics. RRF fusion with calibrated weights covers both. It's not more complex — it's a `merge()` with weights.

**Security needs to be deterministic.** LLM-based sanitization is risky and slow. Regex covers 95% of secret patterns in microseconds, with zero risk of prompt injection.

**Bitemporal isn't a luxury.** Without `valid_to`, the knowledge graph accumulates historical garbage. With it, queries are always about current state, and history is available when needed.

**Budget enforcement is essential.** Dumping 5K tokens of context "just to be safe" wastes the window. 900 well-selected tokens (identity + essentials + on-demand) deliver more value than 5K of unfiltered dump.

**Single binary is the ideal deploy for developer tools.** No Docker, no complex config, no daemon. `curl | bash` and it works. That's a level of practicality no distributed architecture matches.

---

**Links:**

- [Anchored on GitHub](https://github.com/jholhewres/anchored)
- [Previous post: Semantic Search Without API Keys — ONNX + WordPiece in Pure Go](/blog/devclaw-semantic-search-onnx-wordpiece-go)
- [Post: MemPalace, TurboQuant, and Bitemporal KG — DevClaw's Long-Term Memory](/blog/devclaw-mempalace-turboquant-kg-memory)
- [Post: What I Learned Reverse-Engineering Claude Code](/blog/devclaw-claude-code-reverse-engineering)
- [Model Context Protocol](https://modelcontextprotocol.io/)

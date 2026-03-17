---
title: "Implementing Lossless Context Management in DevClaw"
date: 2026-03-16
tags: ["ai", "devclaw", "lcm", "context-management", "research"]
summary: "How I implemented the LCM paper's deterministic context architecture in DevClaw — hierarchical DAG compression, three-level escalation, and zero-cost continuity"
reading_time: 10
---

Yesterday, Voltropy PBC published a paper that immediately caught my attention: **Lossless Context Management (LCM)**. Within hours, I had started implementing its core ideas in DevClaw. Here's what the paper proposes, why it matters, and how I adapted it.

## The Problem LCM Solves

Every AI coding agent hits the same wall: context limits. Even models with 1M+ token windows degrade well before reaching their nominal limit — a phenomenon the paper calls "context rot." When you're deep into a multi-file refactoring session, your agent starts forgetting decisions made 30 minutes ago.

The conventional approaches all have trade-offs:
- **Sliding window**: loses old context entirely
- **RAG/embeddings**: returns decontextualized fragments, stripped of conversational structure
- **Flat file grep**: requires knowing exactly what you're looking for

LCM takes a fundamentally different approach.

## The Core Architecture

LCM introduces a **dual-state memory system**:

1. **Immutable Store** — every message, tool result, and response is persisted verbatim and never modified. This is the source of truth.
2. **Active Context** — the window actually sent to the LLM, assembled from recent raw messages and precomputed summary nodes.

The key insight: summaries are *materialized views* over the immutable history. They're a derived cache. The original data is always there.

```
Active Context (what the LLM sees)
├── Recent messages (raw)
├── Summary Node A (compressed view of messages 1-50)
│   └── Pointer → Original messages in Immutable Store
├── Summary Node B (compressed view of messages 51-120)
│   └── Pointer → Original messages in Immutable Store
└── Current conversation (raw)
```

## The Hierarchical DAG

The paper's data structure is a **Directed Acyclic Graph** of summaries maintained in a persistent store. As the context fills up, older messages are compacted into summary nodes while the originals are preserved. Summary nodes can themselves be compacted into higher-level summaries, creating a multi-resolution map of the entire session history.

This is what I found most elegant: it's not just compression, it's *navigable* compression. The agent can drill down from a high-level summary to the exact original messages when it needs detail.

## Three-Level Escalation

A known problem with LLM-based summarization is "compaction failure" — the summary ends up longer than the input. LCM solves this with a strict escalation protocol:

```
Level 1: Normal summarization (preserve details)
    ↓ if output ≥ input tokens
Level 2: Aggressive summarization (bullet points, half target)
    ↓ if output ≥ input tokens
Level 3: Deterministic truncation (512 tokens, no LLM)
    → Guaranteed convergence
```

Level 3 is the safety net. No matter what the LLM does, compaction *will* reduce tokens. This is the kind of engineering I appreciate — hope for the best, engineer for the worst.

## Zero-Cost Continuity

This was the deciding factor for me. Most recursive context architectures add overhead to *every* interaction, even short ones. LCM introduces three regimes:

| Context Size | Overhead |
|---|---|
| Below soft threshold | **None** — raw LLM latency |
| Between soft and hard | **Async** — compaction runs in background |
| Above hard threshold | **Blocking** — compaction before next turn |

For the vast majority of interactions, the system adds zero latency. The compaction machinery only activates when it's actually needed.

## How I Adapted LCM for DevClaw

DevClaw already had a simple sliding-window context manager. Replacing it with LCM-inspired architecture required changes in three areas:

### 1. The Immutable Store

I implemented the store using an embedded key-value database (Bolt in Go), keeping it aligned with DevClaw's single-binary philosophy. Every message gets a monotonic ID, role tag, token count, and timestamp.

```go
type Message struct {
    ID        uint64
    Role      string
    Content   string
    Tokens    int
    Timestamp time.Time
    SummaryID *uint64 // which summary covers this message
}
```

### 2. The Summary DAG

Summary nodes form the DAG structure. Each node knows which messages (or other summaries) it covers, enabling both upward traversal (zoom out) and downward expansion (zoom in to originals).

```go
type SummaryNode struct {
    ID       uint64
    Kind     string // "leaf" or "condensed"
    Content  string
    Tokens   int
    Covers   []uint64 // message IDs or child summary IDs
    ParentID *uint64  // parent summary, if condensed
}
```

### 3. The Control Loop

The context assembly follows the paper's algorithm. On each turn:

1. Persist the new message to the immutable store
2. Append it to the active context
3. If above the soft threshold, trigger async compaction
4. If above the hard threshold, block and compact the oldest block

The three-level escalation runs inside the compaction step, guaranteeing convergence.

### Security Integration

One thing the LCM paper doesn't address — because it's a research paper, not a production tool — is security. In DevClaw, the immutable store is encrypted at rest using the same AES-256-GCM vault that protects API keys. Your conversation history, which may contain code snippets, architecture decisions, and debugging sessions, gets the same military-grade protection as your credentials.

## What I Didn't Implement (Yet)

The paper also introduces **LLM-Map** and **Agentic-Map** — operator-level recursion primitives that process datasets in parallel outside the model's context. These are powerful for aggregation tasks but less critical for DevClaw's primary use case of interactive coding assistance. They're on the roadmap.

The **scope-reduction invariant** for preventing infinite delegation is also interesting. When a sub-agent spawns another sub-agent, it must declare what work it's delegating and what it's keeping. If it tries to delegate everything, the engine rejects the call. This is a structural guarantee of termination — no arbitrary depth limits needed.

## Results So Far

Early testing shows significant improvements on long coding sessions:

- Context utilization is more efficient — the agent maintains awareness of decisions made earlier in the session
- No more "forgetting" file changes from 20 minutes ago
- The three-level escalation has never needed Level 3 in practice, but it's there as a safety net
- Zero-cost continuity means short interactions feel exactly the same as before

## The Bigger Picture

The LCM paper frames its contribution as a point on a spectrum: RLM (Recursive Language Models) gives the model full autonomy over its memory strategy, while LCM provides deterministic primitives managed by the engine. The analogy to structured programming replacing GOTO is apt.

For DevClaw, the architecture-centric approach is the right fit. DevClaw's philosophy is about reliability and predictability — the same reasons we chose Go, the same reasons we built the security vault. LCM's deterministic context management aligns perfectly.

The paper is available at [papers.voltropy.com/LCM](https://papers.voltropy.com/LCM) and is worth reading in full. DevClaw's implementation is open source at [github.com/jholhewres/devclaw](https://github.com/jholhewres/devclaw).

---
title: "How the Claude Code Leak Helped Me Improve DevClaw"
date: 2026-03-31
tags: ["ai", "devclaw", "claude-code", "agent-architecture", "open-source"]
summary: "After Claude Code's source leaked via npm source maps, I studied the community's analysis and discovered how to solve problems that had been blocking me in DevClaw — tool concurrency, multi-strategy compaction, memory extraction, and a dream system"
reading_time: 10
---

For months I'd been working on problems I knew had solutions, but couldn't quite nail the implementation. How to run tools in parallel without race conditions? How to compact context without losing critical information? How to reliably consolidate memories across sessions?

I understood the concepts in theory. In practice, every attempt in DevClaw had rough edges — edge cases that didn't close, trade-offs I didn't know how to balance. What I needed was to see how someone with more resources had solved these same problems in production.

Then the leak happened.

## What Happened

In the early hours of March 31, 2026, security researcher [Chaofan Shou](https://x.com/shoucccc) discovered that Anthropic had accidentally included a source map file (`cli.js.map`) in version 2.1.88 of the `@anthropic-ai/claude-code` npm package. The file contained Claude Code's complete source — **1,900 TypeScript files, 512,000+ lines of code**, including 44 feature flags for unreleased capabilities.

Within hours, the code was mirrored across public GitHub repositories. Anthropic confirmed it was [a packaging error](https://venturebeat.com/technology/claude-codes-source-code-appears-to-have-leaked-heres-what-we-know), not a security breach — no customer data was exposed. But the entire agent codebase was there for anyone willing to study it.

## The Community's Analysis

What interested me wasn't the raw code — it was the work the community did on top of it. The [instructkr/claw-code](https://github.com/instructkr/claw-code) repository became the **fastest repo in history to reach 50K stars**, hitting the milestone in just 2 hours. But beyond the hype, the project did something valuable: an architectural analysis of Claude Code's agent harness, followed by a clean-room rewrite — first in Python, now in Rust.

The project's tagline says it all: **"Better Harness Tools, not merely storing the archive of leaked Claude Code."**

It was by studying this analysis that the missing pieces in DevClaw clicked into place. I didn't copy code — Claude Code is TypeScript running on Bun, DevClaw is pure Go. But the **architectural patterns** are universal, and I finally saw how to apply them properly.

## What I Improved in DevClaw

### Tool Concurrency

My problem: DevClaw executed all tools sequentially. When the agent needed to run 4 greps and a `git log` to explore a codebase, it waited for each one to finish before starting the next.

What I learned: Claude Code classifies tools as **readonly** and **mutating**. Read tools (`grep`, `ls`, `git log`) are idempotent and can safely run in parallel. Write tools (`file_write`, `git commit`) are serialized. Sounds obvious in retrospect, but the detail I was missing was **cascading abort** — when a critical tool fails, dependent ones are cancelled immediately.

```go
type ToolExecution struct {
    Tool     Tool
    Category ToolCategory // readonly | mutating
    Status   ExecStatus   // pending | running | done | aborted
    Result   chan ToolResult
    Cancel   context.CancelFunc
}
```

The result: on codebase exploration tasks, response time dropped by half.

### Multi-Strategy Compaction

My problem: DevClaw had all-or-nothing compaction. When context grew large, it summarized everything at once — and frequently lost important information in the process.

What I learned: compaction should be a **progressive pipeline** with different strategies for different situations:

1. **Collapse** — collapses verbose tool results keeping essentials
2. **Micro-compact** — compacts old blocks preserving decisions
3. **Auto-compact** — full summarization at threshold
4. **Memory-compact** — extracts durable facts before discarding

The insight I was missing: a **circuit breaker** that disables compaction after consecutive failures. My previous system could loop when the model produced summaries larger than the input. Now, after 3 failures, the circuit breaker stops and falls back to deterministic truncation.

### Memory Extraction Before Compaction

My problem: when DevClaw compacted context, valuable knowledge went with it. Architecture decisions made early in the session simply disappeared.

What I learned: the missing step is to **mine the context before compressing**. Before any compaction, an extraction step identifies:

- **Facts** about the codebase (architecture, patterns, dependencies)
- **Decisions** made during the conversation (why X instead of Y)
- **User preferences** (code style, preferred tools)

These fragments go to long-term memory. Context can be compacted freely — the extracted knowledge survives independently.

```go
type ExtractedMemory struct {
    Category string   // fact | decision | preference
    Content  string
    Source   string   // which conversation part originated this
    Tags     []string
}
```

### Dream System — Automatic Consolidation

My problem: DevClaw's memory accumulated redundant and even contradictory fragments across multiple sessions. There was no intelligent "cleanup" mechanism.

What I implemented: a system inspired by how the brain consolidates memories during sleep. It runs in the background during idle periods and does three things:

1. **Consolidates** fragmented memories into coherent representations
2. **Detects** contradictions between old and new memories
3. **Prioritizes** by access frequency and relevance

The trigger uses three gates — all must be open:

```go
func (d *DreamCycle) ShouldRun() bool {
    elapsed := time.Since(d.LastRun) > 4*time.Hour
    sessions := d.SessionCount >= 3
    available := d.mu.TryLock()
    if available {
        d.mu.Unlock()
    }
    return elapsed && sessions && available
}
```

This prevents running during active interaction and ensures there's enough material to consolidate.

## The Impact

These four improvements together significantly elevated DevClaw:

- **Assistant performance** — concurrent tool execution makes codebase exploration dramatically faster
- **Memory quality** — pre-compaction extraction + dream system = the agent maintains real awareness of previous decisions, even in long sessions
- **Resilience** — compaction circuit breaker + cascading tool abort eliminate entire classes of loops and hangs

The new release is available at [github.com/jholhewres/devclaw](https://github.com/jholhewres/devclaw).

## The Reflection

There's a legitimate debate about the ethics of studying leaked code. My position: Anthropic confirmed it was a packaging error, not a breach. The code was on a public registry (npm). And what I did wasn't copying — it was studying **architectural concepts** that the open source community had already analyzed and publicly documented.

It's the same dynamic as studying how Redis implements its event loop, or how the Go runtime manages goroutines. Production systems teach what papers and blog posts can't: the pragmatic decisions of people who already faced the same problems at scale.

Claude Code is one of the best coding agents out there. DevClaw doesn't compete with it — it solves different problems, for a different audience, in Go. But studying the concepts behind how Anthropic built their agent gave me clarity on problems I'd been trying to solve for months.

Sometimes what's missing isn't new knowledge. It's seeing how someone with more experience applied what you already knew.

---

**Links:**
- DevClaw: [github.com/jholhewres/devclaw](https://github.com/jholhewres/devclaw)
- Community analysis: [github.com/instructkr/claw-code](https://github.com/instructkr/claw-code)
- Leak coverage: [VentureBeat](https://venturebeat.com/technology/claude-codes-source-code-appears-to-have-leaked-heres-what-we-know) · [The Register](https://www.theregister.com/2026/03/31/anthropic_claude_code_source_code/) · [DEV Community](https://dev.to/gabrielanhaia/claude-codes-entire-source-code-was-just-leaked-via-npm-source-maps-heres-whats-inside-cjo)

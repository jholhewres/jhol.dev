---
title: "Anchored 0.5: Scope, Curation, and Team Memory"
date: 2026-05-28
tags: ["ai", "anchored", "mcp", "memory", "knowledge-graph", "go"]
summary: "What changed since the first Anchored: scope separation (personal vs project vs team) in one database, automatic curation that keeps memory clean without deleting anything, per-project sync to share context with friends, and a bitemporal knowledge graph that knows what is still true."
reading_time: 11
---

In the [first post about Anchored](/blog/anchored-cross-tool-ai-memory-mcp) I solved the islanded-memory problem: one database, reachable from any AI tool, over MCP. It worked. But actually using it — every day, across many projects, and now with a few friends in the same flow — surfaced three new problems the initial version didn't address.

First: **a personal preference and a project convention are not the same thing.** "I like small thematic commits" is mine, it holds in any repo. "This project runs Go 1.25 and deploys to a VPS" belongs to the project and shouldn't follow me elsewhere. Throwing both into the same bucket pollutes context.

Second: **memory grows and rots.** After thousands of memories you get duplicates, fragments, stale facts that aren't true anymore. Without maintenance, search starts returning junk.

Third: **I wanted to share project context with friends** — without leaking my personal preferences, my local paths, my secrets.

Versions 0.5.0 through 0.5.8 are the answer to those three. And the part that matters most to me day to day: **none of it requires writing markdown.** I stopped hand-editing `CLAUDE.md`. The tools save on their own, and Anchored handles classifying, scoring, and maintaining.

## Scope: One Database, Separated by Who Owns It

The conceptual shift in 0.5.0 was to stop treating memory as a flat list. Each memory now carries lifecycle metadata — `scope`, `kind`, `importance`, `confidence`, `expires_at`, `pinned` — without complicating the API the AI sees.

`scope` is what separates personal from project. Every **preference** is born with one of three scopes:

| Scope | For | Example |
|---|---|---|
| `user` | Personal preference, holds in any project | "Small thematic commits, no co-author" |
| `project` | That repo's convention | "Single binary, no Postgres/Redis unless asked" |
| `team` | Rule shared with the team | "Compare tokens in constant time" |

Same table, same vector store, same search. Scope is just a field — but it's the field that decides what enters context when I open a project, and what can or cannot be synced to the team.

In practice: my personal preferences follow me everywhere. The `jhol.dev` conventions stay in `jhol.dev`. When a friend pulls the project context, they get the project's rules — never my "I like X".

```bash
anchored save "Small thematic commits, conventional commits, no co-author" \
  --scope user
anchored save "Single Go binary, no external services unless asked" \
  --scope project
```

But I almost never run that by hand. The AI saves on its own, with the right scope, mid-conversation — that's the whole point of "no markdown".

## Lifecycle v2: Memory Knows What It Is

Under scope, each memory got a real metadata struct:

```go
type MetadataV2 struct {
    Kind        string     // decision, learning, rule, handoff...
    Scope       string     // user, project, team
    MemoryType  string     // semantic, operational, episodic
    Origin      string     // where it came from
    Importance  float64    // manual/curated weight
    ContextTier string     // L0, L1, L2
    Pinned      bool        // never demoted, never curated
    ExpiresAt   *time.Time // TTL for operational memory
    Supersedes  []string   // replaces older memories
    Consolidates []string  // merges memories
    Confidence  float64
    ContentHash string     // dedup
}
```

This feeds what I call the **lifecycle boost** in hybrid search — before temporal decay, important/pinned memories gain weight (decision and learning +1.15×, active handoff +1.2×, semantic +1.1×), and superseded memories take a hit (-0.7×). The result: what matters rises, what's obsolete sinks — without anyone deleting anything.

A `semantic` memory (a decision, a fact) is permanent. An `operational` memory (a session handoff, a pre-compaction snapshot) has a TTL and gets swept later. Mixing the two was a silent bug in the old version.

## Curation: Maintenance That Runs Itself and Deletes Nothing

This is the feature that changed my day to day the most. As of 0.5.7, `anchored serve` starts a **curation worker that's on by default.** It runs in small incremental passes (every 15 min, at most 50 memories per pass, newest first) and does exactly one thing: tend to metadata health.

```bash
anchored curation status     # scorer version, corpus state, pending candidates
anchored config set curation.interval_minutes 5
anchored config set curation.max_updates_per_run 25
```

The scorer gives each memory a quality score (length, category, content signals, project association). A low-signal memory gets `curation_status=low_signal` and is **demoted in search, not deleted**. Pinned memories are always exempt.

The crucial safety point: **curation never rewrites content, never soft-deletes, never hard-deletes automatically.** It only adjusts `quality_score`, `importance`, and `curation_status`. Deleting is something I do, explicitly, with `anchored curation clean --dry-run` first.

This is different from `dream`, which stays manual and is the path for destructive operations — dedup, merge, supersede, contradiction review:

| Path | Default | What it does | Safety |
|---|---|---|---|
| `curation` | On | Scores and marks low-signal | Non-destructive |
| `dream` | Manual | Finds duplicates/contradictions, proposes merge/supersede | Destructive actions require explicit apply |

In 0.5.8 I versioned the scorer (`scorer_version` in metadata). When I change the formula, the worker re-flows the score through the entire corpus instead of only touching new memories — so fixing the algorithm retroactively repairs what the old formula had flagged wrong. `anchored curation reconcile` does it in a single pass.

> Honest bug hunt from 0.5.8: search demotion was stacking. `low_signal` (×0.03) and the sub-threshold quality band (×0.15) were being multiplied (~0.0045), which buried legitimate hits. They're now mutually exclusive. And a second one: embeddings weren't persisting because the ID was assigned to a by-value copy *after* save — `embedAsync` ran with an empty ID and the column stayed blank. Assigning the ID before save fixed it. Both are the kind you only hit when you actually use the thing.

## Team Memory: Share a Project Without Leaking the Personal

This is where the "few friends" come in. Local has always been the source of truth and the hot read path. But 0.5.5 added **per-project sync** to an optional server (`anchored_oss`), and the design matters:

```bash
anchored remote sync-per-project --min-memories 5
```

It groups local memories by `project_id`, creates **one remote project per local project**, and pushes each subset separately. Projects don't collapse into a single bucket — `devclaw`, `anchored`, `jhol.dev` stay distinct on the team server.

What does **not** go up is the part that lets me turn this on without worry:

- `user`-scope (personal) preferences — blocked.
- Episodic/operational memory (handoff, precompact) — blocked.
- Local paths, secrets, credentials — blocked by the sanitizer.
- `low_signal` memory or below the minimum `quality_score` (0.55) — blocked.

Secret detection was hardened in 0.5.5 with explicit matchers: Stripe (`sk_live_`), GitHub (`ghp_`, `gho_`…), Slack (`xoxb-`), AWS (`AKIA[0-9A-Z]{16}`), Google (`AIza…`), connection strings with `user:pass@`, and PEM keys. And there's `anchored remote preview` to see — **offline**, before any network — what classifies as syncable / blocked / needs-review.

The knowledge graph syncs too: `PushTriples` sends the project's triples, and the server is idempotent (logical unique on subject+predicate+object+project) with functional supersession and alias resolution.

The result: a friend and I open the same project, in different tools, and both get the same conventions, the same architectural decisions, the same "X runs on Y" graph. My personal quirks stay with me.

## Bitemporal: The Graph Knows What's Still True

This already existed, but it's the glue for everything else, so it's worth restating. Each relationship in the knowledge graph has `valid_from` and `valid_to`. Functional predicates like `deployed_on` allow only one current value — when `jhol.dev`'s deploy moved servers, the old triple got a `valid_to` and a new one was born. Queries always ask for `valid_to IS NULL`.

```go
if isFunctional {
    tx.ExecContext(ctx,
        "UPDATE kg_triples SET valid_to = CURRENT_TIMESTAMP "+
        "WHERE subject_id = ? AND predicate_id = ? AND valid_to IS NULL",
        subjectID, predicateID,
    )
}
```

Bitemporal isn't a luxury: without `valid_to`, the graph becomes a graveyard of contradictory facts. With it, the AI always answers with the current state — and the history stays available if I need to know what was true last month.

## 10 Tools, 10 Tools

The MCP surface grew but stays simple for the AI: `anchored_context`, `_search`, `_save`, `_update`, `_forget`, `_list`, `_stats`, `_session_end`, `_kg_query`, `_kg_add`. The flow is the same as ever — `context` at the start, `search` in the middle, `save` at the end — except now every `save` carries scope and lifecycle without the AI having to think about it.

And `anchored init` now registers across 10 tools: Claude Code, Cursor, OpenCode, Gemini CLI, Antigravity, Windsurf, Cline, VS Code Copilot, Codex CLI, and Devin — each with its own config format (JSON, TOML, VS Code's `servers` key).

## What I Learned

**Scope is what makes memory shareable.** Without separating personal from project, you either share nothing (and lose the team gain) or leak everything (and lose privacy). `scope` in a single field solves both without two databases.

**Maintenance must be non-destructive by default.** A worker that deletes on its own is a worker that eventually deletes what you needed. Demoting in search instead of deleting gives all the benefit without the risk — and leaves deletion as an explicit decision.

**Versioning the scorer pays off.** Changing the quality formula without re-flowing the corpus only fixes the future and leaves the past wrong. `scorer_version` + reconcile makes the fix apply retroactively.

**Privacy in sync needs layers, not trust.** Scope filter + secret sanitizer + quality threshold + offline preview. Defense in depth, because a single filter always has a hole.

**Dropping markdown was the right goal.** The value isn't the database — it's that I no longer administer the memory. The tools save, curation maintains, sync shares. I just work.

---

**Links:**

- [Anchored on GitHub](https://github.com/jholhewres/anchored)
- [Previous post: Anchored — One Memory for Every Tool](/blog/anchored-cross-tool-ai-memory-mcp)
- [Post: MemPalace, TurboQuant and Bitemporal KG](/blog/devclaw-mempalace-turboquant-kg-memory)
- [Post: Semantic Search Without an API Key — ONNX + WordPiece in Pure Go](/blog/devclaw-semantic-search-onnx-wordpiece-go)
- [Model Context Protocol](https://modelcontextprotocol.io/)

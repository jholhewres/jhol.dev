---
title: "Anchored OSS: Team Memory for Devs Who Code with AI"
date: 2026-06-03
tags: ["ai", "anchored", "mcp", "memory", "go", "team", "self-hosted"]
summary: "I'm open-sourcing the server that syncs project memory across dev teams coding with AI. How it works under the hood, the pain it solves — dozens of markdown files, wasted tokens, knowledge that never circulates — and how it lets a whole team make every mistake only once."
reading_time: 12
---

In the [previous post](/blog/anchored-scoped-memory-curation-team-sync) I mentioned I had started sharing project memory with a few friends through an optional server. What began as an experiment among friends turned into a real product — and today I'm opening up its source: **[Anchored OSS](https://github.com/jholhewres/anchored_oss)**, the self-hosted team memory server for people who code with AI.

The premise stays the same: [local Anchored](/blog/anchored-cross-tool-ai-memory-mcp) solves memory **for one dev** — a single database, reachable from any tool over MCP, 100% offline. Anchored OSS solves memory **for a team** — what your colleague's agent learned yesterday, yours knows today. Without anyone writing markdown.

## The Pain: Teams Hand-Writing Docs for AI

Every team that adopted AI agents for real knows this cycle. It starts with a `CLAUDE.md`. Then comes `AGENTS.md`, because another tool reads a different file. Then the project grows and you get `docs/architecture.md`, `docs/auth-flow.md`, one more markdown per feature explaining how it works. Every new decision, every new fact, every "we found out lib X breaks when Y" depends on someone remembering to write it in the right place.

And even when you do everything right, three things keep happening:

**You burn tokens on irrelevant context.** The agent reloads entire files every session, even when 90% of it has nothing to do with the task. You pay — in cost and in context window — for knowledge you're not using.

**The documentation lies.** Docs-as-context age badly. The auth-flow markdown was written three months ago; the flow has changed twice since. The agent reads it, trusts it, and generates code on top of wrong information. Stale documentation is worse than none: it gives the model confidence in the wrong direction.

**Learning doesn't circulate.** That subtle bug dev A spent two hours debugging yesterday? Dev B's agent will trip over it today, from scratch. Each dev keeps their own context, per tool, per machine. The team's knowledge exists — but it doesn't flow.

At the core it's a single problem: **we're using static files to solve a memory problem.** Markdown is great for human documentation. It's a terrible knowledge database for agents.

## The Design: Local-First + a Team Layer

Anchored OSS doesn't replace the local client — it adds to it. The split is deliberate, and it's what makes sharing viable:

```
              DEV MACHINE (local-first)
  [Claude Code]  [Cursor]  [OpenCode]  [Gemini CLI]
        |            |          |           |
        +-------- MCP STDIO ----+-----------+
                       |
            anchored (client, Go)
            SQLite + FTS5 + local ONNX
            hybrid search · KG · curation
                       |
                       |  HTTPS (opt-in, per repository)
                       |  Bearer anc_live_…  ·  routed by git origin
                       v
        +----------------------------------------+
        |     anchored_oss (the team server)      |
        |----------------------------------------|
        |  HTTP /v1 · auth · rate limit · CORS   |
        |  Guardrails (per-item filtering)       |
        |  Postgres (pgx)  or  SQLite (pure Go)  |
        |  Curation worker (score + embed)       |
        |  Web dashboard embedded in the binary  |
        +----------------------------------------+
```

The client remains the source of truth on each dev's machine. The server stores only what belongs to the team: **facts, decisions, learnings, plans, summaries, and the project's knowledge graph**. Personal preferences, machine context, session handoffs — none of that goes up. By design, not by discipline.

The product model is simple:

```
Organization
├── Teams (members + permissions)
├── Guardrails (sync rules, admin-managed)
├── Audit log
└── Projects
    ├── Shared memories
    └── Knowledge graph
```

API keys are scoped (`admin`, `sync`, `readonly`), all access goes through teams, and everything lands in an audit log. It's the minimum governance a company needs to let agents write into a shared database without anyone losing sleep.

## The Link Is the Git Origin, Not the Folder

The design decision that simplifies day-to-day the most: **a project's identity is the repository's git origin.** Not the folder name, not the user, not the tool.

Connecting a repo to the server is this:

```bash
anchored remote configure --server https://memory.yourcompany.dev --key anc_live_…
cd your-repo
anchored remote sync
```

The client resolves the git origin, derives a `remote_key`, and the server maps it to the right project. The same repository cloned on ten different machines — in `~/work/api`, in `/home/dev/projects/api-v2`, doesn't matter — automatically lands in the same remote project. A new dev on the team configures nothing beyond the API key: clone, connect, and their agent already knows the project.

## How Sync Works

The path of a memory, from save to reaching a teammate:

```
anchored save "…"  (or the AI saves on its own during the session)
   |
   v
[1] write LOCAL (SQLite + embedding)        <- always; offline-first
   |
   |  repo connected + auto_sync?
   v
[2] safety classification ON THE CLIENT
   |    syncable  ·  blocked (secret, personal scope)  ·  review
   |    whatever is blocked NEVER leaves the machine
   v
[3] POST /api/v1/sync/push   (Bearer anc_live_…)
   |
   v        server
[4] resolve the project by git origin
[5] org guardrails, item by item  ->  accepted / rejected (with the rule)
[6] dedup by content_hash          ->  re-push never duplicates
[7] curation worker (async)        ->  score + embedding, 5s tick
   |
   v
[8] other devs pull watermark deltas on their next sync/pull
```

Two details that matter in practice:

**Idempotency via `content_hash`.** The hash is byte-identical across client versions. Syncing twice, from two machines, with clients on different versions — never duplicates. A sync that duplicates is a sync nobody turns on.

**Watermark-based pull.** Each client fetches only what changed since last time. The server doesn't resend the whole corpus; the delta is proportional to what the team produced, not to the size of the database.

## Guardrails: The Part That Lets a Company Turn This On

The first objection to any shared memory is obvious: *"what if something leaks that shouldn't?"* An agent sees everything during a session — tokens, local paths, personal data. If all of it went up to the team database, it would be an incident waiting to happen.

The answer is layered defense — the same lesson from the [previous post](/blog/anchored-scoped-memory-curation-team-sync), now with the second layer formalized on the server:

**Layer 1, on the client:** the safety filter classifies everything before a single byte leaves. Secrets, personal scope, operational memory — blocked at the source. And `anchored remote preview` shows offline, before any network, what would go out.

**Layer 2, on the server:** each organization has a set of **guardrails** applied at sync time, item by item. Every org is born with a default set:

| Guardrail | What it catches |
|---|---|
| Secret detection | Stripe/GitHub/Slack/AWS tokens, Google keys, PEM private keys, credential-bearing URIs (`postgres://user:pass@…`) |
| Local paths | `/home/…`, `/Users/…`, `C:\Users\…`, `~/`, `/tmp/…` — forces repo-relative paths |
| Personal scope | A single dev's memory doesn't belong in the team database |
| Local-only categories | `event` and `preference` don't sync by default |

Admins extend it from the dashboard: block more categories, keywords (case-insensitive), or RE2 regexes — internal codenames, ticket IDs, whatever makes sense for the company. Every rejection comes back to the client with the rule that blocked it, and everything lands in the audit log.

Why two layers, if the client already filters? Because **the server can't trust the client.** An outdated client, a fork, a bug — the last line of defense has to belong to whoever owns the data. A single filter always has a hole.

## Day to Day: How the Team Scales Its Learning

The mechanics above are invisible. What the team feels is something else.

A real example, from building Anchored OSS itself: the server compiles without CGO, and the pure-Go SQLite driver (`modernc.org/sqlite`) returns `DATETIME` columns as **strings**, not `time.Time`. Scanning straight into `*time.Time` works on Postgres and panics at runtime on SQLite. It cost a debugging session — and became a `learning` in the project:

```
dev A: 2h of debugging --> learning saved --> sync --> team server
                                                            |
dev B opens a session in the same repo                      |
   anchored_context / anchored_search  <--------------------+
   "Pure-Go SQLite returns DATETIME as string;
    use scanTime/scanNullTime on every timestamp scan"
```

Weeks later, any dev (or their agent) touching a store file gets that context automatically. Nobody wrote a doc, nobody posted on Teams, nobody repeated the mistake. **That's what scaling learning means: the whole team makes every mistake only once.**

The compound effect shows up on four fronts:

**Fewer tokens, more precision.** Instead of dumping entire markdown files into context, the agent retrieves by search — text and semantic (vector KNN) per project — only what's relevant to the task. Cheaper sessions and sharper answers, because the model works with current facts, not three-month-old docs.

**Onboarding that depends on no one.** A new dev clones the repo, connects Anchored, and their agent already knows the architectural decisions, the conventions, the gotchas, and the history. The knowledge that used to live in senior devs' heads is available from the first `git clone`.

**Knowledge that maintains itself.** Decisions, facts, and learnings are captured the moment they happen, as a by-product of the work — not as a documentation chore that's always left for later. The project's memory grows while the team works.

**Consistency across tools and people.** Half the team on Claude Code, half on Cursor — every agent queries the same memory. Answers stop diverging due to context differences.

## Inside the Server

For whoever hosts it, the design follows the same philosophy as the client: the least operations possible.

**One static Go binary, CGO-free.** The dashboard (React + shadcn/ui) is embedded in the binary itself — no Node in production, no CDN, no separate process. API routes take precedence; everything else falls through to the SPA. The server even serves its own install scripts at `/install`, so internal deployments don't depend on GitHub.

**Postgres or SQLite, your choice.** A small team runs SQLite on a $5 VPS. A company runs managed Postgres. The store interface is the same, with both implementations kept in parity — and keeping them in parity is what produced the `scanTime` learning above.

**Async curation worker.** The same concept as local curation, now on the server: every synced memory gets a quality score and an embedding on a 5-second tick, outside the sync hot path. The embedding provider is pluggable — local-hash, ONNX, or OpenAI — and `-reindex` rebuilds the whole corpus when you switch providers.

**A dashboard for whoever administers it.** Overview, projects, devs, teams, API keys, guardrails, audit, and health — all in the same UI served by the binary.

Bringing up the team server is this:

```bash
git clone https://github.com/jholhewres/anchored_oss && cd anchored_oss
docker compose up -d
docker compose run --rm server -bootstrap   # creates org, admin, and the API key
```

And onboarding has two tracks that meet at the API key:

```
Track A — admin (once)                 Track B — each dev
--------------------------             ------------------------------
bring up the server                    install local anchored
web wizard (or -bootstrap)             anchored init --tool claude-code
   org -> admin -> projects            works LOCAL, offline
receives anc_live_…        ----+
                               +--->   anchored remote configure --key …
                                       cd repo && anchored remote sync
```

From there the dev never thinks about it again. Local keeps working offline; when there's network, sync runs in the background.

## What I Learned

**The right unit of sharing is the project, and the right identity is the git origin.** Not the user, not the tool, not the folder. The repository is the only identity every machine on the team already has in common — using it eliminates an entire class of configuration.

**Layered privacy is what unlocks the team.** The client-side filter protects the dev; the server-side guardrails protect the org. Neither is enough alone — the server can't trust the client, and the dev can't depend on the admin having configured the right rule.

**Idempotency is what makes sync trustworthy.** A `content_hash` that's byte-identical across versions means re-push, retries, and multiple machines never duplicate. The system's entire reliability rests on that invariant.

**Two store backends cost parity, but buy adoption.** SQLite for the three-person team, Postgres for the company. The price is implementing every query twice — and discovering the pure-Go driver returns `DATETIME` as a string. The payoff is nobody needing infra to try it.

**Shared memory changes the economics of learning.** Locally, Anchored made *one person* stop repeating context across tools. With the server, the *team* stops repeating mistakes across people. The cost of a hard debugging session is paid once and amortized across the whole team — it's the same argument as an internal library, applied to knowledge.

## Acknowledgments

A special thanks to [Aron](https://github.com/aronpc), a friend and work colleague who's been in it since the beginning — in the logic discussions, in testing the cross-machine flow, and who keeps helping on the project.

---

**Links:**

- [Anchored OSS on GitHub](https://github.com/jholhewres/anchored_oss)
- [Anchored (local client) on GitHub](https://github.com/jholhewres/anchored)
- [Previous post: Anchored 0.5 — Scope, Curation, and Team Memory](/blog/anchored-scoped-memory-curation-team-sync)
- [Post: Anchored — One Memory for Every Tool](/blog/anchored-cross-tool-ai-memory-mcp)
- [Model Context Protocol](https://modelcontextprotocol.io/)

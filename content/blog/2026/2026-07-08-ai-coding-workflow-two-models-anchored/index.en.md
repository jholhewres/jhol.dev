---
title: "My AI Coding Workflow: Two Models, Anchored, and a Plan That Isn't /plan"
date: 2026-07-08
tags: ["ai", "workflow", "anchored", "claude-code", "glm", "productivity"]
summary: "How I code today: two models (Opus 4.8 and GLM 5.2), three Claude Codes, memory across 60+ projects in Anchored, and a four-step flow — context gathering, agent-driven planning, phase-by-phase code review, and real testing — that replaced the /plan I never liked."
image: cover.png
reading_time: 12
---

Since the start of the year I've been testing everything: models, IDEs, CLIs, off-the-shelf skills, skills I wrote myself, PRDs, templates, documentation patterns. The goal was a single one — keep dozens of projects moving at once, between work and personal projects, without losing the thread. Start a task, finish it, and be able to pick it back up later in a fresh or summarized session with the same control and the same speed.

After months of trading ideas almost daily with [Aron](https://github.com/aronpc) — his flow is a close cousin of mine —, I landed on a workflow that finally stopped getting in my way. This post is about it: the models I use, what makes it all possible (spoiler: [Anchored](/blog/anchored-cross-tool-ai-memory-mcp)), and the four-step flow that replaced the `/plan` I never liked.

## The Foundation: Two Models, Three Claude Codes

I spent the year testing the Chinese models — Qwen, Kimi, and so on. In the end, everything converged on **two**:

- **Claude Opus 4.8** — the primary model. Analysis, architecture, planning, the work that can't miss.
- **GLM 5.2** (Z.ai Coding account) — to handle the lighter loads without burning my primary tokens.

And I run this across three Claude Codes, each with its own terminal command:

```
Command     Model                Use
-------     -----                ---
claude   →  Claude Opus 4.8      work (company plan)
claudep  →  Claude Opus 4.8      personal (my own plan)
glm      →  GLM 5.2 (Z.ai)       lighter load / spare primary tokens
```

The logic is about context and cost economy, not capability. GLM 5.2, in my opinion, is at Opus's level — it loses by a few details. So any task that doesn't need the top of the ruler goes to `glm`, and I reserve Opus's primary tokens for what really matters. Two models, three terminals, one clear rule for when to use each.

Running three sessions in parallel creates an immediate problem: knowing which one finished, which is still crunching, and which has been sitting there for ten minutes waiting for you to approve something. For that I use — and recommend — **[AI Traffic Lights](https://github.com/aronpc/ai-traffic-lights)**, another project by Aron: an overlay that shows each agent session as a traffic light (🟢 done · 🟡 working · 🔴 needs you), click to jump straight to the right terminal (and tab), plus per-agent usage meters — including the GLM Coding Plan, which fits nicely with the don't-burn-primary-tokens logic.

## What Makes It All Possible: Anchored

None of this would work if every fresh session started from scratch. What holds the flow together is [Anchored](/blog/anchored-oss-team-memory-server) running day to day, with memory across **60+ projects** and multiple remote servers split by context:

```
                        ANCHORED (cross-tool memory)
        ┌───────────────────┬───────────────────┬───────────────────┐
   PERSONAL remote     EXTERNAL remote      COMPANY remote
   my own projects     external projects    shared with coworkers
        └───────────────────┴───────────────────┴───────────────────┘
                                   │
              60+ projects: stack · access · commands
              learnings · decisions · knowledge graph · history
```

The company remote is what changes the game at work: I see what my coworkers are doing and reuse the context of what they've already built. Knowledge circulates without anyone writing docs.

The best example is picking up a task I don't know. Before any planning, in a single context-gathering pass I can:

- Pull **memories of the resource** the task touches — who implemented it, when, and what got recorded.
- See related **prior tasks**, of any status.
- Know whether my task **depends on another** or **unblocks a frontend**.
- Judge whether it **has enough context to be delivered** — or whether something's missing before I even start.

All of that comes out of the remote memory, much of it from coworkers. It's the difference between starting blind and starting with the map in hand.

## How I Code Today

Plenty of people still live in the tools' `/plan`. I never liked it — I always felt something was missing. And something was: real context before the plan, and real verification after it. The flow I built has four steps:

```
   TASK  ──────────►  [1] CONTEXT GATHERING
   (Jira / personal)        remote memory (Anchored) + coworkers
                            logs · CloudWatch · Datadog
                            deps/blockers · who did it · when
                            "is there enough context to deliver?"
                                   │
                                   ▼
                         [2] PLAN  (OMC — no /plan)
                             architect · designer · frontend
                             └─ spawns Sonnet 5 for the heavy
                                code-reading work
                                   │
                                   ▼
                         [3] EXECUTION  (omc ralph)
                             phase 1 ─► code review ─► fixes
                             phase 2 ─► code review ─► fixes
                             phase N ─► code review ─► fixes
                                   │
                                   ▼
                         [4] REAL TESTING
                             endpoints · jobs · queues · workers
                             error scenarios · validations
                             ┌──── targeted fix + unit test
                             │           (loop until covered)
                             ▼
                            DELIVERY  (reviews usually come back clean)
```

### 1. Context gathering

Feature, simple task, or defect — doesn't matter: context gathering is what guarantees the task carries everything it needs. And doing it with precise information from **remote memory coming from coworkers** makes it much better.

But I don't stop at memory. I grant access to **logs**, dig into **CloudWatch**, into **Datadog**, and assemble all the context I need before thinking about a plan. If something's missing to deliver, it shows up here — not halfway through implementation.

### 2. The plan (no /plan)

Instead of `/plan`, I use the **[Oh My Claude Code (OMC)](https://github.com/yeachan-heo/oh-my-claudecode)** plugin. It gives me the right agents to write the plan: depending on complexity, an **architect**, a **designer**, a **frontend** step in. And when one of them needs more information from the code, it spawns smaller models itself — **Claude Sonnet 5** — to do the "heavy" reading work.

Note the split: Sonnet does the grunt work of combing through code, but **analysis and planning stay with Opus**. Sonnet for planning doesn't cut it — it loses the thread on trade-offs.

### 3. Execution with Ralph + phase-by-phase code review

With the plan in hand — and I tend to write plans with more than two execution phases — I move to execution with **Ralph**, via OMC (`omc ralph`). It carries everything through to the end.

The detail that makes the difference: **I bake a code review cycle into the plan, per phase.** Why? A code review across many files at once isn't useful — unless you can spawn several agents to split it up. So at each completed phase I run the review, apply the fixes, and only then move to the next phase. That's what gives me reliable code without drowning the review in one giant phase.

### 4. Real testing

Done? No. The last step is **real testing**.

Unit tests are essential and a delivery requirement — but your coworker is going to test your task for real. Are you going to trust a delivery just because it has AI-generated unit tests? I'm not.

So I run **everything** that was implemented: endpoints, jobs, queues, workers. I test multiple scenarios, error scenarios, validations, trying to cover as much as possible. **About 80% of the time** something needs a targeted fix or improvement to get every scenario mapped and covered. When I fix or improve, **the unit test goes along with it** — and only then do I have a finished delivery.

Explained this way it sounds like it takes a whole day. It doesn't: it's a few hours even on bigger tasks. Each project has its own red tape to get started or to test, of course, but in the end the code reviews rarely come back with comments and I keep shipping.

## Autonomy Under Guardrails

The same flow runs on any project, but how much slack I give the agent scales with the environment's risk.

On **my own personal projects** I grant more autonomy: a password manager, **guardrails** protecting against leaks and against destructive commands, and from there the agent does deploys, fixes, log reading, API calls, and database access — all fast, because **it's all in Anchored**.

The memory is what sustains this. It records, per project: how to fetch the access and how to use it, where each project lives, what the stack is, which commands need to run, and whether there were errors before. The `learning` category is the one that pays off most — every fresh session starts with a lesson already paid for in a past session.

## What I Learned

**Two models with a clear rule beat five models with no rule.** I spent the year testing everything; the gain didn't come from finding the perfect model, it came from cutting down to two (Opus 4.8 and GLM 5.2) and knowing exactly when to use each. Fewer decisions per session, more primary tokens for what matters.

**Context before the plan is worth more than the plan.** `/plan` bothered me because it attacked the wrong step. The bottleneck was never writing the plan — it was reaching the plan without knowing whether the task depends on something, who touched that code before, or whether it even has enough context to be delivered. Anchored + logs + Datadog solve that before the first line.

**Planning is expensive-model work; reading code is cheap-model work.** Letting Sonnet comb through files and Opus decide architecture is the right split. The reverse — Sonnet planning — is where the trade-offs get lost.

**Code review per phase, not at the end.** Reviewing one phase at a time keeps the review small and useful. Reviewing everything at once at the end, without an army of agents, is where bugs hide.

**Real testing is what separates "CI passed" from "delivered".** AI-generated unit tests validate the logic the AI imagined. Running an actual endpoint, job, queue, and worker, with error scenarios, is what reveals what's missing — and 80% of the time something is. Only then does the unit test become true.

**Memory is what lets the flow scale to 60 projects.** None of this — two models, three terminals, four steps — survives the third project without a cross-tool memory that remembers stack, access, commands, and past mistakes for you. Anchored is the piece that turns "nice process" into "process I actually use every day."

## Acknowledgments

Again to [Aron](https://github.com/aronpc), friend and coworker, with whom I trade ideas about this workflow almost every day. Half of what's here was born from those conversations — his flow is a cousin of mine, and it's by comparing the two that each of us kept sharpening our own.

---

**Links:**

- [Anchored (local client) on GitHub](https://github.com/jholhewres/anchored)
- [Anchored OSS (team server) on GitHub](https://github.com/jholhewres/anchored_oss)
- [AI Traffic Lights (Aron) on GitHub](https://github.com/aronpc/ai-traffic-lights)
- [Post: Anchored — One Memory for All Your Tools](/blog/anchored-cross-tool-ai-memory-mcp)
- [Post: Anchored OSS — Team Memory for Devs Who Code with AI](/blog/anchored-oss-team-memory-server)
- [Model Context Protocol](https://modelcontextprotocol.io/)

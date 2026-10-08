---
title: "One Orchestrator for Every Project: How I Use devpit Day to Day"
date: 2026-10-08
tags: ["ai", "devpit", "ade", "claude-code", "orchestration", "workflow"]
summary: "Two weeks after launching devpit, the way I work changed again. An Orchestrator that keeps work and personal projects apart, and sessions that start from scratch with a complete prompt and report back at every step. I discuss in the chat, the CLI is only for development, and nothing reaches production without my approval."
reading_time: 5
---

In the [devpit launch post](/blog/devpit-agentic-development-environment) I talked about the problem of having too many terminals: five fronts, five Claude Codes, an alarm room. devpit solved that inside each project. What was missing was solving it above them.

That's what the **Orchestrator** does in my day to day. It's a chat that sees all my projects, opens sessions in them, receives what they report and hands me back what needs a decision. This post is about how I use it.

## Work on One Side, Personal on the Other

I use the Orchestrator with my Claude account, and the first thing it solved was keeping work apart from my personal projects.

```
                    ORCHESTRATOR
          ┌──────────────┴──────────────┐
        WORK                         PERSONAL
   work projects                  POCs and MVPs
          │                              │
   artifacts · contexts           artifacts · contexts
   useful information             useful information
```

Each side has its own artifacts, its own contexts and the useful information of each project, organized. No work context leaking into a personal POC, or the other way around.

## Discuss in the Chat, Develop in the CLI

The split that changed my pace the most was this: **I discuss in the Orchestrator chat and use the CLI only for development.**

In the chat I think out loud, compare approaches, ask for recommendations, decide the scope. When it's time to touch code, the Orchestrator opens a session in the project and sends the request. The session doesn't need to hear the whole conversation; it needs to know what to do.

## Every Session Starts From Scratch

This is what makes the biggest difference. Every session is born clean, with a complete prompt:

- **the request**: what needs to be done;
- **my preferences**: visual tests, running the suites, validating the idea before implementing, looking for recommendations, understanding the project's context before touching it;
- **the information it needs**: only what that task requires.

The preferences go with every command, so I don't repeat "run the tests" or "take a screenshot" in every session. And since the session starts from scratch, the context doesn't fill up with what isn't useful: no thirty messages of discussion that have already become a one-line decision.

I can have **several sessions in the same repository or the same worktree**, each with its own request.

## An End-to-End Flow

A generic example: a bug where messages aren't being sent for a customer.

```
  me ──► Orchestrator chat
           │  "messages aren't going out for this customer, look into it"
           ▼
         brief = request + preferences + project context
           │
           ▼
         session in the project  (starts from scratch)
           │
           ├── report: reproduced it, the cause is X
           ├── report: fix ready, suite passing
           └── report: draft ready for review
           ▼
         Orchestrator ──► summarizes what needs my decision
           │
           ▼
         I approve (or not). Nothing reaches production without it.
```

1. **The request.** I describe the problem in the chat, the way it reached me.
2. **The brief.** The Orchestrator builds the session's prompt with the request, my preferences and what it knows about the project.
3. **The session.** It investigates, debugs, fixes and runs the tests, the way I would.
4. **The reports.** At each step the session sends the Orchestrator a summary, a piece of information or a request to move forward.
5. **The decision.** Whatever goes out comes as a draft. I read it and approve it. Commit, deploy, a message to someone: none of that happens without my OK.

The same flow works for a feature, a debugging session or starting a POC.

## The Conversation Between the Orchestrator and the Session

The session doesn't work in isolation until the end. Along the way it asks for context or information: a project detail, a decision that was left open, something that wasn't in the brief. Those requests reach the Orchestrator, and I have two ways of answering them:

- **define it upfront**: whatever is already agreed, the Orchestrator answers on its own, without calling me;
- **explain it and it answers**: when it's something only I know, I explain it to the Orchestrator in my own words and it answers the session.

In the other direction, the session keeps the Orchestrator up to date on its progress at all times. I don't need to ask how things are going; the state of each front is already there when I look.

## Visibility Without Giving Up the Decision

What I gained from all this is **visibility**. Sessions talk to the Orchestrator and report at every step, so I know where each front stands without opening a single terminal. And since each step is well defined, I don't lose the ability to decide: I know what was done, what's left and what's waiting for me.

I can leave decisions pre-arranged ("if the suite passes, open the draft") or decide in the middle of something else, when the report arrives. I'm now in much better control, with visibility over all my projects and POCs at the same time.

That's why I can now do things I used to put off:

- **start POCs and MVPs**, like a SaaS POC that became an MVP in a day;
- **open sessions to solve problems**, debug or build features without dropping what I was doing.

## The Plans Went Further

One consequence I didn't expect: my Claude 5x plan has gone further these last few weeks, and so has the company's. I use the two separately, each with its own context.

## After the Launch

devpit passed **100 upvotes on Product Hunt** and **50 sign-ups** through the app's OAuth. Thanks to everyone who tried it, voted and sent feedback.

## What's Next

I'm preparing the next update, with a lot of cool and useful things, always with the developer in mind. No date yet. When it ships, I'll write about it here.

If you want to try it:

```sh
curl -fsSL https://raw.githubusercontent.com/jholhewres/devpit/main/install.sh | sh
```

---

**Links:**

- [devpit on GitHub](https://github.com/jholhewres/devpit)
- [Site and documentation](https://devpit.jhol.dev)
- [Post: devpit — One Workspace Per Project](/blog/devpit-agentic-development-environment)
- [Post: My AI Coding Workflow](/blog/ai-coding-workflow-two-models-anchored)

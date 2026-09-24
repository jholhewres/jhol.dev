---
title: "devpit: One Workspace Per Project for Coding with Agents"
date: 2026-09-23
tags: ["ai", "devpit", "ade", "claude-code", "rust", "tauri", "open-source"]
summary: "Every project I touched produced plans, reviews, research and artifacts that never went into git, and I kept creating invisible directories to cope. devpit is my attempt at an ADE (Agentic Development Environment): one space per project with the terminal, chat, agents, a board that runs the work, code review, tests, notes and Excalidraw."
image: cover.png
reading_time: 9
---

Every project I work on with agents leaves a trail. A plan, a refinement of the plan, a review, a code review for each phase, reference research, a decision I made and why, a screenshot of a bug, a prompt draft. Much of it shouldn't go into the repository — it's internal material, the story of how the code was born. But it can't get lost either, because it's exactly what I need when I come back to the project two days later.

For a long time my solution was to create directories invisible to git. An `.omc/` here, a `.project/` there, a `.claude/`, each tool with its own folder and its own convention, all thrown into `.gitignore`. It worked, until I had several projects open at once and could no longer remember which folder of which project held the plan I had reviewed the week before.

And then there are ideas. In the middle of a task something always comes up that isn't part of the task — a feature, a refactor, a "what if we did it this way". Write it down where? A loose file, Jira, a conversation with the agent that's about to be compacted? I went as far as writing a skill just to keep appending ideas to an `ideas.md`.

Over time it became clear the problem wasn't a missing tool. The place where I thought the work through, the place where it ran and the place where I reviewed it were three different places, and keeping all three current was manual work I simply didn't do.

That's where **[devpit](https://github.com/jholhewres/devpit)** came from.

## What an ADE Is

An IDE is an *Integrated Development Environment*: the editor is the center and everything orbits the open file. Once the work is driven by agents, that center moves. I no longer spend the day editing files — I spend it describing what needs to be done, following sessions, reviewing diffs, running tests and deciding the next step.

An **ADE — Agentic Development Environment** — is an environment built around that. In devpit the center isn't the file, it's the **project**, and inside each project is everything I need to work with agents:

```
                         devpit
  ┌──────────────────────────────────────────────────────┐
  │  project A      project B      project C      ...    │
  └──────┬───────────────────────────────────────────────┘
         │
         ├── terminal           your usual Claude Code
         ├── chat               with cost, questions and checklist on screen
         ├── board              moving a card is what starts the work
         ├── agents             the markdown files already on your machine
         ├── files + diff       code review without leaving
         ├── runs               what ran, what it cost, what it proved
         ├── browser            the page the project is serving
         ├── notes              markdown saved in the project folder
         └── excalidraw         drawings saved in the project folder
```

The idea is that I open a project and everything is there. Switching projects switches the whole space — terminal, board, notes, conversations — and when I come back, it's the way I left it.

I was also careful about the other side: I didn't want to build yet another IDE. Open tabs, pane layouts, the file being edited — that's screen detail. What matters is the project, the work, and what was done in it.

## What Was Bothering Me

Before starting, I wrote the problem down my own way. There were several requests, but deep down they were three frustrations.

**I was losing sight of my tasks.** Not for lack of a kanban — I already had one. The problem was that the card described the work in one place and the work happened in another. A board that only describes is stale by the first busy day. In devpit I flipped that around: moving the card is what makes the work happen. If the card is in "review", it's because the review ran there.

**The orchestration was vague.** When I called a command that spun up agents on its own, I didn't know how many would start, what it would cost, where the state ended up or which step it was on. The prompts were good; what was missing was contour: one step at a time, visible, with a spending cap.

**There were too many terminals.** Five lines of work in five terminals with five Claude Codes looks productive, but in practice it turns into an alarm room. I wanted one focused terminal per project, with the whole flow talking to it.

## One Terminal Per Project

Each project has its own terminal, and that's where Claude Code runs the way it always has. You open devpit, type, and it's the same Claude Code as always. You don't need to create a card or set anything up to use it — the board is optional.

Switching projects switches what's on screen, but the previous session keeps running in the background. Terminals are **tmux** sessions, so closing devpit kills nothing: I open it again and everything is there, even after the app updates itself.

You can run the same CLI with different accounts using **profiles**. It's my `claude` / `claudep` / `glm` from the [post about my workflow](/blog/ai-coding-workflow-two-models-anchored), just inside the app. devpit recognizes Codex, Gemini, Cursor, Aider and other CLIs and opens them in the terminal; for now, only Claude Code is driven end to end.

## A Board That Runs the Work

This is the part that replaced my orchestration commands. Each project has a board, and the columns are mine: I rename them, reorder them, create whichever I want. A column can do nothing, or it can run a **step** when a card arrives. There are three kinds of step:

- **agent** — an agent runs in the background, without taking the terminal, and returns a structured answer with its cost. Good for refining, reviewing and verifying.
- **session** — a session in the project's terminal, which I drive. That's where implementation happens.
- **command** — a command of mine: tests, build, lint, deploy.

A board set up that way looks roughly like this:

```
   [inbox]      [refine]      [review]      [doing]       [verify]      [ship]
      │             │             │             │             │             │
   I write       planner       critic       executor      verifier       my
   the card      · opus        · opus       · sonnet      · sonnet      command
                  agent         agent        session       agent        command
```

Each column holds a recipe: which agent, on which model, with which part of the card's context and how much it may spend. That solves something that always bugged me — having twenty agents available everywhere means loading twenty agents everywhere and paying for them on every call. Here the column says *this* step is *that* agent, and only that goes out.

And a card only moves as far as I let it. Each column can be manual (nothing moves on its own), ask before moving, or move automatically and fire the next step. Nothing runs behind my back: if a chain of columns moves by itself, it's because I set it up that way.

The agents are the markdown files already on my machine. devpit ships no agents and copies nothing — it reads the ones I already have.

When I'm juggling several projects, the **Manager** puts every board on one screen. The "review" column of three projects becomes a single column with the cards of all three, and I see at once what's waiting on me.

## Code Review and Tests Without Leaving

Reviewing what the agent did shouldn't force me to open another tool. Next to the terminal sits the files panel: the project tree with search, what changed with each file's diff, and what's already been committed. Staging and unstaging happen right there.

For tests, I wanted to fix something that has bitten me more than once: **exit code zero doesn't mean it passed.** A runner that found no test files exits with zero and tested nothing. So every execution in devpit answers three separate questions: what the check said (passed, failed, didn't run, or left no readable result), whether that result still applies to the code in front of me or the code has changed since, and what was found.

Every agent call also writes on the card how much it cost when it finishes. "The agent is doing something" becomes actual information.

## Notes, Drawings and the End of Invisible Directories

This is the part that solves the problem from the start of the post. Each project gets its own folder inside `~/.devpit`, outside the repository, and everything that belongs to the project but isn't code goes there:

```
~/.devpit/projects/my-project/
├── agents/          the agents this project's board uses
├── conversations/   the conversations, with pasted images
├── attachments/     card attachments
├── data/notes/      notes as .md, which open in Obsidian
├── data/excalidraw/ drawings as .excalidraw, which open on excalidraw.com
└── local/           screen layout and worktrees
```

Today there are two extras you turn on per project: **Notes**, in markdown with a block editor, and **Excalidraw**, for hand-drawn diagrams — and you can pin a drawing to a card. Both write plain files, in each tool's own format. Nothing is locked inside devpit: if I stop using it one day, the notes are still markdown and the drawings still open in Excalidraw.

And for the ideas that show up mid-work, the board's "inbox" column exists exactly for that: a card that runs nothing until I decide what to do with it, with the notes right next to it.

## A Few Choices Along the Way

devpit is a desktop app in **Rust with Tauri**. It stays open all day, so memory use and startup time matter. The frontend's types are generated from Rust, so the contract between the two sides never drifts out of sync.

For the terminal, I chose to lean on **tmux** instead of writing my own daemon to hold the processes. tmux already delivers exactly what I wanted — the session outlives the window — and it's a piece everyone knows.

And everything is **local**. Projects, conversations, terminal history and files stay on your computer. There's an account, but it's optional and today it only identifies you.

This isn't my first attempt at solving this either. I had started before, in Go and in Rust, and this time I reused what was already done — the terminal, the git integration, the desktop shell — instead of starting from scratch yet again.

## Where It Stands Today

It's an MVP, and there's still plenty to do. Only Claude Code is driven end to end. Linux is the main platform; macOS has an Apple silicon build, not yet signed, and Windows doesn't exist because tmux doesn't exist there. Focus mode is half done. And there are things I want soon myself, like managing artifacts straight from the sidebar and the file explorer, and opening projects that aren't git repositories.

The project is open source (Apache 2.0). If the goal makes sense to you too, try it, open an issue or send a PR.

To install on Linux or macOS:

```sh
curl -fsSL https://raw.githubusercontent.com/jholhewres/devpit/main/install.sh | sh
```

It needs `tmux` and `claude` on the PATH.

---

**Links:**

- [devpit on GitHub](https://github.com/jholhewres/devpit)
- [Site and documentation](https://devpit.jhol.dev)
- [Post: My AI Coding Workflow](/blog/ai-coding-workflow-two-models-anchored)
- [Post: Anchored — One Memory for All AI Tools](/blog/anchored-cross-tool-ai-memory-mcp)

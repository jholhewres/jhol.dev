---
title: "Semantic Search Without an API Key: ONNX + WordPiece in Pure Go"
date: 2026-04-14
tags: ["ai", "devclaw", "onnx", "embeddings", "compaction", "memory"]
summary: "How I made DevClaw's memory system fully autonomous — local ONNX embeddings that work without API keys, and a compaction system that preserves conversation topics instead of silently discarding them."
reading_time: 10
---

My AI assistant forgot a conversation that happened 17 minutes earlier. The user discussed LiteLLM proxy configuration at 20:50, switched to a server investigation at 20:53, and at 21:07 asked to go back to the LiteLLM topic. The response: "I don't remember anything about LiteLLM."

Seventeen minutes. Same session. Same chat. Gone.

This was another failure in DevClaw's memory system — which I was already [rebuilding from scratch with MemPalace and TurboQuant](/blog/devclaw-mempalace-turboquant-kg-memory). This time, the problem wasn't what got saved. It was what got thrown away.

## Why Context Disappears

DevClaw manages its context window with a multi-level compaction pipeline. When the LLM context fills up, the system progressively summarizes older messages to make room for new ones. The pipeline has four levels:

1. **Collapse** (70%): Truncate oversized tool results
2. **Micro-compact** (80%): Clear old tool results with placeholders
3. **Auto-compact** (93%): LLM-based summarization
4. **Memory-compact** (97%): Extract memories, then aggressive summarization

The LiteLLM conversation was lost at level 3. Here's what went wrong.

### Problem 1: The Summary Prompt Had No Section for Topics

The structured summarization prompt told the LLM to preserve five things:

```
## Decisions
## Open TODOs
## Constraints/Rules
## Pending user asks
## Exact identifiers
```

The LiteLLM discussion was none of these. It was informational — the user asked how something works, the AI explained it, no action was taken. When the summarizer ran, it produced a summary about the SSH investigation (decisions made, servers restarted, identifiers used) and silently dropped the LiteLLM topic because it didn't fit any section.

### Problem 2: Only 8 Recent Messages Survived

`computeAdaptiveKeepRecent()` capped at 8 messages. The SSH investigation generated 30+ tool calls (SSH attempts, GCP secret access, key conversion, Horizon restart). The 8 most recent messages were all SSH-related. Everything before — including the LiteLLM exchange — went into the "middle" section that gets summarized.

In aggressive compaction, the floor was even lower: 2 messages.

### Problem 3: Session-Level Compaction Used a Flat Prompt

The agent-level compaction used a structured prompt with quality guards. But the session-level compaction (between agent runs) used: "Summarize the key points of this conversation in 2-3 sentences." Two to three sentences for potentially hours of conversation.

## The Fix: Topic Anchors

The core insight: the LLM is bad at preserving what it doesn't know is important. "We discussed LiteLLM" feels like small talk to a summarizer trained to extract decisions and action items. But it's context that affects future responses.

### Zero-Cost Topic Extraction

Before the LLM touches the data, `extractTopicAnchors()` scans user messages from the section being compacted and builds a deduplicated bullet list — first 150 characters of each message.

```go
func extractTopicAnchors(messages []chatMessage) string {
    seen := make(map[string]bool)
    var topics []string
    for _, m := range messages {
        if m.Role != "user" {
            continue
        }
        s, ok := m.Content.(string)
        if !ok || len(s) < 15 {
            continue
        }
        if strings.HasPrefix(s, "[System:") {
            continue
        }
        preview := s
        if len([]rune(preview)) > 150 {
            preview = string([]rune(preview)[:150])
        }
        key := strings.ToLower(strings.TrimSpace(preview))
        if !seen[key] {
            seen[key] = true
            topics = append(topics, "- "+preview)
        }
    }
    if len(topics) == 0 {
        return ""
    }
    return strings.Join(topics, "\n")
}
```

No LLM call, no embedding, no cost. Pure string extraction. The anchors are appended to the summary as `## Conversation Topics (pre-compaction)`. When that summary is later re-compacted, the LLM now has a structured section to preserve.

### The New Section

The summarization prompt gained a sixth required section:

```
## Conversation Topics
List ALL distinct subjects discussed, including purely
informational discussions where no action was taken.
```

This was added to every compaction path: the agent-level structured prompt, the session-level prompt, the LCM (Lossless Compaction Manager) prompts at all four depth levels, the multi-chunk merge prompt, and the deterministic fallback (the path that runs when the LLM is unavailable).

### Wider Preservation Window

`keepRecent` cap raised from 8 to 15. Aggressive compaction floor raised from 2 to 4. These are deliberately generous — the cost of keeping a few extra messages is much lower than the cost of losing conversational context.

## Local ONNX Embeddings: Semantic Search Without API Keys

The memory system has two search modes: BM25 keyword search (via SQLite FTS5) and vector similarity search (via embeddings). Vector search catches semantic matches that keyword search misses — "what did we talk about yesterday" matches a memory about "discussed server configuration on April 12th."

The problem: vector search required an embedding API key. The default was `embedding.provider: "none"`, which meant most self-hosted users had keyword search only.

### Pure-Go Sentence Transformer

`ONNXEmbedder` runs `all-MiniLM-L6-v2` (384 dimensions) locally via ONNX Runtime. No Python, no external services, no API keys.

The tricky part was the tokenizer. Every existing Go tokenizer library either requires Python or has heavy CGo dependencies. I wrote a pure-Go WordPiece tokenizer from scratch:

```go
type WordPieceTokenizer struct {
    vocab    map[string]int
    unkToken string
    maxLen   int
}

func (t *WordPieceTokenizer) Tokenize(text string) (
    inputIDs, attentionMask, tokenTypeIDs []int64,
) {
    tokens := []string{"[CLS]"}
    for _, word := range splitOnPunctuation(strings.ToLower(text)) {
        // WordPiece: try full word, then progressively shorter
        // prefixes with "##" continuation marker
        sub := t.wordPieceTokenize(word)
        tokens = append(tokens, sub...)
    }
    tokens = append(tokens, "[SEP]")
    // Pad/truncate to maxLen, build attention mask
    // ...
}
```

The embedder auto-downloads the ONNX Runtime shared library, the model file (87MB), and the vocabulary on first use. Downloads include SHA-256 checksum verification — the runtime `.so` is loaded via `dlopen`, so integrity matters.

### The Priority Inversion

The initial auto-detection tried API keys first, then ONNX. This was backwards. ONNX is zero-cost, fully offline, no API dependency. The fix:

```
Priority: explicit config > ONNX local > API key fallback > NullEmbedder
```

A subtle bug reinforced the wrong order: the main LLM API key was being injected into the embedding auto-detection. If any API key existed (which it always does — the LLM needs one), auto-detect always chose OpenAI over ONNX. The fix: only use the API key for embeddings when the user explicitly configured an embedding provider.

### The Version Saga

Three commits tell a story about CGo binding compatibility:

1. Initial: ONNX Runtime 1.22.0 — error: "Platform-specific initialization failed"
2. Fix attempt: 1.27.0 (matching Go module version) — 404, release doesn't exist
3. Final fix: 1.24.1 — matches the C API headers in the Go binding's README

The Go module version (`yalue/onnxruntime_go v1.27.0`) doesn't match the native library version it requires (`1.24.1`). This is a common CGo trap: the binding version and the runtime version are independently versioned.

## Making It Autonomous

The philosophy behind these changes: the system self-corrects without additional configuration.

DevClaw now includes:
- Semantic search working out of the box (ONNX auto-detected, no API key needed)
- Conversation topics preserved across compaction
- Dream system actually running (was orphaned, now lazy-init on first use)
- Expired memories auto-cleaned (category TTL + FileStore.Compact)
- Credentials blocked from memory, redacted from output (four perimeters)

All without additional configuration.

The dream directory that caused 10-minute error cycles? `os.MkdirAll` before the lock file. The session boundary problem for WhatsApp? Count compactions as session proxies. The pipeline status log showing `"none"` instead of `"onnx"`? A cosmetic bug — the actual embedder is correct, just the log reads the config value instead of the resolved provider.

## What I Learned

**LLMs discard what looks unimportant.** A summarizer optimized for "key decisions and action items" will drop informational discussions. The fix isn't better prompts — it's extracting topic anchors before the LLM runs, so the data is preserved regardless of summarization quality.

**Auto-detection order matters more than auto-detection logic.** ONNX-first vs API-first is a one-line change that completely changes the user experience. The wrong default meant zero users got local embeddings even though the feature was shipped and working.

**Background systems need integration tests, not unit tests.** The dream system had tests. They passed. But `Assistant.Start()` never called `dream.Start()`. The goroutine was orphaned for months. A single integration test — "start the assistant, wait, check that dream ran" — would have caught it immediately.

**Async is not always faster.** The memory flush was made async to avoid blocking compaction. The race detector flagged it. Reverting to synchronous — with a 30-second timeout — fixed the race and didn't measurably impact compaction speed. The bottleneck was the LLM summarization call, not the memory flush.

---

**Links:**

- [Previous post: MemPalace, TurboQuant and Bitemporal KG — DevClaw's Long-Term Memory](/blog/devclaw-mempalace-turboquant-kg-memory)
- [DevClaw on GitHub](https://github.com/jholhewres/devclaw)
- [ONNX Runtime for Go](https://github.com/yalue/onnxruntime_go)
- [all-MiniLM-L6-v2 on HuggingFace](https://huggingface.co/sentence-transformers/all-MiniLM-L6-v2)

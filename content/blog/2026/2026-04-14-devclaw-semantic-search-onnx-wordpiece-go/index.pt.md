---
title: "Busca Semântica Sem API Key: ONNX + WordPiece em Go Puro"
date: 2026-04-14
tags: ["ai", "devclaw", "onnx", "embeddings", "compaction", "memory"]
summary: "Como tornei o sistema de memória do DevClaw totalmente autônomo — embeddings ONNX locais sem chaves de API, e um sistema de compactação que preserva tópicos de conversa em vez de descartá-los silenciosamente."
reading_time: 10
---

Meu assistente de IA esqueceu uma conversa que aconteceu 17 minutos antes. O usuário discutiu configuração de proxy LiteLLM às 20:50, mudou pra uma investigação de servidor às 20:53, e às 21:07 pediu pra voltar ao assunto do LiteLLM. A resposta: "Não encontrei nada sobre litellm na memória."

Dezessete minutos. Mesma sessão. Mesmo chat. Sumiu.

Essa foi mais uma falha no sistema de memória do DevClaw — que eu já vinha [reconstruindo do zero com MemPalace e TurboQuant](/blog/devclaw-mempalace-turboquant-kg-memory). Desta vez, o problema não era o que foi salvo. Era o que foi descartado.

## Por Que o Contexto Desaparece

O DevClaw gerencia sua janela de contexto com um pipeline de compactação multi-nível. Quando o contexto do LLM enche, o sistema progressivamente resume mensagens mais antigas. O pipeline tem quatro níveis:

1. **Collapse** (70%): Trunca resultados de ferramentas grandes
2. **Micro-compact** (80%): Substitui resultados antigos por placeholders
3. **Auto-compact** (93%): Sumarização via LLM
4. **Memory-compact** (97%): Extrai memórias, depois sumarização agressiva

A conversa sobre LiteLLM foi perdida no nível 3. Três causas raiz:

**O prompt não tinha seção pra tópicos.** A sumarização estruturada preservava Decisions, Open TODOs, Constraints, Pending asks e Exact identifiers. A discussão sobre LiteLLM não era nenhuma dessas — era informacional, sem ação tomada. O sumarizador simplesmente descartou.

**Apenas 8 mensagens recentes sobreviviam.** `computeAdaptiveKeepRecent()` tinha cap de 8. A investigação SSH gerou 30+ tool calls. As 8 mais recentes eram todas SSH. Tudo antes — incluindo LiteLLM — foi sumarizado.

**Compactação de sessão usava prompt genérico.** "Resuma os pontos principais em 2-3 frases." Duas a três frases pra potencialmente horas de conversa.

## O Fix: Âncoras de Tópico

O insight central: o LLM é ruim em preservar o que não sabe que é importante. "Discutimos LiteLLM" parece conversa fiada pra um sumarizador treinado pra extrair decisões e itens de ação. Mas é contexto que afeta respostas futuras.

### Extração Zero-Custo

Antes do LLM tocar nos dados, `extractTopicAnchors()` varre mensagens do usuário na seção sendo compactada e constrói uma lista deduplicada — primeiros 150 caracteres de cada mensagem.

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

Nenhuma chamada LLM, nenhum embedding, zero custo. As âncoras são adicionadas ao resumo como `## Conversation Topics (pre-compaction)`. Quando esse resumo for re-compactado, o LLM tem uma seção estruturada pra preservar.

### A Nova Seção

O prompt de sumarização ganhou uma sexta seção obrigatória:

```
## Conversation Topics
Liste TODOS os assuntos distintos discutidos, incluindo discussões
puramente informacionais onde nenhuma ação foi tomada.
```

Adicionei em todos os caminhos de compactação: prompt estruturado do agente, prompt de sessão, os quatro níveis de profundidade do LCM, merge multi-chunk e o fallback determinístico.

### Janela Mais Ampla

Cap de `keepRecent` aumentado de 8 pra 15. Mínimo da compactação agressiva de 2 pra 4. O custo de manter mensagens extras é muito menor que o custo de perder contexto.

## ONNX Local: Busca Semântica Sem Chave de API

O sistema de memória tem dois modos de busca: BM25 por palavras-chave (via SQLite FTS5) e similaridade vetorial (via embeddings). A busca vetorial captura matches semânticos — "o que conversamos ontem" encontra uma memória sobre "discutimos configuração de servidor em 12 de abril."

O problema: busca vetorial exigia uma chave de API de embedding. O default era `embedding.provider: "none"`, ou seja, a maioria dos usuários self-hosted tinha apenas busca por palavras-chave.

### Sentence Transformer em Go Puro

`ONNXEmbedder` roda `all-MiniLM-L6-v2` (384 dimensões) localmente via ONNX Runtime. Sem Python, sem serviços externos, sem chaves de API.

A parte mais desafiadora foi o tokenizador. Toda biblioteca Go de tokenização existente requer Python ou CGo pesado. Escrevi um tokenizador WordPiece do zero:

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
        // WordPiece: tenta palavra completa, depois prefixos
        // progressivamente menores com marcador "##"
        sub := t.wordPieceTokenize(word)
        tokens = append(tokens, sub...)
    }
    tokens = append(tokens, "[SEP]")
    // Pad/truncate para maxLen, constrói attention mask
}
```

O embedder auto-baixa o ONNX Runtime (21MB), o modelo (87MB) e o vocabulário no primeiro uso, com verificação SHA-256 — o `.so` é carregado via `dlopen`, então integridade importa.

### A Inversão de Prioridade

A auto-detecção inicial tentava chaves de API primeiro, depois ONNX. Estava invertido. ONNX é custo zero, totalmente offline, sem dependência externa. A prioridade correta:

```
Config explícita > ONNX local > Fallback de chave API > NullEmbedder
```

Um bug sutil reforçava a ordem errada: a chave do LLM principal estava sendo injetada na auto-detecção de embedding. Se qualquer chave API existisse (e sempre existe), auto-detect sempre escolhia OpenAI sobre ONNX. O fix: usar a chave pra embeddings somente quando o usuário explicitamente configurou um provider.

### A Saga das Versões CGo

Três commits contam uma história sobre bindings CGo:

1. ONNX Runtime 1.22.0 → erro: "Platform-specific initialization failed"
2. Fix: 1.27.0 (igualando versão do módulo Go) → 404, release não existe
3. Fix final: 1.24.1 → corresponde aos headers da C API no README do binding

A versão do módulo Go (`yalue/onnxruntime_go v1.27.0`) não corresponde à versão da biblioteca nativa (`1.24.1`). Armadilha clássica do CGo: binding e runtime são versionados independentemente.

## Tornando Autônomo

A filosofia por trás de tudo: **o sistema se autocorrige sem configuração adicional**.

O DevClaw agora inclui:
- Busca semântica funcionando out-of-the-box (ONNX auto-detectado)
- Tópicos de conversa preservados após compactação
- Dream system rodando de verdade (era órfão, agora lazy-init)
- Memórias expiradas auto-limpas (TTL por categoria + Compact)
- Credenciais bloqueadas na memória, redatadas na saída

Tudo sem configuração adicional.

## O Que Aprendi

**LLMs descartam o que parece irrelevante.** Um sumarizador otimizado pra "decisões-chave e itens de ação" descarta discussões informacionais. O fix não é prompts melhores — é extrair âncoras de tópico antes do LLM rodar.

**A ordem de auto-detecção importa mais que a lógica.** ONNX-first vs API-first é uma mudança de uma linha que muda completamente a experiência do usuário.

**Sistemas de background precisam de testes de integração.** O Dream tinha testes unitários. Passavam. Mas `Assistant.Start()` nunca chamava `dream.Start()`. A goroutine ficou órfã por meses.

**Async nem sempre é mais rápido.** O flush de memória foi tornado async pra performance. O race detector flagrou. Reverter pra síncrono corrigiu a race sem impactar a velocidade — o gargalo era a chamada de sumarização do LLM, não o flush.

---

**Links:**

- [Post anterior: MemPalace, TurboQuant e KG Bitemporal — A Memória de Longo Prazo do DevClaw](/blog/devclaw-mempalace-turboquant-kg-memory)
- [DevClaw no GitHub](https://github.com/jholhewres/devclaw)
- [ONNX Runtime para Go](https://github.com/yalue/onnxruntime_go)
- [all-MiniLM-L6-v2 no HuggingFace](https://huggingface.co/sentence-transformers/all-MiniLM-L6-v2)

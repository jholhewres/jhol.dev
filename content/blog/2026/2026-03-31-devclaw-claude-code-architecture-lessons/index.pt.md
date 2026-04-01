---
title: "Como o Vazamento do Claude Code me Ajudou a Melhorar o DevClaw"
date: 2026-03-31
tags: ["ai", "devclaw", "claude-code", "agent-architecture", "open-source"]
summary: "Após o vazamento do código-fonte do Claude Code via npm source maps, estudei a análise da comunidade e descobri como resolver problemas que me travavam no DevClaw — concorrência de ferramentas, compactação multi-estratégia, extração de memória e um dream system"
reading_time: 10
---

Há meses eu trabalhava em problemas que sabia que tinham solução, mas não conseguia acertar a implementação. Como fazer ferramentas rodarem em paralelo sem race conditions? Como compactar contexto sem perder informação crítica? Como consolidar memórias entre sessões de forma confiável?

Eu entendia os conceitos na teoria. Na prática, cada tentativa no DevClaw tinha arestas — edge cases que não fechavam, trade-offs que eu não sabia como balancear. Faltava ver como alguém com mais recursos tinha resolvido esses mesmos problemas em produção.

Aí veio o vazamento.

## O que Aconteceu

Na madrugada de 31 de março de 2026, o pesquisador de segurança [Chaofan Shou](https://x.com/shoucccc) descobriu que a Anthropic havia incluído acidentalmente um arquivo source map (`cli.js.map`) na versão 2.1.88 do pacote `@anthropic-ai/claude-code` no npm. O arquivo continha o código-fonte completo do Claude Code — **1.900 arquivos TypeScript, 512.000+ linhas de código**, incluindo 44 feature flags de funcionalidades ainda não lançadas.

Em poucas horas, o código já estava espelhado em repositórios públicos no GitHub. A Anthropic confirmou que foi [um erro de empacotamento](https://venturebeat.com/technology/claude-codes-source-code-appears-to-have-leaked-heres-what-we-know), não uma brecha de segurança — nenhum dado de cliente foi exposto. Mas o código do agente inteiro estava ali para quem quisesse estudar.

## O Estudo da Comunidade

O que me interessou não foi o código bruto — foi o trabalho que a comunidade fez em cima dele. O repositório [instructkr/claw-code](https://github.com/instructkr/claw-code) se tornou o **repo mais rápido da história a atingir 50K stars**, alcançando o marco em apenas 2 horas. Mas além do hype, o projeto fez algo valioso: uma análise arquitetural do harness de agente do Claude Code, seguida de um clean-room rewrite — primeiro em Python, agora em Rust.

O tagline do projeto diz tudo: **"Better Harness Tools, not merely storing the archive of leaked Claude Code."**

Foi estudando essa análise que as peças que faltavam no DevClaw se encaixaram. Não copiei código — o Claude Code é TypeScript rodando em Bun, o DevClaw é Go puro. Mas os **padrões arquiteturais** são universais, e finalmente vi como aplicá-los do jeito certo.

## O que Eu Melhorei no DevClaw

### Concorrência no Uso de Ferramentas

Meu problema: o DevClaw executava todas as ferramentas sequencialmente. Quando o agente precisava rodar 4 greps e um `git log` para explorar o codebase, esperava cada um terminar antes de começar o próximo.

O que aprendi: o Claude Code classifica ferramentas em **readonly** e **mutating**. Ferramentas de leitura (`grep`, `ls`, `git log`) são idempotentes e podem rodar em paralelo com segurança. Ferramentas de escrita (`file_write`, `git commit`) são serializadas. Parece óbvio em retrospecto, mas o detalhe que me faltava era o **abort em cascata** — quando uma ferramenta crítica falha, as dependentes são canceladas imediatamente.

```go
type ToolExecution struct {
    Tool     Tool
    Category ToolCategory // readonly | mutating
    Status   ExecStatus   // pending | running | done | aborted
    Result   chan ToolResult
    Cancel   context.CancelFunc
}
```

O resultado: em tarefas de exploração de codebase, o tempo de resposta caiu pela metade.

### Compactação Multi-Estratégia

Meu problema: o DevClaw tinha compactação tudo-ou-nada. Quando o contexto ficava grande, sumarizava tudo de uma vez — e frequentemente perdia informação importante no processo.

O que aprendi: compactação deveria ser um **pipeline progressivo** com estratégias diferentes para cada situação:

1. **Collapse** — colapsa tool results verbosos mantendo o essencial
2. **Micro-compact** — compacta blocos antigos preservando decisões
3. **Auto-compact** — sumarização completa no threshold
4. **Memory-compact** — extrai fatos duráveis antes de descartar

O insight que eu não tinha: um **circuit breaker** que desativa compactação após falhas consecutivas. Meu sistema anterior podia entrar em loop quando o modelo produzia resumos maiores que a entrada. Agora, após 3 falhas, o circuit breaker para e usa truncamento determinístico como fallback.

### Extração de Memórias Antes de Compactar

Meu problema: quando o DevClaw compactava contexto, conhecimento valioso ia junto. Decisões de arquitetura tomadas no início da sessão simplesmente desapareciam.

O que aprendi: o passo que faltava é **minerar o contexto antes de comprimir**. Antes de qualquer compactação, um passo de extração identifica:

- **Fatos** sobre o codebase (arquitetura, padrões, dependências)
- **Decisões** tomadas durante a conversa (por que X ao invés de Y)
- **Preferências** do usuário (estilo de código, ferramentas preferidas)

Esses fragmentos vão para memória de longo prazo. O contexto pode ser compactado à vontade — o conhecimento extraído sobrevive independente.

```go
type ExtractedMemory struct {
    Category string   // fact | decision | preference
    Content  string
    Source   string   // qual parte da conversa originou isso
    Tags     []string
}
```

### Dream System — Consolidação Automática

Meu problema: a memória do DevClaw acumulava fragmentos redundantes e até contraditórios ao longo de várias sessões. Não tinha um mecanismo de "limpeza" inteligente.

O que implementei: um sistema inspirado em como o cérebro consolida memórias durante o sono. Roda em background nos períodos de inatividade e faz três coisas:

1. **Consolida** memórias fragmentadas em representações coerentes
2. **Detecta** contradições entre memórias antigas e novas
3. **Prioriza** por frequência de acesso e relevância

O trigger usa três portas — todas precisam estar abertas:

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

Isso evita rodar durante interação ativa e garante que há material suficiente para consolidar.

## O Impacto

Essas quatro melhorias juntas elevaram significativamente o nível do DevClaw:

- **Performance do assistente** — a execução concorrente de ferramentas torna a exploração de codebase dramaticamente mais rápida
- **Qualidade da memória** — extração pré-compactação + dream system = o agente mantém consciência real de decisões anteriores, mesmo em sessões longas
- **Resiliência** — circuit breaker na compactação + abort em cascata nas ferramentas eliminam classes inteiras de loops e travamentos

A nova release já está disponível em [github.com/jholhewres/devclaw](https://github.com/jholhewres/devclaw).

## A Reflexão

Existe um debate legítimo sobre ética de estudar código vazado. Minha posição: a Anthropic confirmou que foi um erro de empacotamento, não uma brecha. O código estava em um registro público (npm). E o que eu fiz não foi copiar — foi estudar **conceitos arquiteturais** que a comunidade open source já havia analisado e documentado publicamente.

É a mesma dinâmica de estudar como o Redis implementa seu event loop, ou como o Go runtime gerencia goroutines. Sistemas de produção ensinam o que papers e blog posts não conseguem: as decisões pragmáticas de quem já enfrentou os mesmos problemas em escala.

O Claude Code é um dos melhores agentes de coding que existem. O DevClaw não compete com ele — resolve problemas diferentes, para um público diferente, em Go. Mas estudar os conceitos por trás de como a Anthropic construiu seu agente me deu clareza sobre problemas que eu vinha tentando resolver há meses.

Às vezes o que falta não é conhecimento novo. É ver como alguém com mais experiência aplicou o que você já sabia.

---

**Links:**
- DevClaw: [github.com/jholhewres/devclaw](https://github.com/jholhewres/devclaw)
- Análise da comunidade: [github.com/instructkr/claw-code](https://github.com/instructkr/claw-code)
- Cobertura do vazamento: [VentureBeat](https://venturebeat.com/technology/claude-codes-source-code-appears-to-have-leaked-heres-what-we-know) · [The Register](https://www.theregister.com/2026/03/31/anthropic_claude_code_source_code/) · [DEV Community](https://dev.to/gabrielanhaia/claude-codes-entire-source-code-was-just-leaked-via-npm-source-maps-heres-whats-inside-cjo)

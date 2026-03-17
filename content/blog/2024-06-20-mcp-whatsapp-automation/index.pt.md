---
title: "MCP + WhatsApp: Construindo Automacao Inteligente"
date: 2024-06-20
tags: ["ai", "mcp", "whatsapp", "automation"]
summary: "Como usamos Model Context Protocol para construir automacao com IA no WhatsApp na Altaforja"
reading_time: 7
---

Na Altaforja Tecnologia, temos construido automacao com IA que se conecta diretamente ao WhatsApp Business. O ingrediente chave que faz tudo funcionar e o MCP — Model Context Protocol.

## O que e MCP?

MCP (Model Context Protocol) e uma forma padronizada para modelos de IA interagirem com ferramentas externas e fontes de dados. Pense nisso como uma API universal para agentes de IA — em vez de construir integracoes customizadas para cada ferramenta, voce define uma interface padrao que qualquer modelo de IA pode usar.

## A Arquitetura

Nossa stack de automacao funciona assim:

1. **WhatsGo** recebe mensagens da WhatsApp Business API
2. Mensagens sao roteadas para um orquestrador **AgentGo**
3. Agentes usam **ferramentas MCP** para acessar dados de negocios, sistemas CRM e APIs internas
4. Respostas sao geradas e enviadas de volta pelo WhatsGo

```
WhatsApp → WhatsGo → AgentGo Orchestrator → MCP Tools → Resposta
```

## Por Que MCP Importa

Antes do MCP, cada integracao era customizada. Quer conectar sua IA a um banco de dados? Escreva uma ferramenta customizada. Quer acessar um CRM? Outra integracao customizada. MCP padroniza isso:

```go
type Tool struct {
    Name        string
    Description string
    InputSchema json.RawMessage
    Handler     func(ctx context.Context, input json.RawMessage) (any, error)
}
```

Qualquer modelo de IA que fala MCP pode usar qualquer ferramenta MCP. Essa composabilidade e poderosa — voce constroi uma ferramenta uma vez e qualquer agente no seu sistema pode usa-la.

## Impacto no Mundo Real

Com essa stack, automatizamos:

- **Suporte ao cliente**: Agentes de IA que entendem contexto, acessam historico de pedidos e resolvem problemas sem intervencao humana
- **Agendamento**: Reservas em linguagem natural que verificam disponibilidade em tempo real
- **Qualificacao de leads**: Conversas inteligentes que qualificam leads e os direcionam para o time de vendas correto

Tudo rodando em um unico binario Go, lidando com milhares de conversas simultaneas.

## O Que Aprendi

1. **Latencia importa no chat**: Usuarios esperam respostas quase instantaneas. A performance do Go e critica aqui.
2. **Contexto e tudo**: Ferramentas MCP que fornecem contexto relevante aos agentes melhoram dramaticamente a qualidade das respostas.
3. **Degradacao graciosa**: Quando a IA nao esta confiante, direcione para um humano. Nunca frustre usuarios com respostas ruins de IA.

A combinacao da performance do Go, padronizacao do MCP e alcance do WhatsApp cria uma plataforma poderosa para automacao de negocios.

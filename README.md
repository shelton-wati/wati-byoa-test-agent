# wati-byoa-test-agent

Minimal standalone test agent for WATI BYOA (Bring Your Own Agent) AI Operator webhooks.

This service is intentionally small: it receives WATI webhook events, runs a simple tool-calling agent loop, and sends replies back to WhatsApp or Instagram conversations.

## Features

- HTTP webhook endpoint compatible with `whatsapp_inbox` AI Operator webhooks
- Agent loop with OpenAI-compatible chat completions + function calling
- Outbound messaging for WhatsApp and Instagram
- Simple built-in tools: current time, calculator
- Conversation context stored locally per `conversationId`
- New session on `chatAssigned`, append on subsequent customer messages
- Context truncation when estimated tokens exceed `MAX_CONTEXT_TOKENS`
- Webhook deduplication for retries

## Quick start

```bash
cp .env.example .env
# edit .env with your WATI + LLM credentials

./scripts/start.sh
```

Or manually:

```bash
go run ./cmd/wati-byoa-test-agent
```

## Webhook

- URL: `POST /wati/webhook`
- Auth: `Authorization: Bearer <WEBHOOK_TOKEN>`
- Supported events:
  - `message` / `MessageReceived`
  - `chatAssigned` / `ChatAssigned`

Point your WATI AI Operator webhook URL to this service (use ngrok/cloudflared for local testing).

## Environment

See `.env.example` for all settings.

Required:

- `WEBHOOK_TOKEN`
- `WATI_BASE_URL`
- `WATI_API_TOKEN`
- `LLM_BASE_URL` — e.g. `https://api.openai.com/v1` or `https://opencode.ai/zen/go/v1`
- `LLM_API_KEY` (or `OPENAI_API_KEY`)
- `LLM_MODEL`

## Architecture

```
WATI inbox webhook
  -> internal/webhook
  -> internal/agent (loop + context)
  -> internal/llm
  -> internal/outbound (WhatsApp / IG reply)
```

Local state under `.wati-byoa-state/`:

- `sessions/<conversationId>.json` — chat history
- `processed-event-ids.json` — webhook dedup

## Notes

- This is a temporary test agent, not production infrastructure.
- Prompts are in English; replies default to English unless the customer writes in another language.
- Derived from `slack-copilot-agent` `test-byoa-2` BYOA bridge, but fully standalone.

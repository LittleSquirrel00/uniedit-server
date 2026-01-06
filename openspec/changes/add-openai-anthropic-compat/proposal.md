# Proposal: Add OpenAI and Anthropic SDK Compatible APIs

## Summary

Add SDK-compatible API endpoints that allow users to use official OpenAI and Anthropic client libraries with UniEdit server by simply changing the base URL. This enables seamless integration with existing codebases and tooling.

## Motivation

Currently, the AI module exposes custom endpoints at `/api/v1/ai/*`:
- `POST /api/v1/ai/chat` - Non-streaming chat
- `POST /api/v1/ai/chat/stream` - Streaming chat
- `POST /api/v1/ai/embeddings` - Text embeddings

These require users to write custom integration code. By providing SDK-compatible endpoints, users can:
1. Use official SDKs (openai-python, anthropic-sdk, etc.) directly
2. Migrate existing applications with minimal code changes
3. Leverage ecosystem tooling built around these protocols

## Scope

### In Scope
- OpenAI-compatible endpoints:
  - `POST /v1/chat/completions` - Chat completions (streaming and non-streaming)
  - `POST /v1/embeddings` - Text embeddings
  - `GET /v1/models` - List available models
- Anthropic-compatible endpoints:
  - `POST /v1/messages` - Messages API (streaming and non-streaming)
- Request/response format translation to internal unified format
- Error response format translation for each protocol

### Out of Scope
- Multi-port service configuration (separate proposal)
- OpenAI Assistant API, Batch API, Fine-tuning API
- Anthropic Tool Use extended features
- Rate limiting per API key (uses existing auth system)

## Related Specs
- `ai-llm-service` - Existing LLM service specification

## Risks
- **Breaking Changes**: None - this is an additive feature
- **Maintenance**: Two format translation layers to maintain
- **Complexity**: Anthropic streaming format differs significantly from OpenAI

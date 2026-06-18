# Fuse

Set a fuse on your AI spending.

Fuse is a lightweight local proxy that sits between your AI coding agent and provider APIs, tracks spend in SQLite, and blocks requests when configured hard caps would be exceeded.

## Quick Start

```bash
# Install from source
go install github.com/hashir500/Fuse@latest

# Set up in a project directory
fuse init

# Edit fuse.yml with API keys and budgets
$EDITOR fuse.yml

# Start the proxy
fuse proxy
```

Point your AI tool at:

```text
http://localhost:8787
```

## Commands

```bash
fuse init
fuse proxy --addr localhost:8787
fuse status
fuse history --limit 5
fuse config
fuse config validate
```

## Supported Providers

Fuse v0.1 supports:

- Anthropic Messages API routes under `/v1/messages`
- OpenAI Chat Completions and Responses routes under `/v1/chat/completions` and `/v1/responses`
- Google Gemini routes under `/v1beta/models/...` and `/v1/models/...`

Configure API keys through environment-backed values in `fuse.yml`:

```yaml
providers:
  anthropic:
    api_key: "${ANTHROPIC_API_KEY}"
  openai:
    api_key: "${OPENAI_API_KEY}"
  google:
    api_key: "${GEMINI_API_KEY}"
```

## Budget Behavior

Fuse estimates request cost before sending to the provider so hard caps can be enforced before money is spent. Allowed requests are logged again using provider response usage fields where available.

If a hard cap would be exceeded, Fuse returns HTTP 429:

```json
{
  "error": "fuse_hard_cap_exceeded",
  "message": "Daily hard cap of $100.00 exceeded. Current: $98.50, Request would cost: $3.20.",
  "cap_type": "daily",
  "cap_amount": 100.0,
  "current_spend": 98.5
}
```

Soft caps print warnings to stderr and continue allowing traffic.

## Claude Code

```bash
fuse proxy
claude config set apiBaseUrl http://localhost:8787
```

## Cursor and OpenAI-Compatible Tools

Set the API base URL to:

```text
http://localhost:8787
```

Keep provider API keys in your shell environment. Fuse forwards requests to the configured provider base URL and injects the provider API key from `fuse.yml`.

## Development

```bash
go mod tidy
go test ./...
go build ./...
```

This repository uses `modernc.org/sqlite`, so SQLite works without CGO.

# Antigravity Wrapper

A standalone API proxy for the Antigravity (Google Gemini CLI) backend, providing OpenAI and Claude API-compatible endpoints.

## Features

- **OpenAI-Compatible API**: Full support for `/v1/chat/completions` endpoint
- **Claude-Compatible API**: Full support for `/v1/messages` endpoint  
- **OpenAI Responses API**: Support for `/v1/responses` endpoint
- **Reasoning/Thinking Support**: Full `reasoning_content` and thinking budget support
- **Streaming**: Server-Sent Events (SSE) streaming for all endpoints
- **Tool/Function Calling**: Complete function calling support
- **Image Generation**: Support for image modalities
- **OAuth Authentication**: Google OAuth 2.0 login with automatic token refresh
- **Proxy Support**: HTTP, HTTPS, and SOCKS5 proxy configuration

## Quick Start

### 1. Build

```bash
cd antigravity-wrapper
go build -o antigravity-wrapper ./cmd/server
```

### 2. Login

Authenticate with your Google account:

```bash
./antigravity-wrapper -login
```

This will open a browser for OAuth authentication. Your credentials will be saved to `~/.antigravity/`.

### 3. Run the Server

```bash
./antigravity-wrapper
```

The server starts on `http://localhost:8080` by default.

### 4. Make Requests

**OpenAI Chat Completions:**

```bash
curl http://localhost:8080/v1/chat/completions \
  -H "Content-Type: application/json" \
  -d '{
    "model": "gemini-2.5-flash",
    "messages": [{"role": "user", "content": "Hello!"}],
    "stream": true
  }'
```

**Claude Messages API:**

```bash
curl http://localhost:8080/v1/messages \
  -H "Content-Type: application/json" \
  -d '{
    "model": "gemini-2.5-flash",
    "messages": [{"role": "user", "content": [{"type": "text", "text": "Hello!"}]}],
    "max_tokens": 1024,
    "stream": true
  }'
```

## Configuration

### Command Line Flags

| Flag | Description | Default |
|------|-------------|---------|
| `-config` | Path to YAML config file | (none) |
| `-port` | Server port | 8080 |
| `-host` | Server host | 0.0.0.0 |
| `-debug` | Enable debug logging | false |
| `-login` | Run OAuth login flow | false |

### Configuration File

Create a `config.yaml` file:

```yaml
# Server settings
port: 8080
host: "0.0.0.0"

# API key authentication (optional)
api_keys:
  - "sk-your-api-key-1"
  - "sk-your-api-key-2"

# Proxy settings (optional)
proxy_url: "socks5://127.0.0.1:1080"

# Credentials directory
credentials_dir: "~/.antigravity"

# Logging
log_level: "info"
debug: false
```

### Environment Variables

| Variable | Description |
|----------|-------------|
| `ANTIGRAVITY_PORT` | Server port |
| `ANTIGRAVITY_HOST` | Server host |
| `ANTIGRAVITY_PROXY_URL` | Proxy URL |
| `ANTIGRAVITY_CREDENTIALS_DIR` | Credentials directory |
| `ANTIGRAVITY_API_KEYS` | Comma-separated API keys |
| `ANTIGRAVITY_LOG_LEVEL` | Log level (debug, info, warn, error) |
| `ANTIGRAVITY_DEBUG` | Enable debug mode (true/1) |

## Supported Models

| Model ID | Description | Thinking Support |
|----------|-------------|------------------|
| `gemini-2.5-flash` | Fast, efficient model | ✅ (0-24576 tokens) |
| `gemini-2.5-flash-lite` | Lightweight model | ✅ (0-24576 tokens) |
| `gemini-3-pro-preview` | Advanced model | ✅ (128-32768 tokens) |
| `gemini-3-pro-image-preview` | Image generation | ✅ (128-32768 tokens) |
| `gemini-claude-sonnet-4-5-thinking` | Claude via Gemini | ✅ (1024-200000 tokens) |
| `gemini-claude-opus-4-5-thinking` | Claude Opus via Gemini | ✅ (1024-200000 tokens) |

## Reasoning/Thinking Support

### OpenAI Format

Use `reasoning_effort` parameter:

```json
{
  "model": "gemini-2.5-flash",
  "messages": [...],
  "reasoning_effort": "high"
}
```

Supported levels: `none`, `auto`, `minimal`, `low`, `medium`, `high`, `xhigh`

### Claude Format

Use `thinking` parameter:

```json
{
  "model": "gemini-2.5-flash",
  "messages": [...],
  "thinking": {
    "type": "enabled",
    "budget_tokens": 8192
  }
}
```

### Cherry Studio Extension

Use `extra_body.google.thinking_config`:

```json
{
  "model": "gemini-2.5-flash",
  "messages": [...],
  "extra_body": {
    "google": {
      "thinking_config": {
        "thinking_budget": 8192,
        "include_thoughts": true
      }
    }
  }
}
```

## Response Format

### Streaming Responses

The `reasoning_content` field contains the model's thinking process:

```json
{
  "choices": [{
    "delta": {
      "role": "assistant",
      "content": "The answer is...",
      "reasoning_content": "Let me think about this..."
    }
  }]
}
```

### Non-Streaming Responses

```json
{
  "choices": [{
    "message": {
      "role": "assistant",
      "content": "The answer is...",
      "reasoning_content": "Let me think about this..."
    }
  }],
  "usage": {
    "prompt_tokens": 100,
    "completion_tokens": 200,
    "total_tokens": 300,
    "completion_tokens_details": {
      "reasoning_tokens": 50
    }
  }
}
```

## Docker

### Build

```bash
docker build -t antigravity-wrapper .
```

### Run

```bash
# First, login on host to get credentials
./antigravity-wrapper -login

# Run with mounted credentials
docker run -p 8080:8080 \
  -v ~/.antigravity:/root/.antigravity \
  antigravity-wrapper
```

## API Endpoints

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/health` | GET | Health check |
| `/v1/models` | GET | List available models |
| `/v1/chat/completions` | POST | OpenAI Chat Completions |
| `/v1/messages` | POST | Claude Messages API |
| `/v1/responses` | POST | OpenAI Responses API |

## License

MIT License - See LICENSE file for details.
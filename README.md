# tg-bot-demo

Telegram Bot webhook server with session management, written in Go.
Uses [`github.com/go-telegram/bot`](https://github.com/go-telegram/bot) to process webhook updates.

## Features

- **Session Management**: Create, list, and switch between conversation sessions
- **Pagination**: Browse through sessions with inline keyboard pagination
- **Flexible Configuration**: Support for config files, environment variables, and command-line flags
- **SQLite Storage**: Persistent session storage with ACID guarantees
- **File Downloads**: Automatic download of media files from messages

## Quick Start

```bash
# Using environment variable for token
export TELEGRAM_BOT_TOKEN="123456:your-bot-token"
go run .

# Or using command-line flags
go run . -token "123456:your-bot-token" -listen ":3000"

# Or using a config file
go run . -config config.json
```

## Configuration

The bot supports three configuration methods (in order of precedence):

1. **Configuration file** (JSON format)
2. **Environment variables**
3. **Command-line flags**

See [Configuration Guide](docs/configuration.md) for detailed documentation.

### Quick Configuration Reference

| Option | Env Variable | Flag | Default |
|--------|-------------|------|---------|
| Bot Token | `TELEGRAM_BOT_TOKEN` | `-token` | (required) |
| Listen Address | `LISTEN_ADDR` | `-listen` | `:3000` |
| Webhook Path | `WEBHOOK_PATH` | `-path` | `/webhook` |
| Sessions Per Page | `SESSIONS_PER_PAGE` | `-sessions-per-page` | `6` |
| Database Path | `DATABASE_PATH` | `-db` | `./data/sessions.db` |

Example config file (`config.json`):

```json
{
  "token": "123456:your-bot-token",
  "listen_addr": ":3000",
  "webhook_path": "/webhook",
  "sessions_per_page": 6,
  "database_path": "./data/sessions.db"
}
```

## Session Management

The bot provides session management features for organizing conversations:

- **/sessions** - List your conversation sessions
- **/open** - Open a new session and make it active
- **/close** - Close the current active session (history is kept)
- Click a session to switch to it
- Use "↑ Prev" (top) / "↓ Next" (bottom) buttons to navigate pages
- New messages automatically create or use the active session

See [Session Documentation](docs/sessions.md) for more details.

## Run

### Basic Usage

```bash
# Using environment variable
export TELEGRAM_BOT_TOKEN="123456:your-bot-token"
go run .

# Using command-line flags
go run . -token "123456:your-bot-token" -listen ":3000" -path "/webhook"

# Using config file
go run . -config config.json
```

### Command-Line Flags

- `-config`: Path to JSON configuration file (optional)
- `-listen`: HTTP listen address (default: `:3000`)
- `-path`: Webhook path (default: `/webhook`)
- `-token`: Telegram bot token (or set `TELEGRAM_BOT_TOKEN` env var)
- `-secret-token`: Optional webhook secret token for validation
- `-status`: Default HTTP response status code (default: `200`)
- `-db`: Path to SQLite database file (default: `./data/sessions.db`)
- `-sessions-per-page`: Number of sessions per page (default: `6`)

Flags override config file values, and environment variables override both.

## Behavior

- Passes webhook request to `go-telegram/bot` webhook handler.
- Replies `OK` to incoming user messages.
- Prints request details as JSON (2-space indentation) to stdout, including:
  - method / URI / protocol / remote address
  - all HTTP headers
  - request body (auto-parsed as JSON when possible)
  - response status code
- If update contains file media (document/photo/audio/video/voice/video_note/sticker/animation), downloads file to `download/{username}/{file_id}`.
- Returns the configured status code.
- You can override status per request via query parameter `status`.

Examples:

```bash
# Use default status from -status
curl -i -X POST "http://localhost:3000/webhook" \
  -H "Content-Type: application/json" \
  -d '{"update_id":123}'

# Override response status to 500 for retry testing
curl -i -X POST "http://localhost:3000/webhook?status=500" \
  -H "Content-Type: application/json" \
  -d '{"update_id":123}'
```

# tg-bot-demo

Simple Telegram Bot webhook debug server written in Go.
Uses [`github.com/go-telegram/bot`](https://github.com/go-telegram/bot) to process webhook updates.

## Run

```bash
go run . -listen ":3000" -path "/webhook" -status 200 \
  -token "123456:your-bot-token" \
  -secret-token ""
```

Flags:
- `-listen`: HTTP listen address, default `:3000`
- `-path`: webhook path, default `/webhook`
- `-token`: Telegram bot token. Default reads `TELEGRAM_BOT_TOKEN`, fallback is a debug token.
- `-secret-token`: optional secret token for `X-Telegram-Bot-Api-Secret-Token` validation.
- `-status`: default response status code, default `200`

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

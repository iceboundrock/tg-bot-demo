# tg-bot-demo

Simple Telegram Bot webhook debug server written in Go.

## Run

```bash
go run . -listen ":8080" -path "/webhook" -status 200
```

Flags:
- `-listen`: HTTP listen address, default `:8080`
- `-path`: webhook path, default `/webhook`
- `-status`: default response status code, default `200`

## Behavior

- Prints all incoming request headers and body to stdout.
- Returns the configured status code.
- You can override status per request via query parameter `status`.

Examples:

```bash
# Use default status from -status
curl -i -X POST "http://localhost:8080/webhook" \
  -H "Content-Type: application/json" \
  -d '{"update_id":123}'

# Override response status to 500 for retry testing
curl -i -X POST "http://localhost:8080/webhook?status=500" \
  -H "Content-Type: application/json" \
  -d '{"update_id":123}'
```

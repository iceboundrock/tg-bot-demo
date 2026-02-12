# Configuration Guide

The Telegram bot supports flexible configuration through environment variables, configuration files, and command-line flags.

## Configuration Priority

Configuration values are loaded in the following order (later sources override earlier ones):

1. Default values
2. Configuration file (if provided)
3. Environment variables
4. Command-line flags

## Configuration Options

### Bot Configuration

- **token** (required): Telegram bot token
  - Environment: `TELEGRAM_BOT_TOKEN`
  - Flag: `-token`
  - Example: `123456:ABC-DEF1234ghIkl-zyx57W2v1u123ew11`

- **secret_token** (optional): Webhook secret token for additional security
  - Environment: `TELEGRAM_SECRET_TOKEN`
  - Flag: `-secret-token`
  - Example: `my-secret-token-123`

### Server Configuration

- **listen_addr**: HTTP server listen address
  - Environment: `LISTEN_ADDR`
  - Flag: `-listen`
  - Default: `:3000`
  - Example: `:8080`, `0.0.0.0:3000`

- **webhook_path**: Webhook endpoint path
  - Environment: `WEBHOOK_PATH`
  - Flag: `-path`
  - Default: `/webhook`
  - Example: `/telegram-webhook`

- **default_status**: Default HTTP status code for webhook responses
  - Environment: `DEFAULT_STATUS`
  - Flag: `-status`
  - Default: `200`
  - Valid range: 100-599

### Session Configuration

- **sessions_per_page**: Number of sessions to display per page
  - Environment: `SESSIONS_PER_PAGE`
  - Flag: `-sessions-per-page`
  - Default: `6`
  - Minimum: `1`

- **database_path**: Path to SQLite database file
  - Environment: `DATABASE_PATH`
  - Flag: `-db`
  - Default: `./data/sessions.db`
  - Example: `/var/lib/telegram-bot/sessions.db`

## Usage Examples

### Using Environment Variables

```bash
export TELEGRAM_BOT_TOKEN="123456:ABC-DEF1234ghIkl-zyx57W2v1u123ew11"
export LISTEN_ADDR=":8080"
export SESSIONS_PER_PAGE="10"
export DATABASE_PATH="/var/lib/bot/sessions.db"

./tg-bot-demo
```

### Using Configuration File

Create a `config.json` file:

```json
{
  "token": "123456:ABC-DEF1234ghIkl-zyx57W2v1u123ew11",
  "secret_token": "my-secret-token",
  "listen_addr": ":8080",
  "webhook_path": "/telegram-webhook",
  "default_status": 200,
  "sessions_per_page": 10,
  "database_path": "/var/lib/bot/sessions.db"
}
```

Run with config file:

```bash
./tg-bot-demo -config config.json
```

### Using Command-Line Flags

```bash
./tg-bot-demo \
  -token "123456:ABC-DEF1234ghIkl-zyx57W2v1u123ew11" \
  -listen ":8080" \
  -path "/telegram-webhook" \
  -sessions-per-page 10 \
  -db "/var/lib/bot/sessions.db"
```

### Combining Methods

You can combine all three methods. For example, use a config file for base settings, environment variables for secrets, and flags for overrides:

```bash
# config.json contains base settings
# Environment variable for token (more secure)
export TELEGRAM_BOT_TOKEN="123456:ABC-DEF1234ghIkl-zyx57W2v1u123ew11"

# Override listen address with flag
./tg-bot-demo -config config.json -listen ":9000"
```

## Configuration File Format

The configuration file must be valid JSON. See `config.example.json` for a complete example:

```json
{
  "token": "your-telegram-bot-token-here",
  "secret_token": "optional-webhook-secret-token",
  "listen_addr": ":3000",
  "webhook_path": "/webhook",
  "default_status": 200,
  "sessions_per_page": 6,
  "database_path": "./data/sessions.db"
}
```

## Validation

The bot validates configuration on startup and will exit with an error if:

- Bot token is missing or empty
- Default status is outside the range 100-599
- Sessions per page is less than 1
- Database path is empty

## Security Best Practices

1. **Never commit tokens to version control**: Use environment variables or secure secret management
2. **Use webhook secret tokens**: Add an extra layer of security with `secret_token`
3. **Restrict file permissions**: Ensure config files with tokens have restricted permissions (e.g., `chmod 600 config.json`)
4. **Use environment-specific configs**: Maintain separate config files for development, staging, and production

## Docker Configuration

When running in Docker, use environment variables or mount a config file:

```dockerfile
# Using environment variables
docker run -e TELEGRAM_BOT_TOKEN="your-token" \
           -e DATABASE_PATH="/data/sessions.db" \
           -v /host/data:/data \
           telegram-bot

# Using config file
docker run -v /host/config.json:/app/config.json \
           -v /host/data:/data \
           telegram-bot -config /app/config.json
```

## Troubleshooting

### "bot token is required" error

Ensure you've set the token via one of these methods:
- Environment variable: `TELEGRAM_BOT_TOKEN`
- Config file: `"token": "..."`
- Command-line flag: `-token "..."`

### "invalid configuration: default_status must be between 100 and 599"

Check that your `DEFAULT_STATUS` environment variable or `default_status` config value is a valid HTTP status code.

### Database connection issues

Ensure:
1. The database directory exists or the bot has permission to create it
2. The bot has read/write permissions for the database file
3. The database path is absolute or relative to the working directory

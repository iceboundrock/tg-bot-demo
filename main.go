package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"tg-bot-demo/config"
	"tg-bot-demo/handlers"
	"tg-bot-demo/session"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

// initializeBot creates and configures a bot with session management
func initializeBot(cfg *config.Config) (*bot.Bot, *session.SQLiteStore, error) {
	// Initialize SQLite store with database path
	store, err := session.NewSQLiteStore(cfg.DatabasePath)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create session store: %w", err)
	}

	// Create session manager with store
	sessionMgr := session.NewManager(store)

	// Create handler config
	handlerCfg := &handlers.HandlerConfig{
		SessionsPerPage: cfg.SessionsPerPage,
	}

	// Create bot with handlers
	tgBot, err := bot.New(
		cfg.Token,
		bot.WithSkipGetMe(),
		bot.WithDefaultHandler(handleUpdate),
		bot.WithWebhookSecretToken(cfg.SecretToken),
	)
	if err != nil {
		store.Close()
		return nil, nil, fmt.Errorf("failed to create telegram bot: %w", err)
	}

	// Register command handler for /sessions
	tgBot.RegisterHandler(bot.HandlerTypeMessageText, "/sessions", bot.MatchTypeExact,
		handlers.SessionsCommandHandler(sessionMgr, handlerCfg))

	// Register command handler for /open
	tgBot.RegisterHandler(bot.HandlerTypeMessageText, "/open", bot.MatchTypeExact,
		handlers.OpenCommandHandler(sessionMgr))

	// Register command handler for /close
	tgBot.RegisterHandler(bot.HandlerTypeMessageText, "/close", bot.MatchTypeExact,
		handlers.CloseCommandHandler(sessionMgr))

	// Register callback query handler
	tgBot.RegisterHandler(bot.HandlerTypeCallbackQueryData, "", bot.MatchTypePrefix,
		handlers.CallbackQueryHandler(sessionMgr, handlerCfg))

	// Register message handler for regular text messages (non-commands)
	// This will handle messages that don't match other handlers
	tgBot.RegisterHandler(bot.HandlerTypeMessageText, "", bot.MatchTypePrefix,
		handlers.MessageHandler(sessionMgr))

	return tgBot, store, nil
}

func main() {
	// Define command-line flags
	configPath := flag.String("config", "", "Path to config file (optional)")
	listenAddr := flag.String("listen", "", "HTTP listen address (overrides config)")
	path := flag.String("path", "", "Webhook path (overrides config)")
	token := flag.String("token", "", "Telegram bot token (overrides config)")
	secretToken := flag.String("secret-token", "", "Webhook secret token (overrides config)")
	defaultStatus := flag.Int("status", 0, "Default HTTP status code (overrides config)")
	dbPath := flag.String("db", "", "Path to SQLite database file (overrides config)")
	sessionsPerPage := flag.Int("sessions-per-page", 0, "Sessions per page (overrides config)")
	flag.Parse()

	// Load configuration
	cfg, err := config.Load(*configPath)
	if err != nil {
		log.Fatalf("failed to load configuration: %v", err)
	}

	// Override config with command-line flags if provided
	if *listenAddr != "" {
		cfg.ListenAddr = *listenAddr
	}
	if *path != "" {
		cfg.WebhookPath = *path
	}
	if *token != "" {
		cfg.Token = *token
	}
	if *secretToken != "" {
		cfg.SecretToken = *secretToken
	}
	if *defaultStatus != 0 {
		cfg.DefaultStatus = *defaultStatus
	}
	if *dbPath != "" {
		cfg.DatabasePath = *dbPath
	}
	if *sessionsPerPage != 0 {
		cfg.SessionsPerPage = *sessionsPerPage
	}

	// Validate final configuration
	if err := cfg.Validate(); err != nil {
		log.Fatalf("invalid configuration: %v", err)
	}

	// Ensure database directory exists
	dbDir := filepath.Dir(cfg.DatabasePath)
	if err := os.MkdirAll(dbDir, 0o755); err != nil {
		log.Fatalf("failed to create database directory: %v", err)
	}

	// Initialize bot with session management
	tgBot, store, err := initializeBot(cfg)
	if err != nil {
		log.Fatalf("initialize bot: %v", err)
	}
	defer store.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go tgBot.StartWebhook(ctx)

	tgWebhookHandler := tgBot.WebhookHandler()

	mux := http.NewServeMux()
	mux.HandleFunc(cfg.WebhookPath, webhookHandler(tgWebhookHandler, cfg.DefaultStatus))

	server := &http.Server{
		Addr:              cfg.ListenAddr,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}

	log.Printf("webhook server started: listen=%s path=%s default_status=%d sessions_per_page=%d",
		cfg.ListenAddr, cfg.WebhookPath, cfg.DefaultStatus, cfg.SessionsPerPage)
	log.Fatal(server.ListenAndServe())
}

func webhookHandler(tgHandler http.HandlerFunc, defaultStatus int) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			log.Printf("read body error: %v", err)
			http.Error(w, "failed to read body", http.StatusBadRequest)
			return
		}
		defer r.Body.Close()

		status := resolveStatus(defaultStatus, r.URL.Query().Get("status"))
		requestID := time.Now().Format("20060102-150405.000000")
		logRequest(requestID, r, body, status)

		r.Body = io.NopCloser(bytes.NewReader(body))
		tgHandler(newDiscardResponseWriter(), r)

		w.WriteHeader(status)
		_, _ = w.Write([]byte(fmt.Sprintf("status=%d\n", status)))
	}
}

func resolveStatus(defaultStatus int, raw string) int {
	if raw == "" {
		return defaultStatus
	}
	parsed, err := strconv.Atoi(raw)
	if err != nil || parsed < 100 || parsed > 599 {
		return defaultStatus
	}
	return parsed
}

type requestLog struct {
	RequestID     string         `json:"request_id"`
	ReceivedAt    string         `json:"received_at"`
	Method        string         `json:"method"`
	RequestURI    string         `json:"request_uri"`
	Proto         string         `json:"proto"`
	RemoteAddr    string         `json:"remote_addr"`
	ContentLength int            `json:"content_length"`
	Headers       []headerRecord `json:"headers"`
	Body          any            `json:"body"`
	ResponseCode  int            `json:"response_code"`
}

type headerRecord struct {
	Name   string   `json:"name"`
	Values []string `json:"values"`
}

func logRequest(requestID string, r *http.Request, body []byte, status int) {
	logItem := requestLog{
		RequestID:     requestID,
		ReceivedAt:    time.Now().Format(time.RFC3339Nano),
		Method:        r.Method,
		RequestURI:    r.URL.RequestURI(),
		Proto:         r.Proto,
		RemoteAddr:    r.RemoteAddr,
		ContentLength: len(body),
		Headers:       collectHeaders(r.Header),
		Body:          parseBody(body),
		ResponseCode:  status,
	}

	pretty, err := json.MarshalIndent(logItem, "", "  ")
	if err != nil {
		log.Printf("marshal request log error: %v", err)
		return
	}
	fmt.Println(string(pretty))
}

func collectHeaders(headers http.Header) []headerRecord {
	keys := make([]string, 0, len(headers))
	for key := range headers {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	result := make([]headerRecord, 0, len(keys))
	for _, key := range keys {
		result = append(result, headerRecord{
			Name:   key,
			Values: headers[key],
		})
	}
	return result
}

func parseBody(body []byte) any {
	trimmed := bytes.TrimSpace(body)
	if len(trimmed) == 0 {
		return ""
	}

	var decoded any
	if err := json.Unmarshal(trimmed, &decoded); err == nil {
		return decoded
	}

	return string(body)
}

type discardResponseWriter struct {
	headers http.Header
}

func newDiscardResponseWriter() *discardResponseWriter {
	return &discardResponseWriter{
		headers: make(http.Header),
	}
}

func (d *discardResponseWriter) Header() http.Header {
	return d.headers
}

func (d *discardResponseWriter) Write(p []byte) (int, error) {
	return len(p), nil
}

func (d *discardResponseWriter) WriteHeader(int) {}

type fileTarget struct {
	Kind   string
	FileID string
}

func handleUpdate(ctx context.Context, b *bot.Bot, update *models.Update) {
	if incoming := incomingUserMessageFromUpdate(update); shouldReplyOK(incoming) {
		if _, err := b.SendMessage(ctx, buildOKReply(incoming)); err != nil {
			log.Printf("reply failed: chat_id=%v message_id=%d err=%v", incoming.Chat.ID, incoming.ID, err)
		}
	}

	message := messageFromUpdate(update)
	if message == nil {
		return
	}

	targets := collectFileTargets(message)
	if len(targets) == 0 {
		return
	}

	username := messageUsername(message)
	for _, target := range targets {
		outputPath, size, err := downloadTelegramFile(ctx, b, username, target.FileID)
		if err != nil {
			log.Printf("download failed: type=%s username=%s file_id=%s err=%v", target.Kind, username, target.FileID, err)
			continue
		}
		log.Printf("downloaded: type=%s username=%s file_id=%s bytes=%d path=%s", target.Kind, username, target.FileID, size, outputPath)
	}
}

func incomingUserMessageFromUpdate(update *models.Update) *models.Message {
	switch {
	case update.Message != nil:
		return update.Message
	case update.BusinessMessage != nil:
		return update.BusinessMessage
	default:
		return nil
	}
}

func shouldReplyOK(message *models.Message) bool {
	if message == nil {
		return false
	}
	if message.From != nil && message.From.IsBot {
		return false
	}
	if message.Chat.ID == 0 {
		return false
	}
	return true
}

func buildOKReply(message *models.Message) *bot.SendMessageParams {
	params := &bot.SendMessageParams{
		ChatID: message.Chat.ID,
		Text:   "OK",
		ReplyParameters: &models.ReplyParameters{
			MessageID:                message.ID,
			AllowSendingWithoutReply: true,
		},
	}
	if message.MessageThreadID != 0 {
		params.MessageThreadID = message.MessageThreadID
	}
	if message.DirectMessagesTopic != nil {
		params.DirectMessagesTopicID = message.DirectMessagesTopic.TopicID
	}
	return params
}

func messageFromUpdate(update *models.Update) *models.Message {
	switch {
	case update.Message != nil:
		return update.Message
	case update.EditedMessage != nil:
		return update.EditedMessage
	case update.ChannelPost != nil:
		return update.ChannelPost
	case update.EditedChannelPost != nil:
		return update.EditedChannelPost
	case update.BusinessMessage != nil:
		return update.BusinessMessage
	case update.EditedBusinessMessage != nil:
		return update.EditedBusinessMessage
	default:
		return nil
	}
}

func collectFileTargets(message *models.Message) []fileTarget {
	targets := make([]fileTarget, 0, 8)
	seen := make(map[string]struct{})

	add := func(kind, fileID string) {
		if fileID == "" {
			return
		}
		if _, ok := seen[fileID]; ok {
			return
		}
		seen[fileID] = struct{}{}
		targets = append(targets, fileTarget{
			Kind:   kind,
			FileID: fileID,
		})
	}

	if message.Document != nil {
		add("document", message.Document.FileID)
	}
	if message.Animation != nil {
		add("animation", message.Animation.FileID)
	}
	if message.Audio != nil {
		add("audio", message.Audio.FileID)
	}
	if message.Video != nil {
		add("video", message.Video.FileID)
	}
	if message.VideoNote != nil {
		add("video_note", message.VideoNote.FileID)
	}
	if message.Voice != nil {
		add("voice", message.Voice.FileID)
	}
	if message.Sticker != nil {
		add("sticker", message.Sticker.FileID)
	}
	if photo := largestPhoto(message.Photo); photo != nil {
		add("photo", photo.FileID)
	}

	return targets
}

func largestPhoto(photos []models.PhotoSize) *models.PhotoSize {
	if len(photos) == 0 {
		return nil
	}

	bestIndex := 0
	for i := 1; i < len(photos); i++ {
		best := photos[bestIndex]
		current := photos[i]

		if current.FileSize > best.FileSize {
			bestIndex = i
			continue
		}
		if current.FileSize == best.FileSize && current.Width*current.Height > best.Width*best.Height {
			bestIndex = i
		}
	}

	return &photos[bestIndex]
}

func messageUsername(message *models.Message) string {
	if message.From != nil {
		if message.From.Username != "" {
			return message.From.Username
		}
		if message.From.ID != 0 {
			return fmt.Sprintf("user_%d", message.From.ID)
		}
	}
	if message.SenderChat != nil {
		if message.SenderChat.Username != "" {
			return message.SenderChat.Username
		}
		if message.SenderChat.ID != 0 {
			return fmt.Sprintf("chat_%d", message.SenderChat.ID)
		}
	}
	return "unknown"
}

func downloadTelegramFile(ctx context.Context, b *bot.Bot, username, fileID string) (string, int64, error) {
	fileInfo, err := b.GetFile(ctx, &bot.GetFileParams{
		FileID: fileID,
	})
	if err != nil {
		return "", 0, fmt.Errorf("call getFile: %w", err)
	}
	if fileInfo.FilePath == "" {
		return "", 0, fmt.Errorf("empty file_path from getFile")
	}

	downloadURL := b.FileDownloadLink(fileInfo)
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, downloadURL, nil)
	if err != nil {
		return "", 0, fmt.Errorf("create download request: %w", err)
	}

	response, err := http.DefaultClient.Do(request)
	if err != nil {
		return "", 0, fmt.Errorf("download file: %w", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return "", 0, fmt.Errorf("download file status: %d", response.StatusCode)
	}

	safeUsername := sanitizePathSegment(username, "unknown")
	safeFileID := sanitizePathSegment(fileID, "file")
	outputPath := filepath.Join("download", safeUsername, safeFileID)

	if err := os.MkdirAll(filepath.Dir(outputPath), 0o755); err != nil {
		return "", 0, fmt.Errorf("create output directory: %w", err)
	}

	file, err := os.Create(outputPath)
	if err != nil {
		return "", 0, fmt.Errorf("create output file: %w", err)
	}
	defer file.Close()

	written, err := io.Copy(file, response.Body)
	if err != nil {
		return "", 0, fmt.Errorf("write output file: %w", err)
	}

	return outputPath, written, nil
}

func sanitizePathSegment(raw, fallback string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return fallback
	}

	var sb strings.Builder
	sb.Grow(len(raw))
	for _, ch := range raw {
		if ch >= 'a' && ch <= 'z' || ch >= 'A' && ch <= 'Z' || ch >= '0' && ch <= '9' || ch == '.' || ch == '_' || ch == '-' {
			sb.WriteRune(ch)
			continue
		}
		sb.WriteByte('_')
	}

	result := strings.Trim(sb.String(), "._")
	if result == "" {
		return fallback
	}
	return result
}

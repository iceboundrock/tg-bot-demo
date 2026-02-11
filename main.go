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

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

func main() {
	defaultToken := os.Getenv("TELEGRAM_BOT_TOKEN")
	if defaultToken == "" {
		defaultToken = "123456:debug-token"
	}

	listenAddr := flag.String("listen", ":3000", "HTTP listen address")
	path := flag.String("path", "/webhook", "Webhook path")
	token := flag.String("token", defaultToken, "Telegram bot token (or TELEGRAM_BOT_TOKEN env)")
	secretToken := flag.String("secret-token", "", "Optional webhook secret token expected in X-Telegram-Bot-Api-Secret-Token")
	defaultStatus := flag.Int("status", http.StatusOK, "Default HTTP status code returned to webhook requests")
	flag.Parse()

	if *defaultStatus < 100 || *defaultStatus > 599 {
		log.Fatalf("invalid -status: %d, must be between 100 and 599", *defaultStatus)
	}

	tgBot, err := bot.New(
		*token,
		bot.WithSkipGetMe(),
		bot.WithDefaultHandler(handleUpdate),
		bot.WithWebhookSecretToken(*secretToken),
	)
	if err != nil {
		log.Fatalf("create telegram bot: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go tgBot.StartWebhook(ctx)

	tgWebhookHandler := tgBot.WebhookHandler()

	mux := http.NewServeMux()
	mux.HandleFunc(*path, webhookHandler(tgWebhookHandler, *defaultStatus))

	server := &http.Server{
		Addr:              *listenAddr,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}

	log.Printf("webhook server started: listen=%s path=%s default_status=%d", *listenAddr, *path, *defaultStatus)
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

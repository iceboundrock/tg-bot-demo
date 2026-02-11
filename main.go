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
	"sort"
	"strconv"
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
		bot.WithDefaultHandler(func(context.Context, *bot.Bot, *models.Update) {}),
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

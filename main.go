package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"
)

func main() {
	listenAddr := flag.String("listen", ":8080", "HTTP listen address")
	path := flag.String("path", "/webhook", "Webhook path")
	defaultStatus := flag.Int("status", http.StatusOK, "Default HTTP status code returned to webhook requests")
	flag.Parse()

	if *defaultStatus < 100 || *defaultStatus > 599 {
		log.Fatalf("invalid -status: %d, must be between 100 and 599", *defaultStatus)
	}

	mux := http.NewServeMux()
	mux.HandleFunc(*path, webhookHandler(*defaultStatus))

	server := &http.Server{
		Addr:              *listenAddr,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}

	log.Printf("webhook server started: listen=%s path=%s default_status=%d", *listenAddr, *path, *defaultStatus)
	log.Fatal(server.ListenAndServe())
}

func webhookHandler(defaultStatus int) http.HandlerFunc {
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

		log.Printf("[%s] %s %s %s remote=%s content_length=%d", requestID, r.Method, r.URL.RequestURI(), r.Proto, r.RemoteAddr, len(body))
		logHeaders(requestID, r.Header)
		log.Printf("[%s] body:\n%s", requestID, bodyToString(body))
		log.Printf("[%s] response_status=%d", requestID, status)

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

func logHeaders(requestID string, headers http.Header) {
	keys := make([]string, 0, len(headers))
	for key := range headers {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	if len(keys) == 0 {
		log.Printf("[%s] headers: <empty>", requestID)
		return
	}

	log.Printf("[%s] headers:", requestID)
	for _, key := range keys {
		log.Printf("[%s]   %s: %s", requestID, key, strings.Join(headers[key], ", "))
	}
}

func bodyToString(body []byte) string {
	if len(body) == 0 {
		return "<empty>"
	}
	return string(body)
}

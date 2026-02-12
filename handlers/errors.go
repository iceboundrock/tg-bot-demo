package handlers

import (
	"context"
	"errors"
	"log"
	"tg-bot-demo/session"

	"github.com/go-telegram/bot"
)

// ErrorResponse represents a user-facing error with a code
type ErrorResponse struct {
	Message string
	Code    string
}

// Common error responses as defined in the design document
var (
	ErrResponseNotFound = ErrorResponse{
		Message: "Session not found. It may have been deleted.",
		Code:    "SESSION_NOT_FOUND",
	}

	ErrResponseUnauthorized = ErrorResponse{
		Message: "You don't have permission to access this session.",
		Code:    "UNAUTHORIZED",
	}

	ErrResponseGeneric = ErrorResponse{
		Message: "An error occurred. Please try again.",
		Code:    "INTERNAL_ERROR",
	}

	ErrResponseInvalidCallback = ErrorResponse{
		Message: "", // No user-facing message for invalid callbacks
		Code:    "INVALID_CALLBACK",
	}
)

// SendErrorResponse sends an error message to the user based on the error type
func SendErrorResponse(ctx context.Context, b *bot.Bot, chatID int64, err error) {
	var response ErrorResponse

	switch {
	case errors.Is(err, session.ErrSessionNotFound):
		response = ErrResponseNotFound
	case errors.Is(err, session.ErrUnauthorized):
		response = ErrResponseUnauthorized
	default:
		response = ErrResponseGeneric
	}

	if response.Message != "" {
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: chatID,
			Text:   response.Message,
		})
	}
}

// LogError logs an error with context information
func LogError(operation string, userID int64, err error, details map[string]interface{}) {
	logEntry := map[string]interface{}{
		"level":     "error",
		"operation": operation,
		"user_id":   userID,
		"error":     err.Error(),
	}

	// Merge additional details
	for k, v := range details {
		logEntry[k] = v
	}

	log.Printf("[ERROR] operation=%s user_id=%d error=%v details=%+v",
		operation, userID, err, details)
}

// LogWarning logs a warning with context information
func LogWarning(operation string, userID int64, message string, details map[string]interface{}) {
	logEntry := map[string]interface{}{
		"level":     "warning",
		"operation": operation,
		"user_id":   userID,
		"message":   message,
	}

	// Merge additional details
	for k, v := range details {
		logEntry[k] = v
	}

	log.Printf("[WARNING] operation=%s user_id=%d message=%s details=%+v",
		operation, userID, message, details)
}

// LogInfo logs an informational message with context
func LogInfo(operation string, userID int64, message string, details map[string]interface{}) {
	logEntry := map[string]interface{}{
		"level":     "info",
		"operation": operation,
		"user_id":   userID,
		"message":   message,
	}

	// Merge additional details
	for k, v := range details {
		logEntry[k] = v
	}

	log.Printf("[INFO] operation=%s user_id=%d message=%s details=%+v",
		operation, userID, message, details)
}

// LogDebug logs a debug message with context
func LogDebug(operation string, userID int64, message string, details map[string]interface{}) {
	logEntry := map[string]interface{}{
		"level":     "debug",
		"operation": operation,
		"user_id":   userID,
		"message":   message,
	}

	// Merge additional details
	for k, v := range details {
		logEntry[k] = v
	}

	log.Printf("[DEBUG] operation=%s user_id=%d message=%s details=%+v",
		operation, userID, message, details)
}

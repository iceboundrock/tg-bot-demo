package handlers

import (
	"context"
	"fmt"
	"tg-bot-demo/session"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

// Package handlers provides Telegram bot command and callback handlers.
// It includes handlers for session list commands, pagination, and session switching.

// HandlerConfig holds configuration for handlers
type HandlerConfig struct {
	SessionsPerPage int
}

// SessionsCommandHandler handles the /sessions command
func SessionsCommandHandler(sessionMgr *session.Manager, cfg *HandlerConfig) bot.HandlerFunc {
	return func(ctx context.Context, b *bot.Bot, update *models.Update) {
		userID := update.Message.From.ID

		LogInfo("sessions_command", userID, "user requested session list", nil)

		// Get first page of sessions
		sessions, hasMore, err := sessionMgr.ListSessions(ctx, userID, 0, cfg.SessionsPerPage)
		if err != nil {
			LogError("sessions_command", userID, err, map[string]interface{}{
				"offset": 0,
				"limit":  cfg.SessionsPerPage,
			})
			SendErrorResponse(ctx, b, update.Message.Chat.ID, err)
			return
		}

		// Handle empty sessions
		if len(sessions) == 0 {
			LogInfo("sessions_command", userID, "no sessions found", nil)
			b.SendMessage(ctx, &bot.SendMessageParams{
				ChatID: update.Message.Chat.ID,
				Text:   "You don't have any sessions yet. Start chatting to create one!",
			})
			return
		}

		// Build inline keyboard
		keyboard := buildSessionKeyboard(sessions, 0, hasMore, cfg.SessionsPerPage)

		LogInfo("sessions_command", userID, "session list sent", map[string]interface{}{
			"session_count": len(sessions),
			"has_more":      hasMore,
		})

		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID:      update.Message.Chat.ID,
			Text:        "Your sessions:",
			ReplyMarkup: keyboard,
		})
	}
}

// CallbackQueryHandler handles inline keyboard button clicks
func CallbackQueryHandler(sessionMgr *session.Manager, cfg *HandlerConfig) bot.HandlerFunc {
	return func(ctx context.Context, b *bot.Bot, update *models.Update) {
		callback := update.CallbackQuery
		userID := callback.From.ID
		data := callback.Data

		// Answer callback immediately
		b.AnswerCallbackQuery(ctx, &bot.AnswerCallbackQueryParams{
			CallbackQueryID: callback.ID,
		})

		// Route based on callback data prefix
		if len(data) >= 7 && data[:7] == "open_s_" {
			handleOpenSession(ctx, b, callback, sessionMgr, userID, data)
		} else if len(data) >= 14 && data[:14] == "more_sessions_" {
			handleMoreSessions(ctx, b, callback, sessionMgr, userID, data, cfg.SessionsPerPage)
		} else {
			// Invalid callback data, log warning
			LogWarning("callback_query", userID, "invalid callback data format", map[string]interface{}{
				"callback_data": data,
			})
		}
	}
}

// MessageHandler handles regular text messages from users
func MessageHandler(sessionMgr *session.Manager) bot.HandlerFunc {
	return func(ctx context.Context, b *bot.Bot, update *models.Update) {
		// Extract user ID and message text
		userID := update.Message.From.ID
		messageText := update.Message.Text

		LogDebug("message_handler", userID, "processing message", map[string]interface{}{
			"message_length": len(messageText),
		})

		// Get or create active session for this user
		activeSession, err := sessionMgr.GetOrCreateActiveSession(ctx, userID, messageText)
		if err != nil {
			LogError("message_handler", userID, err, map[string]interface{}{
				"message_length": len(messageText),
			})
			SendErrorResponse(ctx, b, update.Message.Chat.ID, err)
			return
		}

		LogInfo("message_handler", userID, "message routed to session", map[string]interface{}{
			"session_id":    activeSession.ID.String(),
			"session_title": activeSession.Title,
		})

		// Route message to active session context
		// In a real implementation, this would forward the message to the AI service
		// For now, we'll send a confirmation that the message was received in the session
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: update.Message.Chat.ID,
			Text:   fmt.Sprintf("Message received in session: %s", activeSession.Title),
		})
	}
}

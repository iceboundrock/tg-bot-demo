package handlers

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"tg-bot-demo/session"
	"time"
	"unicode/utf8"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"github.com/google/uuid"
)

const (
	prevPageButtonText = "↑ 𝐏𝐫𝐞𝐯"
	nextPageButtonText = "↓ 𝐍𝐞𝐱𝐭"
)

// formatTimeAgo converts a timestamp to relative time string
func formatTimeAgo(t time.Time) string {
	duration := time.Since(t)

	switch {
	case duration < time.Minute:
		return "just now"
	case duration < time.Hour:
		mins := int(duration.Minutes())
		return fmt.Sprintf("%dm ago", mins)
	case duration < 24*time.Hour:
		hours := int(duration.Hours())
		return fmt.Sprintf("%dh ago", hours)
	case duration < 7*24*time.Hour:
		days := int(duration.Hours() / 24)
		return fmt.Sprintf("%dd ago", days)
	default:
		return t.Format("Jan 2")
	}
}

// truncate limits string length
func truncate(s string, maxLen int) string {
	if utf8.RuneCountInString(s) <= maxLen {
		return s
	}
	runes := []rune(s)
	return string(runes[:maxLen-3]) + "..."
}

// buildSessionKeyboard creates an inline keyboard for session list
func buildSessionKeyboard(sessions []*session.Session, offset int, hasPrev bool, hasNext bool, sessionsPerPage int) *models.InlineKeyboardMarkup {
	var rows [][]models.InlineKeyboardButton

	// Put previous-page navigation at the top.
	if hasPrev {
		prevOffset := offset - sessionsPerPage
		if prevOffset < 0 {
			prevOffset = 0
		}
		rows = append(rows, []models.InlineKeyboardButton{
			{
				Text:         prevPageButtonText,
				CallbackData: fmt.Sprintf("page_sessions_%d", prevOffset),
			},
		})
	}

	// Add session buttons (one per row)
	for _, s := range sessions {
		button := models.InlineKeyboardButton{
			Text:         formatSessionButton(s),
			CallbackData: fmt.Sprintf("open_s_%s", s.ID.String()),
		}
		rows = append(rows, []models.InlineKeyboardButton{button})
	}

	// Put next-page navigation at the bottom.
	if hasNext {
		rows = append(rows, []models.InlineKeyboardButton{
			{
				Text:         nextPageButtonText,
				CallbackData: fmt.Sprintf("page_sessions_%d", offset+sessionsPerPage),
			},
		})
	}

	return &models.InlineKeyboardMarkup{
		InlineKeyboard: rows,
	}
}

// formatSessionButton formats a session for display in button
func formatSessionButton(s *session.Session) string {
	// Format: "Title - 2h ago"
	timeAgo := formatTimeAgo(s.UpdatedAt)
	return fmt.Sprintf("%s - %s", truncate(s.Title, 40), timeAgo)
}

// handleOpenSession processes session switch requests
func handleOpenSession(ctx context.Context, b *bot.Bot, callback *models.CallbackQuery,
	sessionMgr *session.Manager, userID int64, data string) {
	// Get the message from callback
	msg := callback.Message.Message
	if msg == nil {
		return
	}

	// Parse session ID
	sessionIDStr := data[7:] // Skip "open_s_" prefix
	sessionID, err := uuid.Parse(sessionIDStr)
	if err != nil {
		LogWarning("open_session", userID, "invalid session ID format", map[string]interface{}{
			"session_id_str": sessionIDStr,
			"error":          err.Error(),
		})
		SendErrorResponse(ctx, b, msg.Chat.ID, err)
		return
	}

	LogInfo("open_session", userID, "switching session", map[string]interface{}{
		"session_id": sessionID.String(),
	})

	// Switch session
	sess, err := sessionMgr.SwitchSession(ctx, userID, sessionID)
	if err != nil {
		if errors.Is(err, session.ErrUnauthorized) {
			LogWarning("open_session", userID, "unauthorized access attempt", map[string]interface{}{
				"session_id": sessionID.String(),
			})
		} else if errors.Is(err, session.ErrSessionNotFound) {
			LogWarning("open_session", userID, "session not found", map[string]interface{}{
				"session_id": sessionID.String(),
			})
		} else {
			LogError("open_session", userID, err, map[string]interface{}{
				"session_id": sessionID.String(),
			})
		}
		SendErrorResponse(ctx, b, msg.Chat.ID, err)
		return
	}

	LogInfo("open_session", userID, "session switched successfully", map[string]interface{}{
		"session_id":    sess.ID.String(),
		"session_title": sess.Title,
	})

	// Send confirmation
	b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID: msg.Chat.ID,
		Text:   fmt.Sprintf("✅ Switched to session: %s", sess.Title),
	})
}

// handlePageSessions processes pagination requests.
func handlePageSessions(ctx context.Context, b *bot.Bot, callback *models.CallbackQuery,
	sessionMgr *session.Manager, userID int64, data string, sessionsPerPage int) {
	// Get the message from callback
	msg := callback.Message.Message
	if msg == nil {
		return
	}

	// Parse offset
	if len(data) < 14 || data[:14] != "page_sessions_" {
		LogWarning("page_sessions", userID, "invalid callback data prefix", map[string]interface{}{
			"callback_data": data,
		})
		return
	}
	offsetStr := data[14:]

	offset, err := strconv.Atoi(offsetStr)
	if err != nil {
		LogWarning("page_sessions", userID, "invalid offset format", map[string]interface{}{
			"offset_str": offsetStr,
			"error":      err.Error(),
		})
		return
	}

	if offset < 0 {
		LogWarning("page_sessions", userID, "negative offset", map[string]interface{}{
			"offset": offset,
		})
		return
	}

	LogDebug("page_sessions", userID, "loading page", map[string]interface{}{
		"offset": offset,
		"limit":  sessionsPerPage,
	})

	// Get page
	sessions, hasNext, err := sessionMgr.ListSessions(ctx, userID, offset, sessionsPerPage)
	if err != nil {
		LogError("page_sessions", userID, err, map[string]interface{}{
			"offset": offset,
			"limit":  sessionsPerPage,
		})
		return
	}

	hasPrev := offset > 0

	LogInfo("page_sessions", userID, "pagination successful", map[string]interface{}{
		"offset":        offset,
		"session_count": len(sessions),
		"has_prev":      hasPrev,
		"has_next":      hasNext,
	})

	// Update message with new keyboard
	keyboard := buildSessionKeyboard(sessions, offset, hasPrev, hasNext, sessionsPerPage)

	b.EditMessageReplyMarkup(ctx, &bot.EditMessageReplyMarkupParams{
		ChatID:      msg.Chat.ID,
		MessageID:   msg.ID,
		ReplyMarkup: keyboard,
	})
}

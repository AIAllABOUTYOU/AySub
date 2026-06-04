package service

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
)

const (
	PlaygroundModeChat  = "chat"
	PlaygroundModeImage = "image"
	PlaygroundModeVideo = "video"
	PlaygroundModeAudio = "audio"
)

var (
	ErrPlaygroundSessionNotFound = infraerrors.NotFound("PLAYGROUND_SESSION_NOT_FOUND", "playground session not found")
	ErrPlaygroundInvalidInput    = infraerrors.BadRequest("PLAYGROUND_INVALID_INPUT", "invalid playground session input")
)

type PlaygroundSession struct {
	ID         int64               `json:"id"`
	UserID     int64               `json:"user_id"`
	APIKeyID   *int64              `json:"api_key_id"`
	APIKeyName string              `json:"api_key_name"`
	Title      string              `json:"title"`
	Mode       string              `json:"mode"`
	Model      string              `json:"model"`
	Metadata   json.RawMessage     `json:"metadata"`
	CreatedAt  time.Time           `json:"created_at"`
	UpdatedAt  time.Time           `json:"updated_at"`
	Messages   []PlaygroundMessage `json:"messages,omitempty"`
}

type PlaygroundMessage struct {
	ID        int64           `json:"id"`
	SessionID int64           `json:"session_id"`
	UserID    int64           `json:"user_id"`
	Role      string          `json:"role"`
	Content   string          `json:"content"`
	Payload   json.RawMessage `json:"payload"`
	CreatedAt time.Time       `json:"created_at"`
}

type PlaygroundSessionRepository interface {
	CreateSession(ctx context.Context, session *PlaygroundSession) error
	UpdateSession(ctx context.Context, session *PlaygroundSession) error
	GetSession(ctx context.Context, userID, sessionID int64) (*PlaygroundSession, error)
	ListSessions(ctx context.Context, userID int64, params pagination.PaginationParams) ([]PlaygroundSession, *pagination.PaginationResult, error)
	DeleteSession(ctx context.Context, userID, sessionID int64) error
	AppendMessage(ctx context.Context, message *PlaygroundMessage) error
	ReplaceMessages(ctx context.Context, userID, sessionID int64, messages []PlaygroundMessage) error
}

type PlaygroundSessionService struct {
	repo PlaygroundSessionRepository
}

func NewPlaygroundSessionService(repo PlaygroundSessionRepository) *PlaygroundSessionService {
	return &PlaygroundSessionService{repo: repo}
}

func (s *PlaygroundSessionService) CreateSession(ctx context.Context, session *PlaygroundSession) error {
	if session == nil || session.UserID <= 0 {
		return ErrPlaygroundInvalidInput
	}
	normalizePlaygroundSession(session)
	if session.Title == "" {
		session.Title = defaultPlaygroundTitle(session.Mode)
	}
	return s.repo.CreateSession(ctx, session)
}

func (s *PlaygroundSessionService) UpdateSession(ctx context.Context, session *PlaygroundSession) error {
	if session == nil || session.UserID <= 0 || session.ID <= 0 {
		return ErrPlaygroundInvalidInput
	}
	normalizePlaygroundSession(session)
	return s.repo.UpdateSession(ctx, session)
}

func (s *PlaygroundSessionService) GetSession(ctx context.Context, userID, sessionID int64) (*PlaygroundSession, error) {
	if userID <= 0 || sessionID <= 0 {
		return nil, ErrPlaygroundInvalidInput
	}
	return s.repo.GetSession(ctx, userID, sessionID)
}

func (s *PlaygroundSessionService) ListSessions(ctx context.Context, userID int64, params pagination.PaginationParams) ([]PlaygroundSession, *pagination.PaginationResult, error) {
	if userID <= 0 {
		return nil, nil, ErrPlaygroundInvalidInput
	}
	if params.Page < 1 {
		params.Page = 1
	}
	if params.PageSize < 1 {
		params.PageSize = 20
	}
	if params.PageSize > 100 {
		params.PageSize = 100
	}
	return s.repo.ListSessions(ctx, userID, params)
}

func (s *PlaygroundSessionService) DeleteSession(ctx context.Context, userID, sessionID int64) error {
	if userID <= 0 || sessionID <= 0 {
		return ErrPlaygroundInvalidInput
	}
	return s.repo.DeleteSession(ctx, userID, sessionID)
}

func (s *PlaygroundSessionService) AppendMessage(ctx context.Context, message *PlaygroundMessage) error {
	if message == nil || message.UserID <= 0 || message.SessionID <= 0 {
		return ErrPlaygroundInvalidInput
	}
	normalizePlaygroundMessage(message)
	return s.repo.AppendMessage(ctx, message)
}

func (s *PlaygroundSessionService) ReplaceMessages(ctx context.Context, userID, sessionID int64, messages []PlaygroundMessage) error {
	if userID <= 0 || sessionID <= 0 {
		return ErrPlaygroundInvalidInput
	}
	for i := range messages {
		messages[i].UserID = userID
		messages[i].SessionID = sessionID
		normalizePlaygroundMessage(&messages[i])
	}
	return s.repo.ReplaceMessages(ctx, userID, sessionID, messages)
}

func normalizePlaygroundSession(session *PlaygroundSession) {
	session.Title = strings.TrimSpace(session.Title)
	if len(session.Title) > 120 {
		session.Title = session.Title[:120]
	}
	session.Mode = strings.TrimSpace(session.Mode)
	if session.Mode == "" {
		session.Mode = PlaygroundModeChat
	}
	switch session.Mode {
	case PlaygroundModeChat, PlaygroundModeImage, PlaygroundModeVideo, PlaygroundModeAudio:
	default:
		session.Mode = PlaygroundModeChat
	}
	session.Model = strings.TrimSpace(session.Model)
	session.APIKeyName = strings.TrimSpace(session.APIKeyName)
	if len(session.APIKeyName) > 120 {
		session.APIKeyName = session.APIKeyName[:120]
	}
	if len(session.Metadata) == 0 {
		session.Metadata = json.RawMessage(`{}`)
	}
}

func normalizePlaygroundMessage(message *PlaygroundMessage) {
	message.Role = strings.TrimSpace(message.Role)
	switch message.Role {
	case "system", "user", "assistant", "tool":
	default:
		message.Role = "user"
	}
	if len(message.Payload) == 0 {
		message.Payload = json.RawMessage(`{}`)
	}
}

func defaultPlaygroundTitle(mode string) string {
	switch mode {
	case PlaygroundModeImage:
		return "Image session"
	case PlaygroundModeVideo:
		return "Video session"
	case PlaygroundModeAudio:
		return "Audio session"
	default:
		return "Chat session"
	}
}

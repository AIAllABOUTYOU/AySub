package service

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
)

type playgroundSessionRepoStub struct {
	createdSession   *PlaygroundSession
	updatedSession   *PlaygroundSession
	appendedMessage  *PlaygroundMessage
	replacedMessages []PlaygroundMessage
	getUserID        int64
	getSessionID     int64
	listParams       pagination.PaginationParams
}

func (s *playgroundSessionRepoStub) CreateSession(_ context.Context, session *PlaygroundSession) error {
	s.createdSession = session
	session.ID = 42
	return nil
}

func (s *playgroundSessionRepoStub) UpdateSession(_ context.Context, session *PlaygroundSession) error {
	s.updatedSession = session
	return nil
}

func (s *playgroundSessionRepoStub) GetSession(_ context.Context, userID, sessionID int64) (*PlaygroundSession, error) {
	s.getUserID = userID
	s.getSessionID = sessionID
	return &PlaygroundSession{ID: sessionID, UserID: userID}, nil
}

func (s *playgroundSessionRepoStub) ListSessions(_ context.Context, userID int64, params pagination.PaginationParams) ([]PlaygroundSession, *pagination.PaginationResult, error) {
	s.getUserID = userID
	s.listParams = params
	return nil, &pagination.PaginationResult{Total: 0, Page: params.Page, PageSize: params.PageSize, Pages: 1}, nil
}

func (s *playgroundSessionRepoStub) DeleteSession(_ context.Context, userID, sessionID int64) error {
	s.getUserID = userID
	s.getSessionID = sessionID
	return nil
}

func (s *playgroundSessionRepoStub) AppendMessage(_ context.Context, message *PlaygroundMessage) error {
	s.appendedMessage = message
	message.ID = 7
	return nil
}

func (s *playgroundSessionRepoStub) ReplaceMessages(_ context.Context, _ int64, _ int64, messages []PlaygroundMessage) error {
	s.replacedMessages = messages
	return nil
}

func TestPlaygroundSessionServiceCreateSessionNormalizesInput(t *testing.T) {
	repo := &playgroundSessionRepoStub{}
	svc := NewPlaygroundSessionService(repo)
	session := &PlaygroundSession{
		UserID:     1,
		APIKeyName: "  main key  ",
		Title:      "  hello  ",
		Mode:       "unknown",
		Model:      "  grok-4  ",
	}

	if err := svc.CreateSession(context.Background(), session); err != nil {
		t.Fatalf("CreateSession() error = %v", err)
	}

	if repo.createdSession == nil {
		t.Fatal("CreateSession() did not call repository")
	}
	if session.ID != 42 {
		t.Fatalf("session ID = %d, want 42", session.ID)
	}
	if session.Title != "hello" {
		t.Fatalf("title = %q, want %q", session.Title, "hello")
	}
	if session.Mode != PlaygroundModeChat {
		t.Fatalf("mode = %q, want %q", session.Mode, PlaygroundModeChat)
	}
	if session.Model != "grok-4" {
		t.Fatalf("model = %q, want %q", session.Model, "grok-4")
	}
	if session.APIKeyName != "main key" {
		t.Fatalf("api key name = %q, want %q", session.APIKeyName, "main key")
	}
	if !json.Valid(session.Metadata) {
		t.Fatalf("metadata is not valid json: %s", string(session.Metadata))
	}
}

func TestPlaygroundSessionServiceAppendMessageNormalizesRoleAndPayload(t *testing.T) {
	repo := &playgroundSessionRepoStub{}
	svc := NewPlaygroundSessionService(repo)
	message := &PlaygroundMessage{
		UserID:    1,
		SessionID: 2,
		Role:      "bad-role",
		Content:   "hello",
	}

	if err := svc.AppendMessage(context.Background(), message); err != nil {
		t.Fatalf("AppendMessage() error = %v", err)
	}

	if repo.appendedMessage == nil {
		t.Fatal("AppendMessage() did not call repository")
	}
	if message.Role != "user" {
		t.Fatalf("role = %q, want user", message.Role)
	}
	if !json.Valid(message.Payload) {
		t.Fatalf("payload is not valid json: %s", string(message.Payload))
	}
}

func TestPlaygroundSessionServiceGetSessionPassesUserScope(t *testing.T) {
	repo := &playgroundSessionRepoStub{}
	svc := NewPlaygroundSessionService(repo)

	if _, err := svc.GetSession(context.Background(), 11, 22); err != nil {
		t.Fatalf("GetSession() error = %v", err)
	}

	if repo.getUserID != 11 || repo.getSessionID != 22 {
		t.Fatalf("repo scope = (%d, %d), want (11, 22)", repo.getUserID, repo.getSessionID)
	}
}

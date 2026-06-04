package handler

import (
	"encoding/json"
	"strconv"

	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	middleware2 "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
)

type PlaygroundSessionHandler struct {
	sessionService *service.PlaygroundSessionService
	apiKeyService  *service.APIKeyService
}

func NewPlaygroundSessionHandler(sessionService *service.PlaygroundSessionService, apiKeyService *service.APIKeyService) *PlaygroundSessionHandler {
	return &PlaygroundSessionHandler{
		sessionService: sessionService,
		apiKeyService:  apiKeyService,
	}
}

type playgroundSessionRequest struct {
	APIKeyID *int64          `json:"api_key_id"`
	Title    string          `json:"title"`
	Mode     string          `json:"mode"`
	Model    string          `json:"model"`
	Metadata json.RawMessage `json:"metadata"`
}

type playgroundMessageRequest struct {
	Role    string          `json:"role"`
	Content string          `json:"content"`
	Payload json.RawMessage `json:"payload"`
}

type playgroundReplaceMessagesRequest struct {
	Messages []playgroundMessageRequest `json:"messages"`
}

func (h *PlaygroundSessionHandler) List(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}

	page, pageSize := response.ParsePagination(c)
	items, result, err := h.sessionService.ListSessions(c.Request.Context(), subject.UserID, pagination.PaginationParams{
		Page:      page,
		PageSize:  pageSize,
		SortBy:    "updated_at",
		SortOrder: pagination.SortOrderDesc,
	})
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	response.Paginated(c, items, result.Total, result.Page, result.PageSize)
}

func (h *PlaygroundSessionHandler) Create(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}

	var req playgroundSessionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}

	apiKeyName, err := h.resolveAPIKeyName(c, subject.UserID, req.APIKeyID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	session := &service.PlaygroundSession{
		UserID:     subject.UserID,
		APIKeyID:   req.APIKeyID,
		APIKeyName: apiKeyName,
		Title:      req.Title,
		Mode:       req.Mode,
		Model:      req.Model,
		Metadata:   req.Metadata,
	}
	if err := h.sessionService.CreateSession(c.Request.Context(), session); err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Created(c, session)
}

func (h *PlaygroundSessionHandler) Get(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}
	sessionID, ok := parsePlaygroundSessionID(c)
	if !ok {
		return
	}

	session, err := h.sessionService.GetSession(c.Request.Context(), subject.UserID, sessionID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, session)
}

func (h *PlaygroundSessionHandler) Update(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}
	sessionID, ok := parsePlaygroundSessionID(c)
	if !ok {
		return
	}

	var req playgroundSessionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}

	apiKeyName, err := h.resolveAPIKeyName(c, subject.UserID, req.APIKeyID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	session := &service.PlaygroundSession{
		ID:         sessionID,
		UserID:     subject.UserID,
		APIKeyID:   req.APIKeyID,
		APIKeyName: apiKeyName,
		Title:      req.Title,
		Mode:       req.Mode,
		Model:      req.Model,
		Metadata:   req.Metadata,
	}
	if err := h.sessionService.UpdateSession(c.Request.Context(), session); err != nil {
		response.ErrorFrom(c, err)
		return
	}
	updated, err := h.sessionService.GetSession(c.Request.Context(), subject.UserID, sessionID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, updated)
}

func (h *PlaygroundSessionHandler) Delete(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}
	sessionID, ok := parsePlaygroundSessionID(c)
	if !ok {
		return
	}
	if err := h.sessionService.DeleteSession(c.Request.Context(), subject.UserID, sessionID); err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"deleted": true})
}

func (h *PlaygroundSessionHandler) AppendMessage(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}
	sessionID, ok := parsePlaygroundSessionID(c)
	if !ok {
		return
	}

	var req playgroundMessageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}

	message := &service.PlaygroundMessage{
		SessionID: sessionID,
		UserID:    subject.UserID,
		Role:      req.Role,
		Content:   req.Content,
		Payload:   req.Payload,
	}
	if err := h.sessionService.AppendMessage(c.Request.Context(), message); err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Created(c, message)
}

func (h *PlaygroundSessionHandler) ReplaceMessages(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}
	sessionID, ok := parsePlaygroundSessionID(c)
	if !ok {
		return
	}

	var req playgroundReplaceMessagesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}

	messages := make([]service.PlaygroundMessage, 0, len(req.Messages))
	for _, item := range req.Messages {
		messages = append(messages, service.PlaygroundMessage{
			SessionID: sessionID,
			UserID:    subject.UserID,
			Role:      item.Role,
			Content:   item.Content,
			Payload:   item.Payload,
		})
	}
	if err := h.sessionService.ReplaceMessages(c.Request.Context(), subject.UserID, sessionID, messages); err != nil {
		response.ErrorFrom(c, err)
		return
	}
	session, err := h.sessionService.GetSession(c.Request.Context(), subject.UserID, sessionID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, session)
}

func (h *PlaygroundSessionHandler) resolveAPIKeyName(c *gin.Context, userID int64, apiKeyID *int64) (string, error) {
	if apiKeyID == nil || *apiKeyID <= 0 {
		return "", nil
	}
	key, err := h.apiKeyService.GetByID(c.Request.Context(), *apiKeyID)
	if err != nil {
		return "", err
	}
	if key.UserID != userID {
		return "", service.ErrAPIKeyNotFound
	}
	return key.Name, nil
}

func parsePlaygroundSessionID(c *gin.Context) (int64, bool) {
	sessionID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || sessionID <= 0 {
		response.BadRequest(c, "Invalid session ID")
		return 0, false
	}
	return sessionID, true
}

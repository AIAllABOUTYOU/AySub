package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"math"

	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
	"github.com/Wei-Shaw/sub2api/internal/service"
)

type playgroundSessionRepository struct {
	db *sql.DB
}

func NewPlaygroundSessionRepository(db *sql.DB) service.PlaygroundSessionRepository {
	return &playgroundSessionRepository{db: db}
}

func (r *playgroundSessionRepository) CreateSession(ctx context.Context, session *service.PlaygroundSession) error {
	metadata := normalizeJSONRaw(session.Metadata)
	return r.db.QueryRowContext(ctx, `
		INSERT INTO playground_sessions (user_id, api_key_id, api_key_name, title, mode, model, metadata)
		VALUES ($1, $2, $3, $4, $5, $6, $7::jsonb)
		RETURNING id, created_at, updated_at
	`, session.UserID, session.APIKeyID, session.APIKeyName, session.Title, session.Mode, session.Model, string(metadata)).
		Scan(&session.ID, &session.CreatedAt, &session.UpdatedAt)
}

func (r *playgroundSessionRepository) UpdateSession(ctx context.Context, session *service.PlaygroundSession) error {
	metadata := normalizeJSONRaw(session.Metadata)
	err := r.db.QueryRowContext(ctx, `
		UPDATE playground_sessions
		SET api_key_id = $3,
		    api_key_name = $4,
		    title = $5,
		    mode = $6,
		    model = $7,
		    metadata = $8::jsonb,
		    updated_at = NOW()
		WHERE user_id = $1 AND id = $2 AND deleted_at IS NULL
		RETURNING updated_at
	`, session.UserID, session.ID, session.APIKeyID, session.APIKeyName, session.Title, session.Mode, session.Model, string(metadata)).
		Scan(&session.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return service.ErrPlaygroundSessionNotFound
	}
	return err
}

func (r *playgroundSessionRepository) GetSession(ctx context.Context, userID, sessionID int64) (*service.PlaygroundSession, error) {
	session, err := r.getSessionWithoutMessages(ctx, userID, sessionID)
	if err != nil {
		return nil, err
	}
	messages, err := r.listMessages(ctx, userID, sessionID)
	if err != nil {
		return nil, err
	}
	session.Messages = messages
	return session, nil
}

func (r *playgroundSessionRepository) ListSessions(ctx context.Context, userID int64, params pagination.PaginationParams) ([]service.PlaygroundSession, *pagination.PaginationResult, error) {
	total := int64(0)
	if err := r.db.QueryRowContext(ctx, `
		SELECT COUNT(*)
		FROM playground_sessions
		WHERE user_id = $1 AND deleted_at IS NULL
	`, userID).Scan(&total); err != nil {
		return nil, nil, err
	}

	rows, err := r.db.QueryContext(ctx, `
		SELECT id, user_id, api_key_id, api_key_name, title, mode, model, metadata, created_at, updated_at
		FROM playground_sessions
		WHERE user_id = $1 AND deleted_at IS NULL
		ORDER BY updated_at DESC, id DESC
		OFFSET $2 LIMIT $3
	`, userID, params.Offset(), params.Limit())
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()

	items := make([]service.PlaygroundSession, 0)
	for rows.Next() {
		session, err := scanPlaygroundSession(rows)
		if err != nil {
			return nil, nil, err
		}
		items = append(items, *session)
	}
	if err := rows.Err(); err != nil {
		return nil, nil, err
	}

	pages := int(math.Ceil(float64(total) / float64(params.Limit())))
	if pages < 1 {
		pages = 1
	}
	return items, &pagination.PaginationResult{
		Total:    total,
		Page:     params.Page,
		PageSize: params.Limit(),
		Pages:    pages,
	}, nil
}

func (r *playgroundSessionRepository) DeleteSession(ctx context.Context, userID, sessionID int64) error {
	res, err := r.db.ExecContext(ctx, `
		UPDATE playground_sessions
		SET deleted_at = NOW(), updated_at = NOW()
		WHERE user_id = $1 AND id = $2 AND deleted_at IS NULL
	`, userID, sessionID)
	if err != nil {
		return err
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return service.ErrPlaygroundSessionNotFound
	}
	return nil
}

func (r *playgroundSessionRepository) AppendMessage(ctx context.Context, message *service.PlaygroundMessage) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	if err := ensurePlaygroundSessionTx(ctx, tx, message.UserID, message.SessionID); err != nil {
		return err
	}

	payload := normalizeJSONRaw(message.Payload)
	if err := tx.QueryRowContext(ctx, `
		INSERT INTO playground_messages (session_id, user_id, role, content, payload)
		VALUES ($1, $2, $3, $4, $5::jsonb)
		RETURNING id, created_at
	`, message.SessionID, message.UserID, message.Role, message.Content, string(payload)).
		Scan(&message.ID, &message.CreatedAt); err != nil {
		return err
	}

	if _, err := tx.ExecContext(ctx, `
		UPDATE playground_sessions SET updated_at = NOW()
		WHERE user_id = $1 AND id = $2 AND deleted_at IS NULL
	`, message.UserID, message.SessionID); err != nil {
		return err
	}

	return tx.Commit()
}

func (r *playgroundSessionRepository) ReplaceMessages(ctx context.Context, userID, sessionID int64, messages []service.PlaygroundMessage) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	if err := ensurePlaygroundSessionTx(ctx, tx, userID, sessionID); err != nil {
		return err
	}

	if _, err := tx.ExecContext(ctx, `DELETE FROM playground_messages WHERE user_id = $1 AND session_id = $2`, userID, sessionID); err != nil {
		return err
	}

	for i := range messages {
		payload := normalizeJSONRaw(messages[i].Payload)
		if err := tx.QueryRowContext(ctx, `
			INSERT INTO playground_messages (session_id, user_id, role, content, payload)
			VALUES ($1, $2, $3, $4, $5::jsonb)
			RETURNING id, created_at
		`, sessionID, userID, messages[i].Role, messages[i].Content, string(payload)).
			Scan(&messages[i].ID, &messages[i].CreatedAt); err != nil {
			return err
		}
	}

	if _, err := tx.ExecContext(ctx, `
		UPDATE playground_sessions SET updated_at = NOW()
		WHERE user_id = $1 AND id = $2 AND deleted_at IS NULL
	`, userID, sessionID); err != nil {
		return err
	}

	return tx.Commit()
}

func (r *playgroundSessionRepository) getSessionWithoutMessages(ctx context.Context, userID, sessionID int64) (*service.PlaygroundSession, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT id, user_id, api_key_id, api_key_name, title, mode, model, metadata, created_at, updated_at
		FROM playground_sessions
		WHERE user_id = $1 AND id = $2 AND deleted_at IS NULL
	`, userID, sessionID)
	session, err := scanPlaygroundSession(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, service.ErrPlaygroundSessionNotFound
	}
	if err != nil {
		return nil, err
	}
	return session, nil
}

func (r *playgroundSessionRepository) listMessages(ctx context.Context, userID, sessionID int64) ([]service.PlaygroundMessage, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, session_id, user_id, role, content, payload, created_at
		FROM playground_messages
		WHERE user_id = $1 AND session_id = $2
		ORDER BY id ASC
	`, userID, sessionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	messages := make([]service.PlaygroundMessage, 0)
	for rows.Next() {
		message, err := scanPlaygroundMessage(rows)
		if err != nil {
			return nil, err
		}
		messages = append(messages, *message)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return messages, nil
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanPlaygroundSession(row rowScanner) (*service.PlaygroundSession, error) {
	var session service.PlaygroundSession
	var apiKeyID sql.NullInt64
	var metadata []byte
	if err := row.Scan(
		&session.ID,
		&session.UserID,
		&apiKeyID,
		&session.APIKeyName,
		&session.Title,
		&session.Mode,
		&session.Model,
		&metadata,
		&session.CreatedAt,
		&session.UpdatedAt,
	); err != nil {
		return nil, err
	}
	if apiKeyID.Valid {
		session.APIKeyID = &apiKeyID.Int64
	}
	session.Metadata = normalizeJSONRaw(metadata)
	return &session, nil
}

func scanPlaygroundMessage(row rowScanner) (*service.PlaygroundMessage, error) {
	var message service.PlaygroundMessage
	var payload []byte
	if err := row.Scan(
		&message.ID,
		&message.SessionID,
		&message.UserID,
		&message.Role,
		&message.Content,
		&payload,
		&message.CreatedAt,
	); err != nil {
		return nil, err
	}
	message.Payload = normalizeJSONRaw(payload)
	return &message, nil
}

func ensurePlaygroundSessionTx(ctx context.Context, tx *sql.Tx, userID, sessionID int64) error {
	var exists bool
	if err := tx.QueryRowContext(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM playground_sessions
			WHERE user_id = $1 AND id = $2 AND deleted_at IS NULL
		)
	`, userID, sessionID).Scan(&exists); err != nil {
		return err
	}
	if !exists {
		return service.ErrPlaygroundSessionNotFound
	}
	return nil
}

func normalizeJSONRaw(raw json.RawMessage) json.RawMessage {
	if len(raw) == 0 {
		return json.RawMessage(`{}`)
	}
	if !json.Valid(raw) {
		return json.RawMessage(`{}`)
	}
	return raw
}

package handler

import (
	"context"
	"errors"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

type classifierTimeoutError struct{}

func (classifierTimeoutError) Error() string   { return "dial tcp: i/o timeout" }
func (classifierTimeoutError) Timeout() bool   { return true }
func (classifierTimeoutError) Temporary() bool { return true }

func TestClassifyUpstreamFailoverError(t *testing.T) {
	existing := &service.UpstreamFailoverError{StatusCode: http.StatusServiceUnavailable}

	tests := []struct {
		name       string
		ctx        context.Context
		err        error
		wantOK     bool
		wantStatus int
	}{
		{
			name:       "existing failover error",
			ctx:        context.Background(),
			err:        existing,
			wantOK:     true,
			wantStatus: http.StatusServiceUnavailable,
		},
		{
			name:       "eof",
			ctx:        context.Background(),
			err:        io.EOF,
			wantOK:     true,
			wantStatus: http.StatusBadGateway,
		},
		{
			name:       "unexpected eof",
			ctx:        context.Background(),
			err:        io.ErrUnexpectedEOF,
			wantOK:     true,
			wantStatus: http.StatusBadGateway,
		},
		{
			name:       "context deadline",
			ctx:        context.Background(),
			err:        context.DeadlineExceeded,
			wantOK:     true,
			wantStatus: http.StatusRequestTimeout,
		},
		{
			name:       "net timeout",
			ctx:        context.Background(),
			err:        classifierTimeoutError{},
			wantOK:     true,
			wantStatus: http.StatusRequestTimeout,
		},
		{
			name:       "connection reset text",
			ctx:        context.Background(),
			err:        errors.New("read: connection reset by peer"),
			wantOK:     true,
			wantStatus: http.StatusBadGateway,
		},
		{
			name:   "request body error is not failover",
			ctx:    context.Background(),
			err:    errors.New("invalid request body"),
			wantOK: false,
		},
		{
			name:   "local policy rejection is not failover",
			ctx:    context.Background(),
			err:    errors.New("model not supported"),
			wantOK: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := ClassifyUpstreamFailoverError(tt.ctx, tt.err)
			require.Equal(t, tt.wantOK, ok)
			if !tt.wantOK {
				require.Nil(t, got)
				return
			}
			require.NotNil(t, got)
			require.Equal(t, tt.wantStatus, got.StatusCode)
		})
	}
}

func TestClassifyUpstreamFailoverError_ContextCanceledDoesNotFailover(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	got, ok := ClassifyUpstreamFailoverError(ctx, errors.New("read: connection reset by peer"))
	require.False(t, ok)
	require.Nil(t, got)
}

func TestClassifyUpstreamFailoverError_ContextCanceledExistingFailoverDoesNotFailover(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	got, ok := ClassifyUpstreamFailoverError(ctx, &service.UpstreamFailoverError{StatusCode: http.StatusServiceUnavailable})
	require.False(t, ok)
	require.Nil(t, got)
}

func TestClassifyUpstreamFailoverError_ContextDeadlineOnRequestDoesNotFailover(t *testing.T) {
	ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(-time.Second))
	defer cancel()

	got, ok := ClassifyUpstreamFailoverError(ctx, io.EOF)
	require.False(t, ok)
	require.Nil(t, got)
}

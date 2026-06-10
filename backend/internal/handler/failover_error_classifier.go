package handler

import (
	"context"
	"errors"
	"io"
	"net"
	"net/http"
	"strings"
	"syscall"

	"github.com/Wei-Shaw/sub2api/internal/service"
)

func ClassifyUpstreamFailoverError(ctx context.Context, err error) (*service.UpstreamFailoverError, bool) {
	if err == nil {
		return nil, false
	}
	if ctx != nil && ctx.Err() != nil {
		return nil, false
	}
	var failoverErr *service.UpstreamFailoverError
	if errors.As(err, &failoverErr) {
		return failoverErr, true
	}
	statusCode := classifyTransportFailoverStatus(err)
	if statusCode == 0 {
		return nil, false
	}
	return &service.UpstreamFailoverError{
		StatusCode:   statusCode,
		ResponseBody: []byte(err.Error()),
	}, true
}

func classifyTransportFailoverStatus(err error) int {
	if err == nil {
		return 0
	}
	if errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) {
		return http.StatusBadGateway
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return http.StatusRequestTimeout
	}
	var netErr net.Error
	if errors.As(err, &netErr) && netErr.Timeout() {
		return http.StatusRequestTimeout
	}
	if errors.Is(err, syscall.ECONNRESET) ||
		errors.Is(err, syscall.ECONNREFUSED) ||
		errors.Is(err, syscall.EPIPE) ||
		errors.Is(err, syscall.ETIMEDOUT) {
		return http.StatusBadGateway
	}

	msg := strings.ToLower(err.Error())
	failoverSignals := []string{
		"eof",
		"connection reset",
		"connection refused",
		"broken pipe",
		"i/o timeout",
		"timeout awaiting response headers",
		"tls handshake timeout",
		"upstream request failed",
		"stream data interval timeout",
	}
	for _, signal := range failoverSignals {
		if strings.Contains(msg, signal) {
			if strings.Contains(signal, "timeout") {
				return http.StatusRequestTimeout
			}
			return http.StatusBadGateway
		}
	}
	return 0
}

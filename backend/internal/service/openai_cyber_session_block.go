package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
	"github.com/gin-gonic/gin"
)

type CyberSessionBlockStore interface {
	SetCyberSessionBlocked(context.Context, string, time.Duration) error
	IsCyberSessionBlocked(context.Context, string) (bool, error)
}

func CyberSessionBlockKey(apiKeyID int64, c *gin.Context, body []byte) string {
	raw := explicitOpenAISessionID(c, body)
	if raw == "" {
		return ""
	}
	sum := sha256.Sum256([]byte(isolateOpenAISessionID(apiKeyID, raw)))
	return hex.EncodeToString(sum[:])
}

func (s *OpenAIGatewayService) cyberSessionBlockStore() CyberSessionBlockStore {
	store, _ := s.cache.(CyberSessionBlockStore)
	return store
}

func (s *OpenAIGatewayService) CyberSessionBlockRuntime(ctx context.Context) (bool, time.Duration) {
	if s == nil || s.settingService == nil {
		return false, time.Hour
	}
	return s.settingService.GetCyberSessionBlockRuntime(ctx)
}

func (s *OpenAIGatewayService) MarkCyberSessionBlocked(ctx context.Context, key string) {
	enabled, ttl := s.CyberSessionBlockRuntime(ctx)
	store := s.cyberSessionBlockStore()
	if key == "" || !enabled || store == nil {
		return
	}
	if err := store.SetCyberSessionBlocked(ctx, key, ttl); err != nil {
		logger.LegacyPrintf("service.openai_gateway", "cyber session block write failed: %v", err)
	}
}

func (s *OpenAIGatewayService) IsCyberSessionBlocked(ctx context.Context, key string) bool {
	enabled, _ := s.CyberSessionBlockRuntime(ctx)
	store := s.cyberSessionBlockStore()
	if key == "" || !enabled || store == nil {
		return false
	}
	blocked, err := store.IsCyberSessionBlocked(ctx, key)
	if err != nil {
		logger.LegacyPrintf("service.openai_gateway", "cyber session block read failed: %v", err)
		return false
	}
	return blocked
}

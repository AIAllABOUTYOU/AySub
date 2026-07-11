package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/openai"
)

const codexRestrictionPolicyCacheTTL = 60 * time.Second
const codexRestrictionPolicyDBTimeout = 5 * time.Second

type cachedCodexRestrictionPolicy struct {
	value     CodexRestrictionPolicy
	expiresAt int64
}

var legacyClaudeCodeCodexWhitelistEntry = openai.AllowedClientEntry{
	Originator: "Claude Code",
	UAContains: []string{"Claude Code/"},
}

func (s *SettingService) GetCodexRestrictionPolicy(ctx context.Context) CodexRestrictionPolicy {
	fallback := CodexRestrictionPolicy{EngineFingerprintSignals: openai.DefaultEngineFingerprintSignals}
	if s == nil || s.settingRepo == nil {
		return fallback
	}
	if cached, ok := s.codexRestrictionPolicyCache.Load().(*cachedCodexRestrictionPolicy); ok && cached != nil && time.Now().UnixNano() < cached.expiresAt {
		return cached.value
	}
	result, _, _ := s.codexRestrictionPolicySF.Do("codex_restriction_policy", func() (any, error) {
		dbCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), codexRestrictionPolicyDBTimeout)
		defer cancel()
		policy := fallback
		if v, err := s.settingRepo.GetValue(dbCtx, SettingKeyMinCodexVersion); err == nil {
			policy.MinCodexVersion = strings.TrimSpace(v)
		}
		if v, err := s.settingRepo.GetValue(dbCtx, SettingKeyMaxCodexVersion); err == nil {
			policy.MaxCodexVersion = strings.TrimSpace(v)
		}
		if v, err := s.settingRepo.GetValue(dbCtx, SettingKeyCodexCLIOnlyAllowAppServerClients); err == nil {
			policy.AllowAppServerClients = strings.TrimSpace(v) == "true"
		}
		policy.Whitelist = s.loadCodexClientEntries(dbCtx, SettingKeyCodexCLIOnlyWhitelist)
		policy.Blacklist = s.loadCodexClientEntries(dbCtx, SettingKeyCodexCLIOnlyBlacklist)
		policy.EngineFingerprintSignals = s.loadEngineFingerprintSignals(dbCtx)
		entry := &cachedCodexRestrictionPolicy{value: policy, expiresAt: time.Now().Add(codexRestrictionPolicyCacheTTL).UnixNano()}
		s.codexRestrictionPolicyCache.Store(entry)
		return policy, nil
	})
	if policy, ok := result.(CodexRestrictionPolicy); ok {
		return policy
	}
	return fallback
}

func (s *SettingService) loadCodexClientEntries(ctx context.Context, key string) []openai.AllowedClientEntry {
	v, err := s.settingRepo.GetValue(ctx, key)
	if err != nil || strings.TrimSpace(v) == "" {
		return nil
	}
	var entries []openai.AllowedClientEntry
	if json.Unmarshal([]byte(v), &entries) != nil {
		return nil
	}
	return entries
}

func (s *SettingService) loadEngineFingerprintSignals(ctx context.Context) []openai.EngineFingerprintSignal {
	v, err := s.settingRepo.GetValue(ctx, SettingKeyCodexCLIOnlyEngineFingerprintSignals)
	if err != nil || strings.TrimSpace(v) == "" {
		return openai.DefaultEngineFingerprintSignals
	}
	signals, ok := openai.ParseEngineFingerprintSignals(v)
	if !ok {
		return openai.DefaultEngineFingerprintSignals
	}
	return signals
}

func (s *SettingService) invalidateCodexRestrictionPolicy() {
	if s == nil {
		return
	}
	s.codexRestrictionPolicySF.Forget("codex_restriction_policy")
	s.codexRestrictionPolicyCache.Store(&cachedCodexRestrictionPolicy{expiresAt: 0})
}

func (s *SettingService) MigrateOpenAIAllowClaudeCodeCodexPluginSetting(ctx context.Context) error {
	if s == nil || s.settingRepo == nil {
		return nil
	}
	legacy, err := s.settingRepo.GetValue(ctx, SettingKeyOpenAIAllowClaudeCodeCodexPlugin)
	if errors.Is(err, ErrSettingNotFound) || strings.TrimSpace(legacy) != "true" {
		return nil
	}
	if err != nil {
		return err
	}
	entries := s.loadCodexClientEntries(ctx, SettingKeyCodexCLIOnlyWhitelist)
	for _, entry := range entries {
		if strings.EqualFold(strings.TrimSpace(entry.Originator), legacyClaudeCodeCodexWhitelistEntry.Originator) {
			return nil
		}
	}
	entries = append(entries, legacyClaudeCodeCodexWhitelistEntry)
	raw, err := json.Marshal(entries)
	if err != nil {
		return err
	}
	if err := s.settingRepo.Set(ctx, SettingKeyCodexCLIOnlyWhitelist, string(raw)); err != nil {
		return err
	}
	s.invalidateCodexRestrictionPolicy()
	return nil
}

func (s *SettingService) MigrateCodexBodyFingerprintToSignals(ctx context.Context) error {
	if s == nil || s.settingRepo == nil {
		return nil
	}
	if v, err := s.settingRepo.GetValue(ctx, SettingKeyCodexCLIOnlyEngineFingerprintSignals); err == nil && strings.TrimSpace(v) != "" {
		return nil
	}
	bodyRequired := false
	if v, err := s.settingRepo.GetValue(ctx, SettingKeyCodexCLIOnlyAllowBodyEngineFingerprint); err == nil {
		bodyRequired = strings.TrimSpace(v) == "true"
	}
	signals := append([]openai.EngineFingerprintSignal(nil), openai.DefaultEngineFingerprintSignals...)
	if bodyRequired {
		for i := range signals {
			if signals[i].Type == openai.FingerprintSignalBodyPath {
				signals[i].Required = true
			}
		}
	}
	raw, err := json.Marshal(signals)
	if err != nil {
		return err
	}
	if err := s.settingRepo.Set(ctx, SettingKeyCodexCLIOnlyEngineFingerprintSignals, string(raw)); err != nil {
		return err
	}
	s.invalidateCodexRestrictionPolicy()
	return nil
}

func ValidateCodexClientEntriesJSON(raw string) error {
	if strings.TrimSpace(raw) == "" {
		return nil
	}
	var entries []openai.AllowedClientEntry
	if err := json.Unmarshal([]byte(raw), &entries); err != nil {
		return fmt.Errorf("must be a valid client entry array")
	}
	return nil
}

func ValidateCodexWhitelistEntriesJSON(raw string) error {
	if err := ValidateCodexClientEntriesJSON(raw); err != nil {
		return err
	}
	if strings.TrimSpace(raw) == "" {
		return nil
	}
	var entries []openai.AllowedClientEntry
	_ = json.Unmarshal([]byte(raw), &entries)
	for i, entry := range entries {
		if !entry.IsWhitelistable() {
			return fmt.Errorf("entry %d must include originator and ua_contains", i)
		}
	}
	return nil
}

func ValidateEngineFingerprintSignalsJSON(raw string) error {
	return openai.ValidateEngineFingerprintSignalsJSON(raw)
}

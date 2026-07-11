package service

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/pkg/openai"
	"github.com/stretchr/testify/require"
)

type codexPolicyRepoStub struct {
	values map[string]string
	sets   map[string]string
}

func (s *codexPolicyRepoStub) Get(context.Context, string) (*Setting, error) { panic("unused") }
func (s *codexPolicyRepoStub) GetValue(_ context.Context, key string) (string, error) {
	if value, ok := s.values[key]; ok {
		return value, nil
	}
	return "", ErrSettingNotFound
}
func (s *codexPolicyRepoStub) Set(_ context.Context, key, value string) error {
	if s.sets == nil {
		s.sets = map[string]string{}
	}
	s.sets[key] = value
	s.values[key] = value
	return nil
}
func (s *codexPolicyRepoStub) GetMultiple(context.Context, []string) (map[string]string, error) {
	panic("unused")
}
func (s *codexPolicyRepoStub) SetMultiple(_ context.Context, values map[string]string) error {
	if s.values == nil {
		s.values = map[string]string{}
	}
	for key, value := range values {
		s.values[key] = value
	}
	return nil
}
func (s *codexPolicyRepoStub) GetAll(context.Context) (map[string]string, error) {
	out := make(map[string]string, len(s.values))
	for key, value := range s.values {
		out[key] = value
	}
	return out, nil
}
func (s *codexPolicyRepoStub) Delete(context.Context, string) error { panic("unused") }

func TestGetCodexRestrictionPolicy(t *testing.T) {
	repo := &codexPolicyRepoStub{values: map[string]string{
		SettingKeyMinCodexVersion:                      "0.141.0",
		SettingKeyMaxCodexVersion:                      "0.200.0",
		SettingKeyCodexCLIOnlyWhitelist:                `[{"originator":"opencode","ua_contains":["opencode/"]}]`,
		SettingKeyCodexCLIOnlyBlacklist:                `[{"originator":"evil"}]`,
		SettingKeyCodexCLIOnlyAllowAppServerClients:    "true",
		SettingKeyCodexCLIOnlyEngineFingerprintSignals: `[{"type":"header_exact","match":["session-id"],"required":true}]`,
	}}
	policy := NewSettingService(repo, &config.Config{}).GetCodexRestrictionPolicy(context.Background())
	require.Equal(t, "0.141.0", policy.MinCodexVersion)
	require.Equal(t, "0.200.0", policy.MaxCodexVersion)
	require.Len(t, policy.Whitelist, 1)
	require.Len(t, policy.Blacklist, 1)
	require.True(t, policy.AllowAppServerClients)
	require.Equal(t, "session-id", policy.EngineFingerprintSignals[0].Match[0])
}

func TestGetCodexRestrictionPolicy_DefaultsFailClosed(t *testing.T) {
	policy := NewSettingService(&codexPolicyRepoStub{values: map[string]string{}}, &config.Config{}).
		GetCodexRestrictionPolicy(context.Background())
	require.Empty(t, policy.MinCodexVersion)
	require.Empty(t, policy.Whitelist)
	require.Empty(t, policy.Blacklist)
	require.False(t, policy.AllowAppServerClients)
	require.Equal(t, openai.DefaultEngineFingerprintSignals, policy.EngineFingerprintSignals)
}

func TestRefreshCachedSettings_InvalidatesCodexRestrictionPolicy(t *testing.T) {
	repo := &codexPolicyRepoStub{values: map[string]string{SettingKeyMinCodexVersion: "0.141.0"}}
	svc := NewSettingService(repo, &config.Config{})
	require.Equal(t, "0.141.0", svc.GetCodexRestrictionPolicy(context.Background()).MinCodexVersion)
	repo.values[SettingKeyMinCodexVersion] = "0.150.0"
	svc.refreshCachedSettings(&SystemSettings{MinCodexVersion: "0.150.0"})
	require.Equal(t, "0.150.0", svc.GetCodexRestrictionPolicy(context.Background()).MinCodexVersion)
}

func TestInitializeDefaultSettings_CodexPolicyDefaults(t *testing.T) {
	repo := &codexPolicyRepoStub{values: map[string]string{}}
	svc := NewSettingService(repo, &config.Config{Default: config.DefaultConfig{UserConcurrency: 5}})
	require.NoError(t, svc.InitializeDefaultSettings(context.Background()))
	require.Equal(t, "", repo.values[SettingKeyMinCodexVersion])
	require.Equal(t, "", repo.values[SettingKeyMaxCodexVersion])
	require.Equal(t, "", repo.values[SettingKeyCodexCLIOnlyBlacklist])
	require.Equal(t, "", repo.values[SettingKeyCodexCLIOnlyWhitelist])
	require.Equal(t, "false", repo.values[SettingKeyCodexCLIOnlyAllowAppServerClients])
	require.Equal(t, openai.DefaultEngineFingerprintSignalsJSON(), repo.values[SettingKeyCodexCLIOnlyEngineFingerprintSignals])
}

func TestParseSettings_CodexFingerprintDefaultsMatchRuntime(t *testing.T) {
	svc := NewSettingService(&codexPolicyRepoStub{values: map[string]string{}}, &config.Config{})
	settings := svc.parseSettings(map[string]string{})
	require.Equal(t, openai.DefaultEngineFingerprintSignalsJSON(), settings.CodexCLIOnlyEngineFingerprintSignals)
}

func TestMigrateOpenAIAllowClaudeCodeCodexPluginSetting(t *testing.T) {
	repo := &codexPolicyRepoStub{values: map[string]string{
		SettingKeyOpenAIAllowClaudeCodeCodexPlugin: "true",
		SettingKeyCodexCLIOnlyWhitelist:            `[{"originator":"opencode","ua_contains":["opencode/"]}]`,
	}}
	svc := NewSettingService(repo, &config.Config{})
	require.NoError(t, svc.MigrateOpenAIAllowClaudeCodeCodexPluginSetting(context.Background()))
	var entries []openai.AllowedClientEntry
	require.NoError(t, json.Unmarshal([]byte(repo.values[SettingKeyCodexCLIOnlyWhitelist]), &entries))
	require.Len(t, entries, 2)
	require.Equal(t, "Claude Code", entries[1].Originator)

	repo.sets = nil
	require.NoError(t, svc.MigrateOpenAIAllowClaudeCodeCodexPluginSetting(context.Background()))
	require.Empty(t, repo.sets, "重复迁移不应重写白名单")
}

func TestMigrateCodexBodyFingerprintToSignals(t *testing.T) {
	repo := &codexPolicyRepoStub{values: map[string]string{
		SettingKeyCodexCLIOnlyAllowBodyEngineFingerprint: "true",
	}}
	svc := NewSettingService(repo, &config.Config{})
	require.NoError(t, svc.MigrateCodexBodyFingerprintToSignals(context.Background()))
	signals, ok := openai.ParseEngineFingerprintSignals(repo.values[SettingKeyCodexCLIOnlyEngineFingerprintSignals])
	require.True(t, ok)
	for _, signal := range signals {
		if signal.Type == openai.FingerprintSignalBodyPath {
			require.True(t, signal.Required)
			return
		}
	}
	t.Fatal("missing body fingerprint signal")
}

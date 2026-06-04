//go:build unit

package service

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNormalizeRegistrationEmailSuffixWhitelist(t *testing.T) {
	got, err := NormalizeRegistrationEmailSuffixWhitelist([]string{"example.com", "@EXAMPLE.COM", " @foo.bar ", "*.EDU.CN"})
	require.NoError(t, err)
	require.Equal(t, []string{"@example.com", "@foo.bar", "*.edu.cn"}, got)
}

func TestNormalizeRegistrationEmailSuffixWhitelist_Invalid(t *testing.T) {
	for _, item := range []string{"@invalid_domain", "*.", "*", "*.@", "*.foo"} {
		t.Run(item, func(t *testing.T) {
			_, err := NormalizeRegistrationEmailSuffixWhitelist([]string{item})
			require.Error(t, err)
		})
	}
}

func TestParseRegistrationEmailSuffixWhitelist(t *testing.T) {
	got := ParseRegistrationEmailSuffixWhitelist(`["example.com","@foo.bar","*.EDU.CN","@invalid_domain","*.foo"]`)
	require.Equal(t, []string{"@example.com", "@foo.bar", "*.edu.cn"}, got)
}

func TestNormalizeRegistrationEmailDomainBlacklist(t *testing.T) {
	got, err := NormalizeRegistrationEmailDomainBlacklist([]string{"mail-temp.com", "@MAIL-TEMP.com", " @blocked.example ", "*.DISPOSABLE.NET"})
	require.NoError(t, err)
	require.Equal(t, []string{"@mail-temp.com", "@blocked.example", "*.disposable.net"}, got)
}

func TestNormalizeRegistrationEmailDomainBlacklist_Invalid(t *testing.T) {
	for _, item := range []string{"@invalid_domain", "*.", "*", "*.@", "*.foo"} {
		t.Run(item, func(t *testing.T) {
			_, err := NormalizeRegistrationEmailDomainBlacklist([]string{item})
			require.Error(t, err)
		})
	}
}

func TestParseRegistrationEmailDomainBlacklist(t *testing.T) {
	got := ParseRegistrationEmailDomainBlacklist(`["mail-temp.com","@blocked.example","*.DISPOSABLE.NET","@invalid_domain","*.foo"]`)
	require.Equal(t, []string{"@mail-temp.com", "@blocked.example", "*.disposable.net"}, got)
}

func TestIsRegistrationEmailSuffixAllowed(t *testing.T) {
	require.True(t, IsRegistrationEmailSuffixAllowed("user@example.com", []string{"@example.com"}))
	require.False(t, IsRegistrationEmailSuffixAllowed("user@sub.example.com", []string{"@example.com"}))
	require.True(t, IsRegistrationEmailSuffixAllowed("user@qq.com", []string{"@qq.com"}))
	require.False(t, IsRegistrationEmailSuffixAllowed("user@sub.qq.com", []string{"@qq.com"}))
	require.True(t, IsRegistrationEmailSuffixAllowed("student@cs.edu.cn", []string{"*.edu.cn"}))
	require.True(t, IsRegistrationEmailSuffixAllowed("student@edu.cn", []string{"*.edu.cn"}))
	require.False(t, IsRegistrationEmailSuffixAllowed("student@foo.cn", []string{"*.edu.cn"}))
	require.True(t, IsRegistrationEmailSuffixAllowed("user@a.com", []string{"@a.com", "*.b.cn"}))
	require.True(t, IsRegistrationEmailSuffixAllowed("user@school.b.cn", []string{"@a.com", "*.b.cn"}))
	require.True(t, IsRegistrationEmailSuffixAllowed("user@b.cn", []string{"@a.com", "*.b.cn"}))
	require.False(t, IsRegistrationEmailSuffixAllowed("user@c.cn", []string{"@a.com", "*.b.cn"}))
	require.True(t, IsRegistrationEmailSuffixAllowed("user@any.com", []string{}))
}

func TestIsRegistrationEmailDomainBlocked(t *testing.T) {
	require.True(t, IsRegistrationEmailDomainBlocked("user@mail-temp.com", []string{"@mail-temp.com"}))
	require.False(t, IsRegistrationEmailDomainBlocked("user@sub.mail-temp.com", []string{"@mail-temp.com"}))
	require.True(t, IsRegistrationEmailDomainBlocked("user@sub.disposable.net", []string{"*.disposable.net"}))
	require.True(t, IsRegistrationEmailDomainBlocked("user@disposable.net", []string{"*.disposable.net"}))
	require.False(t, IsRegistrationEmailDomainBlocked("user@example.com", []string{"@mail-temp.com", "*.disposable.net"}))
	require.False(t, IsRegistrationEmailDomainBlocked("invalid", []string{"@mail-temp.com"}))
}

func TestHasRegistrationEmailAlias(t *testing.T) {
	require.True(t, HasRegistrationEmailAlias("user+tag@example.com"))
	require.True(t, HasRegistrationEmailAlias("first.last@gmail.com"))
	require.True(t, HasRegistrationEmailAlias("first.last@googlemail.com"))
	require.False(t, HasRegistrationEmailAlias("first.last@example.com"))
	require.False(t, HasRegistrationEmailAlias("plain@gmail.com"))
	require.False(t, HasRegistrationEmailAlias("invalid"))
}

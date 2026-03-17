package cli

import (
	"os"
	"testing"

	"github.com/steipete/camsnap/internal/config"
)

func TestResolveSlackToken(t *testing.T) {
	cfg := config.Config{
		Slack: config.SlackConfig{
			Token: "config-token",
		},
	}

	// 1. Flag precedence
	if got := resolveSlackToken("flag-token", cfg); got != "flag-token" {
		t.Errorf("resolveSlackToken flag precedence = %q, want %q", got, "flag-token")
	}

	// 2. Env var precedence
	os.Setenv("SLACK_TOKEN", "env-token")
	defer os.Unsetenv("SLACK_TOKEN")
	if got := resolveSlackToken("", cfg); got != "env-token" {
		t.Errorf("resolveSlackToken env precedence = %q, want %q", got, "env-token")
	}

	// 3. Config fallback
	os.Unsetenv("SLACK_TOKEN")
	if got := resolveSlackToken("", cfg); got != "config-token" {
		t.Errorf("resolveSlackToken config fallback = %q, want %q", got, "config-token")
	}
}

func TestResolveSlackChannel(t *testing.T) {
	cfg := config.Config{
		Slack: config.SlackConfig{
			DefaultChannel: "config-channel",
		},
	}

	// 1. Flag precedence
	if got := resolveSlackChannel("flag-channel", cfg); got != "flag-channel" {
		t.Errorf("resolveSlackChannel flag precedence = %q, want %q", got, "flag-channel")
	}

	// 2. Config fallback
	if got := resolveSlackChannel("", cfg); got != "config-channel" {
		t.Errorf("resolveSlackChannel config fallback = %q, want %q", got, "config-channel")
	}
}

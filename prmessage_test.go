package main

import "testing"

func TestParseThreadTS(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{"empty", "", ""},
		{"raw ts unchanged", "1700000000.123456", "1700000000.123456"},
		{"permalink", "https://acme.slack.com/archives/C01ABCD/p1700000000123456", "1700000000.123456"},
		{"permalink with thread query", "https://acme.slack.com/archives/C01ABCD/p1700000000123456?thread_ts=1699999999.000100", "1700000000.123456"},
		{"trims whitespace", "  1700000000.123456  ", "1700000000.123456"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := parseThreadTS(tt.in); got != tt.want {
				t.Errorf("parseThreadTS(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestResolveChannel(t *testing.T) {
	cfg := &Config{
		Slack:    SlackConfig{Channel: "C_DEFAULT"},
		Channels: map[string]string{"repo-a": "C_A"},
	}
	tests := []struct {
		name     string
		override string
		project  string
		want     string
	}{
		{"override wins", "C_OVERRIDE", "repo-a", "C_OVERRIDE"},
		{"per-project map", "", "repo-a", "C_A"},
		{"falls back to default", "", "repo-unknown", "C_DEFAULT"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := cfg.resolveChannel(tt.override, tt.project); got != tt.want {
				t.Errorf("resolveChannel(%q, %q) = %q, want %q", tt.override, tt.project, got, tt.want)
			}
		})
	}
}

func TestIsProtectedBranch(t *testing.T) {
	defaults := &Config{}
	if !defaults.isProtectedBranch("main") || !defaults.isProtectedBranch("master") {
		t.Error("expected main/master protected by default")
	}
	if defaults.isProtectedBranch("feature/x") {
		t.Error("feature branch should not be protected by default")
	}

	custom := &Config{ProtectedBranches: []string{"release", "develop"}}
	if !custom.isProtectedBranch("release") {
		t.Error("expected configured branch to be protected")
	}
	if custom.isProtectedBranch("main") {
		t.Error("main should not be protected when a custom list is set")
	}
}

package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadJiraConfig(t *testing.T) {
	tests := map[string]struct {
		content    string
		wantServer string
		wantLogin  string
		wantErr    string
	}{
		"valid config": {
			content:    "auth_type: basic\nserver: https://example.atlassian.net\nlogin: alice@example.com\n",
			wantServer: "https://example.atlassian.net",
			wantLogin:  "alice@example.com",
		},
		"missing server": {
			content: "login: alice@example.com\n",
			wantErr: "server not found",
		},
		"missing login": {
			content: "server: https://example.atlassian.net\n",
			wantErr: "login not found",
		},
		"fields among other config": {
			content:    "auth_type: basic\nboard:\n    id: 1\nserver: https://test.atlassian.net\nepic:\n    name: cf\nlogin: bob@test.com\n",
			wantServer: "https://test.atlassian.net",
			wantLogin:  "bob@test.com",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			dir := t.TempDir()
			configPath := filepath.Join(dir, ".config.yml")
			if err := os.WriteFile(configPath, []byte(tc.content), 0o600); err != nil {
				t.Fatalf("failed to write config: %v", err)
			}
			t.Setenv("JIRA_CONFIG_FILE", configPath)

			cfg, err := loadJiraConfig()
			if tc.wantErr != "" {
				if err == nil {
					t.Fatalf("expected error containing %q, got nil", tc.wantErr)
				}
				if !strings.Contains(err.Error(), tc.wantErr) {
					t.Fatalf("error %q does not contain %q", err, tc.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if cfg.server != tc.wantServer {
				t.Errorf("server = %q, want %q", cfg.server, tc.wantServer)
			}
			if cfg.login != tc.wantLogin {
				t.Errorf("login = %q, want %q", cfg.login, tc.wantLogin)
			}
		})
	}
}

func TestLoadJiraConfigFileNotFound(t *testing.T) {
	t.Setenv("JIRA_CONFIG_FILE", "/nonexistent/path/.config.yml")

	_, err := loadJiraConfig()
	if err == nil {
		t.Fatal("expected error for missing config file")
	}
	if !strings.Contains(err.Error(), "cannot open") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestLoadJiraConfigEnvOverride(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "custom.yml")
	if err := os.WriteFile(configPath, []byte("server: https://custom.atlassian.net\nlogin: custom@test.com\n"), 0o600); err != nil {
		t.Fatalf("failed to write config: %v", err)
	}
	t.Setenv("JIRA_CONFIG_FILE", configPath)

	cfg, err := loadJiraConfig()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.server != "https://custom.atlassian.net" {
		t.Errorf("server = %q, want https://custom.atlassian.net", cfg.server)
	}
}

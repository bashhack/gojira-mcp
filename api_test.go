package main

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
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

// writeJiraConfigPointingTo creates a jira-cli config file pointing at the
// given server URL and sets JIRA_CONFIG_FILE for the duration of the test.
func writeJiraConfigPointingTo(t *testing.T, server string) {
	t.Helper()
	dir := t.TempDir()
	configPath := filepath.Join(dir, ".config.yml")
	content := "server: " + server + "\nlogin: test@example.com\n"
	if err := os.WriteFile(configPath, []byte(content), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
	t.Setenv("JIRA_CONFIG_FILE", configPath)
}

func TestDefaultJiraAPIFetchGet(t *testing.T) {
	var capturedAuth, capturedAccept, capturedContentType, capturedMethod, capturedPath string
	var capturedBody []byte
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedAuth = r.Header.Get("Authorization")
		capturedAccept = r.Header.Get("Accept")
		capturedContentType = r.Header.Get("Content-Type")
		capturedMethod = r.Method
		capturedPath = r.URL.Path
		capturedBody, _ = io.ReadAll(r.Body) //nolint:errcheck // test handler
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"ok":true}`)) //nolint:errcheck // test handler
	}))
	t.Cleanup(srv.Close)

	writeJiraConfigPointingTo(t, srv.URL)
	t.Setenv("JIRA_API_TOKEN", "tok-123")

	body, err := defaultJiraAPIFetch(context.Background(), "GET", "/rest/api/3/issue/X-1", nil)
	if err != nil {
		t.Fatalf("unexpected: %v", err)
	}
	if string(body) != `{"ok":true}` {
		t.Errorf("body = %q, want {\"ok\":true}", string(body))
	}
	if capturedMethod != "GET" {
		t.Errorf("method = %q, want GET", capturedMethod)
	}
	if capturedPath != "/rest/api/3/issue/X-1" {
		t.Errorf("path = %q", capturedPath)
	}
	if !strings.HasPrefix(capturedAuth, "Basic ") {
		t.Errorf("auth header = %q, want Basic prefix", capturedAuth)
	}
	if capturedAccept != "application/json" {
		t.Errorf("accept = %q", capturedAccept)
	}
	if capturedContentType != "" {
		t.Errorf("nil-body request should not set Content-Type, got %q", capturedContentType)
	}
	if len(capturedBody) != 0 {
		t.Errorf("expected empty request body, got %d bytes", len(capturedBody))
	}
}

func TestDefaultJiraAPIFetchPostWithBody(t *testing.T) {
	var capturedContentType, capturedMethod string
	var capturedBody []byte
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedContentType = r.Header.Get("Content-Type")
		capturedMethod = r.Method
		capturedBody, _ = io.ReadAll(r.Body) //nolint:errcheck // test handler
		w.WriteHeader(http.StatusNoContent)
	}))
	t.Cleanup(srv.Close)

	writeJiraConfigPointingTo(t, srv.URL)
	t.Setenv("JIRA_API_TOKEN", "tok-123")

	payload := []byte(`{"hello":"world"}`)
	body, err := defaultJiraAPIFetch(context.Background(), "POST", "/rest/api/3/whatever", payload)
	if err != nil {
		t.Fatalf("unexpected: %v", err)
	}
	if len(body) != 0 {
		t.Errorf("expected empty 204 body, got %q", string(body))
	}
	if capturedMethod != "POST" {
		t.Errorf("method = %q", capturedMethod)
	}
	if capturedContentType != "application/json" {
		t.Errorf("Content-Type = %q, want application/json", capturedContentType)
	}
	if string(capturedBody) != `{"hello":"world"}` {
		t.Errorf("body = %q", string(capturedBody))
	}
}

func TestDefaultJiraAPIFetchMissingToken(t *testing.T) {
	writeJiraConfigPointingTo(t, "https://example.atlassian.net")
	t.Setenv("JIRA_API_TOKEN", "")

	_, err := defaultJiraAPIFetch(context.Background(), "GET", "/x", nil)
	if err == nil {
		t.Fatal("expected error when token missing")
	}
	if !strings.Contains(err.Error(), "JIRA_API_TOKEN") {
		t.Errorf("error = %v, want mention of JIRA_API_TOKEN", err)
	}
}

func TestDefaultJiraAPIFetchNon2xx(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(`{"errors":["nope"]}`)) //nolint:errcheck // test handler
	}))
	t.Cleanup(srv.Close)

	writeJiraConfigPointingTo(t, srv.URL)
	t.Setenv("JIRA_API_TOKEN", "tok-123")

	_, err := defaultJiraAPIFetch(context.Background(), "GET", "/x", nil)
	if err == nil {
		t.Fatal("expected error on 403")
	}
	if !strings.Contains(err.Error(), "403") || !strings.Contains(err.Error(), "nope") {
		t.Errorf("error = %v, want status code + body", err)
	}
}

func TestDefaultJiraAPIFetchConfigError(t *testing.T) {
	t.Setenv("JIRA_CONFIG_FILE", "/nonexistent/.config.yml")
	t.Setenv("JIRA_API_TOKEN", "tok-123")

	_, err := defaultJiraAPIFetch(context.Background(), "GET", "/x", nil)
	if err == nil {
		t.Fatal("expected error from missing config")
	}
	if !strings.Contains(err.Error(), "cannot open") {
		t.Errorf("error = %v, want config-open failure", err)
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

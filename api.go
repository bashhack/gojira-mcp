package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// jiraConfig holds the subset of jira-cli configuration needed
// for direct REST API calls.
type jiraConfig struct {
	server string
	login  string
}

// jiraAPIFetcher is the function used to make Jira REST API requests.
// Tests replace this with a fake to avoid network calls.
var jiraAPIFetcher = defaultJiraAPIFetch

var apiHTTPClient = &http.Client{Timeout: 15 * time.Second}

// loadJiraConfig reads the jira-cli config file and extracts the
// server URL and login email. It checks JIRA_CONFIG_FILE first,
// then falls back to ~/.config/.jira/.config.yml.
func loadJiraConfig() (cfg jiraConfig, retErr error) {
	path := os.Getenv("JIRA_CONFIG_FILE")
	if path == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return jiraConfig{}, fmt.Errorf("cannot determine home directory: %w", err)
		}
		path = filepath.Join(home, ".config", ".jira", ".config.yml")
	}

	f, err := os.Open(filepath.Clean(path))
	if err != nil {
		return jiraConfig{}, fmt.Errorf("cannot open jira config %s: %w", path, err)
	}
	defer func() {
		if cErr := f.Close(); cErr != nil {
			retErr = errors.Join(retErr, cErr)
		}
	}()
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		if k, v, ok := strings.Cut(line, ": "); ok {
			switch k {
			case "server":
				cfg.server = v
			case "login":
				cfg.login = v
			}
		}
	}
	if err := scanner.Err(); err != nil {
		return jiraConfig{}, fmt.Errorf("error reading jira config: %w", err)
	}

	if cfg.server == "" {
		return jiraConfig{}, fmt.Errorf("server not found in jira config %s", path)
	}
	if cfg.login == "" {
		return jiraConfig{}, fmt.Errorf("login not found in jira config %s", path)
	}
	return cfg, nil
}

// defaultJiraAPIFetch makes an authenticated HTTP request to the Jira
// REST API. It reads credentials from the jira-cli config and
// JIRA_API_TOKEN environment variable. Pass a nil body for GET; pass a
// JSON-encoded body for PUT/POST and the Content-Type header is set
// automatically.
func defaultJiraAPIFetch(ctx context.Context, method, path string, body []byte) (_ []byte, retErr error) {
	cfg, err := loadJiraConfig()
	if err != nil {
		return nil, err
	}

	token := os.Getenv("JIRA_API_TOKEN")
	if token == "" {
		return nil, fmt.Errorf("JIRA_API_TOKEN environment variable not set")
	}

	var reqBody io.Reader = http.NoBody
	if body != nil {
		reqBody = bytes.NewReader(body)
	}

	req, err := http.NewRequestWithContext(ctx, method, cfg.server+path, reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	creds := base64.StdEncoding.EncodeToString([]byte(cfg.login + ":" + token))
	req.Header.Set("Authorization", "Basic "+creds)
	req.Header.Set("Accept", "application/json")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := apiHTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer func() {
		if cErr := resp.Body.Close(); cErr != nil {
			retErr = errors.Join(retErr, cErr)
		}
	}()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("jira API returned %s: %s", resp.Status, string(respBody))
	}
	return respBody, nil
}

// Package services provides external service integrations for the create wizard.
package services

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// GitHubClient provides access to GitHub's public APIs.
type GitHubClient struct {
	httpClient *http.Client
	userAgent  string
	// baseURL and apiBaseURL are used for testing; empty means use defaults
	baseURL    string // For SSH keys (github.com)
	apiBaseURL string // For API (api.github.com)
}

// Profile holds fetched GitHub profile data.
type Profile struct {
	ID    int64  `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
	Login string `json:"login"`
}

// NoReplyEmail returns the GitHub noreply email address.
func (p *Profile) NoReplyEmail() string {
	if p.ID == 0 || p.Login == "" {
		return ""
	}
	return fmt.Sprintf("%d+%s@users.noreply.github.com", p.ID, p.Login)
}

// BestEmail returns the public email if available, otherwise noreply.
func (p *Profile) BestEmail() string {
	if p.Email != "" {
		return p.Email
	}
	return p.NoReplyEmail()
}

// NewGitHubClient creates a new GitHub client with default settings.
func NewGitHubClient() *GitHubClient {
	return &GitHubClient{
		httpClient: &http.Client{Timeout: 10 * time.Second},
		userAgent:  "cloud-init-cli",
	}
}

// FetchSSHKeys fetches public SSH keys from GitHub for a user.
func (c *GitHubClient) FetchSSHKeys(username string) ([]string, error) {
	baseURL := c.baseURL
	if baseURL == "" {
		baseURL = "https://github.com"
	}
	url := fmt.Sprintf("%s/%s.keys", baseURL, username)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", c.userAgent)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to GitHub: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("GitHub user '%s' not found", username)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GitHub returned status %d", resp.StatusCode)
	}

	// Limit to 1MB to prevent memory exhaustion
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	lines := strings.Split(strings.TrimSpace(string(body)), "\n")
	keys := make([]string, 0, len(lines))
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" {
			keys = append(keys, line)
		}
	}

	return keys, nil
}

// FetchProfile fetches public profile from GitHub API.
func (c *GitHubClient) FetchProfile(username string) (*Profile, error) {
	apiBaseURL := c.apiBaseURL
	if apiBaseURL == "" {
		apiBaseURL = "https://api.github.com"
	}
	url := fmt.Sprintf("%s/users/%s", apiBaseURL, username)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/vnd.github.v3+json")
	req.Header.Set("User-Agent", c.userAgent)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to GitHub API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("GitHub user '%s' not found", username)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GitHub API returned status %d", resp.StatusCode)
	}

	// Limit to 1MB to prevent memory exhaustion
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var profile Profile
	if err := json.Unmarshal(body, &profile); err != nil {
		return nil, fmt.Errorf("failed to parse GitHub response: %w", err)
	}

	return &profile, nil
}

// UserData holds the results of fetching both SSH keys and profile data.
type UserData struct {
	Keys       []string
	Profile    *Profile
	KeysErr    error
	ProfileErr error
}

// FetchUserData fetches both SSH keys and profile data for a user.
func (c *GitHubClient) FetchUserData(username string) *UserData {
	data := &UserData{}

	// Fetch SSH keys
	data.Keys, data.KeysErr = c.FetchSSHKeys(username)

	// Fetch profile
	data.Profile, data.ProfileErr = c.FetchProfile(username)

	return data
}

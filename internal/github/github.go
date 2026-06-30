// Package github provides GitHub API integration for reasonix: OAuth
// authentication, repository listing, and search.
package github

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Client handles GitHub API requests with an optional OAuth token.
type Client struct {
	token  string
	client *http.Client
}

// NewClient creates a GitHub API client. token may be empty for
// unauthenticated access (lower rate limits).
func NewClient(token string) *Client {
	return &Client{
		token:  token,
		client: &http.Client{Timeout: 15 * time.Second},
	}
}

// RepoInfo is a lightweight view of a GitHub repository.
type RepoInfo struct {
	Name        string `json:"name"`
	FullName    string `json:"full_name"`
	Description string `json:"description"`
	Private     bool   `json:"private"`
	UpdatedAt   string `json:"updated_at"`
	CloneURL    string `json:"clone_url"`
	HTMLURL     string `json:"html_url"`
	Language    string `json:"language"`
}

func (c *Client) do(ctx context.Context, method, path string) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, method, "https://api.github.com"+path, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "reasonix")
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}
	return c.client.Do(req)
}

// ListUserRepos returns the authenticated user's repositories, sorted by
// most recently updated. max controls how many to return (capped at 100).
func (c *Client) ListUserRepos(ctx context.Context, max int) ([]RepoInfo, error) {
	if max <= 0 || max > 100 {
		max = 30
	}
	resp, err := c.do(ctx, "GET", fmt.Sprintf("/user/repos?sort=updated&per_page=%d", max))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return nil, fmt.Errorf("github api: %s %s", resp.Status, string(body))
	}
	var repos []RepoInfo
	if err := json.NewDecoder(resp.Body).Decode(&repos); err != nil {
		return nil, err
	}
	return repos, nil
}

// ListStarredRepos returns the authenticated user's starred repos.
func (c *Client) ListStarredRepos(ctx context.Context, max int) ([]RepoInfo, error) {
	if max <= 0 || max > 100 {
		max = 30
	}
	resp, err := c.do(ctx, "GET", fmt.Sprintf("/user/starred?per_page=%d", max))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return nil, fmt.Errorf("github api: %s %s", resp.Status, string(body))
	}
	var repos []RepoInfo
	if err := json.NewDecoder(resp.Body).Decode(&repos); err != nil {
		return nil, err
	}
	return repos, nil
}

// SearchRepos searches GitHub repositories by query string.
func (c *Client) SearchRepos(ctx context.Context, query string, max int) ([]RepoInfo, error) {
	if max <= 0 || max > 100 {
		max = 10
	}
	path := fmt.Sprintf("/search/repositories?q=%s&per_page=%d&sort=stars",
		url.QueryEscape(query), max)
	resp, err := c.do(ctx, "GET", path)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return nil, fmt.Errorf("github api: %s %s", resp.Status, string(body))
	}
	var result struct {
		Items []RepoInfo `json:"items"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	return result.Items, nil
}

// OAuthLogin starts the device OAuth flow. Returns the verification URL
// and user code the user must enter in their browser.
func OAuthLogin(ctx context.Context) (*OAuthResponse, error) {
	clientID := os.Getenv("REASONIX_GITHUB_CLIENT_ID")
	if clientID == "" {
		clientID = "Iv23liqEX0RgmVNHSkVS" // reasonix development app
	}

	body := url.Values{
		"client_id": {clientID},
		"scope":     {"repo,read:org"},
	}.Encode()

	req, err := http.NewRequestWithContext(ctx, "POST",
		"https://github.com/login/device/code", strings.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result OAuthResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	if result.Error != "" {
		return nil, fmt.Errorf("github oauth: %s — %s", result.Error, result.ErrorDescription)
	}
	return &result, nil
}

// OAuthResponse is the device-code response from GitHub.
type OAuthResponse struct {
	DeviceCode      string `json:"device_code"`
	UserCode        string `json:"user_code"`
	VerificationURI string `json:"verification_uri"`
	ExpiresIn       int    `json:"expires_in"`
	Interval        int    `json:"interval"`
	Error           string `json:"error"`
	ErrorDescription string `json:"error_description"`
}

// PollForToken polls GitHub for an access token after the user authorizes.
func PollForToken(ctx context.Context, deviceCode string, intervalSec int) (string, error) {
	if intervalSec <= 0 {
		intervalSec = 5
	}
	clientID := os.Getenv("REASONIX_GITHUB_CLIENT_ID")
	if clientID == "" {
		clientID = "Iv23liqEX0RgmVNHSkVS"
	}

	pollBody := url.Values{
		"client_id":   {clientID},
		"device_code": {deviceCode},
		"grant_type":  {"urn:ietf:params:oauth:grant-type:device_code"},
	}

	for {
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		default:
		}

		req, err := http.NewRequestWithContext(ctx, "POST",
			"https://github.com/login/oauth/access_token",
			strings.NewReader(pollBody.Encode()))
		if err != nil {
			return "", err
		}
		req.Header.Set("Accept", "application/json")
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			time.Sleep(time.Duration(intervalSec) * time.Second)
			continue
		}

		var result struct {
			AccessToken string `json:"access_token"`
			Error       string `json:"error"`
		}
		json.NewDecoder(resp.Body).Decode(&result)
		resp.Body.Close()

		if result.AccessToken != "" {
			return result.AccessToken, nil
		}
		if result.Error == "authorization_pending" {
			time.Sleep(time.Duration(intervalSec) * time.Second)
			continue
		}
		if result.Error != "" {
			return "", fmt.Errorf("github oauth: %s", result.Error)
		}

		time.Sleep(time.Duration(intervalSec) * time.Second)
	}
}

// TokenPath returns where the GitHub OAuth token is stored.
func TokenPath() string {
	return credentialsPath()
}

// LoadToken reads the stored GitHub token, or "" if none.
func LoadToken() string {
	data, err := os.ReadFile(TokenPath())
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(data))
}

// SaveToken persists a GitHub OAuth token to the credential store.
func SaveToken(token string) error {
	dir := filepath.Dir(TokenPath())
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}
	return os.WriteFile(TokenPath(), []byte(token), 0600)
}

func credentialsPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".reasonix", "credentials")
}

package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
)

// TokenManager handles token refresh and validation.
type TokenManager struct {
	httpClient *http.Client
	store      *Store
}

// NewTokenManager creates a new token manager.
func NewTokenManager(store *Store, httpClient *http.Client) *TokenManager {
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 30 * time.Second}
	}
	return &TokenManager{
		httpClient: httpClient,
		store:      store,
	}
}

// EnsureValidToken ensures the credentials have a valid access token.
// If the token is expired, it will attempt to refresh it.
func (t *TokenManager) EnsureValidToken(ctx context.Context, creds *Credentials) (*Credentials, error) {
	if creds == nil {
		return nil, fmt.Errorf("credentials are nil")
	}

	if !creds.IsExpired() {
		return creds, nil
	}

	log.Debug("Access token expired, refreshing...")
	return t.RefreshToken(ctx, creds)
}

// RefreshToken obtains a new access token using the refresh token.
func (t *TokenManager) RefreshToken(ctx context.Context, creds *Credentials) (*Credentials, error) {
	if creds == nil {
		return nil, fmt.Errorf("credentials are nil")
	}

	if creds.RefreshToken == "" {
		return nil, fmt.Errorf("no refresh token available")
	}

	data := url.Values{}
	data.Set("client_id", ClientID)
	data.Set("client_secret", ClientSecret)
	data.Set("grant_type", "refresh_token")
	data.Set("refresh_token", creds.RefreshToken)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://oauth2.googleapis.com/token", strings.NewReader(data.Encode()))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Host", "oauth2.googleapis.com")
	req.Header.Set("User-Agent", DefaultAgent)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := t.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("execute request: %w", err)
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("refresh failed with status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	var tokenResp TokenResponse
	if err := json.Unmarshal(bodyBytes, &tokenResp); err != nil {
		return nil, fmt.Errorf("parse response: %w", err)
	}

	// Update credentials with new token
	now := time.Now()
	creds.AccessToken = tokenResp.AccessToken
	if tokenResp.RefreshToken != "" {
		creds.RefreshToken = tokenResp.RefreshToken
	}
	creds.ExpiresIn = tokenResp.ExpiresIn
	creds.Timestamp = now.UnixMilli()
	creds.Expired = now.Add(time.Duration(tokenResp.ExpiresIn) * time.Second).Format(time.RFC3339)

	// Persist updated credentials
	if t.store != nil {
		if err := t.store.Update(creds); err != nil {
			log.Warnf("Failed to persist refreshed credentials: %v", err)
		}
	}

	log.Debug("Token refreshed successfully")
	return creds, nil
}

// ValidateToken checks if the access token is still valid by making a test API call.
func (t *TokenManager) ValidateToken(ctx context.Context, accessToken string) bool {
	if accessToken == "" {
		return false
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://www.googleapis.com/oauth2/v1/userinfo?alt=json", nil)
	if err != nil {
		return false
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)

	resp, err := t.httpClient.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	return resp.StatusCode >= 200 && resp.StatusCode < 300
}
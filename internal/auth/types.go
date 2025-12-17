// Package auth provides OAuth authentication for the Antigravity API.
package auth

import (
	"time"
)

// Credentials represents stored authentication credentials.
type Credentials struct {
	// Type identifies the credential type (always "antigravity")
	Type string `json:"type"`

	// AccessToken is the current OAuth access token
	AccessToken string `json:"access_token"`

	// RefreshToken is used to obtain new access tokens
	RefreshToken string `json:"refresh_token"`

	// ExpiresIn is the token lifetime in seconds
	ExpiresIn int64 `json:"expires_in"`

	// Timestamp is when the token was obtained (Unix milliseconds)
	Timestamp int64 `json:"timestamp"`

	// Expired is the RFC3339 formatted expiry time
	Expired string `json:"expired"`

	// Email is the user's email address (optional)
	Email string `json:"email,omitempty"`

	// ProjectID is the GCP project ID for API calls
	ProjectID string `json:"project_id,omitempty"`

	// UserAgent is the custom user agent string (optional)
	UserAgent string `json:"user_agent,omitempty"`

	// BaseURL is a custom API base URL (optional)
	BaseURL string `json:"base_url,omitempty"`
}

// TokenExpiry returns the token expiration time.
func (c *Credentials) TokenExpiry() time.Time {
	if c.Expired != "" {
		if parsed, err := time.Parse(time.RFC3339, c.Expired); err == nil {
			return parsed
		}
	}
	if c.ExpiresIn > 0 && c.Timestamp > 0 {
		return time.Unix(0, c.Timestamp*int64(time.Millisecond)).Add(time.Duration(c.ExpiresIn) * time.Second)
	}
	return time.Time{}
}

// IsExpired checks if the access token is expired or about to expire.
// Uses a 50-minute skew to refresh before actual expiration.
func (c *Credentials) IsExpired() bool {
	const refreshSkew = 50 * time.Minute
	expiry := c.TokenExpiry()
	if expiry.IsZero() {
		return true
	}
	return time.Now().Add(refreshSkew).After(expiry)
}

// TokenResponse represents the OAuth token endpoint response.
type TokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int64  `json:"expires_in"`
	TokenType    string `json:"token_type"`
}

// UserInfo represents the user info endpoint response.
type UserInfo struct {
	Email string `json:"email"`
}

// LoadCodeAssistResponse represents the loadCodeAssist API response.
type LoadCodeAssistResponse struct {
	CloudAICompanionProject string `json:"cloudaicompanionProject"`
}
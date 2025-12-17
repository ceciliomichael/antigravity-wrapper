package auth

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
)

// OAuth constants for Antigravity authentication.
const (
	ClientID       = "1071006060591-tmhssin2h21lcre235vtolojh4g403ep.apps.googleusercontent.com"
	ClientSecret   = "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"
	CallbackPort   = 51121
	DefaultAgent   = "antigravity/1.11.5 windows/amd64"
	APIEndpoint    = "https://cloudcode-pa.googleapis.com"
	APIVersion     = "v1internal"
)

var oauthScopes = []string{
	"https://www.googleapis.com/auth/cloud-platform",
	"https://www.googleapis.com/auth/userinfo.email",
	"https://www.googleapis.com/auth/userinfo.profile",
	"https://www.googleapis.com/auth/cclog",
	"https://www.googleapis.com/auth/experimentsandconfigs",
}

// Authenticator handles OAuth login flow for Antigravity.
type Authenticator struct {
	httpClient *http.Client
	store      *Store
}

// NewAuthenticator creates a new authenticator instance.
func NewAuthenticator(store *Store, httpClient *http.Client) *Authenticator {
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 30 * time.Second}
	}
	return &Authenticator{
		httpClient: httpClient,
		store:      store,
	}
}

// LoginOptions configures the login behavior.
type LoginOptions struct {
	NoBrowser bool
}

// callbackResult holds the OAuth callback response.
type callbackResult struct {
	Code  string
	Error string
	State string
}

// Login initiates the OAuth login flow.
func (a *Authenticator) Login(ctx context.Context, opts *LoginOptions) (*Credentials, error) {
	if opts == nil {
		opts = &LoginOptions{}
	}

	state, err := generateRandomState()
	if err != nil {
		return nil, fmt.Errorf("generate state: %w", err)
	}

	srv, port, cbChan, err := a.startCallbackServer()
	if err != nil {
		return nil, fmt.Errorf("start callback server: %w", err)
	}
	defer func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = srv.Shutdown(shutdownCtx)
	}()

	redirectURI := fmt.Sprintf("http://localhost:%d/oauth-callback", port)
	authURL := buildAuthURL(redirectURI, state)

	fmt.Println("Opening browser for Antigravity authentication...")
	fmt.Printf("\nVisit the following URL to authenticate:\n%s\n\n", authURL)
	fmt.Println("Waiting for authentication callback...")

	var cbRes callbackResult
	select {
	case res := <-cbChan:
		cbRes = res
	case <-time.After(5 * time.Minute):
		return nil, fmt.Errorf("authentication timed out")
	case <-ctx.Done():
		return nil, ctx.Err()
	}

	if cbRes.Error != "" {
		return nil, fmt.Errorf("authentication failed: %s", cbRes.Error)
	}
	if cbRes.State != state {
		return nil, fmt.Errorf("invalid state parameter")
	}
	if cbRes.Code == "" {
		return nil, fmt.Errorf("missing authorization code")
	}

	tokenResp, err := a.exchangeCode(ctx, cbRes.Code, redirectURI)
	if err != nil {
		return nil, fmt.Errorf("token exchange: %w", err)
	}

	email := ""
	if tokenResp.AccessToken != "" {
		if info, err := a.fetchUserInfo(ctx, tokenResp.AccessToken); err == nil {
			email = strings.TrimSpace(info.Email)
		}
	}

	projectID := ""
	if tokenResp.AccessToken != "" {
		if pid, err := a.fetchProjectID(ctx, tokenResp.AccessToken); err == nil {
			projectID = pid
			log.Infof("Obtained project ID: %s", projectID)
		} else {
			log.Warnf("Failed to fetch project ID: %v", err)
		}
	}

	now := time.Now()
	creds := &Credentials{
		Type:         "antigravity",
		AccessToken:  tokenResp.AccessToken,
		RefreshToken: tokenResp.RefreshToken,
		ExpiresIn:    tokenResp.ExpiresIn,
		Timestamp:    now.UnixMilli(),
		Expired:      now.Add(time.Duration(tokenResp.ExpiresIn) * time.Second).Format(time.RFC3339),
		Email:        email,
		ProjectID:    projectID,
	}

	if a.store != nil {
		if path, err := a.store.Save(creds); err == nil {
			fmt.Printf("Credentials saved to: %s\n", path)
		} else {
			log.Warnf("Failed to save credentials: %v", err)
		}
	}

	fmt.Println("Authentication successful!")
	if projectID != "" {
		fmt.Printf("Using GCP project: %s\n", projectID)
	}

	return creds, nil
}

func (a *Authenticator) startCallbackServer() (*http.Server, int, <-chan callbackResult, error) {
	addr := fmt.Sprintf(":%d", CallbackPort)
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, 0, nil, err
	}
	port := listener.Addr().(*net.TCPAddr).Port
	resultCh := make(chan callbackResult, 1)

	mux := http.NewServeMux()
	mux.HandleFunc("/oauth-callback", func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		res := callbackResult{
			Code:  strings.TrimSpace(q.Get("code")),
			Error: strings.TrimSpace(q.Get("error")),
			State: strings.TrimSpace(q.Get("state")),
		}
		resultCh <- res
		if res.Code != "" && res.Error == "" {
			w.Header().Set("Content-Type", "text/html")
			_, _ = w.Write([]byte("<h1>Login Successful</h1><p>You can close this window.</p>"))
		} else {
			w.Header().Set("Content-Type", "text/html")
			_, _ = w.Write([]byte("<h1>Login Failed</h1><p>Please check the terminal output.</p>"))
		}
	})

	srv := &http.Server{Handler: mux}
	go func() {
		if err := srv.Serve(listener); err != nil && !strings.Contains(err.Error(), "Server closed") {
			log.Warnf("Callback server error: %v", err)
		}
	}()

	return srv, port, resultCh, nil
}

func (a *Authenticator) exchangeCode(ctx context.Context, code, redirectURI string) (*TokenResponse, error) {
	data := url.Values{}
	data.Set("code", code)
	data.Set("client_id", ClientID)
	data.Set("client_secret", ClientSecret)
	data.Set("redirect_uri", redirectURI)
	data.Set("grant_type", "authorization_code")

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://oauth2.googleapis.com/token", strings.NewReader(data.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := a.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("token exchange failed: status %d: %s", resp.StatusCode, string(body))
	}

	var token TokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&token); err != nil {
		return nil, err
	}
	return &token, nil
}

func (a *Authenticator) fetchUserInfo(ctx context.Context, accessToken string) (*UserInfo, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://www.googleapis.com/oauth2/v1/userinfo?alt=json", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)

	resp, err := a.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return &UserInfo{}, nil
	}

	var info UserInfo
	if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
		return nil, err
	}
	return &info, nil
}

func (a *Authenticator) fetchProjectID(ctx context.Context, accessToken string) (string, error) {
	reqBody := map[string]any{
		"metadata": map[string]string{
			"ideType":    "IDE_UNSPECIFIED",
			"platform":   "PLATFORM_UNSPECIFIED",
			"pluginType": "GEMINI",
		},
	}

	rawBody, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("marshal request: %w", err)
	}

	endpointURL := fmt.Sprintf("%s/%s:loadCodeAssist", APIEndpoint, APIVersion)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpointURL, strings.NewReader(string(rawBody)))
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "google-api-nodejs-client/9.15.1")
	req.Header.Set("X-Goog-Api-Client", "google-cloud-sdk vscode_cloudshelleditor/0.1")

	resp, err := a.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("execute request: %w", err)
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("request failed with status %d: %s", resp.StatusCode, strings.TrimSpace(string(bodyBytes)))
	}

	var loadResp map[string]any
	if err := json.Unmarshal(bodyBytes, &loadResp); err != nil {
		return "", fmt.Errorf("decode response: %w", err)
	}

	if id, ok := loadResp["cloudaicompanionProject"].(string); ok && strings.TrimSpace(id) != "" {
		return strings.TrimSpace(id), nil
	}

	if projectMap, ok := loadResp["cloudaicompanionProject"].(map[string]any); ok {
		if id, ok := projectMap["id"].(string); ok {
			return strings.TrimSpace(id), nil
		}
	}

	return "", fmt.Errorf("no cloudaicompanionProject in response")
}

func buildAuthURL(redirectURI, state string) string {
	params := url.Values{}
	params.Set("access_type", "offline")
	params.Set("client_id", ClientID)
	params.Set("prompt", "consent")
	params.Set("redirect_uri", redirectURI)
	params.Set("response_type", "code")
	params.Set("scope", strings.Join(oauthScopes, " "))
	params.Set("state", state)
	return "https://accounts.google.com/o/oauth2/v2/auth?" + params.Encode()
}

func generateRandomState() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(b), nil
}
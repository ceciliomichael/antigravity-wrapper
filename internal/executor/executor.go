package executor

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/anthropics/antigravity-wrapper/internal/auth"
	"github.com/anthropics/antigravity-wrapper/internal/models"
	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
)

// API endpoints and constants
const (
	BaseURLDaily      = "https://daily-cloudcode-pa.sandbox.googleapis.com"
	BaseURLProd       = "https://cloudcode-pa.googleapis.com"
	StreamPath        = "/v1internal:streamGenerateContent"
	GeneratePath      = "/v1internal:generateContent"
	ModelsPath        = "/v1internal:fetchAvailableModels"
	DefaultUserAgent  = "antigravity/1.11.5 windows/amd64"
	StreamScannerSize = 1024 * 1024 // 1MB buffer for streaming
)

var randSource = rand.New(rand.NewSource(time.Now().UnixNano()))

// Executor handles API requests to the Antigravity backend.
type Executor struct {
	httpClient   *http.Client
	tokenManager *auth.TokenManager
	proxyURL     string
}

// NewExecutor creates a new executor instance.
func NewExecutor(proxyURL string, tokenManager *auth.TokenManager) *Executor {
	return &Executor{
		httpClient:   NewHTTPClient(proxyURL, 0),
		tokenManager: tokenManager,
		proxyURL:     proxyURL,
	}
}

// Request represents an API request.
type Request struct {
	Model   string
	Payload []byte
	Stream  bool
}

// Response represents an API response.
type Response struct {
	StatusCode int
	Body       []byte
	Headers    http.Header
}

// StreamChunk represents a chunk from a streaming response.
type StreamChunk struct {
	Data []byte
	Err  error
}

// Execute performs a non-streaming request.
func (e *Executor) Execute(ctx context.Context, creds *auth.Credentials, req Request) (*Response, error) {
	token, err := e.ensureAccessToken(ctx, creds)
	if err != nil {
		return nil, err
	}

	baseURLs := e.baseURLFallbackOrder(creds)

	for idx, baseURL := range baseURLs {
		httpReq, err := e.buildRequest(ctx, creds, token, req.Model, req.Payload, false, baseURL)
		if err != nil {
			return nil, err
		}

		httpResp, err := e.httpClient.Do(httpReq)
		if err != nil {
			log.Debugf("Request error on %s: %v", baseURL, err)
			if idx+1 < len(baseURLs) {
				continue
			}
			return nil, err
		}

		bodyBytes, err := io.ReadAll(httpResp.Body)
		httpResp.Body.Close()
		if err != nil {
			return nil, err
		}

		if httpResp.StatusCode < 200 || httpResp.StatusCode >= 300 {
			if httpResp.StatusCode == http.StatusTooManyRequests && idx+1 < len(baseURLs) {
				log.Debugf("Rate limited on %s, trying fallback", baseURL)
				continue
			}
			return &Response{
				StatusCode: httpResp.StatusCode,
				Body:       bodyBytes,
				Headers:    httpResp.Header,
			}, fmt.Errorf("API error: status %d", httpResp.StatusCode)
		}

		return &Response{
			StatusCode: httpResp.StatusCode,
			Body:       bodyBytes,
			Headers:    httpResp.Header,
		}, nil
	}

	return nil, fmt.Errorf("all base URLs exhausted")
}

// ExecuteStream performs a streaming request.
func (e *Executor) ExecuteStream(ctx context.Context, creds *auth.Credentials, req Request) (<-chan StreamChunk, error) {
	token, err := e.ensureAccessToken(ctx, creds)
	if err != nil {
		return nil, err
	}

	baseURLs := e.baseURLFallbackOrder(creds)

	for idx, baseURL := range baseURLs {
		httpReq, err := e.buildRequest(ctx, creds, token, req.Model, req.Payload, true, baseURL)
		if err != nil {
			return nil, err
		}

		httpResp, err := e.httpClient.Do(httpReq)
		if err != nil {
			log.Debugf("Request error on %s: %v", baseURL, err)
			if idx+1 < len(baseURLs) {
				continue
			}
			return nil, err
		}

		if httpResp.StatusCode < 200 || httpResp.StatusCode >= 300 {
			bodyBytes, _ := io.ReadAll(httpResp.Body)
			httpResp.Body.Close()
			if httpResp.StatusCode == http.StatusTooManyRequests && idx+1 < len(baseURLs) {
				log.Debugf("Rate limited on %s, trying fallback", baseURL)
				continue
			}
			return nil, fmt.Errorf("API error: status %d: %s", httpResp.StatusCode, string(bodyBytes))
		}

		out := make(chan StreamChunk)
		go func() {
			defer close(out)
			defer httpResp.Body.Close()

			scanner := bufio.NewScanner(httpResp.Body)
			scanner.Buffer(nil, StreamScannerSize)

			for scanner.Scan() {
				line := scanner.Bytes()
				// Filter usage metadata for intermediate chunks
				line = FilterSSEUsageMetadata(line)

				payload := extractJSONPayload(line)
				if payload == nil {
					continue
				}

				out <- StreamChunk{Data: bytes.Clone(payload)}
			}

			if err := scanner.Err(); err != nil {
				out <- StreamChunk{Err: err}
			}
		}()

		return out, nil
	}

	return nil, fmt.Errorf("all base URLs exhausted")
}

func (e *Executor) ensureAccessToken(ctx context.Context, creds *auth.Credentials) (string, error) {
	if creds == nil {
		return "", fmt.Errorf("missing credentials")
	}

	if creds.AccessToken != "" && !creds.IsExpired() {
		return creds.AccessToken, nil
	}

	if e.tokenManager != nil {
		refreshed, err := e.tokenManager.EnsureValidToken(ctx, creds)
		if err != nil {
			return "", err
		}
		return refreshed.AccessToken, nil
	}

	return creds.AccessToken, nil
}

func (e *Executor) buildRequest(ctx context.Context, creds *auth.Credentials, token, modelName string, payload []byte, stream bool, baseURL string) (*http.Request, error) {
	if token == "" {
		return nil, fmt.Errorf("missing access token")
	}

	base := strings.TrimSuffix(baseURL, "/")
	path := GeneratePath
	if stream {
		path = StreamPath
	}

	var requestURL strings.Builder
	requestURL.WriteString(base)
	requestURL.WriteString(path)
	if stream {
		requestURL.WriteString("?alt=sse")
	}

	// Transform payload for Antigravity format
	payload = e.transformPayload(modelName, payload, creds.ProjectID)

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, requestURL.String(), bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+token)
	httpReq.Header.Set("User-Agent", e.resolveUserAgent(creds))

	if stream {
		httpReq.Header.Set("Accept", "text/event-stream")
	} else {
		httpReq.Header.Set("Accept", "application/json")
	}

	if host := resolveHost(base); host != "" {
		httpReq.Host = host
	}

	return httpReq, nil
}

func (e *Executor) transformPayload(modelName string, payload []byte, projectID string) []byte {
	// Convert alias to internal model name
	internalModelName := models.Alias2ModelName(modelName)

	template, _ := sjson.Set(string(payload), "model", internalModelName)
	template, _ = sjson.Set(template, "userAgent", "antigravity")

	if projectID != "" {
		template, _ = sjson.Set(template, "project", projectID)
	} else {
		template, _ = sjson.Set(template, "project", generateProjectID())
	}
	template, _ = sjson.Set(template, "requestId", generateRequestID())
	template, _ = sjson.Set(template, "request.sessionId", generateSessionID())

	template, _ = sjson.Delete(template, "request.safetySettings")
	template, _ = sjson.Set(template, "request.toolConfig.functionCallingConfig.mode", "VALIDATED")

	// Handle thinkingLevel for non-gemini-3 models
	if !strings.HasPrefix(modelName, "gemini-3-") {
		if thinkingLevel := gjson.Get(template, "request.generationConfig.thinkingConfig.thinkingLevel"); thinkingLevel.Exists() {
			template, _ = sjson.Delete(template, "request.generationConfig.thinkingConfig.thinkingLevel")
			template, _ = sjson.Set(template, "request.generationConfig.thinkingConfig.thinkingBudget", -1)
		}
	}

	// Handle Claude model specific transformations
	isClaude := strings.Contains(strings.ToLower(modelName), "claude")
	if isClaude {
		template = e.transformClaudePayload(template)
		// Normalize thinking budget for Claude models
		template = e.normalizeClaudeThinking(modelName, template)
	} else {
		// For non-Claude models, delete maxOutputTokens
		template, _ = sjson.Delete(template, "request.generationConfig.maxOutputTokens")
	}

	return []byte(template)
}

// normalizeClaudeThinking ensures thinking budget is properly clamped for Claude models.
// The budget must be less than maxOutputTokens and within model-specific limits.
func (e *Executor) normalizeClaudeThinking(modelName string, template string) string {
	if !models.ModelSupportsThinking(modelName) {
		return template
	}

	budget := gjson.Get(template, "request.generationConfig.thinkingConfig.thinkingBudget")
	if !budget.Exists() {
		return template
	}

	raw := int(budget.Int())
	normalized := models.NormalizeThinkingBudget(modelName, raw)

	// Get effective max tokens
	effectiveMax := 0
	setDefaultMax := false
	if maxTok := gjson.Get(template, "request.generationConfig.maxOutputTokens"); maxTok.Exists() && maxTok.Int() > 0 {
		effectiveMax = int(maxTok.Int())
	} else {
		// Use model default
		cfg := models.GetModelConfig(modelName)
		if cfg != nil && cfg.MaxCompletionTokens > 0 {
			effectiveMax = cfg.MaxCompletionTokens
			setDefaultMax = true
		}
	}

	// Ensure budget < maxOutputTokens
	if effectiveMax > 0 && normalized >= effectiveMax {
		normalized = effectiveMax - 1
	}

	// Check minimum budget
	cfg := models.GetModelConfig(modelName)
	if cfg != nil && cfg.Thinking != nil {
		minBudget := cfg.Thinking.Min
		if minBudget > 0 && normalized >= 0 && normalized < minBudget {
			// Budget is below minimum, remove thinking config entirely
			template, _ = sjson.Delete(template, "request.generationConfig.thinkingConfig")
			return template
		}
	}

	// Set default max tokens if needed
	if setDefaultMax && effectiveMax > 0 {
		template, _ = sjson.Set(template, "request.generationConfig.maxOutputTokens", effectiveMax)
	}

	// Set normalized budget
	template, _ = sjson.Set(template, "request.generationConfig.thinkingConfig.thinkingBudget", normalized)

	return template
}

func (e *Executor) transformClaudePayload(template string) string {
	// Handle parametersJsonSchema -> parameters rename
	paths := findJSONPaths(gjson.Parse(template), "", "parametersJsonSchema")
	for _, p := range paths {
		newPath := p[:len(p)-len("parametersJsonSchema")] + "parameters"
		template = renameJSONKey(template, p, newPath)
	}

	// Remove unsupported schema fields
	template = deleteJSONKey(template, "$schema")
	template = deleteJSONKey(template, "maxItems")
	template = deleteJSONKey(template, "minItems")
	template = deleteJSONKey(template, "minLength")
	template = deleteJSONKey(template, "maxLength")
	template = deleteJSONKey(template, "exclusiveMinimum")
	template = deleteJSONKey(template, "exclusiveMaximum")
	template = deleteJSONKey(template, "$ref")
	template = deleteJSONKey(template, "$defs")

	// Handle anyOf -> first item
	anyOfPaths := findJSONPaths(gjson.Parse(template), "", "anyOf")
	for _, p := range anyOfPaths {
		anyOf := gjson.Get(template, p)
		if anyOf.IsArray() {
			items := anyOf.Array()
			if len(items) > 0 {
				template, _ = sjson.SetRaw(template, p[:len(p)-len(".anyOf")], items[0].Raw)
			}
		}
	}

	return template
}

func (e *Executor) resolveUserAgent(creds *auth.Credentials) string {
	if creds != nil && creds.UserAgent != "" {
		return creds.UserAgent
	}
	return DefaultUserAgent
}

func (e *Executor) baseURLFallbackOrder(creds *auth.Credentials) []string {
	if creds != nil && creds.BaseURL != "" {
		return []string{strings.TrimSuffix(creds.BaseURL, "/")}
	}
	return []string{BaseURLDaily, BaseURLProd}
}

// Helper functions

func generateRequestID() string {
	return "agent-" + uuid.NewString()
}

func generateSessionID() string {
	n := randSource.Int63n(9_000_000_000_000_000_000)
	return "-" + strconv.FormatInt(n, 10)
}

func generateProjectID() string {
	adjectives := []string{"useful", "bright", "swift", "calm", "bold"}
	nouns := []string{"fuze", "wave", "spark", "flow", "core"}
	adj := adjectives[randSource.Intn(len(adjectives))]
	noun := nouns[randSource.Intn(len(nouns))]
	randomPart := strings.ToLower(uuid.NewString())[:5]
	return adj + "-" + noun + "-" + randomPart
}

func extractJSONPayload(line []byte) []byte {
	trimmed := bytes.TrimSpace(line)
	if len(trimmed) == 0 {
		return nil
	}

	// Handle SSE data: prefix
	if bytes.HasPrefix(trimmed, []byte("data:")) {
		trimmed = bytes.TrimSpace(trimmed[5:])
	}

	if len(trimmed) == 0 || !gjson.ValidBytes(trimmed) {
		return nil
	}

	return trimmed
}

func findJSONPaths(result gjson.Result, prefix, targetKey string) []string {
	var paths []string
	if result.IsObject() {
		result.ForEach(func(key, value gjson.Result) bool {
			currentPath := key.String()
			if prefix != "" {
				currentPath = prefix + "." + key.String()
			}
			if key.String() == targetKey {
				paths = append(paths, currentPath)
			}
			paths = append(paths, findJSONPaths(value, currentPath, targetKey)...)
			return true
		})
	} else if result.IsArray() {
		for i, item := range result.Array() {
			currentPath := fmt.Sprintf("%s.%d", prefix, i)
			paths = append(paths, findJSONPaths(item, currentPath, targetKey)...)
		}
	}
	return paths
}

func renameJSONKey(jsonStr, oldPath, newPath string) string {
	value := gjson.Get(jsonStr, oldPath)
	if !value.Exists() {
		return jsonStr
	}
	jsonStr, _ = sjson.SetRaw(jsonStr, newPath, value.Raw)
	jsonStr, _ = sjson.Delete(jsonStr, oldPath)
	return jsonStr
}

func deleteJSONKey(jsonStr, key string) string {
	paths := findJSONPaths(gjson.Parse(jsonStr), "", key)
	for _, p := range paths {
		jsonStr, _ = sjson.Delete(jsonStr, p)
	}
	return jsonStr
}

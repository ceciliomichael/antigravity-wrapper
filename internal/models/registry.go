package models

import (
	"strings"
	"sync"
	"time"
)

// ModelInfo represents information about an available model.
type ModelInfo struct {
	ID                  string           `json:"id"`
	Object              string           `json:"object"`
	Created             int64            `json:"created"`
	OwnedBy             string           `json:"owned_by"`
	Type                string           `json:"type"`
	DisplayName         string           `json:"display_name,omitempty"`
	Name                string           `json:"name,omitempty"`
	Version             string           `json:"version,omitempty"`
	Description         string           `json:"description,omitempty"`
	MaxCompletionTokens int              `json:"max_completion_tokens,omitempty"`
	Thinking            *ThinkingSupport `json:"thinking,omitempty"`
}

// ModelConfig holds static configuration for antigravity models.
type ModelConfig struct {
	Thinking            *ThinkingSupport
	MaxCompletionTokens int
	Name                string
}

// modelConfigs contains static model configurations keyed by model names.
var modelConfigs = map[string]*ModelConfig{
	"gemini-2.5-flash": {
		Thinking: &ThinkingSupport{Min: 0, Max: 24576, ZeroAllowed: true, DynamicAllowed: true},
		Name:     "models/gemini-2.5-flash",
	},
	"gemini-3-flash": {
		Thinking: &ThinkingSupport{Min: 128, Max: 32768, ZeroAllowed: false, DynamicAllowed: true},
		Name:     "models/gemini-3-flash",
	},
	"gemini-2.5-flash-lite": {
		Thinking: &ThinkingSupport{Min: 0, Max: 24576, ZeroAllowed: true, DynamicAllowed: true},
		Name:     "models/gemini-2.5-flash-lite",
	},
	"gemini-3-pro-high": {
		Thinking: &ThinkingSupport{Min: 128, Max: 32768, ZeroAllowed: false, DynamicAllowed: true},
		Name:     "models/gemini-3-pro-high",
	},
	"gemini-3-pro-low": {
		Thinking: &ThinkingSupport{Min: 128, Max: 32768, ZeroAllowed: false, DynamicAllowed: true},
		Name:     "models/gemini-3-pro-low",
	},
	"claude-sonnet-4-5": {
		Name:                "claude-sonnet-4-5",
		MaxCompletionTokens: 64000,
	},
	"claude-sonnet-4-5-thinking": {
		Thinking:            &ThinkingSupport{Min: 1024, Max: 200000, ZeroAllowed: false, DynamicAllowed: true},
		MaxCompletionTokens: 64000,
	},
	"claude-opus-4-5-thinking": {
		Thinking:            &ThinkingSupport{Min: 1024, Max: 200000, ZeroAllowed: false, DynamicAllowed: true},
		Name:                "claude-opus-4-5-thinking",
		MaxCompletionTokens: 64000,
	},
	"claude-opus-4-5": {
		Name:                "claude-opus-4-5-thinking",
		MaxCompletionTokens: 64000,
	},
}

// GetModelConfig returns the static configuration for a model.
func GetModelConfig(model string) *ModelConfig {
	return modelConfigs[model]
}

// ModelName2Alias converts internal model names to user-facing names.
// Now we use the actual model names, so this is mostly a pass-through.
func ModelName2Alias(modelName string) string {
	// Filter out internal-only models that shouldn't be exposed
	switch modelName {
	case "chat_20706", "chat_23310", "gemini-2.5-flash-thinking", "gemini-2.5-pro", "gemini-3-pro-image", "rev19-uic3-1p":
		return ""
	default:
		return modelName
	}
}

// Alias2ModelName converts user-facing names to internal model names.
// It only performs aliasing when the Name field points to a different model ID
// (not a display name like "models/gemini-2.5-flash").
func Alias2ModelName(modelName string) string {
	cfg := GetModelConfig(modelName)
	if cfg != nil && cfg.Name != "" {
		// Only alias if Name is a different model ID (not a display path)
		// Display paths start with "models/" or match the input
		if cfg.Name != modelName && !strings.HasPrefix(cfg.Name, "models/") {
			return cfg.Name
		}
	}
	return modelName
}

// Registry manages available models.
type Registry struct {
	models map[string]*ModelInfo
	mu     sync.RWMutex
}

// NewRegistry creates a new model registry with default models.
func NewRegistry() *Registry {
	r := &Registry{
		models: make(map[string]*ModelInfo),
	}
	r.loadDefaultModels()
	return r
}

// loadDefaultModels populates the registry with known models.
func (r *Registry) loadDefaultModels() {
	now := time.Now().Unix()
	defaultModels := []*ModelInfo{
		{
			ID:          "gemini-2.5-flash",
			Object:      "model",
			Created:     now,
			OwnedBy:     "antigravity",
			Type:        "antigravity",
			DisplayName: "Gemini 2.5 Flash",
			Name:        "models/gemini-2.5-flash",
			Thinking:    &ThinkingSupport{Min: 0, Max: 24576, ZeroAllowed: true, DynamicAllowed: true},
		},
		{
			ID:          "gemini-3-flash",
			Object:      "model",
			Created:     now,
			OwnedBy:     "antigravity",
			Type:        "antigravity",
			DisplayName: "Gemini 3 Flash",
			Name:        "models/gemini-3-flash",
			Thinking:    &ThinkingSupport{Min: 128, Max: 32768, ZeroAllowed: false, DynamicAllowed: true},
		},
		{
			ID:          "gemini-2.5-flash-lite",
			Object:      "model",
			Created:     now,
			OwnedBy:     "antigravity",
			Type:        "antigravity",
			DisplayName: "Gemini 2.5 Flash Lite",
			Name:        "models/gemini-2.5-flash-lite",
			Thinking:    &ThinkingSupport{Min: 0, Max: 24576, ZeroAllowed: true, DynamicAllowed: true},
		},
		{
			ID:          "gemini-3-pro-high",
			Object:      "model",
			Created:     now,
			OwnedBy:     "antigravity",
			Type:        "antigravity",
			DisplayName: "Gemini 3 Pro High",
			Name:        "models/gemini-3-pro-high",
			Thinking:    &ThinkingSupport{Min: 128, Max: 32768, ZeroAllowed: false, DynamicAllowed: true},
		},
		{
			ID:          "gemini-3-pro-low",
			Object:      "model",
			Created:     now,
			OwnedBy:     "antigravity",
			Type:        "antigravity",
			DisplayName: "Gemini 3 Pro Low",
			Name:        "models/gemini-3-pro-low",
			Thinking:    &ThinkingSupport{Min: 128, Max: 8192, ZeroAllowed: false, DynamicAllowed: true},
		},
		{
			ID:                  "claude-sonnet-4-5",
			Object:              "model",
			Created:             now,
			OwnedBy:             "antigravity",
			Type:                "antigravity",
			DisplayName:         "Claude Sonnet 4.5",
			MaxCompletionTokens: 64000,
		},
		{
			ID:                  "claude-sonnet-4-5-thinking",
			Object:              "model",
			Created:             now,
			OwnedBy:             "antigravity",
			Type:                "antigravity",
			DisplayName:         "Claude Sonnet 4.5 (Thinking)",
			MaxCompletionTokens: 64000,
			Thinking:            &ThinkingSupport{Min: 1024, Max: 200000, ZeroAllowed: false, DynamicAllowed: true},
		},
		{
			ID:                  "claude-opus-4-5-thinking",
			Object:              "model",
			Created:             now,
			OwnedBy:             "antigravity",
			Type:                "antigravity",
			DisplayName:         "Claude Opus 4.5 (Thinking)",
			MaxCompletionTokens: 64000,
			Thinking:            &ThinkingSupport{Min: 1024, Max: 200000, ZeroAllowed: false, DynamicAllowed: true},
		},
		{
			ID:                  "claude-opus-4-5",
			Object:              "model",
			Created:             now,
			OwnedBy:             "antigravity",
			Type:                "antigravity",
			DisplayName:         "Claude Opus 4.5",
			MaxCompletionTokens: 64000,
		},
	}

	for _, m := range defaultModels {
		r.models[m.ID] = m
	}
}

// GetModel returns a model by ID.
func (r *Registry) GetModel(id string) *ModelInfo {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.models[id]
}

// ListModels returns all available models.
func (r *Registry) ListModels() []*ModelInfo {
	r.mu.RLock()
	defer r.mu.RUnlock()

	models := make([]*ModelInfo, 0, len(r.models))
	for _, m := range r.models {
		models = append(models, m)
	}
	return models
}

// UpdateModels replaces the model list with fetched models.
func (r *Registry) UpdateModels(models []*ModelInfo) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.models = make(map[string]*ModelInfo)
	for _, m := range models {
		r.models[m.ID] = m
	}
}

// AddModel adds or updates a model in the registry.
func (r *Registry) AddModel(m *ModelInfo) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.models[m.ID] = m
}

// global registry instance
var globalRegistry *Registry
var registryOnce sync.Once

// GetGlobalRegistry returns the global model registry.
func GetGlobalRegistry() *Registry {
	registryOnce.Do(func() {
		globalRegistry = NewRegistry()
	})
	return globalRegistry
}

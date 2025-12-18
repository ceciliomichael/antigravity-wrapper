package models

import "time"

// modelConfigs contains static model configurations keyed by model names.
var modelConfigs = map[string]*ModelConfig{
	"gemini-2.5-flash": {
		Name: "models/gemini-2.5-flash",
	},
	"gemini-2.5-flash-thinking": {
		Thinking: &ThinkingSupport{Min: 0, Max: 24576, ZeroAllowed: true, DynamicAllowed: true},
		Name:     "gemini-2.5-flash",
	},
	"gemini-3-flash-minimal": {
		Thinking: &ThinkingSupport{Min: 128, Max: 512, ZeroAllowed: false, DynamicAllowed: true},
		Name:     "models/gemini-3-flash",
	},
	"gemini-3-flash-thinking": {
		Thinking: &ThinkingSupport{Min: 128, Max: 32768, ZeroAllowed: false, DynamicAllowed: true},
		Name:     "gemini-3-flash",
	},
	"gemini-2.5-flash-lite": {
		Name: "models/gemini-2.5-flash-lite",
	},
	"gemini-2.5-flash-lite-thinking": {
		Thinking: &ThinkingSupport{Min: 0, Max: 24576, ZeroAllowed: true, DynamicAllowed: true},
		Name:     "gemini-2.5-flash-lite",
	},
	"gemini-3-pro-high": {
		Thinking: &ThinkingSupport{Min: 128, Max: 32768, ZeroAllowed: false, DynamicAllowed: true},
		Name:     "models/gemini-3-pro-high",
	},
	"gemini-3-pro-low": {
		Thinking: &ThinkingSupport{Min: 128, Max: 1024, ZeroAllowed: false, DynamicAllowed: true},
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
		Name:                "claude-opus-4-5",
		MaxCompletionTokens: 64000,
	},
	"claude-opus-4-5": {
		Name:                "claude-opus-4-5-thinking",
		MaxCompletionTokens: 64000,
	},
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
		},
		{
			ID:          "gemini-2.5-flash-thinking",
			Object:      "model",
			Created:     now,
			OwnedBy:     "antigravity",
			Type:        "antigravity",
			DisplayName: "Gemini 2.5 Flash (Thinking)",
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
			ID:          "gemini-3-flash-thinking",
			Object:      "model",
			Created:     now,
			OwnedBy:     "antigravity",
			Type:        "antigravity",
			DisplayName: "Gemini 3 Flash (Thinking)",
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
		},
		{
			ID:          "gemini-2.5-flash-lite-thinking",
			Object:      "model",
			Created:     now,
			OwnedBy:     "antigravity",
			Type:        "antigravity",
			DisplayName: "Gemini 2.5 Flash Lite (Thinking)",
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

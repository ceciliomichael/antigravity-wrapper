// Package models provides model registry and thinking budget utilities.
package models

import (
	"strings"

	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
)

// ThinkingSupport describes a model's supported internal reasoning budget range.
type ThinkingSupport struct {
	Min            int  `json:"min,omitempty"`
	Max            int  `json:"max,omitempty"`
	ZeroAllowed    bool `json:"zero_allowed,omitempty"`
	DynamicAllowed bool `json:"dynamic_allowed,omitempty"`
}

// ModelSupportsThinking reports whether the given model has Thinking capability.
func ModelSupportsThinking(model string) bool {
	if model == "" {
		return false
	}
	info := GetModelConfig(model)
	return info != nil && info.Thinking != nil
}

// ModelUsesThinkingLevels reports whether the model uses discrete reasoning
// effort levels instead of numeric budgets.
func ModelUsesThinkingLevels(model string) bool {
	// Antigravity models use numeric budgets, not levels
	return false
}

// NormalizeThinkingBudget clamps the requested thinking budget to the
// supported range for the specified model.
func NormalizeThinkingBudget(model string, budget int) int {
	info := GetModelConfig(model)
	if info == nil || info.Thinking == nil {
		return budget
	}

	t := info.Thinking

	// Handle dynamic budget (-1)
	if budget == -1 {
		if t.DynamicAllowed {
			return -1
		}
		// Approximate mid-range
		mid := (t.Min + t.Max) / 2
		if mid <= 0 && t.ZeroAllowed {
			return 0
		}
		if mid <= 0 {
			return t.Min
		}
		return mid
	}

	// Handle zero budget
	if budget == 0 {
		if t.ZeroAllowed {
			return 0
		}
		return t.Min
	}

	// Clamp to range
	if budget < t.Min {
		return t.Min
	}
	if budget > t.Max {
		return t.Max
	}
	return budget
}

// ReasoningEffortBudgetMapping defines the thinkingBudget values for each reasoning effort level.
var ReasoningEffortBudgetMapping = map[string]int{
	"none":    0,
	"auto":    -1,
	"minimal": 512,
	"low":     1024,
	"medium":  8192,
	"high":    24576,
	"xhigh":   32768,
}

// ThinkingEffortToBudget maps a reasoning effort level to a numeric thinking budget.
func ThinkingEffortToBudget(model, effort string) (int, bool) {
	if effort == "" {
		return 0, false
	}
	normalized := strings.ToLower(strings.TrimSpace(effort))
	budget, ok := ReasoningEffortBudgetMapping[normalized]
	if !ok {
		return 0, false
	}
	if normalized == "none" {
		return 0, true
	}
	return NormalizeThinkingBudget(model, budget), true
}

// ApplyReasoningEffortToPayload applies OpenAI reasoning_effort to the Antigravity payload.
func ApplyReasoningEffortToPayload(modelName string, payload []byte, effort string) []byte {
	normalized := strings.ToLower(strings.TrimSpace(effort))
	if normalized == "" {
		return payload
	}

	budgetPath := "request.generationConfig.thinkingConfig.thinkingBudget"
	includePath := "request.generationConfig.thinkingConfig.include_thoughts"

	if normalized == "none" {
		info := GetModelConfig(modelName)
		// Only disable thinking if the model explicitly allows zero budget
		if info != nil && info.Thinking != nil && info.Thinking.ZeroAllowed {
			payload, _ = sjson.SetBytes(payload, budgetPath, 0)
			payload, _ = sjson.SetBytes(payload, includePath, false)
			return payload
		}
		// Otherwise, do nothing and let the default injection logic take over
		return payload
	}

	budget, ok := ReasoningEffortBudgetMapping[normalized]
	if !ok {
		return payload
	}

	payload, _ = sjson.SetBytes(payload, budgetPath, budget)
	payload, _ = sjson.SetBytes(payload, includePath, true)
	return payload
}

// StripThinkingConfigIfUnsupported removes thinkingConfig when the model doesn't support it
// or if the budget is set to 0 (explicitly disabled).
func StripThinkingConfigIfUnsupported(model string, payload []byte) []byte {
	if len(payload) == 0 {
		return payload
	}

	// Remove if model doesn't support it
	if !ModelSupportsThinking(model) {
		payload, _ = sjson.DeleteBytes(payload, "request.generationConfig.thinkingConfig")
		payload, _ = sjson.DeleteBytes(payload, "generationConfig.thinkingConfig")
		return payload
	}

	// Remove if budget is explicitly 0 (none)
	budget := gjson.GetBytes(payload, "request.generationConfig.thinkingConfig.thinkingBudget")
	if budget.Exists() && budget.Int() <= 0 {
		payload, _ = sjson.DeleteBytes(payload, "request.generationConfig.thinkingConfig")
	}

	return payload
}

// DefaultThinkingBudget is the default reasoning budget for thinking models ("high" level).
const DefaultThinkingBudget = 24576

// ApplyDefaultThinkingIfNeeded injects default thinkingConfig for models that support thinking.
// Uses "high" reasoning effort (24576 tokens) by default, except for gemini-3-flash which uses "minimal".
func ApplyDefaultThinkingIfNeeded(model string, payload []byte) []byte {
	if !ModelHasDefaultThinking(model) {
		return payload
	}

	// We enforce default thinking config, ignoring client overrides
	budget := DefaultThinkingBudget
	if model == "gemini-3-flash" {
		budget = ReasoningEffortBudgetMapping["minimal"]
	}

	// Clamp the budget to the model's supported range (e.g. gemini-3-flash-minimal max 512)
	budget = NormalizeThinkingBudget(model, budget)

	// Ensure budget is less than maxOutputTokens if set
	maxTokens := gjson.GetBytes(payload, "request.generationConfig.maxOutputTokens")
	if maxTokens.Exists() && maxTokens.Int() > 0 {
		mt := int(maxTokens.Int())
		if budget >= mt {
			// Reserve some tokens for response or just cap it below max
			// Using 80% of max tokens or max tokens - 1024 as heuristic,
			// but simply ensuring it's strictly less is the hard requirement.
			// Let's safe-guard it to be at most maxTokens.
			// However, usually we need room for the actual response.
			// Let's cap at maxTokens - 1 (strict requirement is budget < max)
			// A safer bet is likely min(budget, maxTokens) but strictly less.
			newBudget := mt - 1
			if newBudget < 0 {
				newBudget = 0
			}
			// If the clamped budget is below the model's minimum, we might have a problem,
			// but we must respect the max_tokens constraint first to avoid API error.
			if newBudget < budget {
				budget = newBudget
			}
		}
	}

	payload, _ = sjson.SetBytes(payload, "request.generationConfig.thinkingConfig.thinkingBudget", budget)
	payload, _ = sjson.SetBytes(payload, "request.generationConfig.thinkingConfig.include_thoughts", true)
	return payload
}

// ModelHasDefaultThinking returns true if the model should have thinking enabled by default.
// This applies to all models that support thinking.
func ModelHasDefaultThinking(model string) bool {
	return ModelSupportsThinking(model)
}

// ApplyThinkingConfig applies thinking configuration to the payload.
func ApplyThinkingConfig(payload []byte, budget *int, includeThoughts *bool) []byte {
	if budget == nil && includeThoughts == nil {
		return payload
	}

	if budget != nil {
		payload, _ = sjson.SetBytes(payload, "request.generationConfig.thinkingConfig.thinkingBudget", *budget)
	}

	incl := includeThoughts
	if incl == nil && budget != nil && *budget != 0 {
		defaultInclude := true
		incl = &defaultInclude
	}
	if incl != nil {
		payload, _ = sjson.SetBytes(payload, "request.generationConfig.thinkingConfig.include_thoughts", *incl)
	}
	return payload
}

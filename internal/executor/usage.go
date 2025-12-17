package executor

import (
	"bytes"

	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
)

// UsageDetail holds token usage information from API responses.
type UsageDetail struct {
	InputTokens     int64
	OutputTokens    int64
	ReasoningTokens int64
	CachedTokens    int64
	TotalTokens     int64
}

// ParseUsage extracts usage information from an Antigravity response.
func ParseUsage(data []byte) UsageDetail {
	root := gjson.ParseBytes(data)
	node := root.Get("response.usageMetadata")
	if !node.Exists() {
		node = root.Get("usageMetadata")
	}
	if !node.Exists() {
		node = root.Get("usage_metadata")
	}
	if !node.Exists() {
		return UsageDetail{}
	}
	detail := UsageDetail{
		InputTokens:     node.Get("promptTokenCount").Int(),
		OutputTokens:    node.Get("candidatesTokenCount").Int(),
		ReasoningTokens: node.Get("thoughtsTokenCount").Int(),
		TotalTokens:     node.Get("totalTokenCount").Int(),
	}
	if detail.TotalTokens == 0 {
		detail.TotalTokens = detail.InputTokens + detail.OutputTokens + detail.ReasoningTokens
	}
	return detail
}

// ParseStreamUsage extracts usage information from a streaming chunk.
func ParseStreamUsage(line []byte) (UsageDetail, bool) {
	payload := extractJSONPayload(line)
	if len(payload) == 0 || !gjson.ValidBytes(payload) {
		return UsageDetail{}, false
	}
	node := gjson.GetBytes(payload, "response.usageMetadata")
	if !node.Exists() {
		node = gjson.GetBytes(payload, "usageMetadata")
	}
	if !node.Exists() {
		node = gjson.GetBytes(payload, "usage_metadata")
	}
	if !node.Exists() {
		return UsageDetail{}, false
	}
	detail := UsageDetail{
		InputTokens:     node.Get("promptTokenCount").Int(),
		OutputTokens:    node.Get("candidatesTokenCount").Int(),
		ReasoningTokens: node.Get("thoughtsTokenCount").Int(),
		TotalTokens:     node.Get("totalTokenCount").Int(),
	}
	if detail.TotalTokens == 0 {
		detail.TotalTokens = detail.InputTokens + detail.OutputTokens + detail.ReasoningTokens
	}
	return detail, true
}

// FilterSSEUsageMetadata removes usageMetadata from SSE events that are not
// terminal (finishReason present). Stop chunks are left untouched.
func FilterSSEUsageMetadata(payload []byte) []byte {
	if len(payload) == 0 {
		return payload
	}

	lines := bytes.Split(payload, []byte("\n"))
	modified := false
	foundData := false

	for idx, line := range lines {
		trimmed := bytes.TrimSpace(line)
		if len(trimmed) == 0 || !bytes.HasPrefix(trimmed, []byte("data:")) {
			continue
		}
		foundData = true
		dataIdx := bytes.Index(line, []byte("data:"))
		if dataIdx < 0 {
			continue
		}
		rawJSON := bytes.TrimSpace(line[dataIdx+5:])

		cleaned, changed := StripUsageMetadataFromJSON(rawJSON)
		if !changed {
			continue
		}
		var rebuilt []byte
		rebuilt = append(rebuilt, line[:dataIdx]...)
		rebuilt = append(rebuilt, []byte("data:")...)
		if len(cleaned) > 0 {
			rebuilt = append(rebuilt, ' ')
			rebuilt = append(rebuilt, cleaned...)
		}
		lines[idx] = rebuilt
		modified = true
	}

	if !modified {
		if !foundData {
			// Handle payloads that are raw JSON without SSE data: prefix.
			trimmed := bytes.TrimSpace(payload)
			cleaned, changed := StripUsageMetadataFromJSON(trimmed)
			if !changed {
				return payload
			}
			return cleaned
		}
		return payload
	}
	return bytes.Join(lines, []byte("\n"))
}

// StripUsageMetadataFromJSON drops usageMetadata unless finishReason is present.
func StripUsageMetadataFromJSON(rawJSON []byte) ([]byte, bool) {
	jsonBytes := bytes.TrimSpace(rawJSON)
	if len(jsonBytes) == 0 || !gjson.ValidBytes(jsonBytes) {
		return rawJSON, false
	}

	// Check for finishReason in both aistudio and antigravity formats
	finishReason := gjson.GetBytes(jsonBytes, "candidates.0.finishReason")
	if !finishReason.Exists() {
		finishReason = gjson.GetBytes(jsonBytes, "response.candidates.0.finishReason")
	}

	// If finishReason is present, keep usage metadata
	if finishReason.Exists() && finishReason.String() != "" {
		return rawJSON, false
	}

	// Check if usage metadata exists
	hasUsage := gjson.GetBytes(jsonBytes, "usageMetadata").Exists() ||
		gjson.GetBytes(jsonBytes, "response.usageMetadata").Exists() ||
		gjson.GetBytes(jsonBytes, "usage_metadata").Exists()

	if !hasUsage {
		return rawJSON, false
	}

	// Strip usage metadata for non-terminal chunks
	result := string(jsonBytes)
	result, _ = sjson.Delete(result, "usageMetadata")
	result, _ = sjson.Delete(result, "response.usageMetadata")
	result, _ = sjson.Delete(result, "usage_metadata")

	return []byte(result), true
}
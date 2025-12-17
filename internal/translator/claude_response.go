package translator

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
	"sync/atomic"
	"time"

	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
)

// ClaudeStreamState holds state for Claude streaming response conversion.
type ClaudeStreamState struct {
	HasFirstResponse     bool
	ResponseType         int // 0=none, 1=content, 2=thinking, 3=function
	ResponseIndex        int
	HasFinishReason      bool
	FinishReason         string
	HasUsageMetadata     bool
	PromptTokenCount     int64
	CandidatesTokenCount int64
	ThoughtsTokenCount   int64
	TotalTokenCount      int64
	HasSentFinalEvents   bool
	HasToolUse           bool
	HasContent           bool
}

var claudeToolUseIDCounter uint64

// ConvertAntigravityResponseToClaude converts streaming Antigravity responses to Claude SSE format.
func ConvertAntigravityResponseToClaude(modelName string, rawJSON []byte, state *ClaudeStreamState) []string {
	if state == nil {
		state = &ClaudeStreamState{}
	}

	if bytes.Equal(rawJSON, []byte("[DONE]")) {
		output := ""
		if state.HasContent {
			appendClaudeFinalEvents(state, &output, true)
			return []string{output + "event: message_stop\ndata: {\"type\":\"message_stop\"}\n\n\n"}
		}
		return []string{}
	}

	output := ""

	// Initialize streaming session with message_start
	if !state.HasFirstResponse {
		output = "event: message_start\n"
		messageStartTemplate := `{"type": "message_start", "message": {"id": "msg_1nZdL29xx5MUA1yADyHTEsnR8uuvGzszyY", "type": "message", "role": "assistant", "content": [], "model": "claude-3-5-sonnet-20241022", "stop_reason": null, "stop_sequence": null, "usage": {"input_tokens": 0, "output_tokens": 0}}}`

		if modelVersionResult := gjson.GetBytes(rawJSON, "response.modelVersion"); modelVersionResult.Exists() {
			messageStartTemplate, _ = sjson.Set(messageStartTemplate, "message.model", modelVersionResult.String())
		}
		if responseIDResult := gjson.GetBytes(rawJSON, "response.responseId"); responseIDResult.Exists() {
			messageStartTemplate, _ = sjson.Set(messageStartTemplate, "message.id", responseIDResult.String())
		}
		output = output + fmt.Sprintf("data: %s\n\n\n", messageStartTemplate)
		state.HasFirstResponse = true
	}

	// Process response parts
	partsResult := gjson.GetBytes(rawJSON, "response.candidates.0.content.parts")
	if partsResult.IsArray() {
		partResults := partsResult.Array()
		for i := 0; i < len(partResults); i++ {
			partResult := partResults[i]
			partTextResult := partResult.Get("text")
			functionCallResult := partResult.Get("functionCall")

			if partTextResult.Exists() {
				// Handle thinking content
				if partResult.Get("thought").Bool() {
					if thoughtSignature := partResult.Get("thoughtSignature"); thoughtSignature.Exists() && thoughtSignature.String() != "" {
						output = output + "event: content_block_delta\n"
						data, _ := sjson.Set(fmt.Sprintf(`{"type":"content_block_delta","index":%d,"delta":{"type":"signature_delta","signature":""}}`, state.ResponseIndex), "delta.signature", thoughtSignature.String())
						output = output + fmt.Sprintf("data: %s\n\n\n", data)
						state.HasContent = true
					} else if state.ResponseType == 2 {
						output = output + "event: content_block_delta\n"
						data, _ := sjson.Set(fmt.Sprintf(`{"type":"content_block_delta","index":%d,"delta":{"type":"thinking_delta","thinking":""}}`, state.ResponseIndex), "delta.thinking", partTextResult.String())
						output = output + fmt.Sprintf("data: %s\n\n\n", data)
						state.HasContent = true
					} else {
						if state.ResponseType != 0 {
							output = output + "event: content_block_stop\n"
							output = output + fmt.Sprintf(`data: {"type":"content_block_stop","index":%d}`, state.ResponseIndex)
							output = output + "\n\n\n"
							state.ResponseIndex++
						}
						output = output + "event: content_block_start\n"
						output = output + fmt.Sprintf(`data: {"type":"content_block_start","index":%d,"content_block":{"type":"thinking","thinking":""}}`, state.ResponseIndex)
						output = output + "\n\n\n"
						output = output + "event: content_block_delta\n"
						data, _ := sjson.Set(fmt.Sprintf(`{"type":"content_block_delta","index":%d,"delta":{"type":"thinking_delta","thinking":""}}`, state.ResponseIndex), "delta.thinking", partTextResult.String())
						output = output + fmt.Sprintf("data: %s\n\n\n", data)
						state.ResponseType = 2
						state.HasContent = true
					}
				} else {
					finishReasonResult := gjson.GetBytes(rawJSON, "response.candidates.0.finishReason")
					if partTextResult.String() != "" || !finishReasonResult.Exists() {
						if state.ResponseType == 1 {
							output = output + "event: content_block_delta\n"
							data, _ := sjson.Set(fmt.Sprintf(`{"type":"content_block_delta","index":%d,"delta":{"type":"text_delta","text":""}}`, state.ResponseIndex), "delta.text", partTextResult.String())
							output = output + fmt.Sprintf("data: %s\n\n\n", data)
							state.HasContent = true
						} else {
							if state.ResponseType != 0 {
								output = output + "event: content_block_stop\n"
								output = output + fmt.Sprintf(`data: {"type":"content_block_stop","index":%d}`, state.ResponseIndex)
								output = output + "\n\n\n"
								state.ResponseIndex++
							}
							if partTextResult.String() != "" {
								output = output + "event: content_block_start\n"
								output = output + fmt.Sprintf(`data: {"type":"content_block_start","index":%d,"content_block":{"type":"text","text":""}}`, state.ResponseIndex)
								output = output + "\n\n\n"
								output = output + "event: content_block_delta\n"
								data, _ := sjson.Set(fmt.Sprintf(`{"type":"content_block_delta","index":%d,"delta":{"type":"text_delta","text":""}}`, state.ResponseIndex), "delta.text", partTextResult.String())
								output = output + fmt.Sprintf("data: %s\n\n\n", data)
								state.ResponseType = 1
								state.HasContent = true
							}
						}
					}
				}
			} else if functionCallResult.Exists() {
				state.HasToolUse = true
				fcName := functionCallResult.Get("name").String()

				if state.ResponseType == 3 {
					output = output + "event: content_block_stop\n"
					output = output + fmt.Sprintf(`data: {"type":"content_block_stop","index":%d}`, state.ResponseIndex)
					output = output + "\n\n\n"
					state.ResponseIndex++
					state.ResponseType = 0
				}

				if state.ResponseType != 0 {
					output = output + "event: content_block_stop\n"
					output = output + fmt.Sprintf(`data: {"type":"content_block_stop","index":%d}`, state.ResponseIndex)
					output = output + "\n\n\n"
					state.ResponseIndex++
				}

				output = output + "event: content_block_start\n"
				data := fmt.Sprintf(`{"type":"content_block_start","index":%d,"content_block":{"type":"tool_use","id":"","name":"","input":{}}}`, state.ResponseIndex)
				data, _ = sjson.Set(data, "content_block.id", fmt.Sprintf("%s-%d-%d", fcName, time.Now().UnixNano(), atomic.AddUint64(&claudeToolUseIDCounter, 1)))
				data, _ = sjson.Set(data, "content_block.name", fcName)
				output = output + fmt.Sprintf("data: %s\n\n\n", data)

				if fcArgsResult := functionCallResult.Get("args"); fcArgsResult.Exists() {
					output = output + "event: content_block_delta\n"
					data, _ = sjson.Set(fmt.Sprintf(`{"type":"content_block_delta","index":%d,"delta":{"type":"input_json_delta","partial_json":""}}`, state.ResponseIndex), "delta.partial_json", fcArgsResult.Raw)
					output = output + fmt.Sprintf("data: %s\n\n\n", data)
				}
				state.ResponseType = 3
				state.HasContent = true
			}
		}
	}

	if finishReasonResult := gjson.GetBytes(rawJSON, "response.candidates.0.finishReason"); finishReasonResult.Exists() {
		state.HasFinishReason = true
		state.FinishReason = finishReasonResult.String()
	}

	if usageResult := gjson.GetBytes(rawJSON, "response.usageMetadata"); usageResult.Exists() {
		state.HasUsageMetadata = true
		state.PromptTokenCount = usageResult.Get("promptTokenCount").Int()
		state.CandidatesTokenCount = usageResult.Get("candidatesTokenCount").Int()
		state.ThoughtsTokenCount = usageResult.Get("thoughtsTokenCount").Int()
		state.TotalTokenCount = usageResult.Get("totalTokenCount").Int()
		if state.CandidatesTokenCount == 0 && state.TotalTokenCount > 0 {
			state.CandidatesTokenCount = state.TotalTokenCount - state.PromptTokenCount - state.ThoughtsTokenCount
			if state.CandidatesTokenCount < 0 {
				state.CandidatesTokenCount = 0
			}
		}
	}

	if state.HasUsageMetadata && state.HasFinishReason {
		appendClaudeFinalEvents(state, &output, false)
	}

	return []string{output}
}

func appendClaudeFinalEvents(state *ClaudeStreamState, output *string, force bool) {
	if state.HasSentFinalEvents {
		return
	}
	if !state.HasUsageMetadata && !force {
		return
	}
	if !state.HasContent {
		return
	}

	if state.ResponseType != 0 {
		*output = *output + "event: content_block_stop\n"
		*output = *output + fmt.Sprintf(`data: {"type":"content_block_stop","index":%d}`, state.ResponseIndex)
		*output = *output + "\n\n\n"
		state.ResponseType = 0
	}

	stopReason := resolveClaudeStopReason(state)
	usageOutputTokens := state.CandidatesTokenCount + state.ThoughtsTokenCount
	if usageOutputTokens == 0 && state.TotalTokenCount > 0 {
		usageOutputTokens = state.TotalTokenCount - state.PromptTokenCount
		if usageOutputTokens < 0 {
			usageOutputTokens = 0
		}
	}

	*output = *output + "event: message_delta\n"
	*output = *output + "data: "
	delta := fmt.Sprintf(`{"type":"message_delta","delta":{"stop_reason":"%s","stop_sequence":null},"usage":{"input_tokens":%d,"output_tokens":%d}}`, stopReason, state.PromptTokenCount, usageOutputTokens)
	*output = *output + delta + "\n\n\n"

	state.HasSentFinalEvents = true
}

func resolveClaudeStopReason(state *ClaudeStreamState) string {
	if state.HasToolUse {
		return "tool_use"
	}
	switch state.FinishReason {
	case "MAX_TOKENS":
		return "max_tokens"
	case "STOP", "FINISH_REASON_UNSPECIFIED", "UNKNOWN":
		return "end_turn"
	}
	return "end_turn"
}

// ConvertAntigravityResponseToClaudeNonStream converts a non-streaming response to Claude format.
func ConvertAntigravityResponseToClaudeNonStream(modelName string, rawJSON []byte) string {
	root := gjson.ParseBytes(rawJSON)
	promptTokens := root.Get("response.usageMetadata.promptTokenCount").Int()
	candidateTokens := root.Get("response.usageMetadata.candidatesTokenCount").Int()
	thoughtTokens := root.Get("response.usageMetadata.thoughtsTokenCount").Int()
	totalTokens := root.Get("response.usageMetadata.totalTokenCount").Int()
	outputTokens := candidateTokens + thoughtTokens
	if outputTokens == 0 && totalTokens > 0 {
		outputTokens = totalTokens - promptTokens
		if outputTokens < 0 {
			outputTokens = 0
		}
	}

	response := map[string]interface{}{
		"id":            root.Get("response.responseId").String(),
		"type":          "message",
		"role":          "assistant",
		"model":         root.Get("response.modelVersion").String(),
		"content":       []interface{}{},
		"stop_reason":   nil,
		"stop_sequence": nil,
		"usage": map[string]interface{}{
			"input_tokens":  promptTokens,
			"output_tokens": outputTokens,
		},
	}

	parts := root.Get("response.candidates.0.content.parts")
	var contentBlocks []interface{}
	textBuilder := strings.Builder{}
	thinkingBuilder := strings.Builder{}
	toolIDCounter := 0
	hasToolCall := false

	flushText := func() {
		if textBuilder.Len() == 0 {
			return
		}
		contentBlocks = append(contentBlocks, map[string]interface{}{
			"type": "text",
			"text": textBuilder.String(),
		})
		textBuilder.Reset()
	}

	flushThinking := func() {
		if thinkingBuilder.Len() == 0 {
			return
		}
		contentBlocks = append(contentBlocks, map[string]interface{}{
			"type":     "thinking",
			"thinking": thinkingBuilder.String(),
		})
		thinkingBuilder.Reset()
	}

	if parts.IsArray() {
		for _, part := range parts.Array() {
			if text := part.Get("text"); text.Exists() && text.String() != "" {
				if part.Get("thought").Bool() {
					flushText()
					thinkingBuilder.WriteString(text.String())
					continue
				}
				flushThinking()
				textBuilder.WriteString(text.String())
				continue
			}

			if functionCall := part.Get("functionCall"); functionCall.Exists() {
				flushThinking()
				flushText()
				hasToolCall = true

				name := functionCall.Get("name").String()
				toolIDCounter++
				toolBlock := map[string]interface{}{
					"type":  "tool_use",
					"id":    fmt.Sprintf("tool_%d", toolIDCounter),
					"name":  name,
					"input": map[string]interface{}{},
				}

				if args := functionCall.Get("args"); args.Exists() {
					var parsed interface{}
					if err := json.Unmarshal([]byte(args.Raw), &parsed); err == nil {
						toolBlock["input"] = parsed
					}
				}

				contentBlocks = append(contentBlocks, toolBlock)
				continue
			}
		}
	}

	flushThinking()
	flushText()

	response["content"] = contentBlocks

	stopReason := "end_turn"
	if hasToolCall {
		stopReason = "tool_use"
	} else {
		if finish := root.Get("response.candidates.0.finishReason"); finish.Exists() {
			switch finish.String() {
			case "MAX_TOKENS":
				stopReason = "max_tokens"
			case "STOP", "FINISH_REASON_UNSPECIFIED", "UNKNOWN":
				stopReason = "end_turn"
			default:
				stopReason = "end_turn"
			}
		}
	}
	response["stop_reason"] = stopReason

	encoded, err := json.Marshal(response)
	if err != nil {
		return ""
	}
	return string(encoded)
}
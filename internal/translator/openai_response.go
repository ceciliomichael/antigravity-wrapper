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

// OpenAIStreamState holds state for streaming response conversion.
type OpenAIStreamState struct {
	UnixTimestamp int64
	FunctionIndex int
}

var functionCallIDCounter uint64

// ConvertAntigravityResponseToOpenAI converts a streaming Antigravity response to OpenAI format.
func ConvertAntigravityResponseToOpenAI(modelName string, rawJSON []byte, state *OpenAIStreamState) []string {
	if state == nil {
		state = &OpenAIStreamState{}
	}

	if bytes.Equal(rawJSON, []byte("[DONE]")) {
		return []string{}
	}

	template := `{"id":"","object":"chat.completion.chunk","created":12345,"model":"model","choices":[{"index":0,"delta":{"role":null,"content":null,"reasoning_content":null,"tool_calls":null},"finish_reason":null,"native_finish_reason":null}]}`

	// Extract and set the model version
	if modelVersionResult := gjson.GetBytes(rawJSON, "response.modelVersion"); modelVersionResult.Exists() {
		template, _ = sjson.Set(template, "model", modelVersionResult.String())
	}

	// Extract and set the creation timestamp
	if createTimeResult := gjson.GetBytes(rawJSON, "response.createTime"); createTimeResult.Exists() {
		t, err := time.Parse(time.RFC3339Nano, createTimeResult.String())
		if err == nil {
			state.UnixTimestamp = t.Unix()
		}
		template, _ = sjson.Set(template, "created", state.UnixTimestamp)
	} else {
		template, _ = sjson.Set(template, "created", state.UnixTimestamp)
	}

	// Extract and set the response ID
	if responseIDResult := gjson.GetBytes(rawJSON, "response.responseId"); responseIDResult.Exists() {
		template, _ = sjson.Set(template, "id", responseIDResult.String())
	}

	// Extract and set the finish reason
	if finishReasonResult := gjson.GetBytes(rawJSON, "response.candidates.0.finishReason"); finishReasonResult.Exists() {
		template, _ = sjson.Set(template, "choices.0.finish_reason", strings.ToLower(finishReasonResult.String()))
		template, _ = sjson.Set(template, "choices.0.native_finish_reason", strings.ToLower(finishReasonResult.String()))
	}

	// Extract and set usage metadata
	if usageResult := gjson.GetBytes(rawJSON, "response.usageMetadata"); usageResult.Exists() {
		if candidatesTokenCountResult := usageResult.Get("candidatesTokenCount"); candidatesTokenCountResult.Exists() {
			template, _ = sjson.Set(template, "usage.completion_tokens", candidatesTokenCountResult.Int())
		}
		if totalTokenCountResult := usageResult.Get("totalTokenCount"); totalTokenCountResult.Exists() {
			template, _ = sjson.Set(template, "usage.total_tokens", totalTokenCountResult.Int())
		}
		promptTokenCount := usageResult.Get("promptTokenCount").Int()
		thoughtsTokenCount := usageResult.Get("thoughtsTokenCount").Int()
		template, _ = sjson.Set(template, "usage.prompt_tokens", promptTokenCount+thoughtsTokenCount)
		if thoughtsTokenCount > 0 {
			template, _ = sjson.Set(template, "usage.completion_tokens_details.reasoning_tokens", thoughtsTokenCount)
		}
	}

	// Process the main content parts
	partsResult := gjson.GetBytes(rawJSON, "response.candidates.0.content.parts")
	hasFunctionCall := false
	if partsResult.IsArray() {
		partResults := partsResult.Array()
		for i := 0; i < len(partResults); i++ {
			partResult := partResults[i]
			partTextResult := partResult.Get("text")
			functionCallResult := partResult.Get("functionCall")
			thoughtSignatureResult := partResult.Get("thoughtSignature")
			if !thoughtSignatureResult.Exists() {
				thoughtSignatureResult = partResult.Get("thought_signature")
			}
			inlineDataResult := partResult.Get("inlineData")
			if !inlineDataResult.Exists() {
				inlineDataResult = partResult.Get("inline_data")
			}

			hasThoughtSignature := thoughtSignatureResult.Exists() && thoughtSignatureResult.String() != ""
			hasContentPayload := partTextResult.Exists() || functionCallResult.Exists() || inlineDataResult.Exists()

			// Ignore encrypted thoughtSignature but keep any actual content
			if hasThoughtSignature && !hasContentPayload {
				continue
			}

			if partTextResult.Exists() {
				textContent := partTextResult.String()

				// Handle reasoning content vs regular content
				if partResult.Get("thought").Bool() {
					template, _ = sjson.Set(template, "choices.0.delta.reasoning_content", textContent)
				} else {
					template, _ = sjson.Set(template, "choices.0.delta.content", textContent)
				}
				template, _ = sjson.Set(template, "choices.0.delta.role", "assistant")
			} else if functionCallResult.Exists() {
				hasFunctionCall = true
				toolCallsResult := gjson.Get(template, "choices.0.delta.tool_calls")
				functionCallIndex := state.FunctionIndex
				state.FunctionIndex++
				if toolCallsResult.Exists() && toolCallsResult.IsArray() {
					functionCallIndex = len(toolCallsResult.Array())
				} else {
					template, _ = sjson.SetRaw(template, "choices.0.delta.tool_calls", `[]`)
				}

				functionCallTemplate := `{"id": "","index": 0,"type": "function","function": {"name": "","arguments": ""}}`
				fcName := functionCallResult.Get("name").String()
				functionCallTemplate, _ = sjson.Set(functionCallTemplate, "id", fmt.Sprintf("%s-%d-%d", fcName, time.Now().UnixNano(), atomic.AddUint64(&functionCallIDCounter, 1)))
				functionCallTemplate, _ = sjson.Set(functionCallTemplate, "index", functionCallIndex)
				functionCallTemplate, _ = sjson.Set(functionCallTemplate, "function.name", fcName)
				if fcArgsResult := functionCallResult.Get("args"); fcArgsResult.Exists() {
					functionCallTemplate, _ = sjson.Set(functionCallTemplate, "function.arguments", fcArgsResult.Raw)
				}
				template, _ = sjson.Set(template, "choices.0.delta.role", "assistant")
				template, _ = sjson.SetRaw(template, "choices.0.delta.tool_calls.-1", functionCallTemplate)
			} else if inlineDataResult.Exists() {
				data := inlineDataResult.Get("data").String()
				if data == "" {
					continue
				}
				mimeType := inlineDataResult.Get("mimeType").String()
				if mimeType == "" {
					mimeType = inlineDataResult.Get("mime_type").String()
				}
				if mimeType == "" {
					mimeType = "image/png"
				}
				imageURL := fmt.Sprintf("data:%s;base64,%s", mimeType, data)
				imagePayload, err := json.Marshal(map[string]any{
					"type": "image_url",
					"image_url": map[string]string{
						"url": imageURL,
					},
				})
				if err != nil {
					continue
				}
				imagesResult := gjson.Get(template, "choices.0.delta.images")
				if !imagesResult.Exists() || !imagesResult.IsArray() {
					template, _ = sjson.SetRaw(template, "choices.0.delta.images", `[]`)
				}
				template, _ = sjson.Set(template, "choices.0.delta.role", "assistant")
				template, _ = sjson.SetRaw(template, "choices.0.delta.images.-1", string(imagePayload))
			}
		}
	}

	if hasFunctionCall {
		template, _ = sjson.Set(template, "choices.0.finish_reason", "tool_calls")
		template, _ = sjson.Set(template, "choices.0.native_finish_reason", "tool_calls")
	}

	return []string{template}
}

// ConvertAntigravityResponseToOpenAINonStream converts a non-streaming response.
func ConvertAntigravityResponseToOpenAINonStream(modelName string, rawJSON []byte) string {
	responseResult := gjson.GetBytes(rawJSON, "response")
	if !responseResult.Exists() {
		return ""
	}

	root := responseResult

	template := `{"id":"","object":"chat.completion","created":0,"model":"","choices":[{"index":0,"message":{"role":"assistant","content":null,"reasoning_content":null,"tool_calls":null},"finish_reason":"stop"}],"usage":{"prompt_tokens":0,"completion_tokens":0,"total_tokens":0}}`

	// Set model and ID
	if v := root.Get("modelVersion"); v.Exists() {
		template, _ = sjson.Set(template, "model", v.String())
	}
	if v := root.Get("responseId"); v.Exists() {
		template, _ = sjson.Set(template, "id", v.String())
	}

	// Set created timestamp
	if v := root.Get("createTime"); v.Exists() {
		if t, err := time.Parse(time.RFC3339Nano, v.String()); err == nil {
			template, _ = sjson.Set(template, "created", t.Unix())
		}
	}

	// Set finish reason
	if v := root.Get("candidates.0.finishReason"); v.Exists() {
		template, _ = sjson.Set(template, "choices.0.finish_reason", strings.ToLower(v.String()))
	}

	// Set usage
	if usage := root.Get("usageMetadata"); usage.Exists() {
		promptTokens := usage.Get("promptTokenCount").Int()
		candidatesTokens := usage.Get("candidatesTokenCount").Int()
		thoughtsTokens := usage.Get("thoughtsTokenCount").Int()
		totalTokens := usage.Get("totalTokenCount").Int()

		template, _ = sjson.Set(template, "usage.prompt_tokens", promptTokens)
		template, _ = sjson.Set(template, "usage.completion_tokens", candidatesTokens)
		template, _ = sjson.Set(template, "usage.total_tokens", totalTokens)
		if thoughtsTokens > 0 {
			template, _ = sjson.Set(template, "usage.completion_tokens_details.reasoning_tokens", thoughtsTokens)
		}
	}

	// Process parts
	parts := root.Get("candidates.0.content.parts")
	var contentBuilder strings.Builder
	var reasoningBuilder strings.Builder
	var toolCalls []map[string]any
	var images []map[string]any

	if parts.IsArray() {
		for _, part := range parts.Array() {
			if text := part.Get("text"); text.Exists() {
				if part.Get("thought").Bool() {
					reasoningBuilder.WriteString(text.String())
				} else {
					contentBuilder.WriteString(text.String())
				}
				continue
			}

			if fc := part.Get("functionCall"); fc.Exists() {
				toolCall := map[string]any{
					"id":   fmt.Sprintf("%s-%d", fc.Get("name").String(), len(toolCalls)),
					"type": "function",
					"function": map[string]any{
						"name":      fc.Get("name").String(),
						"arguments": fc.Get("args").Raw,
					},
				}
				toolCalls = append(toolCalls, toolCall)
				continue
			}

			if inlineData := part.Get("inlineData"); inlineData.Exists() {
				data := inlineData.Get("data").String()
				mimeType := inlineData.Get("mimeType").String()
				if mimeType == "" {
					mimeType = inlineData.Get("mime_type").String()
				}
				if mimeType == "" {
					mimeType = "image/png"
				}
				if data != "" {
					images = append(images, map[string]any{
						"type": "image_url",
						"image_url": map[string]string{
							"url": fmt.Sprintf("data:%s;base64,%s", mimeType, data),
						},
					})
				}
			}
		}
	}

	// Set content
	if contentBuilder.Len() > 0 {
		template, _ = sjson.Set(template, "choices.0.message.content", contentBuilder.String())
	}
	if reasoningBuilder.Len() > 0 {
		template, _ = sjson.Set(template, "choices.0.message.reasoning_content", reasoningBuilder.String())
	}
	if len(toolCalls) > 0 {
		template, _ = sjson.Set(template, "choices.0.message.tool_calls", toolCalls)
		template, _ = sjson.Set(template, "choices.0.finish_reason", "tool_calls")
	}
	if len(images) > 0 {
		template, _ = sjson.Set(template, "choices.0.message.images", images)
	}

	return template
}
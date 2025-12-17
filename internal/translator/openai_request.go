package translator

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/anthropics/antigravity-wrapper/internal/models"
	log "github.com/sirupsen/logrus"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
)

const geminiCLIFunctionThoughtSignature = "skip_thought_signature_validator"

// ConvertOpenAIRequestToAntigravity converts an OpenAI Chat Completions request
// into a complete Antigravity/Gemini CLI request JSON.
func ConvertOpenAIRequestToAntigravity(modelName string, inputRawJSON []byte, stream bool) []byte {
	rawJSON := bytes.Clone(inputRawJSON)
	_ = stream // stream parameter used for future extensions

	// Base envelope
	out := []byte(`{"project":"","request":{"contents":[]},"model":"gemini-2.5-pro"}`)
	out, _ = sjson.SetBytes(out, "model", modelName)

	// Handle reasoning effort -> thinkingBudget/include_thoughts
	re := gjson.GetBytes(rawJSON, "reasoning_effort")
	hasOfficialThinking := re.Exists()
	if hasOfficialThinking && models.ModelSupportsThinking(modelName) && !models.ModelUsesThinkingLevels(modelName) {
		out = models.ApplyReasoningEffortToPayload(modelName, out, re.String())
	}

	// Cherry Studio extension extra_body.google.thinking_config
	if !hasOfficialThinking && models.ModelSupportsThinking(modelName) && !models.ModelUsesThinkingLevels(modelName) {
		if tc := gjson.GetBytes(rawJSON, "extra_body.google.thinking_config"); tc.Exists() && tc.IsObject() {
			var setBudget bool
			var budget int

			if v := tc.Get("thinkingBudget"); v.Exists() {
				budget = int(v.Int())
				out, _ = sjson.SetBytes(out, "request.generationConfig.thinkingConfig.thinkingBudget", budget)
				setBudget = true
			} else if v := tc.Get("thinking_budget"); v.Exists() {
				budget = int(v.Int())
				out, _ = sjson.SetBytes(out, "request.generationConfig.thinkingConfig.thinkingBudget", budget)
				setBudget = true
			}

			if v := tc.Get("includeThoughts"); v.Exists() {
				out, _ = sjson.SetBytes(out, "request.generationConfig.thinkingConfig.include_thoughts", v.Bool())
			} else if v := tc.Get("include_thoughts"); v.Exists() {
				out, _ = sjson.SetBytes(out, "request.generationConfig.thinkingConfig.include_thoughts", v.Bool())
			} else if setBudget && budget != 0 {
				out, _ = sjson.SetBytes(out, "request.generationConfig.thinkingConfig.include_thoughts", true)
			}
		}
	}

	// Claude/Anthropic API format: thinking.type == "enabled" with budget_tokens
	if !gjson.GetBytes(out, "request.generationConfig.thinkingConfig").Exists() && models.ModelSupportsThinking(modelName) {
		if t := gjson.GetBytes(rawJSON, "thinking"); t.Exists() && t.IsObject() {
			if t.Get("type").String() == "enabled" {
				if b := t.Get("budget_tokens"); b.Exists() && b.Type == gjson.Number {
					budget := int(b.Int())
					out, _ = sjson.SetBytes(out, "request.generationConfig.thinkingConfig.thinkingBudget", budget)
					out, _ = sjson.SetBytes(out, "request.generationConfig.thinkingConfig.include_thoughts", true)
				}
			}
		}
	}

	// Temperature/top_p/top_k/max_tokens
	if tr := gjson.GetBytes(rawJSON, "temperature"); tr.Exists() && tr.Type == gjson.Number {
		out, _ = sjson.SetBytes(out, "request.generationConfig.temperature", tr.Num)
	}
	if tpr := gjson.GetBytes(rawJSON, "top_p"); tpr.Exists() && tpr.Type == gjson.Number {
		out, _ = sjson.SetBytes(out, "request.generationConfig.topP", tpr.Num)
	}
	if tkr := gjson.GetBytes(rawJSON, "top_k"); tkr.Exists() && tkr.Type == gjson.Number {
		out, _ = sjson.SetBytes(out, "request.generationConfig.topK", tkr.Num)
	}
	if maxTok := gjson.GetBytes(rawJSON, "max_tokens"); maxTok.Exists() && maxTok.Type == gjson.Number {
		out, _ = sjson.SetBytes(out, "request.generationConfig.maxOutputTokens", maxTok.Num)
	}

	// Map OpenAI modalities -> Gemini CLI responseModalities
	if mods := gjson.GetBytes(rawJSON, "modalities"); mods.Exists() && mods.IsArray() {
		var responseMods []string
		for _, m := range mods.Array() {
			switch strings.ToLower(m.String()) {
			case "text":
				responseMods = append(responseMods, "TEXT")
			case "image":
				responseMods = append(responseMods, "IMAGE")
			}
		}
		if len(responseMods) > 0 {
			out, _ = sjson.SetBytes(out, "request.generationConfig.responseModalities", responseMods)
		}
	}

	// OpenRouter-style image_config support
	if imgCfg := gjson.GetBytes(rawJSON, "image_config"); imgCfg.Exists() && imgCfg.IsObject() {
		if ar := imgCfg.Get("aspect_ratio"); ar.Exists() && ar.Type == gjson.String {
			out, _ = sjson.SetBytes(out, "request.generationConfig.imageConfig.aspectRatio", ar.Str)
		}
		if size := imgCfg.Get("image_size"); size.Exists() && size.Type == gjson.String {
			out, _ = sjson.SetBytes(out, "request.generationConfig.imageConfig.imageSize", size.Str)
		}
	}

	// messages -> systemInstruction + contents
	messages := gjson.GetBytes(rawJSON, "messages")
	if messages.IsArray() {
		arr := messages.Array()

		// First pass: assistant tool_calls id->name map
		tcID2Name := map[string]string{}
		for i := 0; i < len(arr); i++ {
			m := arr[i]
			if m.Get("role").String() == "assistant" {
				tcs := m.Get("tool_calls")
				if tcs.IsArray() {
					for _, tc := range tcs.Array() {
						if tc.Get("type").String() == "function" {
							id := tc.Get("id").String()
							name := tc.Get("function.name").String()
							if id != "" && name != "" {
								tcID2Name[id] = name
							}
						}
					}
				}
			}
		}

		// Second pass: build systemInstruction/tool responses cache
		toolResponses := map[string]string{}
		for i := 0; i < len(arr); i++ {
			m := arr[i]
			role := m.Get("role").String()
			if role == "tool" {
				toolCallID := m.Get("tool_call_id").String()
				if toolCallID != "" {
					c := m.Get("content")
					toolResponses[toolCallID] = c.Raw
				}
			}
		}

		for i := 0; i < len(arr); i++ {
			m := arr[i]
			role := m.Get("role").String()
			content := m.Get("content")

			if role == "system" && len(arr) > 1 {
				if content.Type == gjson.String {
					out, _ = sjson.SetBytes(out, "request.systemInstruction.role", "user")
					out, _ = sjson.SetBytes(out, "request.systemInstruction.parts.0.text", content.String())
				} else if content.IsObject() && content.Get("type").String() == "text" {
					out, _ = sjson.SetBytes(out, "request.systemInstruction.role", "user")
					out, _ = sjson.SetBytes(out, "request.systemInstruction.parts.0.text", content.Get("text").String())
				}
			} else if role == "user" || (role == "system" && len(arr) == 1) {
				node := []byte(`{"role":"user","parts":[]}`)
				if content.Type == gjson.String {
					node, _ = sjson.SetBytes(node, "parts.0.text", content.String())
				} else if content.IsArray() {
					items := content.Array()
					p := 0
					for _, item := range items {
						switch item.Get("type").String() {
						case "text":
							node, _ = sjson.SetBytes(node, "parts."+itoa(p)+".text", item.Get("text").String())
							p++
						case "image_url":
							imageURL := item.Get("image_url.url").String()
							if len(imageURL) > 5 {
								pieces := strings.SplitN(imageURL[5:], ";", 2)
								if len(pieces) == 2 && len(pieces[1]) > 7 {
									mime := pieces[0]
									data := pieces[1][7:]
									node, _ = sjson.SetBytes(node, "parts."+itoa(p)+".inlineData.mime_type", mime)
									node, _ = sjson.SetBytes(node, "parts."+itoa(p)+".inlineData.data", data)
									p++
								}
							}
						}
					}
				}
				out, _ = sjson.SetRawBytes(out, "request.contents.-1", node)
			} else if role == "assistant" {
				node := []byte(`{"role":"model","parts":[]}`)
				p := 0
				if content.Type == gjson.String {
					node, _ = sjson.SetBytes(node, "parts.-1.text", content.String())
					out, _ = sjson.SetRawBytes(out, "request.contents.-1", node)
					p++
				}

				// Tool calls -> single model content with functionCall parts
				tcs := m.Get("tool_calls")
				if tcs.IsArray() {
					fIDs := make([]string, 0)
					for _, tc := range tcs.Array() {
						if tc.Get("type").String() != "function" {
							continue
						}
						fid := tc.Get("id").String()
						fname := tc.Get("function.name").String()
						fargs := tc.Get("function.arguments").String()
						node, _ = sjson.SetBytes(node, "parts."+itoa(p)+".functionCall.id", fid)
						node, _ = sjson.SetBytes(node, "parts."+itoa(p)+".functionCall.name", fname)
						node, _ = sjson.SetRawBytes(node, "parts."+itoa(p)+".functionCall.args", []byte(fargs))
						node, _ = sjson.SetBytes(node, "parts."+itoa(p)+".thoughtSignature", geminiCLIFunctionThoughtSignature)
						p++
						if fid != "" {
							fIDs = append(fIDs, fid)
						}
					}
					out, _ = sjson.SetRawBytes(out, "request.contents.-1", node)

					// Append function responses
					toolNode := []byte(`{"role":"user","parts":[]}`)
					pp := 0
					for _, fid := range fIDs {
						if name, ok := tcID2Name[fid]; ok {
							toolNode, _ = sjson.SetBytes(toolNode, "parts."+itoa(pp)+".functionResponse.id", fid)
							toolNode, _ = sjson.SetBytes(toolNode, "parts."+itoa(pp)+".functionResponse.name", name)
							resp := toolResponses[fid]
							if resp == "" {
								resp = "{}"
							}
							if resp != "null" {
								parsed := gjson.Parse(resp)
								if parsed.Type == gjson.JSON {
									toolNode, _ = sjson.SetRawBytes(toolNode, "parts."+itoa(pp)+".functionResponse.response.result", []byte(parsed.Raw))
								} else {
									toolNode, _ = sjson.SetBytes(toolNode, "parts."+itoa(pp)+".functionResponse.response.result", resp)
								}
							}
							pp++
						}
					}
					if pp > 0 {
						out, _ = sjson.SetRawBytes(out, "request.contents.-1", toolNode)
					}
				}
			}
		}
	}

	// tools -> request.tools[0].functionDeclarations
	tools := gjson.GetBytes(rawJSON, "tools")
	if tools.IsArray() && len(tools.Array()) > 0 {
		toolNode := []byte(`{}`)
		hasTool := false
		hasFunction := false
		for _, t := range tools.Array() {
			if t.Get("type").String() == "function" {
				fn := t.Get("function")
				if fn.Exists() && fn.IsObject() {
					fnRaw := fn.Raw
					if fn.Get("parameters").Exists() {
						renamed, errRename := renameKey(fnRaw, "parameters", "parametersJsonSchema")
						if errRename != nil {
							log.Warnf("Failed to rename parameters for tool '%s': %v", fn.Get("name").String(), errRename)
							fnRaw, _ = sjson.Set(fnRaw, "parametersJsonSchema.type", "object")
							fnRaw, _ = sjson.Set(fnRaw, "parametersJsonSchema.properties", map[string]interface{}{})
						} else {
							fnRaw = renamed
						}
					} else {
						fnRaw, _ = sjson.Set(fnRaw, "parametersJsonSchema.type", "object")
						fnRaw, _ = sjson.Set(fnRaw, "parametersJsonSchema.properties", map[string]interface{}{})
					}
					fnRaw, _ = sjson.Delete(fnRaw, "strict")
					if !hasFunction {
						toolNode, _ = sjson.SetRawBytes(toolNode, "functionDeclarations", []byte("[]"))
					}
					toolNode, _ = sjson.SetRawBytes(toolNode, "functionDeclarations.-1", []byte(fnRaw))
					hasFunction = true
					hasTool = true
				}
			}
			if gs := t.Get("google_search"); gs.Exists() {
				toolNode, _ = sjson.SetRawBytes(toolNode, "googleSearch", []byte(gs.Raw))
				hasTool = true
			}
		}
		if hasTool {
			out, _ = sjson.SetRawBytes(out, "request.tools", []byte("[]"))
			out, _ = sjson.SetRawBytes(out, "request.tools.0", toolNode)
		}
	}

	return AttachDefaultSafetySettings(out, "request.safetySettings")
}

func itoa(i int) string { return fmt.Sprintf("%d", i) }

func renameKey(jsonStr, oldKey, newKey string) (string, error) {
	value := gjson.Get(jsonStr, oldKey)
	if !value.Exists() {
		return jsonStr, fmt.Errorf("key %s not found", oldKey)
	}
	result, err := sjson.SetRaw(jsonStr, newKey, value.Raw)
	if err != nil {
		return jsonStr, err
	}
	result, _ = sjson.Delete(result, oldKey)
	return result, nil
}

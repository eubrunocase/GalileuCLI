package guardian

import (
	"encoding/json"
	"strings"
)

type ProviderInfo struct {
	Provider string
	Model    string
}

func InferFromPayload(body []byte) ProviderInfo {
	if len(body) == 0 {
		return ProviderInfo{Provider: "unknown", Model: ""}
	}

	var data map[string]interface{}
	if err := json.Unmarshal(body, &data); err != nil {
		return ProviderInfo{Provider: "unknown", Model: ""}
	}

	info := ProviderInfo{Provider: "unknown", Model: ""}

	if model, ok := data["model"].(string); ok {
		info.Model = model
		info.Provider = inferProviderFromModel(model)
	}

	if providerSpecific, ok := detectProviderFromFields(data); ok {
		info.Provider = providerSpecific
	}

	return info
}

func inferProviderFromModel(model string) string {
	modelLower := strings.ToLower(model)

	if strings.HasPrefix(modelLower, "gpt") {
		return "openai"
	}
	if strings.HasPrefix(modelLower, "claude") || strings.HasPrefix(modelLower, "sonnet") || strings.HasPrefix(modelLower, "opus") {
		return "anthropic"
	}
	if strings.HasPrefix(modelLower, "gemini") {
		return "google"
	}
	if strings.HasPrefix(modelLower, "command-") {
		return "claude_code"
	}
	if strings.HasPrefix(modelLower, "cursor-") {
		return "cursor"
	}
	if strings.HasPrefix(modelLower, "windsurf-") {
		return "windsurf"
	}
	if strings.HasPrefix(modelLower, "mistral") {
		return "mistral"
	}
	if strings.HasPrefix(modelLower, "cohere-") {
		return "cohere"
	}

	if strings.Contains(modelLower, "o1") || strings.Contains(modelLower, "o3") || strings.Contains(modelLower, "o4") {
		return "openai"
	}

	if strings.Contains(modelLower, "deepseek") {
		return "deepseek"
	}

	if strings.Contains(modelLower, "llama") || strings.Contains(modelLower, "mixtral") {
		return "openrouter"
	}

	return "unknown"
}

func detectProviderFromFields(data map[string]interface{}) (string, bool) {
	if _, ok := data["anthropic_version"]; ok {
		return "anthropic", true
	}

	if _, ok := data["contents"]; ok {
		if _, hasSystem := data["system_instruction"]; hasSystem {
			return "gemini", true
		}
		return "google", true
	}

	if _, ok := data["prompt"]; ok {
		if _, hasVersion := data["version"]; hasVersion {
			return "cohere", true
		}
	}

	if messages, ok := data["messages"].([]interface{}); ok && len(messages) > 0 {
		if msg, ok := messages[0].(map[string]interface{}); ok {
			if _, hasContent := msg["content"]; hasContent {
				return "openai", true
			}
		}
	}

	return "", false
}

func extractModelFromPayload(body []byte) string {
	var data map[string]interface{}
	if err := json.Unmarshal(body, &data); err != nil {
		return ""
	}

	if model, ok := data["model"].(string); ok {
		return model
	}

	return ""
}

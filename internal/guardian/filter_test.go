package guardian

import (
	"testing"
)

func TestMatchWildcard_ExactMatch(t *testing.T) {
	tests := []struct {
		host    string
		pattern string
		want    bool
	}{
		{"api.openai.com", "api.openai.com", true},
		{"api.openai.com", "api.openai.com", true},
		{"api.openai.com", "api.anthropic.com", false},
		{"api.anthropic.com", "api.openai.com", false},
	}

	for _, tt := range tests {
		t.Run(tt.host+"_"+tt.pattern, func(t *testing.T) {
			if got := matchWildcard(tt.host, tt.pattern); got != tt.want {
				t.Errorf("matchWildcard(%q, %q) = %v, want %v", tt.host, tt.pattern, got, tt.want)
			}
		})
	}
}

func TestMatchWildcard_SingleWildcard(t *testing.T) {
	tests := []struct {
		host    string
		pattern string
		want    bool
	}{
		{"api.openai.com", "*.openai.com", true},
		{"chat.openai.com", "*.openai.com", true},
		{"api.anthropic.com", "*.openai.com", false},
		{"openai.com", "*.openai.com", true},
		{"notopenai.com", "*.openai.com", false},
	}

	for _, tt := range tests {
		t.Run(tt.host+"_"+tt.pattern, func(t *testing.T) {
			if got := matchWildcard(tt.host, tt.pattern); got != tt.want {
				t.Errorf("matchWildcard(%q, %q) = %v, want %v", tt.host, tt.pattern, got, tt.want)
			}
		})
	}
}

func TestMatchWildcard_MultipleWildcards(t *testing.T) {
	tests := []struct {
		host    string
		pattern string
		want    bool
	}{
		{"telemetry.opencode.ai", "*.telemetry.*", true},
		{"events.analytics.azure.com", "*.analytics.*", true},
		{"api.openai.com", "*.telemetry.*", false},
	}

	for _, tt := range tests {
		t.Run(tt.host+"_"+tt.pattern, func(t *testing.T) {
			if got := matchWildcard(tt.host, tt.pattern); got != tt.want {
				t.Errorf("matchWildcard(%q, %q) = %v, want %v", tt.host, tt.pattern, got, tt.want)
			}
		})
	}
}

func TestMatchWildcard_CaseInsensitive(t *testing.T) {
	tests := []struct {
		host    string
		pattern string
		want    bool
	}{
		{"API.OPENAI.COM", "*.openai.com", true},
		{"api.OpenAI.com", "*.OPENAI.COM", true},
		{"Api.Openai.Com", "*.openai.com", true},
	}

	for _, tt := range tests {
		t.Run(tt.host+"_"+tt.pattern, func(t *testing.T) {
			if got := matchWildcard(tt.host, tt.pattern); got != tt.want {
				t.Errorf("matchWildcard(%q, %q) = %v, want %v", tt.host, tt.pattern, got, tt.want)
			}
		})
	}
}

func TestIsJSONContentType(t *testing.T) {
	tests := []struct {
		contentType string
		want        bool
	}{
		{"application/json", true},
		{"application/json; charset=utf-8", true},
		{"Application/Json", true},
		{"application/x-www-form-urlencoded", true},
		{"text/plain", false},
		{"", false},
		{"text/html", false},
		{"multipart/form-data", false},
	}

	for _, tt := range tests {
		t.Run(tt.contentType, func(t *testing.T) {
			if got := isJSONContentType(tt.contentType); got != tt.want {
				t.Errorf("isJSONContentType(%q) = %v, want %v", tt.contentType, got, tt.want)
			}
		})
	}
}

func TestIsAIRequest_OpenAI(t *testing.T) {
	jsonBody := []byte(`{
		"model": "gpt-4",
		"messages": [{"role": "user", "content": "Hello"}]
	}`)
	if !isAIRequest(jsonBody) {
		t.Error("Expected OpenAI request to be detected as AI request")
	}
}

func TestIsAIRequest_Anthropic(t *testing.T) {
	jsonBody := []byte(`{
		"model": "claude-3-sonnet-20240229",
		"messages": [{"role": "user", "content": "Hello"}],
		"anthropic_version": "bedrock-2023-05-31"
	}`)
	if !isAIRequest(jsonBody) {
		t.Error("Expected Anthropic request to be detected as AI request")
	}
}

func TestIsAIRequest_Gemini(t *testing.T) {
	jsonBody := []byte(`{
		"contents": [{"role": "user", "parts": [{"text": "Hello"}]}],
		"generationConfig": {"temperature": 0.9}
	}`)
	if !isAIRequest(jsonBody) {
		t.Error("Expected Gemini request to be detected as AI request")
	}
}

func TestIsAIRequest_SystemInstruction(t *testing.T) {
	jsonBody := []byte(`{
		"contents": [{"role": "user", "parts": [{"text": "Hello"}]}],
		"system_instruction": {"parts": [{"text": "You are helpful"}]}
	}`)
	if !isAIRequest(jsonBody) {
		t.Error("Expected Gemini request with system_instruction to be detected as AI request")
	}
}

func TestIsAIRequest_WithPrompt(t *testing.T) {
	jsonBody := []byte(`{
		"prompt": "Hello, how are you?",
		"model": "command-r-plus"
	}`)
	if !isAIRequest(jsonBody) {
		t.Error("Expected request with prompt to be detected as AI request")
	}
}

func TestIsAIRequest_WithInput(t *testing.T) {
	jsonBody := []byte(`{
		"input": "Hello, how are you?",
		"model": "some-model"
	}`)
	if !isAIRequest(jsonBody) {
		t.Error("Expected request with input to be detected as AI request")
	}
}

func TestIsAIRequest_NonAI(t *testing.T) {
	jsonBody := []byte(`{
		"event": "page_view",
		"timestamp": "2024-01-01T00:00:00Z"
	}`)
	if isAIRequest(jsonBody) {
		t.Error("Expected non-AI request to NOT be detected as AI request")
	}
}

func TestIsAIRequest_Empty(t *testing.T) {
	if isAIRequest([]byte{}) {
		t.Error("Expected empty body to NOT be detected as AI request")
	}
}

func TestIsAIRequest_InvalidJSON(t *testing.T) {
	if isAIRequest([]byte("not json")) {
		t.Error("Expected invalid JSON to NOT be detected as AI request")
	}
}

func TestShouldProcessRequest_PostWithJSON(t *testing.T) {
	GlobalProxyConfig.Mode = "whitelist"
	GlobalProxyConfig.AllowedHosts = []string{"*.openai.com"}
	GlobalProxyConfig.SkipHosts = []string{}

	body := []byte(`{"model": "gpt-4", "messages": []}`)

	shouldProcess, _ := ShouldProcessRequest("api.openai.com", "POST", "application/json", body)
	if !shouldProcess {
		t.Error("Expected POST request with JSON body to be processed")
	}
}

func TestShouldProcessRequest_NonPost(t *testing.T) {
	body := []byte(`{"model": "gpt-4"}`)

	shouldProcess, _ := ShouldProcessRequest("api.openai.com", "GET", "application/json", body)
	if shouldProcess {
		t.Error("Expected GET request to NOT be processed")
	}
}

func TestShouldProcessRequest_NonJSON(t *testing.T) {
	body := []byte(`key=value`)

	shouldProcess, _ := ShouldProcessRequest("api.openai.com", "POST", "text/plain", body)
	if shouldProcess {
		t.Error("Expected non-JSON request to NOT be processed")
	}
}

func TestShouldProcessRequest_SkipHost(t *testing.T) {
	GlobalProxyConfig.SkipHosts = []string{"*.telemetry.*"}

	shouldProcess, _ := ShouldProcessRequest("events.telemetry.opencode.ai", "POST", "application/json", []byte(`{}`))
	if shouldProcess {
		t.Error("Expected telemetry host to be skipped")
	}
}

func TestShouldProcessRequest_WhitelistMode(t *testing.T) {
	GlobalProxyConfig.Mode = "whitelist"
	GlobalProxyConfig.AllowedHosts = []string{"*.openai.com"}

	body := []byte(`{"model": "gpt-4"}`)

	shouldProcess, _ := ShouldProcessRequest("api.anthropic.com", "POST", "application/json", body)
	if shouldProcess {
		t.Error("Expected non-whitelisted host to NOT be processed in whitelist mode")
	}
}

func TestShouldProcessRequest_PassiveMode(t *testing.T) {
	GlobalProxyConfig.Mode = "passive"
	GlobalProxyConfig.SkipHosts = []string{}

	body := []byte(`{"model": "claude-3", "messages": []}`)

	shouldProcess, _ := ShouldProcessRequest("any.host.com", "POST", "application/json", body)
	if !shouldProcess {
		t.Error("Expected any AI request to be processed in passive mode")
	}
}

func TestShouldProcessRequest_NonAIInPassiveMode(t *testing.T) {
	GlobalProxyConfig.Mode = "passive"
	GlobalProxyConfig.SkipHosts = []string{}

	body := []byte(`{"event": "click", "button": "submit"}`)

	shouldProcess, _ := ShouldProcessRequest("analytics.host.com", "POST", "application/json", body)
	if shouldProcess {
		t.Error("Expected non-AI request to NOT be processed even in passive mode")
	}
}

func TestInferFromPayload_OpenAI(t *testing.T) {
	body := []byte(`{"model": "gpt-4-turbo", "messages": []}`)
	info := InferFromPayload(body)

	if info.Provider != "openai" {
		t.Errorf("Expected provider 'openai', got '%s'", info.Provider)
	}
	if info.Model != "gpt-4-turbo" {
		t.Errorf("Expected model 'gpt-4-turbo', got '%s'", info.Model)
	}
}

func TestInferFromPayload_Anthropic(t *testing.T) {
	body := []byte(`{"model": "claude-3-opus-20240229", "messages": [], "anthropic_version": "bedrock-2023-05-31"}`)
	info := InferFromPayload(body)

	if info.Provider != "anthropic" {
		t.Errorf("Expected provider 'anthropic', got '%s'", info.Provider)
	}
}

func TestInferFromPayload_ClaudeCode(t *testing.T) {
	body := []byte(`{"model": "command-r-plus", "messages": []}`)
	info := InferFromPayload(body)

	if info.Provider != "claude_code" {
		t.Errorf("Expected provider 'claude_code', got '%s'", info.Provider)
	}
}

func TestInferFromPayload_Google(t *testing.T) {
	body := []byte(`{"contents": [{"role": "user", "parts": []}], "generationConfig": {}}`)
	info := InferFromPayload(body)

	if info.Provider != "google" {
		t.Errorf("Expected provider 'google', got '%s'", info.Provider)
	}
}

func TestInferFromPayload_Unknown(t *testing.T) {
	body := []byte(`{"some_field": "some_value"}`)
	info := InferFromPayload(body)

	if info.Provider != "unknown" {
		t.Errorf("Expected provider 'unknown', got '%s'", info.Provider)
	}
}

func TestInferFromPayload_Empty(t *testing.T) {
	info := InferFromPayload([]byte{})

	if info.Provider != "unknown" {
		t.Errorf("Expected provider 'unknown' for empty body, got '%s'", info.Provider)
	}
}

func TestInferFromPayload_InvalidJSON(t *testing.T) {
	info := InferFromPayload([]byte("not json"))

	if info.Provider != "unknown" {
		t.Errorf("Expected provider 'unknown' for invalid JSON, got '%s'", info.Provider)
	}
}

func TestShouldProcessRequest_PUT(t *testing.T) {
	GlobalProxyConfig.Mode = "passive"
	GlobalProxyConfig.SkipHosts = []string{}

	body := []byte(`{"model": "gpt-4", "messages": []}`)

	shouldProcess, _ := ShouldProcessRequest("api.openai.com", "PUT", "application/json", body)
	if !shouldProcess {
		t.Error("Expected PUT request with JSON body to be processed")
	}
}

func TestIsLLMProvider(t *testing.T) {
	tests := []struct {
		host string
		want bool
	}{
		{"api.openai.com", true},
		{"api.anthropic.com", true},
		{"generativelanguage.googleapis.com", true},
		{"api.cohere.ai", true},
		{"api.mistral.ai", true},
		{"opencode.ai", true},
		{"some.other.com", false},
	}

	for _, tt := range tests {
		t.Run(tt.host, func(t *testing.T) {
			if got := isLLMProvider(tt.host); got != tt.want {
				t.Errorf("isLLMProvider(%q) = %v, want %v", tt.host, got, tt.want)
			}
		})
	}
}

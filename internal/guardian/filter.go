package guardian

import (
	"encoding/json"
	"regexp"
	"strings"
)

const (
	MaxBodySize = 5 * 1024 * 1024
)

var skipPaths = map[string]bool{
	"/_ping":         true,
	"/health":        true,
	"/healthz":       true,
	"/api/telemetry": true,
	"/telemetry":     true,
	"/v1/telemetry":  true,
	"/api/events":    true,
	"/events":        true,
}

func ShouldProcessRequest(host, method, contentType string, body []byte) (bool, payloadInfo) {
	if method != "POST" && method != "PUT" {
		return false, payloadInfo{}
	}

	if !isJSONContentType(contentType) {
		return false, payloadInfo{}
	}

	if isInSkipHosts(host) {
		return false, payloadInfo{}
	}

	if GlobalProxyConfig.Mode == "whitelist" && !isInAllowedHosts(host) {
		return false, payloadInfo{}
	}

	if isSkippedPath(host) {
		return false, payloadInfo{}
	}

	payloadInfo := extractPayloadInfo(body)

	if !isAIRequest(body) {
		return false, payloadInfo
	}

	return true, payloadInfo
}

func isJSONContentType(contentType string) bool {
	if contentType == "" {
		return false
	}
	contentType = strings.ToLower(contentType)
	return strings.Contains(contentType, "application/json") ||
		strings.Contains(contentType, "application/x-www-form-urlencoded")
}

func isInSkipHosts(host string) bool {
	for _, skipHost := range GlobalProxyConfig.SkipHosts {
		if matchWildcard(host, skipHost) {
			return true
		}
	}
	return false
}

func isInAllowedHosts(host string) bool {
	if len(GlobalProxyConfig.AllowedHosts) == 0 {
		return true
	}
	for _, allowedHost := range GlobalProxyConfig.AllowedHosts {
		if matchWildcard(host, allowedHost) {
			return true
		}
	}
	return false
}

func matchWildcard(host, pattern string) bool {
	host = strings.ToLower(host)
	pattern = strings.ToLower(pattern)

	if pattern == host {
		return true
	}

	if strings.HasPrefix(pattern, "*.") && strings.HasSuffix(pattern, ".*") {
		middle := pattern[2 : len(pattern)-2]
		idx := strings.Index(host, middle)
		return idx >= 0
	}

	if strings.HasPrefix(pattern, "*.") {
		domain := pattern[2:]
		if host == domain {
			return true
		}
		if strings.HasSuffix(host, "."+domain) {
			return true
		}
		return false
	}

	if strings.Contains(pattern, "*") {
		regexPattern := strings.ReplaceAll(pattern, ".", "\\.")
		regexPattern = strings.ReplaceAll(regexPattern, "*", ".*")
		regexPattern = "^" + regexPattern + "$"
		matched, _ := regexp.MatchString(regexPattern, host)
		return matched
	}

	return false
}

func isSkippedPath(host string) bool {
	return skipPaths[host]
}

func isAIRequest(body []byte) bool {
	if len(body) == 0 || len(body) > MaxBodySize {
		return false
	}

	var data map[string]interface{}
	if err := json.Unmarshal(body, &data); err != nil {
		return false
	}

	if _, ok := data["model"]; ok {
		return true
	}

	if messages, ok := data["messages"].([]interface{}); ok && len(messages) > 0 {
		return true
	}

	if _, ok := data["contents"]; ok {
		return true
	}

	if _, ok := data["prompt"]; ok {
		return true
	}

	if _, ok := data["input"]; ok {
		return true
	}

	if _, ok := data["anthropic_version"]; ok {
		return true
	}

	if systemInstruction, ok := data["system_instruction"]; ok {
		if systemInstruction != nil {
			return true
		}
	}

	if generationConfig, ok := data["generationConfig"]; ok {
		if generationConfig != nil {
			return true
		}
	}

	return false
}

func isStreamingRequest(body []byte) bool {
	var data map[string]interface{}
	if err := json.Unmarshal(body, &data); err != nil {
		return false
	}

	if stream, ok := data["stream"].(bool); ok && stream {
		return true
	}

	return false
}

func isLLMProvider(host string) bool {
	h := strings.ToLower(host)
	return strings.Contains(h, "openai.com") ||
		strings.Contains(h, "anthropic.com") ||
		strings.Contains(h, "generativelanguage") ||
		strings.Contains(h, "aistudio") ||
		strings.Contains(h, "cohere.ai") ||
		strings.Contains(h, "mistral.ai") ||
		strings.Contains(h, "opencode.ai")
}

package guardian

import (
	"strings"
)

var skipPaths = map[string]bool{
	"/_ping":   true,
	"/health":  true,
	"/healthz": true,
}

var skipHosts = map[string]bool{
	"mobile.events.data.microsoft.com": true,
	"telemetry.opencode.ai":            true,
}

var llmSuffixes = []string{
	"/completions",
	"/chat/completions",
}

func ShouldAnalyze(host, method, path string) bool {
	if method != "POST" {
		return false
	}

	if skipPaths[path] {
		return false
	}

	if skipHosts[host] {
		return false
	}

	if !isLLMProvider(host) {
		return false
	}

	for _, suffix := range llmSuffixes {
		if strings.HasSuffix(path, suffix) {
			return true
		}
	}

	return true
}

func isLLMProvider(host string) bool {
	h := strings.ToLower(host)
	return strings.Contains(h, "openai.com") ||
		strings.Contains(h, "anthropic.com") ||
		strings.Contains(h, "generativelanguage") ||
		strings.Contains(h, "cohere.ai") ||
		strings.Contains(h, "mistral.ai") ||
		strings.Contains(h, "opencode.ai")
}

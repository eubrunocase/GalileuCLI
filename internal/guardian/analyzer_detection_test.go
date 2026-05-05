package guardian

import (
	"testing"
)

func getTestPatterns() []CompiledPattern {
	cfg := BuiltInConfig{
		OpenAIKey:     true,
		OpenAIProject: true,
		AnthropicKey:  true,
		GoogleKey:     true,
		GitHubToken:   true,
		SlackToken:    true,
		DiscordToken:  true,
		AWSKey:        true,
	}
	return compileBuiltIns(cfg)
}

func TestAnalyzerTruePositives(t *testing.T) {
	cases := []struct {
		name       string
		payload    string
		shouldFind string
	}{
		// OpenAI Keys
		{"openai key", `{"key": "sk-abc1234567890abcdefghij"}`, "openai_key"},
		{"openai key complex", `api_key: "sk-PROJabcdefghijklmnopqrstu1234"}`, "openai_key"},
		{"openai project key", `{"key": "sk-proj-abc1234567890abcdefghij"}`, "openai_project_key"},
		{"openai project key with dots", `{"key": "sk-proj-.abc123-4567-8901-abcd-efghijklmnop"}`, "openai_project_key"},

		// Anthropic Keys
		{"anthropic key", `{"key": "sk-ant-abc1234567890abcdefghij"}`, "anthropic_key"},
		{"anthropic key dash", `{"api_key": "sk-ant-anthropic-abc12345678"}`, "anthropic_key"},

		// Google Keys
		{"google key", `{"key": "AIzaSyABC1234567890abcdefghijKLMNOP12345"}`, "google_key"},
		{"google key with underscore", `{"api_key": "AIzaSy_test_key_1234567890abcdEFGH"}`, "google_key"},

		// GitHub Tokens
		{"github token", `{"token": "ghp_abcdef1234567890abcdef123456"}`, "github_token"},
		{"github token long", `{"auth": "ghp_1234567890abcdef1234567890abcd"}`, "github_token"},

		// Slack/Discord Tokens (mesmo regex: xox[baprs])
		{"slack token xoxb", `{"token": "xoxb-1234567890ab-123456789012"}`, "slack_token"},
		{"slack token xoxp", `{"token": "xoxp-1234567890ab-12345678901234"}`, "slack_token"},
		{"slack token xoxr", `{"token": "xoxr-1234567890ab-123456789012"}`, "slack_token"},
		{"slack token xoxa", `{"token": "xoxa-1234567890ab-123456789012"}`, "slack_token"},
		{"slack token xoxs", `{"token": "xoxs-1234567890ab-123456789012"}`, "slack_token"},
		// Discord tokens têm formato diferente (iniciam com número), este é o padrão suportado atualmente

		// AWS Keys (exact 16 chars after AKIA)
		{"aws access key", `{"key": "AKIAIOSFODNN7EXAMPLE"}`, "aws_access_key"},
	}

	patterns := getTestPatterns()
	analyzer := NewAnalyzer(patterns)

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			result := analyzer.Analyze([]byte(tc.payload))
			if !result.Modified {
				t.Errorf("esperava detecção em %q, mas não foi modificado", tc.payload)
			}
			found := false
			for _, p := range result.DetectedPatterns {
				if p == tc.shouldFind {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("esperava padrão %q, padrões detectados: %v", tc.shouldFind, result.DetectedPatterns)
			}
		})
	}
}

func TestAnalyzerNoFalsePositives(t *testing.T) {
	benign := []struct {
		name    string
		payload string
	}{
		// UUIDs - não devem ser detectados
		{"uuid v4", `{"id": "550e8400-e29b-41d4-a716-446655440000"}`},
		{"uuid in request", `{"request_id": "f47ac10b-58cc-4372-a567-0e02b2c3d479"}`},

		// MD5 hashes - não devem ser detectados
		{"md5 empty", `{"hash": "d41d8cd98f00b204e9800998ecf8427e"}`},
		{"md5 random", `{"checksum": "6dcd4ce23d04e2feabc2cf5ccf5d4b7e"}`},

		// SHA hashes - não devem ser detectados
		{"sha256", `{"hash": "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"}`},
		{"sha1", `{"hash": "da39a3ee5e6b4b0d3255bfef95601890afd80709"}`},

		// Base64 strings - não devem ser detectados
		{"base64 jwt header", `{"token": "Bearer eyJhbGciOiJIUzI1NiJ9"}`},
		{"base64 long", `{"data": "SGVsbG8gV29ybGQgSGVsbG8gV29ybGQgSGVsbG8gV29ybGQh"}`},
		{"base64 encoded", `{"value": "aHR0cHM6Ly9leGFtcGxlLmNvbS9hcGkvdjEvc2VjcmV0"}`},

		// Números e IDs legítimos
		{"numeric id", `{"user_id": 1234567890}`},
		{"numeric key", `{"api_version": 2024}`},

		// Payloads normais de API
		{"openai chat", `{"model": "gpt-4o", "messages": [{"role": "user", "content": "Hello"}], "stream": true}`},
		{"anthropic chat", `{"model": "claude-3-opus-20240229", "max_tokens": 1024, "messages": [{"role": "user", "content": "Hi"}]}`},
		{"google chat", `{"contents": [{"role": "user", "parts": [{"text": "Hello"}]}]}`},

		// Nomes de métodos e funções
		{"go method name", `{"method": "GetUserById"}`},
		{"go function", `{"function": "calculateTotalPrice"}`},
		{"variable", `{"variable": "myApiKey"}`},

		// URLs e caminhos
		{"url", `{"endpoint": "https://api.example.com/v1/users"}`},
		{"file path", `{"path": "/usr/local/bin/gateway"}`},

		// Strings contendo "key" mas não são keys
		{"key name only", `{"key": "my_secret_key_name"}`},
		{"api key name", `{"apiKey": "primary"}`},

		// JSON estruturado normal
		{"json array", `{"users": [{"id": 1, "name": "John"}, {"id": 2, "name": "Jane"}]}`},
		{"nested json", `{"data": {"result": {"status": "success"}}}`},

		// Tokens de outros serviços (não suportados = não deve ser detectado)
		{"notion token", `{"token": "secret_notion_abc123xyz456"}`},
		{"stripe token", `{"token": "sk_test_fake1234567890abcdefghijklmnop"}`},

		// Miscellaneous
		{"empty object", `{}`},
		{"empty array", `[]`},
		{"null value", `{"key": null}`},
		{"boolean", `{"enabled": true}`},
		{"number", `{"count": 42}`},
		{"float", `{"temperature": 0.7}`},
		{"array of strings", `{"keywords": ["hello", "world"]}`},
	}

	patterns := getTestPatterns()
	analyzer := NewAnalyzer(patterns)

	falsePositives := 0
	for _, tc := range benign {
		result := analyzer.Analyze([]byte(tc.payload))
		if result.Modified {
			t.Logf("FALSO POSITIVO: %q → %v", tc.payload, result.DetectedPatterns)
			falsePositives++
		}
	}

	if falsePositives > 0 {
		t.Errorf("Taxa de falso positivo: %d/%d (%.2f%%)", falsePositives, len(benign), float64(falsePositives)*100/float64(len(benign)))
	} else {
		t.Logf("Taxa de falso positivo: 0/%d (0.00%%)", len(benign))
	}
}

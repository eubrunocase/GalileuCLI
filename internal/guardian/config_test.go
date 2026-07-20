package guardian

import (
	"os"
	"path/filepath"
	"testing"
)

// ─── LoadRawConfig ────────────────────────────────────────────────────────────

func TestLoadRawConfig_ReturnsDefaultsWhenFileAbsent(t *testing.T) {
	cfg, err := LoadRawConfig(filepath.Join(t.TempDir(), "nonexistent.yml"))

	if err != nil {
		t.Fatalf("expected no error for missing file, got: %v", err)
	}
	if cfg.Port != 9000 {
		t.Errorf("expected default port 9000, got %d", cfg.Port)
	}
	if cfg.Proxy.Mode != "whitelist" {
		t.Errorf("expected default mode 'whitelist', got %q", cfg.Proxy.Mode)
	}
	if !cfg.Analyzer.BuiltIn.OpenAIKey {
		t.Error("expected openai_key enabled by default")
	}
}

func TestLoadRawConfig_ParsesValidYAML(t *testing.T) {
	yml := `
port: 8080
proxy:
  mode: passive
analyzer:
  built_in:
    openai_key: true
    anthropic_key: false
  custom_patterns:
    - name: "Test"
      type: regex
      pattern: 'foo'
      label: "[FOO]"
      enabled: true
`
	path := writeTemp(t, yml)

	cfg, err := LoadRawConfig(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.Port != 8080 {
		t.Errorf("expected port 8080, got %d", cfg.Port)
	}
	if cfg.Proxy.Mode != "passive" {
		t.Errorf("expected mode 'passive', got %q", cfg.Proxy.Mode)
	}
	if !cfg.Analyzer.BuiltIn.OpenAIKey {
		t.Error("expected openai_key true")
	}
	if cfg.Analyzer.BuiltIn.AnthropicKey {
		t.Error("expected anthropic_key false")
	}
	if len(cfg.Analyzer.CustomPatterns) != 1 {
		t.Fatalf("expected 1 custom pattern, got %d", len(cfg.Analyzer.CustomPatterns))
	}
	if cfg.Analyzer.CustomPatterns[0].Name != "Test" {
		t.Errorf("expected pattern name 'Test', got %q", cfg.Analyzer.CustomPatterns[0].Name)
	}
}

func TestLoadRawConfig_ReturnsErrorForInvalidYAML(t *testing.T) {
	path := writeTemp(t, "port: [not a number")

	_, err := LoadRawConfig(path)
	if err == nil {
		t.Error("expected error for invalid YAML")
	}
}

func TestLoadRawConfig_DefaultPortWhenZero(t *testing.T) {
	path := writeTemp(t, "port: 0\n")

	cfg, err := LoadRawConfig(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Port != 9000 {
		t.Errorf("expected port defaulted to 9000 when 0, got %d", cfg.Port)
	}
}

// ─── SaveConfig ───────────────────────────────────────────────────────────────

func TestSaveConfig_WritesValidYAML(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "out.yml")

	cfg := GalileuConfig{
		Port: 1234,
		Proxy: ProxySection{
			Mode: "passive",
		},
		Analyzer: AnalyzerConfig{
			BuiltIn: BuiltInConfig{OpenAIKey: true},
		},
	}

	if err := SaveConfig(path, cfg); err != nil {
		t.Fatalf("SaveConfig failed: %v", err)
	}

	reloaded, err := LoadRawConfig(path)
	if err != nil {
		t.Fatalf("failed to reload saved config: %v", err)
	}

	if reloaded.Port != 1234 {
		t.Errorf("expected port 1234, got %d", reloaded.Port)
	}
	if reloaded.Proxy.Mode != "passive" {
		t.Errorf("expected mode 'passive', got %q", reloaded.Proxy.Mode)
	}
	if !reloaded.Analyzer.BuiltIn.OpenAIKey {
		t.Error("expected openai_key true after round-trip")
	}
}

func TestSaveConfig_ReturnsErrorForUnwritablePath(t *testing.T) {
	err := SaveConfig("/nonexistent/dir/galileu.yml", GalileuConfig{Port: 9000})
	if err == nil {
		t.Error("expected error when writing to non-existent directory")
	}
}

func TestSaveConfig_RoundTripPreservesCustomPatterns(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "cfg.yml")

	cfg := GalileuConfig{
		Port: 9000,
		Analyzer: AnalyzerConfig{
			CustomPatterns: []CustomPattern{
				{Name: "JWT", Type: "regex", Pattern: "eyJ.*", Label: "[JWT]", Enabled: true},
				{Name: "Tokens", Type: "literal", Values: []string{"abc", "def"}, Label: "[T]", Enabled: false},
			},
		},
	}

	if err := SaveConfig(path, cfg); err != nil {
		t.Fatalf("SaveConfig failed: %v", err)
	}

	reloaded, err := LoadRawConfig(path)
	if err != nil {
		t.Fatalf("reload failed: %v", err)
	}

	if len(reloaded.Analyzer.CustomPatterns) != 2 {
		t.Fatalf("expected 2 custom patterns, got %d", len(reloaded.Analyzer.CustomPatterns))
	}
	if reloaded.Analyzer.CustomPatterns[0].Name != "JWT" {
		t.Errorf("expected first pattern 'JWT', got %q", reloaded.Analyzer.CustomPatterns[0].Name)
	}
	if reloaded.Analyzer.CustomPatterns[1].Enabled {
		t.Error("expected second pattern disabled after round-trip")
	}
}

// ─── ResolveConfigPath ──────────────────────────────────────────────────────
func TestResolveConfigPath_FlagTakesPrecedence(t *testing.T) {
	dir := t.TempDir()
	flagPath := filepath.Join(dir, "team.yml")
	os.WriteFile(flagPath, []byte("port: 8080"), 0o644)
	t.Setenv("GALILEU_CONFIG", "/some/env/path.yml")
	cfg, err := ResolveConfigPath(flagPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Source != "flag" {
		t.Errorf("expected source 'flag', got %q", cfg.Source)
	}
	if cfg.Path != flagPath {
		t.Errorf("expected path %q, got %q", flagPath, cfg.Path)
	}
}
func TestResolveConfigPath_EnvFallback(t *testing.T) {
	dir := t.TempDir()
	envPath := filepath.Join(dir, "env-config.yml")
	os.WriteFile(envPath, []byte("port: 7070"), 0o644)
	t.Setenv("GALILEU_CONFIG", envPath)
	cfg, err := ResolveConfigPath("")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Source != "env" {
		t.Errorf("expected source 'env', got %q", cfg.Source)
	}
	if cfg.Path != envPath {
		t.Errorf("expected path %q, got %q", envPath, cfg.Path)
	}
}
func TestResolveConfigPath_CWDFallback(t *testing.T) {
	dir := t.TempDir()
	cwdPath := filepath.Join(dir, "galileu.yml")
	os.WriteFile(cwdPath, []byte("port: 6060"), 0o644)
	// Salvar e restaurar CWD original
	origDir, _ := os.Getwd()
	defer t.Chdir(origDir)
	t.Chdir(dir)
	t.Setenv("GALILEU_CONFIG", "")
	cfg, err := ResolveConfigPath("")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Source != "cwd" {
		t.Errorf("expected source 'cwd', got %q", cfg.Source)
	}
	if cfg.Path != "galileu.yml" {
		t.Errorf("expected path 'galileu.yml', got %q", cfg.Path)
	}
}
func TestResolveConfigPath_BuiltinFallback(t *testing.T) {
	dir := t.TempDir()
	origDir, _ := os.Getwd()
	defer os.Chdir(origDir)
	os.Chdir(dir)
	t.Setenv("GALILEU_CONFIG", "")
	cfg, err := ResolveConfigPath("")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Source != "builtin" {
		t.Errorf("expected source 'builtin', got %q", cfg.Source)
	}
	if cfg.Path != "" {
		t.Errorf("expected empty path for builtin, got %q", cfg.Path)
	}
}
func TestResolveConfigPath_FlagNonexistent(t *testing.T) {
	_, err := ResolveConfigPath("/nonexistent/path/config.yml")
	if err == nil {
		t.Error("expected error for nonexistent --config path")
	}
}
func TestResolveConfigPath_EnvNonexistent(t *testing.T) {
	t.Setenv("GALILEU_CONFIG", "/nonexistent/env-config.yml")
	_, err := ResolveConfigPath("")
	if err == nil {
		t.Error("expected error for nonexistent GALILEU_CONFIG path")
	}
}

// ─── helper ───────────────────────────────────────────────────────────────────

func writeTemp(t *testing.T, content string) string {
	t.Helper()
	f, err := os.CreateTemp(t.TempDir(), "*.yml")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	if _, err := f.WriteString(content); err != nil {
		t.Fatalf("failed to write temp file: %v", err)
	}
	f.Close()
	return f.Name()
}

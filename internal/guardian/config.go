package guardian

import (
	"fmt"
	"os"
	"regexp"

	"gopkg.in/yaml.v3"
)

// ─── Estruturas que mapeiam o galileu.yml ────────────────────────────────────

type GalileuConfig struct {
	Port     int            `yaml:"port"`
	Proxy    ProxySection   `yaml:"proxy"`
	Analyzer AnalyzerConfig `yaml:"analyzer"`
}

type ProxySection struct {
	Mode         string   `yaml:"mode"`
	AllowedHosts []string `yaml:"allowed_hosts"`
	SkipHosts    []string `yaml:"skip_hosts"`
}

type AnalyzerConfig struct {
	BuiltIn        BuiltInConfig   `yaml:"built_in"`
	CustomPatterns []CustomPattern `yaml:"custom_patterns"`
}

type BuiltInConfig struct {
	OpenAIKey     bool `yaml:"openai_key"`
	OpenAIProject bool `yaml:"openai_project_key"`
	AnthropicKey  bool `yaml:"anthropic_key"`
	GoogleKey     bool `yaml:"google_key"`
	GitHubToken   bool `yaml:"github_token"`
	SlackToken    bool `yaml:"slack_token"`
	DiscordToken  bool `yaml:"discord_token"`
	AWSKey        bool `yaml:"aws_key"`
}

type CustomPattern struct {
	Name    string   `yaml:"name"`
	Type    string   `yaml:"type"`
	Pattern string   `yaml:"pattern"`
	Values  []string `yaml:"values"`
	Label   string   `yaml:"label"`
	Enabled bool     `yaml:"enabled"`
}

type CompiledPattern struct {
	Name  string
	Regex *regexp.Regexp
	Label string
}

var builtInDefinitions = []struct {
	enabledFn func(BuiltInConfig) bool
	pattern   string
	label     string
	name      string
}{
	{
		name:      "openai_key",
		enabledFn: func(b BuiltInConfig) bool { return b.OpenAIKey },
		pattern:   `sk-[a-zA-Z0-9]{20,}`,
		label:     "[REDACTED_BY_GALILEU]",
	},
	{
		name:      "openai_project_key",
		enabledFn: func(b BuiltInConfig) bool { return b.OpenAIProject },
		pattern:   `sk-proj-[a-zA-Z0-9.-]{20,}`,
		label:     "[REDACTED_BY_GALILEU]",
	},
	{
		name:      "anthropic_key",
		enabledFn: func(b BuiltInConfig) bool { return b.AnthropicKey },
		pattern:   `sk-ant-[a-zA-Z0-9.-]{20,}`,
		label:     "[REDACTED_BY_GALILEU]",
	},
	{
		name:      "google_key",
		enabledFn: func(b BuiltInConfig) bool { return b.GoogleKey },
		pattern:   `AIzaSy[a-zA-Z0-9_-]{20,}`,
		label:     "[REDACTED_BY_GALILEU]",
	},
	{
		name:      "github_token",
		enabledFn: func(b BuiltInConfig) bool { return b.GitHubToken },
		pattern:   `ghp_[a-zA-Z0-9]{20,}`,
		label:     "[REDACTED_BY_GALILEU]",
	},
	{
		name:      "slack_token",
		enabledFn: func(b BuiltInConfig) bool { return b.SlackToken },
		pattern:   `xox[baprs]-[a-zA-Z0-9]{10,}`,
		label:     "[REDACTED_BY_GALILEU]",
	},
	{
		name:      "discord_token",
		enabledFn: func(b BuiltInConfig) bool { return b.DiscordToken },
		pattern:   `xox[baprs]-[a-zA-Z0-9]{10,}`,
		label:     "[REDACTED_BY_GALILEU]",
	},
	{
		name:      "aws_access_key",
		enabledFn: func(b BuiltInConfig) bool { return b.AWSKey },
		pattern:   `AKIA[0-9A-Z]{16}`,
		label:     "[REDACTED_BY_GALILEU]",
	},
	{
		name:      "aws_secret_key",
		enabledFn: func(b BuiltInConfig) bool { return b.AWSKey },
		pattern:   `wJalr[a-zA-Z0-9/+=]{20,}`,
		label:     "[REDACTED_BY_GALILEU]",
	},
}

// ─── Função principal de carregamento ────────────────────────────────────────

func LoadConfig(path string) (int, []CompiledPattern, error) {
	var cfg GalileuConfig

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			fmt.Printf("[Galileu] galileu.yml não encontrado em '%s'. A usar padrões built-in por omissão.\n", path)
			return 9000, compileBuiltIns(defaultBuiltInConfig()), nil
		}
		return 0, nil, fmt.Errorf("erro ao ler galileu.yml: %w", err)
	}

	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return 0, nil, fmt.Errorf("erro ao fazer parse do galileu.yml: %w", err)
	}

	var compiled []CompiledPattern

	compiled = append(compiled, compileBuiltIns(cfg.Analyzer.BuiltIn)...)

	for _, cp := range cfg.Analyzer.CustomPatterns {
		if !cp.Enabled {
			continue
		}

		switch cp.Type {
		case "regex":
			rx, err := regexp.Compile(cp.Pattern)
			if err != nil {
				fmt.Printf("[Galileu] Padrão custom '%s' tem regex inválida e será ignorado: %v\n", cp.Name, err)
				continue
			}
			compiled = append(compiled, CompiledPattern{
				Name:  cp.Name,
				Regex: rx,
				Label: cp.Label,
			})

		case "literal":
			for _, val := range cp.Values {
				if val == "" {
					continue
				}
				rx := regexp.MustCompile(regexp.QuoteMeta(val))
				compiled = append(compiled, CompiledPattern{
					Name:  cp.Name + " [" + val + "]",
					Regex: rx,
					Label: cp.Label,
				})
			}

		default:
			fmt.Printf("[Galileu] Padrão custom '%s' tem type desconhecido '%s' e será ignorado.\n", cp.Name, cp.Type)
		}
	}

	port := cfg.Port
	if port == 0 {
		port = 9000
	}

	loadProxyConfig(cfg.Proxy)

	fmt.Printf("[Galileu] %d padrão(ões) de detecção carregado(s) a partir de '%s'.\n", len(compiled), path)
	return port, compiled, nil
}

func loadProxyConfig(proxyCfg ProxySection) {
	if proxyCfg.Mode != "" {
		GlobalProxyConfig.Mode = proxyCfg.Mode
	} else {
		GlobalProxyConfig.Mode = "whitelist"
	}

	if len(proxyCfg.AllowedHosts) > 0 {
		GlobalProxyConfig.AllowedHosts = proxyCfg.AllowedHosts
	}

	if len(proxyCfg.SkipHosts) > 0 {
		GlobalProxyConfig.SkipHosts = proxyCfg.SkipHosts
	}

	fmt.Printf("[Galileu] Modo proxy: %s (hosts permitidos: %d, hosts ignorados: %d)\n",
		GlobalProxyConfig.Mode, len(GlobalProxyConfig.AllowedHosts), len(GlobalProxyConfig.SkipHosts))
}

// ─── LoadRawConfig / SaveConfig (usados pela TUI) ────────────────────────────

// LoadRawConfig reads and unmarshals galileu.yml without compiling patterns.
// Returns default values when the file does not exist.
func LoadRawConfig(path string) (GalileuConfig, error) {
	var cfg GalileuConfig

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return defaultGalileuConfig(), nil
		}
		return cfg, fmt.Errorf("erro ao ler galileu.yml: %w", err)
	}

	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return cfg, fmt.Errorf("erro ao fazer parse do galileu.yml: %w", err)
	}

	if cfg.Port == 0 {
		cfg.Port = 9000
	}

	return cfg, nil
}

// SaveConfig serialises cfg to YAML and writes it to path.
func SaveConfig(path string, cfg GalileuConfig) error {
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("erro ao serializar configuração: %w", err)
	}

	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("erro ao escrever galileu.yml: %w", err)
	}

	return nil
}

func defaultGalileuConfig() GalileuConfig {
	return GalileuConfig{
		Port: 9000,
		Proxy: ProxySection{
			Mode:         "whitelist",
			AllowedHosts: defaultAllowedHosts(),
			SkipHosts:    defaultSkipHosts(),
		},
		Analyzer: AnalyzerConfig{
			BuiltIn: defaultBuiltInConfig(),
		},
	}
}

// ─── Helpers internos ─────────────────────────────────────────────────────────

func compileBuiltIns(cfg BuiltInConfig) []CompiledPattern {
	var result []CompiledPattern
	for _, def := range builtInDefinitions {
		if def.enabledFn(cfg) {
			result = append(result, CompiledPattern{
				Name:  def.name,
				Regex: regexp.MustCompile(def.pattern),
				Label: def.label,
			})
		}
	}
	return result
}

func defaultBuiltInConfig() BuiltInConfig {
	return BuiltInConfig{
		OpenAIKey:     true,
		OpenAIProject: true,
		AnthropicKey:  true,
		GoogleKey:     true,
		GitHubToken:   true,
		SlackToken:    true,
		DiscordToken:  true,
		AWSKey:        true,
	}
}

type ProxyConfig struct {
	Mode         string
	AllowedHosts []string
	SkipHosts    []string
}

var GlobalProxyConfig = ProxyConfig{
	Mode:         "whitelist",
	AllowedHosts: defaultAllowedHosts(),
	SkipHosts:    defaultSkipHosts(),
}

func defaultAllowedHosts() []string {
	return []string{
		"*.openai.com",
		"*.anthropic.com",
		"*.generativelanguage.googleapis.com",
		"*.aistudio.googleapis.com",
		"*.cohere.ai",
		"*.mistral.ai",
		"*.opencode.ai",
	}
}

func defaultSkipHosts() []string {
	return []string{
		"mobile.events.data.microsoft.com",
		"telemetry.opencode.ai",
		"claude.telemetry.anthropic.com",
		"cursor.telemetry",
		"windsurf.telemetry",
		"*.telemetry.*",
		"*.analytics.*",
		"vscode-clientanalytics.azurewebsites.net",
		"dc.services.visualstudio.com",
	}
}

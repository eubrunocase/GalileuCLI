package guardian

import (
	"fmt"
	"os"
	"regexp"

	"gopkg.in/yaml.v3"
)

// ─── Estruturas que mapeiam o galileu.yml ────────────────────────────────────

// GalileuConfig é a raiz do ficheiro YAML.
type GalileuConfig struct {
	Analyzer AnalyzerConfig `yaml:"analyzer"`
}

// AnalyzerConfig contém os built-ins e os padrões customizados.
type AnalyzerConfig struct {
	BuiltIn        BuiltInConfig   `yaml:"built_in"`
	CustomPatterns []CustomPattern `yaml:"custom_patterns"`
}

// BuiltInConfig permite activar/desactivar cada padrão embutido individualmente.
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

// CustomPattern representa um padrão definido pelo utilizador.
// O campo Type aceita dois valores:
//   - "regex"   → o campo Pattern contém uma expressão regular
//   - "literal" → o campo Values contém strings de texto fixo
type CustomPattern struct {
	Name    string   `yaml:"name"`
	Type    string   `yaml:"type"`
	Pattern string   `yaml:"pattern"`
	Values  []string `yaml:"values"`
	Label   string   `yaml:"label"`
	Enabled bool     `yaml:"enabled"`
}

// ─── Estrutura interna compilada ─────────────────────────────────────────────

// CompiledPattern é a representação em memória de um padrão já compilado.
// É esta estrutura que o analyzer.go usa em runtime.
type CompiledPattern struct {
	Name  string
	Regex *regexp.Regexp
	Label string
}

// ─── Padrões built-in (os que existiam hardcoded no analyzer.go) ─────────────

// builtInDefinitions define os padrões embutidos do Galileu.
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
		pattern:   `AIzaSy[a-zA-Z0-9_-]{35}`,
		label:     "[REDACTED_BY_GALILEU]",
	},
	{
		name:      "github_token",
		enabledFn: func(b BuiltInConfig) bool { return b.GitHubToken },
		pattern:   `ghp_[a-zA-Z0-9]{36}`,
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
}

// ─── Função principal de carregamento ────────────────────────────────────────

// LoadConfig lê o ficheiro YAML no caminho indicado e devolve uma lista de
// CompiledPattern prontos a usar pelo analyzer.
//
// Se o ficheiro não existir, devolve os padrões built-in todos activados
// (comportamento legacy idêntico ao da versão anterior hardcoded).
func LoadConfig(path string) ([]CompiledPattern, error) {
	var cfg GalileuConfig

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			fmt.Printf("[Galileu] galileu.yml não encontrado em '%s'. A usar padrões built-in por omissão.\n", path)
			return compileBuiltIns(defaultBuiltInConfig()), nil
		}
		return nil, fmt.Errorf("erro ao ler galileu.yml: %w", err)
	}

	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("erro ao fazer parse do galileu.yml: %w", err)
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

	fmt.Printf("[Galileu] %d padrão(ões) de detecção carregado(s) a partir de '%s'.\n", len(compiled), path)
	return compiled, nil
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

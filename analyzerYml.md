# Galileu — Implementação do Sistema de Configuração YAML para o Analyzer

## Contexto e Objectivo

Este documento é uma instrução técnica para um agente de IA. O objectivo é evoluir o `analyzer.go` do projeto **GalileuForMac** (https://github.com/eubrunocase/GalileuForMac), que atualmente tem padrões de detecção de dados sensíveis **hardcoded**, para um sistema onde os padrões são **definidos pelo utilizador num ficheiro `galileu.yml`**, sem necessidade de tocar no código-fonte Go.

O agente deve seguir as instruções neste documento na ordem apresentada, sem alterar comportamentos existentes que não estejam explicitamente mencionados.

---

## Princípios que Devem Ser Respeitados

- **Zero breaking changes**: o comportamento atual deve ser inteiramente preservado. Os padrões hardcoded atuais tornam-se built-ins ativados por padrão no YAML.
- **Zero novas dependências pesadas**: apenas `gopkg.in/yaml.v3` será adicionada.
- **Zero alterações ao fluxo do proxy**: apenas `analyzer.go`, `main.go` e os novos ficheiros serão modificados.
- **O utilizador nunca toca no código Go**: toda a customização ocorre exclusivamente no `galileu.yml`.

---

## Visão Geral das Alterações

| Ficheiro | Ação |
|---|---|
| `go.mod` / `go.sum` | Adicionar dependência `gopkg.in/yaml.v3` |
| `internal/guardian/config.go` | **CRIAR** — struct e lógica de leitura do YAML |
| `internal/guardian/analyzer.go` | **MODIFICAR** — receber padrões compilados externamente |
| `cmd/sentinel/main.go` | **MODIFICAR** — carregar config antes de iniciar o proxy |
| `galileu.yml` | **CRIAR** — ficheiro de configuração de exemplo para o utilizador |

---

## Passo 1 — Adicionar a Dependência YAML

No terminal, na raiz do projeto, executar:

```bash
go get gopkg.in/yaml.v3
```

Verificar que `go.mod` passou a incluir a linha:

```
require gopkg.in/yaml.v3 vX.X.X
```

---

## Passo 2 — Criar `internal/guardian/config.go`

Criar o ficheiro `internal/guardian/config.go` com o seguinte conteúdo completo:

```go
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
	OpenAIKey      bool `yaml:"openai_key"`
	OpenAIProject  bool `yaml:"openai_project_key"`
	AnthropicKey   bool `yaml:"anthropic_key"`
	GoogleKey      bool `yaml:"google_key"`
	GitHubToken    bool `yaml:"github_token"`
	SlackToken     bool `yaml:"slack_token"`
	DiscordToken   bool `yaml:"discord_token"`
	AWSKey         bool `yaml:"aws_key"`
}

// CustomPattern representa um padrão definido pelo utilizador.
// O campo Type aceita dois valores:
//   - "regex"   → o campo Pattern contém uma expressão regular
//   - "literal" → o campo Values contém strings de texto fixo
type CustomPattern struct {
	Name    string   `yaml:"name"`
	Type    string   `yaml:"type"`    // "regex" ou "literal"
	Pattern string   `yaml:"pattern"` // usado quando Type == "regex"
	Values  []string `yaml:"values"`  // usado quando Type == "literal"
	Label   string   `yaml:"label"`   // texto de substituição, ex: "[DB_PASSWORD_REDACTED]"
	Enabled bool     `yaml:"enabled"`
}

// ─── Estrutura interna compilada ─────────────────────────────────────────────

// CompiledPattern é a representação em memória de um padrão já compilado.
// É esta estrutura que o analyzer.go usa em runtime.
type CompiledPattern struct {
	Name    string
	Regex   *regexp.Regexp
	Label   string
}

// ─── Padrões built-in (os que existiam hardcoded no analyzer.go) ─────────────

// builtInDefinitions define os padrões embutidos do Galileu.
// Cada entrada associa a flag do YAML à regex e ao label correspondentes.
var builtInDefinitions = []struct {
	enabledFn func(BuiltInConfig) bool
	pattern   string
	label     string
	name      string
}{
	{
		name:      "OpenAI API Key",
		enabledFn: func(b BuiltInConfig) bool { return b.OpenAIKey },
		pattern:   `sk-[a-zA-Z0-9]{32,}`,
		label:     "[REDACTED_BY_GALILEU]",
	},
	{
		name:      "OpenAI Project Key",
		enabledFn: func(b BuiltInConfig) bool { return b.OpenAIProject },
		pattern:   `sk-proj-[a-zA-Z0-9\-_]{32,}`,
		label:     "[REDACTED_BY_GALILEU]",
	},
	{
		name:      "Anthropic API Key",
		enabledFn: func(b BuiltInConfig) bool { return b.AnthropicKey },
		pattern:   `sk-ant-[a-zA-Z0-9\-_]{32,}`,
		label:     "[REDACTED_BY_GALILEU]",
	},
	{
		name:      "Google API Key",
		enabledFn: func(b BuiltInConfig) bool { return b.GoogleKey },
		pattern:   `AIzaSy[a-zA-Z0-9\-_]{33}`,
		label:     "[REDACTED_BY_GALILEU]",
	},
	{
		name:      "GitHub Token",
		enabledFn: func(b BuiltInConfig) bool { return b.GitHubToken },
		pattern:   `ghp_[a-zA-Z0-9]{36}`,
		label:     "[REDACTED_BY_GALILEU]",
	},
	{
		name:      "Slack Token",
		enabledFn: func(b BuiltInConfig) bool { return b.SlackToken },
		pattern:   `xox[baprs]-[a-zA-Z0-9\-]+`,
		label:     "[REDACTED_BY_GALILEU]",
	},
	{
		name:      "Discord Token",
		enabledFn: func(b BuiltInConfig) bool { return b.DiscordToken },
		pattern:   `xox[baprs]-[a-zA-Z0-9\-]+`,
		label:     "[REDACTED_BY_GALILEU]",
	},
	{
		name:      "AWS Access Key",
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
			// Ficheiro não encontrado: activar todos os built-ins por omissão
			fmt.Printf("[Galileu] galileu.yml não encontrado em '%s'. A usar padrões built-in por omissão.\n", path)
			return compileBuiltIns(defaultBuiltInConfig()), nil
		}
		return nil, fmt.Errorf("erro ao ler galileu.yml: %w", err)
	}

	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("erro ao fazer parse do galileu.yml: %w", err)
	}

	var compiled []CompiledPattern

	// 1. Compilar built-ins seleccionados
	compiled = append(compiled, compileBuiltIns(cfg.Analyzer.BuiltIn)...)

	// 2. Compilar padrões custom
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
				// Escapar o valor literal para uso seguro como regex
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

// defaultBuiltInConfig activa todos os built-ins — usado quando galileu.yml não existe.
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
```

---

## Passo 3 — Modificar `internal/guardian/analyzer.go`

### 3.1 — O que remover

Localizar e **remover completamente** o bloco onde as regex estão hardcoded. Tipicamente será algo como:

```go
// REMOVER este bloco (ou equivalente):
var sensitivePatterns = []*regexp.Regexp{
    regexp.MustCompile(`sk-[a-zA-Z0-9]{32,}`),
    regexp.MustCompile(`sk-proj-[a-zA-Z0-9\-_]{32,}`),
    // ... demais padrões
}
```

Remover também qualquer `var` ou `init()` que compile padrões inline no ficheiro.

### 3.2 — O que adicionar

O `analyzer.go` deve receber os padrões compilados via injecção. Substituir a struct do Analyzer (ou criar, se não existir) para incluir o campo de padrões:

```go
// Analyzer é o motor de sanitização de dados sensíveis.
type Analyzer struct {
    patterns []CompiledPattern
}

// NewAnalyzer cria um Analyzer com os padrões fornecidos externamente.
// Os padrões são compilados uma vez em LoadConfig e reutilizados aqui.
func NewAnalyzer(patterns []CompiledPattern) *Analyzer {
    return &Analyzer{patterns: patterns}
}
```

### 3.3 — Adaptar a função de sanitização

A função que faz a substituição no corpo da requisição deve iterar sobre `a.patterns` em vez de uma variável global. O padrão de implementação deve ser:

```go
// Sanitize recebe o corpo de uma requisição em bytes e devolve
// o corpo sanitizado e um booleano indicando se houve redacção.
func (a *Analyzer) Sanitize(body []byte) ([]byte, bool) {
    redacted := false
    result := body

    for _, p := range a.patterns {
        if p.Regex.Match(result) {
            result = p.Regex.ReplaceAll(result, []byte(p.Label))
            redacted = true
        }
    }

    return result, redacted
}
```

**Nota importante para o agente**: preservar integralmente qualquer outra lógica existente no `analyzer.go` que não seja a declaração e uso dos padrões hardcoded. Apenas os padrões são migrados para o novo sistema.

---

## Passo 4 — Modificar `cmd/sentinel/main.go`

### 4.1 — Carregar a configuração antes de iniciar o proxy

Localizar o ponto de entrada `main()` e adicionar o carregamento da config **antes** da inicialização do proxy. O caminho padrão do ficheiro YAML é `./galileu.yml` (relativo ao directório onde o binário é executado).

```go
func main() {
    // 1. Carregar padrões a partir do galileu.yml
    patterns, err := guardian.LoadConfig("galileu.yml")
    if err != nil {
        log.Fatalf("[Galileu] Falha ao carregar configuração: %v", err)
    }

    // 2. Criar o Analyzer com os padrões carregados
    analyzer := guardian.NewAnalyzer(patterns)

    // 3. Passar o analyzer ao Guardian/Proxy (ajustar conforme a assinatura actual)
    // Exemplo: guardian.NewProxy(analyzer, ...)
    //
    // O restante do main permanece inalterado.
}
```

**Nota para o agente**: ajustar a chamada de inicialização do proxy para receber o `analyzer` como argumento, se ainda não o fizer. Não alterar nenhuma outra lógica do `main.go`.

---

## Passo 5 — Criar `galileu.yml` na Raiz do Projecto

Criar o ficheiro `galileu.yml` na raiz do projecto com o seguinte conteúdo. Este é o ficheiro que o utilizador irá editar:

```yaml
# ═══════════════════════════════════════════════════════════════════════════════
# galileu.yml — Configuração do Analyzer de Dados Sensíveis
#
# Este ficheiro controla o comportamento do Galileu.
# Não é necessário tocar no código Go para alterar os padrões de detecção.
#
# TIPOS DE PADRÃO CUSTOM:
#   type: regex   → O campo 'pattern' aceita uma expressão regular
#   type: literal → O campo 'values' aceita uma lista de strings exactas
# ═══════════════════════════════════════════════════════════════════════════════

analyzer:

  # ─── Padrões embutidos ───────────────────────────────────────────────────────
  # Activar (true) ou desactivar (false) cada padrão built-in individualmente.
  built_in:
    openai_key:         true
    openai_project_key: true
    anthropic_key:      true
    google_key:         true
    github_token:       true
    slack_token:        true
    discord_token:      true
    aws_key:            true

  # ─── Padrões personalizados ──────────────────────────────────────────────────
  # Adicionar quantos padrões forem necessários.
  # Cada padrão tem um nome descritivo, um tipo, e um label de substituição.
  custom_patterns:

    # Exemplo: Detectar passwords de base de dados em variáveis de ambiente
    - name: "Password de Base de Dados"
      type: regex
      pattern: 'DB_PASSWORD\s*[=:]\s*["\']?([^\s"''<>]+)'
      label: "[DB_PASSWORD_REDACTED]"
      enabled: false   # Mudar para true para activar

    # Exemplo: Detectar connection strings completas (Postgres, MySQL, MongoDB)
    - name: "Connection String de Base de Dados"
      type: regex
      pattern: '(postgres|postgresql|mysql|mongodb|redis):\/\/[^\s"''<>]+'
      label: "[CONNECTION_STRING_REDACTED]"
      enabled: false

    # Exemplo: Detectar nomes de tabelas internas da empresa
    - name: "Tabelas Internas"
      type: literal
      values:
        - "substituir_pelo_nome_real_da_tabela"
        - "outra_tabela_confidencial"
      label: "[INTERNAL_TABLE_REDACTED]"
      enabled: false

    # Exemplo: Detectar nomes de projectos ou operações confidenciais
    - name: "Projectos Confidenciais"
      type: literal
      values:
        - "Projecto Exemplo"
        - "Operação Exemplo"
      label: "[CONFIDENTIAL_PROJECT_REDACTED]"
      enabled: false

    # Exemplo: Detectar tokens JWT
    - name: "JWT Token"
      type: regex
      pattern: 'eyJ[a-zA-Z0-9\-_]+\.eyJ[a-zA-Z0-9\-_]+\.[a-zA-Z0-9\-_]+'
      label: "[JWT_REDACTED]"
      enabled: false

    # Exemplo: Detectar chaves privadas SSH/RSA
    - name: "Chave Privada"
      type: regex
      pattern: '-----BEGIN (RSA |EC |OPENSSH )?PRIVATE KEY-----'
      label: "[PRIVATE_KEY_REDACTED]"
      enabled: false
```

---

## Passo 6 — Adicionar `galileu.yml` ao `.gitignore`

O ficheiro `galileu.yml` pode conter informações sensíveis sobre a estrutura interna da empresa (nomes de tabelas, termos confidenciais, etc.). Por isso, deve ser adicionado ao `.gitignore`:

```gitignore
# Configuração local do Galileu (pode conter termos confidenciais)
galileu.yml
```

Juntamente, criar um ficheiro `galileu.yml.example` na raiz com o mesmo conteúdo do `galileu.yml` criado no Passo 5. Este ficheiro de exemplo **deve** ser versionado no repositório para servir de referência.

---

## Passo 7 — Validação e Testes

### 7.1 — Compilar o projecto

```bash
# Apple Silicon
GOOS=darwin GOARCH=arm64 go build -o galileu ./cmd/sentinel/main.go

# Intel
GOOS=darwin GOARCH=amd64 go build -o galileu ./cmd/sentinel/main.go
```

A compilação não deve produzir erros ou warnings.

### 7.2 — Teste 1: comportamento legacy (sem galileu.yml)

```bash
# Renomear temporariamente o ficheiro YAML
mv galileu.yml galileu.yml.bak
./galileu
```

Resultado esperado no terminal:
```
[Galileu] galileu.yml não encontrado em './galileu.yml'. A usar padrões built-in por omissão.
[Galileu] 8 padrão(ões) de detecção carregado(s)...
```
O proxy deve arrancar normalmente e detectar os padrões built-in como antes.

```bash
# Restaurar
mv galileu.yml.bak galileu.yml
```

### 7.3 — Teste 2: padrões built-in via YAML

No `galileu.yml`, desactivar um padrão (ex: `aws_key: false`), reiniciar o Galileu e confirmar que o log de arranque indica o número correcto de padrões carregados (deve ser 7 em vez de 8).

### 7.4 — Teste 3: padrão custom do tipo `literal`

No `galileu.yml`, activar o padrão de exemplo "Tabelas Internas" e substituir os valores de exemplo por um termo real. Enviar uma requisição de teste através do proxy que contenha esse termo e confirmar que o `galileu_audit.log` regista `"redacted": true`.

### 7.5 — Teste 4: padrão custom do tipo `regex`

Activar o padrão "Connection String de Base de Dados" e enviar uma requisição contendo `postgres://user:password@host:5432/db`. Confirmar que o body interceptado contém `[CONNECTION_STRING_REDACTED]`.

### 7.6 — Teste 5: regex inválida não quebra o arranque

No `galileu.yml`, adicionar um padrão custom com uma regex inválida (ex: `pattern: "[invalid"`). O Galileu deve arrancar normalmente, imprimir um aviso no terminal (`Padrão custom '...' tem regex inválida e será ignorado`), e continuar a funcionar com os demais padrões.

---

## Comportamentos Obrigatórios a Preservar

O agente deve confirmar que nenhuma das seguintes funcionalidades foi alterada:

- O proxy MITM continua a correr na porta 9000.
- Os certificados `ca.pem` e `key.pem` são carregados da mesma forma.
- O ficheiro `galileu_audit.log` continua a ser gerado com o mesmo formato JSON.
- Os campos `redacted` e `pattern_type` do log continuam a ser preenchidos correctamente.
- A performance não é degradada: os padrões continuam a ser compilados **uma única vez** na inicialização, nunca em runtime por requisição.

---

## Estrutura Final de Ficheiros Esperada

```
GalileuForMac/
├── cmd/
│   └── sentinel/
│       └── main.go              # MODIFICADO
├── internal/
│   └── guardian/
│       ├── guardian.go          # SEM ALTERAÇÕES
│       ├── analyzer.go          # MODIFICADO
│       ├── audit.go             # SEM ALTERAÇÕES
│       └── config.go            # CRIADO
├── galileu                      # Binário recompilado
├── galileu.yml                  # CRIADO (não versionado)
├── galileu.yml.example          # CRIADO (versionado)
├── ca.pem
├── key.pem
├── start-opencode.sh
├── go.mod                       # MODIFICADO (nova dependência yaml.v3)
├── go.sum                       # ACTUALIZADO automaticamente
├── .gitignore                   # MODIFICADO (adicionar galileu.yml)
└── galileu_audit.log
```

---

## Dependências Introduzidas

| Pacote | Versão | Justificação |
|---|---|---|
| `gopkg.in/yaml.v3` | latest stable | Parse do ficheiro `galileu.yml`. Biblioteca madura, sem dependências transitivas relevantes, usada extensivamente no ecossistema Go. |

Nenhuma outra dependência deve ser adicionada.

---

## Notas Finais para o Agente

- Não refactorizar código que não esteja no âmbito deste documento.
- Não alterar a assinatura de funções públicas existentes salvo o estritamente necessário para injectar o `Analyzer`.
- Manter os comentários existentes no código e adicionar comentários novos nas secções criadas, seguindo o estilo já presente no projecto.
- Em caso de ambiguidade sobre a assinatura actual de uma função, preferir a abordagem de menor impacto no código existente.
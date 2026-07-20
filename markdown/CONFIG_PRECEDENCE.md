# Resolução de Configuração por Precedência

O Galileu agora resolve o caminho do `galileu.yml` por precedência configurável, em vez de exclusivamente `./galileu.yml` no CWD do processo. Isso viabiliza uma config padrão de equipe (systemd/daemon) coexistindo com overrides por projeto, sem depender de symlink ou de derrubar/subir processo para trocar de config.

---

## Ordem de precedência

| Prioridade | Fonte | Descrição |
|---|---|---|
| 1 (maior) | `--config <path>` | Flag de linha de comando |
| 2 | `GALILEU_CONFIG` | Variável de ambiente |
| 3 | `./galileu.yml` | Arquivo no CWD (comportamento atual) |
| 4 (menor) | Built-in | Padrões embutidos (quando nenhum arquivo existe) |

A primeira fonte encontrada é usada. As fontes seguintes são ignoradas.

---

## Variáveis de ambiente

| Variável | Descrição |
|---|---|
| `GALILEU_CONFIG` | Caminho para o arquivo de configuração. Ignorado se `--config` for passado. |
| `GALILEU_PORT` | Porta do proxy (usado pelo `doctor` para diagnóstico). |

---

## Exemplos de uso

### Config de equipe via systemd

```bash
# Definir no unit file do systemd
Environment="GALILEU_CONFIG=/etc/galileu/team-config.yml"

# Ou via linha de comando
galileu --config /etc/galileu/team-config.yml
```

### Override por projeto

```bash
# No diretório do projeto
galileu --config ./galileu-project.yml
```

### Comportamento padrão (retrocompatível)

```bash
# Sem flag nem env, usa ./galileu.yml do CWD
galileu

# Sem arquivo no CWD, usa padrões built-in
galileu
```

### Via variável de ambiente

```bash
GALILEU_CONFIG=~/.config/galileu/personal.yml galileu
```

---

## Comportamento por caso

### Caso 1: `--config` passado

```bash
galileu --config /path/to/config.yml
```

- Arquivo **deve** existir. Caso contrário, erro fatal com mensagem clara.
- Ignora `GALILEU_CONFIG` e `./galileu.yml`.

**Saída esperada:**
```
[GALILEU] config: usando --config=/path/to/config.yml
[Galileu] XX padrao(oes) de deteccao carregado(s) a partir de '/path/to/config.yml'.
```

**Caso de erro:**
```
[ERRO] Falha ao resolver configuracao: Configuracao explicita: arquivo '/path/inexistente.yml' nao encontrado
```

### Caso 2: `GALILEU_CONFIG` definido

```bash
GALILEU_CONFIG=/path/to/config.yml galileu
```

- Arquivo **deve** existir. Caso contrário, erro fatal.
- Só é verificado se `--config` não foi passado.

**Saída esperada:**
```
[GALILEU] config: usando GALILEU_CONFIG=/path/to/config.yml
```

### Caso 3: `./galileu.yml` no CWD

```bash
galileu
```

- Se o arquivo existe, é carregado normalmente.
- Comportamento idêntico ao anterior à mudança.

**Saída esperada:**
```
[GALILEU] config: usando galileu.yml (CWD)
```

### Caso 4: Nenhum arquivo

```bash
galileu
```

- Sem `--config`, sem `GALILEU_CONFIG`, sem `./galileu.yml` no CWD.
- Todos os padrões built-in são ativados.

**Saída esperada:**
```
[GALILEU] config: nenhum arquivo encontrado, usando padroes built-in
[Galileu] galileu.yml nao especificado. A usar padroes built-in por omissao.
```

---

## Diagnóstico (`galileu doctor`)

O comando `galileu doctor` mostra qual config está ativa:

```bash
galileu doctor
```

**Saída esperada:**
```
=== Diagnostico do Galileu ===
Certificado CA:      [FALHA] Nao instalado
Porta configurada:    [OK] 9000 (padrao)
Porta disponivel:    [OK] Livre
Config ativa:        [OK] galileu.yml (CWD)

Problemas encontrados:
  - certificado CA nao instalado no repositorio do sistema
```

Valores possíveis para "Config ativa":

| Valor | Significado |
|---|---|
| `[OK] /path/config.yml (via --config)` | Flag `--config` usada |
| `[OK] /path/config.yml (via GALILEU_CONFIG)` | Variável de ambiente usada |
| `[OK] galileu.yml (CWD)` | Arquivo no diretório atual |
| `[OK] padroes built-in (nenhum arquivo)` | Nenhum arquivo encontrado |

---

## Arquivos modificados

| Arquivo | Alteração |
|---|---|
| `cmd/galileu/main.go` | Novo flag `--config`, parsing de `extractFlagValue()`, propagação de `configPath`, log de origem, help text atualizado |
| `internal/guardian/config.go` | Nova struct `ConfigSource`, nova função `ResolveConfigPath()`, `LoadConfig` trata path vazio |
| `internal/doctor/doctor.go` | Novos campos `ConfigPath`/`ConfigSource` no `DiagnosticResult`, chamada a `ResolveConfigPath` |
| `internal/tui/model.go` | `New()` recebe `configPath` em vez de hardcoded |
| `internal/tui/tui.go` | `Start()` recebe e repassa `configPath` |
| `internal/guardian/config_test.go` | 6 novos testes de precedência |

---

## Testes unitários

Os testes cobrem os 4 cenários de precedência:

| Teste | Cenário |
|---|---|
| `TestResolveConfigPath_FlagTakesPrecedence` | `--config` definido + `GALILEU_CONFIG` definido → flag vence |
| `TestResolveConfigPath_EnvFallback` | Sem flag, com `GALILEU_CONFIG` → usa env |
| `TestResolveConfigPath_CWDFallback` | Sem flag, sem env, com `galileu.yml` no CWD → usa CWD |
| `TestResolveConfigPath_BuiltinFallback` | Sem flag, sem env, sem CWD → built-in |
| `TestResolveConfigPath_FlagNonexistent` | `--config` apontando para arquivo inexistente → erro |
| `TestResolveConfigPath_EnvNonexistent` | `GALILEU_CONFIG` apontando para arquivo inexistente → erro |

Executar testes:

```bash
make test
# ou
go test ./... -v
```

---

## Fora de escopo

As seguintes funcionalidades **não** foram incluídas nesta mudança:

- **Merge/mescla** entre config de equipe e config de projeto (por ora é substituição total do arquivo, não merge de campos).
- **Hot-reload** de config sem restart do processo.
